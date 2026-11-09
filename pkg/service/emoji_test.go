package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadServiceIconsMatchesEmojiInsteadOfOrder(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/bot123:private-token/getStickerSet", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "TbotMusicServices", r.PostForm.Get("name"))
		io.WriteString(w, `{"ok":true,"result":{"sticker_type":"custom_emoji","stickers":[
			{"emoji":"▶","custom_emoji_id":"400"},
			{"emoji":"🎵","custom_emoji_id":"200"},
			{"emoji":"🎧","custom_emoji_id":"100"},
			{"emoji":"🎶","custom_emoji_id":"300"},
			{"emoji":"🙂","custom_emoji_id":"500"}
		]}}`)
	}))
	defer srv.Close()
	icons, err := loadServiceIcons(context.Background(), srv.Client(), srv.URL, "123:private-token", "TbotMusicServices")
	require.NoError(t, err)
	require.Equal(t, ServiceIcons{
		"open.spotify.com": "100", "music.yandex.com": "200", "music.apple.com": "300", "music.youtube.com": "400",
	}, icons)
}

func TestLoadServiceIconsAllowsPartialSet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":true,"result":{"sticker_type":"custom_emoji","stickers":[
			{"emoji":"🎧","custom_emoji_id":"123"},
			{"emoji":"🎶","custom_emoji_id":"not-an-id"}
		]}}`)
	}))
	defer srv.Close()
	icons, err := loadServiceIcons(context.Background(), srv.Client(), srv.URL, "token", "TbotMusicServices")
	require.NoError(t, err)
	require.Equal(t, ServiceIcons{"open.spotify.com": "123"}, icons)
}

func TestLoadServiceIconsDisabledOrInvalidNameDoesNotCallAPI(t *testing.T) {
	client := &http.Client{Transport: emojiRoundTripper(func(*http.Request) (*http.Response, error) {
		t.Error("unexpected network request")
		return nil, errors.New("unexpected request")
	})}
	for _, name := range []string{"", "https://t.me/addemoji/TbotMusicServices", "pack?name=other", strings.Repeat("a", 65)} {
		icons, err := loadServiceIcons(context.Background(), client, "https://api.telegram.org", "token", name)
		require.Nil(t, icons)
		if name == "" {
			require.NoError(t, err)
		} else {
			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid custom emoji set name")
		}
	}
	icons, err := LoadServiceIcons(context.Background(), "token", "")
	require.NoError(t, err)
	require.Nil(t, icons)
}

func TestLoadServiceIconsErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		status           int
	}{
		{name: "not found", status: http.StatusBadRequest, body: `{"ok":false}`, want: "HTTP 400"},
		{name: "API error", status: http.StatusOK, body: `{"ok":false,"error_code":403,"description":"private-token"}`, want: "code 403"},
		{name: "invalid JSON", status: http.StatusOK, body: "private-token", want: "invalid custom emoji response JSON"},
		{name: "wrong type", status: http.StatusOK, body: `{"ok":true,"result":{"sticker_type":"regular"}}`, want: "not a custom emoji set"},
		{name: "empty set", status: http.StatusOK, body: `{"ok":true,"result":{"sticker_type":"custom_emoji"}}`, want: "no matching service logos"},
		{name: "duplicate fallback", status: http.StatusOK, body: `{"ok":true,"result":{"sticker_type":"custom_emoji","stickers":[{"emoji":"🎧","custom_emoji_id":"123"},{"emoji":"🎧","custom_emoji_id":"456"}]}}`, want: "ambiguous custom emoji"},
		{name: "too large", status: http.StatusOK, body: strings.Repeat("a", (1<<20)+1), want: "too large"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				io.WriteString(w, tc.body)
			}))
			defer srv.Close()
			icons, err := loadServiceIcons(context.Background(), srv.Client(), srv.URL, "private-token", "TbotMusicServices")
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.want)
			require.NotContains(t, err.Error(), "private-token")
			require.Nil(t, icons)
		})
	}
}

func TestLoadServiceIconsNetworkErrorDoesNotExposeToken(t *testing.T) {
	client := &http.Client{Transport: emojiRoundTripper(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("network unavailable")
	})}
	_, err := loadServiceIcons(context.Background(), client, "https://api.telegram.org", "123:private-token", "TbotMusicServices")
	require.Error(t, err)
	require.Contains(t, err.Error(), "network unavailable")
	require.NotContains(t, err.Error(), "private-token")
}

func TestLoadServiceIconsHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := LoadServiceIcons(ctx, "123:private-token", "TbotMusicServices")
	require.ErrorIs(t, err, context.Canceled)
	require.NotContains(t, err.Error(), "private-token")
}

type emojiRoundTripper func(*http.Request) (*http.Response, error)

func (f emojiRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
