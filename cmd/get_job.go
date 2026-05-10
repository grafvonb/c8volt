// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagGetJobKey            string
	flagGetErrorMessageLimit int
)

var getJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Inspect a job by key",
	Long: "Inspect a Camunda job by key.\n\n" +
		"Use the jobKey exposed by incident-aware process-instance output to inspect the matching runtime job directly. Use --json for the stable job payload, or --error-message-limit to shorten long error messages. Getting jobs by key is supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.",
	Example: `  ./c8volt get job --key 2251799813711967
  ./c8volt --json get job --key 2251799813711967`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateGetJobFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		item, err := cli.GetJob(cmd.Context(), flagGetJobKey, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get job: %w", err))
		}
		if err := jobView(cmd, item); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getJobCmd)

	fs := getJobCmd.Flags()
	fs.StringVar(&flagGetJobKey, "key", "", "job key to inspect")
	fs.IntVar(&flagGetErrorMessageLimit, "error-message-limit", 0, "maximum characters to show for error messages; 0 keeps full messages")

	useInvalidInputFlagErrors(getJobCmd)
	setCommandMutation(getJobCmd, CommandMutationReadOnly)
	setContractSupport(getJobCmd, ContractSupportFull)
	setFlagContractRequired(getJobCmd, "key")
}

func validateGetJobFlags(cmd *cobra.Command) error {
	if strings.TrimSpace(flagGetJobKey) == "" {
		return invalidFlagValuef("get job requires a non-empty --key")
	}
	if flagGetErrorMessageLimit < 0 {
		return invalidFlagValuef("--error-message-limit must be non-negative")
	}
	if pickMode() == RenderModeJSON && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--error-message-limit cannot be combined with --json")
	}
	return nil
}
