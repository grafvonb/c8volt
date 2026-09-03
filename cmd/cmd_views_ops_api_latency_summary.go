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

// renderOpsAPILatencyOperatorSummary keeps the default result focused on operational conclusions.
func renderOpsAPILatencyOperatorSummary(cmd *cobra.Command, result ops.APILatencyResult, active bool) {
	renderOpsAPILatencyScope(cmd, result, active)
	renderOpsAPILatencyTopology(cmd, result.Topology)
	renderOpsAPILatencyMeasurementOutcome(cmd, result.Stages, active)
	renderOpsAPILatencyLatencySummary(cmd, result.Stages, active)
	if active {
		renderOpsAPILatencyVisibilitySummary(cmd, result.Visibility)
	}
	renderOpsAPILatencyFindings(cmd, result.Findings)
}

func renderOpsAPILatencyScope(cmd *cobra.Command, result ops.APILatencyResult, active bool) {
	count := result.Request.Count
	if count <= 0 {
		count = result.Plan.PrimarySampleLimit
	}
	load := formatOpsAPILatencyLoad(result.Plan.Stages)
	if active {
		renderHumanLine(cmd, "scope: %d %s; %s", count, opsAPILatencyCountedNoun(count, "test process instance", "test process instances"), load)
		return
	}
	renderHumanLine(cmd, "scope: read only; %d %s; %s", count, opsAPILatencyCountedNoun(count, "measurement cycle", "measurement cycles"), load)
}

func formatOpsAPILatencyLoad(stages []ops.APILatencyStagePlan) string {
	if len(stages) == 0 {
		return "load not available"
	}
	first := stages[0].WorkerCount
	last := stages[len(stages)-1].WorkerCount
	if first == last {
		return fmt.Sprintf("load %d %s", last, opsAPILatencyCountedNoun(last, "worker", "workers"))
	}
	return fmt.Sprintf("load increased from %d to %d workers", first, last)
}

func renderOpsAPILatencyMeasurementOutcome(cmd *cobra.Command, stages []ops.APILatencyStageResult, active bool) {
	errors := 0
	timeouts := 0
	withoutData := 0
	for _, stage := range stages {
		for _, category := range stage.Categories {
			errors += category.Errors
			timeouts += category.Timeouts
			withoutData += category.Unavailable
		}
	}
	line := fmt.Sprintf("result: API measurements completed; request errors %d; timeouts %d", errors, timeouts)
	if !active && withoutData > 0 {
		line += fmt.Sprintf("; checks without data %d", withoutData)
	}
	renderHumanLine(cmd, "%s", line)
}

func renderOpsAPILatencyLatencySummary(cmd *cobra.Command, stages []ops.APILatencyStageResult, active bool) {
	category := ops.APILatencyCategoryProcessInstanceCreate
	latest, latestWorkers, ok := latestOpsAPILatencyCategory(stages, category)
	if !active {
		latest, latestWorkers, ok = slowestOpsAPILatencyCategory(stages)
		if ok {
			category = latest.Category
		}
	}
	if !ok || latest.P95 == nil {
		return
	}
	label := formatOpsAPILatencyCategory(category)
	workers := fmt.Sprintf("%d %s", latestWorkers, opsAPILatencyCountedNoun(latestWorkers, "worker", "workers"))
	if active {
		renderHumanLine(cmd, "create latency: 95%% completed within %s at %s", formatOpsAPILatencyOperatorDuration(*latest.P95), workers)
	} else {
		renderHumanLine(cmd, "latency: %s was slowest; 95%% completed within %s at %s", label, formatOpsAPILatencyOperatorDuration(*latest.P95), workers)
	}
	renderOpsAPILatencyLoadEffect(cmd, stages, category, latest, latestWorkers, active)
}

func renderOpsAPILatencyLoadEffect(cmd *cobra.Command, stages []ops.APILatencyStageResult, category ops.APILatencyMeasurementCategory, latest ops.APILatencyCategorySummary, latestWorkers int, active bool) {
	baseline, baselineWorkers, ok := earliestOpsAPILatencyCategory(stages, category)
	if !ok || baseline.P50 == nil || latest.P50 == nil || baselineWorkers == latestWorkers {
		return
	}
	label := formatOpsAPILatencyCategory(category)
	if active {
		label = "create response"
	}
	delta := *latest.P50 - *baseline.P50
	switch {
	case delta > 0:
		renderHumanLine(cmd, "load effect: median %s increased by %s from %d to %d workers", label, formatOpsAPILatencyOperatorDuration(delta), baselineWorkers, latestWorkers)
	case delta < 0:
		renderHumanLine(cmd, "load effect: median %s decreased by %s from %d to %d workers", label, formatOpsAPILatencyOperatorDuration(-delta), baselineWorkers, latestWorkers)
	default:
		renderHumanLine(cmd, "load effect: median %s was unchanged from %d to %d workers", label, baselineWorkers, latestWorkers)
	}
}

func latestOpsAPILatencyCategory(stages []ops.APILatencyStageResult, category ops.APILatencyMeasurementCategory) (ops.APILatencyCategorySummary, int, bool) {
	for i := len(stages) - 1; i >= 0; i-- {
		if summary, ok := findOpsAPILatencyCategory(stages[i], category); ok && summary.P95 != nil {
			return summary, stages[i].Plan.WorkerCount, true
		}
	}
	return ops.APILatencyCategorySummary{}, 0, false
}

