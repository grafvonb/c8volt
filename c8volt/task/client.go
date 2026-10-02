// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"fmt"
	"log/slog"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	utsvc "github.com/grafvonb/c8volt/internal/services/usertask"
	"github.com/grafvonb/c8volt/toolx"
	types "github.com/grafvonb/c8volt/typex"
)

type client struct {
	pdApi     pdsvc.API
	piApi     pisvc.API
	utApi     utsvc.API
	updateApi utsvc.VariableUpdateAPI
	log       *slog.Logger
}

// New creates a task facade for reads and process-instance resolution; variable updates require
// NewWithVariableUpdates.
func New(pdApi pdsvc.API, piApi pisvc.API, utApi utsvc.API, log *slog.Logger) API {
	return NewWithVariableUpdates(pdApi, piApi, utApi, nil, log)
}

// NewWithVariableUpdates creates a task facade with the composed variable
// update workflow while preserving New for read-only consumers.
func NewWithVariableUpdates(pdApi pdsvc.API, piApi pisvc.API, utApi utsvc.API, updateApi utsvc.VariableUpdateAPI, log *slog.Logger) API {
	return &client{
		pdApi:     pdApi,
		piApi:     piApi,
		utApi:     utApi,
		updateApi: updateApi,
		log:       log,
	}
}

// GetUserTask reads one native user task and maps it to the stable public model.
func (c *client) GetUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (UserTask, error) {
	got, err := c.utApi.GetNativeUserTask(ctx, taskKey, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return UserTask{}, ferr.FromDomain(err)
	}
	return fromDomainUserTask(got), nil
}

// GetUserTasks reads stable-unique native task keys and returns them in first-input order.
func (c *client) GetUserTasks(ctx context.Context, taskKeys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (UserTasks, error) {
	got, err := utsvc.GetUserTasks(ctx, c.utApi, taskKeys, wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return UserTasks{}, ferr.FromDomain(err)
	}
	return fromDomainUserTasks(got), nil
}

// SearchUserTasks delegates collection and traversal to the internal service
// and maps the selected tasks into the stable public collection contract.
func (c *client) SearchUserTasks(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (UserTasks, error) {
	got, err := utsvc.SearchUserTasks(ctx, c.utApi, toDomainSearchRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return UserTasks{}, ferr.FromDomain(err)
	}
	return fromDomainUserTasks(got), nil
}

// SearchUserTasksPages exposes public visitor facts while the internal service
// retains ownership of page advancement, limits, and completion decisions.
func (c *client) SearchUserTasksPages(ctx context.Context, request SearchRequest, visitor SearchPageVisitor, opts ...options.FacadeOption) (SearchPagesResult, error) {
	got, err := utsvc.SearchUserTasksPages(ctx, c.utApi, toDomainSearchRequest(request), toDomainSearchPageVisitor(visitor), options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainSearchPagesResult(got)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}

// SearchUserTasksTotal returns the internal service's exact matching count
// without exposing its capped-total fallback traversal.
func (c *client) SearchUserTasksTotal(ctx context.Context, request SearchRequest, opts ...options.FacadeOption) (int64, error) {
	total, err := utsvc.SearchUserTasksTotal(ctx, c.utApi, toDomainSearchRequest(request), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return 0, ferr.FromDomain(err)
	}
	return total, nil
}

// EnrichUserTasksWithVariables attaches effective variables to selected task
// results without changing their order or re-fetching task metadata.
func (c *client) EnrichUserTasksWithVariables(ctx context.Context, tasks UserTasks, opts ...options.FacadeOption) (VariableEnrichedUserTasks, error) {
	got, err := utsvc.EnrichUserTasksWithVariables(ctx, c.utApi, toolx.MapSlice(tasks.Items, toDomainUserTask), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return VariableEnrichedUserTasks{}, ferr.FromDomain(err)
	}
	return fromDomainVariableEnrichedUserTasks(got), nil
}

// ResolveProcessInstanceKeyFromUserTask keeps single task-key lookup aligned with the multi-key path used by the CLI.
func (c *client) ResolveProcessInstanceKeyFromUserTask(ctx context.Context, taskKey string, opts ...options.FacadeOption) (string, error) {
	keys, err := c.ResolveProcessInstanceKeysFromUserTasks(ctx, types.Keys{taskKey}, opts...)
	if err != nil {
		return "", err
	}
	return keys[0], nil
}

// ResolveProcessInstanceKeysFromUserTasks resolves user tasks through the native task API and returns their owning process-instance keys in input order.
func (c *client) ResolveProcessInstanceKeysFromUserTasks(ctx context.Context, taskKeys types.Keys, opts ...options.FacadeOption) (types.Keys, error) {
	keys, err := utsvc.ResolveProcessInstanceKeys(ctx, c.utApi, taskKeys, options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return nil, ferr.FromDomain(err)
	}
	return keys, nil
}

// PlanUserTaskVariableUpdates delegates complete discovery and planning to the
// composed service and copies all mutable values across the public boundary.
func (c *client) PlanUserTaskVariableUpdates(ctx context.Context, keys types.Keys, variables map[string]any, opts ...options.FacadeOption) (UserTaskVariableUpdatePlan, error) {
	if c.updateApi == nil {
		return UserTaskVariableUpdatePlan{}, ferr.FromDomain(fmt.Errorf("%w: user-task variable planning requires a variable update service", d.ErrPrecondition))
	}
	got, err := c.updateApi.PlanUserTaskVariableUpdates(ctx, append(types.Keys(nil), keys...), copyUserTaskVariableMap(variables), options.MapFacadeOptionsToCallOptions(opts)...)
	if err != nil {
		return UserTaskVariableUpdatePlan{}, ferr.FromDomain(err)
	}
	return fromDomainUserTaskVariableUpdatePlan(got), nil
}

// ExecuteUserTaskVariableUpdates submits the caller's frozen plan through the
// composed service and preserves partial results alongside normalized errors.
func (c *client) ExecuteUserTaskVariableUpdates(ctx context.Context, plan UserTaskVariableUpdatePlan, wantedWorkers int, opts ...options.FacadeOption) (UserTaskVariableUpdateResults, error) {
	if c.updateApi == nil {
		return UserTaskVariableUpdateResults{}, ferr.FromDomain(fmt.Errorf("%w: user-task variable execution requires a variable update service", d.ErrPrecondition))
	}
	got, err := c.updateApi.ExecuteUserTaskVariableUpdates(ctx, toDomainUserTaskVariableUpdatePlan(plan), wantedWorkers, options.MapFacadeOptionsToCallOptions(opts)...)
	out := fromDomainUserTaskVariableUpdateResults(got)
	if err != nil {
		return out, ferr.FromDomain(err)
	}
	return out, nil
}
