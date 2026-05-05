// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	flagGetPIKeys                 []string
	flagGetPIHasUserTasks         []string
	flagGetPIBpmnProcessID        string
	flagGetPIProcessVersion       int32
	flagGetPIProcessVersionTag    string
	flagGetPIProcessDefinitionKey string
	flagGetPIStartDateAfter       string
	flagGetPIStartDateBefore      string
	flagGetPIEndDateAfter         string
	flagGetPIEndDateBefore        string
	flagGetPIStartAfterDays       int
	flagGetPIStartBeforeDays      int
	flagGetPIEndAfterDays         int
	flagGetPIEndBeforeDays        int
	flagGetPITotal                bool
	flagGetPIState                string
	flagGetPIParentKey            string
	flagGetPISize                 int32
	flagGetPILimit                int32
	flagGetPIWithIncidents        bool
	flagGetPIIncidentMessageLimit int
)

// command options
var (
	flagGetPIRootsOnly          bool
	flagGetPIChildrenOnly       bool
	flagGetPIOrphanChildrenOnly bool
	flagGetPIIncidentsOnly      bool
	flagGetPINoIncidentsOnly    bool
)

var getProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "List or fetch process instances",
	Long: "Get process instances by key or by search criteria.\n\n" +
		"Use direct lookup when you know a process-instance key, or combine search filters to inspect matching process instances by process definition, tenant, state, incidents, variables, jobs, user tasks, and time ranges.\n\n" +
		"Search results support interactive paging, scriptable JSON aggregation, and count-only workflows. Direct key lookup stays strict: missing keys return not-found.\n\n" +
		"Use --with-incidents with keyed or list/search output to include direct incident keys and messages under matching process-instance rows. Add --incident-message-limit <chars> to shorten human incident messages; JSON keeps full incident messages.\n\n" +
		"User-task based lookup resolves owning process instances through tenant-aware Camunda v2 user-task search first. On Camunda 8.8 and 8.9, not-found user-task results fall back to deprecated Tasklist V1 lookup for legacy user-task compatibility; Camunda 8.7 remains unsupported.\n\n" +
		"Run `c8volt get pi --help` for the complete flag reference.",
	Example: `  ./c8volt get pi --bpmn-process-id <bpmn-process-id> --state active
  ./c8volt get pi --key <process-instance-key>
  ./c8volt get pi --state active
  ./c8volt get pi --state active --json
  ./c8volt get pi --state active --total
  ./c8volt get pi --has-user-tasks <user-task-key>
  ./c8volt get pi --state active --batch-size 250 --limit 25
  ./c8volt get pi --state active --limit 25 --auto-confirm
  ./c8volt get pi --incidents-only --with-incidents
  ./c8volt get pi --with-incidents --incident-message-limit 80
  ./c8volt get pi --key 2251799813711967 --with-incidents
  ./c8volt get pi --key 2251799813711967 --json
  ./c8volt get pi --key 2251799813711967 --with-incidents --json
  ./c8volt get pi --start-date-after 2026-01-01 --start-date-before 2026-01-31
  ./c8volt get pi --key 2251799813711967 --key 2251799813711977`,
	Aliases: []string{"process-instances", "pi", "pis"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		ctx := cmd.Context()
		fail := func(err error) {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			fail(invalidFlagValuef("--workers must be positive integer"))
		}
		if err := validatePISearchFlags(cmd); err != nil {
			fail(err)
		}
		filterFlagsSet := hasPISearchFilterFlags()

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			fail(err)
		}
		keys := mergeAndValidateKeys(flagGetPIKeys, stdinKeys, log, cfg)
		ukeys := keys.Unique()
		lk := len(ukeys)
		taskKeys, err := normalizeHasUserTasks(types.Keys(flagGetPIHasUserTasks))
		if err != nil {
			fail(err)
		}
		ltk := len(taskKeys)
		if err := validatePIHasUserTasksMode(cmd, ltk, lk, filterFlagsSet); err != nil {
			fail(err)
		}
		if err := validatePIWithIncidentsUsage(lk, filterFlagsSet); err != nil {
			fail(err)
		}
		if lk == 0 && ltk == 0 {
			if err := validatePISearchVersionSupport(cfg); err != nil {
				fail(err)
			}
		}

		log.Debug(fmt.Sprintf("fetching process instances, render mode: %s", pickMode()))
		var pis process.ProcessInstances
		switch {
		case ltk > 0:
			log.Debug(fmt.Sprintf("resolving process instance key(s) from user task key(s) [%s]", taskKeys))
			processInstanceKeys, err := cli.ResolveProcessInstanceKeysFromUserTasks(ctx, taskKeys, collectOptions()...)
			if err != nil {
				fail(fmt.Errorf("resolve process instance key(s) from user task key(s): %w", err))
			}
			processInstanceKeys = processInstanceKeys.Unique()
			wantedWorkers := len(processInstanceKeys)
			if cmd.Flags().Changed("workers") {
				wantedWorkers = flagWorkers
			}
			pis, err = cli.GetProcessInstances(ctx, processInstanceKeys, wantedWorkers, collectOptions()...)
			if err != nil {
				fail(fmt.Errorf("get process instance(s) resolved from user task key(s) [%s]: %w", taskKeys, err))
			}
		case lk > 0:
			log.Debug(fmt.Sprintf("searching for key(s) [%s]", keys))
			if err := validatePIKeyedModeDateFilters(lk); err != nil {
				fail(err)
			}
			if err := validatePIKeyedModeLimit(lk); err != nil {
				fail(err)
			}
			if flagGetPITotal {
				fail(mutuallyExclusiveFlagsf("--total cannot be combined with --key"))
			}
			// Keyed lookups intentionally stay strict; partial orphan warnings are only for traversal/preflight flows.
			if filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPINoIncidentsOnly {
				fail(mutuallyExclusiveFlagsf("--key cannot be combined with other filters"))
			}
			if cmd.Flags().Changed("workers") {
				lk = flagWorkers
			}
			pis, err = cli.GetProcessInstances(ctx, ukeys, lk, collectOptions()...)
			if err != nil {
				msg := fmt.Errorf("get process instances: %w", err)
				if flagVerbose {
					msg = fmt.Errorf("get process instances for key(s) [%s]: %w", ukeys, err)
				}
				fail(msg)
			}
			if flagGetPIWithIncidents {
				enriched, err := cli.EnrichProcessInstancesWithIncidents(ctx, pis, collectOptions()...)
				if err != nil {
					fail(fmt.Errorf("get process instance incidents: %w", err))
				}
				if err := incidentEnrichedProcessInstancesView(cmd, enriched); err != nil {
					fail(fmt.Errorf("render process instances with incidents: %w", err))
				}
				return
			}
		default:
			filter := populatePISearchFilterOpts()
			log.Debug(fmt.Sprintf("using process instance search filter: %s", filter.String()))
			if flagGetPITotal {
				total, err := searchProcessInstancesTotal(cmd, log, cli, cfg, filter)
				if err != nil {
					fail(fmt.Errorf("get process instances total: %w", err))
				}
				if err := processInstanceTotalView(cmd, total); err != nil {
					fail(fmt.Errorf("render process instance total: %w", err))
				}
				return
			}
			var renderedIncrementally bool
			pis, renderedIncrementally, err = searchProcessInstancesWithPaging(cmd, cli, cfg, filter)
			if err != nil {
				fail(fmt.Errorf("get process instances: %w", err))
			}
			if renderedIncrementally {
				return
			}
		}
		if flagGetPIWithIncidents {
			enriched, err := cli.EnrichProcessInstancesWithIncidents(ctx, pis, collectOptions()...)
			if err != nil {
				fail(fmt.Errorf("get process instance incidents: %w", err))
			}
			if err := incidentEnrichedProcessInstancesView(cmd, enriched); err != nil {
				fail(fmt.Errorf("render process instances with incidents: %w", err))
			}
			return
		}
		if err := listProcessInstancesView(cmd, pis); err != nil {
			fail(fmt.Errorf("render process instances: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getProcessInstanceCmd)
	useInvalidInputFlagErrors(getProcessInstanceCmd)

	fs := getProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagGetPIKeys, "key", "k", nil, "process instance key(s) to fetch")
	fs.StringSliceVar(&flagGetPIHasUserTasks, "has-user-tasks", nil, "user task key(s) whose owning process instances should be fetched")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	registerPISharedDateRangeFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to fetch per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching process instances to return or process across all pages")
	fs.BoolVar(&flagGetPITotal, "total", false, "return only the numeric total of matching process instances; capped backend totals are counted by paging")
	fs.BoolVar(&flagGetPIWithIncidents, "with-incidents", false, "include direct incident keys and messages for keyed or list/search process-instance output")
	fs.IntVar(&flagGetPIIncidentMessageLimit, "incident-message-limit", 0, "maximum characters to show for human incident messages when --with-incidents is set; 0 disables truncation")

	// filtering options
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to filter process instances")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")

	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "show only root process instances")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "show only child process instances")

	fs.BoolVar(&flagGetPIOrphanChildrenOnly, "orphan-children-only", false, "show only child instances with missing parents")

	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --batch-size > 1 (default: min(batch-size, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(getProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(getProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(getProcessInstanceCmd, AutomationSupportFull, "supports unattended confirmation-free paging and shared machine output")
}

var relativeDayNow = func() time.Time {
	return time.Now().UTC()
}

type processInstanceContinuationState string

const (
	processInstanceContinuationPrompt          processInstanceContinuationState = "prompt"
	processInstanceContinuationAutoContinue    processInstanceContinuationState = "auto_continue"
	processInstanceContinuationCompleted       processInstanceContinuationState = "completed"
	processInstanceContinuationPartialComplete processInstanceContinuationState = "partial_complete"
	processInstanceContinuationWarningStop     processInstanceContinuationState = "warning_stop"
	processInstanceContinuationLimitReached    processInstanceContinuationState = "limit_reached"
)

// processInstanceProgressSummary describes the current pagination state for user-facing progress output.
//
// It is used to render one-line progress diagnostics (in verbose mode) and to drive continuation behavior
// such as prompting, auto-continue, warning stop, and partial-complete reporting.
type processInstanceProgressSummary struct {
	// PageSize is the configured request size used for the current page fetch.
	PageSize int32
	// CurrentPageCount is the number of process instances returned on this page.
	CurrentPageCount int
	// CumulativeCount is the total number of process instances processed/collected so far.
	CumulativeCount int
	// OverflowState indicates whether additional matching items are known, unknown, or exhausted.
	OverflowState process.ProcessInstanceOverflowState
	// ContinuationState determines the next paging action (prompt/auto-continue/complete/etc.).
	ContinuationState processInstanceContinuationState
}

// processInstancePageImpact captures per-page impact counts used by cancel/delete paging prompts.
//
// These values are accumulated across pages to present users with a continuation prompt that reflects
// both the visible page size and the real operational impact when dependencies are included.
type processInstancePageImpact struct {
	// Requested is the raw number of keys selected from the current search page.
	Requested int
	// Affected is the expanded number of instances impacted after dependency resolution.
	Affected int
	// Roots is the number of root instances in the expanded impact set.
	Roots int
}

type processInstancePageActionResult struct {
	Impact        processInstancePageImpact
	Reports       []process.Reporter
	DryRunPreview *processInstanceDryRunPreview
}

type processInstancePageActionResults struct {
	Reports        []process.Reporter
	DryRunPreviews []processInstanceDryRunPreview
}

func registerPISharedDateRangeFlags(fs *pflag.FlagSet) {
	fs.StringVar(&flagGetPIStartDateAfter, "start-date-after", "", "only include process instances with start date >= YYYY-MM-DD")
	fs.StringVar(&flagGetPIStartDateBefore, "start-date-before", "", "only include process instances with start date <= YYYY-MM-DD")
	fs.StringVar(&flagGetPIEndDateAfter, "end-date-after", "", "only include process instances with end date >= YYYY-MM-DD")
	fs.StringVar(&flagGetPIEndDateBefore, "end-date-before", "", "only include process instances with end date <= YYYY-MM-DD")

	fs.IntVar(&flagGetPIStartAfterDays, "start-date-older-days", -1, "only include process instances N days old or older")
	fs.IntVar(&flagGetPIStartBeforeDays, "start-date-newer-days", -1, "only include process instances N days old or newer (0 means today)")
	fs.IntVar(&flagGetPIEndAfterDays, "end-date-older-days", -1, "only include process instances with end date N days old or older")
	fs.IntVar(&flagGetPIEndBeforeDays, "end-date-newer-days", -1, "only include process instances with end date N days old or newer (0 means today)")
}

func registerPISharedProcessDefinitionFilterFlags(fs *pflag.FlagSet) {
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32Var(&flagGetPIProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "pd-version-tag", "", "process definition version tag")
}

func populatePISearchFilterOpts() process.ProcessInstanceFilter {
	f := process.ProcessInstanceFilter{
		ParentKey:            flagGetPIParentKey,
		BpmnProcessId:        flagGetPIBpmnProcessID,
		ProcessVersion:       flagGetPIProcessVersion,
		ProcessVersionTag:    flagGetPIProcessVersionTag,
		ProcessDefinitionKey: flagGetPIProcessDefinitionKey,
		StartDateAfter:       pickPIDateBound(flagGetPIStartDateAfter, flagGetPIStartBeforeDays),
		StartDateBefore:      pickPIDateUpperBound(flagGetPIStartDateBefore, flagGetPIStartAfterDays),
		EndDateAfter:         pickPIDateBound(flagGetPIEndDateAfter, flagGetPIEndBeforeDays),
		EndDateBefore:        pickPIDateUpperBound(flagGetPIEndDateBefore, flagGetPIEndAfterDays),
	}

	if s := flagGetPIState; s != "" && s != "all" {
		if st, ok := process.ParseState(s); ok {
			f.State = st
		}
	}
	if flagGetPIChildrenOnly {
		f.HasParent = new(true)
	}
	if flagGetPIRootsOnly {
		f.HasParent = new(false)
	}
	if flagGetPIIncidentsOnly {
		f.HasIncident = new(true)
	}
	if flagGetPINoIncidentsOnly {
		f.HasIncident = new(false)
	}
	return f
}

func hasPISearchFilterFlags() bool {
	return flagGetPIParentKey != "" ||
		flagGetPIBpmnProcessID != "" ||
		flagGetPIProcessVersion != 0 ||
		flagGetPIProcessVersionTag != "" ||
		flagGetPIProcessDefinitionKey != "" ||
		hasPIDateFilterFlags() ||
		hasPIRelativeDayFilterFlags() ||
		(flagGetPIState != "" && flagGetPIState != "all")
}

func hasPIDateFilterFlags() bool {
	return flagGetPIStartDateAfter != "" ||
		flagGetPIStartDateBefore != "" ||
		flagGetPIEndDateAfter != "" ||
		flagGetPIEndDateBefore != ""
}

func hasPIRelativeDayFilterFlags() bool {
	return flagGetPIStartAfterDays >= 0 ||
		flagGetPIStartBeforeDays >= 0 ||
		flagGetPIEndAfterDays >= 0 ||
		flagGetPIEndBeforeDays >= 0
}

func validatePIKeyedModeDateFilters(keyCount int) error {
	if keyCount > 0 && (hasPIDateFilterFlags() || hasPIRelativeDayFilterFlags()) {
		return mutuallyExclusiveFlagsf("date filters are only supported for list/search usage and cannot be combined with --key")
	}
	return nil
}

func pickPISearchSize() int32 {
	if flagGetPISize <= 0 || flagGetPISize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return flagGetPISize
}

func resolvePISearchSize(cmd *cobra.Command, cfg *config.Config) int32 {
	if cmd != nil && cmd.Flags().Changed("batch-size") {
		return pickPISearchSize()
	}
	if cfg != nil && cfg.App.ProcessInstancePageSize > 0 && cfg.App.ProcessInstancePageSize <= consts.MaxPISearchSize {
		return cfg.App.ProcessInstancePageSize
	}
	return consts.MaxPISearchSize
}

// newPISearchPageRequest builds the process-instance page request for the current command and config.
func newPISearchPageRequest(cmd *cobra.Command, cfg *config.Config, from int32) process.ProcessInstancePageRequest {
	return process.ProcessInstancePageRequest{
		From: from,
		Size: resolvePISearchSize(cmd, cfg),
	}
}

func pickPIContinuationState(overflow process.ProcessInstanceOverflowState, autoConfirm bool) processInstanceContinuationState {
	switch overflow {
	case process.ProcessInstanceOverflowStateHasMore:
		if autoConfirm {
			return processInstanceContinuationAutoContinue
		}
		return processInstanceContinuationPrompt
	case process.ProcessInstanceOverflowStateIndeterminate:
		return processInstanceContinuationWarningStop
	default:
		return processInstanceContinuationCompleted
	}
}

// newPIProgressSummary converts page metadata into user-facing continuation progress.
func newPIProgressSummary(page process.ProcessInstancePage, cumulative int, autoConfirm bool) processInstanceProgressSummary {
	continuationState := pickPIContinuationState(page.OverflowState, autoConfirm)
	if isPILimitReached(cumulative) {
		continuationState = processInstanceContinuationLimitReached
	}
	return processInstanceProgressSummary{
		PageSize:          page.Request.Size,
		CurrentPageCount:  len(page.Items),
		CumulativeCount:   cumulative,
		OverflowState:     page.OverflowState,
		ContinuationState: continuationState,
	}
}

func isPILimitReached(cumulative int) bool {
	return flagGetPILimit > 0 && cumulative >= int(flagGetPILimit)
}

func limitPIItems(items []process.ProcessInstance, cumulative int) []process.ProcessInstance {
	if flagGetPILimit <= 0 {
		return items
	}
	remaining := int(flagGetPILimit) - cumulative
	if remaining <= 0 {
		return nil
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

func limitPIPageItems(page process.ProcessInstancePage, cumulative int) process.ProcessInstancePage {
	page.Items = limitPIItems(page.Items, cumulative)
	return page
}

func searchProcessInstancesWithPaging(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (process.ProcessInstances, bool, error) {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	var collected process.ProcessInstances
	incremental := shouldRenderPISearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinuePISearchPages(cmd)
	processedTotal := 0
	needsIndirectIncidentWarning := false
	printFoundAndReturn := func() (process.ProcessInstances, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine {
				if needsIndirectIncidentWarning {
					renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
				}
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return process.ProcessInstances{}, true, nil
		}
		return collected, false, nil
	}

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return process.ProcessInstances{}, false, err
		}

		filtered, err := applyPISearchResultFilters(cmd, cli, process.ProcessInstances{
			Total: int32(len(page.Items)),
			Items: page.Items,
		})
		if err != nil {
			return process.ProcessInstances{}, false, err
		}
		filtered.Items = limitPIItems(filtered.Items, processedTotal)
		filtered.Total = int32(len(filtered.Items))
		if incremental {
			if flagGetPIWithIncidents && pickMode() == RenderModeOneLine {
				enriched, err := cli.EnrichProcessInstancesWithIncidents(cmd.Context(), filtered, collectOptions()...)
				if err != nil {
					return process.ProcessInstances{}, false, fmt.Errorf("get process instance incidents: %w", err)
				}
				pageNeedsIndirectIncidentWarning, err := renderIncidentEnrichedProcessInstanceRows(cmd, enriched)
				if err != nil {
					return process.ProcessInstances{}, false, err
				}
				needsIndirectIncidentWarning = needsIndirectIncidentWarning || pageNeedsIndirectIncidentWarning
			} else if pickMode() == RenderModeOneLine {
				if err := renderProcessInstanceFlatRows(cmd, filtered.Items); err != nil {
					return process.ProcessInstances{}, false, err
				}
			} else {
				for _, item := range filtered.Items {
					if err := processInstanceView(cmd, item); err != nil {
						return process.ProcessInstances{}, false, err
					}
				}
			}
		} else {
			collected.Items = append(collected.Items, filtered.Items...)
			collected.Total = int32(len(collected.Items))
		}
		processedTotal += len(filtered.Items)

		summaryPage := page
		summaryPage.Items = filtered.Items
		summary := newPIProgressSummary(summaryPage, processedTotal, autoContinue)
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return printFoundAndReturn()
		case processInstanceContinuationAutoContinue:
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
			continue
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Fetched %d process instance(s) on this page (%d total so far). More matching process instances remain. Continue?", summary.CurrentPageCount, summary.CumulativeCount)
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return printFoundAndReturn()
				}
				return process.ProcessInstances{}, false, err
			}
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
		}
	}
}

func searchProcessInstancesTotal(cmd *cobra.Command, log *slog.Logger, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (int64, error) {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	total := int64(0)
	stopActivity := func() {}
	countingByPaging := false
	defer func() {
		stopActivity()
	}()

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return 0, err
		}
		logPITotalPage(cmd, log, pageReq, page, total)

		if canUsePIExactReportedTotal(page) {
			return page.ReportedTotal.Count, nil
		}
		if !countingByPaging {
			stopActivity = startCommandActivity(cmd, "counting process instances page by page")
			countingByPaging = true
		}

		filtered, err := applyPISearchResultFilters(cmd, cli, process.ProcessInstances{
			Total: int32(len(page.Items)),
			Items: page.Items,
		})
		if err != nil {
			return 0, err
		}

		total += int64(len(filtered.Items))
		logPISearchProgress(cmd, log, newPIProgressSummary(page, int(total), true))

		if len(page.Items) == 0 || page.OverflowState == process.ProcessInstanceOverflowStateNoMore {
			return total, nil
		}
		pageReq = nextPISearchPageRequest(cmd, cfg, pageReq, page)
	}
}

