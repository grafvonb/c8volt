// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

const opsPurgeAllProcessDefinitionsCommandName = "ops purge all-process-definitions"

var (
	flagOpsPurgeAllPDKey               string
	flagOpsPurgeAllPDBpmnProcessID     string
	flagOpsPurgeAllPDProcessVersion    int32
	flagOpsPurgeAllPDProcessVersionTag string
	flagOpsPurgeAllPDLatest            bool
	flagOpsPurgeAllPDReportFile        string
	flagOpsPurgeAllPDReportFormat      string
)

var opsPurgeAllProcessDefinitionsCmd = &cobra.Command{
	Use:   "all-process-definitions",
	Short: "Purge all selected process definitions",
	Long: "Purge all selected process definitions.\n\n" +
		"The workflow discovers candidate process-definition versions using the same filters as `get pd`, freezes the candidate keys, validates the existing delete plan, and then either reports the plan with --dry-run or submits deletion only after confirmation. Use --auto-confirm or --automation for unattended deletion, combine --automation with --json for deterministic machine output, and use --report-file to write an audit report.",
	Example: `  ./c8volt ops purge all-process-definitions --dry-run
  ./c8volt ops purge all-pds --bpmn-process-id invoice --latest --dry-run
  ./c8volt ops purge all-process-definitions --automation --json --dry-run
  ./c8volt ops purge all-process-definitions --dry-run --report-file process-definition-purge.md
  ./c8volt ops purge all-process-definitions --bpmn-process-id invoice --latest --auto-confirm --force
  ./c8volt ops purge all-process-definitions --bpmn-process-id invoice --latest --auto-confirm --force --workers 4 --report-file process-definition-purge.json --report-format json`,
	Aliases: []string{"all-pds"},
	Args:    validateOpsPurgeAllProcessDefinitionsArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	opsPurgeCmd.AddCommand(opsPurgeAllProcessDefinitionsCmd)
	useInvalidInputFlagErrors(opsPurgeAllProcessDefinitionsCmd)

	fs := opsPurgeAllProcessDefinitionsCmd.Flags()
	fs.StringVarP(&flagOpsPurgeAllPDKey, "key", "k", "", "process definition key to select for candidate discovery")
	fs.StringVarP(&flagOpsPurgeAllPDBpmnProcessID, "bpmn-process-id", "b", "", "BPMN process ID to filter candidate process definitions")
	fs.Int32Var(&flagOpsPurgeAllPDProcessVersion, "pd-version", 0, "process definition version to filter candidate discovery")
	fs.StringVar(&flagOpsPurgeAllPDProcessVersionTag, "pd-version-tag", "", "process definition version tag to filter candidate discovery")
	fs.BoolVar(&flagOpsPurgeAllPDLatest, "latest", false, "only include the latest matching process-definition version(s)")
	fs.BoolVar(&flagDryRun, "dry-run", false, "discover and validate process-definition cleanup without submitting deletion requests")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when validating the delete plan and deleting process definitions (default: min(targets, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling validation or deletion work after the first error")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deletion requests are accepted without deletion confirmation")
	fs.BoolVar(&flagForce, "force", false, "force cancellation of affected active process instances before deleting process definitions")
	fs.StringVar(&flagOpsPurgeAllPDReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsPurgeAllPDReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsPurgeAllProcessDefinitionsCmd, CommandMutationStateChanging)
	setContractSupport(opsPurgeAllProcessDefinitionsCmd, ContractSupportFull)
	setAutomationSupport(opsPurgeAllProcessDefinitionsCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed all-process-definitions purges with shared machine output")
	setOutputModes(opsPurgeAllProcessDefinitionsCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}

func validateOpsPurgeAllProcessDefinitionsArgs(cmd *cobra.Command, args []string) error {
	if err := cobra.NoArgs(cmd, args); err != nil {
		return silenceUsageForError(cmd, err)
	}
	return silenceUsageForError(cmd, validateOpsPurgeAllProcessDefinitionsFlags(cmd))
}

// validateOpsPurgeAllProcessDefinitionsFlags keeps static command-shape failures local.
func validateOpsPurgeAllProcessDefinitionsFlags(cmd *cobra.Command) error {
	if flagOpsPurgeAllPDKey != "" {
		if ok, firstBadKey, _ := validateKeys([]string{flagOpsPurgeAllPDKey}); !ok {
			return invalidFlagValuef("process definition key %q is not a valid key", firstBadKey)
		}
	}
	if cmd != nil && cmd.Flags().Changed("pd-version") && flagOpsPurgeAllPDProcessVersion <= 0 {
		return invalidFlagValuef("--pd-version must be positive integer")
	}
	if cmd != nil && cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	return validateOpsWorkflowReportFlags(flagOpsPurgeAllPDReportFile, OpsWorkflowReportFormat(flagOpsPurgeAllPDReportFormat))
}

func populateOpsPurgeAllProcessDefinitionsSelection() ops.ProcessDefinitionSelection {
	return ops.ProcessDefinitionSelection{
		Key:               flagOpsPurgeAllPDKey,
		BpmnProcessId:     flagOpsPurgeAllPDBpmnProcessID,
		ProcessVersion:    flagOpsPurgeAllPDProcessVersion,
		ProcessVersionTag: flagOpsPurgeAllPDProcessVersionTag,
		LatestOnly:        flagOpsPurgeAllPDLatest,
	}
}
