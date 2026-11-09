package provider

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYoutubeProviderGetTitle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><meta property="og:video:tag" content="Highway to Hell — AC/DC"></html>`))
	}))
	defer ts.Close()
	testData := []struct {
		in  string
		out string
	}{
		{
			ts.URL + "/watch?v=ikFFVfObwss&feature=share",
			"Highway to Hell — AC/DC",
		},
	}
	p := youtubeProvider{
		host:   ts.URL,
		client: ts.Client(),
	}
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestYoutubeProviderHTTPFailuresAreNotMissingTracks(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusForbidden, http.StatusNotFound} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = w.Write([]byte("Service Unavailable"))
			}))
			defer ts.Close()
			p := youtubeProvider{host: ts.URL, client: ts.Client()}
			link, err := p.GetURL("Track")
			require.Empty(t, link)
			require.Error(t, err)
			require.Contains(t, err.Error(), fmt.Sprint(status))
			require.NotErrorIs(t, err, ErrURLNotFound)
			title, err := p.GetTitle(ts.URL)
			require.Empty(t, title)
			require.Error(t, err)
			require.Contains(t, err.Error(), fmt.Sprint(status))
			require.NotErrorIs(t, err, ErrTitleNotFound)
		})
	}
}

func TestYoutubeProviderSuccessfulEmptySearchIsMissingTrack(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>No results</html>`))
	}))
	defer ts.Close()
	p := youtubeProvider{host: ts.URL, client: ts.Client()}
	link, err := p.GetURL("Track")
	require.Empty(t, link)
	require.ErrorIs(t, err, ErrURLNotFound)
}

func TestYoutubeProviderGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><script>var ytInitialData = {"contents":{"sectionListRenderer":{"contents":[{"itemSectionRenderer":{"contents":[{"videoRenderer":{"videoId":"ikFFVfObwss"}}]}}]}}};</script></html>`))
	}))
	defer ts.Close()
	testData := []struct {
		in  string
		out string
	}{
		{
			"Highway to Hell — AC/DC",
			"https://music.youtube.com/watch?v=ikFFVfObwss",
		},
	}
	p := youtubeProvider{
		host:   ts.URL,
		client: ts.Client(),
	}
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
