package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/tennuem/tbot/tools/logging"
)

func NewYoutubeProvider(ctx context.Context) Provider {
	logger := logging.FromContext(ctx)
	logger = log.With(logger, "component", "youtube")
	return &youtubeProvider{"https://www.google.ru", logger}
}

type youtubeProvider struct {
	host   string //"https://music.youtube.com"
	logger log.Logger
}

func (p *youtubeProvider) Name() string {
	return "youtube"
}

func (p *youtubeProvider) Host() string {
	return "music.youtube.com"
}

func (p *youtubeProvider) GetTitle(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Googlebot/2.1 (+http://www.googlebot.com/bot.html)")
	client := new(http.Client)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	//og:video:tag
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	var res string
	doc.Find("meta[property=\"og:video:tag\"]").Each(func(i int, s *goquery.Selection) {
		c, ok := s.Attr("content")
		if ok {
			res += fmt.Sprintf("%s ", c)
		}
	})
	if res == "" {
		return "", ErrTitleNotFound
	}
	title := res[0 : len(res)-1]
	level.Info(p.logger).Log("method", "GetTitle", "msg", title)
	return title, nil
}

func (p *youtubeProvider) GetURL(title string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/search", p.host))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("q", fmt.Sprintf("%s %s", title, "youtube"))
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	href, ok := doc.Find("#search .g a").First().Attr("href")
	if !ok {
		return "", ErrURLNotFound
	}
	link := strings.Replace(href, "www.", "music.", -1)
	level.Info(p.logger).Log("method", "GetURL", "msg", link)
	return link, nil
}
