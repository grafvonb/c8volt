// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagGetIncidentKeys         []string
	flagGetIncidentMessageLimit int
)

var getIncidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Fetch incidents by key",
	Long: "Fetch Camunda incidents by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. Each unique incident key is fetched once and rendered through the shared get output modes.\n\n" +
		"Human output is compact for terminal diagnosis, while --json returns the stable incident payload for automation. Use --error-message-limit to shorten long human error messages.",
	Example: `  ./c8volt get incident --key 2251799813685249
  ./c8volt get inc --key 2251799813685249 --key 2251799813685250
  printf '%s\n' 2251799813685249 2251799813685250 | ./c8volt get incident -
  ./c8volt get pi --with-incidents --keys-only | ./c8volt get inc -
  ./c8volt --json get incident --key 2251799813685249
  ./c8volt --keys-only get incident --key 2251799813685249`,
	Aliases: []string{"incidents", "inc"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateOptionalDashArg(args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		return silenceUsageForError(cmd, validateGetIncidentFlagValues(cmd))
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagGetIncidentKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no incident keys provided or found to fetch")))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("incident key %q is not a valid key", firstBadKey))
		}

		log.Debug(fmt.Sprintf("fetching incidents for key(s) [%s], render mode: %s", keys, pickMode()))
		incidents, err := cli.GetIncidents(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("get incidents: %w", err))
		}
		if err := listIncidentsView(cmd, incidents, flagGetIncidentMessageLimit); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render incidents: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getIncidentCmd)

	fs := getIncidentCmd.Flags()
	fs.StringSliceVarP(&flagGetIncidentKeys, "key", "k", nil, "incident key(s) to fetch; repeat or combine with stdin '-'")
	fs.IntVar(&flagGetIncidentMessageLimit, "error-message-limit", 0, "maximum characters to show for human incident messages; 0 keeps full messages")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when fetching multiple incidents (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new incident lookups after the first error")

	useInvalidInputFlagErrors(getIncidentCmd)
	setCommandMutation(getIncidentCmd, CommandMutationReadOnly)
	setContractSupport(getIncidentCmd, ContractSupportFull)
	setAutomationSupport(getIncidentCmd, AutomationSupportFull, "supports shared machine output and stdin key pipelines")
}

func validateGetIncidentFlagValues(cmd *cobra.Command) error {
	if flagGetIncidentMessageLimit < 0 {
		return invalidFlagValuef("--error-message-limit must be non-negative")
	}
	if pickMode() == RenderModeJSON && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--error-message-limit cannot be combined with --json")
	}
	if pickMode() == RenderModeKeysOnly && cmd != nil && cmd.Flags().Changed("error-message-limit") {
		return mutuallyExclusiveFlagsf("--error-message-limit cannot be combined with --keys-only")
	}
	if ok, firstBadKey, _ := validateKeys(flagGetIncidentKeys); !ok {
		return invalidFlagValuef("incident key %q is not a valid key", firstBadKey)
	}
	return nil
}

func resetGetIncidentFlagState() {
	flagGetIncidentKeys = nil
	flagGetIncidentMessageLimit = 0
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
}
