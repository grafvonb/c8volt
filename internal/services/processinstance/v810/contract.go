// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"io"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
)

// API describes the process-instance operations implemented by the native v8.10 adapter.
type API interface {
	CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error)
	GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error)
	SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error)
	UpdateProcessInstanceVariables(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error)
	SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error)
	SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error)
	GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired d.States, opts ...services.CallOption) (d.StateResponse, d.ProcessInstance, error)
	WaitForProcessInstanceExpectation(ctx context.Context, key string, request d.ProcessInstanceExpectationRequest, opts ...services.CallOption) (d.ProcessInstanceExpectationResponse, d.ProcessInstance, error)
	Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error)
	Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	Family(ctx context.Context, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	AncestryResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error)
	DescendantsResult(ctx context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error)
	FamilyResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error)
	GetProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error)
	WaitForProcessInstancesState(ctx context.Context, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error)
	WaitForProcessInstancesExpectation(ctx context.Context, keys typex.Keys, request d.ProcessInstanceExpectationRequest, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error)
}

// GenProcessInstanceClientCamunda captures the generated Camunda calls used by the v8.10 process-instance service.
type GenProcessInstanceClientCamunda interface {
	CancelProcessInstanceWithResponse(ctx context.Context, processInstanceKey string, body camundav810.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CancelProcessInstanceResponse, error)
	CreateProcessInstanceWithResponse(ctx context.Context, body camundav810.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.CreateProcessInstanceResponse, error)
	DeleteProcessInstanceWithResponse(ctx context.Context, processInstanceKey camundav810.ProcessInstanceKey, body camundav810.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav810.RequestEditorFn) (*camundav810.DeleteProcessInstanceResponse, error)
	GetProcessInstanceWithResponse(ctx context.Context, processInstanceKey camundav810.ProcessInstanceKey, reqEditors ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error)
	SearchProcessInstancesWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav810.RequestEditorFn) (*camundav810.SearchProcessInstancesResponse, error)
}

var _ GenProcessInstanceClientCamunda = (*camundav810.ClientWithResponses)(nil)
