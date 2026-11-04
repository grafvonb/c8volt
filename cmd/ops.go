// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var opsCmd = &cobra.Command{
	Use:   "ops",
	Short: "Run operational playbooks",
	Long: `Run playbooks for incident repair, retention, cleanup, runtime analysis,
and connection testing. Start with --dry-run on workflows that change data.`,
	Example: `  c8volt ops repair incident --key <incident-key> --dry-run
  c8volt ops execute retention-policy --retention-days 90 --dry-run
  c8volt ops analyse slow-process-instances --bpmn-process-id order --dur-longer 5m`,
	Aliases: []string{"operations"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"op", "operation"},
}

func init() {
	rootCmd.AddCommand(opsCmd)

	addBackoffFlagsAndBindings(opsCmd)
	setCommandMutation(opsCmd, CommandMutationStateChanging)
}
