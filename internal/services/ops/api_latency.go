// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

var apiLatencyCategoryOrder = []d.APILatencyMeasurementCategory{
	d.APILatencyCategoryTopologyRead,
	d.APILatencyCategoryProcessDefinitionSearch,
	d.APILatencyCategoryProcessInstanceSearch,
	d.APILatencyCategoryProcessDefinitionRead,
	d.APILatencyCategoryProcessInstanceRead,
	d.APILatencyCategoryFixtureDeploy,
	d.APILatencyCategoryProcessInstanceCreate,
	d.APILatencyCategoryConcurrentRead,
	d.APILatencyCategorySearchVisibility,
}

var apiLatencyClassificationOrder = []d.APILatencyClassification{
	d.APILatencyClassificationSuccess,
	d.APILatencyClassificationUnavailable,
	d.APILatencyClassificationUnsupported,
	d.APILatencyClassificationNotFound,
	d.APILatencyClassificationTimeout,
	d.APILatencyClassificationBackpressure,
	d.APILatencyClassificationUnhealthyPartition,
	d.APILatencyClassificationMissingLeader,
	d.APILatencyClassificationAuthentication,
	d.APILatencyClassificationConnectivity,
	d.APILatencyClassificationMalformedResponse,
	d.APILatencyClassificationRequestError,
}

// AnalyseAPILatency validates and plans a read-only API latency diagnostic for the service workflow.
func (s *Service) AnalyseAPILatency(ctx context.Context, request d.APILatencyRequest, opts ...services.CallOption) (d.APILatencyResult, error) {
	return s.analyseAPILatencyReadOnly(ctx, request, opts...)
}

// PlanAPILatency builds the deterministic stage and request-ceiling plan shared by both diagnostic modes.
func PlanAPILatency(request d.APILatencyRequest) (d.APILatencyPlan, error) {
	plan := d.APILatencyPlan{
		RunID:              "",
		Mode:               request.Mode,
		PrimarySampleLimit: request.Count,
	}
	widths, err := apiLatencyStageWidths(request.Workers)
	if err != nil {
		return plan, err
	}
	minimum := apiLatencyMinimumSamples(widths)
	if request.Count < minimum {
		return plan, fmt.Errorf("%w: count %d is too small for worker stages %s; at least %d primary samples are required", d.ErrValidation, request.Count, apiLatencyStageWidthList(widths), minimum)
	}
	if request.Count <= 0 {
		return plan, fmt.Errorf("%w: count must be greater than zero", d.ErrValidation)
	}
	if request.Workers > request.Count {
		return plan, fmt.Errorf("%w: workers must be no greater than count", d.ErrValidation)
	}

	visibilityAttempts := 0
	switch request.Mode {
	case d.APILatencyModeReadOnly, "":
		plan.Mode = d.APILatencyModeReadOnly
	case d.APILatencyModeActive:
		visibilityAttempts = APILatencyVisibilityAttemptLimit(request.Backoff)
		plan.VisibilityAttemptLimit = visibilityAttempts
		plan.SetupOperations = []string{"preflight", "deploy fixture"}
		plan.Cleanup = &d.APILatencyCleanupPlan{
			Requested:            !request.NoCleanup,
			IntentionalRetention: request.NoCleanup,
			IndependentBudget:    apiLatencyCleanupPlanBudget(request),
		}
	default:
		return plan, fmt.Errorf("%w: api latency mode must be read_only or active", d.ErrValidation)
	}

	stages := apiLatencyAllocateStages(widths, request.Count)
	for i := range stages {
		stages[i].DerivedRequestLimit = apiLatencyDerivedRequestsPerPrimary(plan.Mode, visibilityAttempts) * stages[i].PrimarySamples
		plan.PrimarySampleAllocation += stages[i].PrimarySamples
		plan.DerivedRequestLimit += stages[i].DerivedRequestLimit
	}
	plan.Stages = stages
	plan.Notices = apiLatencyPlanNotices(plan.Mode)
	plan.Limitations = apiLatencyReadOnlyLimitations()
	if plan.Mode == d.APILatencyModeActive {
		plan.Limitations = append(plan.Limitations, "active evidence is bounded by the previewed worker, primary-sample, and visibility-attempt ceilings")
	}
	return plan, nil
}

