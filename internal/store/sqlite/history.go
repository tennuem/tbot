package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"time"

	"github.com/tennuem/tbot/internal/history"
	"github.com/tennuem/tbot/pkg/service"
)

// OpenReadOnly opens an existing SQLite file without creating or migrating it.
func OpenReadOnly(path string) (*Store, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	u := url.URL{Scheme: "file", Path: path, RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite3", u.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) RecordShare(ctx context.Context, share service.Share) error {
	archive := history.Archive{Version: history.Version, Shares: []service.Share{share}}
	if err := archive.Validate(true); err != nil {
		return err
	}
	_, err := recordShare(ctx, s.db, archive.Shares[0])
	return err
}

type shareWriter interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func recordShare(ctx context.Context, db shareWriter, share service.Share) (int64, error) {
	payload, err := json.Marshal(share)
	if err != nil {
		return 0, err
	}
	result, err := db.ExecContext(ctx, `INSERT INTO history_shares(share_key,payload,user_id,sent_at) VALUES(?,?,?,?)
		ON CONFLICT(share_key) DO NOTHING`, history.ShareKey(share), string(payload), share.UserID, share.SentAt.UnixMilli())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) ExportHistory(ctx context.Context) (*history.Archive, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	archive := &history.Archive{Version: history.Version, CreatedAt: time.Now().UTC(), Shares: []service.Share{}}
	var hasShares int
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='history_shares'").Scan(&hasShares); err != nil {
		return nil, err
	}
	if hasShares != 0 {
		rows, err := tx.QueryContext(ctx, "SELECT payload FROM history_shares ORDER BY sent_at,share_key")
		if err != nil {
			return nil, err
		}
		if err := readPayloads(rows, &archive.Shares); err != nil {
			return nil, err
		}
	}
	rows, err := tx.QueryContext(ctx, "SELECT payload FROM cache_entries ORDER BY cache_key")
	if err != nil {
		return nil, err
	}
	if err := readPayloads(rows, &archive.Cache); err != nil {
		return nil, err
	}
	rows, err = tx.QueryContext(ctx, "SELECT provider,url,cache_key FROM cache_aliases ORDER BY provider,url")
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var alias history.Alias
		if err := rows.Scan(&alias.Provider, &alias.URL, &alias.CacheKey); err != nil {
			_ = rows.Close()
			return nil, err
		}
		archive.Aliases = append(archive.Aliases, alias)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return archive, archive.Validate(false)
}

func readPayloads[T any](rows *sql.Rows, destination *[]T) error {
	defer rows.Close()
	for rows.Next() {
		var payload string
		var value T
		if err := rows.Scan(&payload); err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(payload), &value); err != nil {
			return err
		}
		*destination = append(*destination, value)
	}
	return rows.Err()
}

type ImportResult struct {
	SharesAdded int64 `json:"shares_added"`
	Duplicates  int64 `json:"duplicate_shares"`
	TrackLinks  int   `json:"track_links"`
	Cache       int   `json:"cache_records"`
}

// ImportHistory merges a validated archive in a single transaction. Reimporting
// the same messages preserves one event per sender, timestamp, message and URL.
func (s *Store) ImportHistory(ctx context.Context, archive *history.Archive) (ImportResult, error) {
	var result ImportResult
	if err := archive.Validate(true); err != nil {
		return result, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	aliases := make(map[string][]service.Link)
	for _, alias := range archive.Aliases {
		aliases[alias.CacheKey] = append(aliases[alias.CacheKey], service.Link{Provider: alias.Provider, URL: alias.URL})
	}
	for i := range archive.Cache {
		m := &archive.Cache[i]
		if err := saveWithAliases(ctx, tx, m, aliases[m.CacheKey]); err != nil {
			return ImportResult{}, fmt.Errorf("import cached track: %w", err)
		}
		result.Cache++
	}
	shares := append([]service.Share(nil), archive.Shares...)
	sort.SliceStable(shares, func(i, j int) bool { return shares[i].SentAt.Before(shares[j].SentAt) })
	seen := make(map[string]bool)
	for _, share := range shares {
		added, err := recordShare(ctx, tx, share)
		if err != nil {
			return ImportResult{}, fmt.Errorf("import share: %w", err)
		}
		result.SharesAdded += added
		result.Duplicates += 1 - added
		if !history.IsMusicTrack(share.URL) || seen[share.URL] {
			continue
		}
		seen[share.URL] = true
		m := &service.Message{CacheKey: share.Provider + "|" + share.URL, Provider: share.Provider, URL: share.URL,
			Title: share.Title, UserID: share.UserID, UserName: share.UserName, ChatID: share.ChatID,
			MessageID: share.MessageID, FirstSeenAt: share.SentAt,
			Links: []service.Link{{Provider: share.Provider, URL: share.URL}}}
		if err := save(ctx, tx, m); err != nil {
			return ImportResult{}, fmt.Errorf("index historical track: %w", err)
		}
		result.TrackLinks++
	}
	if err := tx.Commit(); err != nil {
		return ImportResult{}, err
	}
	return result, nil
}
