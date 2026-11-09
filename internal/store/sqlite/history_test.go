package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/internal/history"
	"github.com/tennuem/tbot/pkg/service"
)

func historicalShare() service.Share {
	return service.Share{MessageID: 123, ChatID: -10042, UserID: 20, UserName: "Original",
		SentAt: time.Date(2019, 7, 2, 9, 25, 35, 0, time.UTC), Provider: "music.yandex.com",
		URL: "https://music.yandex.com/album/1/track/2", Title: "Old title"}
}

func TestHistoryImportIsIdempotentAndKeepsFreshCache(t *testing.T) {
	s := openTestStore(t, ":memory:")
	current := cacheMessage()
	require.NoError(t, s.Save(context.Background(), current))
	share := historicalShare()
	duplicate := share
	duplicate.MessageID++
	duplicate.UserID, duplicate.UserName = 30, "Second"
	duplicate.SentAt = duplicate.SentAt.Add(time.Hour)
	other := share
	other.MessageID += 2
	other.URL, other.Provider = "https://example.test/page", "example.test"
	archive := &history.Archive{Version: history.Version, Shares: []service.Share{duplicate, share, other}}
	result, err := s.ImportHistory(context.Background(), archive)
	require.NoError(t, err)
	require.Equal(t, int64(3), result.SharesAdded)
	require.Equal(t, 1, result.TrackLinks)
	got, err := s.FindByURL(context.Background(), share.Provider, share.URL)
	require.NoError(t, err)
	require.Equal(t, current.Title, got.Title)
	require.Equal(t, current.CheckedAt, got.CheckedAt)
	require.Equal(t, current.Links, got.Links)
	require.Equal(t, share.UserID, got.UserID)
	require.Equal(t, share.SentAt, got.FirstSeenAt)
	require.Equal(t, share.ChatID, got.ChatID)
	require.Equal(t, share.MessageID, got.MessageID)
	for _, id := range []int{10, 20, 30} {
		list, err := s.FindByUser(context.Background(), id)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, id, list[0].UserID)
	}
	result, err = s.ImportHistory(context.Background(), archive)
	require.NoError(t, err)
	require.Zero(t, result.SharesAdded)
	require.Equal(t, int64(3), result.Duplicates)
	exported, err := s.ExportHistory(context.Background())
	require.NoError(t, err)
	require.Len(t, exported.Shares, 4)
	require.Len(t, exported.Cache, 1)
}

func TestHistoryRoundTripPreservesHistoricalAliasesAndMergesConflicts(t *testing.T) {
	ctx := context.Background()
	source := openTestStore(t, ":memory:")
	m := cacheMessage()
	require.NoError(t, source.Save(ctx, m))
	fresh := service.CloneMessage(m)
	fresh.CheckedAt = fresh.CheckedAt.Add(time.Second)
	fresh.Links[1].URL = "https://open.spotify.com/track/new"
	require.NoError(t, source.Save(ctx, fresh))
	share := historicalShare()
	require.NoError(t, source.RecordShare(ctx, share))
	archive, err := source.ExportHistory(ctx)
	require.NoError(t, err)
	destination := openTestStore(t, ":memory:")
	conflict := service.CloneMessage(m)
	conflict.CacheKey, conflict.Provider, conflict.URL = "existing", "open.spotify.com", m.Links[1].URL
	conflict.Links = conflict.Links[1:]
	require.NoError(t, destination.Save(ctx, conflict))
	_, err = destination.ImportHistory(ctx, archive)
	require.NoError(t, err)
	for _, link := range []service.Link{m.Links[0], m.Links[1], fresh.Links[1]} {
		got, err := destination.FindByURL(ctx, link.Provider, link.URL)
		require.NoError(t, err)
		require.Equal(t, historicalShare().UserID, got.UserID)
		require.ElementsMatch(t, fresh.Links, got.Links)
	}
	var count int
	require.NoError(t, destination.db.QueryRow("SELECT COUNT(*) FROM cache_entries").Scan(&count))
	require.Equal(t, 1, count)
}

