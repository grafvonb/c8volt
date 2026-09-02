// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
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

// TestAPILatencyReadOnlyMeasuresSearchesAndDerivedReads verifies US1 read-only measurements reuse discovered keys without mutation calls.
func TestAPILatencyReadOnlyMeasuresSearchesAndDerivedReads(t *testing.T) {
	t.Parallel()

	var createCalls atomic.Int64
	var deleteCalls atomic.Int64
	var pdGetKeys []string
	var piGetKeys []string
	clusterAPI := &stubSmokeTestClusterAPI{topology: d.Topology{
		ClusterSize:     1,
		PartitionsCount: 2,
		Brokers: []d.Broker{{
			Partitions: []d.Partition{
				{PartitionId: 1, Health: d.PartitionHealth("HEALTHY"), Role: d.PartitionRole("LEADER")},
				{PartitionId: 2, Health: d.PartitionHealth("UNHEALTHY"), Role: d.PartitionRole("FOLLOWER")},
			},
		}},
	}}
	pdAPI := stubProcessDefinitionAPI{
		searchProcessDefinitions: func(context.Context, d.ProcessDefinitionFilter, int32, ...services.CallOption) ([]d.ProcessDefinition, error) {
			return []d.ProcessDefinition{{Key: "pd-1", BpmnProcessId: "Process"}}, nil
		},
		getProcessDefinition: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
			pdGetKeys = append(pdGetKeys, key)
			return d.ProcessDefinition{Key: key}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error) {
			createCalls.Add(1)
			return d.ProcessInstanceCreation{}, errors.New("unexpected create")
		},
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return []d.ProcessInstance{{Key: "pi-1", ProcessDefinitionKey: "pd-1"}}, nil
		},
		getProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessInstance, error) {
			piGetKeys = append(piGetKeys, key)
			return d.ProcessInstance{Key: key}, nil
		},
		deleteProcessInstance: func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
			deleteCalls.Add(1)
			return d.DeleteResponse{}, errors.New("unexpected delete")
		},
	}

	got, err := NewWithAnalysisDependencies(clusterAPI, piAPI, nil, pdAPI, nil, nil, nil, toolx.V89).AnalyseAPILatency(context.Background(), d.APILatencyRequest{
		CommandName: "ops analyse api-latency",
		Count:       3,
		Workers:     1,
		TenantID:    "tenant-a",
	})

	require.NoError(t, err)
	require.Equal(t, d.APILatencyOutcomeCompleted, got.Outcome)
	require.Equal(t, d.APILatencyModeReadOnly, got.Plan.Mode)
	require.Equal(t, 6, got.Plan.DerivedRequestLimit)
	require.Equal(t, 1, got.Topology.BrokerCount)
	require.Equal(t, 2, got.Topology.PartitionCount)
	require.Equal(t, []int{2}, got.Topology.UnhealthyPartitions)
	require.Equal(t, []int{2}, got.Topology.LeaderlessPartitions)
	require.Equal(t, []string{"pd-1", "pd-1", "pd-1"}, pdGetKeys)
	require.Equal(t, []string{"pi-1", "pi-1", "pi-1"}, piGetKeys)
	require.Zero(t, createCalls.Load())
	require.Zero(t, deleteCalls.Load())
	require.Len(t, got.Stages, 1)
	require.Equal(t, 9, got.Stages[0].PrimaryAttempts)
	require.Equal(t, 6, got.Stages[0].DerivedAttempts)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryTopologyRead, 3, 3, 0, 0, 0)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryProcessDefinitionSearch, 3, 3, 0, 0, 0)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryProcessInstanceSearch, 3, 3, 0, 0, 0)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryProcessDefinitionRead, 3, 3, 0, 0, 0)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryProcessInstanceRead, 3, 3, 0, 0, 0)
}

