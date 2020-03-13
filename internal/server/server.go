package server

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/configs"
	"github.com/tennuem/tbot/internal/store"
	"github.com/tennuem/tbot/pkg/bot"
	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/tbot/pkg/service"
	"github.com/tennuem/tbot/tools/logger"
)

func NewServer() *Server {
	cfg := configs.NewConfig()
	if err := cfg.Read(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to init config: %s", err)
		os.Exit(1)
	}
	if err := cfg.Print(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to print config: %s", err)
		os.Exit(1)
	}
	logger, err := logger.NewLogger(cfg.Logger.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init logger: %s", err)
		os.Exit(1)
	}
	ms, err := store.NewMongoStore(cfg.MongoDB.Addr)
	if err != nil {
		level.Error(logger).Log("err", errors.Wrap(err, "failed to init mongo client"))
		os.Exit(1)
	}
	svc := service.NewService(
		ms,
		map[string]provider.Provider{
			"music.yandex.com":  provider.NewYandexProvider(log.With(logger, "component", "yandex")),
			"music.youtube.com": provider.NewYoutubeProvider(log.With(logger, "component", "youtube")),
			"music.apple.com":   provider.NewAppleProvider(log.With(logger, "component", "apple")),
		},
		log.With(logger, "component", "service"),
	)
	svr := Server{
		svc:    svc,
		logger: log.With(logger, "component", "server"),
	}
	svr.runBot(cfg.Telegram.Token)
	svr.runSignalHandler()
	return &svr
}

type Server struct {
	svc    service.Service
	logger log.Logger
	group  run.Group
}

func (s *Server) Run() error {
	return s.logger.Log("exit", s.group.Run())
}

func (s *Server) runBot(token string) {
	b, err := bot.NewTelegramBot(token, log.With(s.logger, "component", "tbot"))
	if err != nil {
		level.Error(s.logger).Log("err", errors.Wrap(err, "failed to init telegram bot"))
		os.Exit(1)
	}
	s.group.Add(func() error {
		level.Info(s.logger).Log("msg", "start telegram bot")
		return b.Listen(s.svc)
	}, func(error) {
		b.Close()
	})
}

func (s *Server) runSignalHandler() {
	ch := make(chan struct{})
	s.group.Add(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		select {
		case sig := <-c:
			return errors.Errorf("received signal %s\n", sig)
		case <-ch:
			return nil
		}
	}, func(error) {
		close(ch)
	})
}
