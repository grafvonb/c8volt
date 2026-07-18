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
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// slowProcessAnalysisFixtureTime centralizes timestamp parsing for service analysis fixtures.
func slowProcessAnalysisFixtureTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339Nano, value)
	require.NoError(t, err)
	return parsed
}

// slowProcessAnalysisFixtureProcessInstance returns a minimal process-instance root for timing tests.
func slowProcessAnalysisFixtureProcessInstance(key string, start time.Time, end time.Time) d.ProcessInstance {
	return d.ProcessInstance{
		Key:                    key,
		RootProcessInstanceKey: key,
		ProcessDefinitionKey:   "2251799813687001",
		BpmnProcessId:          "OrderProcess",
		ProcessVersion:         7,
		State:                  d.StateCompleted,
		StartDate:              start.Format(time.RFC3339Nano),
		EndDate:                end.Format(time.RFC3339Nano),
		TenantId:               "tenant-a",
	}
}

// slowProcessAnalysisFixtureElement returns a runtime element row tied to a process instance.
func slowProcessAnalysisFixtureElement(processInstanceKey string, elementInstanceKey string, elementID string, start time.Time, end time.Time) d.Element {
	return d.Element{
		ElementInstanceKey:     elementInstanceKey,
		ElementId:              elementID,
		Type:                   "SERVICE_TASK",
		State:                  "COMPLETED",
		StartDate:              start.Format(time.RFC3339Nano),
		EndDate:                end.Format(time.RFC3339Nano),
		ProcessInstanceKey:     processInstanceKey,
		RootProcessInstanceKey: processInstanceKey,
		ProcessDefinitionKey:   "2251799813687001",
		TenantId:               "tenant-a",
	}
}

// TestSlowProcessAnalysisFixturesBuildConsistentTimingRows protects the shared fixture assumptions.
func TestSlowProcessAnalysisFixturesBuildConsistentTimingRows(t *testing.T) {
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	end := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:05:00Z")

	pi := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, end)
	element := slowProcessAnalysisFixtureElement(pi.Key, "2251799813685250", "ReserveStock", start.Add(time.Second), start.Add(5*time.Second))

	require.Equal(t, pi.Key, pi.RootProcessInstanceKey)
	require.Equal(t, pi.Key, element.ProcessInstanceKey)
	require.Equal(t, pi.ProcessDefinitionKey, element.ProcessDefinitionKey)
	require.Equal(t, "ReserveStock", element.ElementId)
}

// TestSlowProcessAnalysisExplicitKeysDeduplicatesLooksUpAndSortsRoots verifies keyed MVP selection semantics.
func TestSlowProcessAnalysisExplicitKeysDeduplicatesLooksUpAndSortsRoots(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	var lookedUp []string
	instances := map[string]d.ProcessInstance{
		"2251799813685249": slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, start.Add(2*time.Minute)),
		"2251799813685250": func() d.ProcessInstance {
			pi := slowProcessAnalysisFixtureProcessInstance("2251799813685250", start, start.Add(5*time.Minute))
			pi.EndDate = ""
			return pi
		}(),
		"2251799813685251": slowProcessAnalysisFixtureProcessInstance("2251799813685251", start, start.Add(5*time.Minute)),
	}
	piAPI := stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, int32(2), size)
			lookedUp = append(lookedUp, filter.Key)
			if item, ok := instances[filter.Key]; ok {
				return []d.ProcessInstance{item}, nil
			}
			return nil, fmt.Errorf("%w: missing", d.ErrNotFound)
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, nil, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{"2251799813685249", "2251799813685250", "2251799813685249", "2251799813685251"},
		CapturedNow:   captured,
	})

	require.NoError(t, err)
	require.Equal(t, []string{"2251799813685249", "2251799813685250", "2251799813685251"}, lookedUp)
	require.Equal(t, []string{"2251799813685251", "2251799813685249", "2251799813685250"}, []string{got.Items[0].Key, got.Items[1].Key, got.Items[2].Key})
	require.Equal(t, "5m0s", got.Items[0].Duration)
	require.Equal(t, int64(300000), got.Items[0].DurationMillis)
	require.True(t, got.Items[0].DurationAvailable)
	require.False(t, got.Items[2].DurationAvailable)
	require.Equal(t, 3, got.Count)
	require.False(t, got.Empty)
	require.Equal(t, typex.Keys{"2251799813685249", "2251799813685250", "2251799813685251"}, got.Request.InputKeys)
	require.Equal(t, captured, got.CapturedAt)
}

// TestSlowProcessAnalysisExplicitKeysMeasuresActiveFromCapturedNow verifies active roots reuse one analysis timestamp.
func TestSlowProcessAnalysisExplicitKeysMeasuresActiveFromCapturedNow(t *testing.T) {
	captured := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:30:00Z")
	start := slowProcessAnalysisFixtureTime(t, "2026-07-18T10:00:00Z")
	active := slowProcessAnalysisFixtureProcessInstance("2251799813685249", start, time.Time{})
	active.EndDate = ""
	active.State = d.StateActive
	piAPI := stubProcessInstanceAPI{
		search: func(_ context.Context, filter d.ProcessInstanceFilter, _ int32, _ ...services.CallOption) ([]d.ProcessInstance, error) {
			require.Equal(t, "2251799813685249", filter.Key)
			return []d.ProcessInstance{active}, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, nil, toolx.V89).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{"2251799813685249"},
		CapturedNow:   captured,
	})

	require.NoError(t, err)
	require.Equal(t, "30m0s", got.Items[0].Duration)
	require.Equal(t, int64(1800000), got.Items[0].DurationMillis)
	require.True(t, got.Items[0].DurationAvailable)
}

// TestSlowProcessAnalysisExplicitKeysPropagatesLookupFailures verifies missing or unauthorized keys do not become partial success.
func TestSlowProcessAnalysisExplicitKeysPropagatesLookupFailures(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "missing", err: fmt.Errorf("%w: process instance 2251799813685249 not found", d.ErrNotFound)},
		{name: "unauthorized", err: fmt.Errorf("%w: process instance 2251799813685249", d.ErrUnauthorized)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			piAPI := stubProcessInstanceAPI{
				search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
					return nil, tc.err
				},
			}

			got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, nil, toolx.V88).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
				SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
				InputKeys:     typex.Keys{"2251799813685249"},
			})

			require.Error(t, err)
			require.True(t, errors.Is(err, errors.Unwrap(tc.err)))
			require.Empty(t, got.Items)
			require.True(t, got.Empty)
		})
	}
}

// TestSlowProcessAnalysisCamunda87Unsupported verifies version compatibility fails before remote lookup.
func TestSlowProcessAnalysisCamunda87Unsupported(t *testing.T) {
	piAPI := stubProcessInstanceAPI{
		search: func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error) {
			t.Fatal("lookup should not run for unsupported Camunda 8.7")
			return nil, nil
		},
	}

	got, err := NewWithAnalysisDependencies(nil, piAPI, nil, nil, nil, nil, nil, toolx.V87).AnalyseSlowProcessInstances(context.Background(), d.SlowProcessAnalysisRequest{
		SelectionMode: d.SlowProcessAnalysisSelectionModeExplicitKeys,
		InputKeys:     typex.Keys{"2251799813685249"},
	})

	require.ErrorIs(t, err, d.ErrUnsupported)
	require.Empty(t, got.Items)
	require.True(t, got.Empty)
}
