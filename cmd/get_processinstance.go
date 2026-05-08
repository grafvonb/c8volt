// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

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
	flagGetPIIncidentMessageLimit int
	flagGetPIWithVars             bool
	flagGetPIVarValueLimit        int
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
	Long: "Get process instances by key or by search criteria.\n\n" +
		"Use direct lookup when you know a process-instance key, or combine search filters to inspect matching process instances by process definition, tenant, state, incidents, variables, jobs, user tasks, and time ranges.\n\n" +
		"Search results support interactive paging, scriptable JSON aggregation, and count-only workflows. Direct key lookup stays strict: missing keys return not-found.\n\n" +
		"When --bpmn-process-id is set, c8volt validates that the process definition is visible before searching process instances. A missing selector fails with a local diagnostic instead of looking like a valid empty result; --json, --automation, --keys-only, and non-TTY runs never prompt for recovery output.\n\n" +
		"Use --with-incidents to include direct incident details under matching process-instance rows in keyed or list/search output.\n\n" +
		"Use --with-vars to include process-instance-scope variables under matching process-instance rows in keyed or list/search output.\n\n" +
		"Use --has-user-tasks to fetch process instances by their owning user-task keys.\n\n" +
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
  ./c8volt get pi --direct-incidents-only --with-incidents
  ./c8volt get pi --with-incidents --incident-message-limit 80
  ./c8volt get pi --with-vars --var-value-limit 120
  ./c8volt get pi --key 2251799813711967 --with-incidents
  ./c8volt get pi --key 2251799813711967 --with-incidents --incident-state all
  ./c8volt get pi --key 2251799813711967 --with-vars
  ./c8volt get pi --key 2251799813711967 --with-vars --with-incidents
  ./c8volt get pi --key 2251799813711967 --with-vars --var-value-limit 120
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
		if err := validatePIWithIncidentsUsage(cmd, lk, filterFlagsSet); err != nil {
			fail(err)
		}
		if err := validatePIWithVarsUsage(lk, filterFlagsSet); err != nil {
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
			if filterFlagsSet || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPIDirectIncidentsOnly || flagGetPINoIncidentsOnly {
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
			if flagGetPIWithIncidents && flagGetPIWithVars {
				incidentEnriched, err := enrichProcessInstancesWithIncidentActivity(cmd, cli, pis)
				if err != nil {
					fail(fmt.Errorf("get process instance incidents: %w", err))
				}
				variableEnriched, err := enrichProcessInstancesWithVariableActivity(cmd, cli, pis)
				if err != nil {
					fail(fmt.Errorf("get process instance variables: %w", err))
				}
				if err := processInstanceActivityInstancesView(cmd, mergeIncidentAndVariableActivity(incidentEnriched, variableEnriched)); err != nil {
					fail(fmt.Errorf("render process instances with variables and incidents: %w", err))
				}
				return
			}
			if flagGetPIWithIncidents {
				enriched, err := enrichProcessInstancesWithIncidentActivity(cmd, cli, pis)
				if err != nil {
					fail(fmt.Errorf("get process instance incidents: %w", err))
				}
				if err := incidentEnrichedProcessInstancesView(cmd, enriched); err != nil {
					fail(fmt.Errorf("render process instances with incidents: %w", err))
				}
				return
			}
			if flagGetPIWithVars {
				enriched, err := enrichProcessInstancesWithVariableActivity(cmd, cli, pis)
				if err != nil {
					fail(fmt.Errorf("get process instance variables: %w", err))
				}
				if err := variableEnrichedProcessInstancesView(cmd, enriched); err != nil {
					fail(fmt.Errorf("render process instances with variables: %w", err))
				}
				return
			}
		default:
			result, err := validateProcessDefinitionSelectors(ctx, cli, newPIProcessDefinitionSelectorValidationRequest(), collectOptions()...)
			if err != nil {
				fail(err)
			}
			handleProcessDefinitionSelectorValidationError(cmd, log, cfg.App.NoErrCodes, cli, result)
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
		if flagGetPIWithIncidents && flagGetPIWithVars {
			incidentEnriched, err := enrichProcessInstancesWithIncidentActivity(cmd, cli, pis)
			if err != nil {
				fail(fmt.Errorf("get process instance incidents: %w", err))
			}
			variableEnriched, err := enrichProcessInstancesWithVariableActivity(cmd, cli, pis)
			if err != nil {
				fail(fmt.Errorf("get process instance variables: %w", err))
			}
			if err := processInstanceActivityInstancesView(cmd, mergeIncidentAndVariableActivity(incidentEnriched, variableEnriched)); err != nil {
				fail(fmt.Errorf("render process instances with variables and incidents: %w", err))
			}
			return
		}
		if flagGetPIWithIncidents {
			enriched, err := enrichProcessInstancesWithIncidentActivity(cmd, cli, pis)
			if err != nil {
				fail(fmt.Errorf("get process instance incidents: %w", err))
			}
			if err := incidentEnrichedProcessInstancesView(cmd, enriched); err != nil {
				fail(fmt.Errorf("render process instances with incidents: %w", err))
			}
			return
		}
		if flagGetPIWithVars {
			enriched, err := enrichProcessInstancesWithVariableActivity(cmd, cli, pis)
			if err != nil {
				fail(fmt.Errorf("get process instance variables: %w", err))
			}
			if err := variableEnrichedProcessInstancesView(cmd, enriched); err != nil {
				fail(fmt.Errorf("render process instances with variables: %w", err))
			}
			return
		}
		if err := listProcessInstancesView(cmd, pis); err != nil {
			fail(fmt.Errorf("render process instances: %w", err))
		}
	},
}

// init registers the process-instance get command and binds its command-line
// flags. Execution behavior stays in the command Run function; the supporting
// search, paging, filtering, validation, and enrichment code lives in the
// adjacent get_processinstance_* files.
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
	fs.BoolVar(&flagGetPIWithIncidents, "with-incidents", false, "include direct incident keys, states, and messages for keyed or list/search process-instance output")
	fs.StringVar(&flagGetPIIncidentState, "incident-state", "active", "incident state scope for keyed --with-incidents: active, pending, resolved, migrated, unknown, all")
	fs.IntVar(&flagGetPIIncidentMessageLimit, "incident-message-limit", 0, "maximum characters to show for human incident messages when --with-incidents is set; 0 disables truncation")
	fs.BoolVar(&flagGetPIWithVars, "with-vars", false, "include process-instance-scope variables for keyed or list/search process-instance output")
	fs.IntVar(&flagGetPIVarValueLimit, "var-value-limit", 0, "maximum characters to show for human variable values when --with-vars is set; 0 disables truncation")

	// filtering options
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to filter process instances")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")

	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "show only root process instances")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "show only child process instances")

	fs.BoolVar(&flagGetPIOrphanChildrenOnly, "orphan-children-only", false, "show only child instances with missing parents")

	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPIDirectIncidentsOnly, "direct-incidents-only", false, "show only process instances with direct incident details")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")

	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --batch-size > 1 (default: min(batch-size, GOMAXPROCS))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "disable limiting the number of workers to GOMAXPROCS when --workers > 1")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(getProcessInstanceCmd, CommandMutationReadOnly)
	setContractSupport(getProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(getProcessInstanceCmd, AutomationSupportFull, "supports unattended confirmation-free paging and shared machine output")
}
