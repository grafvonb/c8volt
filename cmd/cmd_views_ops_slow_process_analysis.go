// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"math"
	"sort"
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
		rootBarContext := opsSlowProcessAnalysisRootBarContext(result.Items)
		for _, item := range result.Items {
			renderOutputLine(cmd, "%s", formatOpsSlowProcessAnalysisRootRow(item, rootBarContext, showTimezoneOffset))
			if len(item.Timeline) > 0 {
				if flagOpsAnalyseSlowProcessInstanceWithFullTimeline {
					renderOpsSlowProcessAnalysisFullTimeline(cmd, item, showTimezoneOffset)
				} else {
					renderOpsSlowProcessAnalysisHotspotSummary(cmd, item, showTimezoneOffset)
				}
			}
		}
		renderOutputLine(cmd, "process instances: %d", result.Count)
	}
	return nil
}

// renderOpsSlowProcessAnalysisHotspotSummary renders the compact default detail view.
func renderOpsSlowProcessAnalysisHotspotSummary(cmd *cobra.Command, item ops.SlowProcessAnalysisProcessInstance, showTimezoneOffset bool) {
	summary := opsSlowProcessAnalysisDefaultHotspotSummary(item)
	renderOutputLine(cmd, "└─ slowest elements:")
	totalRows := len(summary.Rows)
	if summary.HiddenRowCount > 0 {
		totalRows++
	}
	for i, entry := range summary.Rows {
		renderOutputLine(cmd, "   %s%s", incidentTreeBranch(i, totalRows), formatOpsSlowProcessAnalysisSummaryRow(entry, item, showTimezoneOffset))
		renderOpsSlowProcessAnalysisListenerLines(cmd, "   ", i, totalRows, entry.Listeners, showTimezoneOffset)
	}
	if summary.HiddenRowCount > 0 {
		renderOutputLine(cmd, "   %s%s", incidentTreeBranch(totalRows-1, totalRows), opsSlowProcessAnalysisHiddenRowsSummary(summary.HiddenRowCount))
	}
}

// renderOpsSlowProcessAnalysisFullTimeline restores the chronological detail tree for audit/debug inspection.
func renderOpsSlowProcessAnalysisFullTimeline(cmd *cobra.Command, item ops.SlowProcessAnalysisProcessInstance, showTimezoneOffset bool) {
	renderOutputLine(cmd, "└─ elements:")
	for i, entry := range item.Timeline {
		renderOutputLine(cmd, "   %s%s", incidentTreeBranch(i, len(item.Timeline)), formatOpsSlowProcessAnalysisTimelineRow(entry, item, showTimezoneOffset))
		if entry.Kind == ops.SlowProcessAnalysisTimelineEntryKindElement {
			renderOpsSlowProcessAnalysisListenerLines(cmd, "   ", i, len(item.Timeline), entry.Listeners, showTimezoneOffset)
		}
	}
}

const opsSlowProcessAnalysisHotspotMinimumProcessShare = 1

type opsSlowProcessAnalysisHotspotSummary struct {
	Rows           []ops.SlowProcessAnalysisTimelineEntry
	HiddenRowCount int
}

// opsSlowProcessAnalysisDefaultHotspotSummary selects the default rows that make slow, active, or incident work visible.
func opsSlowProcessAnalysisDefaultHotspotSummary(item ops.SlowProcessAnalysisProcessInstance) opsSlowProcessAnalysisHotspotSummary {
	out := opsSlowProcessAnalysisHotspotSummary{}
	for _, entry := range item.Timeline {
		if !opsSlowProcessAnalysisVisibleInDefaultSummary(entry) {
			out.HiddenRowCount++
			continue
		}
		out.Rows = append(out.Rows, entry)
	}
	sort.SliceStable(out.Rows, func(i, j int) bool {
		if out.Rows[i].ProcessDurationShare != out.Rows[j].ProcessDurationShare {
			return out.Rows[i].ProcessDurationShare > out.Rows[j].ProcessDurationShare
		}
		return out.Rows[i].DurationMillis > out.Rows[j].DurationMillis
	})
	return out
}

