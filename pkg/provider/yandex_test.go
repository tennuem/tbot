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
			"https://music.yandex.ru/album/8834580/track/58314507",
			"The Hard Interchange — Champs",
		},
	}
	p := yandexProvider{
		client: &yandexClientMock{},
	}
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
			"The Hard Interchange — Champs",
			"https://music.yandex.ru/album/8834580/track/58314507",
		},
	}
	p := yandexProvider{
		client: &yandexClientMock{},
	}
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

type yandexClientMock struct {
}

func (c *yandexClientMock) GetTrack(id int) (string, error) {
	return "The Hard Interchange — Champs", nil
}

func (c *yandexClientMock) Search(title string) (string, error) {
	return "https://music.yandex.ru/album/8834580/track/58314507", nil
}
