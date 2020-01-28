package main

import (
	"errors"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func main() {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		panic(errors.New("TELEGRAM_TOKEN is required"))
	}

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
	}
}
