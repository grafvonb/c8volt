// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

const opsPurgeOrphanProcessInstancesCommandName = "ops purge orphan-process-instances"

var opsPurgeOrphanProcessInstancesCmd = &cobra.Command{
	Use:   "orphan-process-instances",
	Short: "Preview orphan process-instance cleanup",
	Long: "Preview orphan child process-instance cleanup.\n\n" +
		"The dry-run workflow discovers child process instances with missing parents, validates the delete plan that would be used for the frozen discovered key set, and reports the planned purge without submitting deletion requests.",
	Example: `  ./c8volt ops purge orphan-process-instances --dry-run
  ./c8volt ops purge orphan-process-instances --dry-run --state active --limit 10
  ./c8volt ops purge orphan-process-instances --dry-run --bpmn-process-id order-process --json`,
	Aliases: []string{"orphan-pi", "orphan-pis"},
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validatePISearchFlags(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if !flagDryRun {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("--dry-run is required until confirmed orphan purge is implemented")))
		}

		request := ops.OrphanPurgeRequest{
			CommandName: opsPurgeOrphanProcessInstancesCommandName,
			DryRun:      flagDryRun,
			AutoConfirm: flagCmdAutoConfirm,
			Automation:  automationModeEnabled(cmd),
			OutputMode:  pickMode().String(),
			Selection:   populatePISearchFilterOpts(),
			BatchSize:   resolvePISearchSize(cmd, cfg),
			Limit:       flagGetPILimit,
			Workers:     flagWorkers,
			StartedAt:   time.Now().UTC(),
		}
		result, err := cli.PurgeOrphanProcessInstances(cmd.Context(), request, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge orphan process instances: %w", err))
		}
		if err := renderOpsPurgeOrphanProcessInstancesResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops purge orphan process instances: %w", err))
		}
	},
}

func init() {
	opsPurgeCmd.AddCommand(opsPurgeOrphanProcessInstancesCmd)
	useInvalidInputFlagErrors(opsPurgeOrphanProcessInstancesCmd)

	fs := opsPurgeOrphanProcessInstancesCmd.Flags()
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview orphan process-instance cleanup without submitting deletion requests")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	registerPISharedDateRangeFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching child process instances to inspect across all pages")
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to narrow orphan-child discovery")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")
	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when validating the delete plan (default: min(targets, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation work after the first error")

	setCommandMutation(opsPurgeOrphanProcessInstancesCmd, CommandMutationStateChanging)
	setContractSupport(opsPurgeOrphanProcessInstancesCmd, ContractSupportFull)
	setAutomationSupport(opsPurgeOrphanProcessInstancesCmd, AutomationSupportFull, "supports unattended dry-run previews with shared machine output")
}
