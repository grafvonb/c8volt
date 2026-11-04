// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsRepairProcessInstanceCommandName = "ops repair process-instance"

var (
	flagOpsRepairProcessInstanceKeys          []string
	flagOpsRepairProcessInstanceRetries       int32
	flagOpsRepairProcessInstanceJobTimeoutRaw string
	flagOpsRepairProcessInstanceVars          string
	flagOpsRepairProcessInstanceVarsFile      string
	flagOpsRepairProcessInstanceReportFile    string
	flagOpsRepairProcessInstanceReportFormat  string
)

var opsRepairProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Repair incidents selected by process instances",
	Long: `Repair active incidents for selected process instances.

Select by key or process filters. The workflow fixes the incident targets,
applies requested variables, restores related job retries, resolves incidents,
and waits for clearance. --set-retries 0 skips retry restoration.

Use --dry-run to review the repair plan and --report-file to save an audit report.
Use --auto-confirm for unattended execution.`,
	Example: `  c8volt ops repair process-instance --key <process-instance-key> --dry-run
  c8volt ops repair process-instance --bpmn-process-id order --set-retries 3 --report-file repair.md`,
	Aliases: []string{"pi", "pis", "process-instances"},
	Args: func(cmd *cobra.Command, args []string) error {
		if searchBusinessIDFlag(cmd) != nil {
			return nil
		}
		if err := validateOptionalDashArg(args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		if len(args) == 1 && args[0] == "-" && hasOpsRepairProcessInstanceSearchModeFlags(cmd) {
			return silenceUsageForError(cmd, mutuallyExclusiveFlagsf("stdin '-' cannot be combined with search filters"))
		}
		return silenceUsageForError(cmd, validateOpsRepairProcessInstanceFlagValues(cmd))
	},
	Run: func(cmd *cobra.Command, args []string) {
		timeout, err := parseOpsRepairProcessInstanceJobTimeout(cmd)
		if err != nil {
			failBeforeCli(cmd, err)
		}
		variables, variablesFile, err := parseOpsRepairVariablesFromFlags(cmd, flagOpsRepairProcessInstanceVars, flagOpsRepairProcessInstanceVarsFile)
		if err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if searchBusinessIDFlag(cmd) != nil {
			if err := validateOptionalDashArg(args); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			if err := validateOpsRepairProcessInstanceFlagValues(cmd); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			if len(args) > 0 {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("stdin '-' cannot be combined with search filters"))
			}
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		reportFormat, err := resolveOpsRepairReportFormat(flagOpsRepairProcessInstanceReportFile, flagOpsRepairProcessInstanceReportFormat)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsRepairProcessInstanceReportFile, OpsWorkflowReportPreserveExisting); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagOpsRepairProcessInstanceKeys, stdinKeys, log, cfg).Unique()
		stdinMode := len(args) == 1 && args[0] == "-"
		keyedMode := len(flagOpsRepairProcessInstanceKeys) > 0 || stdinMode
		explicitSearchMode := hasOpsRepairProcessInstanceSearchModeFlags(cmd)
		if keyedMode && explicitSearchMode {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("--key cannot be combined with process-instance search filters"))
		}
		searchMode := explicitSearchMode || !keyedMode
		if len(keys) == 0 && !searchMode {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process-instance keys provided or found to repair")))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("process-instance key %q is not a valid key", firstBadKey))
		}
		retries := flagOpsRepairProcessInstanceRetries
		mode := ops.RepairDiscoveryModeKeyed
		if searchMode {
			mode = ops.RepairDiscoveryModeSearch
		} else if len(stdinKeys) > 0 && len(flagOpsRepairProcessInstanceKeys) == 0 {
			mode = ops.RepairDiscoveryModeStdin
		}
		selection := populatePISearchFilterOpts(cmd)
		if searchMode {
			selection.HasIncident = new(bool)
			*selection.HasIncident = true
		}
		request := ops.RepairRequest{
			CommandName:              opsRepairProcessInstanceCommandName,
			Target:                   ops.RepairTargetProcessInstance,
			DiscoveryMode:            mode,
			InputKeys:                append(typex.Keys{}, keys...),
			ProcessInstanceSelection: selection,
			DirectIncidentsOnly:      flagGetPIDirectIncidentsOnly,
			BatchSize:                resolveOpsRepairProcessInstanceSearchSize(cmd, cfg),
			Limit:                    flagGetPILimit,
			Workers:                  flagWorkers,
			FailFast:                 flagFailFast,
			NoWorkerLimit:            flagNoWorkerLimit,
			DryRun:                   flagDryRun,
			AutoConfirm:              flagCmdAutoConfirm,
			Automation:               automationModeEnabled(cmd),
			NoWait:                   flagNoWait,
			OutputMode:               pickMode().String(),
			Variables:                variables,
			VariablesFile:            variablesFile,
			RequestedRetries:         &retries,
			RequestedJobTimeout:      timeout,
			ReportFile:               flagOpsRepairProcessInstanceReportFile,
			ReportFormat:             reportFormat,
			StartedAt:                time.Now().UTC(),
		}
		if selection.BusinessId != nil {
			request.IncidentSelection = incident.Filter{State: flagGetPIIncidentState, ErrorType: flagGetPIIncidentErrorType, ErrorMessage: flagGetPIIncidentErrorMessage}
			runOpsRepairProcessInstanceBusinessID(cmd, cli, log, cfg, request)
			return
		}
		initializeOpsTenantContextHumanReporting(cmd, cfg, mode != ops.RepairDiscoveryModeSearch)
		repairProgress := configureOpsRepairProgress(cmd, &request)
		var result ops.RepairResult
		if opsRepairNeedsPreflight(cmd) {
			planRequest := request
			planRequest.DryRun = true
			planned, err := repairProcessInstanceWithCommandActivity(cmd, planRequest, func() (ops.RepairResult, error) {
				return cli.RepairProcessInstances(cmd.Context(), planRequest, collectOptions()...)
			})
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("plan ops repair process-instance: %w", err))
			}
			if opsRepairPlanHasRepairTargets(planned) {
				ctx := attachOpsRepairTenantContext(cmd, cfg, planned)
				printOpsTenantContextForCommand(cmd, ctx)
				if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, opsRepairConfirmationPrompt(planned)); err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
				}
				request = opsRepairConfirmedRequestFromPlan(request, planned)
				result, err = repairProcessInstanceWithCommandActivity(cmd, request, func() (ops.RepairResult, error) {
					return cli.RepairProcessInstances(cmd.Context(), request, collectOptions()...)
				})
				repairProgress.Close()
			} else {
				result = opsRepairResultWithoutMutation(request, planned)
			}
		} else {
			result, err = repairProcessInstanceWithCommandActivity(cmd, request, func() (ops.RepairResult, error) {
				return cli.RepairProcessInstances(cmd.Context(), request, collectOptions()...)
			})
			repairProgress.Close()
		}
		result = attachOpsRepairResultTenantContext(cmd, cfg, result)
		if reportErr := writeOpsRepairReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops repair process-instance: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops repair process-instance audit report: %w", reportErr))
		}
		renderErr := renderOpsRepairProcessInstanceResult(cmd, result)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops repair process-instance: %w", err))
		}
		if renderErr != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops repair process-instance: %w", renderErr))
		}
	},
}

