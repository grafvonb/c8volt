// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsAnalyseSlowProcessInstancesCommandName = "ops analyse slow-process-instances"

var (
	flagOpsAnalyseSlowProcessInstanceKeys            []string
	flagOpsAnalyseSlowProcessInstanceBpmnProcessID   string
	flagOpsAnalyseSlowProcessInstancePDKey           string
	flagOpsAnalyseSlowProcessInstanceState           string
	flagOpsAnalyseSlowProcessInstanceStartDateAfter  string
	flagOpsAnalyseSlowProcessInstanceStartDateBefore string
	flagOpsAnalyseSlowProcessInstanceEndDateAfter    string
	flagOpsAnalyseSlowProcessInstanceEndDateBefore   string
	flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly bool
	flagOpsAnalyseSlowProcessInstanceBatchSize       int32
	flagOpsAnalyseSlowProcessInstanceLimit           int32
	flagOpsAnalyseSlowProcessInstanceElementID       string
	flagOpsAnalyseSlowProcessInstanceType            string
	flagOpsAnalyseSlowProcessInstanceElementState    string
	flagOpsAnalyseSlowProcessInstanceDurationAfter   string
)

var opsAnalyseCmd = &cobra.Command{
	Use:   "analyse",
	Short: "Discover read-only operational analyses",
	Long: "Discover read-only operational analyses.\n\n" +
		"The analyse command family groups inspection workflows that combine existing runtime resources without mutating cluster state.",
	Example: `  ./c8volt ops analyse --help
  ./c8volt ops analyze slow-process-instances --help`,
	Aliases: []string{"analyze"},
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

var opsAnalyseSlowProcessInstancesCmd = &cobra.Command{
	Use:   "slow-process-instances [-]",
	Short: "Analyze slow process-instance timings",
	Long: "Analyze slow process-instance timings.\n\n" +
		"The command is read-only. Select process instances by explicit --key values or by exactly one process-definition selector, then inspect process and runtime element timing without changing cluster state.",
	Example: `  ./c8volt ops analyse slow-process-instances --key 2251799813685249
  ./c8volt ops analyze slow-process-instances --bpmn-process-id OrderProcess --state all --limit 20
  ./c8volt get pi --state active --keys-only | ./c8volt ops analyse slow-process-instances -`,
	Aliases: []string{"slow-pi"},
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		parsed, err := buildOpsSlowProcessAnalysisCommandRequest(cmd, args)
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
	Request        ops.SlowProcessAnalysisRequest
	StdinRequested bool
}

func init() {
	opsCmd.AddCommand(opsAnalyseCmd)
	opsAnalyseCmd.AddCommand(opsAnalyseSlowProcessInstancesCmd)
	useInvalidInputFlagErrors(opsAnalyseSlowProcessInstancesCmd)

	fs := opsAnalyseSlowProcessInstancesCmd.Flags()
	fs.StringSliceVarP(&flagOpsAnalyseSlowProcessInstanceKeys, "key", "k", nil, "process-instance key(s) to analyze; repeat or combine with stdin '-'")
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
	fs.StringVar(&flagOpsAnalyseSlowProcessInstanceDurationAfter, "duration-after", "", "only show element or transition detail rows longer than this duration")

	setCommandMutation(opsAnalyseCmd, CommandMutationReadOnly)
	setCommandMutation(opsAnalyseSlowProcessInstancesCmd, CommandMutationReadOnly)
	setContractSupport(opsAnalyseSlowProcessInstancesCmd, ContractSupportFull)
	setAutomationSupport(opsAnalyseSlowProcessInstancesCmd, AutomationSupportFull, "supports read-only analysis with shared machine output and key pipelines")
	setOutputModes(opsAnalyseSlowProcessInstancesCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: true},
	)
}

