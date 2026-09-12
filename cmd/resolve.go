// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import "github.com/spf13/cobra"

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Resolve operational incidents",
	Long: `Resolve operational incidents.

Choose incident keys or process-instance families. Resolution changes state and waits for clearance unless --no-wait is set.`,
	Example: `  ./c8volt resolve incident --key <incident-key>
  ./c8volt resolve incident --key <incident-key> --key <another-incident-key>
  printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | ./c8volt resolve incident -`,
	Aliases: []string{"res"},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"reslove", "resovle"},
}

func init() {
	rootCmd.AddCommand(resolveCmd)

	addBackoffFlagsAndBindings(resolveCmd)
	setCommandMutation(resolveCmd, CommandMutationStateChanging)
}

func validateResolveJSONGuardrails(target string) error {
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for resolve %s", target)
	}
	return nil
}
