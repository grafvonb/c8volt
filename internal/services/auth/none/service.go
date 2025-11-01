package none

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/internal/services/auth/authenticator"
)

var _ authenticator.Authenticator = (*Service)(nil)

type Service struct {
	log *slog.Logger
}

type Option func(*Service)

func New(log *slog.Logger, opts ...Option) (*Service, error) {
	if log == nil {
		return nil, errors.New("logger must not be nil")
	}
	log.Debug("Using 'none' authenticator: no authentication will be performed")

	s := &Service{log: log}
	for _, opt := range opts {
		opt(s)
	}
	return s, nil
}

func (s *Service) Name() string { return "none" }

func (s *Service) IsAuthenticated() bool { return true }

func (s *Service) Init(_ context.Context) error {
	return nil
}

func (s *Service) Editor() authenticator.RequestEditor {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Accept", "application/json")
		return nil
	}
}

func (s *Service) ClearCache() {
}
