// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

// renderOpsAPILatencyResult dispatches API-latency output through the shared machine contract or compact human view.
func renderOpsAPILatencyResult(cmd *cobra.Command, result ops.APILatencyResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	active := opsAPILatencyResultIsActive(result)
	if active {
		renderHumanLine(cmd, "execute api latency test")
	} else {
		renderHumanLine(cmd, "analyse api latency")
	}
	renderAttachedTenantContext(cmd)
	renderOpsAPILatencyContext(cmd, result.Context)
	renderOpsAPILatencyPlan(cmd, result.Plan)
	renderOpsAPILatencyTopology(cmd, result.Topology)
	renderOpsAPILatencyStages(cmd, result.Stages, result.Plan.Mode)
	renderOpsAPILatencyFindings(cmd, result.Findings)
	if active {
		renderOpsAPILatencyOwnership(cmd, result.Ownership)
		renderOpsAPILatencyVisibility(cmd, result.Visibility)
		renderOpsAPILatencyCleanup(cmd, result.Cleanup)
	}
	renderOpsAPILatencyStringList(cmd, "notice", result.Notices)
	renderOpsAPILatencyStringList(cmd, "limitation", result.Limitations)
	renderOpsAPILatencyReportFile(cmd, result.Request.ReportFile)
	renderOpsAPILatencyOutcome(cmd, result)
	return nil
}

func opsAPILatencyResultIsActive(result ops.APILatencyResult) bool {
	return result.Plan.Mode == ops.APILatencyModeActive || result.Request.Mode == ops.APILatencyModeActive || result.Ownership != nil
}

// renderOpsAPILatencyContext prints only safe invocation identity fields.
func renderOpsAPILatencyContext(cmd *cobra.Command, context ops.APILatencyRunContext) {
	parts := []string{}
	if context.C8voltVersion != "" {
		parts = append(parts, "c8volt "+context.C8voltVersion)
	}
	if context.Profile != "" {
		parts = append(parts, "profile "+context.Profile)
	}
	if context.Tenant != "" {
		parts = append(parts, "tenant "+context.Tenant)
	}
	if context.CamundaVersion != "" {
		parts = append(parts, "camunda "+context.CamundaVersion)
	}
	if len(parts) == 0 {
		return
	}
	renderHumanLine(cmd, "context: %s", strings.Join(parts, "; "))
}

// attachOpsAPILatencyReportRequest records command-owned report flags on the final render payload.
func attachOpsAPILatencyReportRequest(result ops.APILatencyResult, reportFile string, reportFormat string) ops.APILatencyResult {
	result.Request.ReportFile = reportFile
	result.Request.ReportFormat = reportFormat
	return result
}

// renderOpsAPILatencyReportFile prints the compact report location after a successful write.
func renderOpsAPILatencyReportFile(cmd *cobra.Command, path string) {
	if path == "" {
		return
	}
	renderHumanLine(cmd, "report: written %s", path)
}

// writeOpsAPILatencyReport renders and writes the requested API-latency report payload.
func writeOpsAPILatencyReport(result ops.APILatencyResult, cfg *config.Config, mode OpsWorkflowReportWriteMode) error {
	if result.Request.ReportFile == "" {
		return nil
	}
	format, err := opsWorkflowReportFormatForPath(result.Request.ReportFile, OpsWorkflowReportFormat(result.Request.ReportFormat))
	if err != nil {
		return err
	}
	var data []byte
	switch format {
	case OpsWorkflowReportFormatJSON:
		data, err = renderOpsAPILatencyJSONReport(result)
	case OpsWorkflowReportFormatMarkdown:
		data, err = renderOpsAPILatencyMarkdownReport(result, cfg)
	default:
		err = fmt.Errorf("unsupported ops workflow report format %q", format)
	}
	if err != nil {
		return err
	}
	return writeOpsWorkflowReportFile(result.Request.ReportFile, data, mode)
}

