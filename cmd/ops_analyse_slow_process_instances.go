// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsAnalyseSlowProcessInstancesCommandName = "ops analyse slow-process-instances"

const (
	opsSlowProcessAnalysisTimeExpectedFormat = "RFC3339 timestamp, c8volt timestamp YYYY-MM-DDTHH:MM:SS[.fraction], or YYYY-MM-DD"
	opsSlowProcessAnalysisTimeLayout         = "2006-01-02T15:04:05"
	opsSlowProcessAnalysisTimeFractionLayout = "2006-01-02T15:04:05.999999999"
)

var (
	flagOpsAnalyseSlowProcessInstanceKeys                  []string
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID         string
	flagOpsAnalyseSlowProcessInstancePDKey                 string
	flagOpsAnalyseSlowProcessInstanceState                 string
	flagOpsAnalyseSlowProcessInstanceStartDateAfter        string
	flagOpsAnalyseSlowProcessInstanceStartDateBefore       string
	flagOpsAnalyseSlowProcessInstanceEndDateAfter          string
	flagOpsAnalyseSlowProcessInstanceEndDateBefore         string
	flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly       bool
	flagOpsAnalyseSlowProcessInstanceBatchSize             int32
	flagOpsAnalyseSlowProcessInstanceLimit                 int32
	flagOpsAnalyseSlowProcessInstanceElementID             string
	flagOpsAnalyseSlowProcessInstanceType                  string
	flagOpsAnalyseSlowProcessInstanceElementState          string
	flagOpsAnalyseSlowProcessInstanceDurationLonger        string
	flagOpsAnalyseSlowProcessInstanceElementDurationLonger string
	flagOpsAnalyseSlowProcessInstanceWithFullTimeline      bool
	flagOpsAnalyseSlowProcessInstanceWithListeners         bool
)

var opsAnalyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "Discover read-only operational analyses",
	Long: "Discover read-only operational analyses.\n\n" +
		"The analyse command family groups inspection workflows that combine existing runtime resources without mutating cluster state.",
	Example: `  ./c8volt ops analyse --help
  ./c8volt ops analyse slow-process-instances --help`,
	Aliases: []string{"analyze"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var opsAnalyseSlowProcessInstancesCmd = &cobra.Command{
	Use:   "slow-process-instances [-]",
	Short: "Analyse slow process-instance timings",
	Long: "Analyse slow process-instance timings.\n\n" +
		"The command is read-only. Select process instances by explicit --key values or by exactly one process-definition selector, then inspect process and runtime element timing without changing cluster state.\n\n" +
		"Search mode pages through discovered process instances by default. --batch-size controls each discovery page request, --limit caps the frozen analysis scope, and explicit keys bypass discovery paging. JSON and keys-only output stay free of progress text.\n\n" +
		"Use --dur-longer to keep only process-instance roots whose whole duration is above a threshold. Detail filters such as --element-id, --type, --element-state, and --dur-element-longer keep only process instances with matching element or transition detail rows, then show those matching rows under the root.\n\n" +
		"Default output shows compact slowest element contributors. Use --with-full-timeline to inspect complete chronological element and transition detail.\n\n" +
		"Use --with-listeners to include runtime listener jobs under matching element timeline rows.\n\n" +
		"Duration thresholds use Go duration syntax such as 500ms, 30s, 5m, 1h, 1h30m, or 24h. Calendar units such as 1d are not accepted.\n\n" +
		"JSON output exposes stable duration, comparison, and timeline fields. Keys-only output prints selected process-instance keys in result order, one per line.",
	Example: `  ./c8volt ops analyse slow-process-instances --key <process-instance-key>
  ./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --state active --dur-longer 5m
  ./c8volt ops analyse slow-process-instances --pd-key <process-definition-key> --dur-element-longer 30s
  ./c8volt ops analyse slow-process-instances --key <process-instance-key> --with-full-timeline
  ./c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
  ./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --element-id <element-id> --dur-element-longer 30s
  ./c8volt get process-instance --state active --keys-only | ./c8volt ops analyse slow-process-instances -`,
	Aliases: []string{"slow-pi", "spi"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateOpsSlowProcessAnalysisCommandArgs(cmd, args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagOpsAnalyseSlowProcessInstanceKeys, stdinKeys, log, cfg).Unique()
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("process-instance key %q is not a valid key", firstBadKey))
		}
		parsed, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, args, keys)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		configureOpsSlowProcessAnalysisPreflight(cmd, &parsed.Request)
		result, err := cli.AnalyseSlowProcessInstances(cmd.Context(), parsed.Request, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops analyse slow-process-instances: %w", err))
		}
		if err := renderOpsSlowProcessAnalysisResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops analyse slow-process-instances: %w", err))
		}
	},
}

