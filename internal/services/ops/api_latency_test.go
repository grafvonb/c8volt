// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestAPILatencyPlanAllocatesDeterministicStages verifies worker ramps, sample allocation, and read-only derived ceilings.
func TestAPILatencyPlanAllocatesDeterministicStages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		count         int
		workers       int
		wantStages    []d.APILatencyStagePlan
		wantDerived   int
		wantAllocated int
	}{
		{
			name:    "default power of two",
			count:   20,
			workers: 4,
			wantStages: []d.APILatencyStagePlan{
				{Index: 1, WorkerCount: 1, PrimarySamples: 5, DerivedRequestLimit: 10},
				{Index: 2, WorkerCount: 2, PrimarySamples: 6, DerivedRequestLimit: 12},
				{Index: 3, WorkerCount: 4, PrimarySamples: 9, DerivedRequestLimit: 18},
			},
			wantDerived:   40,
			wantAllocated: 20,
		},
		{
			name:    "non power of two maximum",
			count:   20,
			workers: 5,
			wantStages: []d.APILatencyStagePlan{
				{Index: 1, WorkerCount: 1, PrimarySamples: 3, DerivedRequestLimit: 6},
				{Index: 2, WorkerCount: 2, PrimarySamples: 4, DerivedRequestLimit: 8},
				{Index: 3, WorkerCount: 4, PrimarySamples: 6, DerivedRequestLimit: 12},
				{Index: 4, WorkerCount: 5, PrimarySamples: 7, DerivedRequestLimit: 14},
			},
			wantDerived:   40,
			wantAllocated: 20,
		},
		{
			name:    "single worker",
			count:   3,
			workers: 1,
			wantStages: []d.APILatencyStagePlan{
				{Index: 1, WorkerCount: 1, PrimarySamples: 3, DerivedRequestLimit: 6},
			},
			wantDerived:   6,
			wantAllocated: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := PlanAPILatency(d.APILatencyRequest{Mode: d.APILatencyModeReadOnly, Count: tt.count, Workers: tt.workers})

			require.NoError(t, err)
			require.Equal(t, d.APILatencyModeReadOnly, got.Mode)
			require.Equal(t, tt.wantStages, got.Stages)
			require.Equal(t, tt.wantDerived, got.DerivedRequestLimit)
			require.Equal(t, tt.wantAllocated, got.PrimarySampleAllocation)
			require.Equal(t, tt.count, got.PrimarySampleLimit)
		})
	}
}

// TestAPILatencyPlanRejectsTooSmallSampleBudgets protects the minimum count needed to exercise every stage width.
func TestAPILatencyPlanRejectsTooSmallSampleBudgets(t *testing.T) {
	t.Parallel()

	_, err := PlanAPILatency(d.APILatencyRequest{Mode: d.APILatencyModeReadOnly, Count: 4, Workers: 4})

	require.ErrorIs(t, err, d.ErrValidation)
	require.Contains(t, err.Error(), "stages 1, 2, 4")
	require.Contains(t, err.Error(), "at least 7 primary samples")
}

// TestAPILatencyActivePlanDerivesVisibilityBound verifies normalized backoff attempts bound active derived work.
func TestAPILatencyActivePlanDerivesVisibilityBound(t *testing.T) {
	t.Parallel()

	got, err := PlanAPILatency(d.APILatencyRequest{
		Mode:    d.APILatencyModeActive,
		Count:   20,
		Workers: 4,
		Backoff: d.APILatencyBackoff{
			Strategy:     d.APILatencyBackoffFixed,
			InitialDelay: 100 * time.Millisecond,
			Timeout:      250 * time.Millisecond,
			MaxRetries:   10,
		},
	})

	require.NoError(t, err)
	require.Equal(t, 3, got.VisibilityAttemptLimit)
	require.Equal(t, 80, got.DerivedRequestLimit)
	require.Equal(t, []d.APILatencyStagePlan{
		{Index: 1, WorkerCount: 1, PrimarySamples: 5, DerivedRequestLimit: 20},
		{Index: 2, WorkerCount: 2, PrimarySamples: 6, DerivedRequestLimit: 24},
		{Index: 3, WorkerCount: 4, PrimarySamples: 9, DerivedRequestLimit: 36},
	}, got.Stages)
	require.NotNil(t, got.Cleanup)
	require.True(t, got.Cleanup.Requested)
}

