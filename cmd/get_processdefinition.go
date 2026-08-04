// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/watch"
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

const defaultGetPDWatchInterval = time.Second

var processDefinitionWatchSleep watch.SleepFunc

var getProcessDefinitionCmd = &cobra.Command{
	Use:   "process-definition",
	Short: "List or fetch deployed process definitions",
	Long: `List or fetch deployed process definitions.

Inspect deployed BPMN models by key, BPMN process ID, version selectors, or
latest deployed version. Use ` + "`--xml`" + ` only with ` + "`--key`" + `.

Tenant contract: ` + "`--tenant`" + ` scopes list/latest and BPMN selector discovery where
supported. Explicit ` + "`--key`" + ` and XML key lookups are backend-authorized admin input;
c8volt displays returned tenant metadata without rejecting solely because it differs
from the selected tenant.

Watch mode prints repeated terminal snapshots, starting immediately and then
waiting ` + "`1s`" + ` between snapshots unless ` + "`--watch-interval`" + ` is set. Without a
selector, ` + "`--watch`" + ` observes all visible process definitions. JSON, keys-only,
XML, quiet, and automation combinations are rejected before lookup work.
Existing timeout and backoff retry settings bound the watch run; successful
snapshots reset the consecutive retry budget.

When ` + "`--bpmn-process-id`" + ` is set, c8volt validates that at least one visible
process definition matches the selector before rendering output. A missing selector
fails with the shared local diagnostic instead of rendering an ambiguous empty list.

` + "`--stat`" + ` requires Camunda ` + "`8.8`" + ` or ` + "`8.9`" + ` and prints exact-version
counts. Camunda ` + "`8.7`" + ` does not support native statistics.`,
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
		ferrors.HandleAndExit(log, noErrCodes, err)
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
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("get process definition: %w", err))
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
			ferrors.HandleAndExit(log, noErrCodes, err)
		}
		if !result.Valid() {
			handleProcessDefinitionSelectorValidationError(cmd, log, noErrCodes, cli, result)
		}
		if len(result.Request.BpmnProcessIds) > 0 {
			pds = result.MatchesByBpmnProcessID[result.Request.BpmnProcessIds[0]]
		}
	} else if !flagGetPDLatest {
		pds, err = searchProcessDefinitionsWithPaging(cmd, cli, filter)
	} else {
		pds, err = cli.SearchProcessDefinitionsLatest(cmd.Context(), filter, collectOptions()...)
	}
	if err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("search process definitions: %w", err))
	}
	if err := listProcessDefinitionsView(cmd, pds); err != nil {
		ferrors.HandleAndExit(log, noErrCodes, fmt.Errorf("error rendering items view: %w", err))
	}
	log.Debug(fmt.Sprintf("pd search done; found %d", pds.Total))
}

func runGetProcessDefinitionWatch(cmd *cobra.Command, cli c8volt.API, log *slog.Logger, noErrCodes bool, filter process.ProcessDefinitionFilter, timeout time.Duration, maxRetries int) {
	log.Debug(fmt.Sprintf("watching pd; filter %s", filter.String()))
	if err := executeGetProcessDefinitionWatch(cmd, cli, filter, timeout, maxRetries); err != nil {
		handleCommandError(cmd, log, noErrCodes, fmt.Errorf("watch process definitions: %w", err))
	}
}

func executeGetProcessDefinitionWatch(cmd *cobra.Command, cli c8volt.API, filter process.ProcessDefinitionFilter, timeout time.Duration, maxRetries int) error {
	interval, err := resolveGetProcessDefinitionWatchInterval()
	if err != nil {
		return err
	}
	request := newGetProcessDefinitionWatchSnapshotRequest(filter)
	opts := collectOptions()
	if request.Key != "" {
		opts = collectExplicitAdminInputOptions()
	}
	result, err := watch.Run(cmd.Context(), watch.Options{
		Interval:   interval,
		Timeout:    timeout,
		MaxRetries: maxRetries,
		Retryable:  isGetProcessDefinitionWatchRetryable,
		Sleep:      processDefinitionWatchSleep,
		OnRetry: func(event watch.RetryEvent) {
			renderGetProcessDefinitionWatchRetryStatus(cmd, event)
		},
	}, func(ctx context.Context, tick watch.Tick) error {
		snapshot, err := cli.CollectProcessDefinitionWatchSnapshot(ctx, request, opts...)
		if err != nil {
			return err
		}
		return processDefinitionWatchSnapshotView(cmd, tick.Index, snapshot)
	})
	renderGetProcessDefinitionWatchStopStatus(cmd, result)
	return err
}