// TestAPILatencyReadOnlyReportsUnavailableKeyedEvidence verifies missing and unsupported derived reads are measured, not fabricated.
func TestAPILatencyReadOnlyReportsUnavailableKeyedEvidence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		version toolx.CamundaVersion
		pdItems []d.ProcessDefinition
		piItems []d.ProcessInstance
		piGet   func(context.Context, string, ...services.CallOption) (d.ProcessInstance, error)
		wantPD  d.APILatencyClassification
		wantPI  d.APILatencyClassification
	}{
		{
			name:    "empty searches",
			version: toolx.V89,
			wantPD:  d.APILatencyClassificationUnavailable,
			wantPI:  d.APILatencyClassificationUnavailable,
		},
		{
			name:    "process instance keyed read unsupported",
			version: toolx.V87,
			pdItems: []d.ProcessDefinition{{Key: "pd-1"}},
			piItems: []d.ProcessInstance{{Key: "pi-1"}},
			wantPD:  d.APILatencyClassificationSuccess,
			wantPI:  d.APILatencyClassificationUnsupported,
		},
		{
			name:    "process instance disappears",
			version: toolx.V89,
			pdItems: []d.ProcessDefinition{{Key: "pd-1"}},
			piItems: []d.ProcessInstance{{Key: "pi-1"}},
			piGet: func(context.Context, string, ...services.CallOption) (d.ProcessInstance, error) {
				return d.ProcessInstance{}, d.ErrNotFound
			},
			wantPD: d.APILatencyClassificationSuccess,
			wantPI: d.APILatencyClassificationNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pdAPI := stubProcessDefinitionAPI{
				searchProcessDefinitions: func(context.Context, d.ProcessDefinitionFilter, int32, ...services.CallOption) ([]d.ProcessDefinition, error) {
					return tt.pdItems, nil
				},
				getProcessDefinition: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
					return d.ProcessDefinition{Key: key}, nil
				},
			}
			piAPI := stubProcessInstanceAPI{
				search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
					return tt.piItems, nil
				},
				getProcessInstance: tt.piGet,
			}
			if piAPI.getProcessInstance == nil {
				piAPI.getProcessInstance = func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessInstance, error) {
					return d.ProcessInstance{Key: key}, nil
				}
			}

			got, err := NewWithAnalysisDependencies(&stubSmokeTestClusterAPI{}, piAPI, nil, pdAPI, nil, nil, nil, tt.version).AnalyseAPILatency(context.Background(), d.APILatencyRequest{Count: 1, Workers: 1})

			require.NoError(t, err)
			requireAPILatencyCategoryClass(t, got.Stages[0], d.APILatencyCategoryProcessDefinitionRead, tt.wantPD)
			requireAPILatencyCategoryClass(t, got.Stages[0], d.APILatencyCategoryProcessInstanceRead, tt.wantPI)
		})
	}
}

// TestAPILatencyReadOnlyCompletesWithAbnormalSamples verifies request failures become completed diagnostic evidence.
func TestAPILatencyReadOnlyCompletesWithAbnormalSamples(t *testing.T) {
	t.Parallel()

	pdAPI := stubProcessDefinitionAPI{
		searchProcessDefinitions: func(context.Context, d.ProcessDefinitionFilter, int32, ...services.CallOption) ([]d.ProcessDefinition, error) {
			return nil, d.ErrRateLimited
		},
	}
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return nil, nil
		},
	}

	got, err := NewWithAnalysisDependencies(&stubSmokeTestClusterAPI{}, piAPI, nil, pdAPI, nil, nil, nil, toolx.V89).AnalyseAPILatency(context.Background(), d.APILatencyRequest{Count: 1, Workers: 1})

	require.NoError(t, err)
	require.Equal(t, d.APILatencyOutcomeCompleted, got.Outcome)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryProcessDefinitionSearch, 1, 0, 1, 0, 0)
	requireAPILatencyCategoryClass(t, got.Stages[0], d.APILatencyCategoryProcessDefinitionSearch, d.APILatencyClassificationBackpressure)
	require.Equal(t, "backpressure_observed", got.Findings[0].Code)
}

