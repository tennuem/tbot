package service

import (
	"context"
	"sync"

	"github.com/pkg/errors"
)

func NewStoreMock() Store {
	return &storeMock{
		m: map[string]*Message{
			"https://music.yandex.com/album/8508157/track/57016085": &Message{
				URL:   "https://music.yandex.com/album/8508157/track/57016085",
				Title: "Babushka Boi — A$AP Rocky",
				Links: []Link{
					{Provider: "youtube", URL: "https://music.youtube.com/watch?v=KViOTZ62zBg"},
					{Provider: "apple", URL: "https://music.apple.com/us/album/babushka-boi-single/1477644647"},
				},
			},
		},
	}
}

type storeMock struct {
	mu sync.Mutex
	m  map[string]*Message
}

func (s *storeMock) Save(ctx context.Context, m *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return nil
}

func (s *storeMock) FindByURL(ctx context.Context, url string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.m[url]
	if !ok {
		return nil, errors.New("message not found")
	}
	return m, nil
}

func (s *storeMock) FindByUsername(ctx context.Context, username string) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var res []Message
	for _, m := range s.m {
		if m.Username == username {
			res = append(res, *m)
		}
	}
	return res, nil
}