// opsSlowProcessAnalysisCommandRequest keeps command-local parse state separate from facade input.
type opsSlowProcessAnalysisCommandRequest struct {
	Request          ops.SlowProcessAnalysisRequest
	StdinRequested   bool
	WithFullTimeline bool
}

func init() {
	opsCmd.AddCommand(opsAnalyseCmd)
	opsAnalyseCmd.AddCommand(opsAnalyseSlowProcessInstancesCmd)
	useInvalidInputFlagErrors(opsAnalyseSlowProcessInstancesCmd)

	fs := opsAnalyseSlowProcessInstancesCmd.Flags()
	fs.StringSliceVarP(&flagOpsAnalyseSlowProcessInstanceKeys, "key", "k", nil, "process-instance key(s) to analyse; repeat or combine with stdin '-'")
	fs.StringVarP(&flagOpsAnalyseSlowProcessInstanceBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to discover process instances")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstancePDKey, "pd-key", "", "process definition key to discover process instances")
	fs.StringVarP(&flagOpsAnalyseSlowProcessInstanceState, "state", "s", "all", "state to filter discovered process instances: all, active, completed, canceled, terminated")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceStartDateAfter, "start-date-after", "", "only include process instances with start date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceStartDateBefore, "start-date-before", "", "only include process instances with start date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceEndDateAfter, "end-date-after", "", "only include process instances with end date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceEndDateBefore, "end-date-before", "", "only include process instances with end date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.BoolVar(&flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly, "no-incidents-only", false, "only include process instances without incidents during discovery")
	fs.Int32VarP(&flagOpsAnalyseSlowProcessInstanceBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per discovery page; does not cap explicit keys or timeline details (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagOpsAnalyseSlowProcessInstanceLimit, "limit", "l", 0, "maximum number of matching process instances to freeze during discovery; omit to discover all matches")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceElementID, "element-id", "", "BPMN element ID to keep in detail rows")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceType, "type", "", "runtime element type to keep in detail rows")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceElementState, "element-state", "", "runtime element state to keep in detail rows")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceDurationLonger, "dur-longer", "", "only include process instances whose whole duration is longer than this duration, for example 5m or 1h30m")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceElementDurationLonger, "dur-element-longer", "", "only include process instances with element or transition detail rows longer than this duration, for example 30s or 2m")
	fs.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithFullTimeline, "with-full-timeline", false, "show complete chronological element and transition detail")
	fs.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithListeners, "with-listeners", false, "include runtime listener jobs under matching element timeline rows")

	setCommandMutation(opsAnalyseCmd, CommandMutationReadOnly)
	setCommandMutation(opsAnalyseSlowProcessInstancesCmd, CommandMutationReadOnly)
	setContractSupport(opsAnalyseSlowProcessInstancesCmd, ContractSupportFull)
	setAutomationSupport(opsAnalyseSlowProcessInstancesCmd, AutomationSupportFull, "supports read-only analysis with shared machine output and key pipelines")
	setOutputModes(opsAnalyseSlowProcessInstancesCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true, Notes: "stdout remains one JSON document; preflight and frozen-scope metadata are exposed as result fields"},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true, Notes: "stdout remains one process-instance key per line with no progress or preflight text"},
	)
}

