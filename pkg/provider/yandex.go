package provider

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func NewYandexProvider() Provider {
	return &yandexProvider{}
}

type yandexProvider struct{}

func (p *yandexProvider) GetTitle(url string) (string, error) {
	// TODO: запрос для получения html можно вынести в отдельный метод
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	title := doc.Find("title").First().Text()
	ss := strings.Split(title, ". ")
	return ss[0], nil
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

	log.Printf("search link: %v", u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	link, ok := doc.Find("a.d-track__title").First().Attr("href")
	if !ok {
		return "", ErrURLNotFound
	}

	return fmt.Sprintf("%s%s", purl, link), nil
}