func logPITotalPage(cmd *cobra.Command, log *slog.Logger, req process.ProcessInstancePageRequest, page process.ProcessInstancePage, totalBefore int64) {
	if cmd == nil || log == nil {
		return
	}
	mode := "offset"
	if req.After != "" {
		mode = "cursor"
	}
	reportedTotal := int64(-1)
	reportedKind := "unavailable"
	if page.ReportedTotal != nil {
		reportedTotal = page.ReportedTotal.Count
		reportedKind = string(page.ReportedTotal.Kind)
	}
	log.DebugContext(cmd.Context(), fmt.Sprintf(
		"process-instance total page: mode=%s, from=%d, after=%q, limit=%d, items=%d, total before=%d, total after=%d, overflow=%s, reported total=%d, reported kind=%s, end cursor=%q",
		mode,
		req.From,
		req.After,
		req.Size,
		len(page.Items),
		totalBefore,
		totalBefore+int64(len(page.Items)),
		page.OverflowState,
		reportedTotal,
		reportedKind,
		page.EndCursor,
	))
}

func shouldRenderPISearchPageIncrementally(cmd *cobra.Command) bool {
	if flagCmdAutoConfirm {
		return false
	}
	mode := pickMode()
	if automationModeEnabled(cmd) {
		return mode == RenderModeOneLine || mode == RenderModeKeysOnly
	}
	return mode == RenderModeOneLine || mode == RenderModeKeysOnly
}