// validateOpsSlowProcessAnalysisCommandArgs rejects invalid selector combinations before client setup.
func validateOpsSlowProcessAnalysisCommandArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 1 && args[0] != "-" {
		return invalidFlagValuef("unexpected positional argument %q; use '-' to read process-instance keys from stdin", args[0])
	}
	if len(args) > 1 {
		return invalidFlagValuef("unexpected positional arguments %v; use one '-' to read process-instance keys from stdin", args)
	}
	if flagOpsAnalyseSlowProcessInstanceBatchSize <= 0 || flagOpsAnalyseSlowProcessInstanceBatchSize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagOpsAnalyseSlowProcessInstanceBatchSize, consts.MaxPISearchSize)
	}
	if flagOpsAnalyseSlowProcessInstanceLimit < 0 || (flagOpsAnalyseSlowProcessInstanceLimit == 0 && cmd != nil && cmd.Flags().Changed("limit")) {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if _, err := parseOpsSlowProcessAnalysisRootDurationLonger(); err != nil {
		return err
	}
	if _, err := parseOpsSlowProcessAnalysisDetailDurationLonger(); err != nil {
		return err
	}
	if _, err := parseOpsSlowProcessAnalysisElementType(); err != nil {
		return err
	}
	if _, err := parseOpsSlowProcessAnalysisElementState(); err != nil {
		return err
	}
	if flagOpsAnalyseSlowProcessInstanceWithListeners && pickMode() == RenderModeKeysOnly {
		return mutuallyExclusiveFlagsf("--with-listeners cannot be combined with --keys-only")
	}
	stdinRequested := len(args) == 1 && args[0] == "-"
	keyedMode := len(flagOpsAnalyseSlowProcessInstanceKeys) > 0 || stdinRequested
	processDefinitionSelectorMode := flagOpsAnalyseSlowProcessInstanceBpmnProcessID != "" || flagOpsAnalyseSlowProcessInstancePDKey != ""
	if flagOpsAnalyseSlowProcessInstanceBpmnProcessID != "" && flagOpsAnalyseSlowProcessInstancePDKey != "" {
		return mutuallyExclusiveFlagsf("--bpmn-process-id cannot be combined with --pd-key")
	}
	if keyedMode && processDefinitionSelectorMode {
		return mutuallyExclusiveFlagsf("--key/stdin '-' cannot be combined with process-definition selectors")
	}
	if keyedMode && hasOpsSlowProcessAnalysisSearchFilterFlags(cmd) {
		return mutuallyExclusiveFlagsf("explicit process-instance keys cannot be combined with process-instance search filters")
	}
	if !keyedMode && !processDefinitionSelectorMode {
		return localPreconditionError(fmt.Errorf("select process instances with --key, stdin '-', --bpmn-process-id, or --pd-key"))
	}
	if _, err := parseOpsSlowProcessAnalysisState(); err != nil {
		return err
	}
	if _, _, err := parseOpsSlowProcessAnalysisDateRange("--start-date-after", flagOpsAnalyseSlowProcessInstanceStartDateAfter, "--start-date-before", flagOpsAnalyseSlowProcessInstanceStartDateBefore); err != nil {
		return err
	}
	if _, _, err := parseOpsSlowProcessAnalysisDateRange("--end-date-after", flagOpsAnalyseSlowProcessInstanceEndDateAfter, "--end-date-before", flagOpsAnalyseSlowProcessInstanceEndDateBefore); err != nil {
		return err
	}
	if ok, firstBadKey, _ := validateKeys(flagOpsAnalyseSlowProcessInstanceKeys); len(flagOpsAnalyseSlowProcessInstanceKeys) > 0 && !ok {
		return invalidFlagValuef("process-instance key %q is not a valid key", firstBadKey)
	}
	return nil
}

