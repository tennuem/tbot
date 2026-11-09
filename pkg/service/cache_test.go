package service

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/provider"
)

type cacheProvider struct {
	host   string
	title  func(string) (string, error)
	search func(string) (string, error)
}

func (p *cacheProvider) Host() string { return p.host }
func (p *cacheProvider) Name() string { return p.host }
func (p *cacheProvider) GetTitle(raw string) (string, error) {
	if p.title != nil {
		return p.title(raw)
	}
	return "Track — Artist", nil
}
func (p *cacheProvider) GetURL(title string) (string, error) {
	if p.search != nil {
		return p.search(title)
	}
	return "https://" + p.host + "/track/found?utm_source=share", nil
}

func cachedTrack() *Message {
	return &Message{CacheKey: "original", Provider: "source.test", URL: "https://source.test/track/1", Title: "Cached track",
		UserID: 10, UserName: "@first", ChatID: 42, FirstSeenAt: time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC), CheckedAt: time.Now().UTC(),
		Links: []Link{{Provider: "source.test", URL: "https://source.test/track/1", Name: "Source"}, {Provider: "other.test", URL: "https://other.test/track/2", Name: "Other"}}}
}

func TestCacheHitReturnsImmediatelyAndRefreshesOnce(t *testing.T) {
	store := NewStoreMock()
	old := cachedTrack()
	require.NoError(t, store.Save(context.Background(), old))
	started, release := make(chan struct{}, 1), make(chan struct{})
	var calls atomic.Int32
	source := &cacheProvider{host: "source.test", title: func(string) (string, error) {
		calls.Add(1)
		started <- struct{}{}
		<-release
		return "Updated track", nil
	}}
	other := &cacheProvider{host: "other.test"}
	svc := NewService(context.Background(), store, source, other).(*service)
	defer func() { close(release); svc.workers.Wait() }()
	input := &Message{URL: "https://other.test/track/2?utm_medium=copy&ref_id=x", UserID: 20, ChatID: 42}
	result, err := svc.FindLinks(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "hit", result.CacheStatus)
	require.True(t, result.Refreshing)
	require.Equal(t, "Cached track", result.Title)
	require.Equal(t, "https://other.test/track/2", result.URL)
	require.Equal(t, "other.test", result.Provider)
	require.Equal(t, "@first", result.UserName)
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("background refresh did not start")
	}
	_, err = svc.FindLinks(context.Background(), input)
	require.NoError(t, err)
	require.EqualValues(t, 1, calls.Load())
}

func TestBackgroundRefreshUpdatesCacheAndPreservesFirstSender(t *testing.T) {
	store := NewStoreMock()
	old := cachedTrack()
	require.NoError(t, store.Save(context.Background(), old))
	svc := NewService(context.Background(), store, &cacheProvider{host: "source.test"}, &cacheProvider{host: "other.test"}).(*service)
	response, err := svc.FindLinks(context.Background(), &Message{URL: old.URL, UserID: 20, ChatID: old.ChatID})
	require.NoError(t, err)
	svc.workers.Wait()
	fresh, err := store.FindByURL(context.Background(), old.Provider, old.URL)
	require.NoError(t, err)
	require.Equal(t, "Track — Artist", fresh.Title)
	require.Equal(t, "https://other.test/track/found", fresh.Links[1].URL)
	require.Equal(t, old.FirstSeenAt, fresh.FirstSeenAt)
	require.Equal(t, old.UserID, fresh.UserID)
	require.Equal(t, "https://other.test/track/2", response.Links[1].URL)
}

func TestRefreshFailureAndDefinitiveNotFound(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		sourceErr, searchErr error
		wantLinks            int
	}{
		{name: "temporary search failure", searchErr: errors.New("upstream unavailable"), wantLinks: 2},
		{name: "definitive not found", searchErr: provider.ErrURLNotFound, wantLinks: 1},
		{name: "source failure", sourceErr: errors.New("source unavailable"), wantLinks: 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStoreMock()
			old := cachedTrack()
			require.NoError(t, store.Save(context.Background(), old))
			source := &cacheProvider{host: old.Provider, title: func(string) (string, error) { return "Track", tc.sourceErr }}
			other := &cacheProvider{host: "other.test", search: func(string) (string, error) { return "", tc.searchErr }}
			svc := NewService(context.Background(), store, source, other).(*service)
			_, err := svc.FindLinks(context.Background(), &Message{URL: old.URL})
			require.NoError(t, err)
			svc.workers.Wait()
			got, err := store.FindByURL(context.Background(), old.Provider, old.URL)
			require.NoError(t, err)
			require.Len(t, got.Links, tc.wantLinks)
			if tc.sourceErr != nil {
				require.Equal(t, old.CheckedAt, got.CheckedAt)
			}
		})
	}
}

type brokenCache struct {
	Store
	err error
}

func (s brokenCache) FindByURL(context.Context, string, string) (*Message, error) { return nil, s.err }

