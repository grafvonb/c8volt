// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var (
	flagRunPIProcessDefinitionBpmnProcessIds []string
	flagRunPIProcessDefinitionKey            []string
	flagRunPIProcessDefinitionVersion        int32

	flagRunPICount int
	flagRunPIVars  string // JSON string with variables
	// flagRunPIBusinessID is scalar; flag presence distinguishes omission from explicit empty.
	flagRunPIBusinessID string
)

var runProcessInstanceCmd = &cobra.Command{
	Use:   "process-instance",
	Short: "Start process instances and confirm creation",
	Long: `Start process instances and confirm creation.

Use a BPMN process ID for the latest version or a process-definition key for an exact definition. All requested BPMN IDs must be visible before any instance is started.

Creation uses the configured tenant, or the default tenant when none is configured. --all-tenants is not supported because creation requires one destination tenant.

By default c8volt waits until created instances are observable as ACTIVE, SUSPENDED, COMPLETED, CANCELED, or TERMINATED. This confirms creation visibility, not execution completion. Use --no-wait to return after creation is accepted.

On Camunda 8.10, --bid supplies one opaque business ID with either selector. Values are not trimmed or split at commas. A supplied value must be nonempty and at most 256 UTF-16 code units (a supplementary character counts as two). Camunda 8.8 and 8.9 reject this input.

Every --count attempt uses the same business ID. Backend uniqueness enforcement may reject repeated active roots for the same definition and tenant. Successful results survive later failures; JSON includes them in one failure envelope and keys-only prints their keys with the original failure exit status. --no-wait reports acceptance only. Human creation rows and summaries use the configured INFO logger, with bid:<value> beside the process name and quoting only when needed.`,
	Example: `  ./c8volt run pi --bpmn-process-id order --bid 'order,AbC'
  ./c8volt run pi --pd-key 2251799813685249 --bid 'order-123' --no-wait
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id>
  ./c8volt --tenant tenant-a run process-instance --bpmn-process-id <bpmn-process-id>
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --vars '{"customerId":"1234"}'
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --count 3 --workers 2
  ./c8volt --json run process-instance --bpmn-process-id <bpmn-process-id> --vars '{"customerId":"1234"}'
  ./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --keys-only | ./c8volt expect process-instance --state completed -
  ./c8volt run process-instance --bpmn-process-id <long-running-bpmn-process-id> --keys-only | ./c8volt expect process-instance --state active -`,
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
		var businessID *string
		if cmd.Flags().Changed("bid") {
			businessID = toolx.CopyPtr(&flagRunPIBusinessID)
		}
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
					BusinessId:                  toolx.CopyPtr(businessID),
				})
			}

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
					BusinessId:               toolx.CopyPtr(businessID),
				})
			}

		default:
			handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("provide either --pd-key or --bpmn-process-id"))
		}

		tenantCtx := attachCreationTenantContext(cmd, cfg)
		renderTenantContext(cmd, tenantCtx)

		if flagFailFast {
			fopts = append(fopts, foptions.WithFailFast())
		}
		requested := len(datas)
		// The CLI owns final output; service details and raw progress errors must
		// not bypass its aggregate failure and configured logger contract.
		fopts = append(fopts, foptions.WithSuppressWorkflowDetailLogs())
		var created []process.ProcessInstance
		if flagRunPICount <= 1 {
			created, err = cli.CreateProcessInstances(cmd.Context(), datas, fopts...)
		} else {
			if len(datas) > 1 {
				handleCommandError(cmd, log, cfg.App.NoErrCodes,
					forbiddenFlagCombinationf("--count requires exactly one process definition; got %d", len(datas)))
			}
			requested = flagRunPICount
			runProgress := newRunProcessInstanceSemanticProgressReporter(cmd, requested)
			fopts = appendRunProcessInstanceProgressOption(cmd, fopts, runProgress)
			created, err = cli.CreateNProcessInstances(cmd.Context(), datas[0], requested, flagWorkers, fopts...)
			runProgress.Close()
			sortRunProcessInstancesForOutput(created)
		}
		attachTenantContext(cmd, withTenantContextEvidence(tenantCtx, processInstanceTenantIDs(created), 0))
		runErr := ferrors.Normalize(err)
		renderErr := renderRunProcessInstanceResult(cmd, process.ProcessInstances{
			Total: int32(len(created)),
			Items: created,
		}, runErr)
		if runErr != nil {
			if !commandUsesSharedEnvelope(cmd, pickMode()) {
				log.Error(formatRunProcessInstanceFailure(cmd, requested, len(created), runErr))
			}
			exitAfterRenderedResult(cfg.App.NoErrCodes, runErr)
		}
		if renderErr != nil {
			// A write was already attempted; do not render another envelope.
			if !commandUsesSharedEnvelope(cmd, pickMode()) {
				log.Error("render process-instance result: output write failed")
			}
			exitAfterRenderedResult(cfg.App.NoErrCodes, ferrors.Normalize(renderErr))
		}
	},
}

// init registers creation selectors, scalar identity and existing execution controls.
func init() {
	runCmd.AddCommand(runProcessInstanceCmd)

	fs := runProcessInstanceCmd.Flags()
	fs.StringSliceVarP(&flagRunPIProcessDefinitionBpmnProcessIds, "bpmn-process-id", "b", nil, "BPMN process ID(s) to run process instance for (mutually exclusive with --pd-key). Runs latest version unless --pd-version is specified")
	fs.Int32Var(&flagRunPIProcessDefinitionVersion, "pd-version", 0, "specific version of the process definition to use when running by BPMN process ID (supported only with --bpmn-process-id)")
	fs.StringSliceVar(&flagRunPIProcessDefinitionKey, "pd-key", nil, "specific process definition key(s) to run process instance for (mutually exclusive with --bpmn-process-id)")
	fs.IntVarP(&flagRunPICount, "count", "n", 1, "number of instances to start for a single process definition")
	fs.StringVar(&flagRunPIBusinessID, "bid", "", "opaque business ID (Camunda 8.10 only; nonempty, maximum 256 UTF-16 code units; reused unchanged by --count)")
	fs.StringVar(&flagRunPIVars, "vars", "", "JSON-encoded variables to pass to the started process instance(s)")

	fs.BoolVar(&flagNoWait, "no-wait", false, "return after creation is accepted")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(runProcessInstanceCmd, CommandMutationStateChanging)
	setContractSupport(runProcessInstanceCmd, ContractSupportFull)
	setAutomationSupport(runProcessInstanceCmd, AutomationSupportFull, "supports shared machine output and accepted results")
	setAllTenantsSupport(runProcessInstanceCmd, AllTenantsSupportRejectedConcreteDestination)
}