// renderOpsAPILatencyJSONReport encodes the raw API-latency result without the command envelope.
func renderOpsAPILatencyJSONReport(result ops.APILatencyResult) ([]byte, error) {
	var buf bytes.Buffer
	if err := toolx.JSON(&buf, result); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderOpsAPILatencyMarkdownReport renders the API-latency result as compact shareable evidence.
func renderOpsAPILatencyMarkdownReport(result ops.APILatencyResult, cfg *config.Config) ([]byte, error) {
	var out strings.Builder
	active := opsAPILatencyResultIsActive(result)
	if active {
		out.WriteString("# Execute API Latency Test Report\n\n")
	} else {
		out.WriteString("# Analyse API Latency Report\n\n")
	}
	writeMarkdownReportField(&out, "Schema Version", result.SchemaVersion)
	writeMarkdownReportField(&out, "Command", result.Context.CommandName)
	writeMarkdownReportField(&out, "Started", formatOpsPurgeReportTime(result.Context.StartedAt, cfg))
	writeMarkdownReportField(&out, "Finished", formatOpsPurgeReportTime(result.Context.FinishedAt, cfg))
	writeMarkdownReportField(&out, "Duration", result.Context.Duration)
	writeMarkdownReportField(&out, "C8volt Version", result.Context.C8voltVersion)
	writeMarkdownReportField(&out, "Camunda Version", result.Context.CamundaVersion)
	writeMarkdownReportField(&out, "Profile", result.Context.Profile)
	writeMarkdownReportField(&out, "Tenant", result.Context.Tenant)
	writeMarkdownReportField(&out, "Outcome", string(result.Outcome))

	out.WriteString("\n## Request\n\n")
	writeMarkdownReportField(&out, "Mode", string(result.Request.Mode))
	writeMarkdownReportField(&out, "Count", fmt.Sprintf("%d", result.Request.Count))
	writeMarkdownReportField(&out, "Workers", fmt.Sprintf("%d", result.Request.Workers))
	writeMarkdownReportField(&out, "Dry Run", fmt.Sprintf("%t", result.Request.DryRun))
	writeMarkdownReportField(&out, "No Cleanup", fmt.Sprintf("%t", result.Request.NoCleanup))
	writeMarkdownReportField(&out, "Tenant", result.Request.TenantID)
	writeMarkdownReportField(&out, "HTTP Timeout", result.Request.HTTPTimeout.String())
	writeMarkdownReportField(&out, "Output Mode", result.Request.OutputMode)
	writeMarkdownReportField(&out, "Report File", result.Request.ReportFile)
	writeMarkdownReportField(&out, "Report Format", result.Request.ReportFormat)

	out.WriteString("\n## Plan\n\n")
	writeMarkdownReportField(&out, "Mode", string(result.Plan.Mode))
	writeMarkdownReportField(&out, "Run ID", result.Plan.RunID)
	writeMarkdownReportField(&out, "Primary Sample Limit", fmt.Sprintf("%d", result.Plan.PrimarySampleLimit))
	writeMarkdownReportField(&out, "Primary Sample Allocation", fmt.Sprintf("%d", result.Plan.PrimarySampleAllocation))
	writeMarkdownReportField(&out, "Derived Request Limit", fmt.Sprintf("%d", result.Plan.DerivedRequestLimit))
	writeMarkdownReportField(&out, "Visibility Attempt Limit", fmt.Sprintf("%d", result.Plan.VisibilityAttemptLimit))
	if result.Plan.Fixture != nil {
		writeMarkdownReportField(&out, "Fixture File", result.Plan.Fixture.File)
		writeMarkdownReportField(&out, "BPMN Process ID", result.Plan.Fixture.BpmnProcessID)
	}
	if result.Plan.Cleanup != nil {
		writeMarkdownReportField(&out, "Cleanup Requested", fmt.Sprintf("%t", result.Plan.Cleanup.Requested))
		writeMarkdownReportField(&out, "Cleanup Supported", fmt.Sprintf("%t", result.Plan.Cleanup.Supported))
		writeMarkdownReportField(&out, "Intentional Retention", fmt.Sprintf("%t", result.Plan.Cleanup.IntentionalRetention))
		writeMarkdownReportField(&out, "Cleanup Block Reason", result.Plan.Cleanup.BlockReason)
	}
	if len(result.Plan.Stages) > 0 {
		out.WriteString("- Stages:\n")
		for _, stage := range result.Plan.Stages {
			out.WriteString(fmt.Sprintf("  - stage %d: workers %d; primary samples %d; derived requests <= %d\n",
				stage.Index,
				stage.WorkerCount,
				stage.PrimarySamples,
				stage.DerivedRequestLimit,
			))
		}
	}

	out.WriteString("\n## Topology\n\n")
	writeMarkdownReportField(&out, "Brokers", fmt.Sprintf("%d", result.Topology.BrokerCount))
	writeMarkdownReportField(&out, "Partitions", fmt.Sprintf("%d", result.Topology.PartitionCount))
	writeMarkdownReportField(&out, "Health Known", fmt.Sprintf("%t", result.Topology.HealthKnown))
	writeMarkdownReportList(&out, "Unhealthy Partitions", opsAPILatencyIntStrings(result.Topology.UnhealthyPartitions))
	writeMarkdownReportList(&out, "Leaderless Partitions", opsAPILatencyIntStrings(result.Topology.LeaderlessPartitions))

	out.WriteString("\n## Stages\n\n")
	for _, stage := range result.Stages {
		out.WriteString(fmt.Sprintf("- Stage %d: %s; %s; %s\n",
			stage.Plan.Index,
			formatOpsAPILatencyStageAttempts(stage, result.Plan.Mode),
			formatOpsAPILatencyStageOutcomes(stage),
			formatOpsAPILatencyStageLatency(stage),
		))
		for _, category := range stage.Categories {
			out.WriteString(fmt.Sprintf("  - %s: attempts %d; successes %d; errors %d; timeouts %d; unavailable %d; p50 %s; p95 %s; max %s\n",
				category.Category,
				category.Attempts,
				category.Successes,
				category.Errors,
				category.Timeouts,
				category.Unavailable,
				formatOpsAPILatencyDuration(category.P50),
				formatOpsAPILatencyDuration(category.P95),
				formatOpsAPILatencyDuration(category.Max),
			))
		}
	}

	out.WriteString("\n## Findings\n\n")
	for _, finding := range result.Findings {
		out.WriteString(fmt.Sprintf("- %s: %s; confidence %s\n", finding.Code, finding.LikelyArea, finding.Confidence))
		writeMarkdownReportList(&out, "Evidence", finding.Evidence)
		writeMarkdownReportField(&out, "Limitation", finding.Limitation)
		writeMarkdownReportField(&out, "Next Investigation", finding.NextInvestigation)
	}

	if active {
		renderOpsAPILatencyMarkdownActiveEvidence(&out, result)
	}
	writeMarkdownReportList(&out, "Notices", result.Notices)
	writeMarkdownReportList(&out, "Limitations", result.Limitations)

	return []byte(out.String()), nil
}

// renderOpsAPILatencyMarkdownActiveEvidence writes active-only ownership, visibility, and cleanup sections.
func renderOpsAPILatencyMarkdownActiveEvidence(out *strings.Builder, result ops.APILatencyResult) {
	out.WriteString("\n## Ownership\n\n")
	if result.Ownership != nil {
		writeMarkdownReportField(out, "Run ID", result.Ownership.RunID)
		writeMarkdownReportField(out, "Fixture", result.Ownership.FixtureName)
		writeMarkdownReportField(out, "BPMN Process ID", result.Ownership.BpmnProcessID)
		writeMarkdownReportField(out, "Deployment Submitted", fmt.Sprintf("%t", result.Ownership.DeploymentSubmitted))
		writeMarkdownReportField(out, "Process Definition Key", result.Ownership.ProcessDefinitionKey)
		writeMarkdownReportList(out, "Process Instance Keys", result.Ownership.ProcessInstanceKeys)
	}
	out.WriteString("\n## Visibility\n\n")
	for _, item := range result.Visibility {
		out.WriteString(fmt.Sprintf("- %s: visible %t; attempts %d/%d; duration %s; classification %s\n",
			item.ProcessInstanceKey,
			item.Visible,
			item.Attempts,
			item.AttemptLimit,
			item.Duration.String(),
			item.FinalClassification,
		))
	}
	out.WriteString("\n## Cleanup\n\n")
	for _, record := range result.Cleanup {
		out.WriteString(fmt.Sprintf("- %s %s: %s; classification %s",
			record.ResourceType,
			record.Key,
			record.Status,
			record.Classification,
		))
		if record.RecoveryCommand != "" {
			out.WriteString("; recovery: " + record.RecoveryCommand)
		}
		out.WriteString("\n")
	}
}

func renderOpsAPILatencyPlan(cmd *cobra.Command, plan ops.APILatencyPlan) {
	if plan.PrimarySampleLimit == 0 && len(plan.Stages) == 0 {
		return
	}
	if plan.Mode == ops.APILatencyModeActive {
		renderHumanLine(cmd, "request: count %d; primary allocation %d/%d; workers %s; stages %d; derived requests <= %d",
			plan.PrimarySampleLimit,
			plan.PrimarySampleAllocation,
			plan.PrimarySampleLimit,
			formatOpsAPILatencyStageWidths(plan.Stages),
			len(plan.Stages),
			plan.DerivedRequestLimit,
		)
		if plan.RunID != "" {
			renderHumanLine(cmd, "run: %s", plan.RunID)
		}
		if plan.Fixture != nil {
			renderHumanLine(cmd, "fixture: %s (%s)", plan.Fixture.File, plan.Fixture.BpmnProcessID)
		}
		if plan.VisibilityAttemptLimit > 0 {
			renderHumanLine(cmd, "visibility: attempts <= %d", plan.VisibilityAttemptLimit)
		}
		renderOpsAPILatencyCleanupPlan(cmd, plan.Cleanup)
		return
	}
	renderHumanLine(cmd, "request: count %d; workers %s; stages %d; derived requests <= %d",
		plan.PrimarySampleLimit,
		formatOpsAPILatencyStageWidths(plan.Stages),
		len(plan.Stages),
		plan.DerivedRequestLimit,
	)
}

func renderOpsAPILatencyCleanupPlan(cmd *cobra.Command, cleanup *ops.APILatencyCleanupPlan) {
	if cleanup == nil {
		return
	}
	intent := "retained (--no-cleanup)"
	if cleanup.Requested {
		intent = "requested"
	}
	support := "unsupported"
	if cleanup.Supported {
		support = "supported"
	}
	if cleanup.BlockReason != "" {
		renderHumanLine(cmd, "cleanup: %s; %s; blocked: %s", intent, support, cleanup.BlockReason)
		return
	}
	renderHumanLine(cmd, "cleanup: %s; %s", intent, support)
}

func renderOpsAPILatencyTopology(cmd *cobra.Command, topology ops.APILatencyTopologyEvidence) {
	if !topology.HealthKnown {
		return
	}
	parts := []string{
		fmt.Sprintf("brokers %d", topology.BrokerCount),
		fmt.Sprintf("partitions %d", topology.PartitionCount),
	}
	if len(topology.UnhealthyPartitions) > 0 {
		parts = append(parts, fmt.Sprintf("unhealthy %s", formatOpsAPILatencyInts(topology.UnhealthyPartitions)))
	}
	if len(topology.LeaderlessPartitions) > 0 {
		parts = append(parts, fmt.Sprintf("leaderless %s", formatOpsAPILatencyInts(topology.LeaderlessPartitions)))
	}
	renderHumanLine(cmd, "topology: %s", strings.Join(parts, "; "))
}

func renderOpsAPILatencyStages(cmd *cobra.Command, stages []ops.APILatencyStageResult, mode ops.APILatencyMode) {
	for _, stage := range stages {
		renderHumanLine(cmd, "stage %d: workers %d; %s; %s; %s",
			stage.Plan.Index,
			stage.Plan.WorkerCount,
			formatOpsAPILatencyStageAttempts(stage, mode),
			formatOpsAPILatencyStageOutcomes(stage),
			formatOpsAPILatencyStageLatency(stage),
		)
		if flagVerbose {
			renderOpsAPILatencyStageCategories(cmd, stage)
		}
	}
}

func renderOpsAPILatencyOwnership(cmd *cobra.Command, ownership *ops.APILatencyOwnership) {
	if ownership == nil {
		return
	}
	definition := "process definition not recorded"
	if ownership.ProcessDefinitionKey != "" {
		definition = "process definition recorded"
	}
	renderHumanLine(cmd, "ownership: %s; process instances %d", definition, len(ownership.ProcessInstanceKeys))
}

func renderOpsAPILatencyVisibility(cmd *cobra.Command, visibility []ops.APILatencyVisibilityResult) {
	if len(visibility) == 0 {
		return
	}
	visible := 0
	attempts := 0
	limit := 0
	var maxDuration time.Duration
	for _, item := range visibility {
		if item.Visible {
			visible++
		}
		attempts += item.Attempts
		limit += item.AttemptLimit
		if item.Duration > maxDuration {
			maxDuration = item.Duration
		}
	}
	renderHumanLine(cmd, "visibility: visible %d/%d; attempts %d/%d; max %s", visible, len(visibility), attempts, limit, maxDuration.String())
}

func renderOpsAPILatencyCleanup(cmd *cobra.Command, cleanup []ops.APILatencyCleanupRecord) {
	if len(cleanup) == 0 {
		return
	}
	deleted := 0
	retained := 0
	failed := 0
	unknown := 0
	for _, record := range cleanup {
		switch record.Status {
		case ops.APILatencyCleanupStatusDeleted:
			deleted++
		case ops.APILatencyCleanupStatusRetained:
			retained++
		case ops.APILatencyCleanupStatusFailed:
			failed++
		case ops.APILatencyCleanupStatusUnknown:
			unknown++
		}
	}
	renderHumanLine(cmd, "cleanup: deleted %d/%d; retained %d; failed %d; unknown %d", deleted, len(cleanup), retained, failed, unknown)
	renderOpsAPILatencyCleanupRecovery(cmd, cleanup)
}

// renderOpsAPILatencyCleanupRecovery lists only retained or unresolved exact resources.
func renderOpsAPILatencyCleanupRecovery(cmd *cobra.Command, cleanup []ops.APILatencyCleanupRecord) {
	for _, record := range cleanup {
		if !opsAPILatencyCleanupNeedsRecoveryLine(record) {
			continue
		}
		line := fmt.Sprintf("cleanup resource: %s %s; %s", record.ResourceType, record.Key, record.Status)
		if record.RecoveryCommand != "" {
			line += "; recovery: " + record.RecoveryCommand
		}
		renderHumanLine(cmd, "%s", line)
	}
}

// opsAPILatencyCleanupNeedsRecoveryLine keeps successful cleanup compact while surfacing resources that may remain.
func opsAPILatencyCleanupNeedsRecoveryLine(record ops.APILatencyCleanupRecord) bool {
	switch record.Status {
	case ops.APILatencyCleanupStatusRetained, ops.APILatencyCleanupStatusFailed, ops.APILatencyCleanupStatusUnknown:
		return record.Key != ""
	default:
		return false
	}
}

func renderOpsAPILatencyStageCategories(cmd *cobra.Command, stage ops.APILatencyStageResult) {
	for _, category := range stage.Categories {
		renderHumanLine(cmd, "stage %d %s: attempts %d; success %d; errors %d; timeouts %d; unavailable %d; p50 %s; p95 %s; max %s",
			stage.Plan.Index,
			category.Category,
			category.Attempts,
			category.Successes,
			category.Errors,
			category.Timeouts,
			category.Unavailable,
			formatOpsAPILatencyDuration(category.P50),
			formatOpsAPILatencyDuration(category.P95),
			formatOpsAPILatencyDuration(category.Max),
		)
	}
}

func renderOpsAPILatencyFindings(cmd *cobra.Command, findings []ops.APILatencyFinding) {
	if len(findings) == 0 {
		return
	}
	for _, finding := range findings {
		parts := []string{finding.Code}
		if finding.LikelyArea != "" {
			parts = append(parts, string(finding.LikelyArea))
		}
		if finding.Confidence != "" {
			parts = append(parts, "confidence "+string(finding.Confidence))
		}
		renderHumanLine(cmd, "finding: %s", strings.Join(parts, "; "))
		if flagVerbose {
			renderOpsAPILatencyStringList(cmd, "evidence", finding.Evidence)
			if finding.Limitation != "" {
				renderHumanLine(cmd, "limitation: %s", finding.Limitation)
			}
			if finding.NextInvestigation != "" {
				renderHumanLine(cmd, "next: %s", finding.NextInvestigation)
			}
		}
	}
}

func renderOpsAPILatencyStringList(cmd *cobra.Command, label string, values []string) {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		renderHumanLine(cmd, "%s: %s", label, value)
	}
}