// opsSlowProcessAnalysisVisibleInDefaultSummary keeps one inclusion pass so rows matching multiple reasons appear once.
func opsSlowProcessAnalysisVisibleInDefaultSummary(entry ops.SlowProcessAnalysisTimelineEntry) bool {
	if entry.Kind != ops.SlowProcessAnalysisTimelineEntryKindElement {
		return false
	}
	if strings.EqualFold(entry.State, string(process.StateActive)) {
		return true
	}
	if entry.HasIncident {
		return true
	}
	return strings.EqualFold(entry.State, string(process.StateCompleted)) &&
		entry.ProcessDurationShare >= opsSlowProcessAnalysisHotspotMinimumProcessShare
}

// formatOpsSlowProcessAnalysisSummaryRow omits element instance keys except when an incident row has no incident key.
func formatOpsSlowProcessAnalysisSummaryRow(entry ops.SlowProcessAnalysisTimelineEntry, root ops.SlowProcessAnalysisProcessInstance, showTimezoneOffset bool) string {
	row := flatRow{
		entry.Type,
		entry.ElementID,
		entry.State,
		prefixedElementField("s", opsSlowProcessAnalysisDetailTimestamp(entry.StartDate, root.StartDate, showTimezoneOffset)),
		prefixedElementField("e", opsSlowProcessAnalysisDetailTimestamp(entry.EndDate, root.StartDate, showTimezoneOffset)),
		opsSlowProcessAnalysisDurationField(entry.Duration, entry.DurationAvailable),
		opsSlowProcessAnalysisHumanDurationBar(entry.DurationMillis, root.DurationMillis),
	}
	if entry.HasIncident && entry.IncidentKey == "" && entry.ElementInstanceKey != "" {
		row = append(row, "eik:"+entry.ElementInstanceKey)
	}
	if marker := opsSlowProcessAnalysisIncidentMarker(entry); marker != "" {
		row = append(row, marker)
	}
	return compactFlatRow(row)
}

// opsSlowProcessAnalysisHiddenRowsSummary describes omitted detail rows and the escape hatch for chronological detail.
func opsSlowProcessAnalysisHiddenRowsSummary(count int) string {
	suffix := "rows"
	if count == 1 {
		suffix = "row"
	}
	return "hidden: " + strconv.Itoa(count) + " instant/fast timeline " + suffix + "; use --with-full-timeline"
}

type opsSlowProcessAnalysisRootBar struct {
	enabled          bool
	maxDurationMilli int64
}

func opsSlowProcessAnalysisRootBarContext(items []ops.SlowProcessAnalysisProcessInstance) opsSlowProcessAnalysisRootBar {
	out := opsSlowProcessAnalysisRootBar{}
	available := 0
	for _, item := range items {
		if !item.DurationAvailable || item.DurationMillis <= 0 {
			continue
		}
		available++
		if item.DurationMillis > out.maxDurationMilli {
			out.maxDurationMilli = item.DurationMillis
		}
	}
	out.enabled = available > 1 && out.maxDurationMilli > 0
	return out
}

// formatOpsSlowProcessAnalysisRootRow keeps slow-analysis root rows aligned with process-instance list output.
func formatOpsSlowProcessAnalysisRootRow(item ops.SlowProcessAnalysisProcessInstance, rootBarContext opsSlowProcessAnalysisRootBar, showTimezoneOffset bool) string {
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
	if rootBarContext.enabled {
		if bar := opsSlowProcessAnalysisHumanDurationBar(item.DurationMillis, rootBarContext.maxDurationMilli); bar != "" {
			row = append(row, bar)
		}
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
		opsSlowProcessAnalysisHumanDurationBar(entry.DurationMillis, root.DurationMillis),
	}
	if marker := opsSlowProcessAnalysisIncidentMarker(entry); marker != "" {
		row = append(row, marker)
	}
	return compactFlatRow(row)
}

