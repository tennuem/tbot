package service

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

func NewLoggingService(ctx context.Context, s Service) Service { return &loggingService{s} }

type loggingService struct{ Service }

func (s *loggingService) FindLinks(ctx context.Context, m *Message) (*Message, error) {
	begin := time.Now()
	result, err := s.Service.FindLinks(ctx, m)
	if errors.Is(err, ErrLinkNotFound) {
		return result, err // Ordinary chat messages are not errors or log events.
	}
	attrs := []any{"user_id", m.UserID, "duration_ms", time.Since(begin).Milliseconds()}
	if err != nil {
		slog.WarnContext(ctx, "Не удалось обработать ссылку", append(attrs, "error", err)...)
		return result, err
	}
	attrs = append(attrs, "provider", result.Provider, "url", result.URL, "cache", result.CacheStatus, "links", len(result.Links))
	message := "Ссылка не найдена в кеше; ссылки получены от провайдеров"
	switch result.CacheStatus {
	case "hit":
		message = "Готовый ответ получен из кеша"
		attrs = append(attrs, "refreshing", result.Refreshing)
	case "error":
		message = "Кеш недоступен; ссылки получены от провайдеров"
	}
	slog.InfoContext(ctx, message, attrs...)
	return result, nil
}

func (s *loggingService) GetList(ctx context.Context, userID int) (string, error) {
	result, err := s.Service.GetList(ctx, userID)
	if err != nil && !errors.Is(err, ErrLinksNotFound) {
		slog.ErrorContext(ctx, "Не удалось получить список треков", "user_id", userID, "error", err)
	}
	return result, err
}
