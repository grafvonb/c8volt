// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// variableEnrichedProcessInstancesView renders process-instance variable enrichment.
func variableEnrichedProcessInstancesView(cmd *cobra.Command, resp process.VariableEnrichedProcessInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, variableEnrichedProcessInstancesWithAgeMeta(resp))
	}
	renderVariableEnrichedProcessInstanceRows(cmd, resp)
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
}

func renderVariableEnrichedProcessInstanceRows(cmd *cobra.Command, resp process.VariableEnrichedProcessInstances) {
	rows := make([]flatRow, 0, len(resp.Items))
	for _, it := range resp.Items {
		rows = append(rows, flatRowPI(it.Item))
	}
	lines := formatFlatRows(rows)
	for i, it := range resp.Items {
		renderOutputLine(cmd, "%s", lines[i])
		for _, variable := range it.Variables {
			renderOutputLine(cmd, "  %s", processInstanceVariableHumanLine(variable))
		}
	}
}

type variableEnrichedProcessInstancesJSONWithMeta struct {
	Total int32                                     `json:"total,omitempty"`
	Items []process.VariableEnrichedProcessInstance `json:"items,omitempty"`
	Meta  processInstanceAgeMeta                    `json:"meta"`
}

// variableEnrichedProcessInstancesWithAgeMeta keeps enriched JSON compatible with default process-instance age metadata.
func variableEnrichedProcessInstancesWithAgeMeta(resp process.VariableEnrichedProcessInstances) variableEnrichedProcessInstancesJSONWithMeta {
	meta := processInstanceAgeMeta{WithAge: true, AgeDaysBy: map[string]int{}}
	for _, it := range resp.Items {
		if age, ok := processInstanceAgeDays(it.Item.StartDate); ok {
			meta.AgeDaysBy[it.Item.Key] = age
		}
	}
	if len(meta.AgeDaysBy) == 0 {
		meta.AgeDaysBy = nil
	}
	return variableEnrichedProcessInstancesJSONWithMeta{
		Total: resp.Total,
		Items: resp.Items,
		Meta:  meta,
	}
}

func processInstanceVariableHumanLine(variable process.ProcessInstanceVariable) string {
	return fmt.Sprintf("%s = %s", variable.Name, variable.Value)
}
