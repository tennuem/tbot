package server

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
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
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()
	s.group.Add(func() error { <-ctx.Done(); return nil }, func(error) { stop() })
	cfg := configs.NewConfig()
	if err := cfg.Read(); err != nil {
		return errors.Wrap(err, "failed to init config")
	}
	var level slog.Level
	if err := level.UnmarshalText([]byte(cfg.Logger.Level)); err != nil {
		return errors.Wrap(err, "invalid logger level")
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})))
	yandex, err := provider.NewYandexProvider(ctx, cfg.Yandex.ProxyURL)
	if err != nil {
		return errors.Wrap(err, "failed to init Yandex Music provider")
	}
	ms, err := sqlite.NewSqLiteStore(cfg.SqLite.DataSource)
	if err != nil {
		return errors.Wrap(err, "failed to init sqLite store")
	}
	svc := service.NewService(
		ctx,
		ms,
		yandex,
		provider.NewYoutubeProvider(ctx),
		provider.NewAppleProvider(ctx),
		provider.NewSpotifyProvider(ctx, cfg.Spotify.ClientID, cfg.Spotify.ClientSecret),
	)
	defer svc.Close()
	svc = service.NewLoggingService(ctx, svc)

	icons, err := service.LoadServiceIcons(ctx, cfg.Telegram.Token, cfg.Telegram.EmojiSet)
	if err != nil {
		slog.Warn("Не удалось загрузить логотипы сервисов, используются текстовые ссылки", "error", err)
	} else if cfg.Telegram.EmojiSet != "" {
		slog.Info("Логотипы сервисов загружены", "emoji_set", cfg.Telegram.EmojiSet, "icons", len(icons))
	}
	if err := s.bot(cfg.Telegram.Token, service.NewTelegramHandler(svc, icons)); err != nil {
		return errors.Wrap(err, "failed to init telegram bot")
	}
	return s.group.Run()
}

func (s *server) bot(token string, handler telegram.Handler) error {
	svr, close, err := telegram.NewServer(token)
	if err != nil {
		return errors.Errorf("start telegram server: %s", telegramErrorText(err, token))
	}
	s.group.Add(func() error {
		slog.Info("Telegram-бот запущен")
		return svr.Serve(handler)
	}, func(error) {
		slog.Info("Telegram-бот остановлен")
		close()
	})
	return nil
}

func telegramErrorText(err error, token string) string {
	message := err.Error()
	if token != "" {
		message = strings.ReplaceAll(message, token, "[REDACTED]")
	}
	return message
}
