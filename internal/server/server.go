package server

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/configs"
	"github.com/tennuem/tbot/internal/bot"
	"github.com/tennuem/tbot/internal/store"
	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/tbot/pkg/service"
	"github.com/tennuem/tbot/tools/logging"
)

type Server interface {
	Run(ctx context.Context) error
}

func NewServer() Server {
	return &server{}
}

type server struct {
	group run.Group
}

func (s *server) Run(ctx context.Context) error {
	s.signal()
	cfg := configs.NewConfig()
	if err := cfg.Read(); err != nil {
		return errors.Wrap(err, "failed to init config")
	}
	if err := cfg.Print(); err != nil {
		return errors.Wrap(err, "failed to print config")
	}
	logger, err := logging.NewLogger(cfg.Logger.Level)
	if err != nil {
		return errors.Wrap(err, "failed to init logger")
	}
	ctx = logging.WithContext(ctx, logger)
	ms, err := store.NewMongoStore(cfg.MongoDB.Addr)
	if err != nil {
		return errors.Wrap(err, "failed to init mongo store")
	}
	svc := service.NewService(ctx, ms)
	svc.AddProvider(provider.NewYandexProvider(ctx))
	svc.AddProvider(provider.NewYoutubeProvider(ctx))
	svc.AddProvider(provider.NewAppleProvider(ctx))
	svc.AddProvider(provider.NewSpotifyProvider(ctx, cfg.Spotify.ClientID, cfg.Spotify.ClientSecret))
	svc = service.NewLoggingService(ctx, svc)
	if err := s.bot(ctx, cfg.Telegram.Token, service.MakeBotHandler(svc, logger)); err != nil {
		return errors.Wrap(err, "failed to init telegram bot")
	}
	return logger.Log("exit", s.group.Run())
}

func (s *server) bot(ctx context.Context, token string, handler bot.Handler) error {
	logger := logging.FromContext(ctx)
	logger = log.With(logger, "component", "telegram bot")
	listener, err := bot.Listen(token)
	if err != nil {
		return errors.Wrap(err, "start listen")
	}
	s.group.Add(func() error {
		level.Info(logger).Log("msg", "start listen")
		return listener.Listen(handler)
	}, func(error) {
		level.Info(logger).Log("msg", "stop listen")
		listener.Close()
	})
	return nil
}

func (s *server) signal() {
	c := make(chan os.Signal, 1)
	s.group.Add(func() error {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			return errors.Errorf("received signal %s", sig)
		}
	}, func(error) {
		close(c)
	})
}