// apiLatencyCleanupPlanBudget keeps active cleanup bounded by caller retry configuration when present.
func apiLatencyCleanupPlanBudget(request d.APILatencyRequest) time.Duration {
	if request.Backoff.Timeout > 0 {
		return request.Backoff.Timeout
	}
	return apiLatencyCleanupCompletionBudget
}

// APILatencyVisibilityAttemptLimit derives the finite exact-key search-attempt ceiling from normalized backoff settings.
func APILatencyVisibilityAttemptLimit(backoff d.APILatencyBackoff) int {
	retryCeiling := backoff.MaxRetries + 1
	if backoff.MaxRetries < 0 {
		retryCeiling = 1
	}
	if retryCeiling < 1 {
		retryCeiling = 1
	}
	if backoff.Timeout <= 0 || backoff.InitialDelay <= 0 {
		return retryCeiling
	}
	attempts := 1
	elapsed := time.Duration(0)
	delay := backoff.InitialDelay
	for attempts < retryCeiling {
		elapsed += delay
		if elapsed > backoff.Timeout {
			break
		}
		attempts++
		delay = apiLatencyNextBackoffDelay(backoff, delay)
	}
	return attempts
}

// ClassifyAPILatencyError converts domain and transport failures into bounded report-safe classes.
func ClassifyAPILatencyError(err error) d.APILatencyClassification {
	if err == nil {
		return d.APILatencyClassificationSuccess
	}
	switch {
	case errors.Is(err, d.ErrUnsupported):
		return d.APILatencyClassificationUnsupported
	case errors.Is(err, d.ErrNotFound):
		return d.APILatencyClassificationNotFound
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
		return d.APILatencyClassificationTimeout
	case errors.Is(err, d.ErrRateLimited), strings.Contains(strings.ToUpper(err.Error()), "RESOURCE_EXHAUSTED"):
		return d.APILatencyClassificationBackpressure
	case errors.Is(err, d.ErrUnauthorized), errors.Is(err, d.ErrForbidden):
		return d.APILatencyClassificationAuthentication
	case errors.Is(err, d.ErrUnavailable):
		return d.APILatencyClassificationConnectivity
	case errors.Is(err, d.ErrMalformedResponse):
		return d.APILatencyClassificationMalformedResponse
	case errors.Is(err, d.ErrPrecondition):
		return d.APILatencyClassificationUnavailable
	default:
		return d.APILatencyClassificationRequestError
	}
}

// BuildAPILatencyStageResults aggregates logical measurements into deterministic stage summaries.
func BuildAPILatencyStageResults(plan d.APILatencyPlan, measurements []d.APILatencyMeasurement) []d.APILatencyStageResult {
	byStage := make(map[int][]d.APILatencyMeasurement)
	for _, measurement := range measurements {
		byStage[measurement.StageIndex] = append(byStage[measurement.StageIndex], measurement)
	}
	out := make([]d.APILatencyStageResult, 0, len(plan.Stages))
	for _, stagePlan := range plan.Stages {
		result := apiLatencyBuildStageResult(stagePlan, byStage[stagePlan.Index])
		if len(out) > 0 {
			comparison := apiLatencyCompareStage(out[len(out)-1], result)
			result.Comparison = &comparison
		}
		out = append(out, result)
	}
	return out
}

