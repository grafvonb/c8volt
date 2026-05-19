// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsPurgeOrphanProcessInstancesCommandName = "ops purge orphan-process-instances"

var (
	flagOpsPurgeOrphanReportFile   string
	flagOpsPurgeOrphanReportFormat string
)

var opsPurgeOrphanProcessInstancesCmd = &cobra.Command{
	Use:   "orphan-process-instances",
	Short: "Purge orphan child process instances",
	Long: "Purge orphan child process instances.\n\n" +
		"The workflow discovers child process instances with missing parents, freezes the discovered key set, validates the delete plan, and then either reports the plan with --dry-run or submits deletion only after confirmation. Use --auto-confirm or --automation for unattended deletion, combine --automation with --json for deterministic machine output, and use --report-file to write an audit report.",
	Example: `  ./c8volt ops purge orphan-process-instances --dry-run
  ./c8volt ops purge orphan-process-instances --dry-run --bpmn-process-id <bpmn-process-id> --limit 25
  ./c8volt ops purge orphan-process-instances --automation --json --dry-run
  ./c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm
  ./c8volt ops purge orphan-process-instances --dry-run --report-file orphan-purge.md
  ./c8volt ops purge orphan-process-instances --state completed --limit 25 --auto-confirm --report-file orphan-purge.json --report-format json`,
	Aliases: []string{"orphan-pi", "opi"},
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
		if err := validateOpsPurgeOrphanProcessInstancesReportFlags(); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		effectiveAutoConfirm := shouldImplicitlyConfirm(cmd)
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsPurgeOrphanReportFile, opsWorkflowReportWriteModeForConfirmedMutation(effectiveAutoConfirm)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		request := ops.OrphanPurgeRequest{
			CommandName:  opsPurgeOrphanProcessInstancesCommandName,
			DryRun:       flagDryRun,
			AutoConfirm:  flagCmdAutoConfirm,
			Automation:   automationModeEnabled(cmd),
			NoWait:       flagNoWait,
			OutputMode:   pickMode().String(),
			Selection:    populatePISearchFilterOpts(),
			BatchSize:    resolvePISearchSize(cmd, cfg),
			Limit:        flagGetPILimit,
			Workers:      flagWorkers,
			ReportFile:   flagOpsPurgeOrphanReportFile,
			ReportFormat: flagOpsPurgeOrphanReportFormat,
			StartedAt:    time.Now().UTC(),
		}
		if !flagDryRun && !effectiveAutoConfirm {
			planRequest := request
			planRequest.DryRun = true
			planned, err := purgeOrphanProcessInstancesWithCommandActivity(cmd, planRequest, func() (ops.OrphanPurgeResult, error) {
				return cli.PurgeOrphanProcessInstances(cmd.Context(), planRequest, collectOptions()...)
			})
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan ops purge orphan process instances: %w", err))
			}
			if err := rejectDeletePlanRequiringForce(planned.DeletionPlan.DryRunPreview); err != nil {
				abortOpsPurgeOrphanProcessInstancesAfterReport(cmd, log, cfg, markOpsPurgeOrphanProcessInstancesLocalFailure(planned, ops.WorkflowStepStatusBlocked, err), err)
				return
			}
			if planned.Discovery.Count > 0 {
				prompt := fmt.Sprintf("You are about to delete %d affected process instance(s) from %d candidate orphan process instance(s). Do you want to proceed?", len(planned.DeletionPlan.AffectedKeys), planned.Discovery.Count)
				if len(planned.DeletionPlan.AffectedKeys) > planned.Discovery.Count {
					prompt = fmt.Sprintf("You have requested to delete %d candidate orphan process instance(s), but due to dependencies, a total of %d affected process instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", planned.Discovery.Count, len(planned.DeletionPlan.AffectedKeys), len(planned.DeletionPlan.RootKeys))
				}
				if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
					abortOpsPurgeOrphanProcessInstancesAfterReport(cmd, log, cfg, markOpsPurgeOrphanProcessInstancesLocalFailure(planned, ops.WorkflowStepStatusConfirmationFailed, err), err)
					return
				}
			}
			request.DiscoveredKeys = append(typex.Keys{}, planned.Discovery.Keys...)
		}
		result, err := purgeOrphanProcessInstancesWithCommandActivity(cmd, request, func() (ops.OrphanPurgeResult, error) {
			return cli.PurgeOrphanProcessInstances(cmd.Context(), request, collectOptions()...)
		})
		if err != nil {
			if reportErr := writeOpsPurgeOrphanProcessInstancesReport(result, cfg, opsPurgeOrphanProcessInstancesReportWriteMode(result)); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge orphan process instances: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge orphan process instances: %w", err))
		}
		if err := writeOpsPurgeOrphanProcessInstancesReport(result, cfg, opsPurgeOrphanProcessInstancesReportWriteMode(result)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops purge orphan process instances audit report: %w", err))
		}
		if err := renderOpsPurgeOrphanProcessInstancesResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops purge orphan process instances: %w", err))
		}
	},
}