// shouldAutoContinuePISearchPages reports whether paged process-instance search should
// consume additional pages without interactive confirmation.
func shouldAutoContinuePISearchPages(cmd *cobra.Command) bool {
	return shouldImplicitlyConfirm(cmd) || pickMode() == RenderModeJSON
}

func isCmdAborted(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrCmdAborted) {
		return true
	}
	if errors.Is(err, ferrors.ErrLocalPrecondition) && strings.Contains(err.Error(), ErrCmdAborted.Error()) {
		return true
	}
	return false
}

func processPISearchPagesWithAction(
	cmd *cobra.Command,
	cli process.API,
	cfg *config.Config,
	filter process.ProcessInstanceFilter,
	processPage func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error),
) (processInstancePageActionResults, error) {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	cumulative := 0
	cumulativeAffected := 0
	firstPage := true
	var results processInstancePageActionResults

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return processInstancePageActionResults{}, err
		}
		if len(page.Items) == 0 {
			if cumulative == 0 {
				renderOutputLine(cmd, "found: %d", 0)
			}
			return results, nil
		}

		limitedPage := limitPIPageItems(page, cumulative)
		result, err := processPage(limitedPage, firstPage)
		if err != nil {
			if !firstPage && isCmdAborted(err) {
				printPISearchProgress(cmd, processInstanceProgressSummary{
					PageSize:          page.Request.Size,
					CurrentPageCount:  len(limitedPage.Items),
					CumulativeCount:   cumulative,
					OverflowState:     page.OverflowState,
					ContinuationState: processInstanceContinuationPartialComplete,
				})
				return results, nil
			}
			return processInstancePageActionResults{}, err
		}

		impact := result.Impact
		results.Reports = append(results.Reports, result.Reports...)
		if result.DryRunPreview != nil {
			results.DryRunPreviews = append(results.DryRunPreviews, *result.DryRunPreview)
		}
		cumulative += len(limitedPage.Items)
		if impact.Affected > 0 {
			cumulativeAffected += impact.Affected
		} else {
			cumulativeAffected += len(limitedPage.Items)
		}
		summary := newPIProgressSummary(limitedPage, cumulative, flagDryRun || shouldAutoContinuePISearchPages(cmd))
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop, processInstanceContinuationLimitReached:
			return results, nil
		case processInstanceContinuationAutoContinue:
			firstPage = false
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
			continue
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Processed %d process instance(s) on this page (%d requested so far, %d including dependencies). More matching process instances remain. Continue?", summary.CurrentPageCount, summary.CumulativeCount, cumulativeAffected)
			if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
				if isCmdAborted(err) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return results, nil
				}
				return processInstancePageActionResults{}, err
			}
			firstPage = false
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
		}
	}
}

