// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

const (
	indirectProcessTreeIncidentNote    = "no direct incidents found for this process instance"
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
		for _, incident := range it.Incidents {
			renderOutputLine(cmd, "  %s", incidentHumanLine(incident))
		}
		if processInstanceHasIndirectIncidentMarker(it) {
			renderOutputLine(cmd, "  %s", indirectProcessTreeIncidentNote)
			needsIndirectIncidentWarning = true
		}
	}
	return needsIndirectIncidentWarning, nil
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

// incidentHumanLine formats a human-readable incident detail line with a compact incident key prefix.
func incidentHumanLine(incident process.ProcessInstanceIncidentDetail) string {
	key := incident.IncidentKey
	if key == "" {
		key = "unknown"
	}
	return fmt.Sprintf("inc %s: %s", key, truncateIncidentHumanMessage(incident.ErrorMessage, flagGetPIIncidentMessageLimit))
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