func renderOpsAPILatencyOutcome(cmd *cobra.Command, result ops.APILatencyResult) {
	if result.Outcome == "" {
		return
	}
	elapsed := ""
	if result.Context.Duration != "" {
		elapsed = "; elapsed " + result.Context.Duration
	}
	renderHumanLine(cmd, "outcome: %s%s", result.Outcome, elapsed)
}

func formatOpsAPILatencyStageWidths(stages []ops.APILatencyStagePlan) string {
	if len(stages) == 0 {
		return "none"
	}
	out := make([]string, 0, len(stages))
	for _, stage := range stages {
		out = append(out, fmt.Sprintf("%d", stage.WorkerCount))
	}
	return strings.Join(out, ",")
}

func formatOpsAPILatencyInts(values []int) string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprintf("%d", value))
	}
	return strings.Join(out, ",")
}

// opsAPILatencyIntStrings adapts topology partition IDs to Markdown list helpers.
func opsAPILatencyIntStrings(values []int) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, fmt.Sprintf("%d", value))
	}
	return out
}

func formatOpsAPILatencyStageAttempts(stage ops.APILatencyStageResult, mode ops.APILatencyMode) string {
	primaryTotal := stage.Plan.PrimarySamples * 3
	if mode == ops.APILatencyModeActive {
		primaryTotal = stage.Plan.PrimarySamples
	}
	return fmt.Sprintf("primary %d/%d; derived %d/%d", stage.PrimaryAttempts, primaryTotal, stage.DerivedAttempts, stage.Plan.DerivedRequestLimit)
}

