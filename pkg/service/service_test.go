package service

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/provider"
)

func TestFindLinks(t *testing.T) {
	testData := []struct {
		in  *Message
		out *Message
	}{
		{
			&Message{URL: "https://music.yandex.com/album/8508157/track/57016085"},
			&Message{
				Links: []string{
					"https://music.youtube.com/watch?v=KViOTZ62zBg",
					"https://music.apple.com/us/album/babushka-boi-single/1477644647",
				}},
		},
	}
	svc := NewService(
		NewStoreMock(),
		map[string]provider.Provider{
			"music.yandex.com":  provider.NewYandexProvider(log.NewNopLogger()),
			"music.youtube.com": provider.NewYoutubeProvider(log.NewNopLogger()),
			"music.apple.com":   provider.NewAppleProvider(log.NewNopLogger()),
		},
		log.NewNopLogger(),
	)
	for k, c := range testData {
		res, err := svc.FindLinks(context.Background(), c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out.Links, res.Links, fmt.Sprintf("case-%d", k))
	}
}
