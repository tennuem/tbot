package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/tennuem/tbot/pkg/provider"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrLinkNotFound     = errors.New("link not found in message")
	ErrLinksNotFound    = errors.New("links not found")
	ErrCacheMiss        = errors.New("link not found in cache")
)

type Service interface {
	FindLinks(ctx context.Context, m *Message) (*Message, error)
	GetList(ctx context.Context, userID int) (string, error)
	Close() error
}

type Link struct {
	URL      string `bson:"url"`
	Name     string `bson:"name"`
	Provider string `bson:"provider"`
}

type Message struct {
	CacheKey         string    `bson:"cache_key"`
	URL              string    `bson:"url"`
	Provider         string    `bson:"provider"`
	Title            string    `bson:"title"`
	Links            []Link    `bson:"links,omitempty"`
	UserID           int       `bson:"user_id"`
	UserName         string    `bson:"user_name"`
	ChatID           int64     `bson:"chat_id"`
	MessageID        int       `bson:"message_id"`
	FirstSeenAt      time.Time `bson:"first_seen_at"`
	CheckedAt        time.Time `bson:"checked_at"`
	CacheStatus      string    `bson:"-" json:"-"`
	Refreshing       bool      `bson:"-" json:"-"`
	SeenBefore       bool      `bson:"-" json:"-"`
	MissingProviders []string  `bson:"-" json:"-"`
}

// Share records a link's sender independently of the shared track cache.
type Share struct {
	MessageID int       `json:"message_id"`
	ChatID    int64     `json:"chat_id"`
	UserID    int       `json:"user_id"`
	UserName  string    `json:"user_name"`
	SentAt    time.Time `json:"sent_at"`
	URL       string    `json:"url"`
	Provider  string    `json:"provider"`
	Title     string    `json:"title,omitempty"`
}

type shareRecorder interface {
	RecordShare(context.Context, Share) error
}

type Store interface {
	Save(ctx context.Context, m *Message) error
	FindByURL(ctx context.Context, provider, url string) (*Message, error)
	FindByUser(ctx context.Context, userID int) ([]Message, error)
}

func NewService(ctx context.Context, s Store, providers ...provider.Provider) Service {
	ctx, cancel := context.WithCancel(ctx)
	return &service{ctx: ctx, cancel: cancel, store: s, providers: providers, refreshing: make(map[string]bool), slots: make(chan struct{}, 4)}
}

type service struct {
	ctx        context.Context
	cancel     context.CancelFunc
	closed     bool
	store      Store
	providers  []provider.Provider
	mu         sync.Mutex
	refreshing map[string]bool
	slots      chan struct{}
	workers    sync.WaitGroup
}

func (s *service) FindLinks(ctx context.Context, input *Message) (*Message, error) {
	raw, _ := extractLink(input.URL)
	if raw == "" {
		return nil, ErrLinkNotFound
	}
	u, err := parseURL(raw)
	if err != nil {
		return nil, err
	}
	p, err := s.findProvider(u.Host)
	if err != nil {
		return nil, err
	}
	link := u.String()
	if recorder, ok := s.store.(shareRecorder); ok {
		sentAt := input.FirstSeenAt
		if sentAt.IsZero() {
			sentAt = time.Now().UTC()
		}
		if err := recorder.RecordShare(ctx, Share{MessageID: input.MessageID, ChatID: input.ChatID,
			UserID: input.UserID, UserName: input.UserName, SentAt: sentAt, URL: link, Provider: p.Host()}); err != nil {
			slog.ErrorContext(ctx, "Не удалось сохранить отправку ссылки", "provider", p.Host(), "url", link, "error", err)
		}
	}
	cached, err := s.store.FindByURL(ctx, p.Host(), link)
	status := "miss"
	if err == nil && cached != nil && !cached.CheckedAt.IsZero() {
		response := CloneMessage(cached)
		response.URL, response.Provider, response.CacheStatus = link, p.Host(), "hit"
		// Do not expose a private chat's original sender in another chat.
		if cached.ChatID != input.ChatID {
			response.UserID, response.UserName, response.FirstSeenAt = 0, "", time.Time{}
		}
		response.Refreshing = s.refresh(cached)
		return response, nil
	}
	if err != nil && !errors.Is(err, ErrCacheMiss) {
		status = "error"
		slog.ErrorContext(ctx, "Не удалось прочитать кеш", "provider", p.Host(), "url", link, "error", err)
	}
	m := CloneMessage(input)
	m.URL, m.Provider, m.CacheStatus = link, p.Host(), status
	m.CacheKey = p.Host() + "|" + link
	if m.FirstSeenAt.IsZero() {
		m.FirstSeenAt = time.Now().UTC()
	}
	// A legacy record knows its sender but has no complete set of links yet.
	if cached != nil {
		m.CacheKey, m.UserID, m.UserName, m.ChatID, m.FirstSeenAt = cached.CacheKey, cached.UserID, cached.UserName, cached.ChatID, cached.FirstSeenAt
		m.MessageID = cached.MessageID
	}
	if err := s.resolve(ctx, m, p); err != nil {
		return nil, err
	}
	if err := s.store.Save(ctx, m); err != nil {
		slog.ErrorContext(ctx, "Не удалось сохранить кеш", "provider", p.Host(), "url", link, "error", err)
	}
	if cached != nil {
		m.SeenBefore = cached.ChatID != 0 && cached.ChatID == input.ChatID
		if !m.SeenBefore {
			m.UserID, m.UserName, m.FirstSeenAt = 0, "", time.Time{}
		}
	}
	return m, nil
}

