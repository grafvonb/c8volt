// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"errors"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestPurgeAllProcessDefinitionsRecordsControls verifies the foundational request and report model before discovery is implemented.
func TestPurgeAllProcessDefinitionsRecordsControls(t *testing.T) {
	t.Parallel()

	started := time.Date(2026, 5, 16, 17, 30, 0, 0, time.UTC)
	request := d.AllProcessDefinitionsPurgeRequest{
		CommandName:  "ops purge all-process-definitions",
		DryRun:       true,
		AutoConfirm:  true,
		Automation:   true,
		OutputMode:   "json",
		Selection:    d.ProcessDefinitionFilter{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 3, ProcessVersionTag: "stable", IsLatestVersion: true},
		Workers:      2,
		ReportFile:   "all-pds.md",
		ReportFormat: "markdown",
		DiscoveredCandidateProcessDefinitionKeys: typex.Keys{
			"pd-a",
			"pd-a",
			"pd-b",
		},
		StartedAt: started,
	}

	got, err := NewWithProcessDefinitionPurge(
		stubProcessInstanceAPI{},
		nil,
		stubProcessDefinitionAPI{},
		stubResourceAPI{},
	).PurgeAllProcessDefinitions(
		context.Background(),
		request,
		services.WithNoWait(),
		services.WithForce(),
		services.WithFailFast(),
		services.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Equal(t, d.AllProcessDefinitionsPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, started, got.Request.StartedAt)
	require.True(t, got.Request.NoWait)
	require.True(t, got.Request.Force)
	require.True(t, got.Request.FailFast)
	require.True(t, got.Request.NoWorkerLimit)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
	require.Equal(t, request.Selection, got.Discovery.Filters)
	require.Equal(t, []string{"pd-a", "pd-b"}, []string(got.Discovery.CandidateProcessDefinitionKeys))
	require.Equal(t, 2, got.Discovery.CandidateProcessDefinitionCount)
	require.True(t, got.Discovery.LatestOnly)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
	require.Equal(t, []string{"pd-a", "pd-b"}, []string(got.DeletePlan.CandidateProcessDefinitionKeys))
	require.False(t, got.DeletePlan.RequiresForce)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, d.AllProcessDefinitionsPurgeOutcomePlanned, got.Report.Outcome)
	require.True(t, got.Report.NoWait)
	require.True(t, got.Report.Force)
	require.True(t, got.Report.FailFast)
	require.True(t, got.Report.NoWorkerLimit)
	require.Equal(t, request.Selection, got.Report.SelectionFilters)
	require.Equal(t, got.Discovery, got.Report.Discovery)
	require.Empty(t, got.Errors)
}

// TestPurgeAllProcessDefinitionsValidatesServiceDependencies keeps the all-process-definitions service seam explicit.
func TestPurgeAllProcessDefinitionsValidatesServiceDependencies(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		api  API
		want string
	}{
		{
			name: "missing process-instance service",
			api:  NewWithProcessDefinitionPurge(nil, nil, stubProcessDefinitionAPI{}, stubResourceAPI{}),
			want: "process-instance service",
		},
		{
			name: "missing process-definition service",
			api:  NewWithProcessDefinitionPurge(stubProcessInstanceAPI{}, nil, nil, stubResourceAPI{}),
			want: "process-definition service",
		},
		{
			name: "missing resource service",
			api:  NewWithProcessDefinitionPurge(stubProcessInstanceAPI{}, nil, stubProcessDefinitionAPI{}, nil),
			want: "resource service",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := tt.api.PurgeAllProcessDefinitions(context.Background(), d.AllProcessDefinitionsPurgeRequest{})

			require.Error(t, err)
			require.True(t, errors.Is(err, d.ErrValidation), "got %v", err)
			require.Contains(t, err.Error(), tt.want)
			require.Equal(t, d.AllProcessDefinitionsPurgeOutcomeFailed, got.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusFailed, got.Discovery.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
			require.Len(t, got.Errors, 1)
			require.Len(t, got.Discovery.Errors, 1)
			require.Len(t, got.Report.Errors, 1)
		})
	}
}

