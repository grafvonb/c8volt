// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsPurgeProcessInstancesWithIncidentsCommandName = "ops purge process-instances-with-incidents"

var (
	flagOpsPurgeIncidentKeys               []string
	flagOpsPurgeIncidentState              string
	flagOpsPurgeIncidentErrorType          string
	flagOpsPurgeIncidentErrorMessage       string
	flagOpsPurgeIncidentBpmnProcessID      string
	flagOpsPurgeIncidentPDKey              string
	flagOpsPurgeIncidentPIKey              string
	flagOpsPurgeIncidentRootKey            string
	flagOpsPurgeIncidentFlowNodeID         string
	flagOpsPurgeIncidentFNIKey             string
	flagOpsPurgeIncidentCreationTimeAfter  string
	flagOpsPurgeIncidentCreationTimeBefore string
	flagOpsPurgeIncidentBatchSize          int32
	flagOpsPurgeIncidentLimit              int32
	flagOpsPurgeIncidentReportFile         string
	flagOpsPurgeIncidentReportFormat       string
)

var opsPurgeProcessInstancesWithIncidentsCmd = &cobra.Command{
	Use:   "process-instances-with-incidents",
	Short: "Purge process instances selected by incidents",
	Long: "Purge process instances selected by incidents.\n\n" +
		"The workflow discovers candidate incidents from incident filters, freezes the candidate process-instance keys, validates the delete plan, and then either reports the plan with --dry-run or submits deletion only after confirmation. Use --auto-confirm or --automation for unattended deletion, combine --automation with --json for deterministic machine output, and use --report-file to write an audit report.",
	Example: `  ./c8volt ops purge process-instances-with-incidents --dry-run
  ./c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --dry-run
  ./c8volt ops purge process-instances-with-incidents --state active --limit 5 --dry-run
  ./c8volt ops purge pi-with-incidents --state active --dry-run
  ./c8volt ops purge process-instances-with-incidents --automation --json --dry-run
  ./c8volt ops purge process-instances-with-incidents --dry-run --report-file incident-purge.md
  ./c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --limit 5 --auto-confirm --force
  ./c8volt ops purge process-instances-with-incidents --state active --error-type io_mapping_error --limit 5 --auto-confirm --force --workers 4 --report-file incident-purge.json --report-format json`,
	Aliases: []string{"pi-with-incidents", "piwi"},
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateOpsPurgeProcessInstancesWithIncidentsFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		effectiveAutoConfirm := shouldImplicitlyConfirm(cmd)
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsPurgeIncidentReportFile, opsPurgeProcessInstancesWithIncidentsPlanningReportWriteMode(effectiveAutoConfirm)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		request := ops.IncidentPurgeRequest{
			CommandName:   opsPurgeProcessInstancesWithIncidentsCommandName,
			DryRun:        flagDryRun,
			AutoConfirm:   flagCmdAutoConfirm,
			Automation:    automationModeEnabled(cmd),
			OutputMode:    pickMode().String(),
			Selection:     populateOpsPurgeIncidentSelection(),
			BatchSize:     resolveOpsPurgeIncidentSearchSize(cmd, cfg),
			Limit:         flagOpsPurgeIncidentLimit,
			Workers:       flagWorkers,
			FailFast:      flagFailFast,
			NoWorkerLimit: flagNoWorkerLimit,
			NoWait:        flagNoWait,
			Force:         flagForce,
			ReportFile:    flagOpsPurgeIncidentReportFile,
			ReportFormat:  flagOpsPurgeIncidentReportFormat,
			StartedAt:     time.Now().UTC(),
		}
		if !flagDryRun && !effectiveAutoConfirm {
			planRequest := request
			planRequest.DryRun = true
			planned, err := cli.PurgeProcessInstancesWithIncidents(cmd.Context(), planRequest, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan ops purge process-instances with incidents: %w", err))
			}
			if err := rejectOpsPurgeProcessInstancesWithIncidentsPlanRequiringForce(planned.DeletePlan); err != nil {
				abortOpsPurgeProcessInstancesWithIncidentsAfterReport(cmd, log, cfg, markOpsPurgeProcessInstancesWithIncidentsLocalFailure(planned, ops.WorkflowStepStatusBlocked, err), err)
				return
			}
			if len(planned.DeletePlan.ResolvedRootKeys) > 0 {
				prompt := fmt.Sprintf("Incident purge matched %d candidate incident(s) and %d candidate process instance(s); delete planning will delete %d affected process instance(s) across %d root(s). Do you want to proceed?",
					planned.Discovery.IncidentCount,
					len(planned.DeletePlan.CandidateProcessInstanceKeys),
					len(planned.DeletePlan.AffectedKeys),
					len(planned.DeletePlan.ResolvedRootKeys),
				)
				if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
					abortOpsPurgeProcessInstancesWithIncidentsAfterReport(cmd, log, cfg, markOpsPurgeProcessInstancesWithIncidentsLocalFailure(planned, ops.WorkflowStepStatusConfirmationFailed, err), err)
					return
				}
			}
			request.DiscoveredCandidateProcessInstanceKeys = append(typex.Keys{}, planned.Discovery.CandidateProcessInstanceKeys...)
		}
		result, err := cli.PurgeProcessInstancesWithIncidents(cmd.Context(), request, collectOptions()...)
		if err != nil {
			if reportErr := writeOpsPurgeProcessInstancesWithIncidentsReport(result, cfg, opsPurgeProcessInstancesWithIncidentsReportWriteMode(result)); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge process-instances with incidents: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge process-instances with incidents: %w", err))
		}
		if err := writeOpsPurgeProcessInstancesWithIncidentsReport(result, cfg, opsPurgeProcessInstancesWithIncidentsReportWriteMode(result)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops purge process-instances with incidents audit report: %w", err))
		}
		if err := renderOpsPurgeProcessInstancesWithIncidentsResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops purge process-instances with incidents: %w", err))
		}
	},
}

