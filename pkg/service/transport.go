package service

import (
	"context"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/tennuem/telegram"
)

var commandHelp = `
/help - list commands description
/ping - liveness probe
/list - get list of tracks
`

func NewTelegramHandler(svc Service) telegram.Handler {
	mux := telegram.NewServeMux()
	mux.HandleFunc("/help", func(w *telegram.ResponseWriter, r *telegram.Request) {
		w.Text = commandHelp
	})
	mux.HandleFunc("/ping", func(w *telegram.ResponseWriter, r *telegram.Request) {
		w.Text = "pong"
	})
	mux.HandleFunc("*", func(w *telegram.ResponseWriter, r *telegram.Request) {
		resp, err := svc.FindLinks(context.Background(), &Message{
			URL:      r.Message.Text,
			Username: r.Message.From.UserName,
		})
		if err != nil {
			w.Text = err.Error()
			return
		}
		var buttons []tgbotapi.InlineKeyboardButton
		for _, v := range resp.Links {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonURL(v.Provider, v.URL))
		}
		w.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(buttons...),
		)
		w.Text = resp.Title
	})
	mux.HandleFunc("/list", func(w *telegram.ResponseWriter, r *telegram.Request) {
		username := r.Message.CommandArguments()
		username = strings.TrimPrefix(username, "@")
		if len(username) == 0 {
			username = r.Message.From.UserName
		}
		resp, err := svc.GetList(context.Background(), username)
		if err != nil {
			w.Text = err.Error()
			return
		}
		w.Text = resp
	})
	return mux
}
