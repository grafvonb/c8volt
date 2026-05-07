// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// variableEnrichedProcessInstancesView renders process-instance variable enrichment.
func variableEnrichedProcessInstancesView(cmd *cobra.Command, resp process.VariableEnrichedProcessInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, variableEnrichedProcessInstancesWithAgeMeta(resp))
	}
	renderProcessInstanceActivityRows(cmd, activityFromVariableEnriched(resp).Items)
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
}

// renderVariableEnrichedProcessInstanceRows renders aligned process-instance rows followed by their enriched variables.
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

// processInstanceVariableHumanLine keeps values one-line while preserving explicit API/CLI truncation markers.
func processInstanceVariableHumanLine(variable process.ProcessInstanceVariable) string {
	value := compactProcessInstanceVariableValue(variable.Value)
	value, cliTruncated := truncateProcessInstanceVariableHumanValue(value, flagGetPIVarValueLimit)
	labels := processInstanceVariableTruncationLabels(variable.APITruncated, cliTruncated)
	if labels != "" {
		return fmt.Sprintf("%s=%s [%s]", variable.Name, value, labels)
	}
	return fmt.Sprintf("%s=%s", variable.Name, value)
}

// compactProcessInstanceVariableValue JSON-compacts object and array values while leaving other values unchanged.
func compactProcessInstanceVariableValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return value
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return value
	}
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(trimmed)); err != nil {
		return value
	}
	return buf.String()
}

// truncateProcessInstanceVariableHumanValue applies the CLI display limit and reports whether truncation occurred.
func truncateProcessInstanceVariableHumanValue(value string, limit int) (string, bool) {
	if limit <= 0 {
		return value, false
	}
	runes := []rune(value)
	if len(runes) <= limit {
		return value, false
	}
	return string(runes[:limit]) + "...", true
}

// processInstanceVariableTruncationLabels summarizes API-side and CLI-side truncation markers for human output.
func processInstanceVariableTruncationLabels(apiTruncated bool, cliTruncated bool) string {
	switch {
	case apiTruncated && cliTruncated:
		return "api-truncated,cli-truncated"
	case apiTruncated:
		return "api-truncated"
	case cliTruncated:
		return "cli-truncated"
	default:
		return ""
	}
}
