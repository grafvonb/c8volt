// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
)

// variableEnrichedUserTasksView renders the selected enriched collection as
// one JSON envelope or as ordinary task rows with nested effective variables.
func variableEnrichedUserTasksView(cmd *cobra.Command, result task.VariableEnrichedUserTasks, valueLimit int) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	if err := renderVariableEnrichedUserTaskSearchPage(cmd, result.Items, valueLimit); err != nil {
		return err
	}
	if flagQuiet {
		return nil
	}
	return writeUserTaskLine(cmd.OutOrStdout(), fmt.Sprintf("found: %d", result.Total))
}

// renderVariableEnrichedUserTaskSearchPage writes only selected enriched rows;
// callers retain ownership of a single final summary after traversal.
func renderVariableEnrichedUserTaskSearchPage(cmd *cobra.Command, items []task.VariableEnrichedUserTask, valueLimit int) error {
	if flagQuiet {
		return nil
	}
	rows := make([]flatRow, 0, len(items))
	for _, enriched := range items {
		rows = append(rows, flatRowUserTask(enriched.Item))
	}
	lines := formatFlatRows(rows)
	for index, enriched := range items {
		if err := writeUserTaskLine(cmd.OutOrStdout(), lines[index]); err != nil {
			return err
		}
		if len(enriched.Variables) == 0 {
			continue
		}
		if err := writeUserTaskLine(cmd.OutOrStdout(), "└─ vars:"); err != nil {
			return err
		}
		for variableIndex, variable := range enriched.Variables {
			line := "   " + incidentTreeBranch(variableIndex, len(enriched.Variables)) + variableValueHumanLine(variable, valueLimit)
			if err := writeUserTaskLine(cmd.OutOrStdout(), line); err != nil {
				return err
			}
		}
	}
	return nil
}
