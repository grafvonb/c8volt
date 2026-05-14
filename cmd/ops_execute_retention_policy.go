// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsExecuteRetentionPolicyCommandName = "ops execute retention-policy"

var flagOpsExecuteRetentionPolicyRetentionDays int

var opsExecuteRetentionPolicyCmd = &cobra.Command{
	Use:   "retention-policy",
	Short: "Execute process-instance retention cleanup",
	Long: "Execute process-instance retention cleanup.\n\n" +
		"The workflow discovers process instances older than the required retention age, freezes that seed set, validates the delete plan, and then either reports the plan with --dry-run or submits deletion after confirmation. Use --auto-confirm or --automation for unattended deletion.",
	Example: `  ./c8volt ops execute retention-policy --retention-days 90 --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --auto-confirm --no-wait
  ./c8volt ops execute retention-policy --retention-days 90 --automation --json --no-wait`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateOpsExecuteRetentionPolicyFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		if err := validatePISearchFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
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
		boundary := pickPIDateUpperBound("", flagOpsExecuteRetentionPolicyRetentionDays)
		selection := populatePISearchFilterOpts()
		selection.EndDateBefore = boundary
		effectiveAutoConfirm := shouldImplicitlyConfirm(cmd)
		request := ops.RetentionPolicyRequest{
			CommandName:            opsExecuteRetentionPolicyCommandName,
			RetentionDays:          flagOpsExecuteRetentionPolicyRetentionDays,
			DerivedEndDateBoundary: boundary,
			DryRun:                 flagDryRun,
			AutoConfirm:            flagCmdAutoConfirm,
			Automation:             automationModeEnabled(cmd),
			OutputMode:             pickMode().String(),
			Selection:              selection,
			BatchSize:              resolvePISearchSize(cmd, cfg),
			Limit:                  flagGetPILimit,
			Workers:                flagWorkers,
			NoWait:                 flagNoWait,
			NoStateCheck:           flagNoStateCheck,
			Force:                  flagForce,
			FailFast:               flagFailFast,
			NoWorkerLimit:          flagNoWorkerLimit,
			StartedAt:              time.Now().UTC(),
		}
		if !flagDryRun && !effectiveAutoConfirm {
			planRequest := request
			planRequest.DryRun = true
			planned, err := cli.ExecuteRetentionPolicy(cmd.Context(), planRequest, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan ops execute retention-policy: %w", err))
			}
			if err := rejectOpsExecuteRetentionPolicyPlanRequiringForce(planned.DeletePlan); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			if planned.Discovery.Count > 0 {
				prompt := fmt.Sprintf("You are about to delete %d retention process instance(s). Do you want to proceed?", len(planned.DeletePlan.AffectedKeys))
				if len(planned.DeletePlan.AffectedKeys) > planned.Discovery.Count {
					prompt = fmt.Sprintf("You have requested to delete %d retention process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", planned.Discovery.Count, len(planned.DeletePlan.AffectedKeys), len(planned.DeletePlan.ResolvedRootKeys))
				}
				if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
				}
			}
			request.DiscoveredKeys = append(typex.Keys{}, planned.Discovery.SeedKeys...)
		}
		result, err := cli.ExecuteRetentionPolicy(cmd.Context(), request, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops execute retention-policy: %w", err))
		}
		if err := renderOpsExecuteRetentionPolicyResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops execute retention-policy: %w", err))
		}
	},
}

func init() {
	opsExecuteCmd.AddCommand(opsExecuteRetentionPolicyCmd)
	useInvalidInputFlagErrors(opsExecuteRetentionPolicyCmd)

	fs := opsExecuteRetentionPolicyCmd.Flags()
	fs.IntVar(&flagOpsExecuteRetentionPolicyRetentionDays, "retention-days", 0, "required non-negative age in days for process-instance retention eligibility")
	fs.BoolVar(&flagDryRun, "dry-run", false, "discover and validate retention cleanup without submitting deletion requests")
	fs.StringSliceVarP(&flagGetPIKeys, "key", "k", nil, "unsupported explicit process-instance key selector")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching process instances to inspect across all pages")
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to narrow retention discovery")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")
	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "discover only root process instances")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "discover only child process instances")
	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "discover only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "discover only process instances that have no incidents")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when validating the delete plan and deleting roots (default: min(targets, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation or deletion work after the first error")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking process-instance state before deleting")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")

	setCommandMutation(opsExecuteRetentionPolicyCmd, CommandMutationStateChanging)
	setContractSupport(opsExecuteRetentionPolicyCmd, ContractSupportFull)
	setAutomationSupport(opsExecuteRetentionPolicyCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed retention cleanup with shared machine output")
	setFlagContractRequired(opsExecuteRetentionPolicyCmd, "retention-days")
}

func validateOpsExecuteRetentionPolicyFlags(cmd *cobra.Command) error {
	if cmd == nil || !cmd.Flags().Changed("retention-days") {
		return invalidFlagValuef("ops execute retention-policy requires --retention-days")
	}
	if flagOpsExecuteRetentionPolicyRetentionDays < 0 {
		return invalidFlagValuef("invalid value for --retention-days: %d, expected non-negative integer", flagOpsExecuteRetentionPolicyRetentionDays)
	}
	if len(flagGetPIKeys) > 0 {
		return invalidFlagValuef("retention policy discovers eligible process instances and does not accept explicit process-instance keys")
	}
	return nil
}

func rejectOpsExecuteRetentionPolicyPlanRequiringForce(plan ops.RetentionDeletePlan) error {
	if flagForce || len(plan.NonFinalAffectedItems) == 0 {
		return nil
	}
	return localPreconditionError(fmt.Errorf("refusing to delete retention process-instance scope: %d affected process instance(s) are not in a final state; no delete request was submitted; use --force to cancel the entire affected scope before delete", len(plan.NonFinalAffectedItems)))
}
