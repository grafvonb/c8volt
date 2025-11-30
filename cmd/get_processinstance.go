package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

var (
	flagGetPIKey                  string
	flagGetPIBpmnProcessID        string
	flagGetPIProcessVersion       int32
	flagGetPIProcessVersionTag    string
	flagGetPIProcessDefinitionKey string
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
	Use:     "process-instance",
	Short:   "Get process instances",
	Aliases: []string{"process-instances", "pi", "pis"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error creating c8volt client: %w", err))
		}
		fail := func(err error) {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		if err := validatePISearchFlags(); err != nil {
			fail(err)
		}
		log.Debug(fmt.Sprintf("fetching process instances, render mode: %s", pickMode()))
		filter, populated := populatePISearchFilterOpts()

		if flagGetPIKey != "" {
			log.Debug(fmt.Sprintf("searching by key: %s", flagGetPIKey))
			if populated || flagGetPIRootsOnly || flagGetPIChildrenOnly || flagGetPIOrphanChildrenOnly || flagGetPIIncidentsOnly || flagGetPINoIncidentsOnly {
				fail(fmt.Errorf("%w: --key cannot be combined with other filters", ErrMutuallyExclusiveFlags))
			}
			pi, err := cli.GetProcessInstance(cmd.Context(), flagGetPIKey)
			if err != nil {
				fail(fmt.Errorf("error fetching process instance by key %s: %w", flagGetPIKey, err))
			}

			if err := processInstanceView(cmd, pi); err != nil {
				fail(fmt.Errorf("error rendering view: %w", err))
			}
			return
		}

		filter.Key = flagGetPIKey
		log.Debug(fmt.Sprintf("using process instance search filter: %+v", filter))

		pis, err := cli.SearchProcessInstances(cmd.Context(), filter, pickPISearchSize())
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
			pis.Items, err = cli.FilterProcessInstanceWithOrphanParent(cmd.Context(), pis.Items)
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
		if err := listProcessInstancesView(cmd, pis); err != nil {
			fail(fmt.Errorf("error rendering items view: %w", err))
		}
	},
}

func init() {
	getCmd.AddCommand(getProcessInstanceCmd)

	fs := getProcessInstanceCmd.Flags()
	fs.StringVarP(&flagGetPIKey, "key", "k", "", "process instance key to fetch")
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32Var(&flagGetPIProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	fs.Int32VarP(&flagGetPISize, "count", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to fetch (max limit %d enforced by server)", consts.MaxPISearchSize))

	// filtering options
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to filter process instances")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")

	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "show only root process instances, meaning instances with empty parent key")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "show only child process instances, meaning instances that have a parent key set")

	fs.BoolVar(&flagGetPIOrphanChildrenOnly, "orphan-children-only", false, "show only child instances where parent key is set but the parent process instance does not exist (anymore)")

	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")
}

func populatePISearchFilterOpts() (process.ProcessInstanceFilter, bool) {
	var f process.ProcessInstanceFilter
	var populated bool

	if v := flagGetPIParentKey; v != "" {
		f.ParentKey, populated = v, true
	}
	if v := flagGetPIBpmnProcessID; v != "" {
		f.BpmnProcessId, populated = v, true
	}
	if v := flagGetPIProcessVersion; v != 0 {
		f.ProcessVersion, populated = v, true
	}
	if v := flagGetPIProcessVersionTag; v != "" {
		f.ProcessVersionTag, populated = v, true
	}
	if v := flagGetPIProcessDefinitionKey; v != "" {
		f.ProcessDefinitionKey, populated = v, true
	}
	if s := flagGetPIState; s != "" && s != "all" {
		if st, ok := process.ParseState(s); ok {
			f.State, populated = st, true
		}
	}
	return f, populated
}

func pickPISearchSize() int32 {
	if flagGetPISize <= 0 || flagGetPISize > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return flagGetPISize
}

func validatePISearchFlags() error {
	if flagGetPIProcessDefinitionKey != "" &&
		(flagGetPIBpmnProcessID != "" ||
			flagGetPIProcessVersion != 0 ||
			flagGetPIProcessVersionTag != "") {
		return fmt.Errorf("%w: --pd-key is mutually exclusive with --bpmn-process-id, --pd-version, and --pd-version-tag", ErrMutuallyExclusiveFlags)
	}
	if flagGetPIBpmnProcessID == "" &&
		(flagGetPIProcessVersion != 0 || flagGetPIProcessVersionTag != "") {
		return fmt.Errorf("%w: --pd-version and --pd-version-tag require --bpmn-process-id to be set", ErrMissingDependentFlags)
	}
	if flagGetPIChildrenOnly && flagGetPIRootsOnly {
		return fmt.Errorf("%w: using both --children-only and --roots-only filters returns does not make sense", ErrForbiddenFlagCombination)
	}
	if flagGetPIIncidentsOnly && flagGetPINoIncidentsOnly {
		return fmt.Errorf("%w: using both --incidents-only and --no-incidents-only filters does not make sense", ErrForbiddenFlagCombination)
	}
	return nil
}
