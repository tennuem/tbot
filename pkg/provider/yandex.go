package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ndrewnee/go-yamusic/yamusic"
)

type YandexClient interface {
	GetTrack(id int) (string, error)
	Search(title string) (string, error)
}

func NewYandexProvider(ctx context.Context) Provider {
	return &yandexProvider{client: &yandexClient{client: yamusic.NewClient()}}
}

type yandexProvider struct {
	host   string
	client YandexClient
}

func (p *yandexProvider) Name() string {
	return "yandex"
}

func (p *yandexProvider) Host() string {
	return "music.yandex.com"
}

func (p *yandexProvider) GetTitle(url string) (string, error) {
	substr := "track/"
	sid := url[strings.Index(url, substr)+len(substr):]
	id, err := strconv.Atoi(sid)
	if err != nil {
		return "", err
	}
	return p.client.GetTrack(id)
}

func (p *yandexProvider) GetURL(title string) (string, error) {
	return p.client.Search(title)
}

type yandexClient struct {
	client *yamusic.Client
}

func (c *yandexClient) GetTrack(id int) (string, error) {
	tracks, _, err := c.client.Tracks().Get(context.Background(), id)
	if err != nil {
		return "", err
	}
	if len(tracks.Result) == 0 {
		return "", fmt.Errorf("result is empty")
	}
	if len(tracks.Result[0].Artists) == 0 {
		return "", fmt.Errorf("artists is empty")
	}

	track := tracks.Result[0].Title
	author := tracks.Result[0].Artists[0].Name

	return fmt.Sprintf("%s — %s", track, author), nil
}

func (c *yandexClient) Search(title string) (string, error) {
	resp, _, err := c.client.Search().Tracks(context.Background(), title, nil)
	if err != nil {
		return "", err
	}
	if len(resp.Result.Tracks.Results) == 0 {
		return "", fmt.Errorf("result is empty")
	}
	if len(resp.Result.Tracks.Results[0].Albums) == 0 {
		return "", fmt.Errorf("albums is empty")
	}
	trackID := resp.Result.Tracks.Results[0].ID
	albumID := resp.Result.Tracks.Results[0].Albums[0].ID

	return fmt.Sprintf("https://music.yandex.ru/album/%d/track/%d", albumID, trackID), nil
}
