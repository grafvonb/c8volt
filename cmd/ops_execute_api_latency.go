// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

const (
	opsExecuteAPILatencyCommandName    = "ops execute api-latency-test"
	opsExecuteAPILatencyDefaultCount   = 20
	opsExecuteAPILatencyDefaultWorkers = 4
)

var (
	flagOpsExecuteAPILatencyCount        = opsExecuteAPILatencyDefaultCount
	flagOpsExecuteAPILatencyWorkers      = opsExecuteAPILatencyDefaultWorkers
	flagOpsExecuteAPILatencyDryRun       bool
	flagOpsExecuteAPILatencyNoCleanup    bool
	flagOpsExecuteAPILatencyReportFile   string
	flagOpsExecuteAPILatencyReportFormat string
)

var opsExecuteAPILatencyCmd = &cobra.Command{
	Use:   "api-latency-test",
	Short: "Execute a bounded active API latency test",
	Long: "Execute a bounded active API latency test.\n\n" +
		"The command deploys the existing version-matched SimpleUserTask fixture, creates a bounded number of process instances, measures create response, concurrent read, and search visibility evidence, then cleans up exact run-owned resources unless --no-cleanup is set.\n\n" +
		"It requires one concrete tenant because the workflow creates and cleans resources in a single destination. --dry-run validates and previews the active plan without mutation. --no-cleanup explicitly retains run-owned resources and still requires confirmation for a real run.\n\n" +
		"Default result output summarizes the active scope, request errors and timeouts, create latency, load effect, search visibility, findings with next investigation, and cleanup in operator language. Use --verbose for the stage plan, run and fixture metadata, detailed statistics, notices, and limitations.\n\n" +
		"--count is the total primary process-instance create sample budget across all stages. --workers is the maximum closed-loop worker count and final stage width. JSON active execution requires --dry-run, --auto-confirm, or --automation so stdout remains one document. Keys-only output is not meaningful for this diagnostic and is rejected.",
	Example: `  ./c8volt ops execute api-latency-test --dry-run
  ./c8volt ops execute api-latency-test --count 20 --workers 4 --auto-confirm
  ./c8volt ops execute api-latency-test -n 7 -w 4 --dry-run
  ./c8volt --verbose ops execute api-latency-test --auto-confirm
  ./c8volt ops execute api-latency-test --no-cleanup --auto-confirm`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateOpsExecuteAPILatencyFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		if err := validateOpsExecuteAPILatencyJSONGuardrails(cmd); err != nil {
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
		writeMode := opsWorkflowReportWriteModeForConfirmedMutation(effectiveAutoConfirm && !flagOpsExecuteAPILatencyDryRun)
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsExecuteAPILatencyReportFile, writeMode); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		request, err := buildOpsExecuteAPILatencyRequest(cmd, cfg)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		tenantCtx := attachCreationTenantContext(cmd, cfg)
		if !request.DryRun {
			if shouldRenderTenantContextHuman(cmd, tenantCtx) && !automationModeEnabled(cmd) {
				printOpsTenantContext(cmd, tenantCtx, ops.ProgressChannel{Mode: ops.ProgressModeHuman, DurableAllowed: true, StderrAllowed: true})
			}
			if err := confirmCmdOrAbortFn(effectiveAutoConfirm, opsExecuteAPILatencyConfirmationPrompt(request)); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
		}
		progress := configureOpsAPILatencyProgress(cmd, &request)
		result, err := executeAPILatencyWithCommandActivity(cmd, request, func() (ops.APILatencyResult, error) {
			return withOpsExecuteAPILatencyInterruptContext(cmd, request, func() (ops.APILatencyResult, error) {
				return cli.ExecuteAPILatencyTest(cmd.Context(), request, collectOpsExecuteAPILatencyOptions()...)
			})
		})
		progress.Close()
		result = attachOpsAPILatencyResultContext(cfg, result, opsExecuteAPILatencyCommandName)
		result = attachOpsAPILatencyReportRequest(result, flagOpsExecuteAPILatencyReportFile, flagOpsExecuteAPILatencyReportFormat)
		if commandErr := opsAPILatencyCommandError(opsExecuteAPILatencyCommandName, result, err); commandErr != nil {
			if reportErr := writeOpsAPILatencyReport(result, cfg, opsExecuteAPILatencyReportWriteMode(result)); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w; write report: %v", commandErr, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, commandErr)
		}
		if err := writeOpsAPILatencyReport(result, cfg, opsExecuteAPILatencyReportWriteMode(result)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops execute api-latency-test report: %w", err))
		}
		if err := renderOpsAPILatencyResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops execute api-latency-test: %w", err))
		}
	},
}

