package service

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/pkg/provider"
)

var (
	ErrProviderNotFound = errors.New("provider not found")
)

type Service interface {
	FindLinks(ctx context.Context, m *Message) (*Message, error)
	GetList(ctx context.Context, username string) (string, error)
}

type Message struct {
	URL      string   `bson:"url"`
	Title    string   `bson:"title"`
	Links    []string `bson:"links,omitempty"`
	Username string   `bson:"username"`
}

type Store interface {
	Save(ctx context.Context, m *Message) error
	FindByURL(ctx context.Context, url string) (*Message, error)
	FindByUsername(ctx context.Context, username string) ([]Message, error)
}

func NewService(s Store, p map[string]provider.Provider, logger log.Logger) Service {
	return &service{s, p, logger}
}

type service struct {
	store     Store
	providers map[string]provider.Provider
	logger    log.Logger
}

func (s *service) FindLinks(ctx context.Context, m *Message) (*Message, error) {
	p, err := s.findProvider(m.URL)
	if err != nil {
		return nil, err
	}
	msg, err := s.store.FindByURL(ctx, m.URL)
	if err != nil {
		level.Error(s.logger).Log("err", err)
	}
	if msg != nil {
		return nil, errors.Errorf("@%s has already share it", msg.Username)
	}
	title, err := p.GetTitle(m.URL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get title")
	}
	m.Title = title
	var res []string
	for k, v := range s.providers {
		if v == p {
			continue
		}
		u, err := v.GetURL(title)
		if err != nil {
			level.Error(s.logger).Log(fmt.Sprintf("failed to get url from: %v, by title %v", k, err.Error()))
			continue
		}
		if u != "" {
			res = append(res, u)
		}
	}
	m.Links = res
	if err := s.store.Save(ctx, m); err != nil {
		level.Error(s.logger).Log("err", err)
	}
	return m, nil
}

func (s *service) GetList(ctx context.Context, username string) (string, error) {
	m, err := s.store.FindByUsername(ctx, username)
	if err != nil {
		return "", err
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

func (s *service) findProvider(url string) (provider.Provider, error) {
	u, err := parseURL(url)
	if err != nil {
		return nil, errors.Wrap(err, "parse url")
	}
	v, ok := s.providers[u.Host]
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
