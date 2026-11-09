package history

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/service"
)

func htmlMessage(id int, classes, author, body string) string {
	return fmt.Sprintf(`<div class="message default %s" id="message%d">%s<div class="body">
		<div class="date details" title="2 July 2019, 12:25:35"></div>%s</div></div>`, classes, id, author, body)
}

func TestExtractHTMLKeepsAuthorsAcrossPagesAndDoesNotScrapePreviews(t *testing.T) {
	dir := t.TempDir()
	author := `<div class="userpic_wrap"><img src="photos/author_10.jpg"></div>`
	first := htmlMessage(1, "", author, `<div class="from_name">First &amp; Last</div>
		<div class="text"><a href="https://music.yandex.ru/album/1/track/2?utm_source=web">track</a></div>
		<div class="media_wrap"><a class="webpage_preview" href="https://music.yandex.com/album/1/track/2"><div class="webpage_title">Song</div><div class="webpage_description">https://ads.test/unrelated</div></a></div>`)
	second := htmlMessage(2, "joined", "", `<div class="text">https://youtu.be/abc?si=share<br>https://example.test/page?keep=1</div>`)
	bot := htmlMessage(3, "", `<div class="userpic_wrap"><img src="photos/author_99.jpg"></div>`, `<div class="from_name">Bot</div><div class="text"><a href="https://open.spotify.com/track/from-bot">link</a></div>`)
	botJoined := htmlMessage(4, "joined", "", `<div class="text">https://example.test/bot</div>`)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "messages.html"), []byte(`<title>Music</title><div class="history">`+first+`</div>`), 0600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "messages2.html"), []byte(`<div class="history">`+second+bot+botJoined+`</div>`), 0600))
	archive, report, err := ExtractHTML(dir, ExtractOptions{ChatID: -10042, Timezone: "Europe/Moscow", ExcludeUsers: map[int]bool{99: true}})
	require.NoError(t, err)
	require.Equal(t, 4, report.Messages)
	require.Equal(t, 2, report.ExcludedMessages)
	require.Zero(t, report.UnknownAuthors)
	require.Len(t, archive.Shares, 3)
	require.Equal(t, "Music", archive.ChatName)
	require.Equal(t, "Song", archive.Shares[0].Title)
	require.Equal(t, "https://music.yandex.com/album/1/track/2", archive.Shares[0].URL)
	for _, share := range archive.Shares {
		require.Equal(t, 10, share.UserID)
		require.Equal(t, "First & Last", share.UserName)
		require.Equal(t, int64(-10042), share.ChatID)
		require.Equal(t, time.Date(2019, 7, 2, 9, 25, 35, 0, time.UTC), share.SentAt)
		require.NotContains(t, share.URL, "ads.test")
	}
	require.Equal(t, 2, Summarize(archive).MusicTracks)
	require.Equal(t, 3, Summarize(archive).UniqueLinks)
}

func TestExtractHTMLReportsUnknownAuthorsAndBadDates(t *testing.T) {
	dir := t.TempDir()
	unknown := htmlMessage(1, "", "", `<div class="from_name">Unknown</div><div class="text">https://example.test/a</div>`)
	author := `<div class="userpic_wrap"><img src="photos/author_10.jpg"></div>`
	badDate := strings.Replace(htmlMessage(2, "", author, `<div class="from_name">Known</div><div class="text">https://example.test/b</div>`), "2 July 2019, 12:25:35", "invalid", 1)
	path := filepath.Join(dir, "messages.html")
	require.NoError(t, os.WriteFile(path, []byte(`<div class="history">`+unknown+badDate+`</div>`), 0600))
	archive, report, err := ExtractHTML(path, ExtractOptions{Timezone: "Europe/Moscow"})
	require.NoError(t, err)
	require.Len(t, archive.Shares, 1)
	require.Zero(t, archive.Shares[0].UserID)
	require.Equal(t, "Unknown", archive.Shares[0].UserName)
	require.Equal(t, 1, report.UnknownAuthors)
	require.Equal(t, 1, report.InvalidDates)
	_, _, err = ExtractHTML(path, ExtractOptions{Timezone: "invalid/timezone"})
	require.Error(t, err)
	_, _, err = ExtractHTML(t.TempDir(), ExtractOptions{Timezone: "UTC"})
	require.Error(t, err)
}

