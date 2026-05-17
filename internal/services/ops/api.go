// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	csvc "github.com/grafvonb/c8volt/internal/services/cluster"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/toolx"
)

type API interface {
	ExecuteSmokeTest(ctx context.Context, request d.SmokeTestRequest, opts ...services.CallOption) (d.SmokeTestResult, error)
	PurgeOrphanProcessInstances(ctx context.Context, request d.OrphanPurgeRequest, opts ...services.CallOption) (d.OrphanPurgeResult, error)
	ExecuteRetentionPolicy(ctx context.Context, request d.RetentionPolicyRequest, opts ...services.CallOption) (d.RetentionPolicyResult, error)
	PurgeProcessInstancesWithIncidents(ctx context.Context, request d.IncidentPurgeRequest, opts ...services.CallOption) (d.IncidentPurgeResult, error)
	PurgeAllProcessDefinitions(ctx context.Context, request d.AllProcessDefinitionsPurgeRequest, opts ...services.CallOption) (d.AllProcessDefinitionsPurgeResult, error)
}

type Service struct {
	clusterAPI  csvc.API
	piAPI       pisvc.API
	incAPI      incsvc.API
	pdAPI       pdsvc.API
	resourceAPI rsvc.API
	version     toolx.CamundaVersion
	log         *slog.Logger
}

func New(piAPI pisvc.API, incAPI incsvc.API, loggers ...*slog.Logger) API {
	log := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &Service{piAPI: piAPI, incAPI: incAPI, version: toolx.CurrentCamundaVersion, log: log}
}

func NewWithProcessDefinitionPurge(piAPI pisvc.API, incAPI incsvc.API, pdAPI pdsvc.API, resourceAPI rsvc.API, loggers ...*slog.Logger) API {
	log := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	return &Service{piAPI: piAPI, incAPI: incAPI, pdAPI: pdAPI, resourceAPI: resourceAPI, version: toolx.CurrentCamundaVersion, log: log}
}

func NewWithWorkflowDependencies(clusterAPI csvc.API, piAPI pisvc.API, incAPI incsvc.API, pdAPI pdsvc.API, resourceAPI rsvc.API, version toolx.CamundaVersion, loggers ...*slog.Logger) API {
	log := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	return &Service{clusterAPI: clusterAPI, piAPI: piAPI, incAPI: incAPI, pdAPI: pdAPI, resourceAPI: resourceAPI, version: version, log: log}
}

var _ API = (*Service)(nil)