// EvaluateAPILatencyFindings returns deterministic, bounded interpretations from topology and stage evidence.
func EvaluateAPILatencyFindings(topology d.APILatencyTopologyEvidence, stages []d.APILatencyStageResult) []d.APILatencyFinding {
	var findings []d.APILatencyFinding
	if len(topology.LeaderlessPartitions) > 0 {
		findings = append(findings, d.APILatencyFinding{
			Code:              "missing_partition_leader",
			Evidence:          []string{"topology reports leaderless partitions"},
			LikelyArea:        "partition health",
			Confidence:        d.APILatencyFindingConfidenceHigh,
			Limitation:        "topology health alone does not prove the latency root cause",
			NextInvestigation: "inspect broker and partition leadership health",
		})
	}
	if len(topology.UnhealthyPartitions) > 0 {
		findings = append(findings, d.APILatencyFinding{
			Code:              "unhealthy_partition",
			Evidence:          []string{"topology reports unhealthy partitions"},
			LikelyArea:        "partition health",
			Confidence:        d.APILatencyFindingConfidenceHigh,
			Limitation:        "topology health alone does not prove the latency root cause",
			NextInvestigation: "inspect broker and partition health",
		})
	}
	if apiLatencyContainsClassification(stages, d.APILatencyClassificationBackpressure) {
		findings = append(findings, d.APILatencyFinding{
			Code:              "backpressure_observed",
			Evidence:          []string{"measurements include backpressure classifications"},
			LikelyArea:        "cluster pressure",
			Confidence:        d.APILatencyFindingConfidenceHigh,
			Limitation:        "backpressure during a bounded diagnostic is evidence, not a capacity benchmark",
			NextInvestigation: "inspect broker load, partition pressure, and client retry volume",
		})
	}
	if apiLatencyContainsClassification(stages, d.APILatencyClassificationTimeout) {
		findings = append(findings, d.APILatencyFinding{
			Code:              "timeouts_observed",
			Evidence:          []string{"measurements include timeout classifications"},
			LikelyArea:        "gateway/connectivity/authentication",
			Confidence:        d.APILatencyFindingConfidenceMedium,
			Limitation:        "request timeouts do not isolate whether the client, network, gateway, or cluster delayed the response",
			NextInvestigation: "compare c8volt request timing with gateway and broker logs",
		})
	}
	if apiLatencyQueryCategoriesSlower(stages) {
		findings = append(findings, d.APILatencyFinding{
			Code:              "query_path_degradation",
			Evidence:          []string{"query categories degrade more than topology reads as workers increase"},
			LikelyArea:        "query/secondary storage",
			Confidence:        d.APILatencyFindingConfidenceMedium,
			Limitation:        "relative degradation from a bounded sample does not prove secondary storage as the root cause",
			NextInvestigation: "compare process search latency with cluster topology latency and search-store health",
		})
	}
	if len(findings) == 0 {
		findings = append(findings, d.APILatencyFinding{
			Code:              "no_abnormal_evidence",
			Evidence:          []string{"bounded diagnostic completed without priority abnormal evidence"},
			LikelyArea:        "no abnormal evidence",
			Confidence:        d.APILatencyFindingConfidenceLow,
			Limitation:        "a bounded sample cannot prove overall health or capacity",
			NextInvestigation: "rerun with a representative count and compare against an affected time window",
		})
	}
	return findings
}

// apiLatencyStageWidths returns the deterministic worker ramp for a requested maximum.
func apiLatencyStageWidths(workers int) ([]int, error) {
	if workers <= 0 {
		return nil, fmt.Errorf("%w: workers must be greater than zero", d.ErrValidation)
	}
	widths := []int{1}
	for width := 2; width < workers; width *= 2 {
		widths = append(widths, width)
	}
	if workers != 1 {
		widths = append(widths, workers)
	}
	return widths, nil
}

// apiLatencyMinimumSamples calculates the minimum count needed to exercise each planned worker slot.
func apiLatencyMinimumSamples(widths []int) int {
	var total int
	for _, width := range widths {
		total += width
	}
	return total
}

// apiLatencyAllocateStages reserves each stage width, then spreads remaining samples from the highest stage backward.
func apiLatencyAllocateStages(widths []int, count int) []d.APILatencyStagePlan {
	stages := make([]d.APILatencyStagePlan, len(widths))
	minimum := apiLatencyMinimumSamples(widths)
	remaining := count - minimum
	even := 0
	remainder := 0
	if len(widths) > 0 && remaining > 0 {
		even = remaining / len(widths)
		remainder = remaining % len(widths)
	}
	for i, width := range widths {
		stages[i] = d.APILatencyStagePlan{
			Index:          i + 1,
			WorkerCount:    width,
			PrimarySamples: width + even,
		}
	}
	for i := len(stages) - 1; i >= 0 && remainder > 0; i-- {
		stages[i].PrimarySamples++
		remainder--
	}
	return stages
}

// apiLatencyDerivedRequestsPerPrimary returns the mode-specific derived logical call budget.
func apiLatencyDerivedRequestsPerPrimary(mode d.APILatencyMode, visibilityAttempts int) int {
	if mode == d.APILatencyModeActive {
		return 1 + visibilityAttempts
	}
	return 2
}

// apiLatencyNextBackoffDelay advances the normalized visibility backoff delay.
func apiLatencyNextBackoffDelay(backoff d.APILatencyBackoff, previous time.Duration) time.Duration {
	if backoff.Strategy != d.APILatencyBackoffExponential {
		return previous
	}
	multiplier := backoff.Multiplier
	if multiplier <= 1 {
		multiplier = 2
	}
	next := time.Duration(float64(previous) * multiplier)
	if backoff.MaxDelay > 0 && next > backoff.MaxDelay {
		return backoff.MaxDelay
	}
	return next
}

