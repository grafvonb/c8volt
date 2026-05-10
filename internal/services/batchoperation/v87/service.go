// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
)

type Service struct {
	cfg *config.Config
	log *slog.Logger
}

func New(cfg *config.Config, httpClient *http.Client, log *slog.Logger) (*Service, error) {
	deps, err := common.PrepareServiceDeps(cfg, httpClient, log)
	if err != nil {
		return nil, err
	}
	return &Service{cfg: deps.Config, log: deps.Logger}, nil
}

func (s *Service) CheckReadAccess(ctx context.Context, opts ...services.CallOption) error {
	_ = ctx
	_ = services.ApplyCallOptions(opts)

	return fmt.Errorf("%w: batch-operation read access checks require Camunda 8.8 or newer", d.ErrUnsupported)
}

func (s *Service) CancelProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, opts ...services.CallOption) (d.BatchOperation, error) {
	_ = ctx
	_ = filter
	_ = services.ApplyCallOptions(opts)

	return d.BatchOperation{}, fmt.Errorf("%w: process-instance cancellation batch operations require Camunda 8.8 or newer", d.ErrUnsupported)
}

func (s *Service) WaitForCompletion(ctx context.Context, batchOperationKey string, opts ...services.CallOption) (d.BatchOperation, error) {
	_ = ctx
	_ = batchOperationKey
	_ = services.ApplyCallOptions(opts)

	return d.BatchOperation{}, fmt.Errorf("%w: batch-operation completion waiting requires Camunda 8.8 or newer", d.ErrUnsupported)
}
