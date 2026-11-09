package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/tbot/pkg/service"
)

func NewSqLiteStore(dataSource string) (service.Store, error) {
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		return nil, err
	}
	// Serialize short cache transactions and keep :memory: databases on one connection.
	db.SetMaxOpenConns(1)
	s := &sqLiteStore{db: db}
	if _, err = db.Exec("PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); err == nil {
		err = s.migrations()
	}
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("initialize SQLite: %w", err)
	}
	return s, nil
}

type sqLiteStore struct{ db *sql.DB }

func (s *sqLiteStore) Save(ctx context.Context, m *service.Message) error {
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
	query := `SELECT DISTINCT e.payload FROM cache_entries e LEFT JOIN cache_aliases a ON a.cache_key=e.cache_key WHERE e.cache_key=?`
	args := []any{m.CacheKey}
	for _, link := range m.Links {
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
	for _, link := range merged.Links {
		_, err := tx.ExecContext(ctx, `INSERT INTO cache_aliases(provider,url,cache_key) VALUES(?,?,?)
			ON CONFLICT(provider,url) DO UPDATE SET cache_key=excluded.cache_key`, link.Provider, link.URL, merged.CacheKey)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *sqLiteStore) FindByURL(ctx context.Context, host, rawURL string) (*service.Message, error) {
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

func (s *sqLiteStore) FindByUser(ctx context.Context, userID int) ([]service.Message, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT payload FROM cache_entries WHERE user_id=? ORDER BY created_at DESC,rowid DESC LIMIT 100", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []service.Message
	for rows.Next() {
		var payload string
		var m service.Message
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(payload), &m); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (s *sqLiteStore) migrations() error {
	_, err := s.db.Exec(`CREATE TABLE IF NOT EXISTS links(id INTEGER PRIMARY KEY,url TEXT,title TEXT,user_id INTEGER);
		CREATE TABLE IF NOT EXISTS cache_entries(cache_key TEXT PRIMARY KEY,payload TEXT NOT NULL,user_id INTEGER NOT NULL,created_at INTEGER NOT NULL);
		CREATE TABLE IF NOT EXISTS cache_aliases(provider TEXT NOT NULL,url TEXT NOT NULL,cache_key TEXT NOT NULL REFERENCES cache_entries(cache_key),PRIMARY KEY(provider,url));
		CREATE INDEX IF NOT EXISTS cache_aliases_entry ON cache_aliases(cache_key);
		CREATE INDEX IF NOT EXISTS cache_entries_user ON cache_entries(user_id,created_at);
		CREATE TABLE IF NOT EXISTS cache_migrations(version INTEGER PRIMARY KEY);`)
	if err != nil {
		return err
	}
	var version int
	err = s.db.QueryRow("SELECT version FROM cache_migrations WHERE version=1").Scan(&version)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	rows, err := s.db.Query("SELECT url,COALESCE(title,''),COALESCE(user_id,0) FROM links ORDER BY id")
	if err != nil {
		return err
	}
	var legacy []*service.Message
	for rows.Next() {
		m := &service.Message{}
		if err := rows.Scan(&m.URL, &m.Title, &m.UserID); err != nil {
			_ = rows.Close()
			return err
		}
		u, err := provider.NormalizeURL(m.URL)
		if err != nil {
			continue // The original links table remains intact, including malformed legacy URLs.
		}
		m.URL, m.Provider = u.String(), u.Host
		m.CacheKey = m.Provider + "|" + m.URL
		m.Links = []service.Link{{Provider: m.Provider, URL: m.URL, Name: strings.TrimPrefix(m.Provider, "music.")}}
		legacy = append(legacy, m)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, m := range legacy {
		if err := save(context.Background(), tx, m); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("INSERT INTO cache_migrations(version) VALUES(1)"); err != nil {
		return err
	}
	return tx.Commit()
}