func earliestOpsAPILatencyCategory(stages []ops.APILatencyStageResult, category ops.APILatencyMeasurementCategory) (ops.APILatencyCategorySummary, int, bool) {
	for _, stage := range stages {
		if summary, ok := findOpsAPILatencyCategory(stage, category); ok && summary.P50 != nil {
			return summary, stage.Plan.WorkerCount, true
		}
	}
	return ops.APILatencyCategorySummary{}, 0, false
}

func slowestOpsAPILatencyCategory(stages []ops.APILatencyStageResult) (ops.APILatencyCategorySummary, int, bool) {
	for i := len(stages) - 1; i >= 0; i-- {
		var slowest ops.APILatencyCategorySummary
		found := false
		for _, category := range stages[i].Categories {
			if category.P95 == nil {
				continue
			}
			if !found || *category.P95 > *slowest.P95 {
				slowest = category
				found = true
			}
		}
		if found {
			return slowest, stages[i].Plan.WorkerCount, true
		}
	}
	return ops.APILatencyCategorySummary{}, 0, false
}

func findOpsAPILatencyCategory(stage ops.APILatencyStageResult, wanted ops.APILatencyMeasurementCategory) (ops.APILatencyCategorySummary, bool) {
	for _, category := range stage.Categories {
		if category.Category == wanted {
			return category, true
		}
	}
	return ops.APILatencyCategorySummary{}, false
}

func renderOpsAPILatencyVisibilitySummary(cmd *cobra.Command, visibility []ops.APILatencyVisibilityResult) {
	if len(visibility) == 0 {
		return
	}
	visible := 0
	attempts := 0
	var slowest time.Duration
	for _, item := range visibility {
		if item.Visible {
			visible++
		}
		attempts += item.Attempts
		if item.Duration > slowest {
			slowest = item.Duration
		}
	}
	misses := attempts - visible
	if len(visibility) == 1 && visible == 1 {
		if misses == 0 {
			renderHumanLine(cmd, "search visibility: the test process instance became searchable; no visibility retries; slowest wait %s", formatOpsAPILatencyOperatorDuration(slowest))
			return
		}
		renderHumanLine(cmd, "search visibility: the test process instance became searchable; %d earlier %s did not find it; slowest wait %s", misses, opsAPILatencyCountedNoun(misses, "check", "checks"), formatOpsAPILatencyOperatorDuration(slowest))
		return
	}
	if visible == len(visibility) {
		if misses == 0 {
			renderHumanLine(cmd, "search visibility: all %d test process instances became searchable; no visibility retries; slowest wait %s", visible, formatOpsAPILatencyOperatorDuration(slowest))
			return
		}
		renderHumanLine(cmd, "search visibility: all %d test process instances became searchable; %d earlier checks found no matching instance; slowest wait %s", visible, misses, formatOpsAPILatencyOperatorDuration(slowest))
		return
	}
	renderHumanLine(cmd, "search visibility: %d/%d test process instances became searchable; %d checks found no matching instance; slowest wait %s", visible, len(visibility), misses, formatOpsAPILatencyOperatorDuration(slowest))
}

func formatOpsAPILatencyOperatorDuration(value time.Duration) string {
	switch {
	case value >= time.Second:
		return value.Round(100 * time.Millisecond).String()
	case value >= time.Millisecond:
		return value.Round(time.Millisecond).String()
	case value >= time.Microsecond:
		return value.Round(time.Microsecond).String()
	default:
		return value.String()
	}
}

func formatOpsAPILatencyCategory(category ops.APILatencyMeasurementCategory) string {
	switch category {
	case ops.APILatencyCategoryTopologyRead:
		return "topology read"
	case ops.APILatencyCategoryProcessDefinitionSearch:
		return "process definition search"
	case ops.APILatencyCategoryProcessInstanceSearch:
		return "process instance search"
	case ops.APILatencyCategoryProcessDefinitionRead:
		return "process definition read"
	case ops.APILatencyCategoryProcessInstanceRead:
		return "process instance read"
	case ops.APILatencyCategoryFixtureDeploy:
		return "fixture deployment"
	case ops.APILatencyCategoryProcessInstanceCreate:
		return "process instance create"
	case ops.APILatencyCategoryConcurrentRead:
		return "concurrent read"
	case ops.APILatencyCategorySearchVisibility:
		return "search visibility"
	default:
		return strings.ReplaceAll(string(category), "_", " ")
	}
}

func formatOpsAPILatencyFinding(code string) string {
	switch code {
	case "delayed_visibility":
		return "delayed search visibility"
	case "no_abnormal_evidence":
		return "no abnormal evidence found"
	default:
		return strings.ReplaceAll(code, "_", " ")
	}
}

func countOpsAPILatencyActionableFindings(findings []ops.APILatencyFinding) int {
	count := 0
	for _, finding := range findings {
		if finding.Code != "" && finding.Code != "no_abnormal_evidence" {
			count++
		}
	}
	return count
}

func formatOpsAPILatencyDeletedResources(cleanup []ops.APILatencyCleanupRecord) string {
	processInstances := 0
	processDefinitions := 0
	for _, record := range cleanup {
		switch record.ResourceType {
		case ops.APILatencyCleanupResourceProcessInstance:
			processInstances++
		case ops.APILatencyCleanupResourceProcessDefinition:
			processDefinitions++
		}
	}
	parts := make([]string, 0, 2)
	if processInstances > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", processInstances, opsAPILatencyCountedNoun(processInstances, "process instance", "process instances")))
	}
	if processDefinitions > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", processDefinitions, opsAPILatencyCountedNoun(processDefinitions, "process definition", "process definitions")))
	}
	return strings.Join(parts, ", ")
}

func opsAPILatencyCountedNoun(count int, singular string, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}
