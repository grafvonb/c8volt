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
	flagGetJobState          string
	flagGetJobType           string
	flagGetJobProcessKey     string
	flagGetJobElementKey     string
	flagGetJobElementID      string
	flagGetJobWorker         string
	flagGetJobRetries        int32
	flagGetJobKind           string
	flagGetJobListenerEvent  string
	flagGetJobLimit          int32
	flagGetErrorMessageLimit int
)

var getJobCmd = &cobra.Command{
	Use:   "job",
	Short: "Inspect or search jobs",
	Long: "Inspect or search Camunda jobs.\n\n" +
		"Use --key with the jobKey exposed by incident-aware process-instance output to inspect a matching runtime job directly. Search mode will use list filters such as --state, --type, --pi-key, --element-instance-key, --element-id, --worker, --retries, --kind, --listener-event-type, and --limit. Use --json for the stable job payload, or --error-message-limit to shorten long error messages. Job lookup and search are supported for Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error.",
	Example: `  ./c8volt get job --key <job-key>
  ./c8volt get job --state FAILED --limit 50
  ./c8volt --json get job --key <job-key>`,
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
	fs.StringVar(&flagGetJobState, "state", "", "job state to filter in search mode")
	fs.StringVar(&flagGetJobType, "type", "", "job type to filter in search mode")
	fs.StringVar(&flagGetJobProcessKey, "pi-key", "", "process instance key to filter in search mode")
	fs.StringVar(&flagGetJobElementKey, "element-instance-key", "", "element instance key to filter in search mode")
	fs.StringVar(&flagGetJobElementID, "element-id", "", "BPMN element ID to filter in search mode")
	fs.StringVar(&flagGetJobWorker, "worker", "", "worker name to filter in search mode")
	fs.Int32Var(&flagGetJobRetries, "retries", 0, "retry count to filter in search mode")
	fs.StringVar(&flagGetJobKind, "kind", "", "job kind to filter in search mode")
	fs.StringVar(&flagGetJobListenerEvent, "listener-event-type", "", "listener event type to filter in search mode")
	fs.Int32Var(&flagGetJobLimit, "limit", 0, "maximum number of jobs to return in search mode")
	fs.IntVar(&flagGetErrorMessageLimit, "error-message-limit", 0, "maximum characters to show for error messages; 0 keeps full messages")

	useInvalidInputFlagErrors(getJobCmd)
	setCommandMutation(getJobCmd, CommandMutationReadOnly)
	setContractSupport(getJobCmd, ContractSupportFull)
	setAutomationSupport(getJobCmd, AutomationSupportFull, "supports shared machine output and unattended job reads")
}

func validateGetJobFlags(cmd *cobra.Command) error {
	searchFlags := changedGetJobSearchFlags(cmd)
	if strings.TrimSpace(flagGetJobKey) == "" && len(searchFlags) == 0 {
		return invalidFlagValuef("get job requires a non-empty --key")
	}
	if strings.TrimSpace(flagGetJobKey) != "" && len(searchFlags) > 0 {
		return mutuallyExclusiveFlagsf("--key cannot be combined with job search filters: %s", strings.Join(searchFlags, ", "))
	}
	if len(searchFlags) > 0 {
		return invalidFlagValuef("job search flags are reserved for the job search implementation")
	}
	if flagGetErrorMessageLimit < 0 {
		return invalidFlagValuef("--error-message-limit must be non-negative")
	}
	if pickMode() == RenderModeJSON && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--error-message-limit cannot be combined with --json")
	}
	return nil
}

func changedGetJobSearchFlags(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	names := []string{
		"state",
		"type",
		"pi-key",
		"element-instance-key",
		"element-id",
		"worker",
		"retries",
		"kind",
		"listener-event-type",
		"limit",
	}
	changed := make([]string, 0, len(names))
	for _, name := range names {
		if cmd.Flags().Changed(name) {
			changed = append(changed, "--"+name)
		}
	}
	return changed
}