// apiLatencyPlanNotices records stable caveats from the shared bounded diagnostic plan.
func apiLatencyPlanNotices(mode d.APILatencyMode) []string {
	if mode == d.APILatencyModeActive {
		return []string{"primary samples bound process-instance creates; derived reads and visibility checks are disclosed separately"}
	}
	return []string{"direct keyed reads reuse keys returned by measured searches when available and supported"}
}

// apiLatencyReadOnlyLimitations returns the stable evidence boundaries for read-oriented diagnostics.
func apiLatencyReadOnlyLimitations() []string {
	return []string{
		"read-only evidence cannot prove write-path health",
		"read-only evidence cannot prove exporter health or end-to-end process execution health",
		"a bounded sample cannot prove overall cluster health or capacity",
	}
}

// apiLatencyStageWidthList renders the stage sequence in validation errors without leaking implementation detail.
func apiLatencyStageWidthList(widths []int) string {
	parts := make([]string, len(widths))
	for i, width := range widths {
		parts[i] = fmt.Sprintf("%d", width)
	}
	return strings.Join(parts, ", ")
}

// apiLatencyBuildStageResult folds all measurements belonging to one stage into ordered summaries.
func apiLatencyBuildStageResult(plan d.APILatencyStagePlan, measurements []d.APILatencyMeasurement) d.APILatencyStageResult {
	result := d.APILatencyStageResult{
		Plan:   plan,
		Status: d.APILatencyStageStatusCompleted,
	}
	if len(measurements) == 0 {
		result.Status = d.APILatencyStageStatusPlanned
		return result
	}
	categoryDurations := map[d.APILatencyMeasurementCategory][]time.Duration{}
	categoryCounts := map[d.APILatencyMeasurementCategory]*d.APILatencyCategorySummary{}
	classifications := map[d.APILatencyClassification]int{}
	for i, measurement := range measurements {
		if i == 0 || (!measurement.StartedAt.IsZero() && measurement.StartedAt.Before(result.StartedAt)) {
			result.StartedAt = measurement.StartedAt
		}
		finished := measurement.StartedAt.Add(measurement.Duration)
		if finished.After(result.FinishedAt) {
			result.FinishedAt = finished
		}
		if measurement.Kind == d.APILatencyMeasurementKindPrimary {
			result.PrimaryAttempts++
		}
		if measurement.Kind == d.APILatencyMeasurementKindDerived {
			result.DerivedAttempts++
		}
		summary := categoryCounts[measurement.Category]
		if summary == nil {
			summary = &d.APILatencyCategorySummary{Category: measurement.Category}
			categoryCounts[measurement.Category] = summary
		}
		summary.Attempts++
		classifications[measurement.Classification]++
		switch measurement.Outcome {
		case d.APILatencyMeasurementSucceeded:
			summary.Successes++
			categoryDurations[measurement.Category] = append(categoryDurations[measurement.Category], measurement.Duration)
		case d.APILatencyMeasurementTimedOut:
			summary.Timeouts++
		case d.APILatencyMeasurementUnavailable:
			summary.Unavailable++
		default:
			summary.Errors++
		}
	}
	result.ActualMaxConcurrency = min(plan.WorkerCount, len(measurements))
	wallDuration := result.FinishedAt.Sub(result.StartedAt)
	result.Categories = apiLatencyOrderedCategorySummaries(categoryCounts, categoryDurations, wallDuration)
	result.Classifications = apiLatencyOrderedClassificationCounts(classifications)
	return result
}

// apiLatencyOrderedCategorySummaries produces stable category order and successful-sample statistics.
func apiLatencyOrderedCategorySummaries(counts map[d.APILatencyMeasurementCategory]*d.APILatencyCategorySummary, durations map[d.APILatencyMeasurementCategory][]time.Duration, wallDuration time.Duration) []d.APILatencyCategorySummary {
	var out []d.APILatencyCategorySummary
	for _, category := range apiLatencyCategoryOrder {
		summary := counts[category]
		if summary == nil {
			continue
		}
		values := durations[category]
		if len(values) > 0 {
			sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
			summary.P50 = apiLatencyDurationPtr(apiLatencyNearestRank(values, 50))
			summary.P95 = apiLatencyDurationPtr(apiLatencyNearestRank(values, 95))
			summary.Max = apiLatencyDurationPtr(values[len(values)-1])
			if wallDuration > 0 {
				throughput := float64(summary.Successes) / wallDuration.Seconds()
				summary.ThroughputPerSecond = &throughput
			}
		}
		out = append(out, *summary)
	}
	return out
}

