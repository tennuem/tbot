package bot

import (
	"context"
	"sync"

	"github.com/go-kit/kit/endpoint"
	"github.com/go-kit/kit/log"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/pkg/errors"
)

var (
	ErrTokenNotFound = errors.New("token not found")
)

type Listener interface {
	Listen(handler Handler) error
	Close()
}

type Handler interface {
	Serve(ResponseWriter, *Request)
}

func Listen(token string) (Listener, error) {
	if len(token) == 0 {
		return nil, ErrTokenNotFound
	}
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	lc := &listenConfig{
		tgbotapi.UpdateConfig{
			Offset:  0,
			Timeout: 60,
		},
		api,
		make(chan struct{}),
	}
	return lc, nil
}

type listenConfig struct {
	cfg  tgbotapi.UpdateConfig
	api  *tgbotapi.BotAPI
	stop chan struct{}
}

func (l *listenConfig) Listen(handler Handler) error {
	for {
		select {
		case <-l.stop:
			return nil
		default:
		}
		updates, err := l.api.GetUpdates(l.cfg)
		if err != nil {
			return err
		}
		for _, update := range updates {
			if update.UpdateID >= l.cfg.Offset {
				l.cfg.Offset = update.UpdateID + 1
			}
			if update.Message == nil {
				continue
			}
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			handler.Serve(
				ResponseWriter{&msg},
				&Request{context.Background(), update},
			)
			l.api.Send(msg)
		}
	}
}

func (l *listenConfig) Close() {
	close(l.stop)
}

type ResponseWriter struct {
	*tgbotapi.MessageConfig
}

type Request struct {
	ctx context.Context
	tgbotapi.Update
}

func (r *Request) Context() context.Context {
	return r.ctx
}

func (w ResponseWriter) Write(b []byte) {
	w.Text += string(b)
}

type ServerMux struct {
	mu sync.Mutex
	m  map[string]Handler
}

type HandlerFunc func(ResponseWriter, *Request)

func (f HandlerFunc) Serve(w ResponseWriter, r *Request) {
	f(w, r)
}

func NotFound(w ResponseWriter, r *Request) { w.Write([]byte("command not found")) }

func NotFoundHandler() Handler { return HandlerFunc(NotFound) }

func NewServeMux() *ServerMux {
	return &ServerMux{m: make(map[string]Handler)}
}

func (mux *ServerMux) Handle(command string, handler Handler) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.m[command] = handler
}

func (mux *ServerMux) HandleFunc(command string, handler func(ResponseWriter, *Request)) {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	mux.m[command] = HandlerFunc(handler)
}

func (mux *ServerMux) Handler(r *Request) Handler {
	mux.mu.Lock()
	defer mux.mu.Unlock()
	v, ok := mux.m[r.Message.Text]
	if !ok {
		v, ok := mux.m["*"]
		if !ok {
			return NotFoundHandler()
		}
		return v
	}
	return v
}

func (mux *ServerMux) Serve(w ResponseWriter, r *Request) {
	h := mux.Handler(r)
	h.Serve(w, r)
}

type DecodeRequestFunc func(context.Context, *Request) (request interface{}, err error)

type EncodeRequestFunc func(context.Context, *Request, interface{}) error

type EncodeResponseFunc func(context.Context, ResponseWriter, interface{}) error

type Server struct {
	e      endpoint.Endpoint
	dec    DecodeRequestFunc
	enc    EncodeResponseFunc
	logger log.Logger
}

func NewServer(
	e endpoint.Endpoint,
	dec DecodeRequestFunc,
	enc EncodeResponseFunc,
	logger log.Logger,
) *Server {
	return &Server{e, dec, enc, logger}
}

func (s Server) Serve(w ResponseWriter, r *Request) {
	ctx := r.Context()
	request, err := s.dec(ctx, r)
	if err != nil {
		s.logger.Log("err", err)
		return
	}
	response, err := s.e(ctx, request)
	if err != nil {
		s.logger.Log("err", err)
		return
	}
	if err := s.enc(ctx, w, response); err != nil {
		s.logger.Log("err", err)
		return
	}
}
