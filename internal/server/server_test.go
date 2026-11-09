package server

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTelegramStartupErrorRedactsBotToken(t *testing.T) {
	err := errors.New(`Post "https://api.telegram.org/bot123:private-token/getMe": network unavailable`)
	message := telegramErrorText(err, "123:private-token")
	require.NotContains(t, message, "123:private-token")
	require.Contains(t, message, "[REDACTED]")
	require.Contains(t, message, "network unavailable")
	require.Equal(t, err.Error(), telegramErrorText(err, ""))
}
