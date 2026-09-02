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
	renderHumanLine(cmd, "analyse api latency")
	renderAttachedTenantContext(cmd)
	renderOpsAPILatencyPlan(cmd, result.Plan)
	renderOpsAPILatencyTopology(cmd, result.Topology)
	renderOpsAPILatencyStages(cmd, result.Stages)
	renderOpsAPILatencyFindings(cmd, result.Findings)
	renderOpsAPILatencyStringList(cmd, "notice", result.Notices)
	renderOpsAPILatencyStringList(cmd, "limitation", result.Limitations)
	renderOpsAPILatencyOutcome(cmd, result)
	return nil
}

func renderOpsAPILatencyPlan(cmd *cobra.Command, plan ops.APILatencyPlan) {
	if plan.PrimarySampleLimit == 0 && len(plan.Stages) == 0 {
		return
	}
	renderHumanLine(cmd, "request: count %d; workers %s; stages %d; derived requests <= %d",
		plan.PrimarySampleLimit,
		formatOpsAPILatencyStageWidths(plan.Stages),
		len(plan.Stages),
		plan.DerivedRequestLimit,
	)
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

func renderOpsAPILatencyStages(cmd *cobra.Command, stages []ops.APILatencyStageResult) {
	for _, stage := range stages {
		renderHumanLine(cmd, "stage %d: workers %d; %s; %s; %s",
			stage.Plan.Index,
			stage.Plan.WorkerCount,
			formatOpsAPILatencyStageAttempts(stage),
			formatOpsAPILatencyStageOutcomes(stage),
			formatOpsAPILatencyStageLatency(stage),
		)
		if flagVerbose {
			renderOpsAPILatencyStageCategories(cmd, stage)
		}
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

func formatOpsAPILatencyStageAttempts(stage ops.APILatencyStageResult) string {
	primaryTotal := stage.Plan.PrimarySamples * 3
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
