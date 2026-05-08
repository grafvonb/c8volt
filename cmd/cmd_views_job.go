// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
)

func jobLookupView(cmd *cobra.Command, result job.LookupResult) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	if !result.Found {
		renderOutputLine(cmd, "job %s: not found", result.Key)
		return nil
	}
	renderOutputLine(cmd, "%s", oneLineJob(result.Job))
	return nil
}

func jobUpdateResultView(cmd *cobra.Command, result job.UpdateResult) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	switch result.Status {
	case "confirmed":
		renderOutputLine(cmd, "updated job %s: confirmed retries=%d", result.Key, derefInt32(result.ConfirmedRetries))
	case "submitted":
		renderOutputLine(cmd, "updated job %s: submitted", result.Key)
	case "confirmation_failed":
		renderOutputLine(cmd, "updated job %s: confirmation failed: %s", result.Key, result.Error)
	case "mutation_failed":
		renderOutputLine(cmd, "updated job %s: mutation failed: %s", result.Key, result.Error)
	default:
		renderOutputLine(cmd, "updated job %s: %s", result.Key, result.Status)
	}
	return nil
}

func jobUpdatePlanView(cmd *cobra.Command, plan job.UpdatePlan, label string) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, plan)
	}
	status := "pending confirmation"
	if label == "dry run" {
		status = "no changes applied"
	}
	if !plan.HasMaterialChange() {
		renderOutputLine(cmd, "%s: update job %s: nothing to update; %s", label, plan.Key, status)
		return nil
	}
	renderOutputLine(cmd, "%s: update job %s: %s; %s", label, plan.Key, formatJobUpdatePlanItems(plan.Items), status)
	return nil
}

func oneLineJob(item job.Job) string {
	parts := []string{"job " + item.Key}
	if item.State != "" {
		parts = append(parts, "state="+item.State)
	}
	parts = append(parts, "retries="+strconv.FormatInt(int64(item.Retries), 10))
	if item.Deadline != nil {
		parts = append(parts, "deadline="+item.Deadline.Format("2006-01-02T15:04:05Z07:00"))
	}
	if item.ProcessInstanceKey != "" {
		parts = append(parts, "processInstanceKey="+item.ProcessInstanceKey)
	}
	if item.ElementInstanceKey != "" {
		parts = append(parts, "elementInstanceKey="+item.ElementInstanceKey)
	}
	if item.ErrorCode != "" {
		parts = append(parts, "errorCode="+item.ErrorCode)
	}
	if item.ErrorMessage != "" {
		parts = append(parts, "errorMessage="+item.ErrorMessage)
	}
	if item.TenantId != "" {
		parts = append(parts, "tenantId="+item.TenantId)
	}
	return strings.Join(parts, " ")
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
			parts = append(parts, fmt.Sprintf("timeout: submit %s", item.After))
		default:
			parts = append(parts, fmt.Sprintf("%s: %s", item.Name, item.After))
		}
	}
	return strings.Join(parts, "; ")
}

func derefInt32(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}
