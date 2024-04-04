package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/PuerkitoBio/goquery"
)

func NewAppleProvider(ctx context.Context) Provider {
	return &appleProvider{host: "https://google.ru"}
}

type appleProvider struct {
	host string
}

func (p *appleProvider) Name() string {
	return "🍏 apple"
}

func (p *appleProvider) Host() string {
	return "music.apple.com"
}

func (p *appleProvider) GetTitle(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
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
	reg, err := regexp.Compile(`^.+«(.+)» \((.+)\)`)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	sel := doc.Find("title").Text()
	ss := reg.FindStringSubmatch(sel)
	if ss == nil {
		return "", ErrTitleNotFound
	}
	title := fmt.Sprintf("%s - %s", ss[1], ss[2])
	return title, nil
}

func (p *appleProvider) GetURL(title string) (string, error) {
	u, err := url.Parse(fmt.Sprintf("%s/search", p.host))
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
	return link, nil
}
