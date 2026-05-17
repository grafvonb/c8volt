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
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsRepairIncidentCommandName = "ops repair incident"

var (
	flagOpsRepairIncidentKeys               []string
	flagOpsRepairIncidentState              string
	flagOpsRepairIncidentErrorType          string
	flagOpsRepairIncidentErrorMessage       string
	flagOpsRepairIncidentPIKey              string
	flagOpsRepairIncidentRootKey            string
	flagOpsRepairIncidentPDKey              string
	flagOpsRepairIncidentBpmnProcessID      string
	flagOpsRepairIncidentFlowNodeID         string
	flagOpsRepairIncidentFNIKey             string
	flagOpsRepairIncidentCreationTimeAfter  string
	flagOpsRepairIncidentCreationTimeBefore string
	flagOpsRepairIncidentBatchSize          int32
	flagOpsRepairIncidentLimit              int32
	flagOpsRepairIncidentRetries            int32
	flagOpsRepairIncidentJobTimeoutRaw      string
	flagOpsRepairIncidentVars               string
	flagOpsRepairIncidentVarsFile           string
	flagOpsRepairIncidentReportFile         string
	flagOpsRepairIncidentReportFormat       string
)

var opsRepairIncidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Repair incidents by key or filter",
	Long: "Repair incidents by key or filter.\n\n" +
		"The command accepts repeated --key values, newline-separated keys from stdin with '-', or incident search filters. Keyed mode and search mode are mutually exclusive. It freezes the requested incident set before mutation, applies job retry and timeout updates only when an incident has a related job, resolves each incident, and confirms clearance unless --no-wait is set. Incidents without related jobs report job steps as not_applicable and still proceed to incident resolution.",
	Example: `  ./c8volt ops repair incident --key <incident-key>
  ./c8volt ops repair inc --key <incident-key> --key <another-incident-key>
  printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | ./c8volt ops repair incident -
  ./c8volt ops repair incident --state active --error-type io_mapping_error --limit 25 --dry-run
  ./c8volt ops repair incident --key <incident-key> --retries 0
  ./c8volt ops repair incident --key <incident-key> --job-timeout 5m
  ./c8volt ops repair incident --key <incident-key> --dry-run
  ./c8volt --json ops repair incident --key <incident-key> --automation`,
	Aliases: []string{"inc"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateOptionalDashArg(args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		if len(args) == 1 && args[0] == "-" && hasOpsRepairIncidentSearchModeFlags(cmd) {
			return silenceUsageForError(cmd, mutuallyExclusiveFlagsf("stdin '-' cannot be combined with search filters"))
		}
		return silenceUsageForError(cmd, validateOpsRepairIncidentFlagValues(cmd))
	},
	Run: func(cmd *cobra.Command, args []string) {
		timeout, err := parseOpsRepairIncidentJobTimeout(cmd)
		if err != nil {
			failBeforeCli(cmd, err)
		}
		variables, variablesFile, err := parseOpsRepairVariablesFromFlags(cmd, flagOpsRepairIncidentVars, flagOpsRepairIncidentVarsFile)
		if err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		reportFormat, err := resolveOpsRepairReportFormat(flagOpsRepairIncidentReportFile, flagOpsRepairIncidentReportFormat)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsRepairIncidentReportFile, OpsWorkflowReportPreserveExisting); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagOpsRepairIncidentKeys, stdinKeys, log, cfg).Unique()
		searchMode := hasOpsRepairIncidentSearchModeFlags(cmd)
		keyedMode := len(flagOpsRepairIncidentKeys) > 0 || len(stdinKeys) > 0
		if keyedMode && searchMode {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("--key cannot be combined with search filters"))
		}
		if len(keys) == 0 && !searchMode {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no incident keys provided or found to repair")))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("incident key %q is not a valid key", firstBadKey))
		}
		retries := flagOpsRepairIncidentRetries
		mode := ops.RepairDiscoveryModeKeyed
		if searchMode {
			mode = ops.RepairDiscoveryModeSearch
		} else if len(stdinKeys) > 0 && len(flagOpsRepairIncidentKeys) == 0 {
			mode = ops.RepairDiscoveryModeStdin
		}
		result, err := cli.RepairIncidents(cmd.Context(), ops.RepairRequest{
			CommandName:         opsRepairIncidentCommandName,
			Target:              ops.RepairTargetIncident,
			DiscoveryMode:       mode,
			InputKeys:           append(typex.Keys{}, keys...),
			IncidentSelection:   populateOpsRepairIncidentSelection(),
			BatchSize:           resolveOpsRepairIncidentSearchSize(cmd, cfg),
			Limit:               flagOpsRepairIncidentLimit,
			Workers:             flagWorkers,
			FailFast:            flagFailFast,
			NoWorkerLimit:       flagNoWorkerLimit,
			DryRun:              flagDryRun,
			AutoConfirm:         flagCmdAutoConfirm,
			Automation:          automationModeEnabled(cmd),
			NoWait:              flagNoWait,
			OutputMode:          pickMode().String(),
			Variables:           variables,
			VariablesFile:       variablesFile,
			RequestedRetries:    &retries,
			RequestedJobTimeout: timeout,
			ReportFile:          flagOpsRepairIncidentReportFile,
			ReportFormat:        reportFormat,
			StartedAt:           time.Now().UTC(),
		}, collectOptions()...)
		if reportErr := writeOpsRepairReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops repair incident: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops repair incident audit report: %w", reportErr))
		}
		renderErr := renderOpsRepairIncidentResult(cmd, result)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops repair incident: %w", err))
		}
		if renderErr != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops repair incident: %w", renderErr))
		}
	},
}

