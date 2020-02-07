package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

func NewYandexProvider(logger log.Logger) Provider {
	return &yandexProvider{logger}
}

type yandexProvider struct {
	logger log.Logger
}

func (p *yandexProvider) GetTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	ss := strings.Split(doc.Find("title").First().Text(), ". ")
	title := ss[0]
	level.Info(p.logger).Log("method", "GetTitle", "msg", title)
	return title, nil
}

func (p *yandexProvider) GetURL(title string) (string, error) {
	purl := "https://music.yandex.com"
	u, err := url.Parse(fmt.Sprintf("%s/search", purl))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("text", title)
	u.RawQuery = q.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	href, ok := doc.Find("a.d-track__title").First().Attr("href")
	if !ok {
		return "", ErrURLNotFound
	}
	link := fmt.Sprintf("%s%s", purl, href)
	level.Info(p.logger).Log("method", "GetURL", "msg", link)
	return link, nil
}
