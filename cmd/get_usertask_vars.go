// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
)

// renderSelectedUserTasks performs the single eligible enrichment pass before
// selecting the ordinary or enriched view for a keyed result.
func renderSelectedUserTasks(cmd *cobra.Command, cli task.API, result task.UserTasks) error {
	if !shouldEnrichSelectedUserTasks(result) {
		if err := userTasksView(cmd, result); err != nil {
			return fmt.Errorf("render user tasks: %w", err)
		}
		return nil
	}
	enriched, err := enrichSelectedUserTasks(cmd, cli, result)
	if err != nil {
		return err
	}
	if err := variableEnrichedUserTasksView(cmd, enriched); err != nil {
		return fmt.Errorf("render user task variables: %w", err)
	}
	return nil
}

// enrichSelectedUserTasks delegates the single selected-collection pass used
// by keyed, incremental-search, and collected-search execution.
func enrichSelectedUserTasks(cmd *cobra.Command, cli task.API, result task.UserTasks) (task.VariableEnrichedUserTasks, error) {
	enriched, err := cli.EnrichUserTasksWithVariables(cmd.Context(), result, collectOptions()...)
	if err != nil {
		return task.VariableEnrichedUserTasks{}, fmt.Errorf("get user task variables: %w", err)
	}
	return enriched, nil
}

// shouldEnrichSelectedUserTasks excludes successful no-ops and output modes
// whose existing machine contract contains only keys or a numeric total.
func shouldEnrichSelectedUserTasks(result task.UserTasks) bool {
	return flagGetUserTaskWithVars && !flagGetUserTaskTotal && pickMode() != RenderModeKeysOnly && len(result.Items) > 0
}
