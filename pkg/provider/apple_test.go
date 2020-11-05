package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ‎Песня «DLBM» (Miyagi &amp; Эндшпиль &amp; N.E.R.A.K.) в Apple Music
// Песня «Babushka Boi» (A$AP Rocky) в Apple Music
func TestAppleProviderGetTitle(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"https://music.apple.com/ru/album/dlbm/1267895125?i=1267895588",
			"DLBM - Miyagi & Эндшпиль & NERAK",
		},
		{
			"https://music.apple.com/ru/album/babushka-boi/1477644647?i=1477644655",
			"Babushka Boi - A$AP Rocky",
		},
	}
	p := NewAppleProvider(context.Background())
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestAppleProviderGetURL(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"DLBM - Miyagi & Эндшпиль & N.E.R.A.K.",
			"https://music.apple.com/ca/album/dlbm-single/1267895125",
		},
	}
	p := NewAppleProvider(context.Background())
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
