package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/ndrewnee/go-yamusic/yamusic"
)

type YandexClient interface {
	GetTrack(id int) (string, error)
	Search(title string) (string, error)
}

func NewYandexProvider(ctx context.Context, proxyURL string) (Provider, error) {
	httpClient, err := newYandexHTTPClient(proxyURL)
	if err != nil {
		return nil, err
	}
	return &yandexProvider{client: &yandexClient{
		ctx:    ctx,
		client: yamusic.NewClient(yamusic.HTTPClient(httpClient)),
	}}, nil
}

func newYandexHTTPClient(rawProxyURL string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if rawProxyURL != "" {
		proxyURL, err := url.Parse(rawProxyURL)
		if err != nil {
			// Parse errors may contain credentials from the proxy URL.
			return nil, fmt.Errorf("invalid yandex.proxy_url: cannot parse URL")
		}
		if proxyURL.Scheme != "http" && proxyURL.Scheme != "https" {
			return nil, fmt.Errorf("invalid yandex.proxy_url: scheme must be http or https")
		}
		if proxyURL.Hostname() == "" || proxyURL.Opaque != "" {
			return nil, fmt.Errorf("invalid yandex.proxy_url: host is required")
		}
		if proxyURL.Path != "" && proxyURL.Path != "/" || proxyURL.RawQuery != "" || proxyURL.ForceQuery || proxyURL.Fragment != "" {
			return nil, fmt.Errorf("invalid yandex.proxy_url: path, query and fragment are not supported")
		}
		if port := proxyURL.Port(); port != "" {
			n, err := strconv.Atoi(port)
			if err != nil || n < 1 || n > 65535 {
				return nil, fmt.Errorf("invalid yandex.proxy_url: port must be between 1 and 65535")
			}
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{Transport: transport, Timeout: 10 * time.Second}, nil
}

type yandexProvider struct {
	host   string
	client YandexClient
}

func (p *yandexProvider) Name() string {
	return "🎧 yandex"
}

func (p *yandexProvider) Host() string {
	return "music.yandex.com"
}

func (p *yandexProvider) GetTitle(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	trackIDRegex := regexp.MustCompile(`album/\d+/track/(\d+)`)
	matches := trackIDRegex.FindStringSubmatch(parsedURL.Path)
	if len(matches) < 2 {
		return "", fmt.Errorf("не удалось найти идентификатор трека в URL")
	}

	id, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("не удалось преобразовать ид трека '%s' в число: %w", matches[1], err)
	}

	return p.client.GetTrack(id)
}

func (p *yandexProvider) GetURL(title string) (string, error) {
	return p.client.Search(title)
}

type yandexClient struct {
	ctx    context.Context
	client *yamusic.Client
}

func (c *yandexClient) GetTrack(id int) (string, error) {
	tracks, _, err := c.client.Tracks().Get(c.ctx, id)
	if err != nil {
		return "", err
	}
	if len(tracks.Result) == 0 {
		return "", fmt.Errorf("result is empty")
	}
	if len(tracks.Result[0].Artists) == 0 {
		return "", fmt.Errorf("artists is empty")
	}

	track := tracks.Result[0].Title
	author := tracks.Result[0].Artists[0].Name

	return fmt.Sprintf("%s — %s", track, author), nil
}

func (c *yandexClient) Search(title string) (string, error) {
	resp, _, err := c.client.Search().Tracks(c.ctx, title, nil)
	if err != nil {
		return "", err
	}
	if len(resp.Result.Tracks.Results) == 0 {
		return "", fmt.Errorf("result is empty")
	}
	if len(resp.Result.Tracks.Results[0].Albums) == 0 {
		return "", fmt.Errorf("albums is empty")
	}
	trackID := resp.Result.Tracks.Results[0].ID
	albumID := resp.Result.Tracks.Results[0].Albums[0].ID

	return fmt.Sprintf("https://music.yandex.ru/album/%d/track/%d", albumID, trackID), nil
}
