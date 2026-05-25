// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

func jobView(cmd *cobra.Command, item job.Job) error {
	return itemView(cmd, item, pickMode(), oneLineJobWithTimezoneForMode(cmd), jobKey)
}

// jobsView renders searched jobs using the same row format as keyed lookup and
// leaves JSON output as one collection payload.
func jobsView(cmd *cobra.Command, result job.SearchResult) error {
	return listOrJSONFlat(cmd, result, result.Items, pickMode(), flatRowJobWithTimezoneForMode(cmd), jobKey)
}

// oneLineJobWithTimezoneForMode binds the current timezone display setting for
// keyed job rendering.
func oneLineJobWithTimezoneForMode(cmd *cobra.Command) func(job.Job) string {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item job.Job) string {
		return oneLineJobWithTimezone(item, showTimezoneOffset)
	}
}

// flatRowJobWithTimezoneForMode binds the current timezone display setting for
// aligned list/search rows.
func flatRowJobWithTimezoneForMode(cmd *cobra.Command) func(job.Job) flatRow {
	showTimezoneOffset := commandShowTimezoneOffset(cmd)
	return func(item job.Job) flatRow {
		return flatRowJobWithTimezone(item, showTimezoneOffset)
	}
}

// jobKey returns the stable pipeline key for keyed and searched job output.
func jobKey(item job.Job) string {
	return item.Key
}

func jobUpdateResultView(cmd *cobra.Command, result job.UpdateResult) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	switch result.Status {
	case "confirmed":
		parts := []string{fmt.Sprintf("confirmed retries=%d", derefInt32(result.ConfirmedRetries))}
		if result.SubmittedTimeoutMS != nil {
			parts = append(parts, fmt.Sprintf("timeout=%dms submitted", *result.SubmittedTimeoutMS))
		}
		renderHumanLine(cmd, "updated job %s: %s", result.Key, strings.Join(parts, "; "))
	case "submitted":
		parts := []string{"submitted"}
		if result.SubmittedRetries != nil {
			parts = append(parts, fmt.Sprintf("retries=%d", *result.SubmittedRetries))
		}
		if result.SubmittedTimeoutMS != nil {
			parts = append(parts, fmt.Sprintf("timeout=%dms", *result.SubmittedTimeoutMS))
		}
		renderHumanLine(cmd, "updated job %s: %s", result.Key, strings.Join(parts, " "))
	case "confirmation_failed":
		renderHumanLine(cmd, "updated job %s: confirmation failed: %s", result.Key, result.Error)
	case "mutation_failed":
		renderHumanLine(cmd, "updated job %s: mutation failed: %s", result.Key, result.Error)
	default:
		renderHumanLine(cmd, "updated job %s: %s", result.Key, result.Status)
	}
	return nil
}

func jobWorkerOutcomeResultView(cmd *cobra.Command, result job.WorkerOutcomeResult) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	switch result.Status {
	case "submitted":
		parts := []string{fmt.Sprintf("submitted %s", formatJobMutationMode(job.MutationMode(result.Mode)))}
		if result.SubmittedRetries != nil {
			parts = append(parts, fmt.Sprintf("retries=%d", *result.SubmittedRetries))
		}
		if result.SubmittedBackoffMS != nil {
			parts = append(parts, fmt.Sprintf("retryBackoff=%dms", *result.SubmittedBackoffMS))
		}
		renderHumanLine(cmd, "updated job %s: %s", result.Key, strings.Join(parts, " "))
	case "mutation_failed":
		renderHumanLine(cmd, "updated job %s: mutation failed: %s", result.Key, result.Error)
	default:
		renderHumanLine(cmd, "updated job %s: %s", result.Key, result.Status)
	}
	return nil
}

func jobUpdatePlanView(cmd *cobra.Command, plan job.UpdatePlan, label string) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, plan)
	}
	status := jobUpdatePlanHumanStatus(plan, label)
	if !plan.HasMaterialChange() {
		renderHumanLine(cmd, "%s: update job %s: nothing to update; %s", label, plan.Key, status)
		return nil
	}
	if status != "" {
		renderHumanLine(cmd, "%s: update job %s: %s; %s", label, plan.Key, formatJobUpdatePlanItems(plan.Items), status)
		return nil
	}
	renderHumanLine(cmd, "%s: update job %s: %s", label, plan.Key, formatJobUpdatePlanItems(plan.Items))
	return nil
}

func jobUpdatePlanHumanStatus(plan job.UpdatePlan, label string) string {
	if label == "dry run" {
		return "no changes applied"
	}
	if !plan.HasMaterialChange() {
		return "no confirmation required"
	}
	return ""
}

func oneLineJob(item job.Job) string {
	return oneLineJobWithTimezone(item, false)
}

func oneLineJobWithTimezone(item job.Job, showTimezoneOffset bool) string {
	return compactFlatRow(flatRowJobWithTimezone(item, showTimezoneOffset))
}

func flatRowJob(item job.Job) flatRow {
	return flatRowJobWithTimezone(item, false)
}

func flatRowJobWithTimezone(item job.Job, showTimezoneOffset bool) flatRow {
	parts := flatRow{item.Key}
	if item.TenantId != "" {
		parts = append(parts, item.TenantId)
	}
	if item.State != "" {
		parts = append(parts, item.State)
	}
	if item.ProcessInstanceKey != "" {
		parts = append(parts, "pi:"+item.ProcessInstanceKey)
	}
	if item.ElementInstanceKey != "" {
		parts = append(parts, "ei:"+item.ElementInstanceKey)
	}
	parts = append(parts, "r:"+strconv.FormatInt(int64(item.Retries), 10))
	if item.Deadline != nil {
		parts = append(parts, "d:"+toolx.FormatTime(*item.Deadline, showTimezoneOffset))
	}
	if item.ErrorCode != "" {
		parts = append(parts, "ec:"+item.ErrorCode)
	}
	if item.ErrorMessage != "" {
		parts = append(parts, "err:"+truncateHumanMessage(item.ErrorMessage, flagGetErrorMessageLimit))
	}
	return parts
}

func formatJobUpdatePlanItems(items []job.UpdatePlanItem) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		switch item.Name {
		case "retries":
			if item.Status == string(job.RetryChangeUnchanged) {
				parts = append(parts, fmt.Sprintf("retries: %s (unchanged)", item.After))
				continue
			}
			if item.Before == "" {
				parts = append(parts, fmt.Sprintf("retries: %s", item.After))
				continue
			}
			parts = append(parts, fmt.Sprintf("retries: %s -> %s", item.Before, item.After))
		case "timeout":
			parts = append(parts, fmt.Sprintf("timeout: set to %s", item.After))
		case string(job.MutationModeTechnicalFailure):
			parts = append(parts, "technical failure: submit")
		case "retryBackoff":
			parts = append(parts, fmt.Sprintf("retry backoff: %s", item.After))
		case "message":
			parts = append(parts, fmt.Sprintf("message: %s", item.After))
		default:
			parts = append(parts, fmt.Sprintf("%s: %s", item.Name, item.After))
		}
	}
	return strings.Join(parts, "; ")
}

func formatJobMutationMode(mode job.MutationMode) string {
	switch mode {
	case job.MutationModeTechnicalFailure:
		return "technical failure"
	case job.MutationModeBPMNError:
		return "BPMN error"
	case job.MutationModeCompletion:
		return "completion"
	default:
		return string(mode)
	}
}

func derefInt32(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}
