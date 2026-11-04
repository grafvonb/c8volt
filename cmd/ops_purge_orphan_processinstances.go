// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/tenant"
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
	Long: `Delete child process instances whose parent is missing.

The workflow discovers orphan candidates and plans family deletion before
confirmation. Use --dry-run to review the scope and --report-file to save an
audit report. Use --auto-confirm for unattended execution.

--limit bounds child candidates inspected, including those found to have parents.`,
	Example: `  c8volt ops purge orphan-process-instances --dry-run
  c8volt ops purge orphan-process-instances --state completed --report-file orphans.md`,
	Aliases: []string{"orphan-pi", "opi"},
	Args: func(cmd *cobra.Command, args []string) error {
		// BID validation runs after bootstrap so failures use the configured logger and one envelope.
		if searchBusinessIDFlag(cmd) != nil {
			return nil
		}
		return cobra.NoArgs(cmd, args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if searchBusinessIDFlag(cmd) != nil && len(args) > 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("business-ID cleanup does not accept positional keys or stdin"))
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
			Selection:    populatePISearchFilterOpts(cmd),
			BatchSize:    resolvePISearchSize(cmd, cfg),
			Limit:        flagGetPILimit,
			Workers:      flagWorkers,
			ReportFile:   flagOpsPurgeOrphanReportFile,
			ReportFormat: flagOpsPurgeOrphanReportFormat,
			StartedAt:    time.Now().UTC(),
		}
		if request.Selection.BusinessId != nil {
			runBusinessIDOrphanPurge(cmd, cli, log, cfg, request)
			return
		}
		initializeOpsTenantContextHumanReporting(cmd, cfg, false)
		progress := configureOpsPurgeOrphanProcessInstancesProgress(cmd, &request)
		if !flagDryRun && !effectiveAutoConfirm {
			planRequest := request
			planRequest.DryRun = true
			planned, err := purgeOrphanProcessInstancesWithCommandActivity(cmd, planRequest, func() (ops.OrphanPurgeResult, error) {
				return cli.PurgeOrphanProcessInstances(cmd.Context(), planRequest, append(collectOptions(), foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())...)
			})
			if err != nil {
				progress.Close()
				reportErr := writeOpsPurgeOrphanProcessInstancesReport(planned, cfg, OpsWorkflowReportPreserveExisting)
				if renderErr := renderOpsProcessInstancePurgeFailure(cmd, log, planned, "orphan purge", len(planned.DeletionPlan.RequestedKeys), planned.Deletion.Items, err, reportErr); renderErr != nil {
					log.Error("could not render process-instance purge result")
				}
				exitAfterRenderedResult(cfg.App.NoErrCodes, err)
			}
			if err := rejectDeletePlanRequiringForce(planned.DeletionPlan.DryRunPreview); err != nil {
				abortOpsPurgeOrphanProcessInstancesAfterReport(cmd, log, cfg, markOpsPurgeOrphanProcessInstancesLocalFailure(planned, ops.WorkflowStepStatusBlocked, err), err)
				return
			}
			if planned.Discovery.Count > 0 {
				ctx := attachOpsDiscoveryTenantContext(cmd, cfg, planned.DeletionPlan.TenantEvidence)
				printOpsTenantContextForCommand(cmd, ctx)
				prompt := opsPurgeOrphanProcessInstancesConfirmationPrompt(planned)
				if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
					abortOpsPurgeOrphanProcessInstancesAfterReport(cmd, log, cfg, markOpsPurgeOrphanProcessInstancesLocalFailure(planned, ops.WorkflowStepStatusConfirmationFailed, err), err)
					return
				}
			}
			request.DiscoveredKeys = append(typex.Keys{}, planned.Discovery.Keys...)
		}
		result, err := purgeOrphanProcessInstancesWithCommandActivity(cmd, request, func() (ops.OrphanPurgeResult, error) {
			return cli.PurgeOrphanProcessInstances(cmd.Context(), request, append(collectOptions(), foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())...)
		})
		progress.Close()
		result = attachOpsPurgeOrphanProcessInstancesResultTenantContext(cmd, cfg, result)
		if err != nil {
			reportErr := writeOpsPurgeOrphanProcessInstancesReport(result, cfg, opsPurgeOrphanProcessInstancesReportWriteMode(result))
			if renderErr := renderOpsProcessInstancePurgeFailure(cmd, log, result, "orphan purge", len(result.DeletionPlan.RequestedKeys), result.Deletion.Items, err, reportErr); renderErr != nil {
				log.Error("could not render process-instance purge result")
			}
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		if err := writeOpsPurgeOrphanProcessInstancesReport(result, cfg, opsPurgeOrphanProcessInstancesReportWriteMode(result)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops purge orphan process instances audit report: %w", err))
		}
		if err := renderOpsPurgeOrphanProcessInstancesResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops purge orphan process instances: %w", err))
		}
	},
}

