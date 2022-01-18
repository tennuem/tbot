package provider

import (
	"context"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/tennuem/tbot/tools/logging"
	"net/http"
	"net/url"
	"strings"
)

func NewYandexProvider(ctx context.Context) Provider {
	logger := logging.FromContext(ctx)
	logger = log.With(logger, "component", "yandex")
	return &yandexProvider{"https://music.yandex.com", logger}
}

type yandexProvider struct {
	host   string
	logger log.Logger
}

func (p *yandexProvider) Host() string {
	return "music.yandex.com"
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
	track := doc.Find(".sidebar__title a.d-link").Text()
	track = strings.TrimSuffix(track, " ")
	author := doc.Find(".sidebar__info a.d-link").Text()
	author = strings.TrimSuffix(author, " ")
	title := fmt.Sprintf("%s — %s", track, author)
	level.Info(p.logger).Log("method", "GetTitle", "msg", title)
	return title, nil
}

func (p *yandexProvider) GetURL(title string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/search", p.host))
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
	link := fmt.Sprintf("%s%s", p.host, href)
	level.Info(p.logger).Log("method", "GetURL", "msg", link)
	return link, nil
}
