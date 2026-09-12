// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var (
	flagGetPDKey               string
	flagGetPDBpmnProcessId     string
	flagGetPDProcessVersion    int32
	flagGetPDProcessVersionTag string
	flagGetPDLatest            bool
	flagGetPDWithStat          bool
	flagGetPDAsXML             bool
	flagGetPDBatchSize         int32
	flagGetPDWatch             bool
	flagGetPDWatchInterval     string
)

var getProcessDefinitionCmd = &cobra.Command{
	Use:   "process-definition",
	Short: "List or fetch deployed process definitions",
	Long: `List or fetch deployed process definitions.

Select by key, BPMN process ID, version, or version tag. Use --xml only with --key. A --bpmn-process-id selector must match a visible definition.

--tenant limits list and selector discovery. Explicit --key and XML lookups use backend authorization without tenant filtering.

--latest selects the newest definition per exact tenant ID and BPMN process ID, breaking version ties by the lowest exact-text key. Camunda 8.7 selects within its 1000 visible-definition compatibility window; Camunda 8.8 or newer uses native latest filtering.

--stat includes exact-version statistics and requires Camunda 8.8 or newer.

--watch repeats the lookup until interrupted, timed out, or retries are exhausted. It starts immediately; --watch-interval controls subsequent checks. Without a selector it observes all visible definitions. Successful checks reset the consecutive retry budget. --watch cannot be combined with --json, --keys-only, --xml, --quiet, or --automation.`,
	Example: `  ./c8volt get process-definition --latest
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --watch
  ./c8volt get process-definition --watch --watch-interval 2s
  ./c8volt get process-definition --key <process-definition-key> --json
  ./c8volt get process-definition --key <process-definition-key> --xml`,
	Aliases: []string{"pd", "pds"},
	Run:     runGetProcessDefinition,
}

func runGetProcessDefinition(cmd *cobra.Command, args []string) {
	cli, log, cfg, err := NewCli(cmd)
	if err != nil {
		handleNewCliError(cmd, log, cfg, err)
	}
	if err := validateGetProcessDefinitionFlags(cmd); err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
	}

	log.Debug("getting pd")
	filter := populatePDSearchFilterOpts()
	if flagGetPDWatch {
		runGetProcessDefinitionWatch(cmd, cli, log, cfg.App.NoErrCodes, filter, cfg.App.Backoff.Timeout, cfg.App.Backoff.MaxRetries)
		return
	}
	if flagGetPDAsXML {
		runGetProcessDefinitionXML(cmd, cli, log, cfg.App.NoErrCodes, filter)
		return
	}
	if filter.Key != "" {
		runGetProcessDefinitionByKey(cmd, cli, log, cfg.App.NoErrCodes, filter.Key)
		return
	}
	runSearchProcessDefinitions(cmd, cli, log, cfg.App.NoErrCodes, filter)
}

func runGetProcessDefinitionXML(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter process.ProcessDefinitionFilter) {
	if err := validateProcessDefinitionXMLFlags(filter); err != nil {
		handleCommandError(cmd, log, noErrCodes, err)
	}

	log.Debug(fmt.Sprintf("getting pd %s xml", filter.Key))
	xml, err := cli.GetProcessDefinitionXML(cmd.Context(), filter.Key, collectExplicitAdminInputOptions()...)
	if err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("get process definition xml: %w", err))
	}
	if _, err := io.WriteString(cmd.OutOrStdout(), xml); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error writing process definition xml: %w", err))
	}
}

func runGetProcessDefinitionByKey(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, key string) {
	log.Debug(fmt.Sprintf("getting pd %s", key))
	pd, err := cli.GetProcessDefinition(cmd.Context(), key, collectExplicitAdminInputOptions()...)
	if err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("get process definition: %w", err))
	}
	if err := processDefinitionView(cmd, pd); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error rendering key-only view: %w", err))
	}
}

func runSearchProcessDefinitions(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter process.ProcessDefinitionFilter) {
	log.Debug(fmt.Sprintf("searching pd; filter %s", filter.String()))

	var (
		pds process.ProcessDefinitions
		err error
	)
	if filter.BpmnProcessId != "" {
		result, err := validateProcessDefinitionSelectorsForCommand(cmd.Context(), cmd, cli, newGetPDProcessDefinitionSelectorValidationRequest(), collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, noErrCodes, err)
		}
		if !result.Valid() {
			handleProcessDefinitionSelectorValidationError(cmd, log, noErrCodes, cli, result)
		}
		if len(result.Request.BpmnProcessIds) > 0 {
			pds = result.MatchesByBpmnProcessID[result.Request.BpmnProcessIds[0]]
		}
	} else {
		pds, err = searchProcessDefinitionsWithPaging(cmd, cli, filter)
	}
	if err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("search process definitions: %w", err))
	}
	if err := listProcessDefinitionsView(cmd, pds); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error rendering items view: %w", err))
	}
	log.Debug(fmt.Sprintf("pd search done; found %d", pds.Total))
}

