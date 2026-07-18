// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"math"
	"strconv"
	"strings"
	"time"

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
		seen := map[string]struct{}{}
		for _, item := range result.Items {
			if item.Key == "" {
				continue
			}
			if _, ok := seen[item.Key]; ok {
				continue
			}
			seen[item.Key] = struct{}{}
			renderOutputLine(cmd, "%s", item.Key)
		}
	default:
		showTimezoneOffset := commandShowTimezoneOffset(cmd)
		for _, item := range result.Items {
			renderOutputLine(cmd, "%s", formatOpsSlowProcessAnalysisRootRow(item, showTimezoneOffset))
			if len(item.Timeline) > 0 {
				renderOutputLine(cmd, "└─ elements:")
				for i, entry := range item.Timeline {
					renderOutputLine(cmd, "   %s%s", incidentTreeBranch(i, len(item.Timeline)), formatOpsSlowProcessAnalysisTimelineRow(entry, item, showTimezoneOffset))
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
	if relative := opsSlowProcessAnalysisHumanRelativeBar(item.RelativeBar, item.RelativePercentile); relative != "" {
		row = append(row, relative)
	}
	return compactFlatRow(row)
}

// formatOpsSlowProcessAnalysisTimelineRow renders one calculated detail row without implying BPMN causality.
func formatOpsSlowProcessAnalysisTimelineRow(entry ops.SlowProcessAnalysisTimelineEntry, root ops.SlowProcessAnalysisProcessInstance, showTimezoneOffset bool) string {
	switch entry.Kind {
	case ops.SlowProcessAnalysisTimelineEntryKindTransition:
		return formatOpsSlowProcessAnalysisTransitionRow(entry, root)
	default:
		return formatOpsSlowProcessAnalysisElementRow(entry, root, showTimezoneOffset)
	}
}

// formatOpsSlowProcessAnalysisElementRow keeps runtime element details compact beneath a process-instance root.
func formatOpsSlowProcessAnalysisElementRow(entry ops.SlowProcessAnalysisTimelineEntry, root ops.SlowProcessAnalysisProcessInstance, showTimezoneOffset bool) string {
	row := flatRow{
		entry.ElementInstanceKey,
		entry.Type,
		entry.ElementID,
		entry.State,
		prefixedElementField("s", opsSlowProcessAnalysisDetailTimestamp(entry.StartDate, root.StartDate, showTimezoneOffset)),
		prefixedElementField("e", opsSlowProcessAnalysisDetailTimestamp(entry.EndDate, root.StartDate, showTimezoneOffset)),
		opsSlowProcessAnalysisDurationField(entry.Duration, entry.DurationAvailable),
		opsSlowProcessAnalysisHumanRelativeBar(entry.RelativeBar, entry.RelativePercentile),
		opsSlowProcessAnalysisProcessShareField(entry, root),
	}
	if marker := opsSlowProcessAnalysisIncidentMarker(entry); marker != "" {
		row = append(row, marker)
	}
	return compactFlatRow(row)
}

// formatOpsSlowProcessAnalysisTransitionRow renders adjacent chronological timing with the required arrow grammar.
func formatOpsSlowProcessAnalysisTransitionRow(entry ops.SlowProcessAnalysisTimelineEntry, root ops.SlowProcessAnalysisProcessInstance) string {
	row := flatRow{
		entry.FromElementID + " -> " + entry.ToElementID + ":",
		entry.Duration,
		opsSlowProcessAnalysisHumanRelativeBar(entry.RelativeBar, entry.RelativePercentile),
		opsSlowProcessAnalysisProcessShareField(entry, root),
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

// opsSlowProcessAnalysisHumanRelativeBar keeps unlabeled comparison bars adjacent to durations.
func opsSlowProcessAnalysisHumanRelativeBar(raw string, percentile int) string {
	if raw == "" {
		return ""
	}
	if percentile < 0 {
		percentile = 0
	}
	if percentile > 100 {
		percentile = 100
	}
	filled := int(math.Round(float64(percentile) / 10))
	if filled < 0 {
		filled = 0
	}
	if filled > 10 {
		filled = 10
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + " " + strconv.Itoa(percentile) + "%]"
}

// opsSlowProcessAnalysisDetailTimestamp renders child rows compactly when the root row already carries the same date.
func opsSlowProcessAnalysisDetailTimestamp(value string, rootStartDate string, showTimezoneOffset bool) string {
	if value == "" {
		return ""
	}
	if showTimezoneOffset {
		return toolx.FormatTimestamp(value, showTimezoneOffset)
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	rootStart, err := time.Parse(time.RFC3339Nano, rootStartDate)
	if err != nil || t.Year() != rootStart.Year() || t.YearDay() != rootStart.YearDay() {
		return toolx.FormatTimestamp(value, showTimezoneOffset)
	}
	return t.Format("15:04:05.000")
}

// opsSlowProcessAnalysisProcessShareField renders the process-duration share when service calculations provide one.
func opsSlowProcessAnalysisProcessShareField(entry ops.SlowProcessAnalysisTimelineEntry, root ops.SlowProcessAnalysisProcessInstance) string {
	if root.DurationAvailable && root.DurationMillis > 0 && entry.DurationAvailable && entry.DurationMillis > 0 {
		share := float64(entry.DurationMillis) * 100 / float64(root.DurationMillis)
		if share > 0 && share < 1 {
			return "PI:<1%"
		}
		rounded := int(math.Round(share))
		if rounded > 0 {
			return "PI:" + strconv.Itoa(rounded) + "%"
		}
	}
	if entry.ProcessDurationShare <= 0 {
		return ""
	}
	return "PI:" + strconv.Itoa(entry.ProcessDurationShare) + "%"
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
