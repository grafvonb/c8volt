// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	csvc "github.com/grafvonb/c8volt/internal/services/cluster"
	esvc "github.com/grafvonb/c8volt/internal/services/element"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	jsvc "github.com/grafvonb/c8volt/internal/services/job"
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
	RepairIncidents(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error)
	RepairProcessInstances(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error)
	AnalyseSlowProcessInstances(ctx context.Context, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (d.SlowProcessAnalysisResult, error)
}

type Service struct {
	clusterAPI  csvc.API
	elementAPI  esvc.API
	piAPI       pisvc.API
	incAPI      incsvc.API
	jobAPI      jsvc.API
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
	return &Service{piAPI: piAPI, incAPI: incAPI, pdAPI: pdAPI, resourceAPI: resourceAPI, version: toolx.V89, log: log}
}

// NewWithWorkflowDependencies creates an ops service with cross-resource workflow dependencies.
func NewWithWorkflowDependencies(clusterAPI csvc.API, piAPI pisvc.API, incAPI incsvc.API, pdAPI pdsvc.API, resourceAPI rsvc.API, version toolx.CamundaVersion, loggers ...*slog.Logger) API {
	return NewWithRepairDependencies(clusterAPI, piAPI, incAPI, pdAPI, resourceAPI, nil, version, loggers...)
}

// NewWithRepairDependencies creates an ops service with job support for repair workflows.
func NewWithRepairDependencies(clusterAPI csvc.API, piAPI pisvc.API, incAPI incsvc.API, pdAPI pdsvc.API, resourceAPI rsvc.API, jobAPI jsvc.API, version toolx.CamundaVersion, loggers ...*slog.Logger) API {
	return NewWithAnalysisDependencies(clusterAPI, piAPI, incAPI, pdAPI, resourceAPI, jobAPI, nil, version, loggers...)
}

// NewWithAnalysisDependencies creates an ops service with runtime element support for read-only analysis workflows.
func NewWithAnalysisDependencies(clusterAPI csvc.API, piAPI pisvc.API, incAPI incsvc.API, pdAPI pdsvc.API, resourceAPI rsvc.API, jobAPI jsvc.API, elementAPI esvc.API, version toolx.CamundaVersion, loggers ...*slog.Logger) API {
	log := slog.Default()
	if len(loggers) > 0 && loggers[0] != nil {
		log = loggers[0]
	}
	if version == "" {
		version = toolx.CurrentCamundaVersion
	}
	return &Service{clusterAPI: clusterAPI, elementAPI: elementAPI, piAPI: piAPI, incAPI: incAPI, pdAPI: pdAPI, resourceAPI: resourceAPI, jobAPI: jobAPI, version: version, log: log}
}

var _ API = (*Service)(nil)
