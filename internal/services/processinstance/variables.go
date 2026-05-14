// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"log/slog"

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
	result.Status = d.ProcessInstanceVariableUpdateStatusConfirmed
	result.ConfirmationStatus = "confirmed"
	return result, nil
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
