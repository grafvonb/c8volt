// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// repairBusinessIDProcessInstances freezes native discovery and owned incidents before any repair action.
func (s *Service) repairBusinessIDProcessInstances(ctx context.Context, request d.OpsRepairRequest, opts ...services.CallOption) (d.OpsRepairResult, error) {
	result := newRepairResult(request)
	fail := func(err error) (d.OpsRepairResult, error) {
		result.FrozenSet.Status = d.OpsWorkflowStepStatusFailed
		result.FrozenSet.Errors = []string{err.Error()}
		return finishRepairResult(result, s.version, d.OpsRepairOutcomeFailed, err)
	}
	// Both copies of the criterion are checked before keyed, empty or prepared-plan shortcuts.
	if err := common.ValidateBusinessIDSearch(request.ProcessInstanceSelection.BusinessId, s.version); err != nil {
		return fail(err)
	}
	if request.FrozenPlan != nil {
		if err := common.ValidateBusinessIDSearch(request.FrozenPlan.Selection.Request.SearchRequest.Filter.BusinessId, s.version); err != nil {
			return fail(err)
		}
	}
	if request.ProcessInstanceSelection.BusinessId == nil || request.DiscoveryMode != d.OpsRepairDiscoveryModeSearch || len(request.InputKeys) > 0 {
		return fail(fmt.Errorf("%w: business-ID repair requires search criteria and a reviewed plan instead of bare keys", d.ErrValidation))
	}
	if request.RequestedRetries != nil && *request.RequestedRetries < 0 || request.RequestedJobTimeout < 0 {
		return fail(fmt.Errorf("%w: repair retries and timeout must be nonnegative", d.ErrValidation))
	}
	variables, err := toolx.CopyJSONMap(request.Variables)
	if err != nil {
		return fail(fmt.Errorf("%w: invalid repair variables", d.ErrValidation))
	}
	request.Variables = variables
	// Incident-bearing selection is authoritative for the documented seed limit, including library calls.
	filter := pisvc.CopyProcessInstanceSelection(d.ProcessInstanceSelection{Request: d.ProcessInstanceSelectionRequest{SearchRequest: d.ProcessInstanceSearchRequest{Filter: request.ProcessInstanceSelection}}}).Request.SearchRequest.Filter
	if filter.HasIncident != nil && !*filter.HasIncident {
		return fail(fmt.Errorf("%w: repair requires incident-bearing process instances", d.ErrValidation))
	}
	filter.HasIncident = new(true)
	request.ProcessInstanceSelection = filter
	request.RequestedRetries = toolx.CopyPtr(request.RequestedRetries)
	result.Request = request
	result.Request.FrozenPlan = nil
	var plan d.OpsRepairFrozenPlan
	if request.FrozenPlan == nil {
		selection, selectionErr := pisvc.ResolveProcessInstanceSelection(ctx, s.piAPI, s.incAPI, repairBusinessIDSelection(request), opts...)
		plan = d.OpsRepairFrozenPlan{Selection: selection, Tenant: s.piAPI.ProcessInstanceSearchTenant(opts...), Variables: variables, RequestedRetries: requestedRepairRetries(request), RequestedJobTimeout: request.RequestedJobTimeout, DirectIncidentsOnly: request.DirectIncidentsOnly}
		result.FrozenSet = repairBusinessIDFrozenSet(request, plan)
		if selectionErr != nil {
			return fail(selectionErr)
		}
		plan.Incidents, err = incsvc.PlanProcessInstancesIncidents(ctx, s.incAPI, selection.Keys, opts...)
		result.FrozenSet = repairBusinessIDFrozenSet(request, plan)
		if err != nil {
			return fail(err)
		}
	} else {
		plan, err = CopyRepairFrozenPlan(*request.FrozenPlan)
		if err != nil {
			return fail(err)
		}
	}
	if err = s.validateRepairFrozenPlan(request, plan, opts...); err != nil {
		return fail(err)
	}
	if err = ctx.Err(); err != nil {
		return fail(err)
	}
	result.FrozenSet = repairBusinessIDFrozenSet(request, plan)
	// Execution only uses values matched against the reviewed actions; no new discovery is performed.
	request.Variables = plan.Variables
	request.RequestedRetries = plan.RequestedRetries
	request.FrozenPlan = nil
	result.Request = request
	frozenCopy, copyErr := CopyRepairFrozenPlan(plan)
	if copyErr != nil {
		return fail(copyErr)
	}
	result.FrozenPlan = &frozenCopy
	opts = append(slices.Clone(opts), services.WithSuppressWorkflowDetailLogs(), services.WithSuppressProcessInstanceDetailLogs())
	if request.NoWait {
		opts = append(opts, services.WithNoWait())
	}
	if request.FailFast {
		opts = append(opts, services.WithFailFast())
	}
	if request.NoWorkerLimit {
		opts = append(opts, services.WithNoWorkerLimit())
	}
	return s.finishProcessInstanceIncidentRepair(ctx, request, result, result.FrozenSet.OriginalIncidents, opts...)
}

