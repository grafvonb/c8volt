// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// validatePIHasUserTasksMode rejects ambiguous selector combinations before any user-task or process-instance lookup starts.
func validatePIHasUserTasksMode(cmd *cobra.Command, taskKeyCount, keyCount int, filterFlagsSet bool) error {
	if taskKeyCount == 0 {
		return nil
	}
	if keyCount > 0 {
		return mutuallyExclusiveFlagsf("--has-user-tasks cannot be combined with --key or stdin key input")
	}
	if filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPINoIncidentsOnly {
		return mutuallyExclusiveFlagsf("--has-user-tasks cannot be combined with process-instance search filters")
	}
	if flagGetPITotal {
		return mutuallyExclusiveFlagsf("--has-user-tasks cannot be combined with --total")
	}
	if flagGetPILimit > 0 || isPILimitFlagChanged(cmd) {
		return mutuallyExclusiveFlagsf("--has-user-tasks cannot be combined with --limit")
	}
	return nil
}

// normalizeHasUserTasks trims, validates, and deduplicates user-task keys supplied through repeatable --has-user-tasks flags.
func normalizeHasUserTasks(keys types.Keys) (types.Keys, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	out := make(types.Keys, 0, len(keys))
	for i, key := range keys {
		trimmed := strings.TrimSpace(key)
		if !isPositiveDecimalUserTaskKey(trimmed) {
			return nil, invalidFlagValuef("invalid value for --has-user-tasks: %q at index %d is not a positive decimal user task key", key, i)
		}
		out = append(out, trimmed)
	}
	return out.Unique(), nil
}

// isPositiveDecimalUserTaskKey enforces the native Camunda user-task key format accepted by --has-user-tasks.
func isPositiveDecimalUserTaskKey(key string) bool {
	if key == "" {
		return false
	}
	hasNonZero := false
	for _, r := range key {
		if r < '0' || r > '9' {
			return false
		}
		if r != '0' {
			hasNonZero = true
		}
	}
	return hasNonZero
}
