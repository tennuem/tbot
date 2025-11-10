package provider

import (
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
		client: ts.Client(),
	}
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestAppleProviderGetURL(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[{"trackViewUrl":"https://music.apple.com/ru/album/highway-to-hell/574043989?i=574044008"}]}`))
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
		client: ts.Client(),
	}
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
