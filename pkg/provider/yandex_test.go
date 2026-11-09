package provider

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ndrewnee/go-yamusic/yamusic"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewYandexProviderInvalidProxy(t *testing.T) {
	for _, rawURL := range []string{
		"ru-proxy:3128", "socks5://ru-proxy:1080", "http://", "http://:3128",
		"http://ru-proxy:abc", "http://ru-proxy:0", "http://ru-proxy:65536",
		"http://ru-proxy/path", "http://ru-proxy?query=1", "http://ru-proxy?", "http://ru-proxy#fragment",
		"http://proxy-user:proxy-secret@ru-proxy:invalid",
	} {
		t.Run(rawURL, func(t *testing.T) {
			p, err := NewYandexProvider(context.Background(), rawURL)
			require.Error(t, err)
			require.Nil(t, p)
			require.Contains(t, err.Error(), "yandex.proxy_url")
			require.NotContains(t, err.Error(), "proxy-secret")
		})
	}
}

func TestYandexProxySelection(t *testing.T) {
	for _, rawURL := range []string{"http://ru-proxy:3128", "https://user:secret@ru-proxy:3129", "http://[::1]:3128/"} {
		t.Run(rawURL, func(t *testing.T) {
			client, err := newYandexHTTPClient(rawURL)
			require.NoError(t, err)
			defer client.CloseIdleConnections()
			require.Equal(t, 10*time.Second, client.Timeout)
			transport := client.Transport.(*http.Transport)
			require.NotSame(t, http.DefaultTransport, transport)
			// An explicit proxy must also be used for hosts excluded by NO_PROXY.
			t.Setenv("NO_PROXY", "*")
			req, err := http.NewRequest(http.MethodGet, "https://api.music.yandex.net/tracks/58314507", nil)
			require.NoError(t, err)
			proxyURL, err := transport.Proxy(req)
			require.NoError(t, err)
			require.Equal(t, rawURL, proxyURL.String())
		})
	}
}

func TestYandexWithoutDedicatedProxy(t *testing.T) {
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "direct response")
	}))
	defer origin.Close()
	client, err := newYandexHTTPClient("")
	require.NoError(t, err)
	defer client.CloseIdleConnections()
	resp, err := client.Get(origin.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "direct response", string(body))
	// Empty configuration preserves the default environment proxy selection.
	req, err := http.NewRequest(http.MethodGet, "https://api.music.yandex.net", nil)
	require.NoError(t, err)
	want, err := http.ProxyFromEnvironment(req)
	require.NoError(t, err)
	got, err := client.Transport.(*http.Transport).Proxy(req)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestYandexHTTPSProxy(t *testing.T) {
	origin := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/tracks/58314507":
			_, _ = io.WriteString(w, `{"result":[{"id":58314507,"title":"The Hard Interchange","artists":[{"name":"Champs"}]}]}`)
		case "/search":
			assert.Equal(t, "The Hard Interchange — Champs", r.URL.Query().Get("text"))
			_, _ = io.WriteString(w, `{"result":{"tracks":{"results":[{"id":58314507,"albums":[{"id":8834580}]}]}}}`)
		default:
			t.Errorf("unexpected API path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer origin.Close()
	connects := make(chan string, 2)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodConnect, r.Method)
		assert.Equal(t, "Basic dXNlcjpzZWNyZXQ=", r.Header.Get("Proxy-Authorization"))
		connects <- r.Host
		upstream, err := net.DialTimeout("tcp", origin.Listener.Addr().String(), time.Second)
		if err != nil {
			t.Errorf("connect to test origin: %v", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer upstream.Close()
		conn, rw, err := w.(http.Hijacker).Hijack()
		if err != nil {
			t.Errorf("hijack proxy connection: %v", err)
			return
		}
		defer conn.Close()
		_, _ = rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
		if err := rw.Flush(); err != nil {
			t.Errorf("send CONNECT response: %v", err)
			return
		}
		go func() {
			_, _ = io.Copy(upstream, rw)
			_ = upstream.Close()
		}()
		_, _ = io.Copy(conn, upstream)
	}))
	defer proxy.Close()
	proxyURL, err := url.Parse(proxy.URL)
	require.NoError(t, err)
	proxyURL.User = url.UserPassword("user", "secret")
	httpClient, err := newYandexHTTPClient(proxyURL.String())
	require.NoError(t, err)
	defer httpClient.CloseIdleConnections()
	// Trust only the local test server's certificate; production keeps normal TLS verification.
	httpClient.Transport.(*http.Transport).TLSClientConfig = origin.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
	baseURL, err := url.Parse(origin.URL)
	require.NoError(t, err)
	p := &yandexProvider{client: &yandexClient{
		ctx:    context.Background(),
		client: yamusic.NewClient(yamusic.HTTPClient(httpClient), yamusic.BaseURL(baseURL)),
	}}
	title, err := p.GetTitle("https://music.yandex.ru/album/8834580/track/58314507")
	require.NoError(t, err)
	require.Equal(t, "The Hard Interchange — Champs", title)
	link, err := p.GetURL(title)
	require.NoError(t, err)
	require.Equal(t, "https://music.yandex.ru/album/8834580/track/58314507", link)
	select {
	case host := <-connects:
		require.Equal(t, baseURL.Host, host)
	default:
		t.Fatal("Yandex requests did not use the CONNECT proxy")
	}
	// Other HTTP clients still connect directly.
	resp, err := origin.Client().Get(origin.URL + "/tracks/58314507")
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	select {
	case host := <-connects:
		t.Fatalf("unexpected additional proxy connection: %s", host)
	default:
	}
}