func formatOpsAPILatencyStageOutcomes(stage ops.APILatencyStageResult) string {
	errors := 0
	timeouts := 0
	unavailable := 0
	for _, category := range stage.Categories {
		errors += category.Errors
		timeouts += category.Timeouts
		unavailable += category.Unavailable
	}
	return fmt.Sprintf("errors %d; timeouts %d; unavailable %d", errors, timeouts, unavailable)
}

func formatOpsAPILatencyStageLatency(stage ops.APILatencyStageResult) string {
	category, ok := firstOpsAPILatencyLatencyCategory(stage.Categories)
	if !ok {
		return "p95 -" + formatOpsAPILatencyStageComparison(stage, "")
	}
	return fmt.Sprintf("p95 %s %s; throughput %s%s",
		category.Category,
		formatOpsAPILatencyDuration(category.P95),
		formatOpsAPILatencyThroughput(category.ThroughputPerSecond),
		formatOpsAPILatencyStageComparison(stage, category.Category),
	)
}

func firstOpsAPILatencyLatencyCategory(categories []ops.APILatencyCategorySummary) (ops.APILatencyCategorySummary, bool) {
	for _, category := range categories {
		if category.P95 != nil {
			return category, true
		}
	}
	if len(categories) == 0 {
		return ops.APILatencyCategorySummary{}, false
	}
	return categories[0], true
}

