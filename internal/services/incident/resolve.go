// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package incident

import (
	"context"
	"errors"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/incident/waiter"
	"github.com/grafvonb/c8volt/toolx"
)

func ResolveIncidentOperation(ctx context.Context, api API, key string, opts ...services.CallOption) (d.IncidentResolutionResult, error) {
	cfg := services.ApplyCallOptions(opts)
	incident, err := api.GetIncident(ctx, key, opts...)
	result := d.IncidentResolutionResult{
		IncidentKey:       key,
		DryRun:            cfg.DryRun,
		MutationSubmitted: false,
	}
	if err != nil {
		result.Status = d.IncidentResolutionStatusMutationFailed
		result.Error = err.Error()
		return result, err
	}
	result.ProcessInstanceKey = incident.ProcessInstanceKey
	result.IncidentState = incident.State
	result.Incident = &incident
	if !waiter.IncidentIsActive(incident) {
		result.Status = d.IncidentResolutionStatusSkipped
		result.ConfirmationStatus = "invalid_state"
		return result, nil
	}
	if cfg.DryRun {
		result.Status = d.IncidentResolutionStatusPlanned
		result.WouldResolve = true
		return result, nil
	}

	resp, err := api.ResolveIncident(ctx, key, opts...)
	result.MutationAccepted = resp.Ok
	result.StatusCode = resp.StatusCode
	result.Message = resp.Status
	result.MutationSubmitted = resp.Ok
	if err != nil {
		result.Status = d.IncidentResolutionStatusMutationFailed
		result.Error = err.Error()
		return result, err
	}
	if !resp.Ok {
		result.Status = d.IncidentResolutionStatusMutationFailed
		result.Error = mutationNotAcceptedMessage(resp.Status)
		return result, errors.New(result.Error)
	}
	if cfg.NoWait {
		result.Status = d.IncidentResolutionStatusSubmitted
		result.ConfirmationStatus = "skipped"
		return result, nil
	}
	waitResp, waitErr := api.WaitForIncidentResolved(ctx, key, opts...)
	if waitErr != nil {
		result.Status = d.IncidentResolutionStatusConfirmationFailed
		result.ConfirmationStatus = "failed"
		result.Error = waitErr.Error()
		if waitResp.Status != "" {
			result.Message = waitResp.Status
		}
		return result, waitErr
	}
	result.Status = d.IncidentResolutionStatusConfirmed
	result.ConfirmationStatus = "resolved"
	if waitResp.Status != "" {
		result.Message = waitResp.Status
	}
	return result, nil
}

func ResolveProcessInstanceIncidentsOperation(ctx context.Context, api API, key string, opts ...services.CallOption) (d.ProcessInstanceResolutionResult, error) {
	cfg := services.ApplyCallOptions(opts)
	incidents, err := api.SearchProcessInstanceIncidents(ctx, key, opts...)
	result := d.ProcessInstanceResolutionResult{
		ProcessInstanceKey: key,
		DryRun:             cfg.DryRun,
	}
	if err != nil {
		result.Status = d.ProcessInstanceResolutionStatusFailed
		result.Error = err.Error()
		return result, err
	}
	owned := domainIncidentsForProcessInstance(key, incidents)
	result.Incidents = owned
	result.AttemptedIncidentKeys = activeIncidentKeys(owned)
	if len(result.AttemptedIncidentKeys) == 0 {
		result.Status = d.ProcessInstanceResolutionStatusSkipped
		result.ConfirmationStatus = "no_active_incidents"
		return result, nil
	}
	if cfg.DryRun {
		result.Status = d.ProcessInstanceResolutionStatusPlanned
		return result, nil
	}

	for _, incidentKey := range result.AttemptedIncidentKeys {
		resp, err := api.ResolveIncident(ctx, incidentKey, opts...)
		if resp.Ok {
			result.MutationSubmitted = true
			result.ResolvedIncidentKeys = append(result.ResolvedIncidentKeys, incidentKey)
			continue
		}
		result.FailedIncidentKeys = append(result.FailedIncidentKeys, incidentKey)
		if err != nil {
			result.Error = err.Error()
			continue
		}
		if result.Error == "" {
			result.Error = mutationNotAcceptedMessage(resp.Status)
		}
	}
	if len(result.ResolvedIncidentKeys) == 0 {
		result.Status = d.ProcessInstanceResolutionStatusFailed
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
	waitResp, waitErr := api.WaitForProcessInstanceIncidentsResolved(ctx, key, result.ResolvedIncidentKeys, opts...)
	if waitErr != nil {
		result.Status = d.ProcessInstanceResolutionStatusPartialFailed
		result.ConfirmationStatus = "failed"
		result.Error = waitErr.Error()
		result.FailedIncidentKeys = appendUniqueStrings(result.FailedIncidentKeys, result.ResolvedIncidentKeys...)
		result.ResolvedIncidentKeys = nil
		return result, waitErr
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

func processInstanceResolutionStatusForFailures(result d.ProcessInstanceResolutionResult) d.ProcessInstanceResolutionStatus {
	if len(result.FailedIncidentKeys) > 0 {
		if len(result.ResolvedIncidentKeys) > 0 {
			return d.ProcessInstanceResolutionStatusPartialFailed
		}
		return d.ProcessInstanceResolutionStatusFailed
	}
	if result.ConfirmationStatus == "skipped" {
		return d.ProcessInstanceResolutionStatusSubmitted
	}
	return d.ProcessInstanceResolutionStatusConfirmed
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
