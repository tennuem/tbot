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
			"https://music.yandex.com/album/8508157/track/57016085",
			[]string{
				"https://music.yandex.com/album/8508157/track/57016085",
				"https://music.youtube.com/watch?v=KViOTZ62zBg",
				"https://music.apple.com/us/album/babushka-boi-single/1477644647",
			},
		},
	}
	for _, c := range testData {
		res, err := GetLinks(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}
