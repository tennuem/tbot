package provider

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLinks(t *testing.T) {
	testData := []struct {
		in  string
		out []string
	}{
		{
			"https://music.yandex.com/album/4141435/track/33829704",
			[]string{
				"https://music.yandex.com/album/4141435/track/33829704",
			},
		},
		{
			"https://music.youtube.com/watch?v=otl8yjZcg2Y&feature=share",
			[]string{
				"https://music.youtube.com/watch?v=otl8yjZcg2Y&feature=share",
			},
		},
	}
	for _, c := range testData {
		res, err := getLinks(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
