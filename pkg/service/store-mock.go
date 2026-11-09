package service

import (
	"context"
	"sync"
)

func NewStoreMock() Store { return &storeMock{m: make(map[string]*Message)} }

type storeMock struct {
	mu sync.Mutex
	m  map[string]*Message
}

func (s *storeMock) Save(ctx context.Context, m *Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	copy := CloneMessage(m)
	if old := s.m[m.CacheKey]; old != nil {
		copy = MergeCache(old, m)
	}
	s.m[copy.CacheKey] = copy
	return nil
}

func (s *storeMock) FindByURL(ctx context.Context, host, url string) (*Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.m {
		for _, link := range m.Links {
			if link.Provider == host && link.URL == url {
				return CloneMessage(m), nil
			}
		}
	}
	return nil, ErrCacheMiss
}

func (s *storeMock) FindByUser(ctx context.Context, userID int) ([]Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Message
	for _, m := range s.m {
		if m.UserID == userID {
			result = append(result, *CloneMessage(m))
		}
	}
	return result, nil
}
