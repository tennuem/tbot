package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
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
		logger: log.NewNopLogger(),
	}
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestYoutubeProviderGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><div id="search"><div class="g"><a href="https://www.youtube.com/watch?v=ikFFVfObwss&feature=share"></a></div></div></html>`))
	}))
	defer ts.Close()
	testData := []struct {
		in  string
		out string
	}{
		{
			"Highway to Hell — AC/DC",
			"https://music.youtube.com/watch?v=ikFFVfObwss&feature=share",
		},
	}
	p := youtubeProvider{
		host:   ts.URL,
		logger: log.NewNopLogger(),
	}
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
