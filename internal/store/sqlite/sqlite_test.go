package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/service"
)

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	store, err := NewSqLiteStore(path)
	require.NoError(t, err)
	s := store
	t.Cleanup(func() { _ = s.db.Close() })
	return s
}

func cacheMessage() *service.Message {
	return &service.Message{CacheKey: "first", Provider: "music.yandex.com", URL: "https://music.yandex.com/album/1/track/2", Title: "Track — Artist",
		UserID: 10, UserName: "@first", ChatID: 42, FirstSeenAt: time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC), CheckedAt: time.Now().UTC(),
		Links: []service.Link{
			{Provider: "music.yandex.com", URL: "https://music.yandex.com/album/1/track/2", Name: "Яндекс Музыка"},
			{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc", Name: "Spotify"},
		}}
}

func TestCacheStoresAllProviderLinks(t *testing.T) {
	s := openTestStore(t, ":memory:")
	m := cacheMessage()
	require.NoError(t, s.Save(context.Background(), m))
	for _, link := range m.Links {
		got, err := s.FindByURL(context.Background(), link.Provider, link.URL)
		require.NoError(t, err)
		require.Equal(t, m, got)
	}
	_, err := s.FindByURL(context.Background(), "wrong-provider", m.URL)
	require.ErrorIs(t, err, service.ErrCacheMiss)
	_, err = s.FindByURL(context.Background(), m.Provider, "https://music.yandex.com/absent")
	require.ErrorIs(t, err, service.ErrCacheMiss)
	list, err := s.FindByUser(context.Background(), m.UserID)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

func TestCacheKeepsLinksAndSenderAfterReopeningDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "tbot.db")
	s := openTestStore(t, path)
	m := cacheMessage()
	require.NoError(t, s.Save(context.Background(), m))
	require.NoError(t, s.db.Close())

	reopened := openTestStore(t, path)
	for _, link := range m.Links {
		got, err := reopened.FindByURL(context.Background(), link.Provider, link.URL)
		require.NoError(t, err)
		require.Equal(t, m, got)
	}
	list, err := reopened.FindByUser(context.Background(), m.UserID)
	require.NoError(t, err)
	require.Equal(t, []service.Message{*m}, list)
	var count int
	require.NoError(t, reopened.db.QueryRow("SELECT COUNT(*) FROM cache_migrations WHERE version=1").Scan(&count))
	require.Equal(t, 1, count)
}

func TestCacheRefreshKeepsFirstSenderAndHistoricalAliases(t *testing.T) {
	s := openTestStore(t, ":memory:")
	m := cacheMessage()
	require.NoError(t, s.Save(context.Background(), m))
	fresh := service.CloneMessage(m)
	fresh.UserID, fresh.UserName, fresh.ChatID = 20, "@second", 99
	fresh.FirstSeenAt = m.FirstSeenAt.Add(time.Hour)
	fresh.CheckedAt = m.CheckedAt.Add(time.Minute)
	fresh.Links[1].URL = "https://open.spotify.com/track/new"
	require.NoError(t, s.Save(context.Background(), fresh))
	got, err := s.FindByURL(context.Background(), "open.spotify.com", m.Links[1].URL)
	require.NoError(t, err)
	require.Equal(t, 10, got.UserID)
	require.Equal(t, "@first", got.UserName)
	require.Equal(t, int64(42), got.ChatID)
	require.Equal(t, m.FirstSeenAt, got.FirstSeenAt)
	require.Equal(t, fresh.Links, got.Links)
	_, err = s.FindByURL(context.Background(), "open.spotify.com", fresh.Links[1].URL)
	require.NoError(t, err)
	// An older in-flight result must not overwrite a more recent refresh.
	require.NoError(t, s.Save(context.Background(), m))
	got, err = s.FindByURL(context.Background(), m.Provider, m.URL)
	require.NoError(t, err)
	require.Equal(t, fresh.Links, got.Links)

	fresh.MissingProviders = []string{"open.spotify.com"}
	fresh.CheckedAt = fresh.CheckedAt.Add(time.Minute)
	fresh.Links = fresh.Links[:1]
	require.NoError(t, s.Save(context.Background(), fresh))
	got, err = s.FindByURL(context.Background(), m.Provider, m.URL)
	require.NoError(t, err)
	require.Len(t, got.Links, 1)
}

func TestCacheMergesOverlappingRecords(t *testing.T) {
	s := openTestStore(t, ":memory:")
	m := cacheMessage()
	other := service.CloneMessage(m)
	other.CacheKey, other.URL, other.Provider, other.UserID = "second", m.Links[1].URL, "open.spotify.com", 20
	other.FirstSeenAt = m.FirstSeenAt.Add(time.Hour)
	other.Links = other.Links[1:]
	require.NoError(t, s.Save(context.Background(), other))
	require.NoError(t, s.Save(context.Background(), m))
	for _, link := range m.Links {
		got, err := s.FindByURL(context.Background(), link.Provider, link.URL)
		require.NoError(t, err)
		require.Len(t, got.Links, 2)
		require.Equal(t, 10, got.UserID)
	}
	var count int
	require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&count))
	require.Equal(t, 1, count)
}

func TestCachePersistsAndMigratesLegacyRows(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cache.db")
	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE links(id INTEGER PRIMARY KEY,url TEXT,title TEXT,user_id INTEGER);
		INSERT INTO links(url,title,user_id) VALUES('https://music.yandex.ru/album/1/track/2?utm_source=web','Track',10);`)
	require.NoError(t, err)
	require.NoError(t, db.Close())
	s := openTestStore(t, path)
	old, err := s.FindByURL(context.Background(), "music.yandex.com", "https://music.yandex.com/album/1/track/2")
	require.NoError(t, err)
	require.Equal(t, 10, old.UserID)
	require.True(t, old.CheckedAt.IsZero())
	require.True(t, old.FirstSeenAt.IsZero()) // Do not invent a historical send date.
	m := cacheMessage()
	require.NoError(t, s.Save(context.Background(), m))
	require.NoError(t, s.db.Close())
	reopened := openTestStore(t, path)
	got, err := reopened.FindByURL(context.Background(), "open.spotify.com", m.Links[1].URL)
	require.NoError(t, err)
	require.Len(t, got.Links, 2)
	var count int
	require.NoError(t, reopened.db.QueryRow("SELECT COUNT(*) FROM links").Scan(&count))
	require.Equal(t, 1, count) // Original data is retained for recovery.
}

func TestCacheConcurrentSaveAndRealErrors(t *testing.T) {
	s := openTestStore(t, ":memory:")
	m := cacheMessage()
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- s.Save(context.Background(), m) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, s.Save(ctx, m), context.Canceled)
	require.NoError(t, s.db.Close())
	_, err := s.FindByURL(context.Background(), m.Provider, m.URL)
	require.Error(t, err)
	require.NotErrorIs(t, err, service.ErrCacheMiss)
}
