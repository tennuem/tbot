package provider

import (
	"context"
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
			"https://music.youtube.com/watch?v=IA1H1--5GFM&feature=share",
			"A$AP Rocky Babushka Boi",
		},
		{
			"https://music.youtube.com/watch?v=otl8yjZcg2Y&feature=share",
			"Ghostemane Shakewell Pouya & Erick the Architect Death by Dishonor",
		},
	}
	p := NewYoutubeProvider(context.Background())
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
			"Babushka Boi - A$AP Rocky",
			"https://music.youtube.com/watch?v=KViOTZ62zBg",
		},
	}
	p := NewYoutubeProvider(context.Background())
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
