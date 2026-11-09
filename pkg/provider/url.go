package provider

import (
	"fmt"
	"net/url"
	"strings"
)

// NormalizeURL removes sharing metadata while retaining track identifiers.
func NormalizeURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" || u.User != nil {
		return nil, fmt.Errorf("invalid music URL")
	}
	host := strings.ToLower(u.Hostname())
	defaultPort := u.Scheme == "https" && u.Port() == "443" || u.Scheme == "http" && u.Port() == "80"
	if u.Port() != "" && !defaultPort {
		return nil, fmt.Errorf("non-standard music URL port")
	}
	if u.Port() == "" || defaultPort {
		u.Host = host
		if strings.Contains(host, ":") {
			u.Host = "[" + host + "]"
		}
	}
	switch host {
	case "music.yandex.ru", "music.yandex.com", "music.yandex.by", "music.yandex.kz", "music.yandex.uz":
		u.Host = "music.yandex.com"
	case "open.spotify.com", "link.spotify.com":
		u.Host = "open.spotify.com"
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) == 3 && strings.HasPrefix(parts[0], "intl-") {
			u.Path, u.RawPath = "/"+strings.Join(parts[1:], "/"), ""
		}
	case "youtube.com", "www.youtube.com", "m.youtube.com", "music.youtube.com", "youtu.be", "www.youtu.be":
		u.Host = "music.youtube.com"
		if host == "youtu.be" || host == "www.youtu.be" {
			q := u.Query()
			q.Set("v", strings.Trim(u.Path, "/"))
			u.Path, u.RawPath, u.RawQuery = "/watch", "", q.Encode()
		}
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return nil, fmt.Errorf("invalid music URL query")
	}
	for key := range q {
		if strings.HasPrefix(strings.ToLower(key), "utm_") {
			q.Del(key)
			continue
		}
		switch strings.ToLower(key) {
		case "ref", "ref_id", "gclid", "fbclid", "yclid", "ysclid", "si":
			q.Del(key)
		}
	}
	if u.Host == "music.youtube.com" && u.Path == "/watch" && q.Get("v") != "" {
		q = url.Values{"v": {q.Get("v")}}
	}
	if u.Host == "music.apple.com" {
		for _, key := range []string{"at", "ct", "itsct", "itscg", "ls", "app", "uo", "mt"} {
			q.Del(key)
		}
	}
	u.RawQuery, u.ForceQuery = q.Encode(), false
	u.Fragment, u.RawFragment = "", ""
	if u.Host == "music.yandex.com" || u.Host == "open.spotify.com" || u.Host == "music.youtube.com" || u.Host == "music.apple.com" {
		u.Scheme = "https"
	}
	return u, nil
}
