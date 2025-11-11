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
	flagGetPIRootsOnly         bool
	flagGetPIChildrenOnly      bool
	flagGetPIOrphanParentsOnly bool
	flagGetPIIncidentsOnly     bool
	flagGetPINoIncidentsOnly   bool
)

var getProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Get process instances",
	Aliases: []string{"process-instances", "pi", "pis"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error creating c8volt client: %w", err))
		}
		if flagGetPIProcessDefinitionKey != "" && (flagGetPIBpmnProcessID != "" || flagGetPIProcessVersion != 0 || flagGetPIProcessVersionTag != "") {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("%w: --pd-key is mutually exclusive with --bpmn-process-id, --pd-version, and --pd-version-tag", ferrors.ErrBadRequest))
		}
		if flagGetPIBpmnProcessID == "" && (flagGetPIProcessVersion != 0 || flagGetPIProcessVersionTag != "") {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("%w: --pd-version and --pd-version-tag require --bpmn-process-id to be set", ferrors.ErrBadRequest))
		}

		log.Debug(fmt.Sprintf("fetching process instances, render mode: %s", pickMode()))
		filter, _ := populatePISearchFilterOpts()

		if filter.Key != "" {
			log.Debug(fmt.Sprintf("searching by key: %s", filter.Key))
			pi, err := cli.GetProcessInstance(cmd.Context(), filter.Key)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching process instance by key %s: %w", filter.Key, err))
			}
			err = processInstanceView(cmd, pi)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error rendering view: %w", err))
			}
			log.Debug(fmt.Sprintf("searched by key, found process instance with key: %s", pi.Key))
			return
		}

		log.Debug(fmt.Sprintf("searching by filter: %v", filter))
		pisr, err := cli.SearchProcessInstances(cmd.Context(), filter, pickPISearchSize())
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error fetching process instances: %w", err))
		}
		if flagGetPIChildrenOnly && flagGetPIRootsOnly {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("%w: using both --children-only and --roots-only filters returns always no results", ferrors.ErrBadRequest))
		}
		if flagGetPIChildrenOnly {
			pisr = pisr.FilterChildrenOnly()
		}
		if flagGetPIRootsOnly {
			pisr = pisr.FilterRootsOnly()
		}
		if flagGetPIOrphanParentsOnly {
			pisr.Items, err = cli.FilterProcessInstanceWithOrphanParent(cmd.Context(), pisr.Items)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error filtering orphan parents: %w", err))
			}
		}
		if flagGetPIIncidentsOnly {
			pisr = pisr.FilterByHavingIncidents(true)
		}
		if flagGetPINoIncidentsOnly {
			pisr = pisr.FilterByHavingIncidents(false)
		}
		err = listProcessInstancesView(cmd, pisr)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("error rendering items view: %w", err))
		}
		log.Debug(fmt.Sprintf("fetched process instances: %d", pisr.Total))
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
	fs.BoolVar(&flagGetPIOrphanParentsOnly, "orphan-parents-only", false, "show only child instances where parent key is set but the parent process instance does not exist (anymore)")
	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")
}

func populatePISearchFilterOpts() (process.ProcessInstanceFilter, bool) {
	var f process.ProcessInstanceFilter
	var populated bool

	if v := flagGetPIKey; v != "" {
		f.Key, populated = v, true
	}
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
