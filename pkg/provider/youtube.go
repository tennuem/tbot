package provider

import (
	"fmt"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
)

func NewYoutubeProvider() Provider {
	return &youtubeProvider{}
}

type youtubeProvider struct{}

func (p *youtubeProvider) GetTitle(url string) (string, error) {
	return "", ErrTitleNotFound
}

func (p *youtubeProvider) GetURL(title string) (string, error) {
	purl := "https://music.youtube.com"
	u, err := url.Parse(fmt.Sprintf("%s/search", purl))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("q", title)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	n, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	link := findLinkByClass(n, "ytp-title-link")

	return link, nil
}
