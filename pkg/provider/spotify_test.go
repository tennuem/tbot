package provider

import (
	"testing"

	"github.com/go-kit/kit/log"
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
	p := NewSpotifyProvider(log.NewNopLogger(), "cid", "csecret")
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
	p := NewSpotifyProvider(log.NewNopLogger(), "cid", "csecret")
	for _, c := range testCases {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
