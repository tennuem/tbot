package service

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/tennuem/tbot/tools/logging"
)

// NewLoggingService returns a new instance of a logging Service.
func NewLoggingService(ctx context.Context, s Service) Service {
	logger := logging.FromContext(ctx)
	logger = log.With(logger, "component", "service")
	return &loggingService{logger, s}
}

type loggingService struct {
	logger log.Logger
	Service
}

func (s *loggingService) FindLinks(ctx context.Context, m *Message) (res *Message, err error) {
	defer func(begin time.Time) {
		level.Info(s.logger).Log(
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
		level.Info(s.logger).Log(
			"method", "GetList",
			"username", username,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return s.Service.GetList(ctx, username)
}
