package service

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"github.com/tennuem/telegram"
)

func TestListPaginationKeepsTracksAndLinksWithinMessageLimit(t *testing.T) {
	var text strings.Builder
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&text, "Трек %d %s\nhttps://music.yandex.com/album/1/track/%d\n", i, strings.Repeat("🎵", 30), i+1)
	}
	var combined strings.Builder
	for page := 1; ; page++ {
		result := formatListPage(text.String(), page)
		if strings.HasPrefix(result, "Такой страницы нет") {
			break
		}
		require.LessOrEqual(t, textUnits(result), 4096)
		require.True(t, utf8.ValidString(result))
		require.Contains(t, result, fmt.Sprintf("Страница %d", page))
		combined.WriteString(result)
		require.Less(t, page, 100)
	}
	for i := 0; i < 100; i++ {
		require.Equal(t, 1, strings.Count(combined.String(), fmt.Sprintf("/track/%d\n", i+1)))
	}
}

func TestListHandlesLongTitlesAndInvalidPageArguments(t *testing.T) {
	text := formatListPage(strings.Repeat("🎵", 5000)+"\nhttps://music.yandex.com/album/1/track/2\n", 1)
	require.LessOrEqual(t, textUnits(text), 4096)
	require.True(t, utf8.ValidString(text))
	require.Contains(t, text, "https://music.yandex.com/album/1/track/2")
	for _, arg := range []string{"invalid", "0", "-1", "1 2", "99999999999999999999999"} {
		svc := &transportService{}
		w := &telegram.ResponseWriter{}
		NewTelegramHandler(svc, nil, "tbot").Serve(w, telegramCommandRequest("/list@tbot "+arg))
		require.Contains(t, w.Text, "Используйте /list")
		require.Zero(t, svc.listCalls)
	}
}
