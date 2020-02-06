package provider

import (
	"log"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

var (
	ErrEmptyMessage     = errors.New("Message is empty")
	ErrProviderNotFound = errors.New("provider not found")
	ErrTitleNotFound    = errors.New("title not found")
	ErrURLNotFound      = errors.New("URL not found")
)

type Provider interface {
	GetTitle(url string) (string, error)
	GetURL(title string) (string, error)
}

type Service interface {
	GetLinks(msg string) ([]string, error)
}

func NewService(p map[string]Provider) Service {
	return &service{p}
}

type service struct {
	providers map[string]Provider
}

func (s *service) GetLinks(msg string) ([]string, error) {
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
			log.Printf("failed to get url from: %v, by title %v", k, err.Error())
			continue
		}
		if u != "" {
			res = append(res, u)
		}
	}
	return res, nil
}

func (s *service) getProvider(host string) Provider {
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
