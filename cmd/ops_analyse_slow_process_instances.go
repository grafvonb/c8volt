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
	Long: `Analyse process-instance and runtime-element durations without changing cluster state.

Select explicit --key values or exactly one process-definition selector. --batch-size controls each discovery request; --limit caps selected instances across all pages. Explicit keys bypass discovery paging.

--dur-longer selects roots whose total duration exceeds a threshold. --element-id, --type, --element-state, and --dur-element-longer restrict analysis to matching element or transition details. Use --with-full-timeline to inspect the complete chronology, or --with-listeners to include runtime listener jobs. Listener rows use s: for job creation (not worker execution start), e: for job end, and d: for an available deadline only while the state is exactly ACTIVATED; missing times are omitted.

Durations use Go syntax such as 500ms, 30s, 5m, 1h30m, or 24h. Calendar units such as 1d are not supported.`,
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
		keys := mergeAndValidateKeys(cmd, flagOpsAnalyseSlowProcessInstanceKeys, stdinKeys, log, cfg).Unique()
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
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceElementID, "element-id", "", "BPMN element ID to include in the analysis")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceType, "type", "", "runtime element type to include in the analysis")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceElementState, "element-state", "", "runtime element state to include in the analysis")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceDurationLonger, "dur-longer", "", "only include process instances whose whole duration is longer than this duration, for example 5m or 1h30m")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceElementDurationLonger, "dur-element-longer", "", "only include process instances with elements or transitions longer than this duration, for example 30s or 2m")
	fs.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithFullTimeline, "with-full-timeline", false, "show complete chronological element and transition detail")
	fs.BoolVar(&flagOpsAnalyseSlowProcessInstanceWithListeners, "with-listeners", false, "include runtime listener jobs")

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