func init() {
	getCmd.AddCommand(getProcessDefinitionCmd)

	fs := getProcessDefinitionCmd.Flags()
	fs.StringVarP(&flagGetPDKey, "key", "k", "", "process definition key to fetch")
	fs.StringVarP(&flagGetPDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.BoolVar(&flagGetPDLatest, "latest", false, "only include the latest matching process-definition version per exact tenant/BPMN process group")
	fs.Int32Var(&flagGetPDProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagGetPDWithStat, "stat", false, "include process definition statistics; 8.8 or newer includes incident counts, 8.7 unsupported")
	fs.BoolVar(&flagGetPDAsXML, "xml", false, "output the selected process definition as raw XML (requires --key and no other filters)")
	fs.Int32VarP(&flagGetPDBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process definitions to request per discovery page; does not cap total results (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.BoolVar(&flagGetPDWatch, "watch", false, "repeat the process-definition lookup until interrupted, timed out, or retries are exhausted")
	fs.Var(toolx.NewDurationStringValue(defaultGetPDWatchInterval.String(), &flagGetPDWatchInterval), "watch-interval", "interval between process-definition watch refreshes after the immediate first refresh")

	setCommandMutation(getProcessDefinitionCmd, CommandMutationReadOnly)
	setContractSupport(getProcessDefinitionCmd, ContractSupportFull)
	setOutputModes(getProcessDefinitionCmd,
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: true,
			Notes:     "default watch refreshes repaint this normal list view; --watch rejects JSON/keys-only/XML/quiet/automation combinations",
		},
		OutputModeContract{
			Name:             RenderModeJSON.String(),
			Supported:        true,
			MachinePreferred: true,
			Notes:            "preferred for automation when not using --xml or --watch",
		},
		OutputModeContract{
			Name:      RenderModeKeysOnly.String(),
			Supported: true,
			Notes:     "finite key stream for non-watch invocations; --watch rejects keys-only output",
		},
	)
}

func validateGetProcessDefinitionFlags(cmd *cobra.Command) error {
	if cmd != nil && cmd.Flags().Changed("batch-size") && (flagGetPDBatchSize <= 0 || flagGetPDBatchSize > consts.MaxPISearchSize) {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagGetPDBatchSize, consts.MaxPISearchSize)
	}
	if flagGetPDWatch {
		if err := validateGetProcessDefinitionWatchOutputFlags(cmd); err != nil {
			return err
		}
		if _, err := resolveGetProcessDefinitionWatchInterval(); err != nil {
			return err
		}
	}
	return nil
}

// validateGetProcessDefinitionWatchOutputFlags keeps watch repaint output out of
// finite machine-output contracts before any process-definition lookup starts.
func validateGetProcessDefinitionWatchOutputFlags(cmd *cobra.Command) error {
	var incompatible []string
	for _, check := range []struct {
		enabled bool
		flag    string
	}{
		{enabled: flagViewAsJson, flag: "--json"},
		{enabled: flagViewKeysOnly, flag: "--keys-only"},
		{enabled: flagGetPDAsXML, flag: "--xml"},
		{enabled: flagQuiet, flag: "--quiet"},
		{enabled: automationModeEnabled(cmd), flag: "--automation"},
	} {
		if check.enabled {
			incompatible = append(incompatible, check.flag)
		}
	}
	if len(incompatible) == 0 {
		return nil
	}
	return forbiddenFlagCombinationf("--watch cannot be combined with %s; watch repaints terminal output", strings.Join(incompatible, ", "))
}

func populatePDSearchFilterOpts() process.ProcessDefinitionFilter {
	var filter process.ProcessDefinitionFilter
	if flagGetPDKey != "" {
		filter.Key = flagGetPDKey
	}
	if flagGetPDBpmnProcessId != "" {
		filter.BpmnProcessId = flagGetPDBpmnProcessId
	}
	if flagGetPDProcessVersion != 0 {
		filter.ProcessVersion = flagGetPDProcessVersion
	}
	if flagGetPDProcessVersionTag != "" {
		filter.ProcessVersionTag = flagGetPDProcessVersionTag
	}
	return filter
}

func searchProcessDefinitionsWithPaging(cmd *cobra.Command, cli c8volt.API, filter process.ProcessDefinitionFilter) (process.ProcessDefinitions, error) {
	if flagGetPDWithStat {
		stopActivity := startCommandActivity(cmd, "loading process-definition statistics")
		defer stopActivity()
	}

	pageNumber := 0
	result, err := cli.SearchProcessDefinitionsPages(cmd.Context(), process.ProcessDefinitionSearchRequest{
		Filter: filter,
		Page: process.ProcessDefinitionPageRequest{
			Size: resolveGetProcessDefinitionSearchSize(),
		},
		Latest: flagGetPDLatest,
	}, func(step process.ProcessDefinitionSearchPageStep) (process.ProcessDefinitionSearchPageAction, error) {
		page := step.Page
		pageNumber++
		total, totalKind := processDefinitionOpsReportedTotal(page)
		printBasicSearchOpsProgress(cmd, basicSearchProgressMetadata{
			Command:            "get process-definition",
			CoreResource:       "process_definition",
			ResourceLabel:      "process definitions",
			SelectorSummary:    "process-definition search",
			ConsequenceSummary: "process-definition search will discover and render matching process definitions",
			ReportedTotal:      total,
			TotalKind:          totalKind,
			OverflowState:      processDefinitionOpsOverflowState(page.OverflowState),
			PageSize:           page.Request.Size,
			CurrentPage:        pageNumber,
			CurrentPageCount:   len(page.Items),
			CumulativeCount:    int(step.CumulativeCount),
			LimitReached:       step.LimitReached,
		}, pageNumber == 1)
		return process.ProcessDefinitionSearchPageActionContinue, nil
	}, collectOptions()...)
	if err != nil {
		return process.ProcessDefinitions{}, err
	}
	return process.ProcessDefinitions{Total: int32(len(result.Items)), Items: result.Items}, nil
}

func resolveGetProcessDefinitionSearchSize() int32 {
	if flagGetPDBatchSize <= 0 || flagGetPDBatchSize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return flagGetPDBatchSize
}

func processDefinitionOpsReportedTotal(page process.ProcessDefinitionPage) (*int64, ops.TotalCertainty) {
	if page.ReportedTotal == nil {
		return nil, ops.TotalCertaintyUnknown
	}
	total := page.ReportedTotal.Count
	switch page.ReportedTotal.Kind {
	case process.ProcessDefinitionReportedTotalKindExact:
		return &total, ops.TotalCertaintyExact
	case process.ProcessDefinitionReportedTotalKindLowerBound:
		return &total, ops.TotalCertaintyLowerBound
	default:
		return &total, ops.TotalCertaintyUnknown
	}
}

func processDefinitionOpsOverflowState(state process.ProcessInstanceOverflowState) ops.OverflowState {
	switch state {
	case process.ProcessInstanceOverflowStateHasMore:
		return ops.OverflowStateHasMore
	case process.ProcessInstanceOverflowStateIndeterminate:
		return ops.OverflowStateIndeterminate
	case process.ProcessInstanceOverflowStateNoMore:
		return ops.OverflowStateNoMore
	default:
		return ops.OverflowStateUnknown
	}
}

func validateProcessDefinitionXMLFlags(filter process.ProcessDefinitionFilter) error {
	if filter.Key == "" {
		return missingDependentFlagsf("xml output requires --key to select a single process definition")
	}

	var incompatible []string
	for _, check := range []struct {
		enabled bool
		flag    string
	}{
		{enabled: filter.BpmnProcessId != "", flag: "--bpmn-process-id"},
		{enabled: flagGetPDProcessVersion != 0, flag: "--pd-version"},
		{enabled: filter.ProcessVersionTag != "", flag: "--pd-version-tag"},
		{enabled: flagGetPDLatest, flag: "--latest"},
		{enabled: flagGetPDWithStat, flag: "--stat"},
		{enabled: flagViewAsJson, flag: "--json"},
		{enabled: flagViewKeysOnly, flag: "--keys-only"},
	} {
		if check.enabled {
			incompatible = append(incompatible, check.flag)
		}
	}
	if len(incompatible) > 0 {
		return forbiddenFlagCombinationf("xml output only supports --key; incompatible with %s", strings.Join(incompatible, ", "))
	}

	return nil
}
