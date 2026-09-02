// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

const (
	opsAnalyseAPILatencyCommandName    = "ops analyse api-latency"
	opsAnalyseAPILatencyDefaultCount   = 20
	opsAnalyseAPILatencyDefaultWorkers = 4
)

var (
	flagOpsAnalyseAPILatencyCount        = opsAnalyseAPILatencyDefaultCount
	flagOpsAnalyseAPILatencyWorkers      = opsAnalyseAPILatencyDefaultWorkers
	flagOpsAnalyseAPILatencyReportFile   string
	flagOpsAnalyseAPILatencyReportFormat string
)

var opsAnalyseAPILatencyCmd = &cobra.Command{
	Use:   "api-latency",
	Short: "Analyse API latency without changing cluster state",
	Long: "Analyse API latency without changing cluster state.\n\n" +
		"The command is read-only. It measures cluster topology, process-definition search/read, and process-instance search/read paths in bounded closed-loop stages. Search-derived keyed reads reuse keys returned by the measured searches when available and supported.\n\n" +
		"--count is the total primary sample-cycle budget across all stages. --workers is the maximum closed-loop worker count and final stage width. The count must be large enough to exercise every stage width.\n\n" +
		"Default output is compact stage evidence with findings, notices, limitations, and outcome. JSON output uses the shared command envelope. Keys-only output is not meaningful for this diagnostic and is rejected.",
	Example: `  ./c8volt ops analyse api-latency
  ./c8volt ops analyse api-latency --count 20 --workers 4
  ./c8volt ops analyse api-latency -n 6 -w 2
  ./c8volt ops analyse api-latency --json`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateOpsAnalyseAPILatencyFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateOpsWorkflowReportPathForPlanning(flagOpsAnalyseAPILatencyReportFile, OpsWorkflowReportPreserveExisting); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		request, err := buildOpsAnalyseAPILatencyRequest(cmd, cfg)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		progress := configureOpsAPILatencyProgress(cmd, &request)
		result, err := analyseAPILatencyWithCommandActivity(cmd, request, func() (ops.APILatencyResult, error) {
			return cli.AnalyseAPILatency(cmd.Context(), request, collectOptions()...)
		})
		progress.Close()
		result = attachOpsAPILatencyResultContext(cfg, result)
		result = attachOpsAPILatencyReportRequest(result, flagOpsAnalyseAPILatencyReportFile, flagOpsAnalyseAPILatencyReportFormat)
		if err != nil {
			if reportErr := writeOpsAPILatencyReport(result, cfg, OpsWorkflowReportPreserveExisting); reportErr != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops analyse api-latency: %w; write report: %v", err, reportErr))
			}
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops analyse api-latency: %w", err))
		}
		if err := writeOpsAPILatencyReport(result, cfg, OpsWorkflowReportPreserveExisting); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("write ops analyse api-latency report: %w", err))
		}
		if err := renderOpsAPILatencyResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops analyse api-latency: %w", err))
		}
	},
}

func init() {
	opsAnalyseCmd.AddCommand(opsAnalyseAPILatencyCmd)
	useInvalidInputFlagErrors(opsAnalyseAPILatencyCmd)

	fs := opsAnalyseAPILatencyCmd.Flags()
	fs.IntVarP(&flagOpsAnalyseAPILatencyCount, "count", "n", opsAnalyseAPILatencyDefaultCount, "primary sample cycles to measure across all read-only stages")
	fs.IntVarP(&flagOpsAnalyseAPILatencyWorkers, "workers", "w", opsAnalyseAPILatencyDefaultWorkers, "maximum closed-loop workers and final stage width")
	fs.StringVar(&flagOpsAnalyseAPILatencyReportFile, "report-file", "", "write an API latency report to the given path")
	fs.StringVar(&flagOpsAnalyseAPILatencyReportFormat, "report-format", "", "API latency report format: markdown, json (default inferred from report-file extension)")

	setCommandMutation(opsAnalyseAPILatencyCmd, CommandMutationReadOnly)
	setContractSupport(opsAnalyseAPILatencyCmd, ContractSupportFull)
	setAutomationSupport(opsAnalyseAPILatencyCmd, AutomationSupportFull, "supports read-only bounded diagnostics with shared machine output")
	setOutputModes(opsAnalyseAPILatencyCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true, Notes: "stdout remains one JSON document; progress is suppressed"},
		OutputModeContract{Name: RenderModeKeysOnly.String(), Supported: false, Notes: "latency diagnostics do not produce key lists"},
	)
}

