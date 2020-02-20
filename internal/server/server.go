package server

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/pkg/bot"
	"github.com/tennuem/tbot/pkg/provider"
)

func NewServer() *Server {
	var logger log.Logger
	logger = log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		level.Error(logger).Log("err", "TELEGRAM_TOKEN is required")
		os.Exit(1)
	}
	svc := provider.NewService(map[string]provider.Provider{
		"music.yandex.com":  provider.NewYandexProvider(log.With(logger, "component", "yandex")),
		"music.youtube.com": provider.NewYoutubeProvider(log.With(logger, "component", "youtube")),
		"music.apple.com":   provider.NewAppleProvider(log.With(logger, "component", "apple")),
	}, log.With(logger, "component", "service"))
	svr := Server{
		svc:    svc,
		logger: log.With(logger, "component", "server"),
	}
	svr.runBot(token)
	svr.runSignalHandler()
	return &svr
}

type Server struct {
	svc    provider.Service
	logger log.Logger
	group  run.Group
}

func (s *Server) Run() error {
	return s.logger.Log("exit", s.group.Run())
}

func (s *Server) runBot(token string) {
	b, err := bot.NewTelegramBot(token, s.logger)
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
