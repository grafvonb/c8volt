// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strconv"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx"
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
		showTimezoneOffset := commandShowTimezoneOffset(cmd)
		for _, item := range result.Items {
			renderOutputLine(cmd, "%s", formatOpsSlowProcessAnalysisRootRow(item, showTimezoneOffset))
			if len(item.Timeline) > 0 {
				renderOutputLine(cmd, "elements:")
				for _, entry := range item.Timeline {
					renderOutputLine(cmd, "%s", formatOpsSlowProcessAnalysisTimelineRow(entry, showTimezoneOffset))
				}
			}
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

// formatOpsSlowProcessAnalysisTimelineRow renders one calculated detail row without implying BPMN causality.
func formatOpsSlowProcessAnalysisTimelineRow(entry ops.SlowProcessAnalysisTimelineEntry, showTimezoneOffset bool) string {
	switch entry.Kind {
	case ops.SlowProcessAnalysisTimelineEntryKindTransition:
		return formatOpsSlowProcessAnalysisTransitionRow(entry)
	default:
		return formatOpsSlowProcessAnalysisElementRow(entry, showTimezoneOffset)
	}
}

// formatOpsSlowProcessAnalysisElementRow keeps runtime element details compact beneath a process-instance root.
func formatOpsSlowProcessAnalysisElementRow(entry ops.SlowProcessAnalysisTimelineEntry, showTimezoneOffset bool) string {
	row := flatRow{
		entry.ElementInstanceKey,
		entry.Type,
		entry.ElementID,
		entry.State,
		prefixedElementField("s", toolx.FormatTimestamp(entry.StartDate, showTimezoneOffset)),
		prefixedElementField("e", toolx.FormatTimestamp(entry.EndDate, showTimezoneOffset)),
		opsSlowProcessAnalysisDurationField(entry.Duration, entry.DurationAvailable),
		opsSlowProcessAnalysisProcessShareField(entry.ProcessDurationShare),
	}
	if marker := opsSlowProcessAnalysisIncidentMarker(entry); marker != "" {
		row = append(row, marker)
	}
	return compactFlatRow(row)
}

// formatOpsSlowProcessAnalysisTransitionRow renders adjacent chronological timing with the required arrow grammar.
func formatOpsSlowProcessAnalysisTransitionRow(entry ops.SlowProcessAnalysisTimelineEntry) string {
	row := flatRow{
		entry.FromElementID + " -> " + entry.ToElementID + ":",
		entry.Duration,
		opsSlowProcessAnalysisProcessShareField(entry.ProcessDurationShare),
	}
	return compactFlatRow(row)
}

// opsSlowProcessAnalysisDurationField emits measured or unavailable detail durations in the shared dur: shape.
func opsSlowProcessAnalysisDurationField(duration string, available bool) string {
	if available {
		return "dur:" + duration
	}
	return "dur:-"
}

// opsSlowProcessAnalysisProcessShareField renders the process-duration share when service calculations provide one.
func opsSlowProcessAnalysisProcessShareField(value int) string {
	if value <= 0 {
		return ""
	}
	return "PI:" + strconv.Itoa(value) + "%"
}

// opsSlowProcessAnalysisIncidentMarker returns the compact incident marker allowed in element rows.
func opsSlowProcessAnalysisIncidentMarker(entry ops.SlowProcessAnalysisTimelineEntry) string {
	if !entry.HasIncident {
		return ""
	}
	if entry.IncidentKey != "" {
		return "inc!:" + entry.IncidentKey
	}
	return "inc!"
}