func resolveGetProcessDefinitionWatchInterval() (time.Duration, error) {
	interval, err := time.ParseDuration(flagGetPDWatchInterval)
	if err != nil {
		return 0, invalidFlagValuef("invalid value for --watch-interval: %q, expected positive duration such as 1s, 2s, or 500ms", flagGetPDWatchInterval)
	}
	if interval <= 0 {
		return 0, invalidFlagValuef("invalid value for --watch-interval: %q, expected positive duration", flagGetPDWatchInterval)
	}
	return interval, nil
}

func isGetProcessDefinitionWatchRetryable(err error) bool {
	switch ferrors.Classify(err) {
	case ferrors.ClassTimeout, ferrors.ClassUnavailable:
		return true
	default:
		return false
	}
}

func renderGetProcessDefinitionWatchRetryStatus(cmd *cobra.Command, event watch.RetryEvent) {
	if cmd == nil {
		return
	}
	if event.MaxRetries > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "retrying process-definition watch after snapshot %d failed (%d/%d consecutive failures): %v\n", event.Tick, event.ConsecutiveFailures, event.MaxRetries, event.Err)
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "retrying process-definition watch after snapshot %d failed (consecutive failures: %d): %v\n", event.Tick, event.ConsecutiveFailures, event.Err)
}

func renderGetProcessDefinitionWatchStopStatus(cmd *cobra.Command, result watch.Result) {
	if cmd == nil {
		return
	}
	switch result.Reason {
	case watch.TerminationReasonTimeout:
		fmt.Fprintln(cmd.ErrOrStderr(), "watch stopped: timeout reached")
	case watch.TerminationReasonRetryExhausted:
		fmt.Fprintf(cmd.ErrOrStderr(), "watch stopped: retry budget exhausted after %d consecutive failure(s)\n", result.ConsecutiveFailures)
	}
}

func newGetProcessDefinitionWatchSnapshotRequest(filter process.ProcessDefinitionFilter) process.ProcessDefinitionWatchSnapshotRequest {
	request := process.ProcessDefinitionWatchSnapshotRequest{
		Key:    filter.Key,
		Filter: filter,
		Page: process.ProcessDefinitionPageRequest{
			Size: resolveGetProcessDefinitionSearchSize(),
		},
		Latest: flagGetPDLatest,
	}
	request.WatchAllWhenUnselected = request.Key == "" &&
		filter.BpmnProcessId == "" &&
		filter.ProcessVersion == 0 &&
		filter.ProcessVersionTag == "" &&
		!request.Latest
	return request
}

func init() {
	getCmd.AddCommand(getProcessDefinitionCmd)

	fs := getProcessDefinitionCmd.Flags()
	fs.StringVarP(&flagGetPDKey, "key", "k", "", "process definition key to fetch")
	fs.StringVarP(&flagGetPDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.BoolVar(&flagGetPDLatest, "latest", false, "fetch the latest version(s) of the given BPMN process(s)")
	fs.Int32Var(&flagGetPDProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagGetPDWithStat, "stat", false, "include process definition statistics; 8.8/8.9 include incident counts, 8.7 unsupported")
	fs.BoolVar(&flagGetPDAsXML, "xml", false, "output the selected process definition as raw XML (requires --key and no other filters)")
	fs.Int32VarP(&flagGetPDBatchSize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process definitions to request per discovery page; does not cap total returned rows (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.BoolVar(&flagGetPDWatch, "watch", false, "repeat the process-definition lookup as terminal snapshots until interrupted, timed out, or retry-exhausted")
	fs.Var(toolx.NewDurationStringValue(defaultGetPDWatchInterval.String(), &flagGetPDWatchInterval), "watch-interval", "interval between process-definition watch snapshots after the immediate first snapshot")

	setCommandMutation(getProcessDefinitionCmd, CommandMutationReadOnly)
	setContractSupport(getProcessDefinitionCmd, ContractSupportFull)
	setOutputModes(getProcessDefinitionCmd,
		OutputModeContract{
			Name:      RenderModeOneLine.String(),
			Supported: true,
			Notes:     "default watch snapshots use this mode; --watch rejects JSON/keys-only/XML/quiet/automation combinations",
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

// validateGetProcessDefinitionWatchOutputFlags keeps watch snapshots out of
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
	return forbiddenFlagCombinationf("--watch cannot be combined with %s; watch prints repeated terminal snapshots", strings.Join(incompatible, ", "))
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
