// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsExecuteRetentionPolicyCommandName = "ops execute retention-policy"

var (
	flagOpsExecuteRetentionPolicyRetentionDays int
	flagOpsExecuteRetentionPolicyReportFile    string
	flagOpsExecuteRetentionPolicyReportFormat  string
)

var opsExecuteRetentionPolicyCmd = &cobra.Command{
	Use:   "retention-policy",
	Short: "Execute process-instance retention cleanup",
	Long: "Execute process-instance retention cleanup.\n\n" +
		"The workflow discovers process instances older than the required retention age, freezes that candidate set, validates the delete plan, and then either reports the plan with --dry-run or submits deletion after confirmation. Use compatible process-instance filters to narrow discovery, --auto-confirm or --automation for unattended deletion, and --report-file to write an audit report.",
	Example: `  ./c8volt ops execute retention-policy --retention-days 90 --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --state completed --bpmn-process-id order-process --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --automation --json --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --auto-confirm
  ./c8volt ops execute retention-policy --retention-days 90 --auto-confirm --force --workers 4
  ./c8volt ops execute retention-policy --retention-days 90 --dry-run --report-file retention-report.md
  ./c8volt ops execute retention-policy --retention-days 90 --auto-confirm --report-file retention-report.json --report-format json`,
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
		if err := validateOpsExecuteRetentionPolicyReportFlags(); err != nil {
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
			ReportFile:             flagOpsExecuteRetentionPolicyReportFile,
			ReportFormat:           flagOpsExecuteRetentionPolicyReportFormat,
			StartedAt:              time.Now().UTC(),
		}
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsExecuteRetentionPolicyReportFile, opsWorkflowReportWriteModeForConfirmedMutation(effectiveAutoConfirm)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if !flagDryRun && !effectiveAutoConfirm {
			planRequest := request
			planRequest.DryRun = true
			planned, err := cli.ExecuteRetentionPolicy(cmd.Context(), planRequest, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan ops execute retention-policy: %w", err))
			}
			if err := rejectOpsExecuteRetentionPolicyPlanRequiringForce(planned.DeletePlan); err != nil {
				abortOpsExecuteRetentionPolicyAfterReport(cmd, log, cfg, markOpsExecuteRetentionPolicyLocalFailure(planned, ops.WorkflowStepStatusBlocked, err), err)
				return
			}
			if len(planned.DeletePlan.ResolvedRootKeys) > 0 {
				prompt := fmt.Sprintf("Retention matched %d candidate process instance(s); delete planning will delete %d affected process instance(s) across %d final root(s).", planned.Discovery.Count, len(planned.DeletePlan.AffectedKeys), len(planned.DeletePlan.ResolvedRootKeys))
				if len(planned.DeletePlan.SkippedSeedKeys) > 0 {
					prompt = fmt.Sprintf("%s %d candidate(s) were skipped because their root is not final.", prompt, len(planned.DeletePlan.SkippedSeedKeys))
				}
				prompt += " Do you want to proceed?"
				if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
					abortOpsExecuteRetentionPolicyAfterReport(cmd, log, cfg, markOpsExecuteRetentionPolicyLocalFailure(planned, ops.WorkflowStepStatusConfirmationFailed, err), err)
					return
				}
			}
			request.DiscoveredKeys = append(typex.Keys{}, planned.Discovery.SeedKeys...)
		}
		result, err := cli.ExecuteRetentionPolicy(cmd.Context(), request, collectOptions()...)
		if err != nil {
			if reportErr := writeOpsExecuteRetentionPolicyReport(result, cfg, opsExecuteRetentionPolicyReportWriteMode(result)); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops execute retention-policy: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops execute retention-policy: %w", err))
		}
		if err := writeOpsExecuteRetentionPolicyReport(result, cfg, opsExecuteRetentionPolicyReportWriteMode(result)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops execute retention-policy audit report: %w", err))
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
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when validating the delete plan and deleting roots (default: min(targets, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation or deletion work after the first error")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking process-instance state before deleting")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")
	fs.StringVar(&flagOpsExecuteRetentionPolicyReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsExecuteRetentionPolicyReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

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

func validateOpsExecuteRetentionPolicyReportFlags() error {
	return validateOpsWorkflowReportFlags(flagOpsExecuteRetentionPolicyReportFile, OpsWorkflowReportFormat(flagOpsExecuteRetentionPolicyReportFormat))
}

func rejectOpsExecuteRetentionPolicyPlanRequiringForce(plan ops.RetentionDeletePlan) error {
	if flagForce || len(plan.NonFinalAffectedItems) == 0 {
		return nil
	}
	return localPreconditionError(fmt.Errorf(
		"refusing to delete retention process-instance scope: %s; no delete request was submitted; use --force to cancel the non-final affected scope before delete",
		formatOpsExecuteRetentionPolicyNonFinalScope(plan),
	))
}

func formatOpsExecuteRetentionPolicyNonFinalScope(plan ops.RetentionDeletePlan) string {
	items := plan.NonFinalAffectedItems
	blockers := newProcessInstanceDryRunRequiresCancelBeforeDelete(items)
	details := []string{
		fmt.Sprintf("retention matched %d candidate process instance(s)", len(plan.SeedKeys)),
		fmt.Sprintf("delete planning expanded to %d affected process instance(s) across %d root(s)", len(plan.AffectedKeys), len(plan.ResolvedRootKeys)),
		fmt.Sprintf("%d non-final descendant process instance(s) in otherwise final-root retention scope", len(items)),
		fmt.Sprintf("states: %s", formatProcessInstanceDryRunRequiresCancelBeforeDeleteStates(blockers)),
	}
	if flagVerbose {
		details = append(details, formatProcessInstanceDryRunRequiresCancelBeforeDelete(blockers))
	} else {
		details = append(details, "use --verbose to list keys")
	}
	return strings.Join(details, "; ")
}

func abortOpsExecuteRetentionPolicyAfterReport(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, result ops.RetentionPolicyResult, err error) {
	if reportErr := writeOpsExecuteRetentionPolicyReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w; write audit report: %v", err, reportErr))
	}
	handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
}

func markOpsExecuteRetentionPolicyLocalFailure(result ops.RetentionPolicyResult, status ops.WorkflowStepStatus, err error) ops.RetentionPolicyResult {
	finished := time.Now().UTC()
	msg := err.Error()
	result.Outcome = ops.RetentionPolicyOutcomeFailed
	result.Errors = appendOpsExecuteRetentionPolicyError(result.Errors, msg)
	result.Deletion.Status = status
	result.Deletion.Errors = appendOpsExecuteRetentionPolicyError(result.Deletion.Errors, msg)
	result.Report.Outcome = ops.RetentionPolicyOutcomeFailed
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Discovery = result.Discovery
	result.Report.DeletePlan = result.DeletePlan
	result.Report.Deletion = result.Deletion
	result.Report.Errors = appendOpsExecuteRetentionPolicyError(result.Report.Errors, msg)
	return result
}

func appendOpsExecuteRetentionPolicyError(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func opsExecuteRetentionPolicyReportWriteMode(result ops.RetentionPolicyResult) OpsWorkflowReportWriteMode {
	return opsWorkflowReportWriteModeForConfirmedMutation(result.Deletion.Submitted)
}

func writeOpsExecuteRetentionPolicyReport(result ops.RetentionPolicyResult, cfg *config.Config, mode OpsWorkflowReportWriteMode) error {
	if result.Request.ReportFile == "" {
		return nil
	}
	report := enrichOpsExecuteRetentionPolicyReport(result.Report, cfg)
	format, err := opsWorkflowReportFormatForPath(result.Request.ReportFile, OpsWorkflowReportFormat(result.Request.ReportFormat))
	if err != nil {
		return err
	}
	var data []byte
	switch format {
	case OpsWorkflowReportFormatJSON:
		data, err = renderOpsExecuteRetentionPolicyJSONReport(report)
	case OpsWorkflowReportFormatMarkdown:
		data, err = renderOpsExecuteRetentionPolicyMarkdownReport(report, cfg)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return writeOpsWorkflowReportFile(result.Request.ReportFile, data, mode)
}

func enrichOpsExecuteRetentionPolicyReport(report ops.RetentionAuditReport, cfg *config.Config) ops.RetentionAuditReport {
	report.C8voltVersion = CurrentBuildInfo().Version
	if cfg != nil {
		report.CamundaVersion = cfg.App.CamundaVersion.String()
		report.TenantID = cfg.App.ViewTenant()
		if cfg.ActiveProfile != "" {
			report.ProfileIdentity = "profile:" + cfg.ActiveProfile
		} else {
			report.ProfileIdentity = "default"
		}
	}
	return report
}
