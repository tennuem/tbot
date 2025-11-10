package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func NewAppleProvider(ctx context.Context) Provider {
	return &appleProvider{
		host: "https://itunes.apple.com/search",
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

type appleProvider struct {
	host   string
	client *http.Client
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
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	reg, err := regexp.Compile(`^.+«(.+)» \((.+)\)`)
	if err != nil {
		return "", err
	}
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}
	if meta, ok := doc.Find("meta[property='og:title']").Attr("content"); ok {
		meta = strings.TrimSpace(meta)
		if meta != "" {
			return meta, nil
		}
	}
	sel := doc.Find("title").Text()
	ss := reg.FindStringSubmatch(sel)
	if len(ss) == 0 {
		return "", ErrTitleNotFound
	}
	title := fmt.Sprintf("%s - %s", ss[1], ss[2])
	return title, nil
}

func (p *appleProvider) GetURL(title string) (string, error) {
	u, err := url.Parse(p.host)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("term", title)
	q.Set("media", "music")
	q.Set("entity", "song")
	q.Set("limit", "1")
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
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	searchResp := struct {
		Results []struct {
			TrackViewURL string `json:"trackViewUrl"`
		} `json:"results"`
	}{}
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return "", err
	}
	if len(searchResp.Results) == 0 {
		return "", ErrURLNotFound
	}
	link := strings.TrimSpace(searchResp.Results[0].TrackViewURL)
	if link == "" {
		return "", ErrURLNotFound
	}
	link = strings.TrimRight(link, "&")

	return link, nil
}