func TestCacheMissAndStorageFailureStillResolveTrack(t *testing.T) {
	for _, tc := range []struct {
		name, status string
		err          error
	}{
		{"miss", "miss", ErrCacheMiss}, {"database failure", "error", errors.New("database unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := NewStoreMock()
			svc := NewService(context.Background(), brokenCache{store, tc.err}, &cacheProvider{host: "source.test"}, &cacheProvider{host: "other.test"})
			got, err := svc.FindLinks(context.Background(), &Message{URL: "https://source.test/track/1?utm_source=web", UserID: 20})
			require.NoError(t, err)
			require.Equal(t, tc.status, got.CacheStatus)
			require.Equal(t, "https://source.test/track/1", got.URL)
			require.Len(t, got.Links, 2)
			_, err = store.FindByURL(context.Background(), "other.test", "https://other.test/track/found")
			require.NoError(t, err)
		})
	}
}

func TestCacheDoesNotExposeSenderInAnotherChatOrRefreshAfterShutdown(t *testing.T) {
	store := NewStoreMock()
	old := cachedTrack()
	require.NoError(t, store.Save(context.Background(), old))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source := &cacheProvider{host: old.Provider, title: func(string) (string, error) { t.Error("refresh after shutdown"); return "", nil }}
	svc := NewService(ctx, store, source)
	got, err := svc.FindLinks(context.Background(), &Message{URL: old.URL, ChatID: 99})
	require.NoError(t, err)
	require.Equal(t, "hit", got.CacheStatus)
	require.False(t, got.Refreshing)
	require.Zero(t, got.UserID)
	require.Empty(t, got.UserName)
	require.True(t, got.FirstSeenAt.IsZero())
	unchanged, err := store.FindByURL(context.Background(), old.Provider, old.URL)
	require.NoError(t, err)
	require.Equal(t, old.UserID, unchanged.UserID)
}

func TestCacheLogsDescribeHitAndMissWithoutQueryNoise(t *testing.T) {
	var output bytes.Buffer
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(oldLogger) })
	svc := NewService(context.Background(), NewStoreMock(), &cacheProvider{host: "source.test"}).(*service)
	logged := NewLoggingService(context.Background(), svc)
	_, err := logged.FindLinks(context.Background(), &Message{URL: "just a chat message"})
	require.ErrorIs(t, err, ErrLinkNotFound)
	require.Empty(t, output.String())
	input := &Message{URL: "https://source.test/track/1?utm_source=web&ref_id=secret"}
	_, err = logged.FindLinks(context.Background(), input)
	require.NoError(t, err)
	_, err = logged.FindLinks(context.Background(), input)
	require.NoError(t, err)
	svc.workers.Wait()
	logs := output.String()
	require.Contains(t, logs, "cache=miss")
	require.Contains(t, logs, "cache=hit")
	require.Contains(t, logs, "Ссылка не найдена в кеше")
	for _, noise := range []string{"no rows", "<nil>", "utm_", "ref_id", "secret"} {
		require.NotContains(t, logs, noise)
	}
}

func TestColdSourceFailureIsNotCached(t *testing.T) {
	store := NewStoreMock()
	svc := NewService(context.Background(), store, &cacheProvider{host: "source.test", title: func(string) (string, error) { return "", errors.New("unavailable") }})
	_, err := svc.FindLinks(context.Background(), &Message{URL: "https://source.test/track/1"})
	require.Error(t, err)
	_, err = store.FindByURL(context.Background(), "source.test", "https://source.test/track/1")
	require.ErrorIs(t, err, ErrCacheMiss)
}

func TestRefreshConcurrencyIsBounded(t *testing.T) {
	store := NewStoreMock()
	started, release := make(chan struct{}, 4), make(chan struct{})
	source := &cacheProvider{host: "source.test", title: func(string) (string, error) {
		started <- struct{}{}
		<-release
		return "Track", nil
	}}
	svc := NewService(context.Background(), store, source).(*service)
	defer func() { close(release); svc.workers.Wait() }()
	for i := range 5 {
		m := cachedTrack()
		m.URL = m.URL + string(rune('a'+i))
		m.CacheKey = m.URL
		m.Links = []Link{{Provider: m.Provider, URL: m.URL}}
		require.NoError(t, store.Save(context.Background(), m))
		response, err := svc.FindLinks(context.Background(), &Message{URL: m.URL})
		require.NoError(t, err)
		require.Equal(t, i < 4, response.Refreshing)
	}
	for range 4 {
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("refresh worker did not start")
		}
	}
	svc.mu.Lock()
	count := len(svc.refreshing)
	svc.mu.Unlock()
	require.Equal(t, 4, count)
}

func TestGetListIncludesStoredTrackWithoutDuplicateProviderRows(t *testing.T) {
	store := NewStoreMock()
	m := cachedTrack()
	require.NoError(t, store.Save(context.Background(), m))
	svc := NewService(context.Background(), store)
	list, err := svc.GetList(context.Background(), m.UserID)
	require.NoError(t, err)
	require.Equal(t, m.Title+"\n"+m.URL+"\n", list)
	_, err = svc.GetList(context.Background(), 999)
	require.ErrorIs(t, err, ErrLinksNotFound)
}

func TestCloseStopsNewBackgroundRefreshes(t *testing.T) {
	store := NewStoreMock()
	m := cachedTrack()
	require.NoError(t, store.Save(context.Background(), m))
	svc := NewService(context.Background(), store, &cacheProvider{host: m.Provider})
	require.NoError(t, svc.Close())
	require.NoError(t, svc.Close())
	response, err := svc.FindLinks(context.Background(), &Message{URL: m.URL})
	require.NoError(t, err)
	require.Equal(t, "hit", response.CacheStatus)
	require.False(t, response.Refreshing)
}