// TestAPILatencyStageResultsAggregateStatistics verifies nearest-rank latency, throughput, and classification order.
func TestAPILatencyStageResultsAggregateStatistics(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 9, 2, 8, 0, 0, 0, time.UTC)
	plan, err := PlanAPILatency(d.APILatencyRequest{Mode: d.APILatencyModeReadOnly, Count: 3, Workers: 1})
	require.NoError(t, err)
	measurements := []d.APILatencyMeasurement{
		{StageIndex: 1, Category: d.APILatencyCategoryTopologyRead, Kind: d.APILatencyMeasurementKindPrimary, StartedAt: base, Duration: 10 * time.Millisecond, Outcome: d.APILatencyMeasurementSucceeded, Classification: d.APILatencyClassificationSuccess},
		{StageIndex: 1, Category: d.APILatencyCategoryProcessDefinitionSearch, Kind: d.APILatencyMeasurementKindPrimary, StartedAt: base.Add(10 * time.Millisecond), Duration: 5 * time.Millisecond, Outcome: d.APILatencyMeasurementTimedOut, Classification: d.APILatencyClassificationTimeout},
		{StageIndex: 1, Category: d.APILatencyCategoryTopologyRead, Kind: d.APILatencyMeasurementKindPrimary, StartedAt: base.Add(500 * time.Millisecond), Duration: 20 * time.Millisecond, Outcome: d.APILatencyMeasurementSucceeded, Classification: d.APILatencyClassificationSuccess},
		{StageIndex: 1, Category: d.APILatencyCategoryProcessInstanceRead, Kind: d.APILatencyMeasurementKindDerived, StartedAt: base.Add(time.Second), Duration: 5 * time.Millisecond, Outcome: d.APILatencyMeasurementUnavailable, Classification: d.APILatencyClassificationUnsupported},
		{StageIndex: 1, Category: d.APILatencyCategoryTopologyRead, Kind: d.APILatencyMeasurementKindPrimary, StartedAt: base.Add(1970 * time.Millisecond), Duration: 30 * time.Millisecond, Outcome: d.APILatencyMeasurementSucceeded, Classification: d.APILatencyClassificationSuccess},
	}

	got := BuildAPILatencyStageResults(plan, measurements)

	require.Len(t, got, 1)
	require.Equal(t, d.APILatencyStageStatusCompleted, got[0].Status)
	require.Equal(t, 4, got[0].PrimaryAttempts)
	require.Equal(t, 1, got[0].DerivedAttempts)
	require.Equal(t, []d.APILatencyClassificationCount{
		{Classification: d.APILatencyClassificationSuccess, Count: 3},
		{Classification: d.APILatencyClassificationUnsupported, Count: 1},
		{Classification: d.APILatencyClassificationTimeout, Count: 1},
	}, got[0].Classifications)
	topology := got[0].Categories[0]
	require.Equal(t, d.APILatencyCategoryTopologyRead, topology.Category)
	require.Equal(t, 3, topology.Successes)
	require.Equal(t, 20*time.Millisecond, *topology.P50)
	require.Equal(t, 30*time.Millisecond, *topology.P95)
	require.Equal(t, 30*time.Millisecond, *topology.Max)
	require.InDelta(t, 1.5, *topology.ThroughputPerSecond, 0.001)
	require.Equal(t, d.APILatencyCategoryProcessDefinitionSearch, got[0].Categories[1].Category)
	require.Equal(t, 1, got[0].Categories[1].Timeouts)
	require.Equal(t, d.APILatencyCategoryProcessInstanceRead, got[0].Categories[2].Category)
	require.Equal(t, 1, got[0].Categories[2].Unavailable)
}

