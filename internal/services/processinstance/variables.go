// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

func UpdateProcessInstancesVariables(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, variables map[string]any, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResults, error) {
	cfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cfg.NoWorkerLimit)
	logging.InfoIfVerbose(fmt.Sprintf("updating pi variables: requested %d, workers %d", lk, nw), log, cfg.Verbose)
	stopActivity := logging.StartActivity(ctx, processInstanceBulkActivity("updating variables for", lk, 0))
	defer stopActivity()
	rs, err := pool.ExecuteSlice[string, d.ProcessInstanceVariableUpdateResult](ctx, ukeys, nw, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceVariableUpdateResult, error) {
		return UpdateProcessInstanceVariables(ctx, api, d.ProcessInstanceVariableUpdateRequest{Key: key, Variables: variables}, opts...)
	})
	return d.ProcessInstanceVariableUpdateResults{Items: rs}, err
}

func UpdateProcessInstanceVariables(ctx context.Context, api API, request d.ProcessInstanceVariableUpdateRequest, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResult, error) {
	cfg := services.ApplyCallOptions(opts)
	resp, err := api.UpdateProcessInstanceVariables(ctx, request.Key, request.Variables, opts...)
	result := domainVariableUpdateResult(resp, request.Variables)
	if result.Key == "" {
		result.Key = request.Key
	}
	if err != nil {
		result.Error = err.Error()
		if result.MutationAccepted {
			result.Status = d.ProcessInstanceVariableUpdateStatusConfirmationFailed
			result.ConfirmationStatus = "failed"
			return result, err
		}
		result.Status = d.ProcessInstanceVariableUpdateStatusMutationFailed
		result.MutationAccepted = false
		if cfg.NoWait {
			result.ConfirmationStatus = "skipped"
			return result, nil
		}
		return result, err
	}
	if cfg.NoWait {
		result.Status = d.ProcessInstanceVariableUpdateStatusSubmitted
		result.ConfirmationStatus = "skipped"
		return result, nil
	}
	if err := ConfirmProcessInstanceVariables(ctx, api, request.Key, request.Variables, opts...); err != nil {
		result.Status = d.ProcessInstanceVariableUpdateStatusConfirmationFailed
		result.ConfirmationStatus = "failed"
		result.Error = err.Error()
		return result, err
	}
	result.Status = d.ProcessInstanceVariableUpdateStatusConfirmed
	result.ConfirmationStatus = "confirmed"
	return result, nil
}

// ConfirmProcessInstanceVariables verifies only the requested process-scope variables using normalized JSON values.
func ConfirmProcessInstanceVariables(ctx context.Context, api API, key string, requested map[string]any, opts ...services.CallOption) error {
	current, err := api.SearchProcessInstanceVariables(ctx, key, opts...)
	if err != nil {
		return fmt.Errorf("confirm variables for process-instance %s: %w", key, err)
	}
	currentByName := processScopeVariableValuesByName(key, current)
	for name, expected := range requested {
		actual, ok := currentByName[name]
		if !ok {
			return fmt.Errorf("%w: variable %q not confirmed on process-instance %s", d.ErrUpstream, name, key)
		}
		if !normalizedJSONEqual(actual, expected) {
			return fmt.Errorf("%w: variable %q confirmation mismatch on process-instance %s", d.ErrUpstream, name, key)
		}
	}
	return nil
}

func domainVariableUpdateResult(x d.ProcessInstanceVariableUpdateResponse, variables map[string]any) d.ProcessInstanceVariableUpdateResult {
	status := d.ProcessInstanceVariableUpdateStatusSubmitted
	if !x.Ok {
		status = d.ProcessInstanceVariableUpdateStatusMutationFailed
	}
	return d.ProcessInstanceVariableUpdateResult{
		Key:              x.Key,
		Status:           status,
		MutationAccepted: x.Ok,
		StatusCode:       x.StatusCode,
		Message:          x.Status,
		Variables:        toolx.CopyMap(variables),
	}
}

// processScopeVariableValuesByName keeps confirmation aligned with get/update process-instance scope semantics.
func processScopeVariableValuesByName(key string, variables []d.ProcessInstanceVariable) map[string]any {
	out := make(map[string]any, len(variables))
	for _, variable := range variables {
		if variable.ProcessInstanceKey != "" && variable.ProcessInstanceKey != key {
			continue
		}
		if variable.ScopeKey != "" && variable.ScopeKey != key {
			continue
		}
		out[variable.Name] = decodeProcessInstanceVariableValue(variable.Value)
	}
	return out
}

// decodeProcessInstanceVariableValue converts API JSON values before normalized confirmation comparison.
func decodeProcessInstanceVariableValue(raw string) any {
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return raw
	}
	return value
}

// normalizedJSONEqual compares values after JSON round-tripping so numeric and object formatting differences do not matter.
func normalizedJSONEqual(a any, b any) bool {
	return reflect.DeepEqual(normalizeJSONValue(a), normalizeJSONValue(b))
}

// normalizeJSONValue returns the JSON data model representation used by Camunda variable payloads.
func normalizeJSONValue(value any) any {
	data, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		return value
	}
	return out
}
