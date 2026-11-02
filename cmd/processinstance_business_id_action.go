// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// registerBusinessIDActionFlags keeps scalar selection controls local to each action.
func registerBusinessIDActionFlags(cmd *cobra.Command) {
	cmd.Flags().String("bid", "", "exact business ID to select in configured tenant scope (Camunda 8.10)")
	cmd.Flags().Int32P("limit", "l", 0, "maximum distinct matching instances; zero selects all; tree expansion is not limited")
	cmd.Flags().Int32P("batch-size", "n", 1000, "maximum process instances per discovery request")
}

// validateBusinessIDActionInput rejects source mixing before stdin or backend access.
func validateBusinessIDActionInput(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("key") || len(args) > 0 {
		return mutuallyExclusiveFlagsf("--bid cannot be combined with --key or stdin '-'")
	}
	if err := validateSearchBusinessIDFlag(cmd); err != nil {
		return err
	}
	limit, _ := cmd.Flags().GetInt32("limit")
	size, _ := cmd.Flags().GetInt32("batch-size")
	if limit < 0 || size < 1 {
		return invalidFlagValuef("--limit must be nonnegative and --batch-size must be positive")
	}
	return nil
}

// businessIDActionSelection maps command-local controls into the shared collection resolver.
func businessIDActionSelection(cmd *cobra.Command) process.ProcessInstanceSelectionRequest {
	limit, _ := cmd.Flags().GetInt32("limit")
	size, _ := cmd.Flags().GetInt32("batch-size")
	return process.ProcessInstanceSelectionRequest{Cardinality: process.ProcessInstanceSelectionCollection, SearchRequest: process.ProcessInstanceSearchRequest{
		Filter: process.ProcessInstanceFilter{BusinessId: searchBusinessIDFlag(cmd)}, Page: process.ProcessInstancePageRequest{Size: size}, Limit: limit,
	}}
}
