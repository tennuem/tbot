package provider

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var videoIDRegexp = regexp.MustCompile(`"videoId":"([^"]+)"`)

func NewYoutubeProvider(ctx context.Context) Provider {
	return &youtubeProvider{
		host: "https://www.youtube.com",
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type youtubeProvider struct {
	host   string //"https://music.youtube.com"
	client *http.Client
}

func (p *youtubeProvider) Name() string {
	return "▶️ youtube"
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
	resp, err := p.client.Do(req)
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
	return title, nil
}

func (p *youtubeProvider) GetURL(title string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/results", p.host))
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("search_query", title)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_14_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/79.0.3945.130 Safari/537.36")
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	matches := videoIDRegexp.FindSubmatch(body)
	if len(matches) < 2 {
		return "", ErrURLNotFound
	}
	videoID := string(matches[1])
	if videoID == "" {
		return "", ErrURLNotFound
	}
	link := fmt.Sprintf("https://music.youtube.com/watch?v=%s", videoID)
	return link, nil
}
