// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"errors"
	"log/slog"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

// ResumeProcessInstances validates the entire explicit scope, then executes each
// distinct key independently. Partial evidence and skipped slots retain input order.
// Diagnostics belong to the caller; raw joined causes are returned without logging.
func ResumeProcessInstances(ctx context.Context, api API, log *slog.Logger, keys typex.Keys, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceResumptionResults, error) {
	result := d.ProcessInstanceResumptionResults{Items: make([]d.ProcessInstanceResumptionResult, 0)}
	if err := api.ValidateProcessInstanceResumption(); err != nil {
		return result, err
	}
	keys, err := common.NormalizeProcessInstanceKeys(keys)
	if err != nil {
		return result, err
	}
	for _, key := range keys {
		result.Items = append(result.Items, d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusNotAttempted, Submission: d.ProcessInstanceResumptionSubmissionNotSubmitted, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested})
	}
	if err := ctx.Err(); err != nil {
		for i := range result.Items {
			result.Items[i].Error = err.Error()
		}
		return result, err
	}
	if len(keys) == 0 {
		return result, nil
	}
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(keys), wantedWorkers, cfg.NoWorkerLimit)
	// Preview each adapter first, ensuring invalid timing fails before any mutation
	// or pool dispatch. Preview is local, never a state eligibility lookup.
	previewOpts := append([]services.CallOption(nil), opts...)
	previewOpts = append(previewOpts, services.WithDryRun())
	for i, key := range keys {
		preview, err := api.ResumeProcessInstance(ctx, key, previewOpts...)
		if err != nil {
			result.Items[i] = preview
			result.Items[i].Key = key
			return result, err
		}
		if cfg.DryRun {
			result.Items[i] = preview
		}
	}
	if cfg.DryRun {
		return result, nil
	}
	actual, err := pool.ExecuteSlice(ctx, keys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceResumptionResult, error) {
		return api.ResumeProcessInstance(ctx, key, opts...)
	})
	for i, item := range actual {
		if item.Key != "" {
			result.Items[i] = item
		}
	}
	err = errors.Join(err, ctx.Err())
	if err != nil {
		for i := range result.Items {
			if result.Items[i].Status == d.ProcessInstanceResumptionStatusNotAttempted {
				result.Items[i].Error = "resumption work not started after interruption or failure"
			}
		}
	}
	return result, err
}
