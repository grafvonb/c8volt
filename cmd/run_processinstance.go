// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagRunPIProcessDefinitionBpmnProcessIds []string
	flagRunPIProcessDefinitionKey            []string
	flagRunPIProcessDefinitionVersion        int32

	flagRunPICount int
	flagRunPIVars  string // JSON string with variables
)

var runProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Start process instances and confirm activation",
	Long: "Start process instances and confirm activation.\n\n" +
		"Run by BPMN process ID for the latest version, or by process definition key for an exact definition.\n\n" +
		"When running by BPMN process ID, c8volt validates all requested process definitions before creating anything. Mixed visible and missing BPMN IDs fail as one request, so no partial process instances are started; automation-oriented modes never prompt for recovery output.\n\n" +
		"By default c8volt waits for active instances.",
	Example: `  ./c8volt run pi -b C89_SimpleUserTask_Process
  ./c8volt run pi -b C89_SimpleUserTask_Process --vars '{"customerId":"1234"}'
  ./c8volt run pi -b C89_SimpleUserTask_Process -n 3 --workers 2
  ./c8volt --json run pi -b C89_SimpleUserTask_Process --vars '{"customerId":"1234"}'
  ./c8volt expect pi --key <process-instance-key> --state active`,
	Aliases: []string{"pi"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("count") && flagRunPICount < 1 || cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--count and --workers must be positive integers"))
		}
		var vars map[string]any
		if flagRunPIVars != "" {
			if err := json.Unmarshal([]byte(flagRunPIVars), &vars); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("parsing --vars JSON: %v", err))
			}
		}
		var datas []process.ProcessInstanceData
		var contextForErr string
		tenantID := cfg.App.TargetTenant()
		fopts := collectOptions()
		switch {
		case len(flagRunPIProcessDefinitionKey) > 0:
			if len(flagRunPIProcessDefinitionBpmnProcessIds) > 0 {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, mutuallyExclusiveFlagsf("flags --pd-key and --bpmn-process-id are mutually exclusive"))
			}
			if flagRunPIProcessDefinitionVersion != 0 {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("flag --pd-version is only valid with --bpmn-process-id"))
			}

			datas = make([]process.ProcessInstanceData, 0, len(flagRunPIProcessDefinitionKey))
			for _, pdID := range flagRunPIProcessDefinitionKey {
				datas = append(datas, process.ProcessInstanceData{
					ProcessDefinitionSpecificId: pdID,
					Variables:                   vars,
					TenantId:                    tenantID,
				})
			}
			contextForErr = fmt.Sprintf("process definition ID(s) %v", flagRunPIProcessDefinitionKey)

		case len(flagRunPIProcessDefinitionBpmnProcessIds) > 0:
			if len(flagRunPIProcessDefinitionBpmnProcessIds) > 1 && flagRunPIProcessDefinitionVersion != 0 {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, forbiddenFlagCombinationf("cannot specify --pd-version when running multiple BPMN process IDs"))
			}

			result, err := validateProcessDefinitionSelectors(cmd.Context(), cli, newRunPIProcessDefinitionSelectorValidationRequest(), fopts...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			handleProcessDefinitionSelectorValidationError(cmd, log, cfg.App.NoErrCodes, cli, result)

			datas = make([]process.ProcessInstanceData, 0, len(flagRunPIProcessDefinitionBpmnProcessIds))
			for _, bpmnID := range flagRunPIProcessDefinitionBpmnProcessIds {
				datas = append(datas, process.ProcessInstanceData{
					BpmnProcessId:            bpmnID,
					ProcessDefinitionVersion: flagRunPIProcessDefinitionVersion, // 0 = latest
					Variables:                vars,
					TenantId:                 tenantID,
				})
			}
			contextForErr = fmt.Sprintf("BPMN process ID(s) %v", flagRunPIProcessDefinitionBpmnProcessIds)

		default:
			handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("provide either --pd-key or --bpmn-process-id"))
		}

		if flagFailFast {
			fopts = append(fopts, foptions.WithFailFast())
		}
		if flagRunPICount <= 1 {
			var created []process.ProcessInstance
			created, err = cli.CreateProcessInstances(cmd.Context(), datas, fopts...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("running process instance(s) for %s: %w", contextForErr, err))
			}
			if err := renderCommandResult(cmd, process.ProcessInstances{
				Total: int32(len(created)),
				Items: created,
			}); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render process-instance result: %w", err))
			}
			return
		}
		if len(datas) > 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes,
				forbiddenFlagCombinationf("--count requires exactly one process definition; got %d", len(datas)))
		}
		created, err := cli.CreateNProcessInstances(cmd.Context(), datas[0], flagRunPICount, flagWorkers, fopts...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("running %d process instances for %s: %w", flagRunPICount, contextForErr, err))
		}
		if err := renderCommandResult(cmd, process.ProcessInstances{
			Total: int32(len(created)),
			Items: created,
		}); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render process-instance result: %w", err))
		}
	},
}

func init() {
	runCmd.AddCommand(runProcessInstanceCmd)

	fs := runProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagRunPIProcessDefinitionBpmnProcessIds, "bpmn-process-id", "b", nil, "BPMN process ID(s) to run process instance for (mutually exclusive with --pd-key). Runs latest version unless --pd-version is specified")
	fs.Int32Var(&flagRunPIProcessDefinitionVersion, "pd-version", 0, "specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)")
	fs.StringSliceVar(&flagRunPIProcessDefinitionKey, "pd-key", nil, "specific process definition key(s) to run process instance for (mutually exclusive with --bpmn-process-id)")
	fs.IntVarP(&flagRunPICount, "count", "n", 1, "number of instances to start for a single process definition")
	fs.StringVar(&flagRunPIVars, "vars", "", "JSON-encoded variables to pass to the started process instance(s)")

	fs.BoolVar(&flagNoWait, "no-wait", false, "return after creation is accepted")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(runProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(runProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(runProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and accepted results")
}