// TestAPILatencyReadOnlyUsesBoundedStageWorkers verifies the service never exceeds the planned worker ceiling.
func TestAPILatencyReadOnlyUsesBoundedStageWorkers(t *testing.T) {
	var active atomic.Int64
	var maxActive atomic.Int64
	var searchCalls atomic.Int64
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	pdAPI := stubProcessDefinitionAPI{
		searchProcessDefinitions: func(ctx context.Context, _ d.ProcessDefinitionFilter, _ int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
			call := searchCalls.Add(1)
			if call > 1 {
				now := active.Add(1)
				for {
					seen := maxActive.Load()
					if now <= seen || maxActive.CompareAndSwap(seen, now) {
						break
					}
				}
				defer active.Add(-1)
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-release:
				}
			}
			return []d.ProcessDefinition{{Key: "pd-1"}}, nil
		},
		getProcessDefinition: func(_ context.Context, key string, _ ...services.CallOption) (d.ProcessDefinition, error) {
			return d.ProcessDefinition{Key: key}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			return nil, nil
		},
	}
	done := make(chan error, 1)

	go func() {
		_, err := NewWithAnalysisDependencies(&stubSmokeTestClusterAPI{}, piAPI, nil, pdAPI, nil, nil, nil, toolx.V89).AnalyseAPILatency(context.Background(), d.APILatencyRequest{Count: 3, Workers: 2})
		done <- err
	}()

	require.Eventually(t, func() bool { return maxActive.Load() == 2 }, time.Second, 10*time.Millisecond)
	require.Never(t, func() bool { return maxActive.Load() > 2 }, 25*time.Millisecond, 5*time.Millisecond)
	close(release)
	require.NoError(t, <-done)
}

// TestAPILatencyReadOnlyCancellationReturnsInterruptedPartialResult verifies caller cancellation stops measurement scheduling.
func TestAPILatencyReadOnlyCancellationReturnsInterruptedPartialResult(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := NewWithAnalysisDependencies(&stubSmokeTestClusterAPI{}, stubProcessInstanceAPI{}, nil, stubProcessDefinitionAPI{}, nil, nil, nil, toolx.V89).AnalyseAPILatency(ctx, d.APILatencyRequest{Count: 1, Workers: 1})

	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, d.APILatencyOutcomeInterrupted, got.Outcome)
	require.Len(t, got.Stages, 1)
	require.Equal(t, d.APILatencyStageStatusPlanned, got.Stages[0].Status)
}

