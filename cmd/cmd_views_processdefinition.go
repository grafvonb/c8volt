// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// processDefinitionView renders a single process definition through the shared get-output contract.
func processDefinitionView(cmd *cobra.Command, item process.ProcessDefinition) error {
	return itemView(cmd, item, pickMode(), oneLinePD, func(it process.ProcessDefinition) string { return it.Key })
}

// listProcessDefinitionsView renders process-definition collections in human, JSON, or keys-only mode.
func listProcessDefinitionsView(cmd *cobra.Command, resp process.ProcessDefinitions) error {
	return listOrJSONFlat(cmd, resp, resp.Items, pickMode(), flatRowPD, func(it process.ProcessDefinition) string { return it.Key })
}

// processDefinitionWatchView keeps each refresh body identical to normal list output.
func processDefinitionWatchView(cmd *cobra.Command, snapshot process.ProcessDefinitionWatchSnapshot) error {
	return listProcessDefinitionsView(cmd, process.ProcessDefinitions{
		Total: snapshot.Total,
		Items: snapshot.Items,
	})
}

// formatProcessDefinitionFlatRows returns aligned human rows for process-definition lists.
func formatProcessDefinitionFlatRows(items []process.ProcessDefinition) []string {
	rows := make([]flatRow, 0, len(items))
	for _, it := range items {
		rows = append(rows, flatRowPD(it))
	}
	return formatFlatRows(rows)
}

// oneLinePD formats a compact process-definition row for single-item human output.
func oneLinePD(it process.ProcessDefinition) string {
	return compactFlatRow(flatRowPD(it))
}

// flatRowPD mirrors the process-definition human order while allowing statistics to remain an optional tail column.
func flatRowPD(it process.ProcessDefinition) flatRow {
	vTag := ""
	if it.ProcessVersionTag != "" {
		vTag = "/" + it.ProcessVersionTag
	}
	row := flatRow{it.Key, it.TenantId, it.BpmnProcessId, fmt.Sprintf("v%d%s", it.ProcessVersion, vTag)}
	if it.Statistics != nil {
		stats := it.Statistics
		incidentTag := ""
		if stats.IncidentCountSupported {
			incidentTag = fmt.Sprintf(" inc:%s", zeroAsMinus(stats.Incidents))
		}
		row = append(row, fmt.Sprintf("[ac:%s cp:%s cx:%s%s]",
			zeroAsMinus(stats.Active),
			zeroAsMinus(stats.Completed),
			zeroAsMinus(stats.Canceled),
			incidentTag,
		))
	}
	return row
}
