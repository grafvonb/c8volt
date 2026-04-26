// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	types "github.com/grafvonb/c8volt/typex"
)

type TraversalOutcome string

const (
	TraversalOutcomeComplete   TraversalOutcome = "complete"
	TraversalOutcomePartial    TraversalOutcome = "partial"
	TraversalOutcomeUnresolved TraversalOutcome = "unresolved"
)

type TraversalMode string

const (
	TraversalModeAncestry    TraversalMode = "ancestry"
	TraversalModeDescendants TraversalMode = "descendants"
	TraversalModeFamily      TraversalMode = "family"
)

type MissingAncestor struct {
	Key      string
	StartKey string
}

type TraversalResult struct {
	Mode             TraversalMode
	StartKey         string
	RootKey          string
	Keys             []string
	Edges            map[string][]string
	Chain            map[string]ProcessInstance
	MissingAncestors []MissingAncestor
	Warning          string
	Outcome          TraversalOutcome
}

func (r TraversalResult) HasActionableResults() bool {
	return len(r.Keys) > 0 || len(r.Chain) > 0
}

type DryRunPIKeyExpansion struct {
	Roots            types.Keys
	Collected        types.Keys
	MissingAncestors []MissingAncestor
	Warning          string
	Outcome          TraversalOutcome
}

func (r DryRunPIKeyExpansion) HasActionableResults() bool {
	return len(r.Roots) > 0 || len(r.Collected) > 0
}

type API interface {
	SearchProcessDefinitions(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error)
	SearchProcessDefinitionsLatest(ctx context.Context, filter ProcessDefinitionFilter, opts ...options.FacadeOption) (ProcessDefinitions, error)
	GetProcessDefinition(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessDefinition, error)
	GetProcessDefinitionXML(ctx context.Context, key string, opts ...options.FacadeOption) (string, error)

	CreateProcessInstance(ctx context.Context, data ProcessInstanceData, opts ...options.FacadeOption) (ProcessInstance, error)
	CreateProcessInstances(ctx context.Context, datas []ProcessInstanceData, opts ...options.FacadeOption) ([]ProcessInstance, error)
	GetProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error)
	LookupProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstance, error)
	LookupProcessInstanceStateByKey(ctx context.Context, key string, opts ...options.FacadeOption) (StateReport, ProcessInstance, error)
	SearchProcessInstancesPage(ctx context.Context, filter ProcessInstanceFilter, page ProcessInstancePageRequest, opts ...options.FacadeOption) (ProcessInstancePage, error)
	SearchProcessInstances(ctx context.Context, filter ProcessInstanceFilter, size int32, opts ...options.FacadeOption) (ProcessInstances, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (CancelReport, ProcessInstances, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (DeleteReport, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstances, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []ProcessInstance, opts ...options.FacadeOption) ([]ProcessInstance, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired States, opts ...options.FacadeOption) (StateReport, ProcessInstance, error)
	Walker

	GetProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstances, error)
	CreateNProcessInstances(ctx context.Context, data ProcessInstanceData, n int, wantedWorkers int, opts ...options.FacadeOption) ([]ProcessInstance, error)
	CancelProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (CancelReports, error)
	DeleteProcessInstances(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (DeleteReports, error)
	WaitForProcessInstancesState(ctx context.Context, keys types.Keys, desired States, wantedWorkers int, opts ...options.FacadeOption) (StateReports, error)

	DryRunCancelOrDeleteGetPIKeys(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (types.Keys, types.Keys, error)
	DryRunCancelOrDeletePlan(ctx context.Context, keys types.Keys, opts ...options.FacadeOption) (DryRunPIKeyExpansion, error)
}

var _ API = (*client)(nil)