// TestAPILatencyActiveDryRunPlansRunIdentityFixtureAndVersion verifies active preflight produces an immutable zero-mutation preview.
func TestAPILatencyActiveDryRunPlansRunIdentityFixtureAndVersion(t *testing.T) {
	t.Parallel()

	cluster := &stubSmokeTestClusterAPI{topology: d.Topology{
		GatewayVersion:    "8.9.2",
		ClusterSize:       1,
		PartitionsCount:   1,
		ReplicationFactor: 1,
		Brokers: []d.Broker{{
			Partitions: []d.Partition{{PartitionId: 1, Health: d.PartitionHealth("HEALTHY"), Role: d.PartitionRole("LEADER")}},
		}},
	}}
	resource := &stubSmokeTestResourceAPI{}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error) {
			return d.ProcessInstanceCreation{}, errors.New("unexpected active dry-run create")
		},
		deleteProcessInstance: func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
			return d.DeleteResponse{}, errors.New("unexpected active dry-run delete")
		},
	}
	pdAPI := stubProcessDefinitionAPI{
		searchProcessDefinitions: func(context.Context, d.ProcessDefinitionFilter, int32, ...services.CallOption) ([]d.ProcessDefinition, error) {
			return nil, errors.New("unexpected active dry-run search")
		},
	}

	got, err := NewWithAnalysisDependencies(cluster, piAPI, nil, pdAPI, resource, nil, nil, toolx.V89).ExecuteAPILatencyTest(context.Background(), d.APILatencyRequest{
		CommandName: "ops execute api-latency-test",
		Count:       3,
		Workers:     1,
		DryRun:      true,
		TenantID:    "tenant-a",
		Backoff: d.APILatencyBackoff{
			Strategy:     d.APILatencyBackoffFixed,
			InitialDelay: 100 * time.Millisecond,
			Timeout:      time.Second,
			MaxRetries:   2,
		},
	})

	require.NoError(t, err)
	require.Equal(t, 1, cluster.topologyCalls)
	require.Zero(t, resource.deployCalls)
	require.Zero(t, resource.deleteCalls)
	requireAPILatencyRunID(t, got.Plan.RunID)
	require.Equal(t, d.APILatencyOutcomePlanned, got.Outcome)
	require.Equal(t, d.APILatencyModeActive, got.Request.Mode)
	require.Equal(t, d.APILatencyModeActive, got.Plan.Mode)
	require.Equal(t, "8.9", got.Context.CamundaVersion)
	require.Equal(t, "tenant-a", got.Context.Tenant)
	require.Equal(t, 1, got.Topology.PartitionCount)
	require.NotNil(t, got.Plan.Fixture)
	require.Equal(t, "8.9", got.Plan.Fixture.CamundaVersion)
	require.Equal(t, "embedded/processdefinitions/C89_SimpleUserTask.bpmn", got.Plan.Fixture.File)
	require.Equal(t, "C89_SimpleUserTask", got.Plan.Fixture.BpmnProcessID)
	require.True(t, got.Plan.Fixture.Available)
	require.NotNil(t, got.Plan.Cleanup)
	require.True(t, got.Plan.Cleanup.Requested)
	require.True(t, got.Plan.Cleanup.Supported)
	require.False(t, got.Plan.Cleanup.IntentionalRetention)
	require.Empty(t, got.Plan.Cleanup.BlockReason)
	require.Equal(t, []string{"preflight", "deploy fixture"}, got.Plan.SetupOperations)
	require.Contains(t, got.Notices, "configured Camunda 8.9 matches observed gateway 8.9.2")
}

// TestAPILatencyActiveDryRunRejectsObservedVersionMismatch protects preflight from planning mutation for a different gateway line.
func TestAPILatencyActiveDryRunRejectsObservedVersionMismatch(t *testing.T) {
	t.Parallel()

	cluster := &stubSmokeTestClusterAPI{topology: d.Topology{GatewayVersion: "8.8.9"}}
	resource := &stubSmokeTestResourceAPI{}

	got, err := NewWithAnalysisDependencies(cluster, stubProcessInstanceAPI{}, nil, stubProcessDefinitionAPI{}, resource, nil, nil, toolx.V89).ExecuteAPILatencyTest(context.Background(), d.APILatencyRequest{
		Count:    1,
		Workers:  1,
		DryRun:   true,
		TenantID: "tenant-a",
	})

	require.ErrorIs(t, err, d.ErrPrecondition)
	require.Contains(t, err.Error(), "configured Camunda 8.9 does not match observed gateway 8.8.9")
	require.Equal(t, 1, cluster.topologyCalls)
	require.Zero(t, resource.deployCalls)
	require.Equal(t, d.APILatencyOutcomeFailed, got.Outcome)
	require.NotNil(t, got.Plan.Cleanup)
	require.NotEmpty(t, got.Plan.Cleanup.BlockReason)
}

