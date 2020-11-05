package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/tennuem/tbot/tools/logging"
)

func NewAppleProvider(ctx context.Context) Provider {
	logger := logging.FromContext(ctx)
	logger = log.With(logger, "component", "apple")
	return &appleProvider{"music.yandex.com", logger}
}

type appleProvider struct {
	host   string
	logger log.Logger
}

func (p *appleProvider) Host() string {
	return p.host
}

func (p *appleProvider) GetTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	reg, err := regexp.Compile(`^.+«(.+)» \((.+)\)`)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	ss := reg.FindStringSubmatch(doc.Find("title").Text())
	if ss == nil {
		return "", ErrTitleNotFound
	}
	title := fmt.Sprintf("%s - %s", ss[1], ss[2])
	level.Info(p.logger).Log("method", "GetTitle", "msg", title)
	return title, nil
}

func (p *appleProvider) GetURL(title string) (string, error) {
	purl := "https://google.ru"
	u, err := url.Parse(fmt.Sprintf("%s/search", purl))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("q", fmt.Sprintf("%s apple music", title))
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36")
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	link, ok := doc.Find("#search .g a").First().Attr("href")
	if !ok {
		return "", ErrURLNotFound
	}
	level.Info(p.logger).Log("method", "GetURL", "msg", link)
	return link, nil
}
