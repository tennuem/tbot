package service

import (
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
	ErrEmptyMessage     = errors.New("Message is empty")
	ErrProviderNotFound = errors.New("provider not found")
)

type Service interface {
	GetLinks(ctx context.Context, msg string) ([]string, error)
}

type Model struct {
	Msg   string   `bson:"msg"`
	Title string   `bson:"title"`
	URL   []string `bson:"url,omitempty"`
}

type Store interface {
	// Save saves all urls for a message.
	Save(ctx context.Context, m *Model) error
	// FindByMsg find urls by message.
	FindByMsg(ctx context.Context, msg string) *Model
}

func NewService(s Store, p map[string]provider.Provider, logger log.Logger) Service {
	return &service{s, p, logger}
}

type service struct {
	store     Store
	providers map[string]provider.Provider
	logger    log.Logger
}

func (s *service) GetLinks(ctx context.Context, msg string) ([]string, error) {
	if msg == "" {
		return nil, ErrEmptyMessage
	}
	u, err := parseMsg(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse msg")
	}
	p := s.getProvider(u.Host)
	if p == nil {
		return nil, ErrProviderNotFound
	}
	if res := s.store.FindByMsg(ctx, msg); res != nil {
		return res.URL, nil
	}
	title, err := p.GetTitle(msg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get title")
	}
	var res []string
	for k, p := range s.providers {
		if k == u.Host {
			continue
		}
		u, err := p.GetURL(title)
		if err != nil {
			level.Error(s.logger).Log(fmt.Sprintf("failed to get url from: %v, by title %v", k, err.Error()))
			continue
		}
		if u != "" {
			res = append(res, u)
		}
	}
	if err := s.store.Save(ctx, &Model{msg, title, res}); err != nil {
		level.Error(s.logger).Log("err", errors.Wrap(err, "failed to save model"))
	}
	return res, nil
}

func (s *service) getProvider(host string) provider.Provider {
	v, ok := s.providers[host]
	if !ok {
		return nil
	}
	return v
}

func parseMsg(msg string) (*url.URL, error) {
	u, err := url.Parse(msg)
	if err != nil {
		return nil, err
	}
	if strings.HasSuffix(u.Host, ".ru") {
		u.Host = strings.Replace(u.Host, ".ru", ".com", -1)
	}
	return u, nil
}