func TestHistoryImportRollsBackOnWriteFailure(t *testing.T) {
	s := openTestStore(t, ":memory:")
	_, err := s.db.Exec(`CREATE TRIGGER fail_share BEFORE INSERT ON history_shares WHEN NEW.user_id=99 BEGIN SELECT RAISE(ABORT,'test failure'); END;`)
	require.NoError(t, err)
	first := historicalShare()
	second := first
	second.UserID, second.MessageID = 99, 124
	archive := &history.Archive{Version: history.Version, Cache: []service.Message{*cacheMessage()}, Shares: []service.Share{first, second}}
	_, err = s.ImportHistory(context.Background(), archive)
	require.Error(t, err)
	for _, table := range []string{"cache_entries", "cache_aliases", "history_shares"} {
		var count int
		require.NoError(t, s.db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count))
		require.Zero(t, count)
	}
	archive.Shares[0].URL = "javascript:invalid"
	_, err = s.ImportHistory(context.Background(), archive)
	require.Error(t, err)
}

func TestReadOnlyHistoryExportDoesNotCreateMissingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.db")
	_, err := OpenReadOnly(path)
	require.Error(t, err)
	path = filepath.Join(t.TempDir(), "existing.db")
	s := openTestStore(t, path)
	require.NoError(t, s.Save(context.Background(), cacheMessage()))
	_, err = s.db.Exec("DROP TABLE history_shares") // Export from a version before event history existed.
	require.NoError(t, err)
	require.NoError(t, s.Close())
	readOnly, err := OpenReadOnly(path)
	require.NoError(t, err)
	defer readOnly.Close()
	archive, err := readOnly.ExportHistory(context.Background())
	require.NoError(t, err)
	require.Len(t, archive.Cache, 1)
	require.Error(t, readOnly.RecordShare(context.Background(), historicalShare()))
}

func TestImportedPartialCacheDoesNotExposeAnotherChatsSender(t *testing.T) {
	s := openTestStore(t, ":memory:")
	share := historicalShare()
	_, err := s.ImportHistory(context.Background(), &history.Archive{Version: history.Version, Shares: []service.Share{share}})
	require.NoError(t, err)
	svc := service.NewService(context.Background(), s, &historyProvider{})
	defer svc.Close()
	result, err := svc.FindLinks(context.Background(), &service.Message{URL: share.URL, ChatID: 99, UserID: 30, UserName: "Second", MessageID: 2})
	require.NoError(t, err)
	require.False(t, result.SeenBefore)
	require.Zero(t, result.UserID)
	require.Empty(t, result.UserName)
	require.True(t, result.FirstSeenAt.IsZero())
	cached, err := s.FindByURL(context.Background(), share.Provider, share.URL)
	require.NoError(t, err)
	require.Equal(t, share.ChatID, cached.ChatID)
	require.Equal(t, share.UserID, cached.UserID)
}

func TestImportedHistoryIsRecognizedOnFirstUseAndRecordsEverySender(t *testing.T) {
	s := openTestStore(t, ":memory:")
	share := historicalShare()
	_, err := s.ImportHistory(context.Background(), &history.Archive{Version: history.Version, Shares: []service.Share{share}})
	require.NoError(t, err)
	svc := service.NewService(context.Background(), s, &historyProvider{})
	defer svc.Close()
	input := &service.Message{URL: share.URL, UserID: 30, UserName: "Second", ChatID: share.ChatID,
		MessageID: 999, FirstSeenAt: time.Now().UTC()}
	result, err := svc.FindLinks(context.Background(), input)
	require.NoError(t, err)
	require.True(t, result.SeenBefore)
	require.Equal(t, share.UserID, result.UserID)
	input.MessageID++
	result, err = svc.FindLinks(context.Background(), input)
	require.NoError(t, err)
	require.Equal(t, "hit", result.CacheStatus)
	exported, err := s.ExportHistory(context.Background())
	require.NoError(t, err)
	require.Len(t, exported.Shares, 3)
	list, err := s.FindByUser(context.Background(), 30)
	require.NoError(t, err)
	require.Len(t, list, 1)
}

type historyProvider struct{}

func (*historyProvider) Host() string                    { return "music.yandex.com" }
func (*historyProvider) Name() string                    { return "Yandex Music" }
func (*historyProvider) GetTitle(string) (string, error) { return "Resolved track", nil }
func (*historyProvider) GetURL(string) (string, error)   { return "", nil }
