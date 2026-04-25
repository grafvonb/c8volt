package common

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
)

var (
	ErrNilConfig          = errors.New("config must not be nil")
	ErrEmptyCamundaURL    = errors.New("camunda base URL must not be empty")
	ErrNilContext         = errors.New("context must not be nil")
	ErrNoClientConfigured = errors.New("client must not be nil")
)

type ServiceDeps struct {
	Config     *config.Config
	HTTPClient *http.Client
	Logger     *slog.Logger
}

// PrepareServiceDeps validates the common constructor inputs shared by versioned services.
// cfg must include a Camunda base URL; nil httpClient and log are replaced with default implementations.
func PrepareServiceDeps(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (ServiceDeps, error) {
	if cfg == nil {
		return ServiceDeps{}, ErrNilConfig
	}
	if cfg.APIs.Camunda.BaseURL == "" {
		return ServiceDeps{}, ErrEmptyCamundaURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if log == nil {
		log = slog.Default()
	}
	return ServiceDeps{Config: cfg, HTTPClient: httpClient, Logger: log}, nil
}

// EffectiveTenant returns the configured tenant or an empty string for nil configs.
// It is used by request builders where an empty tenant means "do not add a tenant filter".
func EffectiveTenant(cfg *config.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.App.Tenant
}

// EnsureLoggerAndClients validates service dependencies after constructor options have been applied.
// logger is defaulted when nil; at least one non-nil generated client must be supplied.
func EnsureLoggerAndClients(logger *slog.Logger, clients ...any) (*slog.Logger, error) {
	if logger == nil {
		logger = slog.Default()
	}
	if len(clients) == 0 {
		return nil, ErrNoClientConfigured
	}
	for _, client := range clients {
		if client == nil {
			return nil, ErrNoClientConfigured
		}
	}
	return logger, nil
}
