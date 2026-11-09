package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tennuem/tbot/internal/history"
	"github.com/tennuem/tbot/pkg/service"
)

func NewSqLiteStore(dataSource string) (*Store, error) {
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		return nil, err
	}
	// Serialize short cache transactions and keep :memory: databases on one connection.
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if _, err = db.Exec("PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); err == nil {
		err = s.initializeSchema()
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize SQLite: %w", err)
	}
	return s, nil
}

// Store keeps the track cache and link-sharing history in SQLite.
type Store struct{ db *sql.DB }

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Save(ctx context.Context, m *service.Message) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := save(ctx, tx, m); err != nil {
		return err
	}
	return tx.Commit()
}

func save(ctx context.Context, tx *sql.Tx, m *service.Message) error {
	return saveWithAliases(ctx, tx, m, nil)
}

func saveWithAliases(ctx context.Context, tx *sql.Tx, m *service.Message, aliases []service.Link) error {
	query := `SELECT DISTINCT e.payload FROM cache_entries e LEFT JOIN cache_aliases a ON a.cache_key=e.cache_key WHERE e.cache_key=?`
	args := []any{m.CacheKey}
	for _, link := range append(append([]service.Link(nil), m.Links...), aliases...) {
		query += " OR (a.provider=? AND a.url=?)"
		args = append(args, link.Provider, link.URL)
	}
	query += " ORDER BY e.created_at, e.rowid"
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	var old []*service.Message
	for rows.Next() {
		var payload string
		var record service.Message
		if err = rows.Scan(&payload); err == nil {
			err = json.Unmarshal([]byte(payload), &record)
		}
		if err != nil {
			_ = rows.Close()
			return err
		}
		old = append(old, &record)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	merged := service.CloneMessage(m)
	if len(old) > 0 {
		merged = old[0]
		for _, record := range old[1:] {
			merged = service.MergeCache(merged, record)
		}
		merged = service.MergeCache(merged, m)
	}
	if merged.CacheKey == "" {
		merged.CacheKey = merged.Provider + "|" + merged.URL
	}
	// Importing an earlier sender must not erase the previously known submission.
	for _, record := range old {
		if record.ChatID != 0 && !record.FirstSeenAt.IsZero() && (record.UserID != 0 || record.UserName != "") &&
			(!record.FirstSeenAt.Equal(merged.FirstSeenAt) || record.UserID != merged.UserID || record.ChatID != merged.ChatID) {
			if _, err := recordShare(ctx, tx, service.Share{MessageID: record.MessageID, ChatID: record.ChatID,
				UserID: record.UserID, UserName: record.UserName, SentAt: record.FirstSeenAt,
				URL: record.URL, Provider: record.Provider, Title: record.Title}); err != nil {
				return err
			}
		}
	}
	payload, err := json.Marshal(merged)
	if err != nil {
		return err
	}
	created := int64(0)
	if !merged.FirstSeenAt.IsZero() {
		created = merged.FirstSeenAt.UnixMilli()
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO cache_entries(cache_key,payload,user_id,created_at) VALUES(?,?,?,?)
		ON CONFLICT(cache_key) DO UPDATE SET payload=excluded.payload,user_id=excluded.user_id,created_at=excluded.created_at`,
		merged.CacheKey, string(payload), merged.UserID, created)
	if err != nil {
		return err
	}
	// Preserve historical URLs as aliases when a provider returns a new canonical URL.
	for _, record := range old {
		if record.CacheKey == merged.CacheKey {
			continue
		}
		if _, err := tx.ExecContext(ctx, "UPDATE cache_aliases SET cache_key=? WHERE cache_key=?", merged.CacheKey, record.CacheKey); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM cache_entries WHERE cache_key=?", record.CacheKey); err != nil {
			return err
		}
	}
	for _, link := range append(append([]service.Link(nil), merged.Links...), aliases...) {
		_, err := tx.ExecContext(ctx, `INSERT INTO cache_aliases(provider,url,cache_key) VALUES(?,?,?)
			ON CONFLICT(provider,url) DO UPDATE SET cache_key=excluded.cache_key`, link.Provider, link.URL, merged.CacheKey)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) FindByURL(ctx context.Context, host, rawURL string) (*service.Message, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT e.payload FROM cache_aliases a JOIN cache_entries e ON e.cache_key=a.cache_key
		WHERE a.provider=? AND a.url=?`, host, rawURL).Scan(&payload)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, service.ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("read cached links: %w", err)
	}
	var m service.Message
	if err := json.Unmarshal([]byte(payload), &m); err != nil {
		return nil, fmt.Errorf("decode cached links: %w", err)
	}
	return &m, nil
}

