// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"os"
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
		if err := validateOpsPurgeOrphanProcessInstancesReportFlags(); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		request := ops.OrphanPurgeRequest{
			CommandName:  opsPurgeOrphanProcessInstancesCommandName,
			DryRun:       flagDryRun,
			AutoConfirm:  flagCmdAutoConfirm,
			Automation:   automationModeEnabled(cmd),
			OutputMode:   pickMode().String(),
			Selection:    populatePISearchFilterOpts(),
			BatchSize:    resolvePISearchSize(cmd, cfg),
			Limit:        flagGetPILimit,
			Workers:      flagWorkers,
			ReportFile:   flagOpsPurgeOrphanReportFile,
			ReportFormat: flagOpsPurgeOrphanReportFormat,
			StartedAt:    time.Now().UTC(),
		}
		if !flagDryRun && !flagCmdAutoConfirm {
			planRequest := request
			planRequest.DryRun = true
			planned, err := cli.PurgeOrphanProcessInstances(cmd.Context(), planRequest, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan ops purge orphan process instances: %w", err))
			}
			if err := rejectDeletePlanRequiringForce(planned.DeletionPlan.DryRunPreview); err != nil {
				abortOpsPurgeOrphanProcessInstancesAfterReport(cmd, log, cfg, markOpsPurgeOrphanProcessInstancesLocalFailure(planned, ops.WorkflowStepStatusBlocked, err), err)
				return
			}
			if planned.Discovery.Count > 0 {
				if err := validateOpsPurgeAutomationConfirmation(request, planned.Discovery.Count); err != nil {
					abortOpsPurgeOrphanProcessInstancesAfterReport(cmd, log, cfg, markOpsPurgeOrphanProcessInstancesLocalFailure(planned, ops.WorkflowStepStatusBlocked, err), err)
					return
				}
				prompt := fmt.Sprintf("You are about to delete %d orphan process instance(s). Do you want to proceed?", len(planned.DeletionPlan.AffectedKeys))
				if len(planned.DeletionPlan.AffectedKeys) > planned.Discovery.Count {
					prompt = fmt.Sprintf("You have requested to delete %d orphan process instance(s), but due to dependencies, a total of %d instance(s) with %d root instance(s) will be deleted. Do you want to proceed?", planned.Discovery.Count, len(planned.DeletionPlan.AffectedKeys), len(planned.DeletionPlan.RootKeys))
				}
				if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
					abortOpsPurgeOrphanProcessInstancesAfterReport(cmd, log, cfg, markOpsPurgeOrphanProcessInstancesLocalFailure(planned, ops.WorkflowStepStatusConfirmationFailed, err), err)
					return
				}
			}
			request.DiscoveredKeys = append(typex.Keys{}, planned.Discovery.Keys...)
		}
		result, err := cli.PurgeOrphanProcessInstances(cmd.Context(), request, collectOptions()...)
		if err != nil {
			if reportErr := writeOpsPurgeOrphanProcessInstancesReport(result, cfg); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge orphan process instances: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge orphan process instances: %w", err))
		}
		if err := writeOpsPurgeOrphanProcessInstancesReport(result, cfg); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops purge orphan process instances audit report: %w", err))
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
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")
	fs.StringVar(&flagOpsPurgeOrphanReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsPurgeOrphanReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsPurgeOrphanProcessInstancesCmd, CommandMutationStateChanging)
	setContractSupport(opsPurgeOrphanProcessInstancesCmd, ContractSupportFull)
	setAutomationSupport(opsPurgeOrphanProcessInstancesCmd, AutomationSupportFull, "supports unattended dry-run previews and auto-confirmed purges with shared machine output")
}

func validateOpsPurgeAutomationConfirmation(request ops.OrphanPurgeRequest, discoveredCount int) error {
	if request.DryRun || request.AutoConfirm || !request.Automation || discoveredCount == 0 {
		return nil
	}
	return missingDependentFlagsf("%s --automation requires --auto-confirm when deletion targets are discovered", request.CommandName)
}

func validateOpsPurgeOrphanProcessInstancesReportFlags() error {
	if flagOpsPurgeOrphanReportFormat != "" && flagOpsPurgeOrphanReportFile == "" {
		return missingDependentFlagsf("--report-format requires --report-file")
	}
	if flagOpsPurgeOrphanReportFile == "" {
		return nil
	}
	_, err := opsWorkflowReportFormatForPath(flagOpsPurgeOrphanReportFile, OpsWorkflowReportFormat(flagOpsPurgeOrphanReportFormat))
	if err != nil {
		return invalidFlagValuef("%v", err)
	}
	return nil
}

func abortOpsPurgeOrphanProcessInstancesAfterReport(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, result ops.OrphanPurgeResult, err error) {
	if reportErr := writeOpsPurgeOrphanProcessInstancesReport(result, cfg); reportErr != nil {
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

func writeOpsPurgeOrphanProcessInstancesReport(result ops.OrphanPurgeResult, cfg *config.Config) error {
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
		data, err = renderOpsPurgeOrphanProcessInstancesMarkdownReport(report)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(result.Request.ReportFile, data, 0o600)
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
