// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsPurgeAllProcessDefinitionsCommandName = "ops purge all-process-definitions"

var (
	flagOpsPurgeAllPDKey               string
	flagOpsPurgeAllPDBpmnProcessID     string
	flagOpsPurgeAllPDProcessVersion    int32
	flagOpsPurgeAllPDProcessVersionTag string
	flagOpsPurgeAllPDLatest            bool
	flagOpsPurgeAllPDReportFile        string
	flagOpsPurgeAllPDReportFormat      string
)

var opsPurgeAllProcessDefinitionsCmd = &cobra.Command{
	Use:   "all-process-definitions",
	Short: "Purge all selected process definitions",
	Long: "Purge all selected process definitions.\n\n" +
		"The workflow discovers candidate process-definition versions using the same filters as `get pd`, freezes the candidate keys, validates the existing delete plan, and then either reports the plan with --dry-run or submits deletion only after confirmation. Use --auto-confirm or --automation for unattended deletion, combine --automation with --json for deterministic machine output, and use --report-file to write an audit report.",
	Example: `  ./c8volt ops purge all-process-definitions --dry-run
  ./c8volt ops purge all-pds --bpmn-process-id invoice --latest --dry-run
  ./c8volt ops purge all-process-definitions --automation --json --dry-run
  ./c8volt ops purge all-process-definitions --dry-run --report-file process-definition-purge.md
  ./c8volt ops purge all-process-definitions --bpmn-process-id invoice --latest --auto-confirm --force
  ./c8volt ops purge all-process-definitions --bpmn-process-id invoice --latest --auto-confirm --force --workers 4 --report-file process-definition-purge.json --report-format json`,
	Aliases: []string{"all-pds"},
	Args:    validateOpsPurgeAllProcessDefinitionsArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		effectiveAutoConfirm := shouldImplicitlyConfirm(cmd)
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsPurgeAllPDReportFile, opsPurgeAllProcessDefinitionsPlanningReportWriteMode(effectiveAutoConfirm)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		request := ops.AllProcessDefinitionsPurgeRequest{
			CommandName:   opsPurgeAllProcessDefinitionsCommandName,
			DryRun:        flagDryRun,
			AutoConfirm:   flagCmdAutoConfirm,
			Automation:    automationModeEnabled(cmd),
			OutputMode:    pickMode().String(),
			Selection:     populateOpsPurgeAllProcessDefinitionsSelection(),
			Workers:       flagWorkers,
			FailFast:      flagFailFast,
			NoWorkerLimit: flagNoWorkerLimit,
			NoWait:        flagNoWait,
			Force:         flagForce,
			ReportFile:    flagOpsPurgeAllPDReportFile,
			ReportFormat:  flagOpsPurgeAllPDReportFormat,
			StartedAt:     time.Now().UTC(),
		}
		if !flagDryRun && !effectiveAutoConfirm {
			planRequest := request
			planRequest.DryRun = true
			planned, err := cli.PurgeAllProcessDefinitions(cmd.Context(), planRequest, collectOptions()...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan ops purge all process definitions: %w", err))
			}
			if err := rejectOpsPurgeAllProcessDefinitionsPlanRequiringForce(planned.DeletePlan); err != nil {
				abortOpsPurgeAllProcessDefinitionsAfterReport(cmd, log, cfg, markOpsPurgeAllProcessDefinitionsLocalFailure(planned, ops.WorkflowStepStatusBlocked, err), err)
				return
			}
			if len(planned.DeletePlan.CandidateProcessDefinitionKeys) > 0 {
				prompt := fmt.Sprintf("All process-definitions purge matched %d candidate process definition(s); delete planning will affect %d process instance(s) across %d unique process definition(s). Do you want to proceed?",
					planned.Discovery.CandidateProcessDefinitionCount,
					planned.DeletePlan.AffectedProcessInstanceCount,
					len(planned.DeletePlan.CandidateProcessDefinitionKeys),
				)
				if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
					abortOpsPurgeAllProcessDefinitionsAfterReport(cmd, log, cfg, markOpsPurgeAllProcessDefinitionsLocalFailure(planned, ops.WorkflowStepStatusConfirmationFailed, err), err)
					return
				}
			}
			request.DiscoveredCandidateProcessDefinitionKeys = append(typex.Keys{}, planned.Discovery.CandidateProcessDefinitionKeys...)
		}
		result, err := cli.PurgeAllProcessDefinitions(cmd.Context(), request, collectOptions()...)
		if err != nil {
			if reportErr := writeOpsPurgeAllProcessDefinitionsReport(result, cfg, opsPurgeAllProcessDefinitionsReportWriteMode(result)); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge all process definitions: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops purge all process definitions: %w", err))
		}
		if err := writeOpsPurgeAllProcessDefinitionsReport(result, cfg, opsPurgeAllProcessDefinitionsReportWriteMode(result)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops purge all process definitions audit report: %w", err))
		}
		if err := renderOpsPurgeAllProcessDefinitionsResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops purge all process definitions: %w", err))
		}
	},
}