// TestAPILatencyActivePreflightVersionEligibility verifies exact-key ownership and cleanup capability gates before mutation.
func TestAPILatencyActivePreflightVersionEligibility(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		version           toolx.CamundaVersion
		noCleanup         bool
		wantErr           error
		wantFixture       string
		wantCleanup       bool
		wantSupported     bool
		wantRetention     bool
		wantBlockContains string
	}{
		{
			name:              "8.7 rejected even with retention",
			version:           toolx.V87,
			noCleanup:         true,
			wantErr:           d.ErrUnsupported,
			wantFixture:       "embedded/processdefinitions/C87_SimpleUserTask.bpmn",
			wantRetention:     true,
			wantBlockContains: "Camunda 8.7 cannot guarantee exact active latency ownership keys",
		},
		{
			name:              "8.8 cleanup enabled rejected",
			version:           toolx.V88,
			wantErr:           d.ErrUnsupported,
			wantFixture:       "embedded/processdefinitions/C88_SimpleUserTask.bpmn",
			wantCleanup:       true,
			wantBlockContains: "Camunda 8.8 cannot completely clean up active latency process-definition history",
		},
		{
			name:          "8.8 retained eligible",
			version:       toolx.V88,
			noCleanup:     true,
			wantFixture:   "embedded/processdefinitions/C88_SimpleUserTask.bpmn",
			wantSupported: false,
			wantRetention: true,
		},
		{
			name:          "8.9 cleanup eligible",
			version:       toolx.V89,
			wantFixture:   "embedded/processdefinitions/C89_SimpleUserTask.bpmn",
			wantCleanup:   true,
			wantSupported: true,
		},
		{
			name:          "8.10 cleanup eligible",
			version:       toolx.V810,
			wantFixture:   "embedded/processdefinitions/C810_SimpleUserTask.bpmn",
			wantCleanup:   true,
			wantSupported: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resource := &stubSmokeTestResourceAPI{}
			got, err := NewWithAnalysisDependencies(&stubSmokeTestClusterAPI{
				topology: d.Topology{GatewayVersion: tt.version.String() + ".1"},
			}, stubProcessInstanceAPI{}, nil, stubProcessDefinitionAPI{}, resource, nil, nil, tt.version).ExecuteAPILatencyTest(context.Background(), d.APILatencyRequest{
				Count:     1,
				Workers:   1,
				DryRun:    true,
				NoCleanup: tt.noCleanup,
				TenantID:  "tenant-a",
			})

			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, d.APILatencyOutcomeFailed, got.Outcome)
			} else {
				require.NoError(t, err)
				require.Equal(t, d.APILatencyOutcomePlanned, got.Outcome)
			}
			require.Zero(t, resource.deployCalls)
			require.NotNil(t, got.Plan.Fixture)
			require.Equal(t, tt.wantFixture, got.Plan.Fixture.File)
			require.NotNil(t, got.Plan.Cleanup)
			require.Equal(t, tt.wantCleanup, got.Plan.Cleanup.Requested)
			require.Equal(t, tt.wantSupported, got.Plan.Cleanup.Supported)
			require.Equal(t, tt.wantRetention, got.Plan.Cleanup.IntentionalRetention)
			if tt.wantBlockContains == "" {
				require.Empty(t, got.Plan.Cleanup.BlockReason)
			} else {
				require.Contains(t, got.Plan.Cleanup.BlockReason, tt.wantBlockContains)
				require.Contains(t, err.Error(), tt.wantBlockContains)
			}
		})
	}
}

