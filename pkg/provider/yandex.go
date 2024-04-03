package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/ndrewnee/go-yamusic/yamusic"
)

func NewYandexProvider(ctx context.Context) Provider {
	return &yandexProvider{client: yamusic.NewClient()}
}

type yandexProvider struct {
	host   string
	client *yamusic.Client
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
	tracks, _, err := p.client.Tracks().Get(context.Background(), id)
	if err != nil {
		return "", err
	}
	if len(tracks.Result) == 0 {
		return "", ErrTitleNotFound
	}
	if len(tracks.Result[0].Artists) == 0 {
		return "", ErrTitleNotFound
	}

	track := tracks.Result[0].Title
	author := tracks.Result[0].Artists[0].Name

	return fmt.Sprintf("%s — %s", track, author), nil
}

func (p *yandexProvider) GetURL(title string) (string, error) {
	resp, _, err := p.client.Search().Tracks(context.Background(), title, nil)
	if err != nil {
		return "", err
	}
	if len(resp.Result.Tracks.Results) == 0 {
		return "", ErrURLNotFound
	}
	if len(resp.Result.Tracks.Results[0].Albums) == 0 {
		return "", ErrURLNotFound
	}

	trackID := resp.Result.Tracks.Results[0].ID
	albumID := resp.Result.Tracks.Results[0].Albums[0].ID

	return fmt.Sprintf("https://music.yandex.ru/album/%d/track/%d", albumID, trackID), nil
}
