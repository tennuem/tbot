package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYandexProviderGetTitle(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><div class="sidebar__title"><a class="d-link">Highway to Hell </a></div><div class="sidebar__info"><a class="d-link">AC/DC </a></div></html>`))
	}))
	defer ts.Close()
	testData := []struct {
		in  string
		out string
	}{
		{
			ts.URL + "/album/2832579/track/694683",
			"Highway to Hell — AC/DC",
		},
	}
	p := yandexProvider{
		host:   ts.URL,
		logger: log.NewNopLogger(),
	}
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestYandexProviderGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><a href="/album/2832579/track/694683" class="d-track__title"> BFG Division </a></html>`))
	}))
	defer ts.Close()
	testData := []struct {
		in  string
		out string
	}{
		{
			"Highway to Hell — AC/DC",
			ts.URL + "/album/2832579/track/694683",
		},
	}
	p := yandexProvider{
		host:   ts.URL,
		logger: log.NewNopLogger(),
	}
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
