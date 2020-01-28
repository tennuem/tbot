package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func NewYoutubeProvider() Provider {
	return &youtubeProvider{}
}

type youtubeProvider struct{}

func (p *youtubeProvider) GetTitle(url string) (string, error) {
	return "", ErrTitleNotFound
}

func (p *youtubeProvider) GetURL(title string) (string, error) {
	purl := "https://www.google.ru"
	u, err := url.Parse(fmt.Sprintf("%s/search", purl))
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

	link, ok := doc.Find(".srg .g a").First().Attr("href")
	if !ok {
		return "", ErrURLNotFound
	}

	return strings.Replace(link, "www.", "music.", -1), nil
}
