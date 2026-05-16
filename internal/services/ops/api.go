// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

type API interface {
	PurgeOrphanProcessInstances(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error)
	ExecuteRetentionPolicy(ctx context.Context, request d.RetentionPolicyRequest, opts ...services.CallOption) (d.RetentionPolicyResult, error)
	PurgeProcessInstancesWithIncidents(ctx context.Context, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentPurgeResult, error)
}

type Service struct {
	piAPI  pisvc.API
	incAPI incsvc.API
	log    *slog.Logger
}

func New(piAPI pisvc.API, incAPI incsvc.API, loggers ...*slog.Logger) API {
	log := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &Service{piAPI: piAPI, incAPI: incAPI, log: log}
}

var _ API = (*Service)(nil)
