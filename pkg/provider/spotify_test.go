package provider

import (
	"github.com/go-kit/kit/log"
	spotifyAPI "github.com/zmb3/spotify"
	"net/http"
	"net/http/httptest"
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
			"https://open.spotify.com/track/2zYzyRzz6pRmhPzyfMEC8s",
			"Highway to Hell — AC/DC",
		},
	}
	p := spotifyProvider{
		client: &spotifyClientMock{},
		logger: log.NewNopLogger(),
	}
	for _, c := range testCases {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestSpotifyProviderGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><a class="d-track__title" href="/album/2832579/track/694683"></a></html>`))
	}))
	defer ts.Close()
	testCases := []struct {
		in  string
		out string
	}{
		{
			"Highway to Hell — AC/DC",
			"https://open.spotify.com/track/2zYzyRzz6pRmhPzyfMEC8s",
		},
	}
	p := spotifyProvider{
		client: &spotifyClientMock{},
		logger: log.NewNopLogger(),
	}
	for _, c := range testCases {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

type spotifyClientMock struct{}

func (c *spotifyClientMock) GetTrack(id spotifyAPI.ID) (*spotifyAPI.FullTrack, error) {
	return &spotifyAPI.FullTrack{
		SimpleTrack: spotifyAPI.SimpleTrack{
			Artists: []spotifyAPI.SimpleArtist{
				{Name: "AC/DC"},
			},
			Name: "Highway to Hell",
		},
	}, nil
}

func (c *spotifyClientMock) Search(query string, t spotifyAPI.SearchType) (*spotifyAPI.SearchResult, error) {
	return &spotifyAPI.SearchResult{
		Tracks: &spotifyAPI.FullTrackPage{
			Tracks: []spotifyAPI.FullTrack{
				{SimpleTrack: spotifyAPI.SimpleTrack{
					ExternalURLs: map[string]string{
						"spotify": "https://open.spotify.com/track/2zYzyRzz6pRmhPzyfMEC8s",
					},
				}},
			},
		},
	}, nil
}
