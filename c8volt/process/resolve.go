// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"context"
	"errors"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/incident/waiter"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	types "github.com/grafvonb/c8volt/typex"
)

func (c *client) ResolveIncident(ctx context.Context, key string, opts ...options.FacadeOption) (IncidentResolutionResult, error) {
	cfg := options.ApplyFacadeOptions(opts)
	callOpts := options.MapFacadeOptionsToCallOptions(opts)
	incident, err := c.incApi.GetIncident(ctx, key, callOpts...)
	result := IncidentResolutionResult{
		IncidentKey:       key,
		DryRun:            cfg.DryRun,
		MutationSubmitted: false,
	}
	if err != nil {
		ferrErr := ferr.FromDomain(err)
		result.Status = IncidentResolutionStatusMutationFailed
		result.Error = ferrErr.Error()
		return result, ferrErr
	}
	result.ProcessInstanceKey = incident.ProcessInstanceKey
	result.IncidentState = incident.State
	detail := fromDomainProcessInstanceIncidentDetail(incident)
	result.Incident = &detail
	if !waiter.IncidentIsActive(incident) {
		result.Status = IncidentResolutionStatusSkipped
		result.ConfirmationStatus = "invalid_state"
		return result, nil
	}
	if cfg.DryRun {
		result.Status = IncidentResolutionStatusPlanned
		result.WouldResolve = true
		return result, nil
	}

	resp, err := c.incApi.ResolveIncident(ctx, key, callOpts...)
	result.MutationAccepted = resp.Ok
	result.StatusCode = resp.StatusCode
	result.Message = resp.Status
	result.MutationSubmitted = resp.Ok
	if err != nil {
		ferrErr := ferr.FromDomain(err)
		result.Status = IncidentResolutionStatusMutationFailed
		result.Error = ferrErr.Error()
		return result, ferrErr
	}
	if !resp.Ok {
		result.Status = IncidentResolutionStatusMutationFailed
		result.Error = mutationNotAcceptedMessage(resp.Status)
		return result, errors.New(result.Error)
	}
	if cfg.NoWait {
		result.Status = IncidentResolutionStatusSubmitted
		result.ConfirmationStatus = "skipped"
		return result, nil
	}
	waitResp, waitErr := c.incApi.WaitForIncidentResolved(ctx, key, callOpts...)
	if waitErr != nil {
		ferrErr := ferr.FromDomain(waitErr)
		result.Status = IncidentResolutionStatusConfirmationFailed
		result.ConfirmationStatus = "failed"
		result.Error = ferrErr.Error()
		if waitResp.Status != "" {
			result.Message = waitResp.Status
		}
		return result, ferrErr
	}
	result.Status = IncidentResolutionStatusConfirmed
	result.ConfirmationStatus = "resolved"
	if waitResp.Status != "" {
		result.Message = waitResp.Status
	}
	return result, nil
}

func (c *client) ResolveProcessInstanceIncidents(ctx context.Context, key string, opts ...options.FacadeOption) (ProcessInstanceResolutionResult, error) {
	cfg := options.ApplyFacadeOptions(opts)
	callOpts := options.MapFacadeOptionsToCallOptions(opts)
	incidents, err := c.incApi.SearchProcessInstanceIncidents(ctx, key, callOpts...)
	result := ProcessInstanceResolutionResult{
		ProcessInstanceKey: key,
		DryRun:             cfg.DryRun,
	}
	if err != nil {
		ferrErr := ferr.FromDomain(err)
		result.Status = ProcessInstanceResolutionStatusFailed
		result.Error = ferrErr.Error()
		return result, ferrErr
	}
	owned := domainIncidentsForProcessInstance(key, incidents)
	result.Incidents = fromDomainProcessInstanceIncidentDetails(owned)
	result.AttemptedIncidentKeys = activeIncidentKeys(owned)
	if len(result.AttemptedIncidentKeys) == 0 {
		result.Status = ProcessInstanceResolutionStatusSkipped
		result.ConfirmationStatus = "no_active_incidents"
		return result, nil
	}
	if cfg.DryRun {
		result.Status = ProcessInstanceResolutionStatusPlanned
		return result, nil
	}

	resp, err := c.incApi.ResolveProcessInstanceIncidents(ctx, key, callOpts...)
	result.MutationSubmitted = resp.Ok
	if err != nil {
		result.FailedIncidentKeys = append(result.FailedIncidentKeys, result.AttemptedIncidentKeys...)
		result.Status = ProcessInstanceResolutionStatusFailed
		result.Error = ferr.FromDomain(err).Error()
		return result, errorFromResult(result.Error)
	}
	if resp.Ok {
		result.ResolvedIncidentKeys = append(result.ResolvedIncidentKeys, result.AttemptedIncidentKeys...)
	} else {
		result.FailedIncidentKeys = append(result.FailedIncidentKeys, result.AttemptedIncidentKeys...)
		result.Error = mutationNotAcceptedMessage(resp.Status)
	}
	if len(result.ResolvedIncidentKeys) == 0 {
		result.Status = ProcessInstanceResolutionStatusFailed
		if result.Error == "" {
			result.Error = "incident resolution mutation was not accepted"
		}
		return result, errorFromResult(result.Error)
	}
	if cfg.NoWait {
		result.ConfirmationStatus = "skipped"
		result.Status = processInstanceResolutionStatusForFailures(result)
		return result, errorFromResult(result.Error)
	}
	waitResp, waitErr := c.incApi.WaitForProcessInstanceIncidentsResolved(ctx, key, result.ResolvedIncidentKeys, callOpts...)
	if waitErr != nil {
		ferrErr := ferr.FromDomain(waitErr)
		result.Status = ProcessInstanceResolutionStatusPartialFailed
		result.ConfirmationStatus = "failed"
		result.Error = ferrErr.Error()
		result.FailedIncidentKeys = appendUniqueStrings(result.FailedIncidentKeys, result.ResolvedIncidentKeys...)
		result.ResolvedIncidentKeys = nil
		return result, ferrErr
	}
	result.Status = processInstanceResolutionStatusForFailures(result)
	result.ConfirmationStatus = "resolved"
	if waitResp.Status != "" {
		result.ConfirmationStatus = "resolved"
	}
	return result, errorFromResult(result.Error)
}

