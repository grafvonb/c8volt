// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	flagResolvePIKeys []string
)

var resolveProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Resolve process-instance incidents by key",
	Long: "Resolve process-instance incidents by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. For each unique process instance, c8volt discovers active incidents at command start, resolves that fixed incident set, and reports process instances with no active incidents as skipped.\n\n" +
		"By default c8volt waits until the initially discovered incidents are no longer active by polling process-instance incident lookup through the incident service.",
	Example: `  ./c8volt resolve process-instance --key 2251799813685250
  ./c8volt resolve pi --key 2251799813685250 --key 2251799813685260
  printf '%s\n' 2251799813685250 2251799813685260 | ./c8volt resolve process-instance -
  printf '%s\n' 2251799813685250 | ./c8volt resolve pi --key 2251799813685260 -`,
	Aliases: []string{"pi"},
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
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagResolvePIKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process instance keys provided or found to resolve")))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("process instance key %q is not a valid key", firstBadKey))
		}

		results, err := cli.ResolveProcessInstancesIncidents(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("resolve process-instance incidents: %w", err))
		}
		if err := renderProcessInstanceResolutionResults(cmd, results); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render resolve process-instance result: %w", err))
		}
	},
}

func init() {
	resolveCmd.AddCommand(resolveProcessInstanceCmd)

	fs := resolveProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagResolvePIKeys, "key", "k", nil, "process instance key(s) to resolve; repeat or combine with stdin '-'")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when resolving multiple process instances (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new process-instance resolutions after the first error")

	useInvalidInputFlagErrors(resolveProcessInstanceCmd)
	setCommandMutation(resolveProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(resolveProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(resolveProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and per-process-instance incident mutation results")
}
