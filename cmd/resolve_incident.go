// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagResolveIncidentKeys []string
)

var resolveIncidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Resolve incidents by key",
	Long: "Resolve incidents by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. Each unique incident key is submitted for resolution and reported independently.\n\n" +
		"By default c8volt waits until each incident is no longer active by polling incident lookup through the incident service.",
	Example: `  ./c8volt resolve incident --key 2251799813685249
  ./c8volt resolve inc --key 2251799813685249 --key 2251799813685250
  printf '%s\n' 2251799813685249 2251799813685250 | ./c8volt resolve incident -
  printf '%s\n' 2251799813685249 | ./c8volt resolve inc --key 2251799813685250 -`,
	Aliases: []string{"inc"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		if err := validateResolveJSONGuardrails("incident"); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagResolveIncidentKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no incident keys provided or found to resolve")))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("incident key %q is not a valid key", firstBadKey))
		}

		results, err := cli.ResolveIncidents(cmd.Context(), keys, flagWorkers, collectOptions()...)
		renderErr := renderIncidentResolutionResults(cmd, results)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("resolve incidents: %w", err))
		}
		if renderErr != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render resolve incident result: %w", renderErr))
		}
	},
}

func init() {
	resolveCmd.AddCommand(resolveIncidentCmd)

	fs := resolveIncidentCmd.Flags()
	fs.StringSliceVarP(&flagResolveIncidentKeys, "key", "k", nil, "incident key(s) to resolve; repeat or combine with stdin '-'")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview incident resolutions without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after the resolution request is accepted without incident confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when resolving multiple incidents (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new incident resolutions after the first error")

	useInvalidInputFlagErrors(resolveIncidentCmd)
	setCommandMutation(resolveIncidentCmd, CommandMutationStateChanging)
	setContractSupport(resolveIncidentCmd, ContractSupportFull)
	setAutomationSupport(resolveIncidentCmd, AutomationSupportFull, "supports shared machine output and per-incident mutation results")
}
