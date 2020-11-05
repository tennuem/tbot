package provider

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpotifyProviderGetTitle(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{
			"https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7",
			"Babushka Boi — A$AP Rocky",
		},
	}
	p := NewSpotifyProvider(
		context.Background(),
		os.Getenv("TBOT_SPOTIFY_CLIENT_ID"),
		os.Getenv("TBOT_SPOTIFY_CLIENT_SECRET"),
	)
	for _, c := range testCases {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestSpotifyProviderGetURL(t *testing.T) {
	testCases := []struct {
		in  string
		out string
	}{
		{
			"Babushka Boi — A$AP Rocky",
			"https://open.spotify.com/track/643PW82aBMUa1FiWi5VQY7",
		},
	}
	p := NewSpotifyProvider(
		context.Background(),
		os.Getenv("TBOT_SPOTIFY_CLIENT_ID"),
		os.Getenv("TBOT_SPOTIFY_CLIENT_SECRET"),
	)
	for _, c := range testCases {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
