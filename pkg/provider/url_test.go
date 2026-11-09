package provider

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeURL(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"https://music.yandex.ru/album/2539712/track/184372?utm_source=web&utm_medium=copy_link&ref_id=secret#fragment", "https://music.yandex.com/album/2539712/track/184372"},
		{"http://music.yandex.kz/album/1/track/2?UTM_SOURCE=web&yclid=x", "https://music.yandex.com/album/1/track/2"},
		{"https://open.spotify.com/intl-de/track/2zYzyRzz6pRmhPzyfMEC8s?si=abc&utm_medium=copy", "https://open.spotify.com/track/2zYzyRzz6pRmhPzyfMEC8s"},
		{"https://www.youtube.com/watch?v=video123&list=album123&t=15&si=abc", "https://music.youtube.com/watch?v=video123"},
		{"https://youtu.be/video123?si=abc", "https://music.youtube.com/watch?v=video123"},
		{"https://music.apple.com/us/album/name/123?i=456&at=affiliate&itsct=share&utm_source=web", "https://music.apple.com/us/album/name/123?i=456"},
		{"https://example.ru/track?z=2&utm_source=web&a=1#fragment", "https://example.ru/track?a=1&z=2"},
		{"https://music.yandex.com/album/1/track/2?", "https://music.yandex.com/album/1/track/2"},
		{"https://[::1]:443/track", "https://[::1]/track"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			u, err := NormalizeURL(tc.raw)
			require.NoError(t, err)
			require.Equal(t, tc.want, u.String())
			second, err := NormalizeURL(u.String())
			require.NoError(t, err)
			require.Equal(t, u.String(), second.String())
		})
	}
}

func TestNormalizeURLRejectsInvalidInput(t *testing.T) {
	for _, raw := range []string{"", "relative/path", "javascript:alert(1)", "https://user:secret@example.com/track", "https://example.com/%xx", "https://example.com/?a=%xx", "https://open.spotify.com:1234/track/abc"} {
		t.Run(raw, func(t *testing.T) {
			u, err := NormalizeURL(raw)
			require.Error(t, err)
			require.Nil(t, u)
			require.NotContains(t, err.Error(), "secret")
		})
	}
}
