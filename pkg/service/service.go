package service

import (
	"bytes"
	"context"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/tennuem/tbot/pkg/provider"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
	ErrLinkNotFound     = errors.New("link not found in message")
	ErrLinksNotFound    = errors.New("links not found")
	ErrHasAlreadyShare  = errors.New("already share it")
)

type Service interface {
	FindLinks(ctx context.Context, m *Message) (*Message, error)
	GetList(ctx context.Context, userID int) (string, error)
	AddProvider(p provider.Provider)
}

type Link struct {
	URL      string
	Provider string
}

type Message struct {
	URL    string `bson:"url"`
	Title  string `bson:"title"`
	Links  []Link `bson:"links,omitempty"`
	UserID int    `bson:"user_id"`
}

type Store interface {
	Save(ctx context.Context, m *Message) error
	FindByURL(ctx context.Context, url string) (*Message, error)
	FindByUser(ctx context.Context, userID int) ([]Message, error)
}

func NewService(ctx context.Context, s Store) Service {
	return &service{store: s}
}

type service struct {
	store     Store
	providers map[string]provider.Provider
}

func (s *service) FindLinks(ctx context.Context, m *Message) (*Message, error) {
	link, err := extractLink(m.URL)
	if err != nil || len(link) == 0 {
		return nil, ErrLinkNotFound
	}
	u, err := parseURL(link)
	if err != nil {
		return nil, errors.Wrap(err, "parse url")
	}
	if u.Host == "link.spotify.com" {
		u.Host = strings.Replace(u.Host, "link", "open", -1)
	}
	p, err := s.findProvider(u.Host)
	if err != nil {
		return nil, err
	}
	link = u.String()
	msg, err := s.store.FindByURL(ctx, link)
	if err != nil {
		log.Println("find url in store err:", err)
	}
	if msg != nil {
		return nil, ErrHasAlreadyShare
	}
	title, err := p.GetTitle(link)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get title")
	}
	m.Title = title
	var res []Link
	for k, v := range s.providers {
		if v == p {
			continue
		}
		u, err := v.GetURL(title)
		if err != nil {
			log.Printf("failed to get url for provider: %s, by title: %s err: %v\n", k, title, err.Error())
			continue
		}
		if len(u) == 0 {
			continue
		}
		res = append(res, Link{Provider: v.Name(), URL: u})
	}
	m.URL = link
	m.Links = res
	if err := s.store.Save(ctx, m); err != nil {
		log.Println("save result to store:", err)
	}
	return m, nil
}

func (s *service) GetList(ctx context.Context, userID int) (string, error) {
	m, err := s.store.FindByUser(ctx, userID)
	if err != nil {
		return "", errors.Wrap(err, "find by username in store")
	}
	if m == nil {
		return "", ErrLinksNotFound
	}
	var b bytes.Buffer
	for _, v := range m {
		b.WriteString(v.Title)
		b.WriteRune('\n')
		b.WriteString(v.URL)
		b.WriteRune('\n')
	}
	return b.String(), nil
}

func (s *service) AddProvider(p provider.Provider) {
	if s.providers == nil {
		s.providers = make(map[string]provider.Provider)
	}
	s.providers[p.Host()] = p
}

func (s *service) findProvider(host string) (provider.Provider, error) {
	v, ok := s.providers[host]
	if !ok {
		return nil, ErrProviderNotFound
	}
	return v, nil
}

func parseURL(s string) (*url.URL, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(u.Host, ".ru") {
		u.Host = strings.Replace(u.Host, ".ru", ".com", -1)
	}
	return u, nil
}

func extractLink(msg string) (string, error) {
	expr := "https?:\\/\\/(www\\.)?[-a-zA-Z0-9@:%._\\+~#=]{2,256}\\.[a-z]{2,4}\\b([-a-zA-Z0-9@:%_\\+.~#?&//=]*)"
	reg, err := regexp.Compile(expr)
	if err != nil {
		return "", err
	}
	return reg.FindString(msg), nil
}