func TestYandexProxyFailure(t *testing.T) {
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodConnect, r.Method)
		assert.Equal(t, "api.music.yandex.net:443", r.Host)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer proxy.Close()
	p, err := NewYandexProvider(context.Background(), proxy.URL)
	require.NoError(t, err)
	_, err = p.GetTitle("https://music.yandex.ru/album/8834580/track/58314507")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Bad Gateway")
	_, err = p.GetURL("The Hard Interchange — Champs")
	require.Error(t, err)
	require.Contains(t, err.Error(), "Bad Gateway")
}

func TestYandexCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	p, err := NewYandexProvider(ctx, "http://127.0.0.1:1")
	require.NoError(t, err)
	_, err = p.GetTitle("https://music.yandex.ru/album/8834580/track/58314507")
	require.ErrorIs(t, err, context.Canceled)
	_, err = p.GetURL("Champs")
	require.ErrorIs(t, err, context.Canceled)
}

func TestYandexProviderGetTitle(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"https://music.yandex.ru/album/8834580/track/58314507?utm_medium=copy_link",
			"The Hard Interchange — Champs",
		},
	}
	p := yandexProvider{
		client: &yandexClientMock{},
	}
	for _, c := range testData {
		res, err := p.GetTitle(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

func TestYandexProviderGetTitleWithoutTrackID(t *testing.T) {
	p := yandexProvider{client: &yandexClientMock{}}
	for _, rawURL := range []string{"https://music.yandex.ru/album/8834580", "https://music.yandex.ru/", ""} {
		t.Run(rawURL, func(t *testing.T) {
			title, err := p.GetTitle(rawURL)
			require.Error(t, err)
			require.Empty(t, title)
		})
	}
}

func TestYandexProviderGetURL(t *testing.T) {
	testData := []struct {
		in  string
		out string
	}{
		{
			"The Hard Interchange — Champs",
			"https://music.yandex.ru/album/8834580/track/58314507",
		},
	}
	p := yandexProvider{
		client: &yandexClientMock{},
	}
	for _, c := range testData {
		res, err := p.GetURL(c.in)
		require.NoError(t, err)
		assert.Equal(t, c.out, res)
	}
}

type yandexClientMock struct{}

func (c *yandexClientMock) GetTrack(id int) (string, error) {
	return "The Hard Interchange — Champs", nil
}

func (c *yandexClientMock) Search(title string) (string, error) {
	return "https://music.yandex.ru/album/8834580/track/58314507", nil
}
