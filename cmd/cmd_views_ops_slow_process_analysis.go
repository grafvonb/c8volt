// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// renderOpsSlowProcessAnalysisResult dispatches slow-analysis output through the shared render modes.
func renderOpsSlowProcessAnalysisResult(cmd *cobra.Command, result ops.SlowProcessAnalysisResult) error {
	switch pickMode() {
	case RenderModeJSON:
		return renderJSONPayload(cmd, RenderModeJSON, result)
	case RenderModeKeysOnly:
		for _, item := range result.Items {
			renderOutputLine(cmd, "%s", item.Key)
		}
	default:
		for _, item := range result.Items {
			renderOutputLine(cmd, "%s", formatOpsSlowProcessAnalysisRootRow(item, commandShowTimezoneOffset(cmd)))
		}
		renderOutputLine(cmd, "process instances: %d", result.Count)
	}
	return nil
}

// formatOpsSlowProcessAnalysisRootRow keeps slow-analysis root rows aligned with process-instance list output.
func formatOpsSlowProcessAnalysisRootRow(item ops.SlowProcessAnalysisProcessInstance, showTimezoneOffset bool) string {
	row := flatRowPIWithTimezone(process.ProcessInstance{
		BpmnProcessId:          item.BpmnProcessID,
		EndDate:                item.EndDate,
		Incident:               item.Incident,
		Key:                    item.Key,
		ParentKey:              item.ParentKey,
		ProcessDefinitionKey:   item.ProcessDefinitionKey,
		RootProcessInstanceKey: item.RootProcessInstanceKey,
		ProcessVersion:         item.ProcessVersion,
		StartDate:              item.StartDate,
		State:                  item.State,
		TenantId:               item.TenantID,
	}, showTimezoneOffset)
	if item.DurationAvailable {
		row = append(row, "dur:"+item.Duration)
	} else {
		row = append(row, "dur:-")
	}
	return compactFlatRow(row)
}