func (s *Store) FindByUser(ctx context.Context, userID int) ([]service.Message, error) {
	result, seen, err := s.sharedByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(result) == 100 {
		return result, nil
	}
	rows, err := s.db.QueryContext(ctx, "SELECT payload FROM cache_entries WHERE user_id=? ORDER BY created_at DESC,rowid DESC LIMIT 100", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var payload string
		var m service.Message
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			return nil, err
		}
		if !seen[m.CacheKey] && len(result) < 100 {
			seen[m.CacheKey] = true
			result = append(result, m)
		}
	}
	return result, rows.Err()
}

func (s *Store) sharedByUser(ctx context.Context, userID int) ([]service.Message, map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT h.payload,COALESCE(e.payload,'') FROM history_shares h
		LEFT JOIN cache_aliases a ON a.provider=json_extract(h.payload,'$.provider') AND a.url=json_extract(h.payload,'$.url')
		LEFT JOIN cache_entries e ON e.cache_key=a.cache_key WHERE h.user_id=? ORDER BY h.sent_at DESC,h.share_key`, userID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var result []service.Message
	seen := make(map[string]bool)
	for rows.Next() {
		var payload, cached string
		if err := rows.Scan(&payload, &cached); err != nil {
			return nil, nil, err
		}
		var share service.Share
		if err := json.Unmarshal([]byte(payload), &share); err != nil {
			return nil, nil, err
		}
		if !history.IsMusicTrack(share.URL) {
			continue
		}
		m := service.Message{CacheKey: share.Provider + "|" + share.URL, Title: share.Title}
		if cached != "" {
			if err := json.Unmarshal([]byte(cached), &m); err != nil {
				return nil, nil, err
			}
		}
		if seen[m.CacheKey] {
			continue
		}
		seen[m.CacheKey] = true
		m.URL, m.Provider, m.UserID, m.UserName = share.URL, share.Provider, share.UserID, share.UserName
		m.ChatID, m.MessageID, m.FirstSeenAt = share.ChatID, share.MessageID, share.SentAt
		result = append(result, m)
		if len(result) == 100 {
			break
		}
	}
	return result, seen, rows.Err()
}

func (s *Store) initializeSchema() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS cache_entries(cache_key TEXT PRIMARY KEY,payload TEXT NOT NULL,user_id INTEGER NOT NULL,created_at INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS cache_aliases(provider TEXT NOT NULL,url TEXT NOT NULL,cache_key TEXT NOT NULL REFERENCES cache_entries(cache_key),PRIMARY KEY(provider,url));
		CREATE INDEX IF NOT EXISTS cache_aliases_entry ON cache_aliases(cache_key);
		CREATE INDEX IF NOT EXISTS cache_entries_user ON cache_entries(user_id,created_at);
		CREATE TABLE IF NOT EXISTS history_shares(share_key TEXT PRIMARY KEY,payload TEXT NOT NULL,user_id INTEGER NOT NULL,sent_at INTEGER NOT NULL);
		CREATE INDEX IF NOT EXISTS history_shares_user ON history_shares(user_id,sent_at);`)
	return err
}
