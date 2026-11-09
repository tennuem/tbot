// Package history handles portable bot history archives and Telegram HTML exports.
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/tbot/pkg/service"
)

const Version = 1

type Archive struct {
	Version   int               `json:"version"`
	CreatedAt time.Time         `json:"created_at"`
	ChatName  string            `json:"chat_name,omitempty"`
	Timezone  string            `json:"timezone,omitempty"`
	Shares    []service.Share   `json:"shares"`
	Cache     []service.Message `json:"cache,omitempty"`
	Aliases   []Alias           `json:"aliases,omitempty"`
}

type Alias struct {
	Provider string `json:"provider"`
	URL      string `json:"url"`
	CacheKey string `json:"cache_key"`
}

func Read(r io.Reader) (*Archive, error) {
	const maxSize = 100 << 20
	data, err := io.ReadAll(io.LimitReader(r, maxSize+1))
	if err != nil {
		return nil, fmt.Errorf("read history archive: %w", err)
	}
	if len(data) > maxSize {
		return nil, errors.New("history archive exceeds 100 MiB")
	}
	var archive Archive
	if err := json.Unmarshal(data, &archive); err != nil {
		return nil, fmt.Errorf("decode history archive: %w", err)
	}
	if err := archive.Validate(false); err != nil {
		return nil, err
	}
	return &archive, nil
}

func (a *Archive) Validate(requireChat bool) error {
	if a.Version != Version {
		return fmt.Errorf("unsupported history version %d", a.Version)
	}
	if len(a.Shares)+len(a.Cache)+len(a.Aliases) > 200_000 {
		return errors.New("too many history records")
	}
	for i := range a.Shares {
		s := &a.Shares[i]
		if s.UserID < 0 || s.UserID == 0 && strings.TrimSpace(s.UserName) == "" || s.SentAt.IsZero() || s.MessageID < 0 {
			return fmt.Errorf("share %d: sender and timestamp are required", i+1)
		}
		if requireChat && s.ChatID == 0 {
			return errors.New("history contains unbound shares: specify --chat-id")
		}
		if err := normalize(&s.URL, &s.Provider); err != nil {
			return fmt.Errorf("share %d: %w", i+1, err)
		}
	}
	keys := make(map[string]bool)
	for i := range a.Cache {
		m := &a.Cache[i]
		if err := normalize(&m.URL, &m.Provider); err != nil {
			return fmt.Errorf("cached track %d: %w", i+1, err)
		}
		if m.CacheKey == "" || keys[m.CacheKey] {
			return fmt.Errorf("cached track %d: missing or duplicate cache key", i+1)
		}
		keys[m.CacheKey] = true
		for j := range m.Links {
			if err := normalize(&m.Links[j].URL, &m.Links[j].Provider); err != nil {
				return fmt.Errorf("cached track %d link %d: %w", i+1, j+1, err)
			}
		}
		m.CacheStatus, m.Refreshing, m.SeenBefore, m.MissingProviders = "", false, false, nil
	}
	for i := range a.Aliases {
		alias := &a.Aliases[i]
		if !keys[alias.CacheKey] {
			return fmt.Errorf("alias %d: unknown cache key", i+1)
		}
		if err := normalize(&alias.URL, &alias.Provider); err != nil {
			return fmt.Errorf("alias %d: %w", i+1, err)
		}
	}
	return nil
}

func normalize(raw, host *string) error {
	u, err := provider.NormalizeURL(*raw)
	if err != nil {
		return err
	}
	if *host != "" && *host != u.Host {
		return errors.New("URL does not belong to its provider")
	}
	*raw, *host = u.String(), u.Host
	return nil
}

// BindChat only binds unassigned records; it never moves another chat's history.
func (a *Archive) BindChat(chatID int64) {
	for i := range a.Shares {
		if a.Shares[i].ChatID == 0 {
			a.Shares[i].ChatID = chatID
		}
	}
}

func ShareKey(s service.Share) string {
	data := fmt.Sprintf("%d|%d|%d|%s|%s|%s", s.ChatID, s.MessageID, s.UserID,
		s.SentAt.UTC().Format(time.RFC3339Nano), s.Provider, s.URL)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func IsMusicTrack(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	switch u.Host {
	case "music.yandex.com":
		return len(parts) == 2 && parts[0] == "track" && numeric(parts[1]) ||
			len(parts) == 4 && parts[0] == "album" && numeric(parts[1]) && parts[2] == "track" && numeric(parts[3])
	case "open.spotify.com":
		return len(parts) == 2 && parts[0] == "track" && parts[1] != ""
	case "music.youtube.com":
		return u.Path == "/watch" && u.Query().Get("v") != ""
	case "music.apple.com":
		return numeric(u.Query().Get("i")) || len(parts) >= 3 && parts[1] == "song" && numeric(parts[len(parts)-1])
	default:
		return false
	}
}

func numeric(text string) bool {
	n, err := strconv.ParseUint(text, 10, 64)
	return err == nil && n > 0
}

type Summary struct {
	Shares      int            `json:"shares"`
	Cache       int            `json:"cache_records"`
	Aliases     int            `json:"aliases"`
	UniqueLinks int            `json:"unique_links"`
	MusicTracks int            `json:"music_track_links"`
	Unbound     int            `json:"unbound_shares"`
	Authors     map[string]int `json:"authors"`
	Providers   map[string]int `json:"providers"`
}

func Summarize(a *Archive) Summary {
	s := Summary{Shares: len(a.Shares), Cache: len(a.Cache), Aliases: len(a.Aliases), Authors: make(map[string]int), Providers: make(map[string]int)}
	seen := make(map[string]bool)
	for _, share := range a.Shares {
		s.Authors[fmt.Sprintf("%s (%d)", share.UserName, share.UserID)]++
		s.Providers[share.Provider]++
		if share.ChatID == 0 {
			s.Unbound++
		}
		if !seen[share.URL] {
			seen[share.URL] = true
			s.UniqueLinks++
			if IsMusicTrack(share.URL) {
				s.MusicTracks++
			}
		}
	}
	return s
}
