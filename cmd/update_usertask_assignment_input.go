// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// userTaskAssignmentInput freezes exactly one new mode before preview or approval.
// Keys are nonempty, exactly 16 ASCII digits and unique in first-input order.
// Assignee is nonblank and byte-preserved for assign, absent for unassign.
// Override defaults false, requires explicit permission and is false for unassign.
// Omitted CLI workers mean automatic; explicitly supplied workers must be positive.
type userTaskAssignmentInput struct {
	Operation     string
	Keys          typex.Keys
	Assignee      string
	AllowOverride bool
	WantedWorkers int
}

// resolveUpdateUserTaskAssignmentMode checks conflicts before legacy dispatch.
// Assignment selects by presence; boolean selectors select only when true. An
// empty result preserves existing completion/variable dispatch, never a new mode.
func resolveUpdateUserTaskAssignmentMode(cmd *cobra.Command) (string, error) {
	assign := cmd.Flags().Changed("assign")
	unassign, _ := cmd.Flags().GetBool("unassign")
	complete, _ := cmd.Flags().GetBool("complete")
	active := 0
	for _, enabled := range []bool{assign, unassign, complete} {
		if enabled {
			active++
		}
	}
	if active > 1 {
		return "", mutuallyExclusiveFlagsf("--assign, --unassign, and --complete are mutually exclusive")
	}
	if cmd.Flags().Changed("allow-override") && !assign {
		return "", missingDependentFlagsf("--allow-override requires --assign")
	}
	if assign {
		return "assign", nil
	}
	if unassign {
		return "unassign", nil
	}
	return "", nil
}

// parseUpdateUserTaskAssignmentInput validates all local intent without a facade,
// identity resolution, reads or mutations. Invalid input never produces a report.
func parseUpdateUserTaskAssignmentInput(cmd *cobra.Command, args []string) (userTaskAssignmentInput, error) {
	operation, err := resolveUpdateUserTaskAssignmentMode(cmd)
	if err != nil {
		return userTaskAssignmentInput{}, err
	}
	if operation == "" {
		return userTaskAssignmentInput{}, missingDependentFlagsf("--assign or --unassign is required")
	}
	if err := validateOptionalDashArg(args); err != nil {
		return userTaskAssignmentInput{}, err
	}
	if cmd.Flags().Changed("vars") || cmd.Flags().Changed("vars-file") {
		return userTaskAssignmentInput{}, mutuallyExclusiveFlagsf("--%s cannot be combined with --vars or --vars-file", operation)
	}
	if flagNoWait {
		return userTaskAssignmentInput{}, mutuallyExclusiveFlagsf("--%s cannot be combined with --no-wait", operation)
	}
	if cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return userTaskAssignmentInput{}, invalidFlagValuef("--workers must be positive integer")
	}
	if err := validateUpdateUserTaskJSONConfirmation(cmd); err != nil {
		return userTaskAssignmentInput{}, err
	}
	input := userTaskAssignmentInput{Operation: operation}
	if cmd.Flags().Changed("workers") {
		input.WantedWorkers = flagWorkers
	}
	if operation == "assign" {
		input.Assignee, _ = cmd.Flags().GetString("assign")
		if strings.TrimSpace(input.Assignee) == "" {
			return userTaskAssignmentInput{}, invalidFlagValuef("--assign requires a nonblank assignee")
		}
		input.AllowOverride, _ = cmd.Flags().GetBool("allow-override")
	}
	input.Keys, err = collectUserTaskMutationKeys(cmd, args, assignmentOperationLabel(operation))
	if err != nil {
		return userTaskAssignmentInput{}, err
	}
	return input, nil
}
