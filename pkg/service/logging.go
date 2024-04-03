package service

import (
	"context"
	"log"
	"time"
)

func NewLoggingService(ctx context.Context, s Service) Service {
	return &loggingService{s}
}

type loggingService struct {
	Service
}

func (s *loggingService) FindLinks(ctx context.Context, m *Message) (res *Message, err error) {
	defer func(begin time.Time) {
		log.Println(
			"method", "FindLinks",
			"url", m.URL,
			"username", m.Username,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.Service.FindLinks(ctx, m)
}

func (s *loggingService) GetList(ctx context.Context, username string) (res string, err error) {
	defer func(begin time.Time) {
		log.Println(
			"method", "GetList",
			"username", username,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.Service.GetList(ctx, username)
}
