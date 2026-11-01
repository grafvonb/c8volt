// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// validateWalkPISelection requires one starting selector without making BID look like explicit-key input.
func validateWalkPISelection(cmd *cobra.Command) error {
	bid := searchBusinessIDFlag(cmd)
	if bid != nil && cmd.Flags().Changed("key") {
		return mutuallyExclusiveFlagsf("--key and --bid are mutually exclusive")
	}
	if bid == nil && strings.TrimSpace(flagWalkPIKey) == "" {
		return missingDependentFlagsf("one of --key or --bid is required")
	}
	return nil
}

// resolveWalkPIStart keeps tenant-scoped discovery separate from the established key-based traversal authorization.
func resolveWalkPIStart(cmd *cobra.Command, api process.API) (string, process.ProcessInstanceSelection, error) {
	bid := searchBusinessIDFlag(cmd)
	if bid == nil {
		return flagWalkPIKey, process.ProcessInstanceSelection{}, nil
	}
	selection, err := api.ResolveProcessInstanceSelection(cmd.Context(), process.ProcessInstanceSelectionRequest{
		Cardinality:   process.ProcessInstanceSelectionUniqueStart,
		SearchRequest: process.ProcessInstanceSearchRequest{Filter: process.ProcessInstanceFilter{BusinessId: bid}},
	}, collectOptions()...)
	if err != nil {
		return "", selection, err
	}
	if len(selection.Keys) == 0 {
		return "", selection, nil
	}
	return selection.Keys[0], selection, nil
}

// completeWalkPISelector preserves local starting-selector suggestions without marking either alternative required.
func completeWalkPISelector(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 || cmd.Flags().Changed("key") || cmd.Flags().Changed("bid") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var candidates []string
	for _, flag := range []string{"--key", "--bid"} {
		if strings.HasPrefix(flag, toComplete) {
			candidates = append(candidates, flag)
		}
	}
	return candidates, cobra.ShellCompDirectiveNoFileComp
}
