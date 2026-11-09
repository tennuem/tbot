package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/require"
	"github.com/tennuem/telegram"
)

func TestFormatTrackUsesEscapedHTMLLinks(t *testing.T) {
	m := &Message{Title: `<Track & "Artist">`, Provider: "music.yandex.com", CacheStatus: "hit", Refreshing: true,
		UserID: 10, UserName: "@first<script>", FirstSeenAt: time.Date(2026, 11, 9, 10, 34, 0, 0, time.UTC),
		Links: []Link{
			{Provider: "music.yandex.com", URL: "https://music.yandex.com/album/1/track/2"},
			{Provider: "music.apple.com", URL: "https://music.apple.com/us/album/name/1?i=2&at=tracking"},
			{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc?si=tracking"},
			{Provider: "evil.test", URL: "javascript:alert(1)", Name: "Bad"},
		}}
	text := formatTrack(m, nil)
	require.Contains(t, text, "<b>&lt;Track &amp; &#34;Artist&#34;&gt;</b>")
	require.Contains(t, text, `<a href="https://open.spotify.com/track/abc">Spotify</a>`)
	require.Contains(t, text, `<a href="https://music.apple.com/us/album/name/1?i=2">Apple Music</a>`)
	require.Contains(t, text, "09.11.2026 13:34 МСК")
	require.Contains(t, text, "@first&lt;script&gt;")
	for _, absent := range []string{"javascript:", "si=", "at=", "music.yandex.com", "<script>"} {
		require.NotContains(t, text, absent)
	}
}

type transportService struct {
	response *Message
	err      error
	input    *Message
}

func (s *transportService) Close() error { return nil }

func (s *transportService) FindLinks(ctx context.Context, m *Message) (*Message, error) {
	s.input = m
	return s.response, s.err
}
func (s *transportService) GetList(context.Context, int) (string, error) { return "", ErrLinksNotFound }

func TestTelegramHandlerRecordsSenderAndRepliesWithoutButtons(t *testing.T) {
	svc := &transportService{response: &Message{Title: "Track", Links: []Link{{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc"}}}}
	handler := NewTelegramHandler(svc, nil)
	r := &telegram.Request{Update: tgbotapi.Update{Message: &tgbotapi.Message{
		MessageID: 12, Text: "https://music.yandex.ru/album/1/track/2", Date: 1794218400,
		Chat: &tgbotapi.Chat{ID: 42}, From: &tgbotapi.User{ID: 10, UserName: "first"},
	}}}
	w := &telegram.ResponseWriter{}
	handler.Serve(w, r)
	require.Equal(t, 12, w.ReplyToMessageID)
	require.Equal(t, "HTML", w.ParseMode)
	require.True(t, w.DisableWebPagePreview)
	require.Nil(t, w.ReplyMarkup)
	require.Contains(t, w.Text, ">Spotify</a>")
	require.Equal(t, 10, svc.input.UserID)
	require.Equal(t, int64(42), svc.input.ChatID)
	require.Equal(t, "@first", svc.input.UserName)
	require.Equal(t, int64(r.Message.Date), svc.input.FirstSeenAt.Unix())
}

func TestTelegramHandlerDoesNotExposeInternalErrors(t *testing.T) {
	svc := &transportService{err: errors.New("secret internal API error")}
	r := &telegram.Request{Update: tgbotapi.Update{Message: &tgbotapi.Message{Text: "https://example.com", Chat: &tgbotapi.Chat{ID: 42}, From: &tgbotapi.User{ID: 10}}}}
	w := &telegram.ResponseWriter{}
	NewTelegramHandler(svc, nil).Serve(w, r)
	require.NotContains(t, w.Text, "secret")
	require.Contains(t, w.Text, "Попробуйте")
}

func TestFormatTrackAddsLogosBesideLinks(t *testing.T) {
	m := &Message{Title: "Track", Provider: "music.yandex.com", Links: []Link{
		{Provider: "music.yandex.com", URL: "https://music.yandex.com/album/1/track/2"},
		{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc"},
		{Provider: "music.apple.com", URL: "https://music.apple.com/us/album/name/1?i=2"},
		{Provider: "music.youtube.com", URL: "https://music.youtube.com/watch?v=abc"},
	}}
	icons := ServiceIcons{"open.spotify.com": "123", "music.yandex.com": "456", "music.youtube.com": "789"}
	text := formatTrack(m, icons)
	require.Contains(t, text, `<tg-emoji emoji-id="123">🎧</tg-emoji> <a href="https://open.spotify.com/track/abc">Spotify</a>`)
	require.Contains(t, text, `<tg-emoji emoji-id="789">▶️</tg-emoji> <a href="https://music.youtube.com/watch?v=abc">YouTube Music</a>`)
	require.Contains(t, text, `<a href="https://music.apple.com/us/album/name/1?i=2">Apple Music</a>`)
	require.NotContains(t, text, `emoji-id="456"`)
	require.Equal(t, 2, strings.Count(text, "<tg-emoji"))
	// custom_emoji and text_link entities must not overlap.
	require.NotContains(t, text, `">Spotify<tg-emoji`)
}

func TestFormatTrackRejectsInvalidEmojiIDs(t *testing.T) {
	m := &Message{Title: "Track", Links: []Link{{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc"}}}
	for _, id := range []string{"", "0", "-1", "+123", "00123", `123"><script>`, "18446744073709551616"} {
		t.Run(id, func(t *testing.T) {
			text := formatTrack(m, ServiceIcons{"open.spotify.com": id})
			require.NotContains(t, text, "tg-emoji")
			require.Contains(t, text, ">Spotify</a>")
		})
	}
}

func TestTelegramHandlerUsesConfiguredLogos(t *testing.T) {
	svc := &transportService{response: &Message{Title: "Track", Links: []Link{{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc"}}}}
	r := &telegram.Request{Update: tgbotapi.Update{Message: &tgbotapi.Message{Text: "https://music.yandex.com/album/1/track/2", Chat: &tgbotapi.Chat{ID: 42}, From: &tgbotapi.User{ID: 10}}}}
	w := &telegram.ResponseWriter{}
	NewTelegramHandler(svc, ServiceIcons{"open.spotify.com": "123"}).Serve(w, r)
	require.Equal(t, "HTML", w.ParseMode)
	require.Contains(t, w.Text, `<tg-emoji emoji-id="123">🎧</tg-emoji>`)
}