// buildOpsSlowProcessAnalysisCommandRequest turns validated command flags into facade input.
func buildOpsSlowProcessAnalysisCommandRequest(cmd *cobra.Command, args []string) (opsSlowProcessAnalysisCommandRequest, error) {
	if len(args) == 1 && args[0] != "-" {
		return opsSlowProcessAnalysisCommandRequest{}, invalidFlagValuef("unexpected positional argument %q; use '-' to read process-instance keys from stdin", args[0])
	}
	if flagOpsAnalyseSlowProcessInstanceBatchSize <= 0 || flagOpsAnalyseSlowProcessInstanceBatchSize > consts.MaxPISearchSize {
		return opsSlowProcessAnalysisCommandRequest{}, invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagOpsAnalyseSlowProcessInstanceBatchSize, consts.MaxPISearchSize)
	}
	if flagOpsAnalyseSlowProcessInstanceLimit < 0 || (flagOpsAnalyseSlowProcessInstanceLimit == 0 && cmd != nil && cmd.Flags().Changed("limit")) {
		return opsSlowProcessAnalysisCommandRequest{}, invalidFlagValuef("--limit must be positive integer")
	}
	durationAfter, err := parseOpsSlowProcessAnalysisDurationAfter()
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}
	state, err := parseOpsSlowProcessAnalysisState()
	if err != nil {
		return opsSlowProcessAnalysisCommandRequest{}, err
	}

	stdinRequested := len(args) == 1 && args[0] == "-"
	selectionMode := ops.SlowProcessAnalysisSelectionMode("")
	if len(flagOpsAnalyseSlowProcessInstanceKeys) > 0 || stdinRequested {
		selectionMode = ops.SlowProcessAnalysisSelectionModeExplicitKeys
	} else if flagOpsAnalyseSlowProcessInstanceBpmnProcessID != "" || flagOpsAnalyseSlowProcessInstancePDKey != "" {
		selectionMode = ops.SlowProcessAnalysisSelectionModeProcessDefinitionSearch
	}

	return opsSlowProcessAnalysisCommandRequest{
		StdinRequested: stdinRequested,
		Request: ops.SlowProcessAnalysisRequest{
			CommandName:   opsAnalyseSlowProcessInstancesCommandName,
			SelectionMode: selectionMode,
			InputKeys:     append(typex.Keys(nil), flagOpsAnalyseSlowProcessInstanceKeys...),
			ProcessDefinitionSelector: ops.SlowProcessAnalysisProcessDefinitionSelector{
				BpmnProcessID:        flagOpsAnalyseSlowProcessInstanceBpmnProcessID,
				ProcessDefinitionKey: flagOpsAnalyseSlowProcessInstancePDKey,
			},
			ProcessInstanceFilters: ops.SlowProcessAnalysisProcessInstanceSearchFilters{
				State:           state,
				StartDateAfter:  flagOpsAnalyseSlowProcessInstanceStartDateAfter,
				StartDateBefore: flagOpsAnalyseSlowProcessInstanceStartDateBefore,
				EndDateAfter:    flagOpsAnalyseSlowProcessInstanceEndDateAfter,
				EndDateBefore:   flagOpsAnalyseSlowProcessInstanceEndDateBefore,
				NoIncidentsOnly: flagOpsAnalyseSlowProcessInstanceNoIncidentsOnly,
			},
			DetailFilters: ops.SlowProcessAnalysisDetailFilters{
				ElementID:     flagOpsAnalyseSlowProcessInstanceElementID,
				Type:          flagOpsAnalyseSlowProcessInstanceType,
				ElementState:  flagOpsAnalyseSlowProcessInstanceElementState,
				DurationAfter: durationAfter,
			},
			BatchSize:   flagOpsAnalyseSlowProcessInstanceBatchSize,
			Limit:       flagOpsAnalyseSlowProcessInstanceLimit,
			CapturedNow: time.Now().UTC(),
			OutputMode:  pickMode().String(),
		},
	}, nil
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

// parseOpsSlowProcessAnalysisDurationAfter validates duration syntax without applying detail filtering yet.
func parseOpsSlowProcessAnalysisDurationAfter() (time.Duration, error) {
	if flagOpsAnalyseSlowProcessInstanceDurationAfter == "" {
		return 0, nil
	}
	value, err := time.ParseDuration(flagOpsAnalyseSlowProcessInstanceDurationAfter)
	if err != nil {
		return 0, invalidFlagValuef("invalid value for --duration-after: %q, expected duration such as 500ms, 1s, 2m, or 1h", flagOpsAnalyseSlowProcessInstanceDurationAfter)
	}
	if value < 0 {
		return 0, invalidFlagValuef("--duration-after must not be negative")
	}
	return value, nil
}