func TestExtractHTMLDateFormats(t *testing.T) {
	for _, tc := range []struct {
		name, date, timezone, utc string
	}{
		{"legacy", "2 July 2019, 12:25:35", "Europe/Moscow", "2019-07-02T09:25:35Z"},
		{"positive offset", "02.07.2019 12:25:35 UTC+03:00", "UTC", "2019-07-02T09:25:35Z"},
		{"negative offset", "02.07.2019 12:25:35 UTC-07:00", "Europe/Moscow", "2019-07-02T19:25:35Z"},
		{"half hour", "02.07.2019 12:25:35 UTC+05:30", "UTC", "2019-07-02T06:55:35Z"},
		{"UTC", "02.07.2019 12:25:35 UTC", "Europe/Moscow", "2019-07-02T12:25:35Z"},
		{"zero offset", "02.07.2019 12:25:35 UTC+00:00", "Europe/Moscow", "2019-07-02T12:25:35Z"},
		{"bad date", "31.02.2019 12:25:35 UTC+03:00", "UTC", ""},
		{"bad offset", "02.07.2019 12:25:35 UTC+abc", "UTC", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			message := htmlMessage(1, "", `<div class="userpic_wrap"><img src="photos/author_10.jpg"></div>`,
				`<div class="from_name">First</div><div class="text">https://music.yandex.com/album/1/track/2</div>`)
			message = strings.Replace(message, "2 July 2019, 12:25:35", tc.date, 1)
			path := filepath.Join(t.TempDir(), "messages.html")
			require.NoError(t, os.WriteFile(path, []byte(`<div class="history">`+message+`</div>`), 0600))
			archive, report, err := ExtractHTML(path, ExtractOptions{Timezone: tc.timezone})
			require.NoError(t, err)
			if tc.utc == "" {
				require.Equal(t, 1, report.InvalidDates)
				require.Empty(t, archive.Shares)
				return
			}
			require.Zero(t, report.InvalidDates)
			require.Len(t, archive.Shares, 1)
			require.Equal(t, tc.utc, archive.Shares[0].SentAt.Format(time.RFC3339))
		})
	}
}

func TestArchiveValidationAndBinding(t *testing.T) {
	share := service.Share{UserID: 10, UserName: "First", SentAt: time.Now().UTC(), URL: "https://music.yandex.ru/album/1/track/2?utm_source=web"}
	archive := &Archive{Version: Version, Shares: []service.Share{share}}
	require.NoError(t, archive.Validate(false))
	require.Equal(t, "https://music.yandex.com/album/1/track/2", archive.Shares[0].URL)
	require.Equal(t, "music.yandex.com", archive.Shares[0].Provider)
	require.Error(t, archive.Validate(true))
	archive.BindChat(-10042)
	archive.BindChat(-10099)
	require.Equal(t, int64(-10042), archive.Shares[0].ChatID)
	require.NoError(t, archive.Validate(true))
	archive.Shares[0].Provider = "other.test"
	require.Error(t, archive.Validate(true))
	_, err := Read(strings.NewReader(`{"version":999}`))
	require.Error(t, err)
	_, err = Read(strings.NewReader(`{"version":1,"aliases":[{"cache_key":"missing"}]}`))
	require.Error(t, err)
	_, err = Read(strings.NewReader(`{"version":1,"shares":[{"user_id":10}]}`))
	require.Error(t, err)
	_, err = Read(strings.NewReader("invalid JSON"))
	require.Error(t, err)
}

func TestMusicTrackClassificationDoesNotIndexArtistsOrPlaylists(t *testing.T) {
	for _, raw := range []string{
		"https://music.yandex.com/album/1/track/2", "https://music.yandex.com/track/2",
		"https://open.spotify.com/track/abc", "https://music.youtube.com/watch?v=abc",
		"https://music.apple.com/us/album/title/1?i=2", "https://music.apple.com/us/song/title/2",
	} {
		require.True(t, IsMusicTrack(raw), raw)
	}
	for _, raw := range []string{
		"https://music.yandex.com/album/1", "https://music.apple.com/us/artist/name/1",
		"https://open.spotify.com/playlist/abc", "https://example.test/track/abc", "https://music.youtube.com/playlist?list=abc",
	} {
		require.False(t, IsMusicTrack(raw), raw)
	}
}
