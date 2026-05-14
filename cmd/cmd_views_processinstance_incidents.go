// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

const (
	indirectProcessTreeIncidentNote    = "process instance is marked as having incidents, but no direct incidents were found; inspect the process tree for child incident details"
	indirectProcessTreeIncidentWarning = "one or more incident markers may refer to incidents in the process-instance tree; inspect with walk pi --key <key> --with-incidents"
)

// incidentEnrichedProcessInstancesView renders direct process-instance incident enrichment.
func incidentEnrichedProcessInstancesView(cmd *cobra.Command, resp process.IncidentEnrichedProcessInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, incidentEnrichedProcessInstancesWithAgeMeta(resp))
	}
	needsIndirectIncidentWarning := renderProcessInstanceActivityRows(cmd, activityFromIncidentEnriched(resp).Items)
	if needsIndirectIncidentWarning {
		renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
	}
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
}

func renderIncidentEnrichedProcessInstanceRows(cmd *cobra.Command, resp process.IncidentEnrichedProcessInstances) (bool, error) {
	return renderProcessInstanceActivityRows(cmd, activityFromIncidentEnriched(resp).Items), nil
}

func incidentTreeBranch(index, total int) string {
	if index == total-1 {
		return "└─ "
	}
	return "├─ "
}

type incidentEnrichedProcessInstancesJSONWithMeta struct {
	Total int32                                     `json:"total,omitempty"`
	Items []process.IncidentEnrichedProcessInstance `json:"items,omitempty"`
	Meta  processInstanceAgeMeta                    `json:"meta"`
}

// incidentEnrichedProcessInstancesWithAgeMeta keeps incident JSON compatible with default process-instance age metadata.
func incidentEnrichedProcessInstancesWithAgeMeta(resp process.IncidentEnrichedProcessInstances) incidentEnrichedProcessInstancesJSONWithMeta {
	meta := processInstanceAgeMeta{WithAge: true, AgeDaysBy: map[string]int{}}
	for _, it := range resp.Items {
		if age, ok := processInstanceAgeDays(it.Item.StartDate); ok {
			meta.AgeDaysBy[it.Item.Key] = age
		}
	}
	if len(meta.AgeDaysBy) == 0 {
		meta.AgeDaysBy = nil
	}
	return incidentEnrichedProcessInstancesJSONWithMeta{
		Total: resp.Total,
		Items: resp.Items,
		Meta:  meta,
	}
}

// processInstanceHasIndirectIncidentMarker detects tree-propagated incident markers without direct incident details.
func processInstanceHasIndirectIncidentMarker(item process.IncidentEnrichedProcessInstance) bool {
	return item.Item.Incident && len(item.Incidents) == 0
}

// incidentHumanLine formats an incident detail line with compact attributes.
func incidentHumanLine(incident incident.ProcessInstanceIncidentDetail) string {
	return incidentHumanLineWithMessageLimit(incident, flagGetPIIncidentMessageLimit)
}

// incidentHumanLineWithMessageLimit formats shared incident rows for process-instance and plain incident output.
func incidentHumanLineWithMessageLimit(incident incident.ProcessInstanceIncidentDetail, messageLimit int) string {
	row := compactFlatRow(flatRowProcessInstanceIncident(incident))
	message := "m:" + truncateIncidentHumanMessage(incident.ErrorMessage, messageLimit)
	if row == "" {
		return message
	}
	return row + " " + message
}

func flatRowProcessInstanceIncident(incident incident.ProcessInstanceIncidentDetail) flatRow {
	key := incident.IncidentKey
	if key == "" {
		key = "unknown"
	}
	jobKey := incident.JobKey
	if jobKey == "" {
		jobKey = "n/a"
	}
	return flatRow{
		key,
		incident.ErrorType,
		incident.State,
		"j:" + jobKey,
		toolx.FormatNumericZoneTimestamp(incident.CreationTime),
		incidentAgeTag(incident.CreationTime),
		prefixedIncidentField("root", incident.RootProcessInstanceKey),
		prefixedIncidentField("fn", incident.FlowNodeId),
		prefixedIncidentField("fni", incident.FlowNodeInstanceKey),
	}
}

func incidentAgeTag(creationTime string) string {
	age, ok := processInstanceAgeDays(creationTime)
	if !ok {
		return ""
	}
	if age == 0 {
		return "(today)"
	}
	return fmt.Sprintf("(%d days ago)", age)
}

func incidentListHumanLineWithMessageLimit(item incident.ProcessInstanceIncidentDetail, messageLimit int) string {
	lines := formatIncidentListRows([]incident.ProcessInstanceIncidentDetail{item}, messageLimit, false)
	if len(lines) == 0 {
		return ""
	}
	return lines[0]
}

func formatIncidentListRows(incidents []incident.ProcessInstanceIncidentDetail, messageLimit int, omitMessage bool) []string {
	rows := make([]flatRow, 0, len(incidents))
	tails := make([]string, 0, len(incidents))
	for _, incident := range incidents {
		rows = append(rows, flatRowIncident(incident))
		if omitMessage {
			tails = append(tails, "")
			continue
		}
		tails = append(tails, "m:"+truncateIncidentHumanMessage(incident.ErrorMessage, messageLimit))
	}
	return formatFlatRowsWithTails(rows, tails)
}

func flatRowIncident(incident incident.ProcessInstanceIncidentDetail) flatRow {
	key := incident.IncidentKey
	if key == "" {
		key = "unknown"
	}
	jobKey := incident.JobKey
	if jobKey == "" {
		jobKey = "n/a"
	}
	return flatRow{
		key,
		incident.TenantId,
		incident.ErrorType,
		incident.State,
		"j:" + jobKey,
		toolx.FormatNumericZoneTimestamp(incident.CreationTime),
		incidentAgeTag(incident.CreationTime),
		incident.ProcessDefinitionId,
		prefixedIncidentField("pi", incident.ProcessInstanceKey),
		prefixedIncidentField("root", incident.RootProcessInstanceKey),
		prefixedIncidentField("fn", incident.FlowNodeId),
		prefixedIncidentField("fni", incident.FlowNodeInstanceKey),
	}
}

func prefixedIncidentField(prefix string, value string) string {
	if value == "" {
		return ""
	}
	return prefix + ":" + value
}

func formatFlatRowsWithTails(rows []flatRow, tails []string) []string {
	prefixes := formatFlatRows(rows)
	out := make([]string, 0, len(prefixes))
	for i, prefix := range prefixes {
		line := strings.TrimRight(prefix, " ")
		if i < len(tails) && tails[i] != "" {
			if line != "" {
				line += " "
			}
			line += tails[i]
		}
		out = append(out, line)
	}
	return out
}

// truncateIncidentHumanMessage applies the incident message display limit.
func truncateIncidentHumanMessage(message string, limit int) string {
	return truncateHumanMessage(message, limit)
}

func truncateHumanMessage(message string, limit int) string {
	if limit <= 0 {
		return message
	}
	runes := []rune(message)
	if len(runes) <= limit {
		return message
	}
	return string(runes[:limit]) + "..."
}