// TestAPILatencyActiveExecutionUsesExactReturnedKeysAndCleansUp verifies active stages use only returned ownership keys.
func TestAPILatencyActiveExecutionUsesExactReturnedKeysAndCleansUp(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	var createRequests []d.ProcessInstanceData
	var visibilityFilters []d.ProcessInstanceFilter
	var deletedProcessInstances []string
	var deletedProcessDefinitions []string
	cluster := &stubSmokeTestClusterAPI{topology: d.Topology{GatewayVersion: "8.9.4", ClusterSize: 1, PartitionsCount: 1}}
	resource := &stubSmokeTestResourceAPI{
		deploy: func(_ context.Context, units []d.DeploymentUnitData, opts ...services.CallOption) (d.Deployment, error) {
			require.True(t, services.ApplyCallOptions(opts).NoWait)
			require.Len(t, units, 1)
			require.Equal(t, "processdefinitions/C89_SimpleUserTask.bpmn", units[0].Name)
			return d.Deployment{Key: "deployment-1", TenantId: "tenant-a", Units: []d.DeploymentUnit{{
				ProcessDefinition: d.ProcessDefinitionDeployment{
					ProcessDefinitionId:      "C89_SimpleUserTask",
					ProcessDefinitionKey:     "pd-active",
					ProcessDefinitionVersion: 3,
					ResourceName:             units[0].Name,
					TenantId:                 "tenant-a",
				},
			}}}, nil
		},
		delete: func(_ context.Context, key string, _ ...services.CallOption) (d.ResourceDeleteResponse, error) {
			mu.Lock()
			deletedProcessDefinitions = append(deletedProcessDefinitions, key)
			mu.Unlock()
			return d.ResourceDeleteResponse{Key: key, Ok: true, StatusCode: 202, Status: "accepted", DeleteHistory: true}, nil
		},
	}
	var created atomic.Int64
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			key := fmt.Sprintf("pi-%d", created.Add(1))
			mu.Lock()
			createRequests = append(createRequests, data)
			mu.Unlock()
			return d.ProcessInstanceCreation{
				Key:                  key,
				BpmnProcessId:        "C89_SimpleUserTask",
				ProcessDefinitionKey: data.ProcessDefinitionSpecificId,
				TenantId:             data.TenantId,
			}, nil
		},
		search: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, int32(1), size)
			mu.Lock()
			visibilityFilters = append(visibilityFilters, filter)
			mu.Unlock()
			if filter.Key != "" {
				return []d.ProcessInstance{{Key: filter.Key, ProcessDefinitionKey: filter.ProcessDefinitionKey, TenantId: "tenant-a"}}, nil
			}
			return []d.ProcessInstance{{Key: "read-probe", ProcessDefinitionKey: filter.ProcessDefinitionKey, TenantId: "tenant-a"}}, nil
		},
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			mu.Lock()
			deletedProcessInstances = append(deletedProcessInstances, key)
			mu.Unlock()
			return d.DeleteResponse{Ok: true, StatusCode: 202, Status: "accepted"}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(cluster, piAPI, nil, stubProcessDefinitionAPI{}, resource, nil, nil, toolx.V89).ExecuteAPILatencyTest(context.Background(), d.APILatencyRequest{
		CommandName: "ops execute api-latency-test",
		Count:       3,
		Workers:     1,
		TenantID:    "tenant-a",
		Backoff:     d.APILatencyBackoff{MaxRetries: 0},
	})

	require.NoError(t, err)
	require.Equal(t, d.APILatencyOutcomeCompleted, got.Outcome)
	require.Equal(t, 1, cluster.topologyCalls)
	require.Equal(t, 1, resource.deployCalls)
	require.Equal(t, 1, resource.deleteCalls)
	require.NotNil(t, got.Ownership)
	requireAPILatencyRunID(t, got.Ownership.RunID)
	require.Equal(t, got.Plan.RunID, got.Ownership.RunID)
	require.True(t, got.Ownership.DeploymentSubmitted)
	require.Equal(t, "C89_SimpleUserTask", got.Ownership.BpmnProcessID)
	require.Equal(t, "pd-active", got.Ownership.ProcessDefinitionKey)
	require.Equal(t, []string{"pi-1", "pi-2", "pi-3"}, got.Ownership.ProcessInstanceKeys)
	require.Equal(t, []string{"pi-1", "pi-2", "pi-3"}, deletedProcessInstances)
	require.Equal(t, []string{"pd-active"}, deletedProcessDefinitions)
	require.Len(t, got.Visibility, 3)
	for _, item := range got.Visibility {
		require.True(t, item.Visible)
		require.Equal(t, 1, item.Attempts)
		require.Equal(t, 1, item.AttemptLimit)
		require.Equal(t, d.APILatencyClassificationSuccess, item.FinalClassification)
	}
	require.Len(t, got.Cleanup, 4)
	require.Equal(t, d.APILatencyCleanupResourceProcessInstance, got.Cleanup[0].ResourceType)
	require.Equal(t, d.APILatencyCleanupResourceProcessDefinition, got.Cleanup[3].ResourceType)
	require.Empty(t, got.Cleanup[3].RecoveryCommand)
	require.Len(t, got.Stages, 1)
	require.Equal(t, 3, got.Stages[0].PrimaryAttempts)
	require.Equal(t, 6, got.Stages[0].DerivedAttempts)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryFixtureDeploy, 1, 1, 0, 0, 0)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryProcessInstanceCreate, 3, 3, 0, 0, 0)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategoryConcurrentRead, 3, 3, 0, 0, 0)
	requireAPILatencyCategory(t, got.Stages[0], d.APILatencyCategorySearchVisibility, 3, 3, 0, 0, 0)
	require.Equal(t, []d.ProcessInstanceData{
		{ProcessDefinitionSpecificId: "pd-active", TenantId: "tenant-a"},
		{ProcessDefinitionSpecificId: "pd-active", TenantId: "tenant-a"},
		{ProcessDefinitionSpecificId: "pd-active", TenantId: "tenant-a"},
	}, createRequests)
	for _, filter := range visibilityFilters {
		require.NotEqual(t, "C89_SimpleUserTask", filter.BpmnProcessId)
		require.Equal(t, "pd-active", filter.ProcessDefinitionKey)
	}
}