// TestPurgeAllProcessDefinitionsDiscoversCandidates verifies get-pd-equivalent discovery and frozen candidate extraction.
func TestPurgeAllProcessDefinitionsDiscoversCandidates(t *testing.T) {
	tests := []struct {
		name            string
		request         d.AllProcessDefinitionsPurgeRequest
		api             stubProcessDefinitionAPI
		wantKeys        []string
		wantDuplicates  []string
		wantDefinitions int
		wantNotices     []string
	}{
		{
			name:    "default all-version discovery",
			request: d.AllProcessDefinitionsPurgeRequest{DryRun: true},
			api: stubProcessDefinitionAPI{
				searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, size int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
					require.Equal(t, d.ProcessDefinitionFilter{}, filter)
					require.Equal(t, pdsvc.MaxResultSize, size)
					return []d.ProcessDefinition{
						{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 2},
						{Key: "pd-b", BpmnProcessId: "invoice", ProcessVersion: 1},
					}, nil
				},
			},
			wantKeys:        []string{"pd-a", "pd-b"},
			wantDefinitions: 2,
		},
		{
			name:    "BPMN process ID filtering",
			request: d.AllProcessDefinitionsPurgeRequest{DryRun: true, Selection: d.ProcessDefinitionFilter{BpmnProcessId: "invoice"}},
			api: stubProcessDefinitionAPI{
				searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, size int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
					require.Equal(t, d.ProcessDefinitionFilter{BpmnProcessId: "invoice"}, filter)
					require.Equal(t, pdsvc.MaxResultSize, size)
					return []d.ProcessDefinition{{Key: "pd-invoice", BpmnProcessId: "invoice", ProcessVersion: 1}}, nil
				},
			},
			wantKeys:        []string{"pd-invoice"},
			wantDefinitions: 1,
		},
		{
			name:    "version filtering",
			request: d.AllProcessDefinitionsPurgeRequest{DryRun: true, Selection: d.ProcessDefinitionFilter{ProcessVersion: 3}},
			api: stubProcessDefinitionAPI{
				searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, size int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
					require.Equal(t, d.ProcessDefinitionFilter{ProcessVersion: 3}, filter)
					require.Equal(t, pdsvc.MaxResultSize, size)
					return []d.ProcessDefinition{{Key: "pd-v3", BpmnProcessId: "invoice", ProcessVersion: 3}}, nil
				},
			},
			wantKeys:        []string{"pd-v3"},
			wantDefinitions: 1,
		},
		{
			name:    "version tag filtering",
			request: d.AllProcessDefinitionsPurgeRequest{DryRun: true, Selection: d.ProcessDefinitionFilter{ProcessVersionTag: "stable"}},
			api: stubProcessDefinitionAPI{
				searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, size int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
					require.Equal(t, d.ProcessDefinitionFilter{ProcessVersionTag: "stable"}, filter)
					require.Equal(t, pdsvc.MaxResultSize, size)
					return []d.ProcessDefinition{{Key: "pd-stable", BpmnProcessId: "invoice", ProcessVersionTag: "stable"}}, nil
				},
			},
			wantKeys:        []string{"pd-stable"},
			wantDefinitions: 1,
		},
		{
			name:    "latest-only narrowing",
			request: d.AllProcessDefinitionsPurgeRequest{DryRun: true, Selection: d.ProcessDefinitionFilter{BpmnProcessId: "invoice", IsLatestVersion: true}},
			api: stubProcessDefinitionAPI{
				searchProcessDefinitionsLatest: func(_ context.Context, filter d.ProcessDefinitionFilter, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
					require.Equal(t, d.ProcessDefinitionFilter{BpmnProcessId: "invoice", IsLatestVersion: true}, filter)
					return []d.ProcessDefinition{{Key: "pd-latest", BpmnProcessId: "invoice", ProcessVersion: 4}}, nil
				},
			},
			wantKeys:        []string{"pd-latest"},
			wantDefinitions: 1,
			wantNotices:     []string{"latest_only_scope"},
		},
		{
			name:    "duplicate detection",
			request: d.AllProcessDefinitionsPurgeRequest{DryRun: true},
			api: stubProcessDefinitionAPI{
				searchProcessDefinitions: func(_ context.Context, _ d.ProcessDefinitionFilter, _ int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
					return []d.ProcessDefinition{
						{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 2},
						{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 2},
						{Key: "pd-b", BpmnProcessId: "payment", ProcessVersion: 1},
					}, nil
				},
			},
			wantKeys:        []string{"pd-a", "pd-b"},
			wantDuplicates:  []string{"pd-a"},
			wantDefinitions: 2,
			wantNotices:     []string{"duplicate_candidate_process_definitions"},
		},
		{
			name:    "no target behavior",
			request: d.AllProcessDefinitionsPurgeRequest{DryRun: true, Selection: d.ProcessDefinitionFilter{BpmnProcessId: "missing"}},
			api: stubProcessDefinitionAPI{
				searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, _ int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
					require.Equal(t, d.ProcessDefinitionFilter{BpmnProcessId: "missing"}, filter)
					return nil, nil
				},
			},
			wantNotices: []string{"no_candidate_process_definitions"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewWithProcessDefinitionPurge(
				stubProcessInstanceAPI{},
				nil,
				tt.api,
				stubResourceAPI{},
			).PurgeAllProcessDefinitions(context.Background(), tt.request)

			require.NoError(t, err)
			require.Equal(t, d.AllProcessDefinitionsPurgeOutcomePlanned, got.Outcome)
			require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.Discovery.Status)
			require.Equal(t, tt.request.Selection, got.Discovery.Filters)
			require.Equal(t, tt.request.Selection.IsLatestVersion, got.Discovery.LatestOnly)
			require.Equal(t, emptyStringSliceIfNil(tt.wantKeys), emptyStringSliceIfNil([]string(got.Discovery.CandidateProcessDefinitionKeys)))
			require.Equal(t, len(tt.wantKeys), got.Discovery.CandidateProcessDefinitionCount)
			require.Equal(t, emptyStringSliceIfNil(tt.wantDuplicates), emptyStringSliceIfNil([]string(got.Discovery.DuplicateCandidateProcessDefinitionKeys)))
			require.Len(t, got.Discovery.CandidateProcessDefinitions, tt.wantDefinitions)
			if len(tt.wantKeys) == 0 {
				require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.DeletePlan.Status)
			} else {
				require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
			}
			require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
			require.Equal(t, got.Discovery, got.Report.Discovery)
			require.Equal(t, got.DeletePlan, got.Report.DeletePlan)
			requireNoticeCodes(t, got.Discovery.Notices, tt.wantNotices)
		})
	}
}