// repairBusinessIDSelection keeps local incident eligibility inside the shared selector before distinct limits.
func repairBusinessIDSelection(request d.OpsRepairRequest) d.ProcessInstanceSelectionRequest {
	return d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{
		Filter: request.ProcessInstanceSelection, Page: d.ProcessInstancePageRequest{Size: repairProcessInstanceDiscoverySize(request)}, Limit: request.Limit,
		LocalFilters: d.ProcessInstanceSearchLocalFilters{DirectIncidentsOnly: request.DirectIncidentsOnly, IncidentState: request.IncidentSelection.State, IncidentErrorType: request.IncidentSelection.ErrorType, IncidentErrorMessage: request.IncidentSelection.ErrorMessage},
	}}
}

// validateRepairFrozenPlan rejects changed criteria, incomplete reads and changed action payloads before shortcuts.
func (s *Service) validateRepairFrozenPlan(request d.OpsRepairRequest, plan d.OpsRepairFrozenPlan, opts ...services.CallOption) error {
	if err := pisvc.ValidateProcessInstanceSelection(s.piAPI, plan.Selection, plan.Tenant, opts...); err != nil {
		return err
	}
	if err := incsvc.ValidateProcessInstancesIncidentPlan(plan.Incidents); err != nil {
		return err
	}
	if !reflect.DeepEqual(repairBusinessIDSelection(request), plan.Selection.Request) || !slices.Equal(plan.Selection.Keys, plan.Incidents.Keys) ||
		!reflect.DeepEqual(request.Variables, plan.Variables) || !reflect.DeepEqual(requestedRepairRetries(request), plan.RequestedRetries) ||
		request.RequestedJobTimeout != plan.RequestedJobTimeout || request.DirectIncidentsOnly != plan.DirectIncidentsOnly {
		return fmt.Errorf("%w: repair plan does not match the reviewed selection and actions", d.ErrPrecondition)
	}
	return nil
}

// repairBusinessIDFrozenSet retains selected observations separately from incident-eligible repair targets.
func repairBusinessIDFrozenSet(request d.OpsRepairRequest, plan d.OpsRepairFrozenPlan) d.OpsRepairFrozenSet {
	var active []d.ProcessInstanceIncidentDetail
	for _, item := range plan.Incidents.Incidents {
		if incidentIsActive(item) {
			active = append(active, item)
		}
	}
	frozen := freezeProcessInstanceRepairSet(request, plan.Selection.Keys, plan.Selection.Items, active)
	selection := pisvc.CopyProcessInstanceSelection(plan.Selection)
	frozen.Selection = &selection
	frozen.ProcessFilters = selection.Request.SearchRequest.Filter
	frozen.DiscoveryScopeStatus = d.DiscoveryScopeStatus{BatchSize: selection.Request.SearchRequest.Page.Size, Limit: selection.Request.SearchRequest.Limit, Pages: int(selection.Pages), CandidatesSeen: len(selection.Keys), CandidatesFrozen: len(selection.Keys), Complete: selection.Completion == d.ProcessInstanceSearchCompletionExhausted, Limited: selection.Completion == d.ProcessInstanceSearchCompletionLimitReached}
	return frozen
}

// CopyRepairFrozenPlan takes a defensive snapshot of nested evidence and exact mutation payloads.
func CopyRepairFrozenPlan(plan d.OpsRepairFrozenPlan) (d.OpsRepairFrozenPlan, error) {
	plan.Selection = pisvc.CopyProcessInstanceSelection(plan.Selection)
	plan.Incidents.Keys = slices.Clone(plan.Incidents.Keys)
	plan.Incidents.CompletedKeys = slices.Clone(plan.Incidents.CompletedKeys)
	plan.Incidents.Incidents = slices.Clone(plan.Incidents.Incidents)
	plan.RequestedRetries = toolx.CopyPtr(plan.RequestedRetries)
	var err error
	plan.Variables, err = toolx.CopyJSONMap(plan.Variables)
	if err != nil {
		return plan, fmt.Errorf("%w: invalid frozen repair variables", d.ErrValidation)
	}
	return plan, nil
}

// executeFrozenRepairVariables submits only reviewed exact scope/payload pairs and retains error classes under no-wait.
func (s *Service) executeFrozenRepairVariables(ctx context.Context, request d.OpsRepairRequest, keys typex.Keys, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResults, error) {
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(keys), request.Workers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice(ctx, keys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceVariableUpdateResult, error) {
		response, err := s.piAPI.UpdateProcessInstanceVariables(ctx, key, request.Variables, opts...)
		item := d.ProcessInstanceVariableUpdateResult{Key: key, Variables: request.Variables, MutationAccepted: response.Ok, StatusCode: response.StatusCode, Message: response.Status, Status: d.ProcessInstanceVariableUpdateStatusSubmitted, ConfirmationStatus: "skipped"}
		if err == nil && !response.Ok {
			err = fmt.Errorf("%w: variable update was not accepted", d.ErrUpstream)
		}
		if err == nil && !request.NoWait {
			err = pisvc.ConfirmProcessInstanceVariables(ctx, s.piAPI, key, request.Variables, opts...)
			if err == nil {
				item.Status = d.ProcessInstanceVariableUpdateStatusConfirmed
				item.ConfirmationStatus = "confirmed"
			}
		}
		if err != nil {
			item.Error = err.Error()
			item.Status = d.ProcessInstanceVariableUpdateStatusMutationFailed
			if response.Ok {
				item.Status = d.ProcessInstanceVariableUpdateStatusConfirmationFailed
				item.ConfirmationStatus = "failed"
			}
		}
		return item, err
	})
	return d.ProcessInstanceVariableUpdateResults{Items: items}, errors.Join(err, ctx.Err())
}
