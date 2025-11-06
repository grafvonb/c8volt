package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagRunPIProcessDefinitionBpmnProcessIds []string
	flagRunPIProcessDefinitionSpecificId     []string
	flagRunPIProcessDefinitionVersion        int32
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

		_, err = cli.CreateProcessInstances(cmd.Context(), datas, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("running process instance(s) for %s: %w", contextForErr, err))
		}
	},
}

func init() {
	runCmd.AddCommand(runProcessInstanceCmd)

	runProcessInstanceCmd.Flags().StringSliceVarP(&flagRunPIProcessDefinitionBpmnProcessIds, "bpmn-process-id", "b", nil, "BPMN process ID(s) to run process instance for (mutually exclusive with --pd-id). Runs latest version unless --pd-version is specified")
	runProcessInstanceCmd.Flags().Int32Var(&flagRunPIProcessDefinitionVersion, "pd-version", 0, "Specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)")
	runProcessInstanceCmd.Flags().StringSliceVar(&flagRunPIProcessDefinitionSpecificId, "pd-id", nil, "Specific process definition ID(s) to run process instance for (mutually exclusive with --bpmn-process-id)")
}