// TestPurgeAllProcessDefinitionsDiscoversSingleKey verifies key selection follows get-pd lookup semantics.
func TestPurgeAllProcessDefinitionsDiscoversSingleKey(t *testing.T) {
	t.Parallel()

	got, err := NewWithProcessDefinitionPurge(
		stubProcessInstanceAPI{},
		nil,
		stubProcessDefinitionAPI{
			getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
				require.Equal(t, "2251799813685255", key)
				if services.ApplyCallOptions(opts).WithStat {
					return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
				}
				return d.ProcessDefinition{Key: key, BpmnProcessId: "invoice", ProcessVersion: 1}, nil
			},
		},
		stubResourceAPI{},
	).PurgeAllProcessDefinitions(context.Background(), d.AllProcessDefinitionsPurgeRequest{
		DryRun:    true,
		Selection: d.ProcessDefinitionFilter{Key: "2251799813685255", BpmnProcessId: "ignored-by-get-pd"},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"2251799813685255"}, []string(got.Discovery.CandidateProcessDefinitionKeys))
	require.Equal(t, "invoice", got.Discovery.CandidateProcessDefinitions[0].BpmnProcessId)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
	require.Equal(t, []string{"2251799813685255"}, []string(got.DeletePlan.CandidateProcessDefinitionKeys))
}

// TestPurgeAllProcessDefinitionsBuildsDeletePlan verifies frozen candidates flow through the delete-pd preflight.
func TestPurgeAllProcessDefinitionsBuildsDeletePlan(t *testing.T) {
	t.Parallel()

	got, err := NewWithProcessDefinitionPurge(
		stubProcessInstanceAPI{},
		nil,
		stubProcessDefinitionAPI{
			searchProcessDefinitions: func(_ context.Context, _ d.ProcessDefinitionFilter, _ int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
				return []d.ProcessDefinition{
					{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 2},
					{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 2},
					{Key: "pd-b", BpmnProcessId: "payment", ProcessVersion: 1},
				}, nil
			},
			getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
				require.True(t, services.ApplyCallOptions(opts).WithStat)
				stats := map[string]int64{"pd-a": 2, "pd-b": 0}
				return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{Active: stats[key]}}, nil
			},
		},
		stubResourceAPI{},
	).PurgeAllProcessDefinitions(context.Background(), d.AllProcessDefinitionsPurgeRequest{DryRun: true})

	require.NoError(t, err)
	require.Equal(t, d.AllProcessDefinitionsPurgeOutcomePlanned, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
	require.Equal(t, []string{"pd-a", "pd-b"}, []string(got.DeletePlan.CandidateProcessDefinitionKeys))
	require.Equal(t, []string{"pd-a"}, []string(got.DeletePlan.DuplicateCandidateProcessDefinitionKeys))
	require.Len(t, got.DeletePlan.Items, 2)
	require.Equal(t, "pd-a", got.DeletePlan.Items[0].Key)
	require.EqualValues(t, 2, got.DeletePlan.Items[0].ActiveProcessInstanceCount)
	require.EqualValues(t, 2, got.DeletePlan.AffectedProcessInstanceCount)
	require.EqualValues(t, 2, got.DeletePlan.ActiveProcessInstanceCount)
	require.False(t, got.DeletePlan.RequiresConfirmation)
	require.True(t, got.DeletePlan.RequiresForce)
	require.Equal(t, d.OpsWorkflowStepStatusSkipped, got.Deletion.Status)
	require.Equal(t, got.DeletePlan, got.Report.DeletePlan)
}