// formatOpsAPILatencyStageComparison keeps compact stage rows honest about prior-stage deltas.
func formatOpsAPILatencyStageComparison(stage ops.APILatencyStageResult, category ops.APILatencyMeasurementCategory) string {
	if stage.Plan.Index <= 1 && stage.Comparison == nil {
		return ""
	}
	comparison, ok := opsAPILatencyCategoryComparison(stage.Comparison, category)
	if !ok {
		return "; p50 delta -; throughput delta -"
	}
	return fmt.Sprintf("; p50 delta %s; throughput delta %s",
		formatOpsAPILatencySignedDuration(comparison.P50Delta),
		formatOpsAPILatencySignedThroughput(comparison.ThroughputDelta, comparison.ThroughputDeltaPercent),
	)
}

// opsAPILatencyCategoryComparison selects the comparison for the latency category used in the stage row.
func opsAPILatencyCategoryComparison(comparison *ops.APILatencyStageComparison, category ops.APILatencyMeasurementCategory) (ops.APILatencyCategoryComparison, bool) {
	if comparison == nil {
		return ops.APILatencyCategoryComparison{}, false
	}
	for _, item := range comparison.Categories {
		if item.Category == category {
			return item, true
		}
	}
	return ops.APILatencyCategoryComparison{}, false
}

func formatOpsAPILatencyDuration(value *time.Duration) string {
	if value == nil {
		return "-"
	}
	return value.String()
}

// formatOpsAPILatencySignedDuration prefixes positive deltas so regressions and improvements scan distinctly.
func formatOpsAPILatencySignedDuration(value *time.Duration) string {
	if value == nil {
		return "-"
	}
	if *value > 0 {
		return "+" + value.String()
	}
	return value.String()
}

func formatOpsAPILatencyThroughput(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f/s", *value)
}

// formatOpsAPILatencySignedThroughput pairs absolute throughput deltas with percent deltas when both are known.
func formatOpsAPILatencySignedThroughput(value *float64, percent *float64) string {
	if value == nil {
		return "-"
	}
	out := fmt.Sprintf("%+.1f/s", *value)
	if percent != nil {
		out += fmt.Sprintf(" (%+.1f%%)", *percent)
	}
	return out
}
