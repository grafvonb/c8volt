// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

var (
	flagDeletePDKeys              []string
	flagDeletePDBpmnProcessId     string
	flagDeletePDProcessVersion    int32
	flagDeletePDProcessVersionTag string
	flagDeletePDLatest            bool
)

var deleteProcessDefinitionCmd = &cobra.Command{
	Use:   "process-definition",
	Short: "Delete process definition resources",
	Long: `Delete process definition resources from Camunda 8.9 or newer.

Before mutation, c8volt checks unfinished-instance impact (ACTIVE and SUSPENDED on Camunda 8.10; ACTIVE on older versions), required cancellation roots, affected instance families, and batch-operation read access. With --force, it cancels root instances, verifies that no unfinished instances remain, deletes affected instance history, then deletes the definition and remaining associated history.

Camunda 8.10 can defer history deletion without returning a batch operation. Unless --no-wait is set, c8volt waits for a returned batch to complete and for the definition to disappear. Without a returned batch, definition disappearance is confirmed, but completion of all associated history deletion is not independently verified.

--no-wait skips final deletion confirmation, but still verifies unfinished work has drained before dependent history deletion. --no-state-check skips impact reads and automatic cancellation, retaining the existing backend-enforced bypass.

--tenant limits BPMN selector discovery. An empty tenant or --all-tenants leaves discovery unfiltered across accessible tenants. Explicit --key and stdin keys use backend authorization without tenant filtering. A --bpmn-process-id selector must match visible definitions before impact planning.

Use --dry-run to preview impact without mutation, or --auto-confirm for unattended deletion. To delete only a definition's instances, use delete process-instance --bpmn-process-id <bpmn-process-id>.`,
	Example: `  ./c8volt delete process-definition --key <process-definition-key> --auto-confirm
  ./c8volt delete process-definition --key <process-definition-key> --dry-run
  ./c8volt --tenant tenant-a delete process-definition --key <process-definition-key> --dry-run
  ./c8volt --tenant tenant-a delete process-definition --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt --all-tenants delete process-definition --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --force
  ./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --dry-run
  ./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
  ./c8volt --verbose delete process-definition --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --json
  ./c8volt get process-definition --bpmn-process-id <bpmn-process-id> --latest --keys-only | ./c8volt delete process-definition --auto-confirm -`,
	Aliases: []string{"pd"},
	Args: func(cmd *cobra.Command, args []string) error {
		return validateOptionalDashArg(args)
	},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if cmd.Flags().Changed("workers") && flagWorkers < 1 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("--workers must be positive integer"))
		}
		stdinKeys, err := readKeysIfDash(args) // only reads when args == []{"-"}
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(cmd, flagDeletePDKeys, stdinKeys, log, cfg)
		explicitInput := len(keys) > 0
		callOpts := collectOptions()
		if explicitInput {
			callOpts = collectExplicitAdminInputOptions()
		}
		if len(keys) == 0 && flagDeletePDBpmnProcessId == "" {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("either --key, stdin keys, or --bpmn-process-id must be provided to delete process definition(s)"))
		}
		if err := validateDeleteProcessDefinitionSupportedVersion(cfg.App.CamundaVersion); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		switch {
		case len(keys) > 0:
		default:
			var pds process.ProcessDefinitions
			if flagDeletePDBpmnProcessId != "" {
				result, err := validateProcessDefinitionSelectorsForCommand(cmd.Context(), cmd, cli, newDeletePDProcessDefinitionSelectorValidationRequest(), collectOptions()...)
				if err != nil {
					handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
				}
				if !result.Valid() {
					handleProcessDefinitionSelectorValidationError(cmd, log, cfg.App.NoErrCodes, cli, result)
				}
				if len(result.Request.BpmnProcessIds) > 0 {
					pds = result.MatchesByBpmnProcessID[result.Request.BpmnProcessIds[0]]
				}
			} else {
				filter := process.ProcessDefinitionFilter{
					BpmnProcessId:     flagDeletePDBpmnProcessId,
					ProcessVersion:    flagDeletePDProcessVersion,
					ProcessVersionTag: flagDeletePDProcessVersionTag,
				}
				if flagDeletePDLatest {
					pds, err = cli.SearchProcessDefinitionsLatest(cmd.Context(), filter)
				} else {
					pds, err = cli.SearchProcessDefinitions(cmd.Context(), filter)
				}
			}
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("searching for process definitions to delete: %w", err))
			}
			keys = make([]string, 0, len(pds.Items))
			for _, pd := range pds.Items {
				keys = append(keys, pd.Key)
			}
		}
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no process definitions found to delete")))
		}

		impactPlan, err := cli.PreviewDeleteProcessDefinitions(cmd.Context(), keys, callOpts...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("checking process-definition delete impact: %w", err), formatProcessDefinitionDeleteFailure("checking delete impact", len(keys), resource.DeleteReports{}, err))
		}
		tenantCtx := newDiscoveryTenantContext(configuredTenantID(cfg))
		if explicitInput {
			tenantCtx = newExplicitKeysTenantContext(configuredTenantID(cfg))
		}
		evidence := impactPlan.TenantEvidence()
		attachTenantContext(cmd, withTenantContextEvidence(tenantCtx, evidence.ResolvedTenantIDs, evidence.UnknownTargetCount))
		if flagDryRun {
			if err := renderDeleteProcessDefinitionDryRun(cmd, impactPlan); err != nil {
				log.Error("could not render process-definition delete preview")
				exitAfterRenderedResult(cfg.App.NoErrCodes, err)
			}
			return
		}
		renderDeleteProcessDefinitionImpact(cmd, impactPlan)
		totals := impactPlan.Totals()
		if !flagNoStateCheck && !flagForce && totals.UnfinishedProcessInstances > 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("%s block deletion; use --force to cancel them before deleting process definitions", processDefinitionUnfinishedSummary(totals))))
		}
		if !flagNoWait {
			if err := cli.CheckBatchOperationReadAccess(cmd.Context(), callOpts...); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("cannot confirm asynchronous history deletion because this identity cannot read Camunda batch operations: %w", err)))
			}
		}
		prompt := "Proceed with this deletion?"
		if err := confirmCmdOrAbort(cmd.ErrOrStderr(), shouldImplicitlyConfirm(cmd), prompt); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		deletionProgress := newProcessDefinitionDeleteSemanticProgress(cmd, len(keys))
		deletionProgress.Start(len(keys))
		deleteOpts := append([]foptions.FacadeOption{}, callOpts...)
		deleteOpts = append(deleteOpts, foptions.WithProgress(deletionProgress.FacadeProgress))
		if !flagVerbose {
			deleteOpts = append(deleteOpts, foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())
		}
		reports, err := cli.DeleteProcessDefinitions(cmd.Context(), keys, flagWorkers, deleteOpts...)
		deletionProgress.Close()
		if err != nil {
			runErr := fmt.Errorf("deleting process definition(s): %w", err)
			if renderErr := renderDeleteProcessDefinitionFailure(cmd, log, len(keys), reports, runErr); renderErr != nil {
				log.Error("could not render process-definition delete result")
			}
			exitAfterRenderedResult(cfg.App.NoErrCodes, runErr)
		}
		if err := renderDeleteProcessDefinitionResult(cmd, reports); err != nil {
			log.Error("could not render process-definition delete result")
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
	},
}