// buildOpsSlowProcessAnalysisCommandRequest turns validated command flags into facade input.
func buildOpsSlowProcessAnalysisCommandRequest(cmd *cobra.Command, args []string, keys typex.Keys) (opsSlowProcessAnalysisCommandRequest, error) {
	if err := validateOpsSlowProcessAnalysisCommandArgs(cmd, args); err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	rootDurationLonger, err := parseOpsSlowProcessAnalysisRootDurationLonger()
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	detailDurationLonger, err := parseOpsSlowProcessAnalysisDetailDurationLonger()
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	elementType, err := parseOpsSlowProcessAnalysisElementType()
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	elementState, err := parseOpsSlowProcessAnalysisElementState()
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	state, err := parseOpsSlowProcessAnalysisState()
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	startDateAfter, startDateBefore, err := parseOpsSlowProcessAnalysisDateRange("--start-date-after", flagOpsAnalyseSlowProcessInstanceStartDateAfter, "--start-date-before", flagOpsAnalyseSlowProcessInstanceStartDateBefore)
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	endDateAfter, endDateBefore, err := parseOpsSlowProcessAnalysisDateRange("--end-date-after", flagOpsAnalyseSlowProcessInstanceEndDateAfter, "--end-date-before", flagOpsAnalyseSlowProcessInstanceEndDateBefore)
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}

	stdinRequested := len(args) == 1 && args[0] == "-"
	selectionMode := ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch
	inputKeys := typex.Keys(nil)
	if len(keys) > 0 || stdinRequested {
		selectionMode = ops.SlowProcessAnalysisSelectionModeExplicitKeys
		inputKeys = append(typex.Keys(nil), keys...)
	}

	return opsSlowProcessAnalysisCommandRequest{
		StdinRequested:   stdinRequested,
		WithFullTimeline: flagOpsAnalyseSlowProcessInstanceWithFullTimeline,
		Request: ops.SlowProcessAnalysisRequest{
			CommandName:   opsAnalyseSlowProcessInstancesCommandName,
			SelectionMode: selectionMode,
			InputKeys:     inputKeys,
			ProcessDefinitionSelector: ops.SlowProcessAnalysisProcessDefinitionSelector{
				BpmnProcessID:        flagOpsAnalyseSlowProcessInstanceBpmnProcessID,
				ProcessDefinitionKey: flagOpsAnalyseSlowProcessInstancePDKey,
			},
			ProcessInstanceFilters: ops.SlowProcessAnalysisProcessInstanceSearchFilters{
				State:           state,
				StartDateAfter:  startDateAfter,
				StartDateBefore: startDateBefore,
				EndDateAfter:    endDateAfter,
				EndDateBefore:   endDateBefore,
				NoIncidentsOnly: flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly,
			},
			DetailFilters: ops.SlowProcessAnalysisDetailFilters{
				ElementID:     flagOpsAnalyseSlowProcessInstanceElementID,
				Type:          elementType,
				ElementState:  elementState,
				DurationAfter: detailDurationLonger,
			},
			RootDurationLonger: rootDurationLonger,
			BatchSize:          flagOpsAnalyseSlowProcessInstanceBatchSize,
			Limit:              flagOpsAnalyseSlowProcessInstanceLimit,
			CapturedNow:        time.Now().UTC(),
			OutputMode:         pickMode().String(),
			WithListeners:      flagOpsAnalyseSlowProcessInstanceWithListeners,
		},
	}, nil
}

// configureOpsSlowProcessAnalysisPreflight wires command-owned rendering and prompting into service-owned discovery.
func configureOpsSlowProcessAnalysisPreflight(cmd *cobra.Command, request *ops.SlowProcessAnalysisRequest) {
	if request == nil || request.SelectionMode != ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch {
		return
	}
	channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
	request.Progress = func(event ops.ProgressEvent) {
		switch event.Kind {
		case ops.ProgressEventKindPreflight:
			if event.Preflight != nil {
				printOpsPreflightScope(cmd, *event.Preflight, channel)
			}
		case ops.ProgressEventKindPage:
			if event.Page != nil {
				printOpsSlowProcessAnalysisProgress(cmd, formatOpsPageProgress(*event.Page, "process instance(s)"), channel)
			}
		case ops.ProgressEventKindFrozenScope:
			if event.FrozenScope != nil {
				printOpsSlowProcessAnalysisProgress(cmd, formatOpsFrozenScopeProgress(*event.FrozenScope), channel)
			}
		case ops.ProgressEventKindETA:
			if event.ETA != nil && opsETAAllowed(*event.ETA) {
				printOpsSlowProcessAnalysisProgress(cmd, formatOpsETASampleWindow(*event.ETA), channel)
			}
		}
	}
	request.ConfirmPreflight = func(scope ops.PreflightScope) error {
		if !opsSlowProcessAnalysisPreflightConfirmationAllowed(channel) || !scope.RequiresConfirmation {
			return nil
		}
		prompt := strings.TrimSpace(scope.ConsequenceSummary.ConfirmationText)
		if prompt == "" {
			prompt = "Continue slow analysis?"
		}
		return confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt)
	}
}