// purgeOrphanProcessInstancesWithCommandActivity distinguishes reviewed-plan execution from fresh discovery activity.
func purgeOrphanProcessInstancesWithCommandActivity(cmd *cobra.Command, request ops.OrphanPurgeRequest, run func() (ops.OrphanPurgeResult, error)) (ops.OrphanPurgeResult, error) {
	msg := formatOpsPurgeOrphanProcessInstancesActivity(request, false)
	if request.DiscoveredKeys != nil || request.FrozenPlan != nil {
		msg = "deleting orphan process-instance purge scope"
	} else if !request.DryRun {
		msg = formatOpsPurgeOrphanProcessInstancesActivity(request, true)
	}
	stopActivity := startCommandActivity(cmd, msg)
	defer stopActivity()
	return run()
}

func formatOpsPurgeOrphanProcessInstancesActivity(request ops.OrphanPurgeRequest, beforeDeletion bool) string {
	if beforeDeletion {
		return "checking orphan process-instance candidates before delete"
	}
	return "discovering orphan process-instance candidates"
}

func opsPurgeOrphanProcessInstancesConfirmationPrompt(planned ops.OrphanPurgeResult) string {
	return fmt.Sprintf(
		"orphan purge: %d orphan candidate(s), %d affected process instance(s) across %d root(s) will be deleted. Do you want to proceed?",
		planned.Discovery.Count,
		len(planned.DeletionPlan.AffectedKeys),
		len(planned.DeletionPlan.RootKeys),
	)
}

func init() {
	opsPurgeCmd.AddCommand(opsPurgeOrphanProcessInstancesCmd)
	useInvalidInputFlagErrors(opsPurgeOrphanProcessInstancesCmd)

	fs := opsPurgeOrphanProcessInstancesCmd.Flags()
	fs.String("bid", "", "exact opaque business ID to select within tenant scope (only c8.10+)")
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
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation work after the first error")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagForce, "force", false, "allow cancellation when deletion encounters nonterminal process instances")
	fs.StringVar(&flagOpsPurgeOrphanReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsPurgeOrphanReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsPurgeOrphanProcessInstancesCmd, CommandMutationStateChanging)
	setContractSupport(opsPurgeOrphanProcessInstancesCmd, ContractSupportFull)
	setAutomationSupport(opsPurgeOrphanProcessInstancesCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed purges with shared machine output")
	setOutputModes(opsPurgeOrphanProcessInstancesCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
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
	ctx := attachOpsPurgeOrphanProcessInstancesReportTenantContext(report, cfg)
	report.TenantContext = cloneTenantContextPtr(ctx)
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

// attachOpsPurgeOrphanProcessInstancesResultTenantContext freezes orphan-purge
// tenant context before command result rendering.
func attachOpsPurgeOrphanProcessInstancesResultTenantContext(cmd *cobra.Command, cfg *config.Config, result ops.OrphanPurgeResult) ops.OrphanPurgeResult {
	ctx := attachOpsDiscoveryTenantContext(cmd, cfg, result.DeletionPlan.TenantEvidence)
	result.Report.TenantContext = cloneTenantContextPtr(ctx)
	return result
}

// attachOpsPurgeOrphanProcessInstancesReportTenantContext derives audit context
// from the frozen orphan-purge delete plan.
func attachOpsPurgeOrphanProcessInstancesReportTenantContext(report ops.OrphanPurgeReport, cfg *config.Config) tenant.Context {
	return opsTenantContextWithEvidence(newDiscoveryTenantContext(configuredTenantID(cfg)), report.DeletionPlan.TenantEvidence)
}
