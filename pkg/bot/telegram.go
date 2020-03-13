package bot

import (
	"context"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/pkg/service"
)

var ErrTokenNotFound = errors.New("token not found")

type Bot interface {
	Listen(service.Service) error
	Close()
}

func NewTelegramBot(token string, logger log.Logger) (Bot, error) {
	if len(token) == 0 {
		return nil, ErrTokenNotFound
	}
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &tbot{
		tgbotapi.UpdateConfig{
			Offset:  0,
			Timeout: 60,
		},
		api,
		logger,
		make(chan struct{}),
	}, nil
}

type tbot struct {
	cfg    tgbotapi.UpdateConfig
	api    *tgbotapi.BotAPI
	logger log.Logger
	stop   chan struct{}
}

func (b *tbot) Listen(svc service.Service) error {
	for {
		select {
		case <-b.stop:
			return nil
		default:
		}
		updates, err := b.api.GetUpdates(b.cfg)
		if err != nil {
			return errors.Wrap(err, "failed to get updates")
		}
		for _, update := range updates {
			if update.UpdateID >= b.cfg.Offset {
				b.cfg.Offset = update.UpdateID + 1
			}
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
				b.api.Send(msg)
				continue
			}
			resp, err := svc.GetLinks(context.TODO(), update.Message.Text)
			if err != nil {
				level.Error(b.logger).Log("err", errors.Errorf("get links from message %s: %v", update.Message.Text, err))
				continue
			}
			msg.Text = strings.Join(resp, "\n\n")
			b.api.Send(msg)
		}
	}
}

func (b *tbot) Close() {
	close(b.stop)
}
