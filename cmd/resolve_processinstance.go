// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/incident"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

var (
	flagResolvePIKeys []string
)

type resolveProcessInstanceAPI interface {
	process.API
	incident.API
}

var resolveProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Resolve process-instance incidents by key or business ID",
	Long: `Resolve active incidents in selected process-instance families.

Select by key or exact --bid (only c8.10+). The workflow fixes the family and
incident scope before confirmation, then waits for clearance. It does not
change variables or job retries; use ops repair for those changes.

Use --dry-run to review the plan. Families without active incidents need no change.`,
	Example: `  c8volt resolve process-instance --key <process-instance-key> --dry-run
  c8volt resolve process-instance --key <process-instance-key>`,
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
		if cmd.Flags().Changed("bid") {
			if err := validateBusinessIDActionInput(cmd, args); err != nil {
				handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
				return
			}
			runResolveProcessInstanceBusinessID(cmd, cli, cfg, log)
			return
		}
		if cmd.Flags().Changed("limit") || cmd.Flags().Changed("batch-size") {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--limit and --batch-size require --bid"))
			return
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagResolvePIKeys, stdinKeys, log, cfg).Unique()
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

func resolveProcessInstancesWithPlan(cmd *cobra.Command, cli resolveProcessInstanceAPI, keys types.Keys, firstPage bool) (incident.ProcessInstanceResolutionResults, error) {
	planned, err := planProcessInstanceDryRunPreviewWithOptions(cmd, cli, "resolve", keys, collectExplicitPIAdminInputOptions())
	if err != nil {
		return incident.ProcessInstanceResolutionResults{}, err
	}
	plan := planned.Plan
	if flagDryRun {
		if pickMode() != RenderModeJSON {
			if err := renderProcessInstanceDryRunPreview(cmd, planned.Preview); err != nil {
				return incident.ProcessInstanceResolutionResults{}, fmt.Errorf("render resolve dry-run scope: %w", err)
			}
		}
		opts := append(collectExplicitPIAdminInputOptions(), processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)))
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
	renderAttachedTenantContext(cmd)
	printDryRunExpansionWarning(cmd, plan)

	if firstPage {
		impact := planned.Impact
		affectedCount, rootCount, requestedCount := impact.Affected, impact.Roots, impact.Requested
		prompt := fmt.Sprintf("You are about to inspect %d process instance(s) and resolve active incidents found in that family. Do you want to proceed?", affectedCount)
		if affectedCount > requestedCount {
			prompt = fmt.Sprintf("You have requested to resolve incidents for %d process instance(s), but due to the process-instance family scope, %d instance(s) with %d root instance(s) will be inspected and active incidents found in that family will be resolved. Do you want to proceed?", requestedCount, affectedCount, rootCount)
		}
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
			return incident.ProcessInstanceResolutionResults{}, err
		}
	}

	opts := append(collectExplicitPIAdminInputOptions(), processOptions.WithAffectedProcessInstanceCount(len(plan.Collected)))
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
	registerBusinessIDActionFlags(resolveProcessInstanceCmd)
	fs.StringSliceVarP(&flagResolvePIKeys, "key", "k", nil, "process instance key(s) to resolve; repeat or combine with stdin '-'")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview process-instance incident resolutions without submitting mutation")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after resolution requests are accepted without incident confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new process-instance resolutions after the first error")

	useInvalidInputFlagErrors(resolveProcessInstanceCmd)
	setCommandMutation(resolveProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(resolveProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(resolveProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and per-process-instance incident mutation results")
}