// renderOpsSlowProcessAnalysisListenerLines nests listener rows below the owning element timeline row.
func renderOpsSlowProcessAnalysisListenerLines(cmd *cobra.Command, prefix string, branchIndex int, totalBranches int, listeners *[]ops.RuntimeListenerJob, showTimezoneOffset bool) {
	listenerRows := formatOpsSlowProcessAnalysisListenerRows(listeners, showTimezoneOffset)
	if len(listenerRows) == 0 {
		return
	}
	childPrefix := treeChildPrefix(prefix, branchIndex, totalBranches)
	renderOutputLine(cmd, "%s└─ listeners:", childPrefix)
	listenerPrefix := childPrefix + "   "
	for i, row := range listenerRows {
		renderOutputLine(cmd, "%s%s%s", listenerPrefix, incidentTreeBranch(i, len(listenerRows)), row)
	}
}

// formatOpsSlowProcessAnalysisListenerRows renders slow-analysis listener jobs using the shared row grammar.
func formatOpsSlowProcessAnalysisListenerRows(listeners *[]ops.RuntimeListenerJob, showTimezoneOffset bool) []string {
	if listeners == nil || len(*listeners) == 0 {
		return nil
	}
	rows := make([]flatRow, 0, len(*listeners))
	for _, listener := range *listeners {
		rows = append(rows, flatRowOpsSlowProcessAnalysisListenerWithTimezone(listener, showTimezoneOffset))
	}
	return formatFlatRows(rows)
}

// flatRowOpsSlowProcessAnalysisListenerWithTimezone formats one runtime listener job below a timeline element row.
func flatRowOpsSlowProcessAnalysisListenerWithTimezone(item ops.RuntimeListenerJob, showTimezoneOffset bool) flatRow {
	parts := flatRow{
		item.JobKey,
		item.Kind,
		prefixedJobField("lsnr", item.ListenerEventType),
		item.State,
		prefixedJobField("tp", item.Type),
		"r:" + strconv.FormatInt(int64(item.Retries), 10),
		prefixedJobField("worker", item.Worker),
	}
	if item.Deadline != nil {
		parts = append(parts, "d:"+toolx.FormatTime(*item.Deadline, showTimezoneOffset))
	} else {
		parts = append(parts, "")
	}
	if item.ErrorCode != "" {
		parts = append(parts, "ec:"+item.ErrorCode)
	} else {
		parts = append(parts, "")
	}
	if item.ErrorMessage != "" {
		parts = append(parts, "err:"+truncateHumanMessage(item.ErrorMessage, flagGetErrorMessageLimit))
	} else {
		parts = append(parts, "")
	}
	return parts
}

// formatOpsSlowProcessAnalysisTransitionRow renders adjacent chronological timing with the required arrow grammar.
func formatOpsSlowProcessAnalysisTransitionRow(entry ops.SlowProcessAnalysisTimelineEntry, root ops.SlowProcessAnalysisProcessInstance) string {
	row := flatRow{
		entry.FromElementID + " -> " + entry.ToElementID + ":",
		entry.Duration,
		opsSlowProcessAnalysisHumanDurationBar(entry.DurationMillis, root.DurationMillis),
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

// opsSlowProcessAnalysisHumanDurationBar renders a duration as a share of the visible comparison duration.
func opsSlowProcessAnalysisHumanDurationBar(durationMillis int64, comparisonMillis int64) string {
	if durationMillis <= 0 || comparisonMillis <= 0 {
		return ""
	}
	share := float64(durationMillis) * 100 / float64(comparisonMillis)
	if share > 0 && share < 1 {
		return "[" + strings.Repeat("░", 10) + "] <1%"
	}
	percent := int(math.Round(share))
	if percent <= 0 {
		return ""
	}
	if percent > 100 {
		percent = 100
	}
	filled := int(math.Round(float64(percent) / 10))
	if filled > 10 {
		filled = 10
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "] " + strconv.Itoa(percent) + "%"
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
