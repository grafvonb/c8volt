package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
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

	flagRunPIVars string // JSON string with variables
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
		if cmd.Flags().Changed("count") && flagRunPICount < 1 || cmd.Flags().Changed("workers") && flagRunPIWorkers < 1 {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("--count and --workers must be positive integers"))
		}
		var vars map[string]interface{}
		if flagRunPIVars != "" {
			if err := json.Unmarshal([]byte(flagRunPIVars), &vars); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("parsing --vars JSON: %w", err))
			}
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
					Variables:                   vars,
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
					Variables:                vars,
					TenantId:                 cfg.App.Tenant,
				})
			}
			contextForErr = fmt.Sprintf("BPMN process ID(s) %v", flagRunPIProcessDefinitionBpmnProcessIds)

		default:
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("provide either --pd-id or --bpmn-process-id"))
		}

		fopts := collectOptions()
		if flagRunPIFailFast {
			fopts = append(fopts, foptions.WithFailFast())
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
		_, err = cli.CreateNProcessInstances(cmd.Context(), datas[0], flagRunPICount, flagRunPIWorkers, fopts...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("running %d process instances for %s: %w", flagRunPICount, contextForErr, err))
		}
	},
}

func init() {
	runCmd.AddCommand(runProcessInstanceCmd)

	fs := runProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagRunPIProcessDefinitionBpmnProcessIds, "bpmn-process-id", "b", nil, "BPMN process ID(s) to run process instance for (mutually exclusive with --pd-id). Runs latest version unless --pd-version is specified")
	fs.Int32Var(&flagRunPIProcessDefinitionVersion, "pd-version", 0, "specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)")
	fs.StringSliceVar(&flagRunPIProcessDefinitionSpecificId, "pd-id", nil, "specific process definition ID(s) to run process instance for (mutually exclusive with --bpmn-process-id)")

	fs.IntVarP(&flagRunPICount, "count", "n", 1, "number of instances to start for a single process definition")
	fs.IntVarP(&flagRunPIWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, GOMAXPROCS))")
	fs.BoolVar(&flagRunPIFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	fs.StringVar(&flagRunPIVars, "vars", "", "JSON-encoded variables to pass to the started process instance(s)")
}
