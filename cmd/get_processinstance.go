package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

const maxPISearchSize int32 = 1000

var (
	flagGetPIKey               string
	flagGetPIBpmnProcessID     string
	flagGetPIProcessVersion    int32
	flagGetPIProcessVersionTag string
	flagGetPIState             string
	flagGetPIParentKey         string
)

// command options
var (
	flagGetPIParentsOnly       bool
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
		cli, log, _, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, err)
		}

		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("error creating c8volt client: %w", err))
		}

		log.Debug(fmt.Sprintf("fetching process instances, render mode: %s", pickMode()))
		searchFilterOpts := populatePISearchFilterOpts()
		printFilter(cmd)
		if searchFilterOpts.Key != "" {
			log.Debug(fmt.Sprintf("searching by key: %s", searchFilterOpts.Key))
			pi, err := cli.GetProcessInstanceByKey(cmd.Context(), searchFilterOpts.Key)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("error fetching process instance by key %s: %w", searchFilterOpts.Key, err))
			}
			err = processInstanceView(cmd, pi)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("error rendering key-only view: %w", err))
			}
			log.Debug(fmt.Sprintf("searched by key, found process instance with key: %s", pi.Key))
		} else {
			log.Debug(fmt.Sprintf("searching by filter: %v", searchFilterOpts))
			pisr, err := cli.SearchProcessInstances(cmd.Context(), searchFilterOpts, maxPISearchSize)
			if err != nil {
				ferrors.HandleAndExit(log, fmt.Errorf("error fetching process instances: %w", err))
			}
			if flagGetPIChildrenOnly && flagGetPIParentsOnly {
				ferrors.HandleAndExit(log, fmt.Errorf("%w: using both --children-only and --parents-only filters returns always no results", ferrors.ErrBadRequest))
			}
			if flagGetPIChildrenOnly {
				pisr = pisr.FilterChildrenOnly()
			}
			if flagGetPIParentsOnly {
				pisr = pisr.FilterParentsOnly()
			}
			if flagGetPIOrphanParentsOnly {
				pisr.Items, err = cli.FilterProcessInstanceWithOrphanParent(cmd.Context(), pisr.Items)
				if err != nil {
					ferrors.HandleAndExit(log, fmt.Errorf("error filtering orphan parents: %w", err))
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
				ferrors.HandleAndExit(log, fmt.Errorf("error rendering items view: %w", err))
			}
			log.Debug(fmt.Sprintf("fetched process instances: %d", pisr.Total))
		}
	},
}

func init() {
	getCmd.AddCommand(getProcessInstanceCmd)

	fs := getProcessInstanceCmd.Flags()
	fs.StringVarP(&flagGetPIKey, "key", "k", "", "process instance key to fetch")
	fs.StringVarP(&flagGetPIBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter process instances")
	fs.Int32VarP(&flagGetPIProcessVersion, "process-version", "v", 0, "process definition version")
	fs.StringVar(&flagGetPIProcessVersionTag, "process-version-tag", "", "process definition version tag")

	// filtering options
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to filter process instances")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled")
	fs.BoolVar(&flagGetPIParentsOnly, "parents-only", false, "show only parent process instances, meaning instances with no parent key set")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "show only child process instances, meaning instances that have a parent key set")
	fs.BoolVar(&flagGetPIOrphanParentsOnly, "orphan-parents-only", false, "show only child instances whose parent does not exist (return 404 on get by key)")
	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "show only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "show only process instances that have no incidents")
}

func populatePISearchFilterOpts() process.ProcessInstanceSearchFilterOpts {
	var filter process.ProcessInstanceSearchFilterOpts
	if flagGetPIKey != "" {
		filter.Key = flagGetPIKey
	}
	if flagGetPIParentKey != "" {
		filter.ParentKey = flagGetPIParentKey
	}
	if flagGetPIBpmnProcessID != "" {
		filter.BpmnProcessId = flagGetPIBpmnProcessID
	}
	if flagGetPIProcessVersion != 0 {
		filter.ProcessVersion = flagGetPIProcessVersion
	}
	if flagGetPIProcessVersionTag != "" {
		filter.ProcessVersionTag = flagGetPIProcessVersionTag
	}
	if flagGetPIState != "" && flagGetPIState != "all" {
		if state, ok := process.ParseState(flagGetPIState); ok {
			filter.State = state
		}
	}
	return filter
}
