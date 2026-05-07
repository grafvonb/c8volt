// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagUpdatePIKeys []string
	flagUpdatePIVars string
)

var updateProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Update process-instance variables by key",
	Long: "Update process-instance variables by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. By default c8volt waits until requested process-instance-scope variables are visible through the same lookup path as `get pi --with-vars`; add --no-wait to return after the update request is accepted.",
	Example: `  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update process-instance --key 2251799813711967 --vars '{"customerTier":"gold"}'
  printf '%s\n' 2251799813711967 2251799813711968 | ./c8volt update pi - --vars '{"customerTier":"gold"}'
  ./c8volt --json update pi --key 2251799813711967 --vars '{"customerTier":"gold"}' --no-wait`,
	Aliases: []string{"pi"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("update process-instance is not implemented yet")
	},
}

func init() {
	updateCmd.AddCommand(updateProcessInstanceCmd)

	fs := updateProcessInstanceCmd.Flags()
	fs.StringSliceVar(&flagUpdatePIKeys, "key", nil, "process instance key(s) to update; repeat or combine with stdin '-'")
	fs.StringVar(&flagUpdatePIVars, "vars", "", "JSON object with variables to set on each process instance")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the update request is accepted without variable confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when updating multiple process instances (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new updates after the first error")
	_ = updateProcessInstanceCmd.MarkFlagRequired("vars")

	setCommandMutation(updateProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(updateProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(updateProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and accepted results with --no-wait")
}