func validateDeleteProcessDefinitionSupportedVersion(version toolx.CamundaVersion) error {
	if toolx.SupportsFullProcessDefinitionHistoryDeletion(version) {
		return nil
	}
	return fmt.Errorf("%w: process-definition deletion requires the full process-definition history deletion capability, currently Camunda 8.9 or newer; configured Camunda version is %s; to delete process instances for a process definition instead, use c8volt delete process-instance --bpmn-process-id <bpmn-process-id>", d.ErrUnsupported, version.String())
}

func init() {
	deleteCmd.AddCommand(deleteProcessDefinitionCmd)

	fs := deleteProcessDefinitionCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip final deletion confirmation; required unfinished-instance drain checks still run")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking process-instance state before deleting")
	fs.BoolVar(&flagDryRun, "dry-run", false, "preview process-definition delete impact without submitting deletion or cancellation requests")
	fs.StringSliceVarP(&flagDeletePDKeys, "key", "k", nil, "process definition key(s) to delete")
	fs.StringVarP(&flagDeletePDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID of the process definition (all versions) to delete")
	fs.Int32Var(&flagDeletePDProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagDeletePDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagDeletePDLatest, "latest", false, "fetch the latest version(s) of the given BPMN process(s)")

	fs.BoolVar(&flagForce, "force", false, "cancel unfinished process-instance roots before deleting history and definitions")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(deleteProcessDefinitionCmd, CommandMutationStateChanging)
	setContractSupport(deleteProcessDefinitionCmd, ContractSupportFull)
	setAutomationSupport(deleteProcessDefinitionCmd, AutomationSupportFull, "supports unattended destructive confirmation and non-mutating dry-run previews")
}