// TestAPILatencyStageComparisonOmitsZeroBaselines verifies deltas are unavailable when the prior stage has a zero baseline.
func TestAPILatencyStageComparisonOmitsZeroBaselines(t *testing.T) {
	t.Parallel()

	base := time.Date(2026, 9, 2, 8, 30, 0, 0, time.UTC)
	plan := d.APILatencyPlan{Stages: []d.APILatencyStagePlan{
		{Index: 1, WorkerCount: 1, PrimarySamples: 1},
		{Index: 2, WorkerCount: 2, PrimarySamples: 2},
	}}
	measurements := []d.APILatencyMeasurement{
		{StageIndex: 1, Category: d.APILatencyCategoryTopologyRead, Kind: d.APILatencyMeasurementKindPrimary, StartedAt: base, Outcome: d.APILatencyMeasurementSucceeded, Classification: d.APILatencyClassificationSuccess},
		{StageIndex: 2, Category: d.APILatencyCategoryTopologyRead, Kind: d.APILatencyMeasurementKindPrimary, StartedAt: base.Add(time.Second), Duration: 10 * time.Millisecond, Outcome: d.APILatencyMeasurementSucceeded, Classification: d.APILatencyClassificationSuccess},
	}

	got := BuildAPILatencyStageResults(plan, measurements)

	require.Len(t, got, 2)
	require.NotNil(t, got[1].Comparison)
	require.Len(t, got[1].Comparison.Categories, 1)
	require.Nil(t, got[1].Comparison.Categories[0].P50Delta)
	require.Nil(t, got[1].Comparison.Categories[0].P50DeltaPercent)
	require.Nil(t, got[1].Comparison.Categories[0].ThroughputDelta)
	require.Nil(t, got[1].Comparison.Categories[0].ThroughputDeltaPercent)
}

// TestAPILatencyClassifyErrorKeepsSafeBoundedClasses protects error-to-class mapping without raw upstream output.
func TestAPILatencyClassifyErrorKeepsSafeBoundedClasses(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want d.APILatencyClassification
	}{
		{name: "success", err: nil, want: d.APILatencyClassificationSuccess},
		{name: "unsupported", err: d.ErrUnsupported, want: d.APILatencyClassificationUnsupported},
		{name: "not found", err: d.ErrNotFound, want: d.APILatencyClassificationNotFound},
		{name: "deadline", err: context.DeadlineExceeded, want: d.APILatencyClassificationTimeout},
		{name: "rate limited", err: d.ErrRateLimited, want: d.APILatencyClassificationBackpressure},
		{name: "resource exhausted", err: fmt.Errorf("upstream 503 RESOURCE_EXHAUSTED details omitted"), want: d.APILatencyClassificationBackpressure},
		{name: "authentication", err: errors.Join(d.ErrForbidden, errors.New("denied")), want: d.APILatencyClassificationAuthentication},
		{name: "connectivity", err: d.ErrUnavailable, want: d.APILatencyClassificationConnectivity},
		{name: "malformed", err: d.ErrMalformedResponse, want: d.APILatencyClassificationMalformedResponse},
		{name: "other", err: errors.New("arbitrary upstream detail"), want: d.APILatencyClassificationRequestError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tt.want, ClassifyAPILatencyError(tt.err))
		})
	}
}

// TestAPILatencyFindingsUseDeterministicSafeOrder verifies high-priority topology/error evidence precedes fallback findings.
func TestAPILatencyFindingsUseDeterministicSafeOrder(t *testing.T) {
	t.Parallel()

	stages := []d.APILatencyStageResult{{
		Classifications: []d.APILatencyClassificationCount{
			{Classification: d.APILatencyClassificationBackpressure, Count: 1},
			{Classification: d.APILatencyClassificationTimeout, Count: 1},
		},
	}}
	topology := d.APILatencyTopologyEvidence{
		HealthKnown:          true,
		UnhealthyPartitions:  []int{2},
		LeaderlessPartitions: []int{3},
	}

	got := EvaluateAPILatencyFindings(topology, stages)

	require.Equal(t, []string{
		"missing_partition_leader",
		"unhealthy_partition",
		"backpressure_observed",
		"timeouts_observed",
	}, []string{got[0].Code, got[1].Code, got[2].Code, got[3].Code})
	for _, finding := range got {
		require.NotContains(t, finding.Limitation, "definitive root cause")
		require.NotEmpty(t, finding.NextInvestigation)
	}
}

// TestAPILatencyFindingsFallbackAvoidsHealthClaims verifies no-signal evidence remains explicitly bounded.
func TestAPILatencyFindingsFallbackAvoidsHealthClaims(t *testing.T) {
	t.Parallel()

	got := EvaluateAPILatencyFindings(d.APILatencyTopologyEvidence{HealthKnown: false}, nil)

	require.Len(t, got, 1)
	require.Equal(t, "no_abnormal_evidence", got[0].Code)
	require.Equal(t, d.APILatencyFindingConfidenceLow, got[0].Confidence)
	require.Contains(t, got[0].Limitation, "cannot prove overall health")
}