func init() {
	opsExecuteCmd.AddCommand(opsExecuteAPILatencyCmd)
	useInvalidInputFlagErrors(opsExecuteAPILatencyCmd)

	fs := opsExecuteAPILatencyCmd.Flags()
	fs.IntVarP(&flagOpsExecuteAPILatencyCount, "count", "n", opsExecuteAPILatencyDefaultCount, "primary process-instance create samples across active stages")
	fs.IntVarP(&flagOpsExecuteAPILatencyWorkers, "workers", "w", opsExecuteAPILatencyDefaultWorkers, "maximum closed-loop workers and final active stage width")
	fs.BoolVar(&flagOpsExecuteAPILatencyDryRun, "dry-run", false, "validate and preview the active API latency plan without mutation")
	fs.BoolVar(&flagOpsExecuteAPILatencyNoCleanup, "no-cleanup", false, "retain active API latency resources after execution")
	fs.StringVar(&flagOpsExecuteAPILatencyReportFile, "report-file", "", "write an API latency test report to the given path")
	fs.StringVar(&flagOpsExecuteAPILatencyReportFormat, "report-format", "", "API latency test report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsExecuteAPILatencyCmd, CommandMutationStateChanging)
	setContractSupport(opsExecuteAPILatencyCmd, ContractSupportFull)
	setAutomationSupport(opsExecuteAPILatencyCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed active API latency diagnostics")
	setAllTenantsSupport(opsExecuteAPILatencyCmd, AllTenantsSupportRejectedConcreteDestination)
	setOutputModes(opsExecuteAPILatencyCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true, Notes: "stdout remains one JSON document; active execution requires dry-run, auto-confirm, or automation"},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: false, Notes: "latency diagnostics do not produce key lists"},
	)
}

// validateOpsExecuteAPILatencyFlags rejects invalid local inputs before client creation or remote work.
func validateOpsExecuteAPILatencyFlags(cmd *cobra.Command) error {
	if flagOpsExecuteAPILatencyCount < 1 {
		return invalidFlagValuef("invalid value for --count: %d, expected positive integer", flagOpsExecuteAPILatencyCount)
	}
	if flagOpsExecuteAPILatencyWorkers < 1 {
		return invalidFlagValuef("invalid value for --workers: %d, expected positive integer", flagOpsExecuteAPILatencyWorkers)
	}
	if flagOpsExecuteAPILatencyWorkers > flagOpsExecuteAPILatencyCount {
		return invalidFlagValuef("--workers must be no greater than --count")
	}
	minimum := opsAnalyseAPILatencyMinimumSamples(flagOpsExecuteAPILatencyWorkers)
	if flagOpsExecuteAPILatencyCount < minimum {
		return invalidFlagValuef("count %d is too small for worker stages %s; at least %d primary samples are required", flagOpsExecuteAPILatencyCount, opsAnalyseAPILatencyStageWidthList(flagOpsExecuteAPILatencyWorkers), minimum)
	}
	if pickMode() == RenderModeKeysOnly {
		return invalidFlagValuef("--keys-only is not supported by ops execute api-latency-test")
	}
	if err := validateOpsWorkflowReportFlags(flagOpsExecuteAPILatencyReportFile, OpsWorkflowReportFormat(flagOpsExecuteAPILatencyReportFormat)); err != nil {
		return err
	}
	_ = cmd
	return nil
}