func mutationNotAcceptedMessage(status string) string {
	if status != "" {
		return status
	}
	return "incident resolution mutation was not accepted"
}

func summarizeIncidentResolutionResults(items []IncidentResolutionResult) IncidentResolutionResults {
	out := IncidentResolutionResults{Operation: ResolutionOperationIncident, Items: items, Total: len(items)}
	for _, item := range items {
		out.DryRun = out.DryRun || item.DryRun
		out.MutationSubmitted = out.MutationSubmitted || item.MutationSubmitted
		switch item.Status {
		case IncidentResolutionStatusSubmitted:
			out.Submitted++
		case IncidentResolutionStatusConfirmed:
			out.Confirmed++
		case IncidentResolutionStatusSkipped, IncidentResolutionStatusPlanned:
			out.Skipped++
		case IncidentResolutionStatusMutationFailed, IncidentResolutionStatusConfirmationFailed:
			out.Failed++
		}
	}
	return out
}

func summarizeProcessInstanceResolutionResults(items []ProcessInstanceResolutionResult) ProcessInstanceResolutionResults {
	out := ProcessInstanceResolutionResults{Operation: ResolutionOperationProcessInstance, Items: items, Total: len(items)}
	for _, item := range items {
		out.DryRun = out.DryRun || item.DryRun
		out.MutationSubmitted = out.MutationSubmitted || item.MutationSubmitted
		switch item.Status {
		case ProcessInstanceResolutionStatusSubmitted:
			out.Submitted++
		case ProcessInstanceResolutionStatusConfirmed:
			out.Confirmed++
		case ProcessInstanceResolutionStatusSkipped, ProcessInstanceResolutionStatusPlanned:
			out.Skipped++
		case ProcessInstanceResolutionStatusFailed, ProcessInstanceResolutionStatusPartialFailed:
			out.Failed++
		}
	}
	return out
}

func domainIncidentsForProcessInstance(key string, incidents []d.ProcessInstanceIncidentDetail) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(incidents))
	for _, incident := range incidents {
		if incident.ProcessInstanceKey == key {
			out = append(out, incident)
		}
	}
	return out
}

func activeIncidentKeys(incidents []d.ProcessInstanceIncidentDetail) []string {
	keys := make([]string, 0, len(incidents))
	for _, incident := range incidents {
		if waiter.IncidentIsActive(incident) {
			keys = append(keys, incident.IncidentKey)
		}
	}
	return toolx.UniqueSlice(keys)
}

func processInstanceResolutionStatusForFailures(result ProcessInstanceResolutionResult) ProcessInstanceResolutionStatus {
	if len(result.FailedIncidentKeys) > 0 {
		if len(result.ResolvedIncidentKeys) > 0 {
			return ProcessInstanceResolutionStatusPartialFailed
		}
		return ProcessInstanceResolutionStatusFailed
	}
	if result.ConfirmationStatus == "skipped" {
		return ProcessInstanceResolutionStatusSubmitted
	}
	return ProcessInstanceResolutionStatusConfirmed
}

func appendUniqueStrings(base []string, extras ...string) []string {
	seen := make(map[string]struct{}, len(base)+len(extras))
	out := make([]string, 0, len(base)+len(extras))
	for _, item := range base {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	for _, item := range extras {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func errorFromResult(message string) error {
	if message == "" {
		return nil
	}
	return errors.New(message)
}

func (c *client) ResolveIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (IncidentResolutionResults, error) {
	cfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	nw := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice[string, IncidentResolutionResult](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (IncidentResolutionResult, error) {
		return c.ResolveIncident(ctx, key, opts...)
	})
	items = compactIncidentResolutionResults(items)
	return summarizeIncidentResolutionResults(items), err
}

func (c *client) ResolveProcessInstancesIncidents(ctx context.Context, keys types.Keys, wantedWorkers int, opts ...options.FacadeOption) (ProcessInstanceResolutionResults, error) {
	cfg := options.ApplyFacadeOptions(opts)
	ukeys := keys.Unique()
	nw := toolx.DetermineNoOfWorkers(len(ukeys), wantedWorkers, cfg.NoWorkerLimit)
	items, err := pool.ExecuteSlice[string, ProcessInstanceResolutionResult](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (ProcessInstanceResolutionResult, error) {
		return c.ResolveProcessInstanceIncidents(ctx, key, opts...)
	})
	items = compactProcessInstanceResolutionResults(items)
	return summarizeProcessInstanceResolutionResults(items), err
}

func compactIncidentResolutionResults(items []IncidentResolutionResult) []IncidentResolutionResult {
	out := items[:0]
	for _, item := range items {
		if item.IncidentKey == "" && item.Status == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func compactProcessInstanceResolutionResults(items []ProcessInstanceResolutionResult) []ProcessInstanceResolutionResult {
	out := items[:0]
	for _, item := range items {
		if item.ProcessInstanceKey == "" && item.Status == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}
