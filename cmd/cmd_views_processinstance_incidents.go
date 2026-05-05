// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

const (
	indirectProcessTreeIncidentNote    = "inc: process instance is marked as having incidents, but no direct incidents were found; details may be in the process tree"
	indirectProcessTreeIncidentWarning = "warning: one or more incident markers may refer to incidents in the process-instance tree; inspect with walk pi --key <key> --with-incidents"
)

// incidentEnrichedProcessInstancesView renders direct process-instance incident enrichment.
func incidentEnrichedProcessInstancesView(cmd *cobra.Command, resp process.IncidentEnrichedProcessInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, incidentEnrichedProcessInstancesWithAgeMeta(resp))
	}
	needsIndirectIncidentWarning, err := renderIncidentEnrichedProcessInstanceRows(cmd, resp)
	if err != nil {
		return err
	}
	if needsIndirectIncidentWarning {
		renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
	}
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
}

func renderIncidentEnrichedProcessInstanceRows(cmd *cobra.Command, resp process.IncidentEnrichedProcessInstances) (bool, error) {
	rows := make([]flatRow, 0, len(resp.Items))
	for _, it := range resp.Items {
		rows = append(rows, flatRowPI(it.Item))
	}
	lines := formatFlatRows(rows)
	needsIndirectIncidentWarning := false
	for i, it := range resp.Items {
		renderOutputLine(cmd, "%s", lines[i])
		for j, incident := range it.Incidents {
			renderOutputLine(cmd, "%s%s", incidentTreeBranch(j, len(it.Incidents)), incidentHumanLine(incident))
		}
		if processInstanceHasIndirectIncidentMarker(it) {
			renderOutputLine(cmd, "└─ %s", indirectProcessTreeIncidentNote)
			needsIndirectIncidentWarning = true
		}
	}
	return needsIndirectIncidentWarning, nil
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

// incidentHumanLine formats a human-readable incident detail line with compact attributes.
func incidentHumanLine(incident process.ProcessInstanceIncidentDetail) string {
	key := incident.IncidentKey
	if key == "" {
		key = "unknown"
	}
	message := truncateIncidentHumanMessage(incident.ErrorMessage, flagGetPIIncidentMessageLimit)
	fields := incidentHumanFields(incident, key)
	return fmt.Sprintf("inc: %s message=%s", fields, message)
}

func incidentHumanFields(incident process.ProcessInstanceIncidentDetail, key string) string {
	fields := make([]string, 0, 5)
	fields = append(fields, "key="+key)
	if incident.FlowNodeId != "" {
		fields = append(fields, "flowNodeId="+incident.FlowNodeId)
	}
	if incident.FlowNodeInstanceKey != "" {
		fields = append(fields, "flowNodeInstanceKey="+incident.FlowNodeInstanceKey)
	}
	if incident.ErrorType != "" {
		fields = append(fields, "errorType="+incident.ErrorType)
	}
	if incident.JobKey != "" {
		fields = append(fields, "jobKey="+incident.JobKey)
	}
	return strings.Join(fields, " ")
}

// truncateIncidentHumanMessage applies the human-only incident message display limit.
func truncateIncidentHumanMessage(message string, limit int) string {
	if limit <= 0 {
		return message
	}
	runes := []rune(message)
	if len(runes) <= limit {
		return message
	}
	return string(runes[:limit]) + "..."
}
