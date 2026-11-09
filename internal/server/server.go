package server

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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
	defer ms.Close()
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
	if err := s.bot(ctx, cfg.Telegram.Token, svc, icons); err != nil {
		return errors.Wrap(err, "failed to init telegram bot")
	}
	return s.group.Run()
}

func (s *server) bot(ctx context.Context, token string, svc service.Service, icons service.ServiceIcons) error {
	api, err := tgbotapi.NewBotAPIWithClient(token, &http.Client{Timeout: 70 * time.Second})
	if err != nil {
		return errors.Errorf("start telegram server: %s", telegramErrorText(err, token))
	}
	config := tgbotapi.NewUpdate(0)
	config.Timeout = 60
	updates, err := api.GetUpdatesChan(config)
	if err != nil {
		return errors.Errorf("receive telegram updates: %s", telegramErrorText(err, token))
	}
	handler := service.NewTelegramHandler(svc, icons, api.Self.UserName)
	ctx, cancel := context.WithCancel(ctx)
	s.group.Add(func() error {
		slog.Info("Telegram-бот запущен")
		return serveTelegram(ctx, api, updates, handler, token)
	}, func(error) {
		cancel()
		api.StopReceivingUpdates()
		slog.Info("Telegram-бот остановлен")
	})
	return nil
}

type telegramSender interface {
	Send(tgbotapi.Chattable) (tgbotapi.Message, error)
}

func serveTelegram(ctx context.Context, api telegramSender, updates tgbotapi.UpdatesChannel, handler telegram.Handler, token string) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case update, ok := <-updates:
			if !ok {
				return errors.New("telegram updates channel closed")
			}
			if update.Message == nil || update.Message.Chat == nil {
				continue
			}
			response := &telegram.ResponseWriter{MessageConfig: tgbotapi.NewMessage(update.Message.Chat.ID, "")}
			handler.Serve(response, &telegram.Request{Update: update})
			if response.Text == "" {
				continue
			}
			if _, err := api.Send(response.MessageConfig); err != nil {
				slog.ErrorContext(ctx, "Не удалось отправить ответ Telegram", "chat_id", update.Message.Chat.ID,
					"error", telegramErrorText(err, token))
			}
		}
	}
}

func telegramErrorText(err error, token string) string {
	message := err.Error()
	if token != "" {
		message = strings.ReplaceAll(message, token, "[REDACTED]")
	}
	return message
}
