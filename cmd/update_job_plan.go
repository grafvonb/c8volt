// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/job"
)

// planUpdateJob loads current backend state and builds the command-facing mutation plan.
func planUpdateJob(ctx context.Context, cli c8volt.API, request job.UpdateRequest) (job.UpdatePlan, error) {
	current, err := cli.GetJob(ctx, request.Key, collectOptions()...)
	if err != nil {
		return job.UpdatePlan{}, err
	}
	return buildUpdateJobPlan(current, request), nil
}

// buildUpdateJobPlan compares retry state and records timeout submission intent without mutating.
func buildUpdateJobPlan(current job.Job, request job.UpdateRequest) job.UpdatePlan {
	plan := job.UpdatePlan{
		Key:               request.Key,
		Current:           current,
		Mode:              job.MutationModeUpdate,
		RetryStatus:       job.RetryChangeNotRequested,
		DryRun:            request.DryRun,
		MutationSubmitted: false,
	}
	if request.WorkerOutcome != nil {
		return buildWorkerOutcomeUpdatePlan(current, request)
	}
	if request.Retries != nil {
		retries := *request.Retries
		plan.RequestedRetries = &retries
		status := job.RetryChangeChanged
		before := strconv.FormatInt(int64(current.Retries), 10)
		if current.Retries == retries {
			status = job.RetryChangeUnchanged
		}
		plan.RetryStatus = status
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "retries",
			Before: before,
			After:  strconv.FormatInt(int64(retries), 10),
			Status: string(status),
		})
		if status == job.RetryChangeChanged {
			plan.MaterialChange = true
		}
	}
	if request.TimeoutMillis != nil {
		timeoutMillis := *request.TimeoutMillis
		plan.RequestedTimeout = request.TimeoutRaw
		plan.TimeoutMillis = &timeoutMillis
		plan.MaterialChange = true
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "timeout",
			After:  request.TimeoutRaw,
			Status: "submit",
		})
	}
	return plan
}

// buildWorkerOutcomeUpdatePlan renders worker outcomes through the same dry-run and confirmation plan shape used by retry and timeout updates.
func buildWorkerOutcomeUpdatePlan(current job.Job, request job.UpdateRequest) job.UpdatePlan {
	outcome := request.WorkerOutcome
	plan := job.UpdatePlan{
		Key:               request.Key,
		Current:           current,
		Mode:              job.MutationMode(outcome.Mode),
		RequestedRetries:  outcome.Retries,
		RetryStatus:       job.RetryChangeNotRequested,
		Message:           outcome.Message,
		RetryBackoff:      outcome.RetryBackoffRaw,
		RetryBackoffMS:    outcome.RetryBackoffMillis,
		ErrorCode:         outcome.ErrorCode,
		Variables:         outcome.Variables,
		MaterialChange:    true,
		DryRun:            request.DryRun,
		MutationSubmitted: false,
	}
	plan.Items = append(plan.Items, job.UpdatePlanItem{
		Name:   string(outcome.Mode),
		After:  "submit",
		Status: "submit",
	})
	if outcome.Retries != nil {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "retries",
			After:  strconv.FormatInt(int64(*outcome.Retries), 10),
			Status: "submit",
		})
	}
	if outcome.RetryBackoffMillis != nil {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "retryBackoff",
			After:  outcome.RetryBackoffRaw,
			Status: "submit",
		})
	}
	if outcome.ErrorCode != "" {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "errorCode",
			After:  outcome.ErrorCode,
			Status: "submit",
		})
	}
	if outcome.Message != "" {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "message",
			After:  outcome.Message,
			Status: "submit",
		})
	}
	if outcome.Variables != nil {
		plan.Items = append(plan.Items, job.UpdatePlanItem{
			Name:   "variables",
			After:  "submit",
			Status: "submit",
		})
	}
	return plan
}

// validateUpdateJobPlanPreconditions rejects planned updates that Camunda cannot accept for the current job state.
func validateUpdateJobPlanPreconditions(plan job.UpdatePlan, request job.UpdateRequest) error {
	if request.WorkerOutcome != nil {
		return nil
	}
	if request.TimeoutMillis == nil {
		return nil
	}
	if plan.Current.Key == "" {
		return nil
	}
	if strings.EqualFold(plan.Current.State, "CREATED") {
		return nil
	}
	state := plan.Current.State
	if state == "" {
		state = "unknown"
	}
	return localPreconditionError(fmt.Errorf("job timeout can be updated only for active jobs; job %s is %s", request.Key, state))
}
