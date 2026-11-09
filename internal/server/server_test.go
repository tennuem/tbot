package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/stretchr/testify/require"
	"github.com/tennuem/tbot/pkg/provider"
	"github.com/tennuem/tbot/pkg/service"
)

func TestTelegramStartupErrorRedactsBotToken(t *testing.T) {
	err := errors.New(`Post "https://api.telegram.org/bot123:private-token/getMe": network unavailable`)
	message := telegramErrorText(err, "123:private-token")
	require.NotContains(t, message, "123:private-token")
	require.Contains(t, message, "[REDACTED]")
	require.Contains(t, message, "network unavailable")
	require.Equal(t, err.Error(), telegramErrorText(err, ""))
}

func TestServeTelegramSendsGroupCommandsAndSkipsIgnoredUpdates(t *testing.T) {
	updates := make(chan tgbotapi.Update, 5)
	updates <- tgbotapi.Update{} // Service and non-message updates have no reply.
	updates <- tgbotapi.Update{Message: &tgbotapi.Message{}}
	updates <- commandUpdate("/help@tbot")
	updates <- commandUpdate("/help@other_bot")
	updates <- tgbotapi.Update{Message: &tgbotapi.Message{Chat: &tgbotapi.Chat{ID: 42}}}
	close(updates)
	sender := &recordingTelegramSender{}
	handler := service.NewTelegramHandler(nil, nil, "tbot")
	err := serveTelegram(context.Background(), sender, updates, handler, "token")
	require.EqualError(t, err, "telegram updates channel closed")
	require.Len(t, sender.sent, 1)
	require.Contains(t, sender.sent[0].Text, "/help")
	require.Equal(t, int64(42), sender.sent[0].ChatID)
	require.Equal(t, 12, sender.sent[0].ReplyToMessageID)
}

func TestServeTelegramPreservesTrackReplyFormatting(t *testing.T) {
	svc := service.NewService(context.Background(), service.NewStoreMock(), provider.NewMockProvider())
	defer svc.Close()
	update := commandUpdate("/help")
	update.Message.Text = "https://example.com/track/1"
	update.Message.Entities = nil
	updates := make(chan tgbotapi.Update, 1)
	updates <- update
	close(updates)
	sender := &recordingTelegramSender{}
	err := serveTelegram(context.Background(), sender, updates, service.NewTelegramHandler(svc, nil, "tbot"), "token")
	require.EqualError(t, err, "telegram updates channel closed")
	require.Len(t, sender.sent, 1)
	require.Contains(t, sender.sent[0].Text, "<b>test_title</b>")
	require.Equal(t, "HTML", sender.sent[0].ParseMode)
	require.True(t, sender.sent[0].DisableWebPagePreview)
	require.Equal(t, 12, sender.sent[0].ReplyToMessageID)
}

func TestServeTelegramLogsSendErrorsWithoutTokenAndContinues(t *testing.T) {
	var output bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&output, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })
	updates := make(chan tgbotapi.Update, 2)
	updates <- commandUpdate("/ping")
	updates <- commandUpdate("/help")
	close(updates)
	sender := &recordingTelegramSender{err: errors.New(`Post "https://api.telegram.org/bot123:private-token/sendMessage": request failed`)}
	err := serveTelegram(context.Background(), sender, updates, service.NewTelegramHandler(nil, nil, "tbot"), "123:private-token")
	require.EqualError(t, err, "telegram updates channel closed")
	require.Len(t, sender.sent, 2)
	require.Contains(t, sender.sent[0].Text, "pong")
	require.Contains(t, output.String(), "Не удалось отправить ответ Telegram")
	require.Contains(t, output.String(), "chat_id=42")
	require.Contains(t, output.String(), "[REDACTED]")
	require.NotContains(t, output.String(), "private-token")
}

func TestServeTelegramStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sender := &recordingTelegramSender{}
	err := serveTelegram(ctx, sender, make(chan tgbotapi.Update), nil, "token")
	require.NoError(t, err)
	require.Empty(t, sender.sent)
}

type recordingTelegramSender struct {
	sent []tgbotapi.MessageConfig
	err  error
}

func (s *recordingTelegramSender) Send(message tgbotapi.Chattable) (tgbotapi.Message, error) {
	s.sent = append(s.sent, message.(tgbotapi.MessageConfig))
	return tgbotapi.Message{}, s.err
}

func commandUpdate(text string) tgbotapi.Update {
	entities := []tgbotapi.MessageEntity{{Type: "bot_command", Offset: 0, Length: len(text)}}
	return tgbotapi.Update{Message: &tgbotapi.Message{
		MessageID: 12, Text: text, Entities: &entities,
		Chat: &tgbotapi.Chat{ID: 42, Type: "supergroup"}, From: &tgbotapi.User{ID: 10},
	}}
}
