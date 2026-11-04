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
	Use:     "analyse",
	Short:   "Discover read-only operational analyses",
	Long:    `Inspect runtime performance without changing cluster data.`,
	Example: `  c8volt ops analyse slow-process-instances --bpmn-process-id order --dur-longer 5m`,
	Aliases: []string{"analyze"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var opsAnalyseSlowProcessInstancesCmd = &cobra.Command{
	Use:   "slow-process-instances [-]",
	Short: "Analyse slow process-instance timings",
	Long: `Find slow process instances and the elements contributing to their duration.

Select a process with --bpmn-process-id or inspect one instance with --key.
Use --dur-longer for total duration, or --element-id and --dur-element-longer
for a specific element. Add --with-listeners to include listener jobs.

Durations use values such as 30s, 5m, or 24h. Use --limit to bound analysis.
Business-ID selection with --bid is available only c8.10+.`,
	Example: `  c8volt ops analyse slow-process-instances --bpmn-process-id order --state active --dur-longer 5m
  c8volt ops analyse slow-process-instances --key <process-instance-key> --with-listeners
  c8volt ops analyse slow-process-instances --bpmn-process-id order --element-id payment --dur-element-longer 30s`,
	Aliases: []string{"slow-pi", "spi"},
	Args: func(cmd *cobra.Command, args []string) error {
		// Validate BID after bootstrap for configured diagnostics and one JSON envelope.
		if searchBusinessIDFlag(cmd) != nil {
			return nil
		}
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
		if searchBusinessIDFlag(cmd) != nil {
			if err := validateOpsSlowProcessAnalysisCommandArgs(cmd, args); err != nil {
				handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err)
			}
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
		if searchBusinessIDFlag(cmd) != nil {
			runOpsSlowProcessAnalysisBusinessID(cmd, cli, log, cfg, parsed.Request)
			return
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

// init registers read-only analysis flags and local state completion.
func init() {
	opsCmd.AddCommand(opsAnalyseCmd)
	opsAnalyseCmd.AddCommand(opsAnalyseSlowProcessInstancesCmd)
	useInvalidInputFlagErrors(opsAnalyseSlowProcessInstancesCmd)

	fs := opsAnalyseSlowProcessInstancesCmd.Flags()
	fs.String("bid", "", "exact process business ID to select (only c8.10+)")
	fs.StringSliceVarP(&flagOpsAnalyseSlowProcessInstanceKeys, "key", "k", nil, "process-instance key(s) to analyse; repeat or combine with stdin '-'")
	fs.StringVarP(&flagOpsAnalyseSlowProcessInstanceBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to discover process instances")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstancePDKey, "pd-key", "", "process definition key to discover process instances")
	fs.StringVarP(&flagOpsAnalyseSlowProcessInstanceState, "state", "s", "all", "state to filter discovered process instances: all, active, suspended (only c8.10+), completed, canceled, terminated")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceStartDateAfter, "start-date-after", "", "only include process instances with start date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceStartDateBefore, "start-date-before", "", "only include process instances with start date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceEndDateAfter, "end-date-after", "", "only include process instances with end date >= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceEndDateBefore, "end-date-before", "", "only include process instances with end date <= RFC3339 timestamp, c8volt timestamp, or YYYY-MM-DD")
	fs.BoolVar(&flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly, "no-incidents-only", false, "only include process instances without incidents during discovery")
	fs.Int32VarP(&flagOpsAnalyseSlowProcessInstanceBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("process instances per discovery page (server maximum %d)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagOpsAnalyseSlowProcessInstanceLimit, "limit", "l", 0, "maximum matching instances to analyse; omit for all matches")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceElementID, "element-id", "", "BPMN element ID to include in the analysis")
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceType, "type", "", "runtime element type to include in the analysis")
	_ = opsAnalyseSlowProcessInstancesCmd.RegisterFlagCompletionFunc("state", processInstanceStateCompletion(false))
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
	if searchBusinessIDFlag(cmd) != nil && flagOpsAnalyseSlowProcessInstanceBpmnProcessID == "" && flagOpsAnalyseSlowProcessInstancePDKey == "" {
		selectionMode = ops.SlowProcessAnalysisSelectionModeProcessInstanceSearch
	}
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
				BusinessId:      searchBusinessIDFlag(cmd),
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