func applyPISearchResultFilters(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.ProcessInstances, error) {
	var err error
	// Keep the local fallback path in place so versions without reliable
	// request-side support still preserve the existing filter semantics.
	if flagGetPIChildrenOnly {
		pis = pis.FilterChildrenOnly()
	}
	if flagGetPIRootsOnly {
		pis = pis.FilterRootsOnly()
	}
	if flagGetPIOrphanChildrenOnly {
		stopActivity := startCommandActivity(cmd, fmt.Sprintf("checking orphan parents for %d process instance(s)", len(pis.Items)))
		pis.Items, err = cli.FilterProcessInstanceWithOrphanParent(cmd.Context(), pis.Items, collectOptions()...)
		stopActivity()
		if err != nil {
			return process.ProcessInstances{}, fmt.Errorf("error filtering orphan children: %w", err)
		}
		pis.Total = int32(len(pis.Items))
	}
	if flagGetPIIncidentsOnly {
		pis = pis.FilterByHavingIncidents(true)
	}
	if flagGetPINoIncidentsOnly {
		pis = pis.FilterByHavingIncidents(false)
	}
	return pis, nil
}

func canUsePIReportedTotal() bool {
	return !(flagGetPIChildrenOnly || flagGetPIRootsOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPINoIncidentsOnly)
}

