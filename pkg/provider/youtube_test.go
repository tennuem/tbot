package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestYoutubeProviderGetTitle(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"https://music.youtube.com/watch?v=otl8yjZcg2Y&feature=share",
			"Death by Dishonor — Ghostemane, Pouya, Shakewell, Erick the Architect",
		},
	}
	p := NewYoutubeProvider()
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestYoutubeProviderGetURL(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"Death by Dishonor — Ghostemane, Pouya, Shakewell, Erick the Architect",
			"https://www.youtube.com/watch?list=RDAMVMotl8yjZcg2Y&v=otl8yjZcg2Y",
		},
	}
	p := NewYoutubeProvider()
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
