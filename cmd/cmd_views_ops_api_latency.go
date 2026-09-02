// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
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
	renderOpsAPILatencyOutcome(cmd, result)
	return nil
}

func opsAPILatencyResultIsActive(result ops.APILatencyResult) bool {
	return result.Plan.Mode == ops.APILatencyModeActive || result.Request.Mode == ops.APILatencyModeActive || result.Ownership != nil
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
		return "p95 -"
	}
	return fmt.Sprintf("p95 %s %s; throughput %s", category.Category, formatOpsAPILatencyDuration(category.P95), formatOpsAPILatencyThroughput(category.ThroughputPerSecond))
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

func formatOpsAPILatencyDuration(value *time.Duration) string {
	if value == nil {
		return "-"
	}
	return value.String()
}

func formatOpsAPILatencyThroughput(value *float64) string {
	if value == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f/s", *value)
}