func canUsePIExactReportedTotal(page process.ProcessInstancePage) bool {
	return canUsePIReportedTotal() &&
		page.ReportedTotal != nil &&
		page.ReportedTotal.Kind == process.ProcessInstanceReportedTotalKindExact
}

func nextPISearchPageRequest(cmd *cobra.Command, cfg *config.Config, current process.ProcessInstancePageRequest, page process.ProcessInstancePage) process.ProcessInstancePageRequest {
	if page.EndCursor != "" {
		req := newPISearchPageRequest(cmd, cfg, 0)
		req.After = page.EndCursor
		return req
	}
	return newPISearchPageRequest(cmd, cfg, current.From+int32(len(page.Items)))
}

func startCommandActivity(cmd *cobra.Command, msg string) func() {
	if cmd == nil {
		return func() {}
	}
	return logging.StartActivity(cmd.Context(), msg)
}

func printPISearchProgress(cmd *cobra.Command, summary processInstanceProgressSummary) {
	if cmd == nil || !flagVerbose || pickMode() != RenderModeOneLine {
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), formatPISearchProgress(summary))
}

func logPISearchProgress(cmd *cobra.Command, log *slog.Logger, summary processInstanceProgressSummary) {
	if cmd == nil || log == nil || !flagVerbose || pickMode() != RenderModeOneLine {
		return
	}
	log.InfoContext(cmd.Context(), formatPISearchProgress(summary))
}

