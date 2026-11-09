package provider

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	spotifyAPI "github.com/zmb3/spotify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2/clientcredentials"
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
	}
	for _, c := range testCases {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

type recordingSpotifyClient struct {
	spotifyClientMock
	id spotifyAPI.ID
}

func (c *recordingSpotifyClient) GetTrack(id spotifyAPI.ID) (*spotifyAPI.FullTrack, error) {
	c.id = id
	return c.spotifyClientMock.GetTrack(id)
}

func TestSpotifyTrackIDDoesNotIncludeSharingQuery(t *testing.T) {
	c := &recordingSpotifyClient{}
	p := &spotifyProvider{client: c}
	_, err := p.GetTitle("https://open.spotify.com/intl-de/track/2zYzyRzz6pRmhPzyfMEC8s?si=share&utm_source=web")
	require.NoError(t, err)
	require.Equal(t, spotifyAPI.ID("2zYzyRzz6pRmhPzyfMEC8s"), c.id)
	for _, raw := range []string{"https://open.spotify.com/album/123", "https://open.spotify.com/track/", "https://example.com/track/2zYzyRzz6pRmhPzyfMEC8s"} {
		_, err := p.GetTitle(raw)
		require.ErrorIs(t, err, ErrTitleNotFound)
	}
}

func TestSpotifyTokensAreLazyAndSharedByConcurrentRequests(t *testing.T) {
	var tokens atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/token" {
			user, password, ok := r.BasicAuth()
			assert.True(t, ok)
			assert.Equal(t, "client", user)
			assert.Equal(t, "secret", password)
			tokens.Add(1)
			_, _ = io.WriteString(w, `{"access_token":"shared-token","token_type":"Bearer","expires_in":3600}`)
			return
		}
		assert.Equal(t, "Bearer shared-token", r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	client := newSpotifyHTTPClient(context.Background(), clientcredentials.Config{ClientID: "client", ClientSecret: "secret", TokenURL: server.URL + "/token"})
	defer client.CloseIdleConnections()
	require.Zero(t, tokens.Load())
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := client.Get(server.URL + "/track")
			if err == nil {
				err = resp.Body.Close()
			}
			errors <- err
		}()
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
	require.EqualValues(t, 1, tokens.Load())
}

func TestSpotifyTokenFailureReturnsAnErrorWithoutStoppingBot(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"error":"invalid_client"}`)
	}))
	defer server.Close()
	client := newSpotifyHTTPClient(context.Background(), clientcredentials.Config{ClientID: "client", ClientSecret: "secret", TokenURL: server.URL})
	_, err := client.Get(server.URL + "/track")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid_client")
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
