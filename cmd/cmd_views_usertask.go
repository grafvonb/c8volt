// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"io"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
)

// userTasksView renders keyed and collected user tasks with one stable
// collection shape in every cardinality.
func userTasksView(cmd *cobra.Command, result task.UserTasks) error {
	switch pickMode() {
	case RenderModeJSON:
		return renderJSONPayload(cmd, RenderModeJSON, result)
	case RenderModeKeysOnly:
		for _, item := range result.Items {
			if err := writeUserTaskLine(cmd.OutOrStdout(), item.Key); err != nil {
				return err
			}
		}
		return nil
	default:
		if flagQuiet {
			return nil
		}
		rows := make([]flatRow, 0, len(result.Items))
		for _, item := range result.Items {
			rows = append(rows, flatRowUserTask(item))
		}
		for _, line := range formatFlatRows(rows) {
			if err := writeUserTaskLine(cmd.OutOrStdout(), line); err != nil {
				return err
			}
		}
		return writeUserTaskLine(cmd.OutOrStdout(), fmt.Sprintf("found: %d", len(result.Items)))
	}
}

// flatRowUserTask keeps optional columns in contract order so list alignment
// retains intentional empty name/assignee cells.
func flatRowUserTask(item task.UserTask) flatRow {
	name := item.Name
	if name == "" {
		name = item.ElementId
	}
	return flatRow{
		item.Key,
		item.State,
		name,
		item.Assignee,
		prefixedElementField("pi", item.ProcessInstanceKey),
		item.TenantId,
	}
}

// writeUserTaskLine returns output failures to the command instead of allowing
// a truncated result to appear successful.
func writeUserTaskLine(writer io.Writer, line string) error {
	_, err := fmt.Fprintln(writer, line)
	return err
}
