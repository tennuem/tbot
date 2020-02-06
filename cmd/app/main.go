package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/tennuem/tbot/pkg/provider"
)

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		panic(errors.New("TELEGRAM_TOKEN is required"))
	}

	svc := provider.NewService(map[string]provider.Provider{
		"music.yandex.com":  provider.NewYandexProvider(),
		"music.youtube.com": provider.NewYoutubeProvider(),
		"music.apple.com":   provider.NewAppleProvider(),
	})

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		panic(err)
	}
	bot.Debug = true

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		panic(err)
	}
	for update := range updates {
		if update.Message == nil {
			continue
		}
		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "help":
				msg.Text = "type /ping."
			case "ping":
				msg.Text = "pong"
			default:
				msg.Text = "I don't know that command"
			}
			bot.Send(msg)
		}
		res, err := svc.GetLinks(update.Message.Text)
		if err != nil {
			fmt.Printf("get links from message %s: %v", update.Message.Text, err)
			continue
		}
		bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, strings.Join(res, "\n\n")))
	}
}
