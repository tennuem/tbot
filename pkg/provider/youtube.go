package provider

import (
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

func NewYoutubeProvider() Provider {
	return &youtubeProvider{}
}

type youtubeProvider struct{}

func (p *youtubeProvider) GetTitle(rawUrl string) (string, error) {
	req, err := http.NewRequest("GET", rawUrl, nil)
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
	return res[0 : len(res)-1], nil
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

	log.Printf("search link: %v", u.String())

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