// validateOpsExecuteAPILatencyJSONGuardrails keeps machine output from prompting.
func validateOpsExecuteAPILatencyJSONGuardrails(cmd *cobra.Command) error {
	if flagOpsExecuteAPILatencyDryRun || pickMode() != RenderModeJSON || flagCmdAutoConfirm || flagCmdAutomation || automationModeEnabled(cmd) {
		return nil
	}
	return missingDependentFlagsf("--json ops execute api-latency-test requires --dry-run, --auto-confirm, or --automation")
}

// buildOpsExecuteAPILatencyRequest turns command flags and safe config context into facade input.
func buildOpsExecuteAPILatencyRequest(cmd *cobra.Command, cfg *config.Config) (ops.APILatencyRequest, error) {
	if err := validateOpsExecuteAPILatencyFlags(cmd); err != nil {
		return ops.APILatencyRequest{}, err
	}
	httpTimeout, err := time.ParseDuration(cfg.HTTP.Timeout)
	if err != nil {
		return ops.APILatencyRequest{}, invalidFlagValuef("invalid HTTP timeout %q: %v", cfg.HTTP.Timeout, err)
	}
	return ops.APILatencyRequest{
		CommandName:  opsExecuteAPILatencyCommandName,
		Mode:         ops.APILatencyModeActive,
		Count:        flagOpsExecuteAPILatencyCount,
		Workers:      flagOpsExecuteAPILatencyWorkers,
		DryRun:       flagOpsExecuteAPILatencyDryRun,
		NoCleanup:    flagOpsExecuteAPILatencyNoCleanup,
		TenantID:     cfg.App.Tenant,
		HTTPTimeout:  httpTimeout,
		Backoff:      opsAPILatencyBackoffFromConfig(cfg.App.Backoff),
		OutputMode:   pickMode().String(),
		ReportFile:   flagOpsExecuteAPILatencyReportFile,
		ReportFormat: flagOpsExecuteAPILatencyReportFormat,
		StartedAt:    time.Now().UTC(),
	}, nil
}

// collectOpsExecuteAPILatencyOptions suppresses low-level mutation chatter unless verbose output is requested.
func collectOpsExecuteAPILatencyOptions() []foptions.FacadeOption {
	opts := collectOptions()
	if !flagVerbose {
		opts = append(opts, foptions.WithSuppressWorkflowDetailLogs(), foptions.WithSuppressProcessInstanceDetailLogs())
	}
	return opts
}

// executeAPILatencyWithCommandActivity wraps the active facade call with semantic activity text.
func executeAPILatencyWithCommandActivity(cmd *cobra.Command, request ops.APILatencyRequest, run func() (ops.APILatencyResult, error)) (ops.APILatencyResult, error) {
	stopActivity := startCommandActivity(cmd, formatOpsExecuteAPILatencyActivity(request))
	defer stopActivity()
	return run()
}

// formatOpsExecuteAPILatencyActivity keeps transient activity wording compact and endpoint-free.
func formatOpsExecuteAPILatencyActivity(request ops.APILatencyRequest) string {
	if request.DryRun {
		return "validating active API latency plan"
	}
	return "running active API latency test"
}

// opsExecuteAPILatencyConfirmationPrompt summarizes mutation and cleanup intent.
func opsExecuteAPILatencyConfirmationPrompt(request ops.APILatencyRequest) string {
	cleanup := "then clean up run-owned resources"
	if request.NoCleanup {
		cleanup = "then retain run-owned resources"
	}
	return fmt.Sprintf(
		"active API latency test: deploy fixture, create %d process instance(s), measure read and visibility latency, %s. Do you want to proceed?",
		request.Count,
		cleanup,
	)
}

// opsExecuteAPILatencyReportWriteMode allows overwrite only after exact active ownership indicates mutation.
func opsExecuteAPILatencyReportWriteMode(result ops.APILatencyResult) OpsWorkflowReportWriteMode {
	confirmed := false
	if result.Ownership != nil {
		confirmed = result.Ownership.DeploymentSubmitted ||
			result.Ownership.ProcessDefinitionKey != "" ||
			len(result.Ownership.ProcessInstanceKeys) > 0
	}
	return opsWorkflowReportWriteModeForConfirmedMutation(confirmed)
}
