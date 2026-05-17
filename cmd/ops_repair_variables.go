// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

// parseOpsRepairVariablesFromFlags reuses the update-pi JSON object parser only when repair variable flags are present.
func parseOpsRepairVariablesFromFlags(cmd *cobra.Command, raw string, filePath string) (map[string]any, string, error) {
	if cmd == nil || (!cmd.Flags().Changed("vars") && !cmd.Flags().Changed("vars-file")) {
		return nil, "", nil
	}
	variables, err := parseUpdateProcessInstanceVariablesFromFlags(cmd, raw, filePath)
	if err != nil {
		return nil, "", err
	}
	if cmd.Flags().Changed("vars-file") {
		return variables, filePath, nil
	}
	return variables, "", nil
}
