// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
)

// parseUpdateVariablesFromFlags selects exactly one variable payload source
// for resource update commands and decodes it as a JSON object.
func parseUpdateVariablesFromFlags(cmd *cobra.Command, raw string, filePath string) (map[string]any, error) {
	varsChanged := cmd.Flags().Changed("vars")
	varsFileChanged := cmd.Flags().Changed("vars-file")
	if varsChanged && varsFileChanged {
		return nil, mutuallyExclusiveFlagsf("--vars cannot be combined with --vars-file")
	}
	if varsFileChanged {
		if filePath == "" {
			return nil, invalidFlagValuef("--vars-file requires a file path")
		}
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, invalidFlagValuef("--vars-file could not be read: %v", err)
		}
		return parseUpdateVariables(string(data), "--vars-file")
	}
	return parseUpdateVariables(raw, "--vars")
}

// parseUpdateVariables enforces the shared non-null JSON-object payload contract.
func parseUpdateVariables(raw string, source string) (map[string]any, error) {
	if raw == "" {
		return nil, invalidFlagValuef("--vars or --vars-file is required and must be a JSON object")
	}
	var variables map[string]any
	if err := json.Unmarshal([]byte(raw), &variables); err != nil {
		return nil, invalidFlagValuef("%s must be a valid JSON object: %v", source, err)
	}
	if variables == nil {
		return nil, invalidFlagValuef("%s must be a JSON object", source)
	}
	return variables, nil
}
