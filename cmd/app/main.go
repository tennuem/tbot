package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/tennuem/tbot/pkg/provider"
)

func main() {
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

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
	bot.Debug = false

	updates, err := bot.GetUpdatesChan(tgbotapi.UpdateConfig{
		Offset:  0,
		Timeout: 60,
	})
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "start service")
	for update := range updates {
		if update.Message == nil {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "help":
				msg.Text = "type /ping."
			case "ping":
				msg.Text = "pong"
			default:
				msg.Text = "I don't know that command"
			}
		}
		res, err := svc.GetLinks(update.Message.Text)
		if err != nil {
			level.Error(logger).Log("err", fmt.Sprintf("get links from message %s: %v", update.Message.Text, err))
			continue
		}
		msg.Text = strings.Join(res, "\n\n")
		bot.Send(msg)
	}
	level.Info(logger).Log("msg", "stop service")
}
