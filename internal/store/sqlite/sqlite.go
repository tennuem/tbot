package sqlite

import (
	"context"
	"database/sql"
	"log"

	sq "github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/pkg/service"
)

func NewSqLiteStore(dataSource string) (service.Store, error) {
	db, err := sql.Open("sqlite3", dataSource)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &sqLiteStore{db: db}
	if err := s.migrations(); err != nil {
		return nil, err
	}
	return s, nil
}

type sqLiteStore struct {
	db *sql.DB
}

func (s *sqLiteStore) Save(ctx context.Context, m *service.Message) error {
	b := sq.Insert("links").Columns("url", "title", "username")
	b = b.Values(m.URL, m.Title, m.Username)
	for _, l := range m.Links {
		b = b.Values(l, m.Title, m.Username)
	}
	query, args, err := b.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return errors.Wrap(err, "could not build query")
	}
	_, err = s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return errors.Wrap(err, "save msg to db")
	}
	return nil
}

func (s *sqLiteStore) FindByURL(ctx context.Context, url string) (*service.Message, error) {
	var m service.Message
	b := sq.Select("url", "title", "username").From("links").Where(sq.Eq{"url": url})
	query, args, err := b.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "could not build query")
	}
	err = s.db.QueryRowContext(ctx, query, args...).Scan(&m.URL, &m.Title, &m.Username)
	if err != nil {
		return nil, errors.Wrap(err, "find message by url")
	}
	return &m, nil
}

func (s *sqLiteStore) FindByUsername(ctx context.Context, username string) ([]service.Message, error) {
	var res []service.Message
	b := sq.Select("url", "title", "username").From("links").Where(sq.Eq{"username": username})
	query, args, err := b.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		return nil, errors.Wrap(err, "could not build query")
	}
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.Wrap(err, "find messages by username")
	}
	defer rows.Close()
	for rows.Next() {
		var m service.Message
		err = rows.Scan(&m.URL, &m.Title, &m.Username)
		if err != nil {
			continue
		}
		res = append(res, m)
	}
	return res, nil
}

func (s *sqLiteStore) migrations() error {
	query := `
		CREATE TABLE IF NOT EXISTS links(
			id INTEGER PRIMARY KEY,
			url TEXT,
			title TEXT,
			username TEXT
		);
	`
	_, err := s.db.Exec(query)
	if err != nil {
		return errors.Wrap(err, "make migrations")
	}
	return nil
}