// TestAPILatencyActiveExecutionBoundsWorkersAndClassifiesVisibilityErrors verifies bounded stage work and safe active evidence.
func TestAPILatencyActiveExecutionBoundsWorkersAndClassifiesVisibilityErrors(t *testing.T) {
	var activeCreates atomic.Int64
	var maxActiveCreates atomic.Int64
	var createCalls atomic.Int64
	var visibilityAttempts atomic.Int64
	release := make(chan struct{})
	t.Cleanup(func() {
		select {
		case <-release:
		default:
			close(release)
		}
	})
	cluster := &stubSmokeTestClusterAPI{topology: d.Topology{GatewayVersion: "8.9.0"}}
	resource := &stubSmokeTestResourceAPI{
		deploy: func(context.Context, []d.DeploymentUnitData, ...services.CallOption) (d.Deployment, error) {
			return d.Deployment{Units: []d.DeploymentUnit{{ProcessDefinition: d.ProcessDefinitionDeployment{
				ProcessDefinitionId:  "C89_SimpleUserTask",
				ProcessDefinitionKey: "pd-active",
			}}}}, nil
		},
		delete: func(context.Context, string, ...services.CallOption) (d.ResourceDeleteResponse, error) {
			return d.ResourceDeleteResponse{Ok: true}, nil
		},
	}
	piAPI := stubProcessInstanceAPI{
		createProcessInstance: func(ctx context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			call := createCalls.Add(1)
			now := activeCreates.Add(1)
			for {
				seen := maxActiveCreates.Load()
				if now <= seen || maxActiveCreates.CompareAndSwap(seen, now) {
					break
				}
			}
			defer activeCreates.Add(-1)
			if call > 1 {
				select {
				case <-ctx.Done():
					return d.ProcessInstanceCreation{}, ctx.Err()
				case <-release:
				}
			}
			return d.ProcessInstanceCreation{Key: fmt.Sprintf("pi-%d", call), ProcessDefinitionKey: data.ProcessDefinitionSpecificId}, nil
		},
		search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			if filter.Key == "pi-2" {
				visibilityAttempts.Add(1)
				return nil, d.ErrRateLimited
			}
			if filter.Key != "" {
				visibilityAttempts.Add(1)
				return []d.ProcessInstance{{Key: filter.Key, ProcessDefinitionKey: filter.ProcessDefinitionKey}}, nil
			}
			return []d.ProcessInstance{{Key: "read-probe", ProcessDefinitionKey: filter.ProcessDefinitionKey}}, nil
		},
		deleteProcessInstance: func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
			return d.DeleteResponse{Ok: true}, nil
		},
	}
	done := make(chan struct {
		result d.APILatencyResult
		err    error
	}, 1)

	go func() {
		got, err := NewWithAnalysisDependencies(cluster, piAPI, nil, stubProcessDefinitionAPI{}, resource, nil, nil, toolx.V89).ExecuteAPILatencyTest(context.Background(), d.APILatencyRequest{
			Count:    3,
			Workers:  2,
			TenantID: "tenant-a",
			Backoff:  d.APILatencyBackoff{MaxRetries: 0},
		})
		done <- struct {
			result d.APILatencyResult
			err    error
		}{result: got, err: err}
	}()

	require.Eventually(t, func() bool { return maxActiveCreates.Load() == 2 }, time.Second, 10*time.Millisecond)
	require.Never(t, func() bool { return maxActiveCreates.Load() > 2 }, 25*time.Millisecond, 5*time.Millisecond)
	close(release)
	finished := <-done

	require.NoError(t, finished.err)
	require.Equal(t, d.APILatencyOutcomeCompleted, finished.result.Outcome)
	require.Equal(t, int64(3), createCalls.Load())
	require.Equal(t, int64(3), visibilityAttempts.Load())
	require.Len(t, finished.result.Stages, 2)
	require.LessOrEqual(t, finished.result.Stages[0].ActualMaxConcurrency, 1)
	require.LessOrEqual(t, finished.result.Stages[1].ActualMaxConcurrency, 2)
	requireAPILatencyCategoryClass(t, finished.result.Stages[1], d.APILatencyCategorySearchVisibility, d.APILatencyClassificationBackpressure)
	require.Equal(t, "backpressure_observed", finished.result.Findings[0].Code)
	require.Len(t, finished.result.Visibility, 3)
	backpressureVisibility := requireAPILatencyVisibility(t, finished.result.Visibility, "pi-2")
	require.False(t, backpressureVisibility.Visible)
	require.Equal(t, d.APILatencyClassificationBackpressure, backpressureVisibility.FinalClassification)
}