// apiLatencyOrderedClassificationCounts emits only present classifications in stable order.
func apiLatencyOrderedClassificationCounts(counts map[d.APILatencyClassification]int) []d.APILatencyClassificationCount {
	var out []d.APILatencyClassificationCount
	for _, class := range apiLatencyClassificationOrder {
		if count := counts[class]; count > 0 {
			out = append(out, d.APILatencyClassificationCount{Classification: class, Count: count})
		}
	}
	return out
}

// apiLatencyNearestRank calculates nearest-rank percentile values over an already sorted set.
func apiLatencyNearestRank(values []time.Duration, percentile int) time.Duration {
	if len(values) == 0 {
		return 0
	}
	rank := int(math.Ceil(float64(percentile) / 100 * float64(len(values))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(values) {
		rank = len(values)
	}
	return values[rank-1]
}

// apiLatencyDurationPtr keeps duration pointer construction local to statistics generation.
func apiLatencyDurationPtr(value time.Duration) *time.Duration {
	return &value
}

// apiLatencyCompareStage calculates category deltas only when prior baselines are non-zero.
func apiLatencyCompareStage(previous d.APILatencyStageResult, current d.APILatencyStageResult) d.APILatencyStageComparison {
	previousByCategory := map[d.APILatencyMeasurementCategory]d.APILatencyCategorySummary{}
	for _, summary := range previous.Categories {
		previousByCategory[summary.Category] = summary
	}
	comparison := d.APILatencyStageComparison{}
	for _, currentSummary := range current.Categories {
		previousSummary, ok := previousByCategory[currentSummary.Category]
		item := d.APILatencyCategoryComparison{
			Category:                 currentSummary.Category,
			ComparisonSampleCount:    previousSummary.Successes + currentSummary.Successes,
			PreviousStageUnavailable: !ok,
		}
		if ok && previousSummary.P50 != nil && currentSummary.P50 != nil && *previousSummary.P50 > 0 {
			delta := *currentSummary.P50 - *previousSummary.P50
			percent := float64(delta) / float64(*previousSummary.P50) * 100
			item.P50Delta = &delta
			item.P50DeltaPercent = &percent
		}
		if ok && previousSummary.ThroughputPerSecond != nil && currentSummary.ThroughputPerSecond != nil && *previousSummary.ThroughputPerSecond > 0 {
			delta := *currentSummary.ThroughputPerSecond - *previousSummary.ThroughputPerSecond
			percent := delta / *previousSummary.ThroughputPerSecond * 100
			item.ThroughputDelta = &delta
			item.ThroughputDeltaPercent = &percent
		}
		comparison.Categories = append(comparison.Categories, item)
	}
	return comparison
}

// apiLatencyContainsClassification checks ordered stage summaries for a bounded class.
func apiLatencyContainsClassification(stages []d.APILatencyStageResult, class d.APILatencyClassification) bool {
	for _, stage := range stages {
		for _, count := range stage.Classifications {
			if count.Classification == class && count.Count > 0 {
				return true
			}
		}
	}
	return false
}

// apiLatencyQueryCategoriesSlower detects a simple relative query-path degradation signature.
func apiLatencyQueryCategoriesSlower(stages []d.APILatencyStageResult) bool {
	for _, stage := range stages {
		if stage.Comparison == nil {
			continue
		}
		var topologyDelta float64
		var topologyKnown bool
		for _, item := range stage.Comparison.Categories {
			if item.Category == d.APILatencyCategoryTopologyRead && item.P50DeltaPercent != nil {
				topologyDelta = *item.P50DeltaPercent
				topologyKnown = true
			}
		}
		if !topologyKnown {
			continue
		}
		for _, item := range stage.Comparison.Categories {
			if item.P50DeltaPercent == nil {
				continue
			}
			if (item.Category == d.APILatencyCategoryProcessDefinitionSearch || item.Category == d.APILatencyCategoryProcessInstanceSearch) && *item.P50DeltaPercent >= topologyDelta+100 {
				return true
			}
		}
	}
	return false
}
