// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"
	"io"
	"strings"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
)

// API describes the process-instance operations implemented by the v8.9 adapter.
type API interface {
	// ValidateProcessInstanceResumption checks mutation capability without I/O.
	ValidateProcessInstanceResumption() error
	// ResumeProcessInstance resumes exactly one key, confirming unless NoWait is set.
	ResumeProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error)
	// ValidateProcessInstanceSuspension checks mutation capability without I/O.
	ValidateProcessInstanceSuspension() error
	// SuspendProcessInstance suspends exactly one key, confirming unless NoWait is set.
	SuspendProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error)
	// ValidateProcessInstanceStates checks explicit state capabilities without I/O.
	ValidateProcessInstanceStates(states d.States) error
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

type GenProcessInstanceClientCamunda interface {
	CancelProcessInstanceWithResponse(ctx context.Context, processInstanceKey string, body camundav89.CancelProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CancelProcessInstanceResponse, error)
	CreateProcessInstanceWithResponse(ctx context.Context, body camundav89.CreateProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.CreateProcessInstanceResponse, error)
	DeleteProcessInstanceWithResponse(ctx context.Context, processInstanceKey camundav89.ProcessInstanceKey, body camundav89.DeleteProcessInstanceJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.DeleteProcessInstanceResponse, error)
	GetProcessInstanceWithResponse(ctx context.Context, processInstanceKey camundav89.ProcessInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetProcessInstanceResponse, error)
	SearchProcessInstancesWithBodyWithResponse(ctx context.Context, contentType string, body io.Reader, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchProcessInstancesResponse, error)
}

var _ GenProcessInstanceClientCamunda = (*camundav89.ClientWithResponses)(nil)

// ValidateProcessInstanceStates rejects explicit suspension on this older adapter.
// Implicit creation visibility uses the generic waiter and bypasses this guard.
func (s *Service) ValidateProcessInstanceStates(states d.States) error {
	for _, state := range states {
		if strings.EqualFold(strings.TrimSpace(string(state)), string(d.StateSuspended)) {
			return fmt.Errorf("%w: SUSPENDED process-instance state requires Camunda 8.10 (configured 8.9)", d.ErrUnsupported)
		}
	}
	return nil
}
