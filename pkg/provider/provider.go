package provider

import (
	"log"
	"net/url"
	"strings"

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
func GetLinks(rawURL string) ([]string, error) {
	if rawURL == "" {
		return nil, ErrEmptyURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse url")
	}
	if strings.HasSuffix(u.Host, ".ru") {
		u.Host = strings.Replace(u.Host, ".ru", ".com", -1)
	}

	providers := map[string]Provider{
		"music.yandex.com":  NewYandexProvider(),
		"music.youtube.com": NewYoutubeProvider(),
		"music.apple.com":   NewAppleProvider(),
	}

	mainProvider, ok := providers[u.Host]
	if !ok {
		return nil, ErrProviderNotFound
	}
	title, err := mainProvider.GetTitle(rawURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get title")
	}

	var res []string
	for k, p := range providers {
		if k == u.Host {
			continue
		}
		u, err := p.GetURL(title)
		if err != nil {
			log.Printf("failed to get url from: %v, by title %v", k, err.Error())
			continue
		}
		if u != "" {
			res = append(res, u)
		}
	}
	return res, nil
}