func init() {
	opsPurgeCmd.AddCommand(opsPurgeAllProcessDefinitionsCmd)
	useInvalidInputFlagErrors(opsPurgeAllProcessDefinitionsCmd)

	fs := opsPurgeAllProcessDefinitionsCmd.Flags()
	fs.StringVarP(&flagOpsPurgeAllPDKey, "key", "k", "", "process definition key to select for candidate discovery")
	fs.StringVarP(&flagOpsPurgeAllPDBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter candidate process definitions")
	fs.Int32Var(&flagOpsPurgeAllPDProcessVersion, "pd-version", 0, "process definition version to filter candidate discovery")
	fs.StringVar(&flagOpsPurgeAllPDProcessVersionTag, "pd-version-tag", "", "process definition version tag to filter candidate discovery")
	fs.BoolVar(&flagOpsPurgeAllPDLatest, "latest", false, "only include the latest matching process-definition version(s)")
	fs.BoolVar(&flagDryRun, "dry-run", false, "discover and validate process-definition cleanup without submitting deletion requests")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when validating the delete plan and deleting process definitions (default: min(targets, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation or deletion work after the first error")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of affected active process instances before deleting process definitions")
	fs.StringVar(&flagOpsPurgeAllPDReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsPurgeAllPDReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsPurgeAllProcessDefinitionsCmd, CommandMutationStateChanging)
	setContractSupport(opsPurgeAllProcessDefinitionsCmd, ContractSupportFull)
	setAutomationSupport(opsPurgeAllProcessDefinitionsCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed all-process-definitions purges with shared machine output")
	setOutputModes(opsPurgeAllProcessDefinitionsCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}

func validateOpsPurgeAllProcessDefinitionsArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return silenceUsageForError(cmd, err)
	}
	return silenceUsageForError(cmd, validateOpsPurgeAllProcessDefinitionsFlags(cmd))
}

// validateOpsPurgeAllProcessDefinitionsFlags keeps static command-shape failures local.
func validateOpsPurgeAllProcessDefinitionsFlags(cmd *cobra.Command) error {
	if flagOpsPurgeAllPDKey != "" {
		if ok, firstBadKey, _ := validateKeys([]string{flagOpsPurgeAllPDKey}); !ok {
			return invalidFlagValuef("process definition key %q is not a valid key", firstBadKey)
		}
	}
	if cmd != nil && cmd.Flags().Changed("pd-version") && flagOpsPurgeAllPDProcessVersion <= 0 {
		return invalidFlagValuef("--pd-version must be positive integer")
	}
	if cmd != nil && cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	return validateOpsWorkflowReportFlags(flagOpsPurgeAllPDReportFile, OpsWorkflowReportFormat(flagOpsPurgeAllPDReportFormat))
}

// rejectOpsPurgeAllProcessDefinitionsPlanRequiringForce blocks mutation before prompting when active process instances are affected.
func rejectOpsPurgeAllProcessDefinitionsPlanRequiringForce(plan ops.AllProcessDefinitionsPurgeDeletePlan) error {
	if flagForce || !plan.RequiresForce {
		return nil
	}
	return localPreconditionError(fmt.Errorf(
		"refusing to delete all-process-definitions purge scope: %d active process instance(s) are affected; no delete request was submitted; use --force to cancel active process instances before delete",
		plan.ActiveProcessInstanceCount,
	))
}

// abortOpsPurgeAllProcessDefinitionsAfterReport writes available audit data before surfacing local aborts.
func abortOpsPurgeAllProcessDefinitionsAfterReport(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, result ops.AllProcessDefinitionsPurgeResult, err error) {
	if reportErr := writeOpsPurgeAllProcessDefinitionsReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w; write audit report: %v", err, reportErr))
	}
	handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
}

// markOpsPurgeAllProcessDefinitionsLocalFailure records local blockers in the audit report shape.
func markOpsPurgeAllProcessDefinitionsLocalFailure(result ops.AllProcessDefinitionsPurgeResult, status ops.WorkflowStepStatus, err error) ops.AllProcessDefinitionsPurgeResult {
	finished := time.Now().UTC()
	msg := err.Error()
	result.Outcome = ops.AllProcessDefinitionsPurgeOutcomeFailed
	result.Errors = appendOpsPurgeAllProcessDefinitionsError(result.Errors, msg)
	result.Deletion.Status = status
	result.Deletion.Errors = appendOpsPurgeAllProcessDefinitionsError(result.Deletion.Errors, msg)
	result.Report.Outcome = ops.AllProcessDefinitionsPurgeOutcomeFailed
	result.Report.FinishedAt = finished
	if !result.Request.StartedAt.IsZero() {
		result.Report.Duration = finished.Sub(result.Request.StartedAt).String()
	}
	result.Report.Discovery = result.Discovery
	result.Report.DeletePlan = result.DeletePlan
	result.Report.Deletion = result.Deletion
	result.Report.Errors = appendOpsPurgeAllProcessDefinitionsError(result.Report.Errors, msg)
	return result
}

// appendOpsPurgeAllProcessDefinitionsError preserves stable error order while avoiding duplicates.
func appendOpsPurgeAllProcessDefinitionsError(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

// opsPurgeAllProcessDefinitionsPlanningReportWriteMode preserves reports unless this run is already confirmed for mutation.
func opsPurgeAllProcessDefinitionsPlanningReportWriteMode(effectiveAutoConfirm bool) OpsWorkflowReportWriteMode {
	if flagDryRun {
		return OpsWorkflowReportPreserveExisting
	}
	return opsWorkflowReportWriteModeForConfirmedMutation(effectiveAutoConfirm)
}

// opsPurgeAllProcessDefinitionsReportWriteMode overwrites only after deletion was actually submitted.
func opsPurgeAllProcessDefinitionsReportWriteMode(result ops.AllProcessDefinitionsPurgeResult) OpsWorkflowReportWriteMode {
	return opsWorkflowReportWriteModeForConfirmedMutation(result.Deletion.Submitted)
}

// writeOpsPurgeAllProcessDefinitionsReport renders and writes the requested audit report.
func writeOpsPurgeAllProcessDefinitionsReport(result ops.AllProcessDefinitionsPurgeResult, cfg *config.Config, mode OpsWorkflowReportWriteMode) error {
	if result.Request.ReportFile == "" {
		return nil
	}
	report := enrichOpsPurgeAllProcessDefinitionsReport(result.Report, cfg)
	format, err := opsWorkflowReportFormatForPath(result.Request.ReportFile, OpsWorkflowReportFormat(result.Request.ReportFormat))
	if err != nil {
		return err
	}
	var data []byte
	switch format {
	case OpsWorkflowReportFormatJSON:
		data, err = renderOpsPurgeAllProcessDefinitionsJSONReport(report)
	case OpsWorkflowReportFormatMarkdown:
		data, err = renderOpsPurgeAllProcessDefinitionsMarkdownReport(report, cfg)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return writeOpsWorkflowReportFile(result.Request.ReportFile, data, mode)
}

// enrichOpsPurgeAllProcessDefinitionsReport adds runtime config metadata that is not owned by services.
func enrichOpsPurgeAllProcessDefinitionsReport(report ops.AllProcessDefinitionsPurgeReport, cfg *config.Config) ops.AllProcessDefinitionsPurgeReport {
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

func populateOpsPurgeAllProcessDefinitionsSelection() ops.ProcessDefinitionSelection {
	return ops.ProcessDefinitionSelection{
		Key:               flagOpsPurgeAllPDKey,
		BpmnProcessId:     flagOpsPurgeAllPDBpmnProcessID,
		ProcessVersion:    flagOpsPurgeAllPDProcessVersion,
		ProcessVersionTag: flagOpsPurgeAllPDProcessVersionTag,
		LatestOnly:        flagOpsPurgeAllPDLatest,
	}
}
