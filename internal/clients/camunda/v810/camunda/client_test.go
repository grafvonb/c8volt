// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package camunda

import (
	"context"
	"testing"
)

var _ ClientWithResponsesInterface = (*ClientWithResponses)(nil)

type searchBatchOperationsWithResponseFunc func(*ClientWithResponses, context.Context, SearchBatchOperationsJSONRequestBody, ...RequestEditorFn) (*SearchBatchOperationsResponse, error)
type getTopologyWithResponseFunc func(*ClientWithResponses, context.Context, ...RequestEditorFn) (*GetTopologyResponse, error)
type searchElementInstancesWithResponseFunc func(*ClientWithResponses, context.Context, SearchElementInstancesJSONRequestBody, ...RequestEditorFn) (*SearchElementInstancesResponse, error)
type searchIncidentsWithResponseFunc func(*ClientWithResponses, context.Context, SearchIncidentsJSONRequestBody, ...RequestEditorFn) (*SearchIncidentsResponse, error)
type resolveIncidentWithResponseFunc func(*ClientWithResponses, context.Context, IncidentKey, ResolveIncidentJSONRequestBody, ...RequestEditorFn) (*ResolveIncidentResponse, error)
type searchJobsWithResponseFunc func(*ClientWithResponses, context.Context, SearchJobsJSONRequestBody, ...RequestEditorFn) (*SearchJobsResponse, error)
type searchProcessDefinitionsWithResponseFunc func(*ClientWithResponses, context.Context, SearchProcessDefinitionsJSONRequestBody, ...RequestEditorFn) (*SearchProcessDefinitionsResponse, error)
type searchProcessInstancesWithResponseFunc func(*ClientWithResponses, context.Context, SearchProcessInstancesJSONRequestBody, ...RequestEditorFn) (*SearchProcessInstancesResponse, error)
type cancelProcessInstanceWithResponseFunc func(*ClientWithResponses, context.Context, ProcessInstanceKey, CancelProcessInstanceJSONRequestBody, ...RequestEditorFn) (*CancelProcessInstanceResponse, error)
type getResourceWithResponseFunc func(*ClientWithResponses, context.Context, ResourceKey, ...RequestEditorFn) (*GetResourceResponse, error)
type deleteResourceOpWithResponseFunc func(*ClientWithResponses, context.Context, ResourceKey, DeleteResourceOpJSONRequestBody, ...RequestEditorFn) (*DeleteResourceOpResponse, error)
type searchTenantsWithResponseFunc func(*ClientWithResponses, context.Context, SearchTenantsJSONRequestBody, ...RequestEditorFn) (*SearchTenantsResponse, error)
type searchUserTasksWithResponseFunc func(*ClientWithResponses, context.Context, SearchUserTasksJSONRequestBody, ...RequestEditorFn) (*SearchUserTasksResponse, error)
type searchVariablesWithResponseFunc func(*ClientWithResponses, context.Context, *SearchVariablesParams, SearchVariablesJSONRequestBody, ...RequestEditorFn) (*SearchVariablesResponse, error)

var requiredV810Operations = []any{
	NewClientWithResponses,
	searchBatchOperationsWithResponseFunc((*ClientWithResponses).SearchBatchOperationsWithResponse),
	getTopologyWithResponseFunc((*ClientWithResponses).GetTopologyWithResponse),
	searchElementInstancesWithResponseFunc((*ClientWithResponses).SearchElementInstancesWithResponse),
	searchIncidentsWithResponseFunc((*ClientWithResponses).SearchIncidentsWithResponse),
	resolveIncidentWithResponseFunc((*ClientWithResponses).ResolveIncidentWithResponse),
	searchJobsWithResponseFunc((*ClientWithResponses).SearchJobsWithResponse),
	searchProcessDefinitionsWithResponseFunc((*ClientWithResponses).SearchProcessDefinitionsWithResponse),
	searchProcessInstancesWithResponseFunc((*ClientWithResponses).SearchProcessInstancesWithResponse),
	cancelProcessInstanceWithResponseFunc((*ClientWithResponses).CancelProcessInstanceWithResponse),
	getResourceWithResponseFunc((*ClientWithResponses).GetResourceWithResponse),
	deleteResourceOpWithResponseFunc((*ClientWithResponses).DeleteResourceOpWithResponse),
	searchTenantsWithResponseFunc((*ClientWithResponses).SearchTenantsWithResponse),
	searchUserTasksWithResponseFunc((*ClientWithResponses).SearchUserTasksWithResponse),
	searchVariablesWithResponseFunc((*ClientWithResponses).SearchVariablesWithResponse),
}

var requiredV810ResultTypes = []any{
	BatchOperationSearchQueryResult{},
	TopologyResponse{},
	ElementInstanceSearchQueryResult{},
	IncidentSearchQueryResult{},
	JobSearchQueryResult{},
	ProcessDefinitionSearchQueryResult{},
	ProcessInstanceSearchQueryResult{},
	ResourceResult{},
	TenantSearchQueryResult{},
	UserTaskSearchQueryResult{},
	VariableSearchQueryResult{},
}

func TestV810GeneratedClientRequiredSymbols(t *testing.T) {
	if len(requiredV810Operations) != 15 {
		t.Fatalf("expected 15 required operations, got %d", len(requiredV810Operations))
	}
	if len(requiredV810ResultTypes) != 11 {
		t.Fatalf("expected 11 required result types, got %d", len(requiredV810ResultTypes))
	}
}

func TestV810GeneratedClientConstructsWithResponses(t *testing.T) {
	client, err := NewClientWithResponses("https://example.com")
	if err != nil {
		t.Fatalf("NewClientWithResponses() error = %v", err)
	}
	if client == nil {
		t.Fatal("NewClientWithResponses() returned nil client")
	}
}
