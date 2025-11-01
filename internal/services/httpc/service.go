package httpc

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
)

var (
	ErrNoHttpServiceInContext  = errors.New("no http service in context")
	ErrInvalidServiceInContext = errors.New("invalid http service in context")
)

type Service struct {
	c   *http.Client
	cfg *config.Config
	log *slog.Logger
}

type Option func(*Service)

func WithTimeout(d time.Duration) Option {
	return func(s *Service) { s.c.Timeout = d }
}

func WithTimeoutString(v string) Option {
	return func(s *Service) {
		if v == "" {
			return
		}
		if d, err := time.ParseDuration(v); err == nil {
			s.c.Timeout = d
		}
	}
}

// WithCookieJar Ensure cookie jar (needed for XSRF)
func WithCookieJar() Option {
	return func(s *Service) { _ = s.InstallCookieJar() }
}

// WithAuthEditor Install an auth editor transport now
func WithAuthEditor(ed authenticator.RequestEditor) Option {
	return func(s *Service) { s.InstallAuthEditor(ed) }
}

func New(cfg *config.Config, log *slog.Logger, opts ...Option) (*Service, error) {
	if cfg == nil {
		return nil, errors.New("cfg is nil")
	}
	d, err := time.ParseDuration(cfg.HTTP.Timeout)
	if err != nil {
		return nil, err
	}
	httpClient := &http.Client{Timeout: d, Transport: &LogTransport{Log: log, WithBody: false}}
	s := &Service{c: httpClient, cfg: cfg, log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Service) Client() *http.Client { return s.c }

func (s *Service) UseClient(c *http.Client) { s.c = c }

func (s *Service) InstallCookieJar() error {
	if s.c.Jar != nil {
		return nil
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return err
	}
	s.c.Jar = jar
	return nil
}

func (s *Service) InstallAuthEditor(ed authenticator.RequestEditor) {
	s.c.Transport = &AuthTransport{base: s.c.Transport, Editor: ed}
}

type ctxKey struct{}

func (s *Service) ToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

func FromContext(ctx context.Context) (*Service, error) {
	v := ctx.Value(ctxKey{})
	if v == nil {
		return nil, ErrNoHttpServiceInContext
	}
	s, ok := v.(*Service)
	if !ok || s == nil {
		return nil, ErrInvalidServiceInContext
	}
	return s, nil
}

func MustClient(ctx context.Context) *http.Client {
	if s, err := FromContext(ctx); err == nil && s != nil {
		return s.c
	}
	return http.DefaultClient
}