func init() {
	opsRepairCmd.AddCommand(opsRepairIncidentCmd)
	useInvalidInputFlagErrors(opsRepairIncidentCmd)

	fs := opsRepairIncidentCmd.Flags()
	fs.StringSliceVarP(&flagOpsRepairIncidentKeys, "key", "k", nil, "incident key(s) to repair; repeat or combine with stdin '-'")
	fs.StringVarP(&flagOpsRepairIncidentState, "state", "s", "active", "incident state scope for search: active, pending, resolved, migrated, unknown, all")
	fs.StringVar(&flagOpsRepairIncidentErrorType, "error-type", "", "case-insensitive incident error type filter for search")
	fs.StringVar(&flagOpsRepairIncidentErrorMessage, "error-message", "", "case-insensitive incident error message substring filter for search")
	fs.StringVarP(&flagOpsRepairIncidentBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter incidents")
	fs.StringVar(&flagOpsRepairIncidentPDKey, "pd-key", "", "process definition key to filter incidents")
	fs.StringVar(&flagOpsRepairIncidentPIKey, "pi-key", "", "process instance key to filter incidents")
	fs.StringVar(&flagOpsRepairIncidentRootKey, "root-key", "", "root process instance key to filter incidents")
	fs.StringVar(&flagOpsRepairIncidentFlowNodeID, "flow-node-id", "", "flow node ID to filter incidents")
	fs.StringVar(&flagOpsRepairIncidentFNIKey, "fni-key", "", "flow node instance key to filter incidents")
	fs.StringVar(&flagOpsRepairIncidentCreationTimeAfter, "creation-time-after", "", "only include incidents with creation time >= RFC3339 timestamp or YYYY-MM-DD")
	fs.StringVar(&flagOpsRepairIncidentCreationTimeBefore, "creation-time-before", "", "only include incidents with creation time <= RFC3339 timestamp or YYYY-MM-DD")
	fs.Int32VarP(&flagOpsRepairIncidentBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of incidents to inspect per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagOpsRepairIncidentLimit, "limit", "l", 0, "maximum number of matching incidents to repair")
	fs.Int32Var(&flagOpsRepairIncidentRetries, "retries", 1, "retry count to set on related jobs; 0 skips retry restoration")
	fs.StringVar(&flagOpsRepairIncidentJobTimeoutRaw, "job-timeout", "", "timeout duration to submit for related jobs, for example 60s, 5m, or 1h")
	fs.StringVar(&flagOpsRepairIncidentVars, "vars", "", "JSON object with variables to set once per process-instance scope before resolving dependent incidents")
	fs.StringVar(&flagOpsRepairIncidentVarsFile, "vars-file", "", "path to JSON object file with variables to set once per process-instance scope")
	fs.StringVar(&flagOpsRepairIncidentReportFile, "report-file", "", "plan an audit report at the given path")
	fs.StringVar(&flagOpsRepairIncidentReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")
	fs.BoolVar(&flagDryRun, "dry-run", false, "freeze repair targets and preview repair steps without submitting mutations")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after repair mutations are accepted without incident or retry confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when repairing multiple incidents (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling incident repairs after the first error")

	setCommandMutation(opsRepairIncidentCmd, CommandMutationStateChanging)
	setContractSupport(opsRepairIncidentCmd, ContractSupportFull)
	setAutomationSupport(opsRepairIncidentCmd, AutomationSupportFull, "supports shared machine output, stdin key pipelines, dry-run previews, and unattended repair")
	setOutputModes(opsRepairIncidentCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}

func validateOpsRepairIncidentFlagValues(cmd *cobra.Command) error {
	if flagOpsRepairIncidentBatchSize <= 0 || flagOpsRepairIncidentBatchSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagOpsRepairIncidentBatchSize, consts.MaxPISearchSize)
	}
	if flagOpsRepairIncidentLimit < 0 || (flagOpsRepairIncidentLimit == 0 && cmd != nil && cmd.Flags().Changed("limit")) {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if cmd != nil && cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	if err := validateGetIncidentStateFlag(flagOpsRepairIncidentState); err != nil {
		return err
	}
	if err := validateGetIncidentErrorTypeFlag(flagOpsRepairIncidentErrorType); err != nil {
		return err
	}
	if err := validateGetIncidentCreationTimeFlag("--creation-time-after", flagOpsRepairIncidentCreationTimeAfter); err != nil {
		return err
	}
	if err := validateGetIncidentCreationTimeFlag("--creation-time-before", flagOpsRepairIncidentCreationTimeBefore); err != nil {
		return err
	}
	if flagOpsRepairIncidentRetries < 0 {
		return invalidFlagValuef("invalid value for --retries: %d, expected non-negative integer", flagOpsRepairIncidentRetries)
	}
	if cmd != nil && cmd.Flags().Changed("vars") && cmd.Flags().Changed("vars-file") {
		return mutuallyExclusiveFlagsf("--vars cannot be combined with --vars-file")
	}
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for ops repair incident")
	}
	if err := validateOpsWorkflowReportFlags(flagOpsRepairIncidentReportFile, OpsWorkflowReportFormat(flagOpsRepairIncidentReportFormat)); err != nil {
		return err
	}
	if ok, firstBadKey, _ := validateKeys(flagOpsRepairIncidentKeys); len(flagOpsRepairIncidentKeys) > 0 && !ok {
		return invalidFlagValuef("incident key %q is not a valid key", firstBadKey)
	}
	if len(flagOpsRepairIncidentKeys) > 0 && hasOpsRepairIncidentSearchModeFlags(cmd) {
		return mutuallyExclusiveFlagsf("--key cannot be combined with search filters")
	}
	for flag, value := range map[string]string{
		"--pi-key":   flagOpsRepairIncidentPIKey,
		"--root-key": flagOpsRepairIncidentRootKey,
		"--pd-key":   flagOpsRepairIncidentPDKey,
		"--fni-key":  flagOpsRepairIncidentFNIKey,
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

func parseOpsRepairIncidentJobTimeout(cmd *cobra.Command) (time.Duration, error) {
	if cmd == nil || !cmd.Flags().Changed("job-timeout") {
		return 0, nil
	}
	timeout, err := time.ParseDuration(flagOpsRepairIncidentJobTimeoutRaw)
	if err != nil || timeout <= 0 {
		return 0, invalidFlagValuef("invalid value for --job-timeout: %q, expected positive duration such as 60s, 5m, or 1h", flagOpsRepairIncidentJobTimeoutRaw)
	}
	if timeout.Milliseconds() <= 0 {
		return 0, invalidFlagValuef("invalid value for --job-timeout: %q, duration must be at least 1ms", flagOpsRepairIncidentJobTimeoutRaw)
	}
	return timeout, nil
}

// hasOpsRepairIncidentSearchModeFlags detects explicit incident search mode without treating default state as an implicit mutation target.
func hasOpsRepairIncidentSearchModeFlags(cmd *cobra.Command) bool {
	if cmd == nil {
		return false
	}
	for _, name := range []string{
		"state",
		"error-type",
		"error-message",
		"pi-key",
		"root-key",
		"pd-key",
		"bpmn-process-id",
		"flow-node-id",
		"fni-key",
		"creation-time-after",
		"creation-time-before",
		"batch-size",
		"limit",
	} {
		if cmd.Flags().Changed(name) {
			return true
		}
	}
	return false
}

// populateOpsRepairIncidentSelection converts repair command search flags into the public incident filter model.
func populateOpsRepairIncidentSelection() incident.Filter {
	errorType, _ := incidentfilter.NormalizeErrorType(flagOpsRepairIncidentErrorType)
	return incident.Filter{
		State:                  flagOpsRepairIncidentState,
		ErrorType:              errorType,
		ErrorMessage:           flagOpsRepairIncidentErrorMessage,
		ProcessDefinitionId:    flagOpsRepairIncidentBpmnProcessID,
		ProcessDefinitionKey:   flagOpsRepairIncidentPDKey,
		ProcessInstanceKey:     flagOpsRepairIncidentPIKey,
		RootProcessInstanceKey: flagOpsRepairIncidentRootKey,
		FlowNodeId:             flagOpsRepairIncidentFlowNodeID,
		FlowNodeInstanceKey:    flagOpsRepairIncidentFNIKey,
		CreationTimeAfter:      flagOpsRepairIncidentCreationTimeAfter,
		CreationTimeBefore:     flagOpsRepairIncidentCreationTimeBefore,
	}
}

// resolveOpsRepairIncidentSearchSize applies the existing incident page-size default policy.
func resolveOpsRepairIncidentSearchSize(cmd *cobra.Command, cfg *config.Config) int32 {
	if cmd != nil && cmd.Flags().Changed("batch-size") {
		return pickOpsRepairIncidentSearchSize()
	}
	if cfg != nil && cfg.App.ProcessInstancePageSize > 0 && cfg.App.ProcessInstancePageSize <= consts.MaxPISearchSize {
		return cfg.App.ProcessInstancePageSize
	}
	return consts.MaxPISearchSize
}

// pickOpsRepairIncidentSearchSize returns a safe search size after validation has run.
func pickOpsRepairIncidentSearchSize() int32 {
	if flagOpsRepairIncidentBatchSize <= 0 || flagOpsRepairIncidentBatchSize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return flagOpsRepairIncidentBatchSize
}

// resolveOpsRepairReportFormat returns the explicit or inferred repair report format for planned output.
func resolveOpsRepairReportFormat(reportPath string, requested string) (string, error) {
	if reportPath == "" {
		return "", nil
	}
	format, err := opsWorkflowReportFormatForPath(reportPath, OpsWorkflowReportFormat(requested))
	if err != nil {
		return "", invalidFlagValuef("%v", err)
	}
	return format.String(), nil
}
