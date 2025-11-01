package cmd

import (
	"context"
	"fmt"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

type Services struct {
	Config *config.Config
	HTTP   *httpc.Service
}

func NewFromContext(ctx context.Context) (*Services, error) {
	cfg, err := config.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("config from context: %w", err)
	}
	httpSvc, err := httpc.FromContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("http service from context: %w", err)
	}
	return &Services{Config: cfg, HTTP: httpSvc}, nil
}