// requireAPILatencyRunID verifies the active run identity is a nonzero 128-bit hex string.
func requireAPILatencyRunID(t *testing.T, runID string) {
	t.Helper()
	require.Len(t, runID, 32)
	decoded, err := hex.DecodeString(runID)
	require.NoError(t, err)
	require.Len(t, decoded, 16)
	require.NotEqual(t, make([]byte, 16), decoded)
}

// requireAPILatencyCategory finds a stage category and checks its aggregate counts.
func requireAPILatencyCategory(t *testing.T, stage d.APILatencyStageResult, category d.APILatencyMeasurementCategory, attempts int, successes int, errors int, timeouts int, unavailable int) {
	t.Helper()
	for _, summary := range stage.Categories {
		if summary.Category == category {
			require.Equal(t, attempts, summary.Attempts)
			require.Equal(t, successes, summary.Successes)
			require.Equal(t, errors, summary.Errors)
			require.Equal(t, timeouts, summary.Timeouts)
			require.Equal(t, unavailable, summary.Unavailable)
			return
		}
	}
	t.Fatalf("missing API latency category %s", category)
}

// requireAPILatencyCategoryClass finds a stage category and verifies its classification is present.
func requireAPILatencyCategoryClass(t *testing.T, stage d.APILatencyStageResult, category d.APILatencyMeasurementCategory, class d.APILatencyClassification) {
	t.Helper()
	for _, summary := range stage.Categories {
		if summary.Category != category {
			continue
		}
		for _, count := range stage.Classifications {
			if count.Classification == class && count.Count > 0 {
				return
			}
		}
		t.Fatalf("missing classification %s for API latency category %s", class, category)
	}
	t.Fatalf("missing API latency category %s", category)
}

// requireAPILatencyVisibility finds visibility evidence for one exact process-instance key.
func requireAPILatencyVisibility(t *testing.T, items []d.APILatencyVisibilityResult, key string) d.APILatencyVisibilityResult {
	t.Helper()
	for _, item := range items {
		if item.ProcessInstanceKey == key {
			return item
		}
	}
	t.Fatalf("missing API latency visibility evidence for %s", key)
	return d.APILatencyVisibilityResult{}
}