// Register repair inputs and diagnostic replacements without accepting legacy flags.
func init() {
	opsRepairCmd.AddCommand(opsRepairProcessInstanceCmd)
	useInvalidInputFlagErrors(opsRepairProcessInstanceCmd)

	setFlagMigrations(opsRepairProcessInstanceCmd, map[string]flagMigration{
		"retries":     {Replacement: "--set-retries", TakesValue: true},
		"job-timeout": {Replacement: "--set-job-timeout", TakesValue: true},
		"vars":        {Replacement: "--set-vars", TakesValue: true},
		"vars-file":   {Replacement: "--set-vars-file", TakesValue: true},
	})
	fs := opsRepairProcessInstanceCmd.Flags()
	fs.String("bid", "", "exact opaque business ID to select incident-bearing process instances (only c8.10+)")
	fs.StringSliceVarP(&flagOpsRepairProcessInstanceKeys, "key", "k", nil, "process-instance key(s) whose active incidents should be repaired; repeat or combine with stdin '-'")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	registerPISharedDateRangeFlags(fs)
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to filter process instances")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")
	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "select only root process instances")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "select only child process instances")
	fs.BoolVar(&flagGetPIDirectIncidentsOnly, "direct-incidents-only", false, "select only process instances with direct active incidents")
	fs.StringVar(&flagGetPIIncidentState, "incident-state", "active", "incident state scope for --direct-incidents-only: active, pending, resolved, migrated, unknown, all")
	fs.StringVar(&flagGetPIIncidentErrorType, "incident-error-type", "", "case-insensitive incident error type filter for --direct-incidents-only")
	fs.StringVar(&flagGetPIIncidentErrorMessage, "incident-error-message", "", "case-insensitive incident error message substring filter for --direct-incidents-only")
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("process instances per discovery page (server maximum %d)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching process instances to freeze for repair; omit to discover all matches")
	fs.Int32Var(&flagOpsRepairProcessInstanceRetries, "set-retries", 1, "retry count to set on related jobs; 0 skips retry restoration")
	fs.StringVar(&flagOpsRepairProcessInstanceJobTimeoutRaw, "set-job-timeout", "", "timeout duration to submit for related jobs, for example 60s, 5m, or 1h")
	fs.StringVar(&flagOpsRepairProcessInstanceVars, "set-vars", "", "JSON object with variables to set once per process-instance scope before resolving dependent incidents")
	fs.StringVar(&flagOpsRepairProcessInstanceVarsFile, "set-vars-file", "", "path to JSON object file with variables to set once per process-instance scope")
	fs.StringVar(&flagOpsRepairProcessInstanceReportFile, "report-file", "", "plan an audit report at the given path")
	fs.StringVar(&flagOpsRepairProcessInstanceReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")
	fs.BoolVar(&flagDryRun, "dry-run", false, "freeze repair targets and preview repair steps without submitting mutations")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after repair mutations are accepted without incident or retry confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling incident repairs after the first error")

	setCommandMutation(opsRepairProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(opsRepairProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(opsRepairProcessInstanceCmd, AutomationSupportFull, "supports shared machine output, stdin key pipelines, dry-run previews, and unattended process-instance selected repair")
	setOutputModes(opsRepairProcessInstanceCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}

// validateOpsRepairProcessInstanceFlagValues checks local process-instance repair inputs before remote work.
func validateOpsRepairProcessInstanceFlagValues(cmd *cobra.Command) error {
	if err := validatePISearchFlags(cmd); err != nil {
		return err
	}
	if cmd != nil && cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	if flagOpsRepairProcessInstanceRetries < 0 {
		return invalidFlagValuef("invalid value for --set-retries: %d, expected non-negative integer", flagOpsRepairProcessInstanceRetries)
	}
	if cmd != nil && cmd.Flags().Changed("set-vars") && cmd.Flags().Changed("set-vars-file") {
		return mutuallyExclusiveFlagsf("--set-vars cannot be combined with --set-vars-file")
	}
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for ops repair process-instance")
	}
	if err := validateOpsWorkflowReportFlags(flagOpsRepairProcessInstanceReportFile, OpsWorkflowReportFormat(flagOpsRepairProcessInstanceReportFormat)); err != nil {
		return err
	}
	if ok, firstBadKey, _ := validateKeys(flagOpsRepairProcessInstanceKeys); len(flagOpsRepairProcessInstanceKeys) > 0 && !ok {
		return invalidFlagValuef("process-instance key %q is not a valid key", firstBadKey)
	}
	if len(flagOpsRepairProcessInstanceKeys) > 0 && hasOpsRepairProcessInstanceSearchModeFlags(cmd) {
		return mutuallyExclusiveFlagsf("--key cannot be combined with process-instance search filters")
	}
	for flag, value := range map[string]string{
		"--parent-key": flagGetPIParentKey,
		"--pd-key":     flagGetPIProcessDefinitionKey,
	} {
		if value == "" {
			continue
		}
		if ok, firstBadKey, _ := validateKeys([]string{value}); !ok {
			return invalidFlagValuef("%s value %q is not a valid key", flag, firstBadKey)
		}
	}
	return nil
}

// parseOpsRepairProcessInstanceJobTimeout validates the optional related-job timeout request.
func parseOpsRepairProcessInstanceJobTimeout(cmd *cobra.Command) (time.Duration, error) {
	if cmd == nil || !cmd.Flags().Changed("set-job-timeout") {
		return 0, nil
	}
	timeout, err := time.ParseDuration(flagOpsRepairProcessInstanceJobTimeoutRaw)
	if err != nil || timeout <= 0 {
		return 0, invalidFlagValuef("invalid value for --set-job-timeout: %q, expected positive duration such as 60s, 5m, or 1h", flagOpsRepairProcessInstanceJobTimeoutRaw)
	}
	if timeout.Milliseconds() <= 0 {
		return 0, invalidFlagValuef("invalid value for --set-job-timeout: %q, duration must be at least 1ms", flagOpsRepairProcessInstanceJobTimeoutRaw)
	}
	return timeout, nil
}

func repairProcessInstanceWithCommandActivity(cmd *cobra.Command, request ops.RepairRequest, run func() (ops.RepairResult, error)) (ops.RepairResult, error) {
	stopActivity := startCommandActivity(cmd, formatOpsRepairProcessInstanceActivity(request))
	defer stopActivity()
	return run()
}

func formatOpsRepairProcessInstanceActivity(request ops.RepairRequest) string {
	if request.DryRun {
		if request.DiscoveryMode == ops.RepairDiscoveryModeSearch {
			return "discovering process-instance repair targets"
		}
		return "planning process-instance incident repair"
	}
	return "repairing process-instance incidents"
}

// hasOpsRepairProcessInstanceSearchModeFlags detects explicit process-instance search mode without treating default state as a selector.
func hasOpsRepairProcessInstanceSearchModeFlags(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for _, name := range []string{
		"bid",
		"bpmn-process-id",
		"pd-version",
		"pd-version-tag",
		"pd-key",
		"parent-key",
		"state",
		"roots-only",
		"children-only",
		"direct-incidents-only",
		"incident-state",
		"incident-error-type",
		"incident-error-message",
		"start-date-after",
		"start-date-before",
		"start-date-older-days",
		"start-date-newer-days",
		"end-date-after",
		"end-date-before",
		"end-date-older-days",
		"end-date-newer-days",
		"batch-size",
		"limit",
	} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

// resolveOpsRepairProcessInstanceSearchSize applies the existing process-instance page-size default policy.
func resolveOpsRepairProcessInstanceSearchSize(cmd *cobra.Command, cfg *config.Config) int32 {
	if cmd != nil && cmd.Flags().Changed("batch-size") {
		return flagGetPISize
	}
	if cfg != nil && cfg.App.ProcessInstancePageSize > 0 && cfg.App.ProcessInstancePageSize <= consts.MaxPISearchSize {
		return cfg.App.ProcessInstancePageSize
	}
	return consts.MaxPISearchSize
}