// validateOpsAnalyseAPILatencyFlags rejects invalid local inputs before client creation or remote work.
func validateOpsAnalyseAPILatencyFlags(cmd *cobra.Command) error {
	if flagOpsAnalyseAPILatencyCount < 1 {
		return invalidFlagValuef("invalid value for --count: %d, expected positive integer", flagOpsAnalyseAPILatencyCount)
	}
	if flagOpsAnalyseAPILatencyWorkers < 1 {
		return invalidFlagValuef("invalid value for --workers: %d, expected positive integer", flagOpsAnalyseAPILatencyWorkers)
	}
	if flagOpsAnalyseAPILatencyWorkers > flagOpsAnalyseAPILatencyCount {
		return invalidFlagValuef("--workers must be no greater than --count")
	}
	minimum := opsAnalyseAPILatencyMinimumSamples(flagOpsAnalyseAPILatencyWorkers)
	if flagOpsAnalyseAPILatencyCount < minimum {
		return invalidFlagValuef("count %d is too small for worker stages %s; at least %d primary samples are required", flagOpsAnalyseAPILatencyCount, opsAnalyseAPILatencyStageWidthList(flagOpsAnalyseAPILatencyWorkers), minimum)
	}
	if pickMode() == RenderModeKeysOnly {
		return invalidFlagValuef("--keys-only is not supported by ops analyse api-latency")
	}
	if err := validateOpsWorkflowReportFlags(flagOpsAnalyseAPILatencyReportFile, OpsWorkflowReportFormat(flagOpsAnalyseAPILatencyReportFormat)); err != nil {
		return err
	}
	_ = cmd
	return nil
}

// buildOpsAnalyseAPILatencyRequest turns command flags and safe config context into facade input.
func buildOpsAnalyseAPILatencyRequest(cmd *cobra.Command, cfg *config.Config) (ops.APILatencyRequest, error) {
	if err := validateOpsAnalyseAPILatencyFlags(cmd); err != nil {
		return ops.APILatencyRequest{}, err
	}
	httpTimeout, err := time.ParseDuration(cfg.HTTP.Timeout)
	if err != nil {
		return ops.APILatencyRequest{}, invalidFlagValuef("invalid HTTP timeout %q: %v", cfg.HTTP.Timeout, err)
	}
	return ops.APILatencyRequest{
		CommandName:  opsAnalyseAPILatencyCommandName,
		Mode:         ops.APILatencyModeReadOnly,
		Count:        flagOpsAnalyseAPILatencyCount,
		Workers:      flagOpsAnalyseAPILatencyWorkers,
		TenantID:     cfg.App.Tenant,
		HTTPTimeout:  httpTimeout,
		Backoff:      opsAPILatencyBackoffFromConfig(cfg.App.Backoff),
		OutputMode:   pickMode().String(),
		ReportFile:   flagOpsAnalyseAPILatencyReportFile,
		ReportFormat: flagOpsAnalyseAPILatencyReportFormat,
		StartedAt:    time.Now().UTC(),
	}, nil
}

// analyseAPILatencyWithCommandActivity wraps the facade call with semantic activity text.
func analyseAPILatencyWithCommandActivity(cmd *cobra.Command, request ops.APILatencyRequest, run func() (ops.APILatencyResult, error)) (ops.APILatencyResult, error) {
	stopActivity := startCommandActivity(cmd, formatOpsAPILatencyActivity(request))
	defer stopActivity()
	return run()
}

// formatOpsAPILatencyActivity keeps transient activity wording compact and endpoint-free.
func formatOpsAPILatencyActivity(ops.APILatencyRequest) string {
	return "measuring read-only API latency"
}

// opsAPILatencyBackoffFromConfig records normalized backoff context for the reportable request.
func opsAPILatencyBackoffFromConfig(backoff config.BackoffConfig) ops.APILatencyBackoff {
	return ops.APILatencyBackoff{
		Strategy:     ops.APILatencyBackoffStrategy(backoff.Strategy),
		InitialDelay: backoff.InitialDelay,
		MaxDelay:     backoff.MaxDelay,
		Multiplier:   backoff.Multiplier,
		Timeout:      backoff.Timeout,
		MaxRetries:   backoff.MaxRetries,
	}
}

// attachOpsAPILatencyResultContext fills command-owned safe context fields.
func attachOpsAPILatencyResultContext(cfg *config.Config, result ops.APILatencyResult) ops.APILatencyResult {
	if result.Context.SchemaVersion == "" {
		result.Context.SchemaVersion = ops.APILatencySchemaVersion
	}
	if result.Context.C8voltVersion == "" {
		result.Context.C8voltVersion = CurrentBuildInfo().Version
	}
	if result.Context.CamundaVersion == "" && cfg != nil {
		result.Context.CamundaVersion = cfg.App.CamundaVersion.String()
	}
	if result.Context.Profile == "" && cfg != nil {
		result.Context.Profile = cfg.ActiveProfile
	}
	if result.Context.Tenant == "" && cfg != nil {
		result.Context.Tenant = cfg.App.Tenant
	}
	return result
}

// opsAnalyseAPILatencyStageWidths mirrors the command-local flag validation contract.
func opsAnalyseAPILatencyStageWidths(workers int) []int {
	widths := []int{1}
	for width := 2; width < workers; width *= 2 {
		widths = append(widths, width)
	}
	if workers != 1 {
		widths = append(widths, workers)
	}
	return widths
}

// opsAnalyseAPILatencyMinimumSamples calculates the local minimum sample budget for stage validation.
func opsAnalyseAPILatencyMinimumSamples(workers int) int {
	total := 0
	for _, width := range opsAnalyseAPILatencyStageWidths(workers) {
		total += width
	}
	return total
}

// opsAnalyseAPILatencyStageWidthList formats stage widths for validation errors.
func opsAnalyseAPILatencyStageWidthList(workers int) string {
	widths := opsAnalyseAPILatencyStageWidths(workers)
	out := ""
	for i, width := range widths {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%d", width)
	}
	return out
}
