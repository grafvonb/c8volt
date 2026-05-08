// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update existing resources",
	Long: `Update existing resources.

The process-instance command updates process-instance-scope variables on
existing Camunda 8.8 and 8.9 process instances. The job command updates
job retries and timeout by key, with dry-run planning, confirmation prompts,
and optional no-wait submitted output. Camunda 8.7 configurations return an
unsupported-version error before these mutations.`,
	Example: `  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}'
  ./c8volt update pi --key 2251799813711967 --vars-file ./vars.json
  ./c8volt update pi --key 2251799813711967 --vars '{"customerTier":"gold"}' --dry-run
  ./c8volt update job --key 2251799813711967 --retries 3 --dry-run
  ./c8volt update job --key 2251799813711967 --timeout 5m --auto-confirm
  ./c8volt update job --key 2251799813711967 --retries 3 --no-wait --auto-confirm
  ./c8volt update process-instance --key 2251799813711967 --vars '{"customerTier":"gold"}'
  printf '%s\n' 2251799813711967 2251799813711968 | ./c8volt update pi - --vars '{"customerTier":"gold"}'
  ./c8volt --automation --json update pi --key 2251799813711967 --vars '{"customerTier":"gold"}' --no-wait`,
	Aliases: []string{"u"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"updte", "set"},
}

func init() {
	rootCmd.AddCommand(updateCmd)

	addBackoffFlagsAndBindings(updateCmd)
	setCommandMutation(updateCmd, CommandMutationStateChanging)
}
