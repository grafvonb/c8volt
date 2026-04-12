package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	flagGetPIKeys                 []string
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
	flagGetPIWithAge              bool
	flagGetPIState                string
	flagGetPIParentKey            string
	flagGetPISize                 int32
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
	Example: `  ./c8volt get pi --state active
  ./c8volt get pi --bpmn-process-id C88_SimpleUserTask_Process --state active
  ./c8volt get pi --bpmn-process-id C88_SimpleUserTask_Process --count 250
  ./c8volt get pi --state active --auto-confirm
  ./c8volt get pi --start-date-after 2026-01-01 --start-date-before 2026-01-31
		  ./c8volt get pi --start-date-older-days 7 --start-date-newer-days 30
  ./c8volt get pi --end-date-before 2026-03-31 --state completed
		  ./c8volt get pi --end-date-newer-days 14 --state completed
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
		ctx := cmd.Context()
		fail := func(err error) {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			fail(invalidFlagValuef("--workers must be positive integer"))
		}
		if err := validatePISearchFlags(); err != nil {
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

		log.Debug(fmt.Sprintf("fetching process instances, render mode: %s", pickMode()))
		var pis process.ProcessInstances
		switch {
		case lk > 0:
			log.Debug(fmt.Sprintf("searching for key(s) [%s]", keys))
			if err := validatePIKeyedModeDateFilters(lk); err != nil {
				fail(err)
			}
			if filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPINoIncidentsOnly {
				fail(mutuallyExclusiveFlagsf("--key cannot be combined with other filters"))
			}
			if cmd.Flags().Changed("workers") {
				lk = flagWorkers
			}
			pis, err = cli.GetProcessInstances(ctx, ukeys, lk, collectOptions()...)
			if err != nil {
				msg := fmt.Errorf("error fetching %d process instances: %w", lk, err)
				if flagVerbose {
					msg = fmt.Errorf("error fetching %d process instances for key(s) [%s]: %w", lk, ukeys, err)
				}
				fail(msg)
			}
		default:
			filter := populatePISearchFilterOpts()
			log.Debug(fmt.Sprintf("using process instance search filter: %+v", filter))
			pis, err = searchProcessInstancesWithPaging(cmd, cli, cfg, filter)
			if err != nil {
				fail(fmt.Errorf("error fetching process instances: %w", err))
			}
		}
		if err := listProcessInstancesView(cmd, pis); err != nil {
			fail(fmt.Errorf("error rendering items view: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getProcessInstanceCmd)

	fs := getProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagGetPIKeys, "key", "k", nil, "process instance key(s) to fetch")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	registerPISharedDateRangeFlags(fs)
	registerPISharedRenderFlags(fs)
	fs.Int32VarP(&flagGetPISize, "count", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to fetch per page (max limit %d enforced by server)", consts.MaxPISearchSize))

	// filtering options
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to filter process instances")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")

	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "show only root process instances, meaning instances with empty parent key")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "show only child process instances, meaning instances that have a parent key set")

	fs.BoolVar(&flagGetPIOrphanChildrenOnly, "orphan-children-only", false, "show only child instances where parent key is set but the parent process instance does not exist (anymore)")

	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")
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
)

type processInstanceProgressSummary struct {
	PageSize          int32
	CurrentPageCount  int
	CumulativeCount   int
	OverflowState     process.ProcessInstanceOverflowState
	ContinuationState processInstanceContinuationState
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

func registerPISharedRenderFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&flagGetPIWithAge, "with-age", false, "include process instance age in one-line output and JSON meta")
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
	if cmd != nil && cmd.Flags().Changed("count") {
		return pickPISearchSize()
	}
	if cfg != nil && cfg.App.ProcessInstancePageSize > 0 && cfg.App.ProcessInstancePageSize <= consts.MaxPISearchSize {
		return cfg.App.ProcessInstancePageSize
	}
	return consts.MaxPISearchSize
}

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

func newPIProgressSummary(page process.ProcessInstancePage, cumulative int, autoConfirm bool) processInstanceProgressSummary {
	return processInstanceProgressSummary{
		PageSize:          page.Request.Size,
		CurrentPageCount:  len(page.Items),
		CumulativeCount:   cumulative,
		OverflowState:     page.OverflowState,
		ContinuationState: pickPIContinuationState(page.OverflowState, autoConfirm),
	}
}

func searchProcessInstancesWithPaging(cmd *cobra.Command, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (process.ProcessInstances, error) {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	var collected process.ProcessInstances

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return process.ProcessInstances{}, err
		}

		filtered, err := applyPISearchResultFilters(cmd, cli, process.ProcessInstances{
			Total: int32(len(page.Items)),
			Items: page.Items,
		})
		if err != nil {
			return process.ProcessInstances{}, err
		}
		collected.Items = append(collected.Items, filtered.Items...)
		collected.Total = int32(len(collected.Items))

		summary := newPIProgressSummary(page, int(collected.Total), flagCmdAutoConfirm)
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop:
			return collected, nil
		case processInstanceContinuationAutoContinue:
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
			continue
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Fetched %d process instance(s) on this page (%d total so far). More matching process instances remain. Continue?", summary.CurrentPageCount, summary.CumulativeCount)
			if err := confirmCmdOrAbortFn(flagCmdAutoConfirm, prompt); err != nil {
				if errors.Is(err, ErrCmdAborted) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return collected, nil
				}
				return process.ProcessInstances{}, err
			}
			pageReq = newPISearchPageRequest(cmd, cfg, pageReq.From+int32(len(page.Items)))
		}
	}
}

func processPISearchPagesWithAction(
	cmd *cobra.Command,
	cli process.API,
	cfg *config.Config,
	filter process.ProcessInstanceFilter,
	processPage func(page process.ProcessInstancePage, firstPage bool) error,
) error {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	cumulative := 0
	firstPage := true

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return err
		}
		if len(page.Items) == 0 {
			if cumulative == 0 {
				cmd.Println("found:", 0)
			}
			return nil
		}

		if err := processPage(page, firstPage); err != nil {
			if !firstPage && errors.Is(err, ErrCmdAborted) {
				printPISearchProgress(cmd, processInstanceProgressSummary{
					PageSize:          page.Request.Size,
					CurrentPageCount:  len(page.Items),
					CumulativeCount:   cumulative,
					OverflowState:     page.OverflowState,
					ContinuationState: processInstanceContinuationPartialComplete,
				})
				return nil
			}
			return err
		}

		cumulative += len(page.Items)
		summary := newPIProgressSummary(page, cumulative, flagCmdAutoConfirm)
		printPISearchProgress(cmd, summary)

		switch summary.ContinuationState {
		case processInstanceContinuationCompleted, processInstanceContinuationWarningStop:
			return nil
		case processInstanceContinuationAutoContinue:
			firstPage = false
			pageReq = newPISearchPageRequest(cmd, cfg, 0)
			continue
		case processInstanceContinuationPrompt:
			prompt := fmt.Sprintf("Processed %d process instance(s) on this page (%d total so far). More matching process instances remain. Continue?", summary.CurrentPageCount, summary.CumulativeCount)
			if err := confirmCmdOrAbortFn(flagCmdAutoConfirm, prompt); err != nil {
				if errors.Is(err, ErrCmdAborted) {
					printPISearchProgress(cmd, processInstanceProgressSummary{
						PageSize:          summary.PageSize,
						CurrentPageCount:  summary.CurrentPageCount,
						CumulativeCount:   summary.CumulativeCount,
						OverflowState:     summary.OverflowState,
						ContinuationState: processInstanceContinuationPartialComplete,
					})
					return nil
				}
				return err
			}
			firstPage = false
			pageReq = newPISearchPageRequest(cmd, cfg, 0)
		}
	}
}

func applyPISearchResultFilters(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.ProcessInstances, error) {
	var err error
	if flagGetPIChildrenOnly {
		pis = pis.FilterChildrenOnly()
	}
	if flagGetPIRootsOnly {
		pis = pis.FilterRootsOnly()
	}
	if flagGetPIOrphanChildrenOnly {
		pis.Items, err = cli.FilterProcessInstanceWithOrphanParent(cmd.Context(), pis.Items, collectOptions()...)
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

func printPISearchProgress(cmd *cobra.Command, summary processInstanceProgressSummary) {
	if cmd == nil || pickMode() != RenderModeOneLine {
		return
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "page size: %d, current page: %d, total so far: %d, more matches: %s, next step: %s\n",
		summary.PageSize,
		summary.CurrentPageCount,
		summary.CumulativeCount,
		describePIOverflowState(summary.OverflowState),
		describePIContinuationState(summary.ContinuationState),
	)
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
	default:
		return "complete"
	}
}

func validatePISearchFlags() error {
	if flagGetPIState != "" && flagGetPIState != "all" {
		if _, ok := process.ParseState(flagGetPIState); !ok {
			return invalidFlagValuef("invalid value for --state: %q, valid values are: %s", flagGetPIState, process.ValidStateStrings())
		}
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