func formatPISearchProgress(summary processInstanceProgressSummary) string {
	line := fmt.Sprintf("page size: %d, current page: %d, total so far: %d, more matches: %s, next step: %s",
		summary.PageSize,
		summary.CurrentPageCount,
		summary.CumulativeCount,
		describePIOverflowState(summary.OverflowState),
		describePIContinuationState(summary.ContinuationState),
	)
	if detail := describePIProgressDetail(summary); detail != "" {
		line += ", " + detail
	}
	return line
}

func describePIOverflowState(state process.ProcessInstanceOverflowState) string {
	switch state {
	case process.ProcessInstanceOverflowStateHasMore:
		return "yes"
	case process.ProcessInstanceOverflowStateIndeterminate:
		return "unknown"
	default:
		return "no"
	}
}

func describePIContinuationState(state processInstanceContinuationState) string {
	switch state {
	case processInstanceContinuationPrompt:
		return "prompt"
	case processInstanceContinuationAutoContinue:
		return "auto-continue"
	case processInstanceContinuationPartialComplete:
		return "partial-complete"
	case processInstanceContinuationWarningStop:
		return "warning-stop"
	case processInstanceContinuationLimitReached:
		return "limit-reached"
	default:
		return "complete"
	}
}

func describePIProgressDetail(summary processInstanceProgressSummary) string {
	switch summary.ContinuationState {
	case processInstanceContinuationPrompt:
		return "detail: prompt before processing the next page"
	case processInstanceContinuationAutoContinue:
		return "detail: auto-confirm will continue with the next page"
	case processInstanceContinuationPartialComplete:
		return fmt.Sprintf("detail: stopped after %d processed process instance(s); remaining matches were left untouched", summary.CumulativeCount)
	case processInstanceContinuationWarningStop:
		return fmt.Sprintf("warning: stopped after %d processed process instance(s) because more matching process instances may remain", summary.CumulativeCount)
	case processInstanceContinuationLimitReached:
		return fmt.Sprintf("detail: stopped after reaching limit of %d process instance(s)", flagGetPILimit)
	default:
		return "detail: no additional matching process instances remain"
	}
}

