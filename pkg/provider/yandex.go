package provider

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

func NewYandexProvider() Provider {
	return &yandexProvider{}
}

type yandexProvider struct{}

func (p *yandexProvider) GetTitle(url string) (string, error) {
	// TODO: запрос для получения html можно вынести в отдельный метож
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	n, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	ss := strings.Split(pageTitle(n), ". ")
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

	resp, err := http.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	n, err := html.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	link := findLinkByClass(n, "d-track__title deco-link deco-link_stronger")

	return fmt.Sprintf("%s%s", purl, link), nil
}

func pageTitle(n *html.Node) string {
	var title string
	if n.Type == html.ElementNode && n.Data == "title" {
		return n.FirstChild.Data
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		title = pageTitle(c)
		if title != "" {
			break
		}
	}
	return title
}

// d-track__title deco-link deco-link_stronger
func findLinkByClass(n *html.Node, class string) string {
	var link string
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, v := range n.Attr {
			if v.Key == "class" && v.Val == class {
				fmt.Println(n)
				return getHref(n)
			}
		}

	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		link = findLinkByClass(c, class)
		if link != "" {
			break
		}
	}
	return link
}

func getHref(n *html.Node) string {
	for _, v := range n.Attr {
		if v.Key == "href" {
			return v.Val
		}
	}
	return ""
}