func init() {
	opsPurgeCmd.AddCommand(opsPurgeProcessInstancesWithIncidentsCmd)
	useInvalidInputFlagErrors(opsPurgeProcessInstancesWithIncidentsCmd)

	fs := opsPurgeProcessInstancesWithIncidentsCmd.Flags()
	fs.StringSliceVarP(&flagOpsPurgeIncidentKeys, "key", "k", nil, "incident key(s) to select for candidate discovery")
	fs.StringVarP(&flagOpsPurgeIncidentState, "state", "s", "active", "incident state scope for discovery: active, pending, resolved, migrated, unknown, all")
	fs.StringVar(&flagOpsPurgeIncidentErrorType, "error-type", "", "case-insensitive incident error type filter for discovery")
	fs.StringVar(&flagOpsPurgeIncidentErrorMessage, "error-message", "", "case-insensitive incident error message substring filter for discovery")
	fs.StringVarP(&flagOpsPurgeIncidentBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter incidents")
	fs.StringVar(&flagOpsPurgeIncidentPDKey, "pd-key", "", "process definition key to filter incidents")
	fs.StringVar(&flagOpsPurgeIncidentPIKey, "pi-key", "", "process instance key to filter incidents")
	fs.StringVar(&flagOpsPurgeIncidentRootKey, "root-key", "", "root process instance key to filter incidents")
	fs.StringVar(&flagOpsPurgeIncidentFlowNodeID, "flow-node-id", "", "flow node ID to filter incidents")
	fs.StringVar(&flagOpsPurgeIncidentFNIKey, "fni-key", "", "flow node instance key to filter incidents")
	fs.StringVar(&flagOpsPurgeIncidentCreationTimeAfter, "creation-time-after", "", "only include incidents with creation time >= RFC3339 timestamp or YYYY-MM-DD")
	fs.StringVar(&flagOpsPurgeIncidentCreationTimeBefore, "creation-time-before", "", "only include incidents with creation time <= RFC3339 timestamp or YYYY-MM-DD")
	fs.Int32VarP(&flagOpsPurgeIncidentBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of incidents to inspect per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagOpsPurgeIncidentLimit, "limit", "l", 0, "maximum number of matching incidents to inspect before candidate process-instance dedupe")
	fs.BoolVar(&flagDryRun, "dry-run", false, "discover and validate incident-based process-instance cleanup without submitting deletion requests")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when validating the delete plan and deleting roots (default: min(targets, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation or deletion work after the first error")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")
	fs.StringVar(&flagOpsPurgeIncidentReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsPurgeIncidentReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsPurgeProcessInstancesWithIncidentsCmd, CommandMutationStateChanging)
	setContractSupport(opsPurgeProcessInstancesWithIncidentsCmd, ContractSupportFull)
	setAutomationSupport(opsPurgeProcessInstancesWithIncidentsCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed incident-based purges with shared machine output")
	setOutputModes(opsPurgeProcessInstancesWithIncidentsCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}

// validateOpsPurgeProcessInstancesWithIncidentsFlags keeps static command-shape failures local.
func validateOpsPurgeProcessInstancesWithIncidentsFlags(cmd *cobra.Command) error {
	if flagOpsPurgeIncidentBatchSize <= 0 || flagOpsPurgeIncidentBatchSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagOpsPurgeIncidentBatchSize, consts.MaxPISearchSize)
	}
	if flagOpsPurgeIncidentLimit < 0 || (flagOpsPurgeIncidentLimit == 0 && cmd != nil && cmd.Flags().Changed("limit")) {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if cmd != nil && cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	if err := validateGetIncidentStateFlag(flagOpsPurgeIncidentState); err != nil {
		return err
	}
	if err := validateGetIncidentErrorTypeFlag(flagOpsPurgeIncidentErrorType); err != nil {
		return err
	}
	if err := validateGetIncidentCreationTimeFlag("--creation-time-after", flagOpsPurgeIncidentCreationTimeAfter); err != nil {
		return err
	}
	if err := validateGetIncidentCreationTimeFlag("--creation-time-before", flagOpsPurgeIncidentCreationTimeBefore); err != nil {
		return err
	}
	for flag, value := range map[string]string{
		"--pi-key":   flagOpsPurgeIncidentPIKey,
		"--root-key": flagOpsPurgeIncidentRootKey,
		"--pd-key":   flagOpsPurgeIncidentPDKey,
		"--fni-key":  flagOpsPurgeIncidentFNIKey,
	} {
		if value == "" {
			continue
		}
		if ok, firstBadKey, _ := validateKeys([]string{value}); !ok {
			return invalidFlagValuef("%s value %q is not a valid key", flag, firstBadKey)
		}
	}
	if ok, firstBadKey, _ := validateKeys(flagOpsPurgeIncidentKeys); !ok {
		return invalidFlagValuef("incident key %q is not a valid key", firstBadKey)
	}
	return validateOpsWorkflowReportFlags(flagOpsPurgeIncidentReportFile, OpsWorkflowReportFormat(flagOpsPurgeIncidentReportFormat))
}

// rejectOpsPurgeProcessInstancesWithIncidentsPlanRequiringForce blocks mutation before prompting when the plan has non-final affected instances.
func rejectOpsPurgeProcessInstancesWithIncidentsPlanRequiringForce(plan ops.IncidentPurgeDeletePlan) error {
	if flagForce || len(plan.NonFinalAffectedItems) == 0 {
		return nil
	}
	return localPreconditionError(fmt.Errorf(
		"refusing to delete incident purge process-instance scope: %s; no delete request was submitted; use --force to cancel the non-final affected scope before delete",
		formatOpsPurgeProcessInstancesWithIncidentsNonFinalScope(plan),
	))
}

// abortOpsPurgeProcessInstancesWithIncidentsAfterReport writes any available audit data before surfacing local aborts.
func abortOpsPurgeProcessInstancesWithIncidentsAfterReport(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, result ops.IncidentPurgeResult, err error) {
	if reportErr := writeOpsPurgeProcessInstancesWithIncidentsReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w; write audit report: %v", err, reportErr))
	}
	handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
}

// markOpsPurgeProcessInstancesWithIncidentsLocalFailure records local blockers in the same report shape as service failures.
func markOpsPurgeProcessInstancesWithIncidentsLocalFailure(result ops.IncidentPurgeResult, status ops.WorkflowStepStatus, err error) ops.IncidentPurgeResult {
	finished := time.Now().UTC()
	msg := err.Error()
	result.Outcome = ops.IncidentPurgeOutcomeFailed
	result.Errors = appendOpsPurgeProcessInstancesWithIncidentsError(result.Errors, msg)
	result.Deletion.Status = status
	result.Deletion.Errors = appendOpsPurgeProcessInstancesWithIncidentsError(result.Deletion.Errors, msg)
	result.Report.Outcome = ops.IncidentPurgeOutcomeFailed
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Discovery = result.Discovery
	result.Report.DeletePlan = result.DeletePlan
	result.Report.Deletion = result.Deletion
	result.Report.Errors = appendOpsPurgeProcessInstancesWithIncidentsError(result.Report.Errors, msg)
	return result
}

// appendOpsPurgeProcessInstancesWithIncidentsError preserves stable error order while avoiding duplicates.
func appendOpsPurgeProcessInstancesWithIncidentsError(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

// opsPurgeProcessInstancesWithIncidentsPlanningReportWriteMode preserves reports unless this run is already confirmed for mutation.
func opsPurgeProcessInstancesWithIncidentsPlanningReportWriteMode(effectiveAutoConfirm bool) OpsWorkflowReportWriteMode {
	if flagDryRun {
		return OpsWorkflowReportPreserveExisting
	}
	return opsWorkflowReportWriteModeForConfirmedMutation(effectiveAutoConfirm)
}

// opsPurgeProcessInstancesWithIncidentsReportWriteMode overwrites only after deletion was actually submitted.
func opsPurgeProcessInstancesWithIncidentsReportWriteMode(result ops.IncidentPurgeResult) OpsWorkflowReportWriteMode {
	return opsWorkflowReportWriteModeForConfirmedMutation(result.Deletion.Submitted)
}

// writeOpsPurgeProcessInstancesWithIncidentsReport renders and writes the requested incident-purge audit report.
func writeOpsPurgeProcessInstancesWithIncidentsReport(result ops.IncidentPurgeResult, cfg *config.Config, mode OpsWorkflowReportWriteMode) error {
	if result.Request.ReportFile == "" {
		return nil
	}
	report := enrichOpsPurgeProcessInstancesWithIncidentsReport(result.Report, cfg)
	format, err := opsWorkflowReportFormatForPath(result.Request.ReportFile, OpsWorkflowReportFormat(result.Request.ReportFormat))
	if err != nil {
		return err
	}
	var data []byte
	switch format {
	case OpsWorkflowReportFormatJSON:
		data, err = renderOpsPurgeProcessInstancesWithIncidentsJSONReport(report)
	case OpsWorkflowReportFormatMarkdown:
		data, err = renderOpsPurgeProcessInstancesWithIncidentsMarkdownReport(report, cfg)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return writeOpsWorkflowReportFile(result.Request.ReportFile, data, mode)
}

// enrichOpsPurgeProcessInstancesWithIncidentsReport adds runtime config metadata that is not owned by services.
func enrichOpsPurgeProcessInstancesWithIncidentsReport(report ops.IncidentPurgeReport, cfg *config.Config) ops.IncidentPurgeReport {
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

// formatOpsPurgeProcessInstancesWithIncidentsNonFinalScope summarizes the post-planning blocker without listing keys by default.
func formatOpsPurgeProcessInstancesWithIncidentsNonFinalScope(plan ops.IncidentPurgeDeletePlan) string {
	items := newProcessInstanceDryRunRequiresCancelBeforeDelete(plan.NonFinalAffectedItems)
	details := []string{
		fmt.Sprintf("incident purge matched %d candidate process instance(s)", len(plan.CandidateProcessInstanceKeys)),
		fmt.Sprintf("delete planning expanded to %d affected process instance(s) across %d root(s)", len(plan.AffectedKeys), len(plan.ResolvedRootKeys)),
		fmt.Sprintf("%d affected process instance(s) are not in a final state", len(plan.NonFinalAffectedItems)),
		fmt.Sprintf("states: %s", formatProcessInstanceDryRunRequiresCancelBeforeDeleteStates(items)),
	}
	if flagVerbose {
		details = append(details, formatProcessInstanceDryRunRequiresCancelBeforeDelete(items))
	} else {
		details = append(details, "use --verbose to list keys")
	}
	return strings.Join(details, "; ")
}

// populateOpsPurgeIncidentSelection converts command flags into the public incident filter model.
func populateOpsPurgeIncidentSelection() incident.Filter {
	errorType, _ := incidentfilter.NormalizeErrorType(flagOpsPurgeIncidentErrorType)
	return incident.Filter{
		Keys:                   append([]string(nil), flagOpsPurgeIncidentKeys...),
		State:                  flagOpsPurgeIncidentState,
		ErrorType:              errorType,
		ErrorMessage:           flagOpsPurgeIncidentErrorMessage,
		ProcessDefinitionId:    flagOpsPurgeIncidentBpmnProcessID,
		ProcessDefinitionKey:   flagOpsPurgeIncidentPDKey,
		ProcessInstanceKey:     flagOpsPurgeIncidentPIKey,
		RootProcessInstanceKey: flagOpsPurgeIncidentRootKey,
		FlowNodeId:             flagOpsPurgeIncidentFlowNodeID,
		FlowNodeInstanceKey:    flagOpsPurgeIncidentFNIKey,
		CreationTimeAfter:      flagOpsPurgeIncidentCreationTimeAfter,
		CreationTimeBefore:     flagOpsPurgeIncidentCreationTimeBefore,
	}
}

// resolveOpsPurgeIncidentSearchSize applies the existing incident page-size default policy.
func resolveOpsPurgeIncidentSearchSize(cmd *cobra.Command, cfg *config.Config) int32 {
	if cmd != nil && cmd.Flags().Changed("batch-size") {
		return pickOpsPurgeIncidentSearchSize()
	}
	if cfg != nil && cfg.App.ProcessInstancePageSize > 0 && cfg.App.ProcessInstancePageSize <= consts.MaxPISearchSize {
		return cfg.App.ProcessInstancePageSize
	}
	return consts.MaxPISearchSize
}

// pickOpsPurgeIncidentSearchSize returns a safe search size after validation has run.
func pickOpsPurgeIncidentSearchSize() int32 {
	if flagOpsPurgeIncidentBatchSize <= 0 || flagOpsPurgeIncidentBatchSize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return flagOpsPurgeIncidentBatchSize
}
