package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
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
			pis, err = cli.SearchProcessInstances(ctx, filter, pickPISearchSize())
			if err != nil {
				fail(fmt.Errorf("error fetching process instances: %w", err))
			}
			if flagGetPIChildrenOnly {
				pis = pis.FilterChildrenOnly()
			}
			if flagGetPIRootsOnly {
				pis = pis.FilterRootsOnly()
			}
			if flagGetPIOrphanChildrenOnly {
				pis.Items, err = cli.FilterProcessInstanceWithOrphanParent(ctx, pis.Items)
				if err != nil {
					fail(fmt.Errorf("error filtering orphan children: %w", err))
				}
			}
			if flagGetPIIncidentsOnly {
				pis = pis.FilterByHavingIncidents(true)
			}
			if flagGetPINoIncidentsOnly {
				pis = pis.FilterByHavingIncidents(false)
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
	fs.Int32VarP(&flagGetPISize, "count", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to fetch (max limit %d enforced by server)", consts.MaxPISearchSize))

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

