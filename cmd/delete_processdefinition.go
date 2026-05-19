// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

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
	Long: "Delete process definition resources from Camunda.\n\n" +
		"By default c8volt first checks delete impact without changing anything: active process instances, required cancellation roots and process-instance tree scope when --force is used, and batch-operation read access before prompting. Process-definition deletion requires Camunda 8.9 or newer so c8volt can request full process-definition history deletion. With --force, it cancels the root process instances, deletes the affected process-instance history, then asks Camunda to delete the process definition and remaining associated history. If you only want to delete process instances for a definition, use `c8volt delete process-instance --bpmn-process-id <bpmn-process-id>`.\n\n" +
		"Use --auto-confirm for unattended destructive runs.",
	Example: `  ./c8volt delete pd --key <process-definition-key> --auto-confirm
  ./c8volt delete pd --bpmn-process-id <bpmn-process-id> --latest --force
  ./c8volt delete pd --bpmn-process-id <bpmn-process-id> --latest --auto-confirm
  ./c8volt get pd --bpmn-process-id <bpmn-process-id> --latest --json
  ./c8volt get pd --bpmn-process-id <bpmn-process-id> --latest --keys-only | ./c8volt delete pd --auto-confirm -`,
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
		keys := mergeAndValidateKeys(flagDeletePDKeys, stdinKeys, log, cfg)
		if len(keys) == 0 && flagDeletePDBpmnProcessId == "" {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, missingDependentFlagsf("either --key, stdin keys, or --bpmn-process-id must be provided to delete process definition(s)"))
		}
		if err := validateDeleteProcessDefinitionSupportedVersion(cfg.App.CamundaVersion); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}

		switch {
		case len(keys) > 0:
		default:
			filter := process.ProcessDefinitionFilter{
				BpmnProcessId:     flagDeletePDBpmnProcessId,
				ProcessVersion:    flagDeletePDProcessVersion,
				ProcessVersionTag: flagDeletePDProcessVersionTag,
			}
			var pds process.ProcessDefinitions
			if !flagDeletePDLatest {
				pds, err = cli.SearchProcessDefinitions(cmd.Context(), filter)
			} else {
				pds, err = cli.SearchProcessDefinitionsLatest(cmd.Context(), filter)
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

		impactPlan, err := cli.PreviewDeleteProcessDefinitions(cmd.Context(), keys, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("checking process-definition delete impact: %w", err))
		}
		renderDeleteProcessDefinitionImpact(cmd, impactPlan)
		totals := impactPlan.Totals()
		if !flagNoStateCheck && !flagForce && totals.ActiveProcessInstances > 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("%d active process instance(s) block deletion; use --force to cancel them before deleting process definitions", totals.ActiveProcessInstances)))
		}
		if !flagNoWait {
			if err := cli.CheckBatchOperationReadAccess(cmd.Context(), collectOptions()...); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("cannot confirm asynchronous history deletion because this identity cannot read Camunda batch operations: %w", err)))
			}
		}
		prompt := "Proceed with this deletion?"
		if err := confirmCmdOrAbort(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		reports, err := cli.DeleteProcessDefinitions(cmd.Context(), keys, flagWorkers, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("deleting process definition(s): %w", err))
		}
		if err := renderCommandResult(cmd, reports); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render delete result: %w", err))
		}
	},
}

func validateDeleteProcessDefinitionSupportedVersion(version toolx.CamundaVersion) error {
	if version == toolx.V89 {
		return nil
	}
	return fmt.Errorf("%w: process-definition deletion requires Camunda 8.9 or newer for full history deletion; configured Camunda version is %s; to delete process instances for a process definition instead, use c8volt delete process-instance --bpmn-process-id <bpmn-process-id>", d.ErrUnsupported, version.String())
}

func renderDeleteProcessDefinitionImpact(cmd *cobra.Command, plan resource.DeleteProcessDefinitionPlan) {
	totals := plan.Totals()
	if plan.StateCheckSkipped {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); process-instance state check skipped; no changes made yet", totals.ProcessDefinitions)
		renderHumanWarningLine(cmd, "Deletion is irreversible: process-definition resources and associated history will be removed.")
		return
	}
	if totals.ActiveProcessInstances == 0 {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); no active process instances found; no changes made yet", totals.ProcessDefinitions)
		renderHumanWarningLine(cmd, "Deletion is irreversible: process-definition resources and associated history will be removed.")
		return
	}
	if flagForce {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); %d active process instance(s) found; no changes made yet", totals.ProcessDefinitions, totals.ActiveProcessInstances)
		renderHumanLine(cmd, "--force will cancel %d root process instance(s), then delete %d affected process instance(s), before deleting process definitions", totals.CancellationRoots, totals.CancellationAffected)
	} else {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); %d active process instance(s) found; no changes made yet", totals.ProcessDefinitions, totals.ActiveProcessInstances)
		return
	}
	for _, warning := range plan.Warnings {
		renderHumanWarningLine(cmd, "%s", warning)
	}
	renderHumanWarningLine(cmd, "Deletion is irreversible: process-definition resources and associated history will be removed.")
}

func init() {
	deleteCmd.AddCommand(deleteProcessDefinitionCmd)

	fs := deleteProcessDefinitionCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion work is accepted")
	fs.BoolVar(&flagNoStateCheck, "no-state-check", false, "skip checking process-instance state before deleting")
	fs.StringSliceVarP(&flagDeletePDKeys, "key", "k", nil, "process definition key(s) to delete")
	fs.StringVarP(&flagDeletePDBpmnProcessId, "bpmn-process-id", "b", "", "BPMN process ID of the process definition (all versions) to delete")
	fs.Int32Var(&flagDeletePDProcessVersion, "pd-version", 0, "process definition version")
	fs.StringVar(&flagDeletePDProcessVersionTag, "pd-version-tag", "", "process definition version tag")
	fs.BoolVar(&flagDeletePDLatest, "latest", false, "fetch the latest version(s) of the given BPMN process(s)")

	fs.BoolVar(&flagForce, "force", false, "force cancellation of the process instance(s), prior to deletion")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when --count > 1 (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling new instances after the first error")

	setCommandMutation(deleteProcessDefinitionCmd, CommandMutationStateChanging)
	setContractSupport(deleteProcessDefinitionCmd, ContractSupportFull)
	setAutomationSupport(deleteProcessDefinitionCmd, AutomationSupportFull, "supports unattended destructive confirmation")
}
