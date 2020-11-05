package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/kit/log"
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
			&Message{URL: "https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7"},
			&Message{
				URL:   "https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7",
				Title: "test_title",
			},
		},
	}
	svc := NewService(
		NewStoreMock(),
		map[string]provider.Provider{
			"open.spotify.com": provider.NewMockProvider(),
		},
		log.NewNopLogger(),
	)
	for k, c := range testData {
		res, err := svc.FindLinks(context.Background(), c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res, fmt.Sprintf("case-%d", k))
	}
}

func TestFindProvider(t *testing.T) {
	testCases := []struct {
		in  string
		out provider.Provider
	}{
		{"https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7", provider.NewMockProvider()},
		{"https://link.spotify.com/track/643PW82aBMUa1FiWi5VQY7", provider.NewMockProvider()},
	}
	svc := &service{providers: map[string]provider.Provider{
		"open.spotify.com": provider.NewMockProvider(),
	}}
	for k, c := range testCases {
		res, err := svc.findProvider(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res, fmt.Sprintf("case-%d", k))
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
