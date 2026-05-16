// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
)

type API interface {
	PurgeOrphanProcessInstances(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error)
	ExecuteRetentionPolicy(ctx context.Context, request d.RetentionPolicyRequest, opts ...services.CallOption) (d.RetentionPolicyResult, error)
	PurgeProcessInstancesWithIncidents(ctx context.Context, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentPurgeResult, error)
	PurgeAllProcessDefinitions(ctx context.Context, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error)
}

type Service struct {
	piAPI       pisvc.API
	incAPI      incsvc.API
	pdAPI       pdsvc.API
	resourceAPI rsvc.API
	log         *slog.Logger
}

func New(piAPI pisvc.API, incAPI incsvc.API, loggers ...*slog.Logger) API {
	log := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &Service{piAPI: piAPI, incAPI: incAPI, log: log}
}

func NewWithProcessDefinitionPurge(piAPI pisvc.API, incAPI incsvc.API, pdAPI pdsvc.API, resourceAPI rsvc.API, loggers ...*slog.Logger) API {
	log := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &Service{piAPI: piAPI, incAPI: incAPI, pdAPI: pdAPI, resourceAPI: resourceAPI, log: log}
}

var _ API = (*Service)(nil)
