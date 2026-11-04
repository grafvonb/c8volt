// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt"
	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
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
	flagGetPIIncidentState        string
	flagGetPIIncidentErrorType    string
	flagGetPIIncidentErrorMessage string
	flagGetPIIncidentMessageLimit int
	flagGetPIWithVars             bool
	flagGetPIVarValueLimit        int
	flagGetPIWithElements         bool
	flagGetPIWithListeners        bool
	flagGetPIVarExists            []string
	flagGetPIVars                 []string
	flagGetPIVarLikes             []string
)

// command options
var (
	flagGetPIRootsOnly           bool
	flagGetPIChildrenOnly        bool
	flagGetPIOrphanChildrenOnly  bool
	flagGetPIIncidentsOnly       bool
	flagGetPIDirectIncidentsOnly bool
	flagGetPINoIncidentsOnly     bool
)

var getProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "List or fetch process instances",
	Long: `Inspect process instances by key or search by definition, state, incidents,
variables, or time range. Search filters combine with AND.

Use --bid for one exact business ID (only c8.10+); commas and wildcards are literal.
Add --with-incidents for incident details, --with-vars for root variables, or
--with-elements for runtime elements. Add --with-listeners to --with-elements
for listener jobs.

Use --limit to bound search results or --total to count matches. Explicit keys
cannot be combined with search filters.`,
	Example: `  c8volt get process-instance --key <process-instance-key> --with-incidents
  c8volt get process-instance --bpmn-process-id order --state active --limit 10
  c8volt get process-instance --bid ORDER-123 --json`,
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
		var pis process.ProcessInstances
		ctx := cmd.Context()
		fail := func(err error) {
			handleBusinessIDSearchError(cmd, log, cfg.App.NoErrCodes, err, len(pis.Items))
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			fail(invalidFlagValuef("--workers must be positive integer"))
		}
		if err := validatePISearchFlags(cmd); err != nil {
			fail(err)
		}
		filterFlagsSet := hasPISearchFilterFlags(cmd)

		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			fail(err)
		}
		keys := mergeAndValidateKeys(cmd, flagGetPIKeys, stdinKeys, log, cfg)
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
		if err := validatePIIncidentDetailFilterUsage(cmd, lk); err != nil {
			fail(err)
		}
		if err := validatePIWithIncidentsUsage(cmd, lk, filterFlagsSet); err != nil {
			fail(err)
		}
		if err := validatePIWithVarsUsage(lk, filterFlagsSet); err != nil {
			fail(err)
		}
		if err := validatePIWithElementsUsage(lk, filterFlagsSet); err != nil {
			fail(err)
		}

		log.Debug(fmt.Sprintf("getting pi; mode %s", pickMode()))
		switch {
		case ltk > 0:
			log.Debug(fmt.Sprintf("resolving pi keys from user tasks [%s]", taskKeys))
			processInstanceKeys, err := resolveProcessInstanceKeysFromUserTasksForCommand(cmd, cli, taskKeys, collectOptions())
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
			log.Debug(fmt.Sprintf("getting pi keys [%s]", keys))
			adminInputOpts := collectExplicitPIAdminInputOptions()
			adminInputIncidentOpts := append(collectIncidentEnrichmentOptions(), processOptions.WithIgnoreTenant())
			if err := validatePIKeyedModeDateFilters(lk); err != nil {
				fail(err)
			}
			if err := validatePIKeyedModeLimit(lk); err != nil {
				fail(err)
			}
			if flagGetPITotal {
				fail(mutuallyExclusiveFlagsf("--total cannot be combined with --key"))
			}
			// Keyed lookups intentionally stay strict; partial orphan warnings are only for traversal and impact-check flows.
			if filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPIDirectIncidentsOnly || flagGetPINoIncidentsOnly {
				fail(mutuallyExclusiveFlagsf("--key cannot be combined with other filters"))
			}
			if cmd.Flags().Changed("workers") {
				lk = flagWorkers
			}
			pis, err = cli.GetProcessInstances(ctx, ukeys, lk, adminInputOpts...)
			if err != nil {
				msg := fmt.Errorf("get process instances: %w", err)
				if flagVerbose {
					msg = fmt.Errorf("get process instances for key(s) [%s]: %w", ukeys, err)
				}
				fail(msg)
			}
			if flagGetPIWithIncidents || flagGetPIWithVars || flagGetPIWithElements {
				activity, err := collectRequestedProcessInstanceActivityOptions(cmd, cli, pis, adminInputOpts, adminInputIncidentOpts)
				if err != nil {
					fail(err)
				}
				if err := processInstanceActivityInstancesView(cmd, activity); err != nil {
					fail(fmt.Errorf("render enriched process instances: %w", err))
				}
				return
			}
		default:
			filter := populatePISearchFilterOpts(cmd)
			if filter.BusinessId != nil {
				if err := cli.ValidateProcessInstanceSearchFilter(filter); err != nil {
					fail(err)
				}
			}
			result, err := validateProcessDefinitionSelectors(ctx, cli, newPIProcessDefinitionSelectorValidationRequest(), collectOptions()...)
			if err != nil {
				fail(err)
			}
			handleProcessDefinitionSelectorValidationError(cmd, log, cfg.App.NoErrCodes, cli, result)
			log.Debug(fmt.Sprintf("searching pi; filter %s", filter.String()))
			if flagGetPITotal {
				if flagGetPIOrphanChildrenOnly {
					pis, _, err := searchOrphanProcessInstancesWithSharedDiscovery(cmd, cli, cfg, filter)
					if err != nil {
						fail(fmt.Errorf("get process instances total: %w", err))
					}
					if err := processInstanceTotalView(cmd, int64(len(pis.Items))); err != nil {
						fail(fmt.Errorf("render process instance total: %w", err))
					}
					return
				}
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
			if flagGetPIOrphanChildrenOnly {
				pis, renderedIncrementally, err = searchOrphanProcessInstancesWithSharedDiscovery(cmd, cli, cfg, filter)
			} else {
				pis, renderedIncrementally, err = searchProcessInstancesWithPaging(cmd, cli, cfg, filter)
			}
			if err != nil {
				fail(fmt.Errorf("get process instances: %w", err))
			}
			if renderedIncrementally {
				return
			}
		}
		if flagGetPIWithIncidents || flagGetPIWithVars || flagGetPIWithElements {
			activity, err := collectRequestedProcessInstanceActivity(cmd, cli, pis)
			if err != nil {
				fail(err)
			}
			if err := processInstanceActivityInstancesView(cmd, activity); err != nil {
				fail(fmt.Errorf("render enriched process instances: %w", err))
			}
			return
		}
		if err := listProcessInstancesView(cmd, pis); err != nil {
			fail(fmt.Errorf("render process instances: %w", err))
		}
	},
}

