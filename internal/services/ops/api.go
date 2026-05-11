// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

type API interface {
	PurgeOrphanProcessInstances(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error)
}

type Service struct {
	piAPI pisvc.API
}

func New(piAPI pisvc.API) API {
	return &Service{piAPI: piAPI}
}

func (s *Service) PurgeOrphanProcessInstances(_ context.Context, request d.OrphanPurgeRequest, _ ...services.CallOption) (d.OrphanPurgeResult, error) {
	result := d.OrphanPurgeResult{
		Request: request,
		Report: d.OrphanPurgeReport{
			CommandName:      request.CommandName,
			StartedAt:        request.StartedAt,
			DryRun:           request.DryRun,
			AutoConfirm:      request.AutoConfirm,
			Automation:       request.Automation,
			SelectionFilters: request.Selection,
			Outcome:          d.OrphanPurgeOutcomeFailed,
		},
		Outcome: d.OrphanPurgeOutcomeFailed,
	}
	return result, fmt.Errorf("%w: ops orphan process-instance purge is not implemented yet", d.ErrUnsupported)
}

var _ API = (*Service)(nil)
