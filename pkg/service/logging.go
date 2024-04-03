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
			"user_id", m.UserID,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.Service.FindLinks(ctx, m)
}

func (s *loggingService) GetList(ctx context.Context, userID int) (res string, err error) {
	defer func(begin time.Time) {
		log.Println(
			"method", "GetList",
			"user_id", userID,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.Service.GetList(ctx, userID)
}
