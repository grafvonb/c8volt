package logging

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
)

var (
	ErrNoLoggerInContext = errors.New("no logger in context")
)

type LoggerConfig struct {
	Level           string
	Format          string
	WithSource      bool
	WithRequestBody bool
	Writer          io.Writer
}

type ctxKey struct{}

type synchronizedWriter struct {
	w  io.Writer
	mu sync.Mutex
}

func (w *synchronizedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}

func ensureSynchronizedWriter(w io.Writer) io.Writer {
	if w == nil {
		return nil
	}
	if _, ok := w.(*synchronizedWriter); ok {
		return w
	}
	return &synchronizedWriter{w: w}
}

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
	writer := cfg.Writer
	if writer == nil {
		writer = os.Stderr
	}
	writer = ensureSynchronizedWriter(writer)
	var handler slog.Handler
	switch strings.ToLower(cfg.Format) {
	case "json":
		handler = slog.NewJSONHandler(writer, opts)
	case "plain":
		handler = NewPlainHandler(writer, opts.Level).
			WithSource(cfg.WithSource).
			WithTimestamp(lv < slog.LevelInfo)
	default:
		handler = slog.NewTextHandler(writer, opts)
	}
	return slog.New(handler)
}

func ToContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

func FromContext(ctx context.Context) (*slog.Logger, error) {
	l, ok := ctx.Value(ctxKey{}).(*slog.Logger)
	if !ok || l == nil {
		return slog.Default(), nil
	}
	return l, nil
}

func InfoIfVerbose(msg string, log *slog.Logger, verbose bool) {
	if verbose {
		log.Info(msg)
	}
}

func InfoOrVerbose(info, vinfo string, log *slog.Logger, verbose bool) {
	if verbose {
		log.Info(vinfo)
	} else {
		log.Info(info)
	}
}