// resolveProcessInstanceKeysFromUserTasksForCommand keeps user-task resolution testable with the command context.
func resolveProcessInstanceKeysFromUserTasksForCommand(cmd *cobra.Command, cli c8volt.API, taskKeys types.Keys, opts []processOptions.FacadeOption) (types.Keys, error) {
	return cli.ResolveProcessInstanceKeysFromUserTasks(cmd.Context(), taskKeys, opts...)
}

// init registers the process-instance get command and binds its command-line
// flags. Execution behavior stays in the command Run function; the supporting
// search, paging, filtering, validation, and enrichment code lives in the
// adjacent get_processinstance_* files.
func init() {
	getCmd.AddCommand(getProcessInstanceCmd)
	useInvalidInputFlagErrors(getProcessInstanceCmd)

	fs := getProcessInstanceCmd.Flags()
	fs.String("bid", "", "exact business ID to filter by (only c8.10+); commas, spaces and wildcards are literal")
	fs.StringSliceVarP(&flagGetPIKeys, "key", "k", nil, "process instance key(s) to fetch")
	fs.StringSliceVar(&flagGetPIHasUserTasks, "has-user-tasks", nil, "user task key(s) whose owning process instances should be fetched")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	registerPISharedDateRangeFlags(fs)
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("process instances per discovery page (server maximum %d)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum matching instances to return; omit for all matches")
	fs.BoolVar(&flagGetPITotal, "total", false, "return only the numeric total of matching process instances; capped backend totals are counted by paging")
	fs.BoolVar(&flagGetPIWithIncidents, "with-incidents", false, "include direct incident keys, states, and messages for keyed or list/search process-instance output")
	fs.StringVar(&flagGetPIIncidentState, "incident-state", "active", "incident state scope for keyed --with-incidents: active, pending, resolved, migrated, unknown, all")
	fs.StringVar(&flagGetPIIncidentErrorType, "incident-error-type", "", "case-insensitive incident error type filter for keyed --with-incidents or list/search --direct-incidents-only")
	fs.StringVar(&flagGetPIIncidentErrorMessage, "incident-error-message", "", "case-insensitive incident error message substring filter for keyed --with-incidents or list/search --direct-incidents-only")
	fs.IntVar(&flagGetPIIncidentMessageLimit, "incident-message-limit", 0, "maximum characters to show for incident messages when --with-incidents is set; 0 disables truncation")
	fs.BoolVar(&flagGetPIWithVars, "with-vars", false, "include process-instance-scope variables for keyed or list/search process-instance output")
	fs.IntVar(&flagGetPIVarValueLimit, "var-value-limit", 0, "maximum characters to show for variable values when --with-vars is set; 0 disables truncation")
	fs.BoolVar(&flagGetPIWithElements, "with-elements", false, "include runtime element instances for keyed or list/search process-instance output")
	fs.BoolVar(&flagGetPIWithListeners, "with-listeners", false, "include runtime listener jobs; requires --with-elements")
	fs.StringArrayVar(&flagGetPIVarExists, "var-exists", nil, "require variable name(s) to exist; repeat or separate names with commas")
	fs.StringArrayVar(&flagGetPIVars, "var", nil, "require variable equality or advanced clause(s); repeat or separate clauses with commas")
	fs.StringArrayVar(&flagGetPIVarLikes, "var-like", nil, "require variable value pattern clause(s); repeat or separate clauses with commas")

	// filtering options
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to filter process instances")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, suspended (only c8.10+), completed, canceled, terminated")

	_ = getProcessInstanceCmd.RegisterFlagCompletionFunc("state", processInstanceStateCompletion(false))

	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "show only root process instances")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "show only child process instances")

	fs.BoolVar(&flagGetPIOrphanChildrenOnly, "orphan-children-only", false, "show only child instances with missing parents")

	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPIDirectIncidentsOnly, "direct-incidents-only", false, "show only process instances with direct incident details")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers; omit for an automatic limit")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(getProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(getProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(getProcessInstanceCmd, AutomationSupportFull, "supports unattended confirmation-free paging and shared machine output")
}