// printOpsSlowProcessAnalysisProgress routes transient and verbose progress without touching command stdout.
func printOpsSlowProcessAnalysisProgress(cmd *cobra.Command, line string, channel ops.ProgressChannel) {
	if cmd == nil || strings.TrimSpace(line) == "" {
		return
	}
	if channel.TransientAllowed {
		logging.UpdateActivity(cmd.Context(), line)
	}
	if !opsSlowProcessAnalysisDurableProgressAllowed(channel) {
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), line)
}

// printOpsPreflightScope writes durable preflight lines only to the command's stderr/activity channel.
func printOpsPreflightScope(cmd *cobra.Command, scope ops.PreflightScope, channel ops.ProgressChannel) {
	if cmd == nil || !channel.DurableAllowed || !channel.StderrAllowed {
		return
	}
	for _, line := range formatOpsPreflightScope(scope) {
		fmt.Fprintln(cmd.ErrOrStderr(), line)
	}
}

// opsSlowProcessAnalysisDurableProgressAllowed keeps page/counter detail behind verbose or debug while activity remains compact.
func opsSlowProcessAnalysisDurableProgressAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed && (channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

// opsSlowProcessAnalysisPreflightConfirmationAllowed limits prompts to human progress modes.
func opsSlowProcessAnalysisPreflightConfirmationAllowed(channel ops.ProgressChannel) bool {
	return channel.Mode == ops.ProgressModeHuman || channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug
}

// parseOpsSlowProcessAnalysisElementType keeps detail type filters aligned with runtime element search values.
func parseOpsSlowProcessAnalysisElementType() (string, error) {
	if strings.TrimSpace(flagOpsAnalyseSlowProcessInstanceType) == "" {
		return "", nil
	}
	if !validElementType(flagOpsAnalyseSlowProcessInstanceType) {
		return "", invalidFlagValuef("invalid value for --type: %q, valid values are: %s", flagOpsAnalyseSlowProcessInstanceType, strings.Join(validElementTypes, ", "))
	}
	return normalizedElementType(flagOpsAnalyseSlowProcessInstanceType), nil
}

// parseOpsSlowProcessAnalysisElementState keeps detail state filters separate from the process-instance --state flag.
func parseOpsSlowProcessAnalysisElementState() (string, error) {
	if strings.TrimSpace(flagOpsAnalyseSlowProcessInstanceElementState) == "" {
		return "", nil
	}
	if !validElementState(flagOpsAnalyseSlowProcessInstanceElementState) {
		return "", invalidFlagValuef("invalid value for --element-state: %q, valid values are: %s", flagOpsAnalyseSlowProcessInstanceElementState, strings.Join(validElementStates, ", "))
	}
	return normalizedElementState(flagOpsAnalyseSlowProcessInstanceElementState), nil
}

// hasOpsSlowProcessAnalysisSearchFilterFlags identifies flags valid only for process-definition discovery.
func hasOpsSlowProcessAnalysisSearchFilterFlags(cmd *cobra.Command) bool {
	return flagOpsAnalyseSlowProcessInstanceStartDateAfter != "" ||
		flagOpsAnalyseSlowProcessInstanceStartDateBefore != "" ||
		flagOpsAnalyseSlowProcessInstanceEndDateAfter != "" ||
		flagOpsAnalyseSlowProcessInstanceEndDateBefore != "" ||
		flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly ||
		(cmd != nil && cmd.Flags().Changed("state")) ||
		(cmd != nil && cmd.Flags().Changed("batch-size")) ||
		(cmd != nil && cmd.Flags().Changed("limit"))
}

// parseOpsSlowProcessAnalysisState keeps accepted state tokens aligned with process-instance search.
func parseOpsSlowProcessAnalysisState() (process.State, error) {
	state, ok := process.ParseState(flagOpsAnalyseSlowProcessInstanceState)
	if !ok || state == process.StateAbsent {
		return "", invalidFlagValuef("invalid value for --state: %q, expected active, completed, canceled, terminated, or all", flagOpsAnalyseSlowProcessInstanceState)
	}
	if state == process.StateAll {
		return "", nil
	}
	return state, nil
}

// parseOpsSlowProcessAnalysisRootDurationLonger validates root process-instance duration filtering.
func parseOpsSlowProcessAnalysisRootDurationLonger() (time.Duration, error) {
	return parseOpsSlowProcessAnalysisDurationFlag("--dur-longer", flagOpsAnalyseSlowProcessInstanceDurationLonger)
}

// parseOpsSlowProcessAnalysisDetailDurationLonger validates detail duration filtering.
func parseOpsSlowProcessAnalysisDetailDurationLonger() (time.Duration, error) {
	return parseOpsSlowProcessAnalysisDurationFlag("--dur-element-longer", flagOpsAnalyseSlowProcessInstanceElementDurationLonger)
}

// parseOpsSlowProcessAnalysisDurationFlag validates Go duration syntax for ops thresholds.
func parseOpsSlowProcessAnalysisDurationFlag(flagName string, raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return 0, invalidFlagValuef("invalid value for %s: %q, expected duration such as 500ms, 1s, 2m, or 1h", flagName, raw)
	}
	if value < 0 {
		return 0, invalidFlagValuef("%s must not be negative", flagName)
	}
	return value, nil
}