func validatePISearchFlags(cmds ...*cobra.Command) error {
	var cmd *cobra.Command
	if len(cmds) > 0 {
		cmd = cmds[0]
	}
	if flagGetPISize <= 0 || flagGetPISize > consts.MaxPISearchSize {
		return invalidFlagValuef("invalid value for --batch-size: %d, expected positive integer up to %d", flagGetPISize, consts.MaxPISearchSize)
	}
	if flagGetPILimit < 0 || (flagGetPILimit == 0 && isPILimitFlagChanged(cmd)) {
		return invalidFlagValuef("--limit must be positive integer")
	}
	if flagGetPIState != "" && flagGetPIState != "all" {
		if _, ok := process.ParseState(flagGetPIState); !ok {
			return invalidFlagValuef("invalid value for --state: %q, valid values are: %s", flagGetPIState, process.ValidStateStrings())
		}
	}
	if flagGetPITotal {
		switch {
		case flagGetPILimit > 0:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --limit")
		case flagViewAsJson:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --json")
		case flagViewKeysOnly:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --keys-only")
		case flagGetPIWithIncidents:
			return mutuallyExclusiveFlagsf("--total cannot be combined with --with-incidents")
		}
	}
	if flagGetPIIncidentMessageLimit < 0 {
		return invalidFlagValuef("invalid value for --incident-message-limit: %d, expected non-negative integer", flagGetPIIncidentMessageLimit)
	}
	if isPIIncidentMessageLimitFlagChanged(cmd) && !flagGetPIWithIncidents {
		return missingDependentFlagsf("--incident-message-limit requires --with-incidents")
	}
	if flagGetPIProcessDefinitionKey != "" &&
		(flagGetPIBpmnProcessID != "" ||
			flagGetPIProcessVersion != 0 ||
			flagGetPIProcessVersionTag != "") {
		return mutuallyExclusiveFlagsf("--pd-key is mutually exclusive with --bpmn-process-id, --pd-version, and --pd-version-tag")
	}
	if err := validatePIDateFlag("--start-date-after", flagGetPIStartDateAfter); err != nil {
		return err
	}
	if err := validatePIDateFlag("--start-date-before", flagGetPIStartDateBefore); err != nil {
		return err
	}
	if err := validatePIDateFlag("--end-date-after", flagGetPIEndDateAfter); err != nil {
		return err
	}
	if err := validatePIDateFlag("--end-date-before", flagGetPIEndDateBefore); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--start-date-older-days", flagGetPIStartAfterDays); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--start-date-newer-days", flagGetPIStartBeforeDays); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--end-date-older-days", flagGetPIEndAfterDays); err != nil {
		return err
	}
	if err := validatePIRelativeDayFlag("--end-date-newer-days", flagGetPIEndBeforeDays); err != nil {
		return err
	}
	if err := validatePIMixedDateFilterInputs(); err != nil {
		return err
	}
	if err := validatePIDateRange("--start-date-after", flagGetPIStartDateAfter, "--start-date-before", flagGetPIStartDateBefore); err != nil {
		return err
	}
	if err := validatePIDateRange("--end-date-after", flagGetPIEndDateAfter, "--end-date-before", flagGetPIEndDateBefore); err != nil {
		return err
	}
	if err := validatePIDateRange("--start-date-newer-days", pickPIDateBound("", flagGetPIStartBeforeDays), "--start-date-older-days", pickPIDateUpperBound("", flagGetPIStartAfterDays)); err != nil {
		return err
	}
	if err := validatePIDateRange("--end-date-newer-days", pickPIDateBound("", flagGetPIEndBeforeDays), "--end-date-older-days", pickPIDateUpperBound("", flagGetPIEndAfterDays)); err != nil {
		return err
	}
	if flagGetPIBpmnProcessID == "" &&
		(flagGetPIProcessVersion != 0 || flagGetPIProcessVersionTag != "") {
		return missingDependentFlagsf("--pd-version and --pd-version-tag require --bpmn-process-id to be set")
	}
	if flagGetPIChildrenOnly && flagGetPIRootsOnly {
		return forbiddenFlagCombinationf("using both --children-only and --roots-only filters returns does not make sense")
	}
	if flagGetPIIncidentsOnly && flagGetPINoIncidentsOnly {
		return forbiddenFlagCombinationf("using both --incidents-only and --no-incidents-only filters does not make sense")
	}
	return nil
}

func isPILimitFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("limit")
}

func isPIIncidentMessageLimitFlagChanged(cmd *cobra.Command) bool {
	return cmd != nil && cmd.Flags().Changed("incident-message-limit")
}

func validatePIKeyedModeLimit(keyCount int) error {
	if keyCount > 0 && flagGetPILimit > 0 {
		return mutuallyExclusiveFlagsf("--limit cannot be combined with --key")
	}
	return nil
}

// validatePIWithIncidentsUsage keeps incident enrichment out of modes that cannot attach details unambiguously.
func validatePIWithIncidentsUsage(keyCount int, filterFlagsSet bool) error {
	if !flagGetPIWithIncidents {
		return nil
	}
	if keyCount > 0 && (filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPINoIncidentsOnly || flagGetPITotal) {
		return mutuallyExclusiveFlagsf("--with-incidents cannot be combined with search-mode filters")
	}
	return nil
}

func useInvalidInputFlagErrors(cmd *cobra.Command) {
	cmd.SetFlagErrorFunc(func(cmd *cobra.Command, err error) error {
		return invalidInputError(err)
	})
}

func validatePISearchVersionSupport(cfg *config.Config) error {
	if cfg == nil {
		return nil
	}
	if flagGetPIOrphanChildrenOnly && cfg.App.CamundaVersion == toolx.V87 {
		return ferrors.WrapClass(ferrors.ErrUnsupported,
			fmt.Errorf("--orphan-children-only is not supported in Camunda 8.7 because orphan-parent follow-up lookup is not tenant-safe"))
	}
	return nil
}

func validatePIMixedDateFilterInputs() error {
	if hasStartPIAbsoluteDateFlags() && hasStartPIRelativeDayFlags() {
		return mutuallyExclusiveFlagsf("start-date absolute and relative day filters cannot be combined")
	}
	if hasEndPIAbsoluteDateFlags() && hasEndPIRelativeDayFlags() {
		return mutuallyExclusiveFlagsf("end-date absolute and relative day filters cannot be combined")
	}
	return nil
}

func hasStartPIAbsoluteDateFlags() bool {
	return flagGetPIStartDateAfter != "" || flagGetPIStartDateBefore != ""
}

func hasEndPIAbsoluteDateFlags() bool {
	return flagGetPIEndDateAfter != "" || flagGetPIEndDateBefore != ""
}

func hasStartPIRelativeDayFlags() bool {
	return flagGetPIStartAfterDays >= 0 || flagGetPIStartBeforeDays >= 0
}

func hasEndPIRelativeDayFlags() bool {
	return flagGetPIEndAfterDays >= 0 || flagGetPIEndBeforeDays >= 0
}

func validatePIDateFlag(flagName, value string) error {
	if value == "" {
		return nil
	}
	if _, err := time.Parse(time.DateOnly, value); err != nil {
		return invalidFlagValuef("invalid value for %s: %q, expected YYYY-MM-DD", flagName, value)
	}
	return nil
}

func validatePIRelativeDayFlag(flagName string, value int) error {
	if value < 0 {
		if value == -1 {
			return nil
		}
		return invalidFlagValuef("invalid value for %s: %d, expected non-negative integer", flagName, value)
	}
	return nil
}

func validatePIDateRange(afterFlag, afterValue, beforeFlag, beforeValue string) error {
	if afterValue == "" || beforeValue == "" {
		return nil
	}

	after, err := time.Parse(time.DateOnly, afterValue)
	if err != nil {
		return err
	}
	before, err := time.Parse(time.DateOnly, beforeValue)
	if err != nil {
		return err
	}
	if after.After(before) {
		return invalidFlagValuef("invalid range for %s and %s: %q is later than %q", afterFlag, beforeFlag, afterValue, beforeValue)
	}
	return nil
}

func pickPIDateBound(absolute string, relativeDays int) string {
	if absolute != "" {
		return absolute
	}
	if relativeDays < 0 {
		return ""
	}
	return derivePIDateBound(relativeDays)
}

func pickPIDateUpperBound(absolute string, relativeDays int) string {
	if absolute != "" {
		return absolute
	}
	if relativeDays < 0 {
		return ""
	}
	return derivePIUpperDateBound(relativeDays)
}

func derivePIDateBound(relativeDays int) string {
	return relativeDayNow().AddDate(0, 0, -relativeDays).Format(time.DateOnly)
}

func derivePIUpperDateBound(relativeDays int) string {
	return relativeDayNow().AddDate(0, 0, -relativeDays).Format(time.DateOnly)
}
