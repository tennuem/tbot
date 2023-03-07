package provider

import "errors"

type Provider interface {
	Host() string
	GetTitle(url string) (string, error)
	GetURL(title string) (string, error)
}

var (
	ErrTitleNotFound = errors.New("title not found")
	ErrURLNotFound   = errors.New("URL not found")
)
