// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
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
	renderOutputLine(cmd, "updated job %s: %s", result.Key, result.Status)
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
