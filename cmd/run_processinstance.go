package cmd

import (
	"fmt"
	"runtime"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/options"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagRunPIProcessDefinitionBpmnProcessIds []string
	flagRunPIProcessDefinitionSpecificId     []string
	flagRunPIProcessDefinitionVersion        int32

	flagRunPICount    int
	flagRunPIWorkers  int
	flagRunPIFailFast bool
)

var runProcessInstanceCmd = &cobra.Command{
	Use:     "process-instance",
	Short:   "Run process instance(s) by process definition",
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		var datas []process.ProcessInstanceData
		var contextForErr string
		switch {
		case len(flagRunPIProcessDefinitionSpecificId) > 0:
			if len(flagRunPIProcessDefinitionBpmnProcessIds) > 0 {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("flags --pd-id and --bpmn-process-id are mutually exclusive"))
			}
			if flagRunPIProcessDefinitionVersion != 0 {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("flag --pd-version is only valid with --bpmn-process-id"))
			}

			datas = make([]process.ProcessInstanceData, 0, len(flagRunPIProcessDefinitionSpecificId))
			for _, pdID := range flagRunPIProcessDefinitionSpecificId {
				datas = append(datas, process.ProcessInstanceData{
					ProcessDefinitionSpecificId: pdID,
					TenantId:                    cfg.App.Tenant,
				})
			}
			contextForErr = fmt.Sprintf("process definition ID(s) %v", flagRunPIProcessDefinitionSpecificId)

		case len(flagRunPIProcessDefinitionBpmnProcessIds) > 0:
			if len(flagRunPIProcessDefinitionBpmnProcessIds) > 1 && flagRunPIProcessDefinitionVersion != 0 {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("cannot specify --pd-version when running multiple BPMN process IDs"))
			}

			datas = make([]process.ProcessInstanceData, 0, len(flagRunPIProcessDefinitionBpmnProcessIds))
			for _, bpmnID := range flagRunPIProcessDefinitionBpmnProcessIds {
				datas = append(datas, process.ProcessInstanceData{
					BpmnProcessId:            bpmnID,
					ProcessDefinitionVersion: flagRunPIProcessDefinitionVersion, // 0 = latest
					TenantId:                 cfg.App.Tenant,
				})
			}
			contextForErr = fmt.Sprintf("BPMN process ID(s) %v", flagRunPIProcessDefinitionBpmnProcessIds)

		default:
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("provide either --pd-id or --bpmn-process-id"))
		}

		fopts := collectOptions()
		if flagRunPIFailFast {
			fopts = append(fopts, options.WithFailFast())
		}
		if flagRunPICount <= 1 {
			_, err = cli.CreateProcessInstances(cmd.Context(), datas, fopts...)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("running process instance(s) for %s: %w", contextForErr, err))
			}
			return
		}
		if len(datas) > 1 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes,
				fmt.Errorf("--count requires exactly one target definition; got %d", len(datas)))
		}

		workers := flagRunPIWorkers
		if workers <= 0 {
			workers = flagRunPICount
			if gp := runtime.GOMAXPROCS(0); gp < workers {
				workers = gp
			}
		}
		if workers > flagRunPICount {
			workers = flagRunPICount
		}

		_, err = cli.CreateNProcessInstances(cmd.Context(), datas[0], flagRunPICount, workers, fopts...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("running %d process instances for %s: %w", flagRunPICount, contextForErr, err))
		}
	},
}

func init() {
	runCmd.AddCommand(runProcessInstanceCmd)

	runProcessInstanceCmd.Flags().StringSliceVarP(&flagRunPIProcessDefinitionBpmnProcessIds, "bpmn-process-id", "b", nil,
		"BPMN process ID(s) to run process instance for (mutually exclusive with --pd-id). Runs latest version unless --pd-version is specified",
	)
	runProcessInstanceCmd.Flags().Int32Var(&flagRunPIProcessDefinitionVersion, "pd-version", 0,
		"Specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)",
	)
	runProcessInstanceCmd.Flags().StringSliceVar(
		&flagRunPIProcessDefinitionSpecificId, "pd-id", nil,
		"Specific process definition ID(s) to run process instance for (mutually exclusive with --bpmn-process-id)",
	)
	runProcessInstanceCmd.Flags().IntVarP(&flagRunPICount, "count", "n", 1,
		"Number of instances to start for a single process definition",
	)
	runProcessInstanceCmd.Flags().IntVarP(&flagRunPIWorkers, "workers", "w", 0,
		"Maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))",
	)
	runProcessInstanceCmd.Flags().BoolVar(&flagRunPIFailFast, "fail-fast", false,
		"Stop scheduling new instances after the first error",
	)
}
