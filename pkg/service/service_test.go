package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/provider"
)

func TestFindLinks(t *testing.T) {
	testData := []struct {
		in  *Message
		out *Message
	}{
		{
			&Message{URL: "https://example.com/track/643PW82aBMUa1FiWi5VQY7"},
			&Message{
				URL:   "https://example.com/track/643PW82aBMUa1FiWi5VQY7",
				Title: "test_title",
			},
		},
	}
	svc := NewService(context.Background(), NewStoreMock(), provider.NewMockProvider())
	for k, c := range testData {
		res, err := svc.FindLinks(context.Background(), c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out.Title, res.Title, fmt.Sprintf("case-%d", k))
		assert.Equal(t, c.out.URL, res.URL)
		assert.Equal(t, "miss", res.CacheStatus)
	}
}

func TestExtractLink(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{"foo", ""},
		{"foo https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7", "https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7"},
		{"foo\nhttps://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7", "https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7"},
	}
	for k, c := range testCases {
		res, err := extractLink(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res, fmt.Sprintf("case-%d", k))
	}
}

func TestParseURL(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"https://music.yandex.ru/album/1/track/2", "https://music.yandex.com/album/1/track/2"},
		{"https://music.yandex.com/album/1/track/2", "https://music.yandex.com/album/1/track/2"},
		{"https://example.ru/track?source=music.yandex.ru", "https://example.ru/track?source=music.yandex.ru"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			u, err := parseURL(tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.want, u.String())
		})
	}
}
