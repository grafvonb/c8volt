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
		"Use the jobKey exposed by incident-aware process-instance output to inspect the matching runtime job directly. Human output is compact for terminal diagnosis, while --json returns the stable lookup payload for automation. Use --error-message-limit to shorten long human error messages. Job lookup is supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.",
	Example: `  ./c8volt get job --key 2251799813711967
  ./c8volt --json get job --key 2251799813711967`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateGetJobFlags(); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		result, err := cli.LookupJob(cmd.Context(), flagGetJobKey, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get job: %w", err))
		}
		if err := jobLookupView(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render job lookup result: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getJobCmd)

	fs := getJobCmd.Flags()
	fs.StringVar(&flagGetJobKey, "key", "", "job key to inspect")
	fs.IntVar(&flagGetErrorMessageLimit, "error-message-limit", 0, "truncate error messages in human output to this many characters; 0 keeps full messages")

	useInvalidInputFlagErrors(getJobCmd)
	setCommandMutation(getJobCmd, CommandMutationReadOnly)
	setContractSupport(getJobCmd, ContractSupportFull)
	setFlagContractRequired(getJobCmd, "key")
}

func validateGetJobFlags() error {
	if strings.TrimSpace(flagGetJobKey) == "" {
		return invalidFlagValuef("job lookup requires a non-empty --key")
	}
	if flagGetErrorMessageLimit < 0 {
		return invalidFlagValuef("--error-message-limit must be non-negative")
	}
	return nil
}
