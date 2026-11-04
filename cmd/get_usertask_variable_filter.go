// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/grafvonb/c8volt/c8volt/task"

var (
	flagGetUserTaskVarExists []string
	flagGetUserTaskVars      []string
	flagGetUserTaskVarLikes  []string
)

// parseUserTaskVariableFilters applies the established process-instance
// grammar to task-owned inputs without reading or mutating process globals.
func parseUserTaskVariableFilters() (task.VariableFilterSet, error) {
	return parseVariableFilters(flagGetUserTaskVarExists, flagGetUserTaskVars, flagGetUserTaskVarLikes)
}
