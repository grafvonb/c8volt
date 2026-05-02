// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

const indirectProcessTreeIncidentWarning = "no direct incidents on this process instance; check the process tree with walk pi --with-incidents"

// incidentEnrichedProcessInstancesView renders direct process-instance incident enrichment.
func incidentEnrichedProcessInstancesView(cmd *cobra.Command, resp process.IncidentEnrichedProcessInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, incidentEnrichedProcessInstancesWithAgeMeta(resp))
	}
	warnedIndirectIncident := false
	for _, it := range resp.Items {
		renderOutputLine(cmd, "%s", oneLinePI(it.Item))
		for _, incident := range it.Incidents {
			renderOutputLine(cmd, "  %s", incidentHumanLine(incident))
		}
		if processInstanceHasIndirectIncidentMarker(it) && !warnedIndirectIncident {
			renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
			warnedIndirectIncident = true
		}
	}
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
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

// incidentHumanLine formats a human-readable incident detail line with a stable incident key prefix.
func incidentHumanLine(incident process.ProcessInstanceIncidentDetail) string {
	key := incident.IncidentKey
	if key == "" {
		key = "unknown"
	}
	return fmt.Sprintf("incident %s: %s", key, incident.ErrorMessage)
}
