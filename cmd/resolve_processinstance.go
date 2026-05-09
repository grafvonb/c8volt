// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

var (
	flagResolvePIKeys []string
)

var resolveProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Resolve process-instance incidents by key",
	Long: "Resolve process-instance incidents by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. For each unique process instance, c8volt expands to the process-instance family, discovers active incidents at command start for direct incidents on in-scope instances, resolves that fixed incident set, and reports process instances with no active incidents as skipped.\n\n" +
		"By default c8volt validates the affected root and descendant instances and asks for confirmation before resolving active incidents in the family. Use --dry-run to preview the family scope and incident resolution plan without submitting mutations.\n\n" +
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
		if err := validateResolveJSONGuardrails("process-instance"); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
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

		results, err := resolveProcessInstancesWithPlan(cmd, cli, keys, true)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		_ = results
	},
}

func resolveProcessInstancesWithPlan(cmd *cobra.Command, cli process.API, keys types.Keys, firstPage bool) (process.ProcessInstanceResolutionResults, error) {
	planned, err := planProcessInstanceDryRunPreview(cmd, cli, "resolve", keys)
	if err != nil {
		return process.ProcessInstanceResolutionResults{}, err
	}
	plan := planned.Plan
	if flagDryRun {
		if pickMode() != RenderModeJSON {
			if err := renderProcessInstanceDryRunPreview(cmd, planned.Preview); err != nil {
				return process.ProcessInstanceResolutionResults{}, fmt.Errorf("render resolve dry-run scope: %w", err)
			}
		}
		opts := append(collectOptions(), processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)))
		results, err := cli.ResolveProcessInstancesIncidents(cmd.Context(), plan.Collected, flagWorkers, opts...)
		renderErr := renderProcessInstanceResolutionResults(cmd, results)
		if err != nil {
			return results, fmt.Errorf("resolve process-instance incident dry-run: %w", err)
		}
		if renderErr != nil {
			return results, fmt.Errorf("render resolve process-instance dry-run result: %w", renderErr)
		}
		return results, nil
	}
	printDryRunExpansionWarning(cmd, plan)

	if firstPage {
		impact := planned.Impact
		affectedCount, rootCount, requestedCount := impact.Affected, impact.Roots, impact.Requested
		prompt := fmt.Sprintf("You are about to inspect %d process instance(s) and resolve active incidents found in that family. Do you want to proceed?", affectedCount)
		if affectedCount > requestedCount {
			prompt = fmt.Sprintf("You have requested to resolve incidents for %d process instance(s), but due to the process-instance family scope, %d instance(s) with %d root instance(s) will be inspected and active incidents found in that family will be resolved. Do you want to proceed?", requestedCount, affectedCount, rootCount)
		}
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			return process.ProcessInstanceResolutionResults{}, err
		}
	}

	opts := append(collectOptions(), processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)))
	results, err := cli.ResolveProcessInstancesIncidents(cmd.Context(), plan.Collected, flagWorkers, opts...)
	renderErr := renderProcessInstanceResolutionResults(cmd, results)
	if err != nil {
		return results, fmt.Errorf("resolve process-instance incidents: %w", err)
	}
	if renderErr != nil {
		return results, fmt.Errorf("render resolve process-instance result: %w", renderErr)
	}
	return results, nil
}

func init() {
	resolveCmd.AddCommand(resolveProcessInstanceCmd)

	fs := resolveProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagResolvePIKeys, "key", "k", nil, "process instance key(s) to resolve; repeat or combine with stdin '-'")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview process-instance incident resolutions without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after resolution requests are accepted without incident confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when resolving multiple process instances (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new process-instance resolutions after the first error")

	useInvalidInputFlagErrors(resolveProcessInstanceCmd)
	setCommandMutation(resolveProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(resolveProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(resolveProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and per-process-instance incident mutation results")
}
