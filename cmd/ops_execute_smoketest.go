// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

const opsExecuteSmokeTestCommandName = "ops execute smoke-test"

var (
	flagOpsExecuteSmokeTestCount        = 1
	flagOpsExecuteSmokeTestNoCleanup    bool
	flagOpsExecuteSmokeTestReportFile   string
	flagOpsExecuteSmokeTestReportFormat string
)

var opsExecuteSmokeTestCmd = &cobra.Command{
	Use:   "smoke-test",
	Short: "Execute a cluster smoke test workflow",
	Long: "Execute a cluster smoke test workflow.\n\n" +
		"The workflow validates the configured profile, selects the embedded multiple-subprocess fixture for the configured Camunda version, deploys it, creates process instances, walks their families, and cleans up resources it can safely attribute to the run unless --no-cleanup is set. Cleanup always removes created process instances. Process-definition cleanup runs only when no unrelated instances still use the deployed fixture definition; dirty clusters skip that final definition cleanup and report retained resources instead of failing the smoke proof. Use --dry-run to validate the requested plan without submitting mutation requests.",
	Example: `  ./c8volt ops execute smoke-test --dry-run
  ./c8volt ops execute smoke-test --report-file smoke-test.md
  ./c8volt ops execute smoke-test --count 5 --report-file smoke-test.md`,
	Aliases: []string{"st"},
	Args:    cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateOpsExecuteSmokeTestFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		effectiveAutoConfirm := shouldImplicitlyConfirm(cmd)
		request := ops.SmokeTestRequest{
			CommandName:   opsExecuteSmokeTestCommandName,
			DryRun:        flagDryRun,
			Count:         flagOpsExecuteSmokeTestCount,
			Workers:       flagWorkers,
			FailFast:      flagFailFast,
			NoWorkerLimit: flagNoWorkerLimit,
			NoCleanup:     flagOpsExecuteSmokeTestNoCleanup,
			AutoConfirm:   flagCmdAutoConfirm,
			Automation:    automationModeEnabled(cmd),
			NoWait:        flagNoWait,
			OutputMode:    pickMode().String(),
			ReportFile:    flagOpsExecuteSmokeTestReportFile,
			ReportFormat:  flagOpsExecuteSmokeTestReportFormat,
			StartedAt:     time.Now().UTC(),
		}
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsExecuteSmokeTestReportFile, opsWorkflowReportWriteModeForConfirmedMutation(effectiveAutoConfirm && !flagDryRun)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if !flagDryRun && !flagOpsExecuteSmokeTestNoCleanup {
			prompt := opsExecuteSmokeTestConfirmationPrompt(request)
			if err := confirmCmdOrAbortFn(effectiveAutoConfirm, prompt); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
		}
		result, err := executeSmokeTestWithCommandActivity(cmd, request, func() (ops.SmokeTestResult, error) {
			return cli.ExecuteSmokeTest(cmd.Context(), request, collectOptions()...)
		})
		if err != nil {
			if reportErr := writeOpsExecuteSmokeTestReport(result, cfg, opsExecuteSmokeTestReportWriteMode(result)); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops execute smoke-test: %w; write audit report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops execute smoke-test: %w", err))
		}
		if err := writeOpsExecuteSmokeTestReport(result, cfg, opsExecuteSmokeTestReportWriteMode(result)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops execute smoke-test audit report: %w", err))
		}
		if err := renderOpsExecuteSmokeTestResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops execute smoke-test: %w", err))
		}
	},
}

func init() {
	opsExecuteCmd.AddCommand(opsExecuteSmokeTestCmd)
	useInvalidInputFlagErrors(opsExecuteSmokeTestCmd)

	fs := opsExecuteSmokeTestCmd.Flags()
	fs.IntVarP(&flagOpsExecuteSmokeTestCount, "count", "n", 1, "number of process instances to create from the deployed smoke-test definition")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when creating, walking, or cleaning smoke-test resources (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued smoke-test jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling smoke-test work after the first error")
	fs.BoolVar(&flagOpsExecuteSmokeTestNoCleanup, "no-cleanup", false, "retain created process instances and the deployed process definition")
	fs.BoolVar(&flagDryRun, "dry-run", false, "validate the smoke-test plan without submitting mutation requests")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after cleanup requests are accepted without deletion confirmation")
	fs.StringVar(&flagOpsExecuteSmokeTestReportFile, "report-file", "", "write an audit report to the given path")
	fs.StringVar(&flagOpsExecuteSmokeTestReportFormat, "report-format", "", "audit report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsExecuteSmokeTestCmd, CommandMutationStateChanging)
	setContractSupport(opsExecuteSmokeTestCmd, ContractSupportFull)
	setAutomationSupport(opsExecuteSmokeTestCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed smoke-test cleanup with shared machine output")
}

func validateOpsExecuteSmokeTestFlags(cmd *cobra.Command) error {
	if flagOpsExecuteSmokeTestCount < 1 {
		return invalidFlagValuef("invalid value for --count: %d, expected positive integer", flagOpsExecuteSmokeTestCount)
	}
	if cmd != nil && cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	return validateOpsExecuteSmokeTestReportFlags()
}

func validateOpsExecuteSmokeTestReportFlags() error {
	return validateOpsWorkflowReportFlags(flagOpsExecuteSmokeTestReportFile, OpsWorkflowReportFormat(flagOpsExecuteSmokeTestReportFormat))
}

func executeSmokeTestWithCommandActivity(cmd *cobra.Command, request ops.SmokeTestRequest, run func() (ops.SmokeTestResult, error)) (ops.SmokeTestResult, error) {
	stopActivity := startCommandActivity(cmd, formatOpsExecuteSmokeTestActivity(request))
	defer stopActivity()
	return run()
}

func formatOpsExecuteSmokeTestActivity(request ops.SmokeTestRequest) string {
	if request.DryRun {
		return "validating smoke-test plan"
	}
	return "running smoke-test workflow"
}

func opsExecuteSmokeTestConfirmationPrompt(request ops.SmokeTestRequest) string {
	cleanup := "then clean up created resources"
	if request.NoCleanup {
		cleanup = "then retain created resources"
	}
	return fmt.Sprintf(
		"smoke test: deploy fixture, start %d process instance(s), walk process-instance families, %s. Do you want to proceed?",
		request.Count,
		cleanup,
	)
}

func abortOpsExecuteSmokeTestAfterReport(cmd *cobra.Command, log *slog.Logger, cfg *config.Config, result ops.SmokeTestResult, err error) {
	if reportErr := writeOpsExecuteSmokeTestReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w; write audit report: %v", err, reportErr))
	}
	handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
}

func opsExecuteSmokeTestReportWriteMode(result ops.SmokeTestResult) OpsWorkflowReportWriteMode {
	return opsWorkflowReportWriteModeForConfirmedMutation(result.Cleanup.ProcessInstanceCleanup.Submitted || result.Cleanup.ProcessDefinitionCleanup.Submitted)
}
