package store

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/service"
)

func TestStore(t *testing.T) {
	addr := "mongodb://root:root@localhost:27017/?ssl=false"
	testData := []struct {
		in  *service.Message
		out *service.Message
	}{
		{
			in:  &service.Message{"link1", "title1", []string{"a", "b", "c"}, ""},
			out: &service.Message{"link1", "title1", []string{"a", "b", "c"}, ""},
		},
		{
			in:  &service.Message{"link2", "title2", []string{"a", "b"}, ""},
			out: &service.Message{"link2", "title2", []string{"a", "b"}, ""},
		},
		{
			in: &service.Message{
				"https://music.yandex.com/album/8508157/track/57016085",
				"title3",
				[]string{
					"https://music.youtube.com/watch?v=KViOTZ62zBg",
					"https://music.apple.com/us/album/babushka-boi-single/1477644647",
				},
				"",
			},
			out: &service.Message{
				"https://music.yandex.com/album/8508157/track/57016085",
				"title3",
				[]string{
					"https://music.youtube.com/watch?v=KViOTZ62zBg",
					"https://music.apple.com/us/album/babushka-boi-single/1477644647",
				},
				"",
			},
		},
	}
	s, err := NewMongoStore(addr)
	require.NoError(t, err)
	for i, c := range testData {
		require.NoError(t, s.Save(context.Background(), c.in), fmt.Sprintf("case-%d", i))
		res, err := s.FindByURL(context.Background(), c.in.URL)
		require.NoError(t, err, fmt.Sprintf("case-%d", i))
		assert.Equal(t, c.out, res, fmt.Sprintf("case-%d", i))
	}
}