func (s *service) resolve(ctx context.Context, m *Message, source provider.Provider) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	title, err := source.GetTitle(m.URL)
	if err != nil {
		return fmt.Errorf("get track title: %w", err)
	}
	m.Title = title
	links := append([]Link(nil), m.Links...)
	m.MissingProviders = nil
	links = setLink(links, Link{URL: m.URL, Provider: source.Host(), Name: source.Name()})
	for _, p := range s.providers {
		if p.Host() == source.Host() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		raw, err := p.GetURL(title)
		if errors.Is(err, provider.ErrURLNotFound) || err == nil && raw == "" {
			links = removeLink(links, p.Host())
			m.MissingProviders = append(m.MissingProviders, p.Host())
			continue
		}
		if err != nil {
			slog.WarnContext(ctx, "Провайдер не вернул ссылку", "provider", p.Host(), "error", err)
			continue
		}
		u, err := provider.NormalizeURL(raw)
		if err != nil || u.Host != p.Host() {
			slog.WarnContext(ctx, "Провайдер вернул некорректную ссылку", "provider", p.Host())
			continue
		}
		links = setLink(links, Link{URL: u.String(), Provider: p.Host(), Name: p.Name()})
	}
	m.Links, m.CheckedAt = links, time.Now().UTC()
	return nil
}

func (s *service) refresh(cached *Message) bool {
	if s.ctx.Err() != nil {
		return false
	}
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return false
	}
	if s.refreshing[cached.CacheKey] {
		s.mu.Unlock()
		return true
	}
	select {
	case s.slots <- struct{}{}:
	default:
		s.mu.Unlock()
		return false
	}
	s.refreshing[cached.CacheKey] = true
	s.workers.Add(1)
	s.mu.Unlock()
	go func() {
		defer s.workers.Done()
		defer func() {
			s.mu.Lock()
			delete(s.refreshing, cached.CacheKey)
			s.mu.Unlock()
			<-s.slots
		}()
		ctx, cancel := context.WithTimeout(s.ctx, 45*time.Second)
		defer cancel()
		m := CloneMessage(cached)
		p, err := s.findProvider(m.Provider)
		if err == nil {
			err = s.resolve(ctx, m, p)
		}
		if err == nil {
			err = s.store.Save(ctx, m)
		}
		if err != nil {
			if ctx.Err() != nil {
				slog.DebugContext(ctx, "Фоновая проверка отменена", "provider", m.Provider)
				return
			}
			slog.WarnContext(ctx, "Фоновая проверка не завершена; кеш сохранён", "provider", m.Provider, "url", m.URL, "error", err)
			return
		}
		slog.InfoContext(ctx, "Кеш ссылок обновлён", "provider", m.Provider, "url", m.URL, "links", len(m.Links))
	}()
	return true
}

func (s *service) Close() error {
	s.mu.Lock()
	s.closed = true
	s.cancel()
	s.mu.Unlock()
	s.workers.Wait()
	return nil
}

func (s *service) GetList(ctx context.Context, userID int) (string, error) {
	messages, err := s.store.FindByUser(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("find user's tracks: %w", err)
	}
	if len(messages) == 0 {
		return "", ErrLinksNotFound
	}
	var b strings.Builder
	for _, m := range messages {
		fmt.Fprintf(&b, "%s\n%s\n", m.Title, m.URL)
	}
	return b.String(), nil
}

func (s *service) findProvider(host string) (provider.Provider, error) {
	for _, p := range s.providers {
		if p.Host() == host {
			return p, nil
		}
	}
	return nil, ErrProviderNotFound
}

func CloneMessage(m *Message) *Message {
	copy := *m
	copy.Links = append([]Link(nil), m.Links...)
	copy.MissingProviders = append([]string(nil), m.MissingProviders...)
	return &copy
}

// MergeCache preserves the first sender and successful links during partial refreshes.
func MergeCache(old, fresh *Message) *Message {
	if fresh.CheckedAt.Before(old.CheckedAt) {
		result := CloneMessage(old)
		mergeFirstSender(result, fresh)
		return result
	}
	result := CloneMessage(fresh)
	result.CacheKey = old.CacheKey
	result.UserID, result.UserName, result.ChatID, result.FirstSeenAt = old.UserID, old.UserName, old.ChatID, old.FirstSeenAt
	result.MessageID = old.MessageID
	mergeFirstSender(result, fresh)
	links := append([]Link(nil), old.Links...)
	for _, link := range fresh.Links {
		links = setLink(links, link)
	}
	for _, host := range fresh.MissingProviders {
		links = removeLink(links, host)
	}
	result.Links = links
	return result
}

func mergeFirstSender(result, incoming *Message) {
	if !incoming.FirstSeenAt.IsZero() && (result.FirstSeenAt.IsZero() || incoming.FirstSeenAt.Before(result.FirstSeenAt)) {
		result.UserID, result.UserName, result.ChatID = incoming.UserID, incoming.UserName, incoming.ChatID
		result.FirstSeenAt, result.MessageID = incoming.FirstSeenAt, incoming.MessageID
	}
}

func setLink(links []Link, link Link) []Link {
	for i := range links {
		if links[i].Provider == link.Provider {
			links[i] = link
			return links
		}
	}
	return append(links, link)
}

func removeLink(links []Link, host string) []Link {
	result := links[:0]
	for _, link := range links {
		if link.Provider != host {
			result = append(result, link)
		}
	}
	return result
}

func parseURL(raw string) (*url.URL, error) { return provider.NormalizeURL(raw) }

var linkPattern = regexp.MustCompile(`https?://[^\s<>"']+`)

func extractLink(message string) (string, error) {
	link := strings.TrimRight(linkPattern.FindString(message), ".,;!")
	for strings.HasSuffix(link, ")") && strings.Count(link, ")") > strings.Count(link, "(") {
		link = strings.TrimSuffix(link, ")")
	}
	return link, nil
}