// parseOpsSlowProcessAnalysisDateRange validates and normalizes inclusive search bounds for process-instance discovery.
func parseOpsSlowProcessAnalysisDateRange(afterFlag string, afterValue string, beforeFlag string, beforeValue string) (string, string, error) {
	after, err := normalizeOpsSlowProcessAnalysisLowerBound(afterValue)
	if err != nil {
		return "", "", invalidFlagValuef("invalid value for %s: %q, expected %s", afterFlag, afterValue, opsSlowProcessAnalysisTimeExpectedFormat)
	}
	before, err := normalizeOpsSlowProcessAnalysisUpperBound(beforeValue)
	if err != nil {
		return "", "", invalidFlagValuef("invalid value for %s: %q, expected %s", beforeFlag, beforeValue, opsSlowProcessAnalysisTimeExpectedFormat)
	}
	if after != "" && before != "" {
		afterTime, err := time.Parse(time.RFC3339Nano, after)
		if err != nil {
			return "", "", err
		}
		beforeTime, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			return "", "", err
		}
		if afterTime.After(beforeTime) {
			return "", "", invalidFlagValuef("invalid range for %s and %s: %q is later than %q", afterFlag, beforeFlag, afterValue, beforeValue)
		}
	}
	return after, before, nil
}

// normalizeOpsSlowProcessAnalysisLowerBound preserves precise timestamps and expands date-only lower bounds to day start.
func normalizeOpsSlowProcessAnalysisLowerBound(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if t, ok := parseOpsSlowProcessAnalysisTimestamp(raw); ok {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	return "", fmt.Errorf("parse %q as process-instance date bound", raw)
}

// normalizeOpsSlowProcessAnalysisUpperBound preserves precise timestamps and expands date-only upper bounds to day end.
func normalizeOpsSlowProcessAnalysisUpperBound(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if t, ok := parseOpsSlowProcessAnalysisTimestamp(raw); ok {
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	if t, err := time.Parse(time.DateOnly, raw); err == nil {
		t = t.AddDate(0, 0, 1).Add(-time.Nanosecond)
		return t.UTC().Format(time.RFC3339Nano), nil
	}
	return "", fmt.Errorf("parse %q as process-instance date bound", raw)
}

// parseOpsSlowProcessAnalysisTimestamp accepts RFC3339 and c8volt's compact UTC timestamp forms.
func parseOpsSlowProcessAnalysisTimestamp(raw string) (time.Time, bool) {
	if t, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(opsSlowProcessAnalysisTimeFractionLayout, raw, time.UTC); err == nil {
		return t, true
	}
	if t, err := time.ParseInLocation(opsSlowProcessAnalysisTimeLayout, raw, time.UTC); err == nil {
		return t, true
	}
	return time.Time{}, false
}
