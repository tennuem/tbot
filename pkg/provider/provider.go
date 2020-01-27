package provider

import (
	"log"
	"net/url"

	"github.com/pkg/errors"
)

var (
	ErrEmptyURL         = errors.New("URL is empty")
	ErrProviderNotFound = errors.New("provider not found")
	ErrTitleNotFound    = errors.New("title not found")
	ErrURLNotFound      = errors.New("URL not found")
)

type Provider interface {
	GetTitle(url string) (string, error)
	GetURL(title string) (string, error)
}

// getLinks возвращает список ссылок по которым удалось найти трек.
func getLinks(rawURL string) ([]string, error) {
	if rawURL == "" {
		return nil, ErrEmptyURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}

	providers := map[string]Provider{
		"music.yandex.com":  NewYandexProvider(),
		"music.youtube.com": NewYoutubeProvider(),
		"music.apple.com":   NewAppleProvider(),
	}

	mainProvider, ok := providers[u.Hostname()]
	if !ok {
		return nil, ErrProviderNotFound
	}
	title, err := mainProvider.GetTitle(rawURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get title")
	}

	var res []string
	res = append(res, rawURL)
	for k, p := range providers {
		if k == u.Hostname() {
			continue
		}
		u, err := p.GetURL(title)
		if err != nil {
			log.Printf("failed to get url by title %v", err)
			continue
		}
		res = append(res, u)
	}
	return res, nil
}
