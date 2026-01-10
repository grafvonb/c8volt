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

type CamundaDeps struct {
	Config     *config.Config
	HTTPClient *http.Client
	Logger     *slog.Logger
}

func PrepareCamundaDeps(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (CamundaDeps, error) {
	if cfg == nil {
		return CamundaDeps{}, ErrNilConfig
	}
	if cfg.APIs.Camunda.BaseURL == "" {
		return CamundaDeps{}, ErrEmptyCamundaURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	if log == nil {
		log = slog.Default()
	}
	return CamundaDeps{Config: cfg, HTTPClient: httpClient, Logger: log}, nil
}

func EnsureLoggerAndClients(logger *slog.Logger, clients ...interface{}) (*slog.Logger, error) {
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
