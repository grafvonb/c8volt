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
func variableEnrichedUserTasksView(cmd *cobra.Command, result task.VariableEnrichedUserTasks) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, result)
	}
	if flagQuiet {
		return nil
	}
	rows := make([]flatRow, 0, len(result.Items))
	for _, enriched := range result.Items {
		rows = append(rows, flatRowUserTask(enriched.Item))
	}
	lines := formatFlatRows(rows)
	for index, enriched := range result.Items {
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
			line := "   " + incidentTreeBranch(variableIndex, len(enriched.Variables)) + variableValueHumanLine(variable, 0)
			if err := writeUserTaskLine(cmd.OutOrStdout(), line); err != nil {
				return err
			}
		}
	}
	return writeUserTaskLine(cmd.OutOrStdout(), fmt.Sprintf("found: %d", result.Total))
}
