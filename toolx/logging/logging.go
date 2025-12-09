package logging

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
)

var (
	ErrNoLoggerInContext = errors.New("no logger in context")
)

type LoggerConfig struct {
	Level           string
	Format          string
	WithSource      bool
	WithRequestBody bool
}

type ctxKey struct{}

func New(cfg LoggerConfig) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		lv = slog.LevelDebug
	case "info":
		lv = slog.LevelInfo
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{
		Level:     lv,
		AddSource: cfg.WithSource,
	}
	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(os.Stdout, opts)
	case "plain":
		handler = NewPlainHandler(os.Stdout, opts.Level).
			WithSource(cfg.WithSource).
			WithTimestamp(lv < slog.LevelInfo)
	default:
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return slog.New(handler)
}

func ToContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

func FromContext(ctx context.Context) (*slog.Logger, error) {
	l, ok := ctx.Value(ctxKey{}).(*slog.Logger)
	if !ok || l == nil {
		l.Debug("no logger in context, using default logger")
		return slog.Default(), nil
	}
	return l, nil
}

func InfoV(msg string, log *slog.Logger, verbose bool) {
	if verbose {
		log.Info(msg)
	}
}
