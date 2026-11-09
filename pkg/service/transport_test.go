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

func TestFormatTrackHidesCacheStatus(t *testing.T) {
	for _, refreshing := range []bool{false, true} {
		for _, userID := range []int{0, 10} {
			m := &Message{Title: "Track", CacheStatus: "hit", Refreshing: refreshing, UserID: userID, UserName: "@first",
				Links: []Link{{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc"}}}
			if userID == 0 {
				m.UserName = ""
			}
			text := formatTrack(m, nil)
			require.NotContains(t, text, "Из кеша")
			require.NotContains(t, text, "перепроверяются")
			require.NotContains(t, text, "<i>")
			require.Contains(t, text, ">Spotify</a>")
			if userID == 0 {
				require.NotContains(t, text, "Впервые прислал")
				require.True(t, strings.HasSuffix(text, "</a>"))
			} else {
				require.Contains(t, text, "\n\nВпервые прислал: @first")
			}
		}
	}
}

func TestFormatTrackShowsImportedDeletedAccountOnFirstUse(t *testing.T) {
	m := &Message{Title: "Track", SeenBefore: true, UserName: "Deleted Account",
		FirstSeenAt: time.Date(2019, 7, 2, 9, 25, 35, 0, time.UTC)}
	text := formatTrack(m, nil)
	require.Contains(t, text, "Впервые прислал: Deleted Account · 02.07.2019 12:25 МСК")
	require.NotContains(t, text, "Из кеша")
}

type transportService struct {
	response     *Message
	err          error
	input        *Message
	listResponse string
	listErr      error
	listCalls    int
	listUserID   int
}

func (s *transportService) Close() error { return nil }

func (s *transportService) FindLinks(ctx context.Context, m *Message) (*Message, error) {
	s.input = m
	return s.response, s.err
}
func (s *transportService) GetList(_ context.Context, userID int) (string, error) {
	s.listCalls++
	s.listUserID = userID
	return s.listResponse, s.listErr
}

func TestTelegramHandlerRecordsSenderAndRepliesWithoutButtons(t *testing.T) {
	svc := &transportService{response: &Message{Title: "Track", Links: []Link{{Provider: "open.spotify.com", URL: "https://open.spotify.com/track/abc"}}}}
	handler := NewTelegramHandler(svc, nil, "tbot")
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
	NewTelegramHandler(svc, nil, "tbot").Serve(w, r)
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
	NewTelegramHandler(svc, ServiceIcons{"open.spotify.com": "123"}, "tbot").Serve(w, r)
	require.Equal(t, "HTML", w.ParseMode)
	require.Contains(t, w.Text, `<tg-emoji emoji-id="123">🎧</tg-emoji>`)
}

func TestTelegramCommandsInPrivateChatsAndGroups(t *testing.T) {
	for _, command := range []string{"start", "help", "ping", "list", "chatid"} {
		for _, suffix := range []string{"", " ", " argument", "@tbot", "@TBOT", "@tbot argument", "@tbot\nargument"} {
			t.Run(command+suffix, func(t *testing.T) {
				svc := &transportService{listResponse: "Track\nhttps://example.com/track/1\n"}
				text := "/" + command + suffix
				if command == "list" {
					text = strings.ReplaceAll(text, "argument", "1")
				}
				r := telegramCommandRequest(text)
				if strings.Contains(suffix, "@") {
					r.Message.Chat.Type = "supergroup"
				}
				w := &telegram.ResponseWriter{}
				NewTelegramHandler(svc, nil, "tbot").Serve(w, r)
				require.Equal(t, 12, w.ReplyToMessageID)
				require.Equal(t, text, r.Message.Text)
				require.Nil(t, svc.input, "commands must not search for tracks")
				switch command {
				case "start", "help":
					require.Equal(t, commandHelp, w.Text)
				case "ping":
					require.Contains(t, w.Text, "pong 🏓")
					require.Contains(t, w.Text, "Время с отправки:")
					require.Contains(t, w.Text, "Бот работает:")
				case "list":
					require.Equal(t, svc.listResponse, w.Text)
					require.Equal(t, 1, svc.listCalls)
					require.Equal(t, 10, svc.listUserID)
					require.True(t, w.DisableWebPagePreview)
				case "chatid":
					require.Equal(t, "ID чата: 42", w.Text)
				}
			})
		}
	}
}

func TestTelegramCommandsAddressedToOtherBotsAreIgnored(t *testing.T) {
	for _, command := range []string{"start", "help", "ping", "list", "chatid", "unknown"} {
		t.Run(command, func(t *testing.T) {
			svc := &transportService{}
			w := &telegram.ResponseWriter{}
			NewTelegramHandler(svc, nil, "tbot").Serve(w, telegramCommandRequest("/"+command+"@other_bot"))
			require.Empty(t, w.Text)
			require.Zero(t, w.ReplyToMessageID)
			require.Nil(t, svc.input)
			require.Zero(t, svc.listCalls)
		})
	}
}

func TestTelegramListEmptyAndError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{name: "empty", err: ErrLinksNotFound, want: "Вы ещё не присылали треки."},
		{name: "database error", err: errors.New("private database error"), want: "Не удалось загрузить список треков."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := &transportService{listErr: tc.err}
			w := &telegram.ResponseWriter{}
			NewTelegramHandler(svc, nil, "tbot").Serve(w, telegramCommandRequest("/list@tbot"))
			require.Equal(t, tc.want, w.Text)
			require.NotContains(t, w.Text, "private")
			require.Equal(t, 10, svc.listUserID)
		})
	}
}

func TestTelegramUnknownCommandGivesHelp(t *testing.T) {
	w := &telegram.ResponseWriter{}
	NewTelegramHandler(&transportService{}, nil, "tbot").Serve(w, telegramCommandRequest("/unknown@tbot"))
	require.Equal(t, "Неизвестная команда. Используйте /help.", w.Text)
}

func TestTelegramHandlerIgnoresMissingMessages(t *testing.T) {
	for _, r := range []*telegram.Request{nil, {}} {
		w := &telegram.ResponseWriter{}
		NewTelegramHandler(&transportService{}, nil, "tbot").Serve(w, r)
		require.Empty(t, w.Text)
	}
}

func TestFormatPingShowsApproximateMessageAgeAndUptime(t *testing.T) {
	now := time.Date(2026, 11, 9, 12, 0, 2, 400_000_000, time.UTC)
	startedAt := now.Add(-2*time.Hour - 3*time.Minute - 4*time.Second - 800*time.Millisecond)
	text := formatPing(now, startedAt, int(now.Add(-2400*time.Millisecond).Unix()))
	require.Equal(t, "pong 🏓\nВремя с отправки: ≈2 с (точность 1 с)\nБот работает: 2 ч 3 мин 4 с", text)
	text = formatPing(now, now.Add(time.Hour), int(now.Add(time.Minute).Unix()))
	require.Equal(t, "pong 🏓\nВремя с отправки: ≈0 с (точность 1 с)\nБот работает: 0 ч 0 мин 0 с", text)
	text = formatPing(now, now, 0)
	require.Equal(t, "pong 🏓\nБот работает: 0 ч 0 мин 0 с", text)
}

func telegramCommandRequest(text string) *telegram.Request {
	entities := []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(strings.Fields(text)[0])}}
	return &telegram.Request{Update: tgbotapi.Update{Message: &tgbotapi.Message{
		MessageID: 12, Text: text, Date: int(time.Now().Unix()), Entities: &entities,
		Chat: &tgbotapi.Chat{ID: 42, Type: "private"}, From: &tgbotapi.User{ID: 10, UserName: "first"},
	}}}
}