// TestPurgeAllProcessDefinitionsBlocksUnsafeActiveInstancesWithoutForce verifies destructive planning stops before mutation.
func TestPurgeAllProcessDefinitionsBlocksUnsafeActiveInstancesWithoutForce(t *testing.T) {
	t.Parallel()

	got, err := NewWithProcessDefinitionPurge(
		stubProcessInstanceAPI{},
		nil,
		stubProcessDefinitionAPI{
			searchProcessDefinitions: func(_ context.Context, _ d.ProcessDefinitionFilter, _ int32, _ ...services.CallOption) ([]d.ProcessDefinition, error) {
				return []d.ProcessDefinition{{Key: "pd-active", BpmnProcessId: "invoice", ProcessVersion: 1}}, nil
			},
			getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
				require.Equal(t, "pd-active", key)
				require.True(t, services.ApplyCallOptions(opts).WithStat)
				return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{Active: 3}}, nil
			},
		},
		stubResourceAPI{},
	).PurgeAllProcessDefinitions(context.Background(), d.AllProcessDefinitionsPurgeRequest{})

	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrPrecondition), "got %v", err)
	require.Contains(t, err.Error(), "active process instance")
	require.Equal(t, d.AllProcessDefinitionsPurgeOutcomeFailed, got.Outcome)
	require.Equal(t, d.OpsWorkflowStepStatusPlanned, got.DeletePlan.Status)
	require.EqualValues(t, 3, got.DeletePlan.ActiveProcessInstanceCount)
	require.True(t, got.DeletePlan.RequiresConfirmation)
	require.True(t, got.DeletePlan.RequiresForce)
	require.Equal(t, d.OpsWorkflowStepStatusBlocked, got.Deletion.Status)
	require.Len(t, got.Deletion.Errors, 1)
	require.Len(t, got.Report.Errors, 1)
}

// emptyStringSliceIfNil lets tests compare logical empty collections independent of nil slice representation.
func emptyStringSliceIfNil(items []string) []string {
	if items == nil {
		return []string{}
	}
	return items
}

// requireNoticeCodes compares semantic discovery notice codes without binding tests to message text.
func requireNoticeCodes(t *testing.T, notices []d.AllProcessDefinitionsPurgeWorkflowNotice, want []string) {
	t.Helper()

	got := make([]string, 0, len(notices))
	for _, notice := range notices {
		got = append(got, notice.Code)
	}
	require.Equal(t, emptyStringSliceIfNil(want), emptyStringSliceIfNil(got))
}

type stubProcessDefinitionAPI struct {
	pdsvc.API
	searchProcessDefinitions       func(context.Context, d.ProcessDefinitionFilter, int32, ...services.CallOption) ([]d.ProcessDefinition, error)
	searchProcessDefinitionsLatest func(context.Context, d.ProcessDefinitionFilter, ...services.CallOption) ([]d.ProcessDefinition, error)
	getProcessDefinition           func(context.Context, string, ...services.CallOption) (d.ProcessDefinition, error)
}

type stubResourceAPI struct {
	rsvc.API
}

// SearchProcessDefinitions delegates to the per-test process-definition search callback.
func (s stubProcessDefinitionAPI) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	if s.searchProcessDefinitions == nil {
		panic("unexpected SearchProcessDefinitions call")
	}
	return s.searchProcessDefinitions(ctx, filter, size, opts...)
}

// SearchProcessDefinitionsLatest delegates to the per-test latest process-definition search callback.
func (s stubProcessDefinitionAPI) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	if s.searchProcessDefinitionsLatest == nil {
		panic("unexpected SearchProcessDefinitionsLatest call")
	}
	return s.searchProcessDefinitionsLatest(ctx, filter, opts...)
}

// GetProcessDefinition delegates to the per-test single process-definition lookup callback.
func (s stubProcessDefinitionAPI) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	if s.getProcessDefinition != nil {
		return s.getProcessDefinition(ctx, key, opts...)
	}
	if services.ApplyCallOptions(opts).WithStat {
		return d.ProcessDefinition{Key: key, Statistics: &d.ProcessDefinitionStatistics{}}, nil
	}
	panic("unexpected GetProcessDefinition call")
}
