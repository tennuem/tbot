package logging

import (
	"context"
	"fmt"
	"os"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
)

type loggerKey struct{}

var fallbackLogger log.Logger

func init() {
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = level.NewFilter(logger, level.AllowDebug())
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	fallbackLogger = logger
}

func WithContext(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func FromContext(ctx context.Context) log.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(log.Logger); ok {
		return logger
	}
	return fallbackLogger
}

func NewLogger(lvl string) (log.Logger, error) {
	levelOption, err := getLevel(lvl)
	if err != nil {
		return nil, errors.Wrap(err, "get level")
	}
	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	logger = level.NewFilter(logger, levelOption)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	return logger, nil
}

func getLevel(lvl string) (level.Option, error) {
	switch lvl {
	case "emerg":
		return level.AllowError(), nil
	case "alert":
		return level.AllowError(), nil
	case "crit":
		return level.AllowError(), nil
	case "err":
		return level.AllowError(), nil
	case "warning":
		return level.AllowWarn(), nil
	case "notice":
		return level.AllowInfo(), nil
	case "info":
		return level.AllowInfo(), nil
	case "debug":
		return level.AllowDebug(), nil
	default:
		return nil, fmt.Errorf("level %s is incorrect. Level can be (emerg, alert, crit, err, warn, notice, info, debug)", lvl)
	}
}
