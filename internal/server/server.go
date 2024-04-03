package server

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/configs"
	"github.com/tennuem/tbot/internal/store/sqlite"
	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/tbot/pkg/service"
	"github.com/tennuem/telegram"
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
	ms, err := sqlite.NewSqLiteStore(cfg.SqLite.DataSource)
	if err != nil {
		return errors.Wrap(err, "failed to init sqLite store")
	}
	svc := service.NewService(ctx, ms)
	svc.AddProvider(provider.NewYandexProvider(ctx))
	svc.AddProvider(provider.NewYoutubeProvider(ctx))
	svc.AddProvider(provider.NewAppleProvider(ctx))
	svc.AddProvider(provider.NewSpotifyProvider(ctx, cfg.Spotify.ClientID, cfg.Spotify.ClientSecret))
	svc = service.NewLoggingService(ctx, svc)

	if err := s.bot(cfg.Telegram.Token, service.NewTelegramHandler(svc)); err != nil {
		return errors.Wrap(err, "failed to init telegram bot")
	}
	return s.group.Run()
}

func (s *server) bot(token string, handler telegram.Handler) error {
	svr, close, err := telegram.NewServer(token)
	if err != nil {
		return errors.Wrap(err, "start telegram server")
	}
	s.group.Add(func() error {
		log.Println("start telegram server")
		return svr.Serve(handler)
	}, func(error) {
		log.Println("stop telegram server")
		close()
	})
	return nil
}

func (s *server) signal() {
	c := make(chan os.Signal, 1)
	s.group.Add(func() error {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		return errors.Errorf("received signal %s", <-c)
	}, func(error) {
		close(c)
	})
}