func purgeOrphanProcessInstancesWithCommandActivity(cmd *cobra.Command, request ops.OrphanPurgeRequest, run func() (ops.OrphanPurgeResult, error)) (ops.OrphanPurgeResult, error) {
	msg := formatOpsPurgeOrphanProcessInstancesActivity(request, false)
	if request.DiscoveredKeys != nil {
		msg = "deleting orphan process-instance purge scope"
	} else if !request.DryRun {
		msg = formatOpsPurgeOrphanProcessInstancesActivity(request, true)
	}
	stopActivity := startCommandActivity(cmd, msg)
	defer stopActivity()
	return run()
}

func formatOpsPurgeOrphanProcessInstancesActivity(request ops.OrphanPurgeRequest, beforeDeletion bool) string {
	msg := "checking orphan child pi parents"
	if beforeDeletion {
		msg += " before delete"
	}
	if request.BatchSize > 0 {
		msg += fmt.Sprintf("; page size %d", request.BatchSize)
	}
	return msg
}

func init() {
	opsPurgeCmd.AddCommand(opsPurgeOrphanProcessInstancesCmd)
	useInvalidInputFlagErrors(opsPurgeOrphanProcessInstancesCmd)

	fs := opsPurgeOrphanProcessInstancesCmd.Flags()
	fs.BoolVar(&flagDryRun, "dry-run", false, "discover and validate orphan process-instance cleanup without submitting deletion requests")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	registerPISharedDateRangeFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching child process instances to inspect across all pages")
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to narrow orphan-child discovery")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")
	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when validating the delete plan (default: min(targets, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation work after the first error")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")
	fs.StringVar(&flagOpsPurgeOrphanReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsPurgeOrphanReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsPurgeOrphanProcessInstancesCmd, CommandMutationStateChanging)
	setContractSupport(opsPurgeOrphanProcessInstancesCmd, ContractSupportFull)
	setAutomationSupport(opsPurgeOrphanProcessInstancesCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed purges with shared machine output")
}

func validateOpsPurgeOrphanProcessInstancesReportFlags() error {
	return validateOpsWorkflowReportFlags(flagOpsPurgeOrphanReportFile, OpsWorkflowReportFormat(flagOpsPurgeOrphanReportFormat))
}

func abortOpsPurgeOrphanProcessInstancesAfterReport(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, result ops.OrphanPurgeResult, err error) {
	if reportErr := writeOpsPurgeOrphanProcessInstancesReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w; write audit report: %v", err, reportErr))
	}
	handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
}

func markOpsPurgeOrphanProcessInstancesLocalFailure(result ops.OrphanPurgeResult, status ops.WorkflowStepStatus, err error) ops.OrphanPurgeResult {
	finished := time.Now().UTC()
	msg := err.Error()
	result.Outcome = ops.OrphanPurgeOutcomeFailed
	result.Errors = appendOpsPurgeOrphanProcessInstancesError(result.Errors, msg)
	result.Deletion.Status = status
	result.Deletion.Errors = appendOpsPurgeOrphanProcessInstancesError(result.Deletion.Errors, msg)
	result.Report.Outcome = ops.OrphanPurgeOutcomeFailed
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Discovery = result.Discovery
	result.Report.DeletionPlan = result.DeletionPlan
	result.Report.Deletion = result.Deletion
	result.Report.Errors = appendOpsPurgeOrphanProcessInstancesError(result.Report.Errors, msg)
	return result
}

func appendOpsPurgeOrphanProcessInstancesError(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func opsPurgeOrphanProcessInstancesReportWriteMode(result ops.OrphanPurgeResult) OpsWorkflowReportWriteMode {
	return opsWorkflowReportWriteModeForConfirmedMutation(result.DeleteRequested)
}

func writeOpsPurgeOrphanProcessInstancesReport(result ops.OrphanPurgeResult, cfg *config.Config, mode OpsWorkflowReportWriteMode) error {
	if result.Request.ReportFile == "" {
		return nil
	}
	report := enrichOpsPurgeOrphanProcessInstancesReport(result.Report, cfg)
	format, err := opsWorkflowReportFormatForPath(result.Request.ReportFile, OpsWorkflowReportFormat(result.Request.ReportFormat))
	if err != nil {
		return err
	}
	var data []byte
	switch format {
	case OpsWorkflowReportFormatJSON:
		data, err = renderOpsPurgeOrphanProcessInstancesJSONReport(report)
	case OpsWorkflowReportFormatMarkdown:
		data, err = renderOpsPurgeOrphanProcessInstancesMarkdownReport(report, cfg)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return writeOpsWorkflowReportFile(result.Request.ReportFile, data, mode)
}

func enrichOpsPurgeOrphanProcessInstancesReport(report ops.OrphanPurgeReport, cfg *config.Config) ops.OrphanPurgeReport {
	report.C8voltVersion = CurrentBuildInfo().Version
	if cfg != nil {
		report.CamundaVersion = cfg.App.CamundaVersion.String()
		if cfg.ActiveProfile != "" {
			report.ProfileIdentity = "profile:" + cfg.ActiveProfile
		} else {
			report.ProfileIdentity = "default"
		}
	}
	return report
}
