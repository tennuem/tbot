package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYandexProviderGetTitle(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"https://music.yandex.com/album/4141435/track/33829704",
			"Death by Dishonor — Ghostemane, Pouya, Shakewell, Erick the Architect",
		},
	}
	p := NewYandexProvider()
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestYandexProviderGetURL(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"Death by Dishonor — Ghostemane, Pouya, Shakewell, Erick the Architect",
			"https://music.yandex.com/album/4141435/track/33829704",
		},
	}
	p := NewYandexProvider()
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
