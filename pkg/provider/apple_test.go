package provider

import (
	"github.com/go-kit/kit/log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppleProviderGetTitle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><title>Песня «Highway to Hell» (AC/DC) в Apple Music</title></html>`))
	}))
	defer ts.Close()
	testData := []struct {
		in  string
		out string
	}{
		{
			ts.URL + "/ru/album/highway-to-hell/574043989?i=574044008",
			"Highway to Hell - AC/DC",
		},
	}
	p := appleProvider{
		host:   ts.URL,
		logger: log.NewNopLogger(),
	}
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestAppleProviderGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><div id="search"><div class="g"><a href="https://music.apple.com/ru/album/highway-to-hell/574043989?i=574044008"></a></div></div></html>`))
	}))
	defer ts.Close()
	testData := []struct {
		in  string
		out string
	}{
		{
			"Highway to Hell - AC/DC",
			"https://music.apple.com/ru/album/highway-to-hell/574043989?i=574044008",
		},
	}
	p := appleProvider{
		host:   ts.URL,
		logger: log.NewNopLogger(),
	}
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
