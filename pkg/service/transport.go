package service

import (
	"context"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/tennuem/tbot/internal/bot"
)

func MakeBotHandler(s Service, logger log.Logger) bot.Handler {
	h := bot.NewServeMux()
	h.Handle("*", bot.NewServer(
		makeFindLinksEndpoint(s),
		decodeFindLinksRequest,
		encodeFindLinksResponse,
		logger,
	))
	h.Handle("/list", bot.NewServer(
		makeGetListEndpoint(s),
		decodeGetListRequest,
		encodeGetListResponse,
		logger,
	))
	h.HandleFunc("/help", func(w bot.ResponseWriter, r *bot.Request) {
		w.Write([]byte("/ping - heathcheck\n/list - get list music\n"))
	})
	h.HandleFunc("/ping", func(w bot.ResponseWriter, r *bot.Request) {
		w.Write([]byte("pong"))
	})
	return h
}

func decodeFindLinksRequest(_ context.Context, r *bot.Request) (interface{}, error) {
	return FindLinksRequest{
		URL:      r.Message.Text,
		Username: r.Message.From.UserName,
	}, nil
}

func encodeFindLinksResponse(ctx context.Context, w bot.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	resp := response.(FindLinksResponse)
	w.Write([]byte(strings.Join(resp.Links, "\n\n")))
	return nil
}

func decodeGetListRequest(_ context.Context, r *bot.Request) (interface{}, error) {
	username := r.Message.CommandArguments()
	if strings.HasPrefix(username, "@") {
		username = strings.TrimPrefix(username, "@")
	}
	if len(username) == 0 {
		username = r.Message.From.UserName
	}
	return GetListRequest{
		Username: username,
	}, nil
}

func encodeGetListResponse(ctx context.Context, w bot.ResponseWriter, response interface{}) error {
	if e, ok := response.(errorer); ok && e.error() != nil {
		encodeError(ctx, e.error(), w)
		return nil
	}
	resp := response.(GetListResponse)
	w.Write([]byte(resp.Msg))
	return nil
}

type errorer interface {
	error() error
}

func encodeError(_ context.Context, err error, w bot.ResponseWriter) {
	switch errors.Cause(err) {
	case ErrProviderNotFound:
		return
	case ErrLinkNotFound:
		return
	default:
		w.Write([]byte(err.Error()))
	}
}
