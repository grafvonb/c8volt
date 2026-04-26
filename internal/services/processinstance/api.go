// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	v87 "github.com/grafvonb/c8volt/internal/services/processinstance/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/processinstance/v88"
	v89 "github.com/grafvonb/c8volt/internal/services/processinstance/v89"
	"github.com/grafvonb/c8volt/typex"
)

type API interface {
	CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error)
	GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error)
	GetDirectChildrenOfProcessInstance(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstance, error)
	FilterProcessInstanceWithOrphanParent(ctx context.Context, items []d.ProcessInstance, opts ...services.CallOption) ([]d.ProcessInstance, error)
	SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error)
	SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error)
	CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error)
	GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error)
	WaitForProcessInstanceState(ctx context.Context, key string, desired d.States, opts ...services.CallOption) (d.StateResponse, d.ProcessInstance, error)
	Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (rootKey string, path []string, chain map[string]d.ProcessInstance, err error)
	Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) (desc []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	Family(ctx context.Context, startKey string, opts ...services.CallOption) (fam []string, edges map[string][]string, chain map[string]d.ProcessInstance, err error)
	AncestryResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error)
	DescendantsResult(ctx context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error)
	FamilyResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error)

	GetProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) ([]d.ProcessInstance, error)
	WaitForProcessInstancesState(ctx context.Context, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error)
}

type TenantSafeLookupSearcher interface {
	SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error)
}

func TenantSafeLookupUnsupported(operation string) error {
	return fmt.Errorf("%w: %s", d.ErrUnsupported, operation)
}

func LookupProcessInstance(ctx context.Context, api TenantSafeLookupSearcher, key string, opts ...services.CallOption) (d.ProcessInstance, error) {
	items, err := api.SearchForProcessInstances(ctx, d.ProcessInstanceFilter{Key: key}, 2, opts...)
	if err != nil {
		return d.ProcessInstance{}, err
	}
	return common.RequireSingleProcessInstance(items, key)
}

func LookupProcessInstanceStateByKey(ctx context.Context, api TenantSafeLookupSearcher, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error) {
	pi, err := LookupProcessInstance(ctx, api, key, opts...)
	if err != nil {
		return "", d.ProcessInstance{}, err
	}
	return pi.State, pi, nil
}

// Both supported versioned services must continue to satisfy the shared
// processinstance service surface while the internals are refactored.
var _ API = (*v87.Service)(nil)
var _ API = (*v88.Service)(nil)
var _ API = (*v89.Service)(nil)
var _ API = (v87.API)(nil)
var _ API = (v88.API)(nil)
var _ API = (v89.API)(nil)
