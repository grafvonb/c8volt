// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsAnalyseSlowProcessInstancesCommandName = "ops analyse slow-process-instances"

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
		"Search mode pages through discovered process instances by default. Broad search selectors show preflight scope from the first discovery page before timeline loading, including exact, lower-bound, or unknown total wording and page-count context when available. Explicit keys bypass discovery paging and broad preflight.\n\n" +
		"--batch-size controls each discovery page request; it does not cap the frozen analysis scope, explicit keys, or timeline detail loading. --limit caps the number of matching process instances frozen for analysis across all discovery pages.\n\n" +
		"Default human mode uses terminal activity for preflight, discovery progress, and exact frozen-scope counters. Verbose and debug modes keep durable progress lines on stderr. JSON, keys-only, quiet, and automation output stay free of progress text.\n\n" +
		"Use --dur-longer to keep only process-instance roots whose whole duration is above a threshold. Detail filters such as --element-id, --type, --element-state, and --dur-element-longer keep only process instances with matching element or transition detail rows, then show those matching rows under the root.\n\n" +
		"Default output shows compact slowest element contributors. Use --with-full-timeline to inspect complete chronological element and transition detail.\n\n" +
		"Use --with-listeners to include runtime listener jobs under matching element timeline rows.\n\n" +
		"Duration thresholds use Go duration syntax such as 500ms, 30s, 5m, 1h, 1h30m, or 24h. Calendar units such as 1d are not accepted.\n\n" +
		"JSON output exposes stable duration, comparison, and timeline fields. Keys-only output prints selected process-instance keys in result order, one per line.",
	Example: `  ./c8volt ops analyse slow-process-instances --key <process-instance-key>
  ./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --state active --dur-longer 5m
  ./c8volt ops analyse slow-process-instances --bpmn-process-id <bpmn-process-id> --batch-size 500 --limit 2000
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
		progress := configureOpsSlowProcessAnalysisPreflight(cmd, &parsed.Request)
		result, err := cli.AnalyseSlowProcessInstances(cmd.Context(), parsed.Request, collectOptions()...)
		progress.Close()
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
	fs.Int32VarP(&flagOpsAnalyseSlowProcessInstanceBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per discovery page; does not cap frozen analysis scope, explicit keys, or timeline details (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagOpsAnalyseSlowProcessInstanceLimit, "limit", "l", 0, "maximum number of matching process instances to freeze for analysis across all discovery pages; omit to discover all matches")
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
