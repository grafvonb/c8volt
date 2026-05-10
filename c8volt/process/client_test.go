// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_GetProcessDefinitionXML verifies facade option translation for
// XML retrieval. The key and call options must pass through unchanged because
// the service layer owns the Camunda-specific request details.
func TestClient_GetProcessDefinitionXML(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		getProcessDefinitionXML: func(_ context.Context, key string, opts ...services.CallOption) (string, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, "2251799813685255", key)
			assert.True(t, cfg.Verbose)
			assert.True(t, cfg.WithStat)
			return "<definitions id=\"order-process\"/>", nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, stubIncidentAPI{}, slog.Default())
	xml, err := cli.GetProcessDefinitionXML(ctx, "2251799813685255", options.WithVerbose(), options.WithStat())

	require.NoError(t, err)
	assert.Equal(t, "<definitions id=\"order-process\"/>", xml)
}

// TestClient_GetProcessDefinition_MapsIncidentCountSupportState protects the
// distinction between an incident count of zero and a version where incident
// counts are actually supported.
func TestClient_GetProcessDefinition_MapsIncidentCountSupportState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		getProcessDefinition: func(_ context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, "2251799813685255", key)
			assert.True(t, cfg.WithStat)
			return d.ProcessDefinition{
				Key: "2251799813685255",
				Statistics: &d.ProcessDefinitionStatistics{
					Active:                 7,
					Canceled:               3,
					Completed:              11,
					Incidents:              0,
					IncidentCountSupported: true,
				},
			}, nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, stubIncidentAPI{}, slog.Default())
	pd, err := cli.GetProcessDefinition(ctx, "2251799813685255", options.WithStat())

	require.NoError(t, err)
	require.NotNil(t, pd.Statistics)
	assert.Equal(t, int64(7), pd.Statistics.Active)
	assert.Equal(t, int64(0), pd.Statistics.Incidents)
	assert.True(t, pd.Statistics.IncidentCountSupported)
}

// TestClient_SearchProcessDefinitions_PreservesUnsupportedIncidentCountBoundary
// keeps the public model from implying that zero incidents were measured when a
// Camunda version cannot provide incident-count statistics.
func TestClient_SearchProcessDefinitions_PreservesUnsupportedIncidentCountBoundary(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, d.ProcessDefinitionFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, pdsvc.MaxResultSize, size)
			assert.True(t, cfg.WithStat)
			return []d.ProcessDefinition{
				{
					Key:           "2251799813685255",
					BpmnProcessId: "order-process",
					Statistics: &d.ProcessDefinitionStatistics{
						Active:                 7,
						Canceled:               3,
						Completed:              11,
						Incidents:              0,
						IncidentCountSupported: false,
					},
				},
			}, nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, stubIncidentAPI{}, slog.Default())
	items, err := cli.SearchProcessDefinitions(ctx, ProcessDefinitionFilter{BpmnProcessId: "order-process"}, options.WithStat())

	require.NoError(t, err)
	require.Len(t, items.Items, 1)
	require.NotNil(t, items.Items[0].Statistics)
	assert.Equal(t, int64(7), items.Items[0].Statistics.Active)
	assert.Equal(t, int64(0), items.Items[0].Statistics.Incidents)
	assert.False(t, items.Items[0].Statistics.IncidentCountSupported)
}

// TestClient_SearchProcessDefinitions_MapsProcessDefinitionSelectorFilter
// verifies BPMN process ID, exact version, and version tag selectors reach the
// process-definition service unchanged.
func TestClient_SearchProcessDefinitions_MapsProcessDefinitionSelectorFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		searchProcessDefinitions: func(_ context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
			assert.Equal(t, d.ProcessDefinitionFilter{
				BpmnProcessId:     "order-process",
				ProcessVersion:    7,
				ProcessVersionTag: "stable",
			}, filter)
			assert.Equal(t, pdsvc.MaxResultSize, size)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessDefinition{{Key: "2251799813685255", BpmnProcessId: "order-process"}}, nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, stubIncidentAPI{}, slog.Default())
	items, err := cli.SearchProcessDefinitions(ctx, ProcessDefinitionFilter{
		BpmnProcessId:     "order-process",
		ProcessVersion:    7,
		ProcessVersionTag: "stable",
	}, options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, items.Items, 1)
	assert.Equal(t, "order-process", items.Items[0].BpmnProcessId)
}

// TestClient_SearchProcessDefinitionsLatest_MapsProcessDefinitionSelectorFilter
// extends facade coverage to latest-definition searches used by BPMN starts.
func TestClient_SearchProcessDefinitionsLatest_MapsProcessDefinitionSelectorFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pdAPI := &stubProcessDefinitionAPI{
		searchProcessDefinitionsLatest: func(_ context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
			assert.Equal(t, d.ProcessDefinitionFilter{
				BpmnProcessId:     "order-process",
				ProcessVersionTag: "stable",
			}, filter)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessDefinition{{Key: "2251799813685255", BpmnProcessId: "order-process"}}, nil
		},
	}

	cli := New(pdAPI, stubProcessInstanceAPI{}, stubIncidentAPI{}, slog.Default())
	items, err := cli.SearchProcessDefinitionsLatest(ctx, ProcessDefinitionFilter{
		BpmnProcessId:     "order-process",
		ProcessVersionTag: "stable",
	}, options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, items.Items, 1)
	assert.Equal(t, "order-process", items.Items[0].BpmnProcessId)
}

// TestClient_SearchProcessInstances_MapsDateBoundsToDomainFilter verifies that
// facade filters, presence booleans, and verbose options reach the domain layer
// without losing exact date-bound strings.
func TestClient_SearchProcessInstances_MapsDateBoundsToDomainFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hasParent := new(false)
	hasIncident := new(true)
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, int32(25), size)
			assert.Equal(t, d.ProcessInstanceFilter{
				BpmnProcessId:        "order-process",
				ProcessDefinitionKey: "2251799813685255",
				StartDateAfter:       "2026-01-01",
				StartDateBefore:      "2026-01-31",
				EndDateAfter:         "2026-02-01",
				EndDateBefore:        "2026-02-28",
				State:                d.StateCompleted,
				ParentKey:            "12345",
				HasParent:            hasParent,
				HasIncident:          hasIncident,
			}, filter)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstance{}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	_, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		BpmnProcessId:        "order-process",
		ProcessDefinitionKey: "2251799813685255",
		StartDateAfter:       "2026-01-01",
		StartDateBefore:      "2026-01-31",
		EndDateAfter:         "2026-02-01",
		EndDateBefore:        "2026-02-28",
		State:                StateCompleted,
		ParentKey:            "12345",
		HasParent:            hasParent,
		HasIncident:          hasIncident,
	}, 25, options.WithVerbose())

	require.NoError(t, err)
}

// TestClient_SearchProcessInstances_PreservesDerivedRelativeDayBoundsAsCanonicalDateFields
// documents that relative-day CLI handling happens before this facade call.
// Once dates arrive here they are canonical absolute strings and must not be
// recomputed or interpreted again.
func TestClient_SearchProcessInstances_PreservesDerivedRelativeDayBoundsAsCanonicalDateFields(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, int32(10), size)
			assert.Equal(t, d.ProcessInstanceFilter{
				StartDateAfter:  "2026-03-11",
				StartDateBefore: "2026-04-03",
				EndDateAfter:    "2026-02-09",
				EndDateBefore:   "2026-03-27",
			}, filter)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstance{}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	_, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		StartDateAfter:  "2026-03-11",
		StartDateBefore: "2026-04-03",
		EndDateAfter:    "2026-02-09",
		EndDateBefore:   "2026-03-27",
	}, 10, options.WithVerbose())

	require.NoError(t, err)
}

// TestClient_EnrichProcessInstancesWithIncidents_PreservesOrderAndPerKeyAssociation prevents incident details from leaking across keyed results.
func TestClient_EnrichProcessInstancesWithIncidents_PreservesOrderAndPerKeyAssociation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls []string
	incAPI := stubIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			calls = append(calls, key)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			switch key {
			case "123":
				return []d.ProcessInstanceIncidentDetail{
					{IncidentKey: "i-123", ProcessInstanceKey: "123", ErrorMessage: "first"},
					{IncidentKey: "wrong", ProcessInstanceKey: "999", ErrorMessage: "wrong key"},
				}, nil
			case "124":
				return nil, nil
			default:
				t.Fatalf("unexpected incident lookup for key %s", key)
				return nil, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, stubProcessInstanceAPI{}, incAPI, slog.Default())
	got, err := cli.EnrichProcessInstancesWithIncidents(ctx, ProcessInstances{
		Total: 2,
		Items: []ProcessInstance{
			{Key: "123", BpmnProcessId: "order-process"},
			{Key: "124", BpmnProcessId: "invoice-process"},
		},
	}, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, []string{"123", "124"}, calls)
	require.Equal(t, int32(2), got.Total)
	require.Len(t, got.Items, 2)
	require.Equal(t, "123", got.Items[0].Item.Key)
	require.Equal(t, []ProcessInstanceIncidentDetail{
		{IncidentKey: "i-123", ProcessInstanceKey: "123", ErrorMessage: "first"},
	}, got.Items[0].Incidents)
	require.Equal(t, "124", got.Items[1].Item.Key)
	require.Empty(t, got.Items[1].Incidents)
	require.NotNil(t, got.Items[1].Incidents)
}

func TestClient_EnrichProcessInstancesWithVariables_PreservesOrderAndPerKeyAssociation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls []string
	piAPI := stubProcessInstanceAPI{
		searchProcessInstanceVariables: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			calls = append(calls, key)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			switch key {
			case "123":
				return []d.ProcessInstanceVariable{
					{Name: "zeta", Value: "2", VariableKey: "v-2", ProcessInstanceKey: "123", ScopeKey: "123"},
					{Name: "elementLocal", Value: "ignored", VariableKey: "v-local", ProcessInstanceKey: "123", ScopeKey: "element-1"},
					{Name: "wrongOwner", Value: "ignored", VariableKey: "v-wrong", ProcessInstanceKey: "999", ScopeKey: "999"},
					{Name: "alpha", Value: "1", VariableKey: "v-1", ProcessInstanceKey: "123", ScopeKey: "123"},
				}, nil
			case "124":
				return nil, nil
			default:
				t.Fatalf("unexpected variable lookup for key %s", key)
				return nil, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.EnrichProcessInstancesWithVariables(ctx, ProcessInstances{
		Total: 2,
		Items: []ProcessInstance{
			{Key: "123", BpmnProcessId: "order-process"},
			{Key: "124", BpmnProcessId: "invoice-process"},
		},
	}, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, []string{"123", "124"}, calls)
	require.Equal(t, int32(2), got.Total)
	require.Len(t, got.Items, 2)
	require.Equal(t, "123", got.Items[0].Item.Key)
	require.Equal(t, []ProcessInstanceVariable{
		{Name: "alpha", Value: "1", VariableKey: "v-1", ProcessInstanceKey: "123", ScopeKey: "123"},
		{Name: "zeta", Value: "2", VariableKey: "v-2", ProcessInstanceKey: "123", ScopeKey: "123"},
	}, got.Items[0].Variables)
	require.Equal(t, "124", got.Items[1].Item.Key)
	require.Empty(t, got.Items[1].Variables)
	require.NotNil(t, got.Items[1].Variables)
}

func TestClient_EnrichProcessInstancesWithVariables_SortsVariablesAndPreservesJSONMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchProcessInstanceVariables: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			require.Equal(t, "123", key)
			return []d.ProcessInstanceVariable{
				{Name: "zeta", Value: "2", VariableKey: "v-2", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant", APITruncated: false},
				{Name: "alpha", Value: `"C-123"`, VariableKey: "v-1", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant", APITruncated: true},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.EnrichProcessInstancesWithVariables(ctx, ProcessInstances{
		Total: 1,
		Items: []ProcessInstance{{Key: "123", BpmnProcessId: "order-process"}},
	})

	require.NoError(t, err)
	require.Equal(t, []ProcessInstanceVariable{
		{Name: "alpha", Value: `"C-123"`, VariableKey: "v-1", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant", APITruncated: true},
		{Name: "zeta", Value: "2", VariableKey: "v-2", ProcessInstanceKey: "123", ScopeKey: "123", TenantId: "tenant", APITruncated: false},
	}, got.Items[0].Variables)
}

func TestUpdateProcessInstanceVariablesMapsConfirmedServiceResponse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		updateProcessInstanceVariables: func(_ context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
			require.Equal(t, "123", key)
			require.Equal(t, map[string]any{
				"foo":    "bar",
				"nested": map[string]any{"count": float64(2)},
			}, variables)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstanceVariableUpdateResponse{Key: key, Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.UpdateProcessInstanceVariables(ctx, ProcessInstanceVariableUpdateRequest{
		Key: "123",
		Variables: map[string]any{
			"foo":    "bar",
			"nested": map[string]any{"count": float64(2)},
		},
	}, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, ProcessInstanceVariableUpdateResult{
		Key:                "123",
		Status:             ProcessInstanceVariableUpdateStatusConfirmed,
		MutationAccepted:   true,
		ConfirmationStatus: "confirmed",
		StatusCode:         204,
		Message:            "204 No Content",
		Variables: map[string]any{
			"foo":    "bar",
			"nested": map[string]any{"count": float64(2)},
		},
	}, got)
}

func TestUpdateProcessInstanceVariablesMultipleKeysRespectWorkersAndFailFastOptions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var active int32
	var maxActive int32
	var updates int32
	seen := make(chan string, 3)
	piAPI := stubProcessInstanceAPI{
		updateProcessInstanceVariables: func(_ context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
			cfg := services.ApplyCallOptions(opts)
			if !cfg.FailFast {
				return d.ProcessInstanceVariableUpdateResponse{}, errors.New("expected fail-fast call option")
			}
			if !cfg.NoWorkerLimit {
				return d.ProcessInstanceVariableUpdateResponse{}, errors.New("expected no-worker-limit call option")
			}
			if variables["foo"] != "bar" || len(variables) != 1 {
				return d.ProcessInstanceVariableUpdateResponse{}, errors.New("unexpected variables payload")
			}

			current := atomic.AddInt32(&active, 1)
			for {
				previous := atomic.LoadInt32(&maxActive)
				if current <= previous || atomic.CompareAndSwapInt32(&maxActive, previous, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			atomic.AddInt32(&active, -1)
			atomic.AddInt32(&updates, 1)
			seen <- key
			return d.ProcessInstanceVariableUpdateResponse{Key: key, Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
		},
		searchProcessInstanceVariables: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
			cfg := services.ApplyCallOptions(opts)
			if !cfg.FailFast {
				return nil, errors.New("expected fail-fast call option")
			}
			if !cfg.NoWorkerLimit {
				return nil, errors.New("expected no-worker-limit call option")
			}
			return []d.ProcessInstanceVariable{{Name: "foo", Value: `"bar"`, ProcessInstanceKey: key, ScopeKey: key}}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.UpdateProcessInstancesVariables(ctx,
		typex.Keys{"2251799813711967", "2251799813711968", "2251799813711967", "2251799813711969"},
		map[string]any{"foo": "bar"},
		2,
		options.WithFailFast(),
		options.WithNoWorkerLimit(),
	)

	require.NoError(t, err)
	require.Len(t, got.Items, 3)
	require.Equal(t, int32(3), atomic.LoadInt32(&updates))
	require.LessOrEqual(t, atomic.LoadInt32(&maxActive), int32(2))
	close(seen)
	require.ElementsMatch(t, []string{"2251799813711967", "2251799813711968", "2251799813711969"}, drainStringChannel(seen))
	for _, item := range got.Items {
		require.Equal(t, ProcessInstanceVariableUpdateStatusConfirmed, item.Status)
		require.True(t, item.MutationAccepted)
		require.Equal(t, "confirmed", item.ConfirmationStatus)
	}
}

func TestUpdateProcessInstanceVariablesNoWaitReportsMutationFailurePerKey(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		updateProcessInstanceVariables: func(_ context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
			require.Equal(t, "123", key)
			require.Equal(t, map[string]any{"foo": "bar"}, variables)
			require.True(t, services.ApplyCallOptions(opts).NoWait)
			return d.ProcessInstanceVariableUpdateResponse{
				Key:        key,
				Ok:         false,
				StatusCode: 500,
				Status:     "500 Internal Server Error",
			}, errors.New("mutation rejected")
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.UpdateProcessInstancesVariables(ctx,
		typex.Keys{"123"},
		map[string]any{"foo": "bar"},
		1,
		options.WithNoWait(),
	)

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	require.Equal(t, ProcessInstanceVariableUpdateResult{
		Key:                "123",
		Status:             ProcessInstanceVariableUpdateStatusMutationFailed,
		MutationAccepted:   false,
		ConfirmationStatus: "skipped",
		StatusCode:         500,
		Message:            "500 Internal Server Error",
		Error:              "mutation rejected",
		Variables:          map[string]any{"foo": "bar"},
	}, got.Items[0])
	require.False(t, got.Items[0].OK())
}

func TestUpdateProcessInstanceVariablesConfirmationTimeoutReportsPerKeyFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		updateProcessInstanceVariables: func(_ context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
			require.Equal(t, "123", key)
			require.Equal(t, map[string]any{"foo": "bar"}, variables)
			return d.ProcessInstanceVariableUpdateResponse{Key: key, Ok: true, StatusCode: 204, Status: "204 No Content"}, context.DeadlineExceeded
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.UpdateProcessInstanceVariables(ctx, ProcessInstanceVariableUpdateRequest{
		Key:       "123",
		Variables: map[string]any{"foo": "bar"},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "operation timed out")
	require.Contains(t, err.Error(), context.DeadlineExceeded.Error())
	require.Equal(t, ProcessInstanceVariableUpdateResult{
		Key:                "123",
		Status:             ProcessInstanceVariableUpdateStatusConfirmationFailed,
		MutationAccepted:   true,
		ConfirmationStatus: "failed",
		StatusCode:         204,
		Message:            "204 No Content",
		Error:              "operation timed out: context deadline exceeded",
		Variables:          map[string]any{"foo": "bar"},
	}, got)
	require.False(t, got.OK())
}

func drainStringChannel(ch <-chan string) []string {
	out := make([]string, 0)
	for s := range ch {
		out = append(out, s)
	}
	return out
}

// TestClient_EnrichTraversalWithIncidents_PreservesTraversalMetadataAndPerKeyAssociation keeps walk metadata stable while adding incidents per walked key.
func TestClient_EnrichTraversalWithIncidents_PreservesTraversalMetadataAndPerKeyAssociation(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls []string
	incAPI := stubIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			calls = append(calls, key)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			switch key {
			case "root":
				return nil, nil
			case "child":
				return []d.ProcessInstanceIncidentDetail{
					{IncidentKey: "incident-child", ProcessInstanceKey: "child", ErrorMessage: "child failed"},
					{IncidentKey: "incident-other", ProcessInstanceKey: "other", ErrorMessage: "not walked"},
				}, nil
			default:
				t.Fatalf("unexpected incident lookup for key %s", key)
				return nil, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, stubProcessInstanceAPI{}, incAPI, slog.Default())
	got, err := cli.EnrichTraversalWithIncidents(ctx, TraversalResult{
		Mode:     TraversalModeDescendants,
		Outcome:  TraversalOutcomePartial,
		StartKey: "root",
		RootKey:  "root",
		Keys:     []string{"root", "child"},
		Edges:    map[string][]string{"root": {"child"}},
		Chain: map[string]ProcessInstance{
			"root":  {Key: "root", BpmnProcessId: "order-process"},
			"child": {Key: "child", BpmnProcessId: "invoice-process"},
		},
		MissingAncestors: []MissingAncestor{{Key: "missing", StartKey: "child"}},
		Warning:          "one or more parent process instances were not found",
	}, options.WithVerbose())

	require.NoError(t, err)
	require.Equal(t, []string{"root", "child"}, calls)
	require.Equal(t, TraversalModeDescendants, got.Mode)
	require.Equal(t, TraversalOutcomePartial, got.Outcome)
	require.Equal(t, "root", got.StartKey)
	require.Equal(t, "root", got.RootKey)
	require.Equal(t, []string{"root", "child"}, got.Keys)
	require.Equal(t, map[string][]string{"root": {"child"}}, got.Edges)
	require.Equal(t, []MissingAncestor{{Key: "missing", StartKey: "child"}}, got.MissingAncestors)
	require.Equal(t, "one or more parent process instances were not found", got.Warning)
	require.Len(t, got.Items, 2)
	require.Equal(t, "root", got.Items[0].Item.Key)
	require.Empty(t, got.Items[0].Incidents)
	require.NotNil(t, got.Items[0].Incidents)
	require.Equal(t, "child", got.Items[1].Item.Key)
	require.Equal(t, []ProcessInstanceIncidentDetail{
		{IncidentKey: "incident-child", ProcessInstanceKey: "child", ErrorMessage: "child failed"},
	}, got.Items[1].Incidents)
}

// TestClient_EnrichTraversalWithIncidents_PassesConfiguredOptionsToIncidentLookup verifies tenant and verbosity options flow into every enrichment lookup.
func TestClient_EnrichTraversalWithIncidents_PassesConfiguredOptionsToIncidentLookup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls []string
	incAPI := stubIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			calls = append(calls, key)
			cfg := services.ApplyCallOptions(opts)
			assert.True(t, cfg.Verbose)
			assert.True(t, cfg.WithStat)
			return nil, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, stubProcessInstanceAPI{}, incAPI, slog.Default())
	got, err := cli.EnrichTraversalWithIncidents(ctx, TraversalResult{
		Mode:    TraversalModeDescendants,
		Outcome: TraversalOutcomeComplete,
		Keys:    []string{"root", "child"},
		Chain: map[string]ProcessInstance{
			"root":  {Key: "root"},
			"child": {Key: "child"},
		},
	}, options.WithVerbose(), options.WithStat())

	require.NoError(t, err)
	require.Equal(t, []string{"root", "child"}, calls)
	require.Len(t, got.Items, 2)
}

// TestClient_EnrichTraversalWithIncidents_LooksUpOnlyTraversalResultKeys avoids querying incidents for chain entries outside the selected walk path.
func TestClient_EnrichTraversalWithIncidents_LooksUpOnlyTraversalResultKeys(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var calls []string
	incAPI := stubIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			calls = append(calls, key)
			switch key {
			case "root", "walked":
				return nil, nil
			default:
				t.Fatalf("unexpected incident lookup for key %s", key)
				return nil, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, stubProcessInstanceAPI{}, incAPI, slog.Default())
	got, err := cli.EnrichTraversalWithIncidents(ctx, TraversalResult{
		Mode:    TraversalModeDescendants,
		Outcome: TraversalOutcomeComplete,
		Keys:    []string{"root", "missing-chain", "walked"},
		Chain: map[string]ProcessInstance{
			"root":        {Key: "root"},
			"walked":      {Key: "walked"},
			"chain-extra": {Key: "chain-extra"},
		},
	})

	require.NoError(t, err)
	require.Equal(t, []string{"root", "walked"}, calls)
	require.Len(t, got.Items, 2)
	require.Equal(t, "root", got.Items[0].Item.Key)
	require.Equal(t, "walked", got.Items[1].Item.Key)
}

// TestClient_EnrichTraversalWithIncidents_PropagatesIncidentLookupFailure prevents rendering partially enriched traversal output after lookup errors.
func TestClient_EnrichTraversalWithIncidents_PropagatesIncidentLookupFailure(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	lookupErr := errors.New("incident lookup failed")
	var calls []string
	incAPI := stubIncidentAPI{
		searchProcessInstanceIncidents: func(_ context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
			calls = append(calls, key)
			if key == "child" {
				return nil, lookupErr
			}
			return nil, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, stubProcessInstanceAPI{}, incAPI, slog.Default())
	got, err := cli.EnrichTraversalWithIncidents(ctx, TraversalResult{
		Mode:    TraversalModeDescendants,
		Outcome: TraversalOutcomeComplete,
		Keys:    []string{"root", "child"},
		Chain: map[string]ProcessInstance{
			"root":  {Key: "root"},
			"child": {Key: "child"},
		},
	})

	require.ErrorIs(t, err, lookupErr)
	require.Equal(t, []string{"root", "child"}, calls)
	require.Empty(t, got.Items)
}

func TestClient_WaitForProcessInstancesExpectation_MapsIncidentTrueRequestAndReports(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wantIncident := IncidentExpectationTrue
	domainIncident := true
	piAPI := stubProcessInstanceAPI{
		waitForProcessInstancesExpectation: func(_ context.Context, keys typex.Keys, request d.ProcessInstanceExpectationRequest, workers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
			cfg := services.ApplyCallOptions(opts)
			require.Equal(t, typex.Keys{"123"}, keys)
			require.Empty(t, request.States)
			require.NotNil(t, request.Incident)
			require.True(t, *request.Incident)
			require.Equal(t, 1, workers)
			require.True(t, cfg.Verbose)
			return d.ProcessInstanceExpectationResponses{
				Items: []d.ProcessInstanceExpectationResponse{
					{Key: "123", Ok: true, State: d.StateActive, Incident: &domainIncident, Status: "process instance 123 satisfied expectation(s)"},
				},
			}, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())

	got, err := cli.WaitForProcessInstancesExpectation(ctx, typex.Keys{"123", "123"}, ProcessInstanceExpectationRequest{Incident: &wantIncident}, 1, options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, ProcessInstanceExpectationReport{Key: "123", Ok: true, State: StateActive, Incident: &wantIncident, Status: "process instance 123 satisfied expectation(s)"}, got.Items[0])
}

func TestClient_WaitForProcessInstancesExpectation_MapsIncidentFalseRequestAndReports(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wantIncident := IncidentExpectationFalse
	domainIncident := false
	piAPI := stubProcessInstanceAPI{
		waitForProcessInstancesExpectation: func(_ context.Context, keys typex.Keys, request d.ProcessInstanceExpectationRequest, workers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
			cfg := services.ApplyCallOptions(opts)
			require.Equal(t, typex.Keys{"123"}, keys)
			require.Empty(t, request.States)
			require.NotNil(t, request.Incident)
			require.False(t, *request.Incident)
			require.Equal(t, 1, workers)
			require.True(t, cfg.Verbose)
			return d.ProcessInstanceExpectationResponses{
				Items: []d.ProcessInstanceExpectationResponse{
					{Key: "123", Ok: true, State: d.StateActive, Incident: &domainIncident, Status: "process instance 123 satisfied expectation(s)"},
				},
			}, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())

	got, err := cli.WaitForProcessInstancesExpectation(ctx, typex.Keys{"123", "123"}, ProcessInstanceExpectationRequest{Incident: &wantIncident}, 1, options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, ProcessInstanceExpectationReport{Key: "123", Ok: true, State: StateActive, Incident: &wantIncident, Status: "process instance 123 satisfied expectation(s)"}, got.Items[0])
}

func TestClient_WaitForProcessInstancesExpectation_MapsStateAndIncidentRequestAndReports(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	wantIncident := IncidentExpectationTrue
	domainIncident := true
	piAPI := stubProcessInstanceAPI{
		waitForProcessInstancesExpectation: func(_ context.Context, keys typex.Keys, request d.ProcessInstanceExpectationRequest, workers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
			cfg := services.ApplyCallOptions(opts)
			require.Equal(t, typex.Keys{"123"}, keys)
			require.Equal(t, d.States{d.StateActive}, request.States)
			require.NotNil(t, request.Incident)
			require.True(t, *request.Incident)
			require.Equal(t, 1, workers)
			require.True(t, cfg.Verbose)
			return d.ProcessInstanceExpectationResponses{
				Items: []d.ProcessInstanceExpectationResponse{
					{Key: "123", Ok: true, State: d.StateActive, Incident: &domainIncident, Status: "process instance 123 satisfied expectation(s)"},
				},
			}, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())

	got, err := cli.WaitForProcessInstancesExpectation(
		ctx,
		typex.Keys{"123", "123"},
		ProcessInstanceExpectationRequest{
			States:   States{StateActive},
			Incident: &wantIncident,
		},
		1,
		options.WithVerbose(),
	)

	require.NoError(t, err)
	require.Len(t, got.Items, 1)
	assert.Equal(t, ProcessInstanceExpectationReport{Key: "123", Ok: true, State: StateActive, Incident: &wantIncident, Status: "process instance 123 satisfied expectation(s)"}, got.Items[0])
}

// TestClient_SearchProcessInstancesPage_MapsPagingMetadata checks the full page
// contract: request echoing, overflow state, reported total kind, and item
// mapping all need to survive the facade boundary.
func TestClient_SearchProcessInstancesPage_MapsPagingMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{From: 25, Size: 10, After: "cursor-0"}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateIndeterminate,
				ReportedTotal: &d.ProcessInstanceReportedTotal{
					Count: 17,
					Kind:  d.ProcessInstanceReportedTotalKindExact,
				},
				EndCursor: "cursor-1",
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{From: 25, Size: 10, After: "cursor-0"}, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, ProcessInstancePageRequest{From: 25, Size: 10, After: "cursor-0"}, page.Request)
	assert.Equal(t, ProcessInstanceOverflowStateIndeterminate, page.OverflowState)
	require.NotNil(t, page.ReportedTotal)
	assert.Equal(t, int64(17), page.ReportedTotal.Count)
	assert.Equal(t, ProcessInstanceReportedTotalKindExact, page.ReportedTotal.Kind)
	assert.Equal(t, "cursor-1", page.EndCursor)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "2251799813711967", page.Items[0].Key)
}

// TestClient_SearchProcessInstancesPage_LeavesReportedTotalNilWhenUnavailable
// preserves the "unknown total" state. Nil is meaningful here and must not be
// converted into a misleading zero total.
func TestClient_SearchProcessInstancesPage_LeavesReportedTotalNilWhenUnavailable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{Size: 1}, page)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{Size: 1})

	require.NoError(t, err)
	assert.Nil(t, page.ReportedTotal)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstancesPage_MapsLowerBoundReportedTotal protects
// Camunda's lower-bound total semantics, where large result sets report "at
// least this many" rather than an exact count.
func TestClient_SearchProcessInstancesPage_MapsLowerBoundReportedTotal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{From: 100, Size: 25}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateHasMore,
				ReportedTotal: &d.ProcessInstanceReportedTotal{
					Count: 10000,
					Kind:  d.ProcessInstanceReportedTotalKindLowerBound,
				},
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{From: 100, Size: 25}, options.WithVerbose())

	require.NoError(t, err)
	require.NotNil(t, page.ReportedTotal)
	assert.Equal(t, int64(10000), page.ReportedTotal.Count)
	assert.Equal(t, ProcessInstanceReportedTotalKindLowerBound, page.ReportedTotal.Kind)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstancesPage_MapsPresenceFiltersToDomainFilter
// verifies pointer-backed booleans for has-parent and has-incident filters.
// Pointer identity matters because nil means "unspecified" while false means
// an explicit negative filter.
func TestClient_SearchProcessInstancesPage_MapsPresenceFiltersToDomainFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	hasParent := new(true)
	hasIncident := new(false)
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{
				HasParent:   hasParent,
				HasIncident: hasIncident,
			}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{From: 5, Size: 10}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		HasParent:   hasParent,
		HasIncident: hasIncident,
	}, ProcessInstancePageRequest{From: 5, Size: 10}, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, ProcessInstancePageRequest{From: 5, Size: 10}, page.Request)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstancesPage_PreservesCrossVersionOverflowStates
// ensures overflow state remains a cross-version domain signal instead of being
// flattened by the facade into item-count heuristics.
func TestClient_SearchProcessInstancesPage_PreservesCrossVersionOverflowStates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{Size: 2}, page)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateNoMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{Size: 2})

	require.NoError(t, err)
	assert.Equal(t, ProcessInstanceOverflowStateNoMore, page.OverflowState)
	require.Len(t, page.Items, 1)
}

// TestClient_SearchProcessInstances_UsesPagedSearchWrapper documents the legacy
// list facade over the newer paged service call. Total is derived from returned
// items here, while page metadata remains available through the page API.
func TestClient_SearchProcessInstances_UsesPagedSearchWrapper(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{Size: 2}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateHasMore,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
					{Key: "2251799813711968", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	items, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, 2, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, int32(2), items.Total)
	require.Len(t, items.Items, 2)
	assert.Equal(t, "2251799813711967", items.Items[0].Key)
	assert.Equal(t, "2251799813711968", items.Items[1].Key)
}

// TestClient_LookupProcessInstance_UsesSearchBackedLookup protects the lookup
// strategy for versions where direct key retrieval is implemented through a
// tenant-aware search filter.
func TestClient_LookupProcessInstance_UsesSearchBackedLookup(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, d.ProcessInstanceFilter{Key: "2251799813711967"}, filter)
			assert.Equal(t, int32(2), size)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstance{
				{Key: "2251799813711967", State: d.StateActive, TenantId: "tenant-a"},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	pi, err := cli.LookupProcessInstance(ctx, "2251799813711967", options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, "2251799813711967", pi.Key)
	assert.Equal(t, StateActive, pi.State)
	assert.Equal(t, "tenant-a", pi.TenantId)
}

// TestClient_LookupProcessInstanceStateByKey_MapsSearchBackedState verifies the
// public state report created from a search-backed lookup, including the stable
// uppercase status string used by command output.
func TestClient_LookupProcessInstanceStateByKey_MapsSearchBackedState(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstances: func(_ context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
			assert.Equal(t, d.ProcessInstanceFilter{Key: "2251799813711967"}, filter)
			assert.Equal(t, int32(2), size)
			return []d.ProcessInstance{
				{Key: "2251799813711967", State: d.StateCompleted, TenantId: "tenant-a"},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	report, pi, err := cli.LookupProcessInstanceStateByKey(ctx, "2251799813711967")

	require.NoError(t, err)
	assert.Equal(t, StateCompleted, report.State)
	assert.Equal(t, "COMPLETED", report.Status)
	assert.Equal(t, "2251799813711967", report.Key)
	assert.Equal(t, "2251799813711967", pi.Key)
	assert.Equal(t, StateCompleted, pi.State)
}

// TestClient_DryRunCancelOrDeleteGetPIKeys_DeduplicatesRootsAndCollected covers
// the legacy dry-run expansion contract. Multiple selected children can share a
// root, so roots and collected keys must keep deterministic first-seen order.
func TestClient_DryRunCancelOrDeleteGetPIKeys_DeduplicatesRootsAndCollected(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestry: func(_ context.Context, startKey string, _ ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
			switch startKey {
			case "c1", "c2":
				return "r1", nil, nil, nil
			case "c3":
				return "r2", nil, nil, nil
			default:
				t.Fatalf("unexpected key %q", startKey)
				return "", nil, nil, nil
			}
		},
		descendants: func(_ context.Context, rootKey string, _ ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
			switch rootKey {
			case "r1":
				return []string{"r1", "c1", "c2"}, nil, nil, nil
			case "r2":
				return []string{"r2", "c3"}, nil, nil, nil
			default:
				t.Fatalf("unexpected root %q", rootKey)
				return nil, nil, nil, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(ctx, typex.Keys{"c1", "c2", "c3"}, 0)

	require.NoError(t, err)
	assert.Equal(t, typex.Keys{"r1", "r2"}, roots)
	assert.Equal(t, typex.Keys{"r1", "c1", "c2", "r2", "c3"}, collected)
}

// TestClient_AncestryResult_MapsStructuredTraversalContract verifies the newer
// structured traversal model, including partial outcomes and missing-ancestor
// metadata used by automation and warning output.
func TestClient_AncestryResult_MapsStructuredTraversalContract(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			assert.Equal(t, "child", startKey)
			return pitraversal.Result{
				Mode:     pitraversal.ModeAncestry,
				StartKey: "child",
				RootKey:  "child",
				Keys:     []string{"child"},
				Chain: map[string]d.ProcessInstance{
					"child": {Key: "child", ParentKey: "missing"},
				},
				MissingAncestors: []pitraversal.MissingAncestor{{Key: "missing", StartKey: "child"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          pitraversal.OutcomePartial,
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.AncestryResult(ctx, "child")

	require.NoError(t, err)
	assert.Equal(t, TraversalModeAncestry, got.Mode)
	assert.Equal(t, "child", got.StartKey)
	assert.Equal(t, "child", got.RootKey)
	assert.Equal(t, []string{"child"}, got.Keys)
	assert.Equal(t, "missing", got.MissingAncestors[0].Key)
	assert.Equal(t, "child", got.Chain["child"].Key)
	assert.Equal(t, TraversalOutcomePartial, got.Outcome)
}

// TestClient_DryRunCancelOrDeletePlan_ReturnsStructuredExpansion exercises the
// full cancellation/deletion impact check: selected children resolve to roots,
// descendants expand per root, and partial ancestry warnings are preserved.
func TestClient_DryRunCancelOrDeletePlan_ReturnsStructuredExpansion(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			switch startKey {
			case "c1":
				return pitraversal.Result{
					Mode:             pitraversal.ModeAncestry,
					StartKey:         "c1",
					RootKey:          "r1",
					Keys:             []string{"c1", "r1"},
					Chain:            map[string]d.ProcessInstance{"c1": {Key: "c1", State: d.StateCanceled}, "r1": {Key: "r1", State: d.StateActive}},
					MissingAncestors: []pitraversal.MissingAncestor{{Key: "missing", StartKey: "c1"}},
					Warning:          "one or more parent process instances were not found",
					Outcome:          pitraversal.OutcomePartial,
				}, nil
			case "c2":
				return pitraversal.Result{
					Mode:     pitraversal.ModeAncestry,
					StartKey: "c2",
					RootKey:  "r2",
					Keys:     []string{"c2", "r2"},
					Chain:    map[string]d.ProcessInstance{"c2": {Key: "c2", State: d.StateActive}, "r2": {Key: "r2", State: d.StateActive}},
					Outcome:  pitraversal.OutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected key %q", startKey)
				return pitraversal.Result{}, nil
			}
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			switch rootKey {
			case "r1":
				return pitraversal.Result{
					Mode:    pitraversal.ModeDescendants,
					RootKey: "r1",
					Keys:    []string{"r1", "c1"},
					Chain:   map[string]d.ProcessInstance{"r1": {Key: "r1", State: d.StateActive}, "c1": {Key: "c1", State: d.StateCanceled}},
					Outcome: pitraversal.OutcomeComplete,
				}, nil
			case "r2":
				return pitraversal.Result{
					Mode:    pitraversal.ModeDescendants,
					RootKey: "r2",
					Keys:    []string{"r2", "c2"},
					Chain:   map[string]d.ProcessInstance{"r2": {Key: "r2", State: d.StateActive}, "c2": {Key: "c2", State: d.StateActive}},
					Outcome: pitraversal.OutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected root %q", rootKey)
				return pitraversal.Result{}, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.DryRunCancelOrDeletePlan(ctx, typex.Keys{"c1", "c2"}, 0)

	require.NoError(t, err)
	assert.Equal(t, typex.Keys{"r1", "r2"}, got.Roots)
	assert.Equal(t, typex.Keys{"r1", "c1", "r2", "c2"}, got.Collected)
	assert.Equal(t, []MissingAncestor{{Key: "missing", StartKey: "c1"}}, got.MissingAncestors)
	assert.Equal(t, []ProcessInstance{{Key: "c1", State: StateCanceled}}, got.SelectedFinalState)
	assert.Equal(t, []ProcessInstance{
		{Key: "r1", State: StateActive},
		{Key: "r2", State: StateActive},
		{Key: "c2", State: StateActive},
	}, got.RequiresCancelBeforeDelete)
	assert.Equal(t, TraversalOutcomePartial, got.Outcome)
	assert.NotEmpty(t, got.Warning)
}

// TestClient_DryRunCancelOrDeletePlan_UsesWorkersForStructuredTraversal keeps
// impact checks on the same worker path as the later mutation, because both
// ancestry and descendant expansion are pure IO.
func TestClient_DryRunCancelOrDeletePlan_UsesWorkersForStructuredTraversal(t *testing.T) {
	ctx := context.Background()

	var ancestryActive atomic.Int32
	var ancestryMax atomic.Int32
	var ancestryReleased atomic.Bool
	ancestryRelease := make(chan struct{})

	var descendantsActive atomic.Int32
	var descendantsMax atomic.Int32
	var descendantsReleased atomic.Bool
	descendantsRelease := make(chan struct{})

	waitForTraversalOverlap := func(active, max *atomic.Int32, released *atomic.Bool, release chan struct{}, phase string) func() {
		current := active.Add(1)
		for {
			seen := max.Load()
			if current <= seen || max.CompareAndSwap(seen, current) {
				break
			}
		}
		if current >= 2 && released.CompareAndSwap(false, true) {
			close(release)
		}
		select {
		case <-release:
		case <-time.After(2 * time.Second):
			if released.CompareAndSwap(false, true) {
				close(release)
			}
			t.Errorf("%s did not use concurrent workers", phase)
		}
		return func() {
			active.Add(-1)
		}
	}

	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			defer waitForTraversalOverlap(&ancestryActive, &ancestryMax, &ancestryReleased, ancestryRelease, "ancestry")()
			switch startKey {
			case "c1":
				return pitraversal.Result{
					Mode:     pitraversal.ModeAncestry,
					StartKey: "c1",
					RootKey:  "r1",
					Keys:     []string{"c1", "r1"},
					Chain:    map[string]d.ProcessInstance{"c1": {Key: "c1", State: d.StateActive}, "r1": {Key: "r1", State: d.StateActive}},
					Outcome:  pitraversal.OutcomeComplete,
				}, nil
			case "c2":
				return pitraversal.Result{
					Mode:     pitraversal.ModeAncestry,
					StartKey: "c2",
					RootKey:  "r2",
					Keys:     []string{"c2", "r2"},
					Chain:    map[string]d.ProcessInstance{"c2": {Key: "c2", State: d.StateActive}, "r2": {Key: "r2", State: d.StateActive}},
					Outcome:  pitraversal.OutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected key %q", startKey)
				return pitraversal.Result{}, nil
			}
		},
		descendantsResult: func(_ context.Context, rootKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			defer waitForTraversalOverlap(&descendantsActive, &descendantsMax, &descendantsReleased, descendantsRelease, "descendants")()
			switch rootKey {
			case "r1":
				return pitraversal.Result{
					Mode:    pitraversal.ModeDescendants,
					RootKey: "r1",
					Keys:    []string{"r1", "c1"},
					Chain:   map[string]d.ProcessInstance{"r1": {Key: "r1", State: d.StateActive}, "c1": {Key: "c1", State: d.StateActive}},
					Outcome: pitraversal.OutcomeComplete,
				}, nil
			case "r2":
				return pitraversal.Result{
					Mode:    pitraversal.ModeDescendants,
					RootKey: "r2",
					Keys:    []string{"r2", "c2"},
					Chain:   map[string]d.ProcessInstance{"r2": {Key: "r2", State: d.StateActive}, "c2": {Key: "c2", State: d.StateActive}},
					Outcome: pitraversal.OutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected root %q", rootKey)
				return pitraversal.Result{}, nil
			}
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	got, err := cli.DryRunCancelOrDeletePlan(ctx, typex.Keys{"c1", "c2"}, 2)

	require.NoError(t, err)
	assert.Equal(t, int32(2), ancestryMax.Load())
	assert.Equal(t, int32(2), descendantsMax.Load())
	assert.Equal(t, typex.Keys{"r1", "r2"}, got.Roots)
	assert.Equal(t, typex.Keys{"r1", "c1", "r2", "c2"}, got.Collected)
}

// TestClient_DryRunCancelOrDeletePlan_FailsWhenNoActionableResultsResolve keeps
// unresolved ancestry from silently producing an empty successful plan. That
// protects destructive commands from proceeding when nothing can be targeted.
func TestClient_DryRunCancelOrDeletePlan_FailsWhenNoActionableResultsResolve(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		ancestryResult: func(_ context.Context, startKey string, _ ...services.CallOption) (pitraversal.Result, error) {
			assert.Equal(t, "c1", startKey)
			return pitraversal.Result{
				Mode:             pitraversal.ModeAncestry,
				StartKey:         "c1",
				MissingAncestors: []pitraversal.MissingAncestor{{Key: "missing", StartKey: "c1"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          pitraversal.OutcomeUnresolved,
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())
	_, err := cli.DryRunCancelOrDeletePlan(ctx, typex.Keys{"c1"}, 0)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no process instances resolved during dependency expansion")
}

// TestClient_CancelProcessInstances_LogsExpandedAffectedScope verifies cancel
// logs describe expanded dry-run scope counts.
func TestClient_CancelProcessInstances_LogsExpandedAffectedScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var logBuf bytes.Buffer
	piAPI := stubProcessInstanceAPI{
		cancelProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			assert.Equal(t, "root-1", key)
			return d.CancelResponse{Ok: true, StatusCode: 202, Status: "202 Accepted"}, nil, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)))

	reports, err := cli.CancelProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4), options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, reports.Items, 1)
	assert.Contains(t, logBuf.String(), "cancelling process instances requested for 4 affected instance(s) across 1 root key(s)")
	assert.Contains(t, logBuf.String(), "cancelling 4 process instance(s) completed via 1 root request(s): 1 root request(s) succeeded or already cancelled/terminated, 0 failed")
	assert.NotContains(t, logBuf.String(), "cancelling 1 process instance(s) completed")
}

// TestClient_CancelProcessInstances_UsesActivityIndicator verifies bulk cancel emits activity lifecycle messages.
func TestClient_CancelProcessInstances_UsesActivityIndicator(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	ctx := logging.ToActivityContext(context.Background(), sink)
	piAPI := stubProcessInstanceAPI{
		cancelProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
			assert.Equal(t, "root-1", key)
			return d.CancelResponse{Ok: true, StatusCode: 202, Status: "202 Accepted"}, nil, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.Default())

	_, err := cli.CancelProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4))

	require.NoError(t, err)
	started, stopped, msgs := sink.Snapshot()
	assert.Equal(t, 1, started)
	assert.Equal(t, 1, stopped)
	assert.Equal(t, []string{"cancelling 4 process instance(s) via 1 root request(s)"}, msgs)
}

// TestClient_DeleteProcessInstances_LogsExpandedAffectedScope verifies delete
// logs describe expanded dry-run scope counts.
func TestClient_DeleteProcessInstances_LogsExpandedAffectedScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var logBuf bytes.Buffer
	piAPI := stubProcessInstanceAPI{
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			assert.Equal(t, "root-1", key)
			return d.DeleteResponse{Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)))

	reports, err := cli.DeleteProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4), options.WithVerbose())

	require.NoError(t, err)
	require.Len(t, reports.Items, 1)
	assert.Contains(t, logBuf.String(), "deleting process instances requested for 4 affected instance(s) across 1 root key(s)")
	assert.Contains(t, logBuf.String(), "deleting 4 process instance(s) completed via 1 root request(s): 1 root request(s) succeeded, 0 failed")
	assert.NotContains(t, logBuf.String(), "deleting 1 process instances completed")
}

// TestClient_DeleteProcessInstances_LogsConsolidatedWrongStateForExpandedScope
// verifies delete failures summarize non-terminal expanded scope.
func TestClient_DeleteProcessInstances_LogsConsolidatedWrongStateForExpandedScope(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var logBuf bytes.Buffer
	piAPI := stubProcessInstanceAPI{
		deleteProcessInstance: func(_ context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			assert.Equal(t, "root-1", key)
			return d.DeleteResponse{StatusCode: 409, Status: "409 Conflict"}, nil
		},
	}
	cli := New(&stubProcessDefinitionAPI{}, piAPI, stubIncidentAPI{}, slog.New(logging.NewPlainHandler(&logBuf, slog.LevelDebug)))

	reports, err := cli.DeleteProcessInstances(ctx, typex.Keys{"root-1"}, 0, options.WithAffectedProcessInstanceCount(4))

	require.NoError(t, err)
	require.Len(t, reports.Items, 1)
	assert.Contains(t, logBuf.String(), "cannot delete expanded process-instance scope of 4 process instance(s): one or more affected process instances are not in a terminated state; use --force flag to cancel and then delete them")
	assert.Contains(t, logBuf.String(), "deleting 4 process instance(s) completed via 1 root request(s): 0 root request(s) succeeded, 1 failed")
}

type stubProcessDefinitionAPI struct {
	searchProcessDefinitions       func(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	searchProcessDefinitionsLatest func(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error)
	getProcessDefinition           func(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error)
	getProcessDefinitionXML        func(ctx context.Context, key string, opts ...services.CallOption) (string, error)
}

// SearchProcessDefinitions delegates to the per-test callback and panics when a
// test did not authorize this service method.
func (s *stubProcessDefinitionAPI) SearchProcessDefinitions(ctx context.Context, filter d.ProcessDefinitionFilter, size int32, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	if s.searchProcessDefinitions == nil {
		panic("unexpected call")
	}
	return s.searchProcessDefinitions(ctx, filter, size, opts...)
}

// SearchProcessDefinitionsLatest delegates to the per-test callback and panics
// when a test did not authorize this service method.
func (s *stubProcessDefinitionAPI) SearchProcessDefinitionsLatest(ctx context.Context, filter d.ProcessDefinitionFilter, opts ...services.CallOption) ([]d.ProcessDefinition, error) {
	if s.searchProcessDefinitionsLatest == nil {
		panic("unexpected call")
	}
	return s.searchProcessDefinitionsLatest(ctx, filter, opts...)
}

// GetProcessDefinition delegates to the per-test callback and panics on
// unexpected lookups to keep facade tests narrowly scoped.
func (s *stubProcessDefinitionAPI) GetProcessDefinition(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessDefinition, error) {
	if s.getProcessDefinition == nil {
		panic("unexpected call")
	}
	return s.getProcessDefinition(ctx, key, opts...)
}

// GetProcessDefinitionXML delegates to the per-test callback for XML-specific
// facade assertions.
func (s *stubProcessDefinitionAPI) GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error) {
	if s.getProcessDefinitionXML == nil {
		panic("unexpected call")
	}
	return s.getProcessDefinitionXML(ctx, key, opts...)
}

var _ pdsvc.API = (*stubProcessDefinitionAPI)(nil)

type stubProcessInstanceAPI struct {
	searchForProcessInstances          func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error)
	searchForProcessInstancesPage      func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
	searchProcessInstanceVariables     func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceVariable, error)
	updateProcessInstanceVariables     func(context.Context, string, map[string]any, ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error)
	ancestry                           func(context.Context, string, ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error)
	descendants                        func(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error)
	cancelProcessInstance              func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	deleteProcessInstance              func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error)
	ancestryResult                     func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	descendantsResult                  func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	familyResult                       func(context.Context, string, ...services.CallOption) (pitraversal.Result, error)
	waitForProcessInstanceExpectation  func(context.Context, string, d.ProcessInstanceExpectationRequest, ...services.CallOption) (d.ProcessInstanceExpectationResponse, d.ProcessInstance, error)
	waitForProcessInstancesExpectation func(context.Context, typex.Keys, d.ProcessInstanceExpectationRequest, int, ...services.CallOption) (d.ProcessInstanceExpectationResponses, error)
}

// CreateProcessInstance panics when a facade test accidentally starts a process instance.
func (stubProcessInstanceAPI) CreateProcessInstance(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error) {
	panic("unexpected call")
}

// GetProcessInstance panics when a facade test accidentally performs direct lookup.
func (stubProcessInstanceAPI) GetProcessInstance(context.Context, string, ...services.CallOption) (d.ProcessInstance, error) {
	panic("unexpected call")
}

// SearchProcessInstanceVariables delegates to the per-test callback used by variable enrichment facade tests.
func (s stubProcessInstanceAPI) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	if s.searchProcessInstanceVariables == nil {
		panic("unexpected call")
	}
	return s.searchProcessInstanceVariables(ctx, key, opts...)
}

type stubIncidentAPI struct {
	getIncident                    func(context.Context, string, ...services.CallOption) (d.ProcessInstanceIncidentDetail, error)
	resolveIncident                func(context.Context, string, ...services.CallOption) (d.IncidentResolutionResponse, error)
	searchIncidents                func(context.Context, d.IncidentFilter, int32, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	searchIncidentsPage            func(context.Context, d.IncidentFilter, d.IncidentPageRequest, ...services.CallOption) (d.IncidentPage, error)
	searchProcessInstanceIncidents func(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
	waitForIncidentResolved        func(context.Context, string, ...services.CallOption) (d.IncidentResolutionResponse, error)
	waitForPIIncidentsResolved     func(context.Context, string, []string, ...services.CallOption) (d.IncidentResolutionResponse, error)
}

// GetIncident delegates to the per-test callback used by direct incident resolution facade tests.
func (s stubIncidentAPI) GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	if s.getIncident == nil {
		panic("unexpected call")
	}
	return s.getIncident(ctx, key, opts...)
}

// ResolveIncident delegates to the per-test callback used by incident mutation facade tests.
func (s stubIncidentAPI) ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.resolveIncident == nil {
		panic("unexpected call")
	}
	return s.resolveIncident(ctx, key, opts...)
}

// SearchIncidents delegates to the per-test callback used by incident search facade tests.
func (s stubIncidentAPI) SearchIncidents(ctx context.Context, filter d.IncidentFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.searchIncidents == nil {
		panic("unexpected call")
	}
	return s.searchIncidents(ctx, filter, size, opts...)
}

// SearchIncidentsPage delegates to the per-test callback used by incident search facade tests.
func (s stubIncidentAPI) SearchIncidentsPage(ctx context.Context, filter d.IncidentFilter, page d.IncidentPageRequest, opts ...services.CallOption) (d.IncidentPage, error) {
	if s.searchIncidentsPage == nil {
		panic("unexpected call")
	}
	return s.searchIncidentsPage(ctx, filter, page, opts...)
}

// SearchProcessInstanceIncidents delegates to the per-test callback used by incident enrichment facade tests.
func (s stubIncidentAPI) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	if s.searchProcessInstanceIncidents == nil {
		panic("unexpected call")
	}
	return s.searchProcessInstanceIncidents(ctx, key, opts...)
}

// WaitForIncidentResolved delegates to the per-test callback used by incident confirmation facade tests.
func (s stubIncidentAPI) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.waitForIncidentResolved == nil {
		panic("unexpected call")
	}
	return s.waitForIncidentResolved(ctx, key, opts...)
}

// WaitForProcessInstanceIncidentsResolved delegates to the per-test callback used by process-instance resolution facade tests.
func (s stubIncidentAPI) WaitForProcessInstanceIncidentsResolved(ctx context.Context, key string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	if s.waitForPIIncidentsResolved == nil {
		panic("unexpected call")
	}
	return s.waitForPIIncidentsResolved(ctx, key, incidentKeys, opts...)
}

var _ incsvc.API = stubIncidentAPI{}

// UpdateProcessInstanceVariables delegates to the per-test callback and panics on unexpected update calls.
func (s stubProcessInstanceAPI) UpdateProcessInstanceVariables(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
	if s.updateProcessInstanceVariables == nil {
		panic("unexpected call")
	}
	return s.updateProcessInstanceVariables(ctx, key, variables, opts...)
}

// GetDirectChildrenOfProcessInstance panics when a test takes the direct-children path unexpectedly.
func (stubProcessInstanceAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

// FilterProcessInstanceWithOrphanParent panics when a test takes orphan filtering unexpectedly.
func (stubProcessInstanceAPI) FilterProcessInstanceWithOrphanParent(context.Context, []d.ProcessInstance, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

// SearchForProcessInstances delegates to the per-test callback and panics when
// a test accidentally takes the legacy list path.
func (s stubProcessInstanceAPI) SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	if s.searchForProcessInstances == nil {
		panic("unexpected call")
	}
	return s.searchForProcessInstances(ctx, filter, size, opts...)
}

// SearchForProcessInstancesPage delegates to an explicit page callback when a
// test provides one, otherwise it adapts the legacy list callback for older
// facade tests that only care about returned items.
func (s stubProcessInstanceAPI) SearchForProcessInstancesPage(ctx context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
	if s.searchForProcessInstancesPage != nil {
		return s.searchForProcessInstancesPage(ctx, filter, page, opts...)
	}
	if s.searchForProcessInstances == nil {
		panic("unexpected call")
	}
	items, err := s.searchForProcessInstances(ctx, filter, page.Size, opts...)
	if err != nil {
		return d.ProcessInstancePage{}, err
	}
	return d.ProcessInstancePage{
		Request: page,
		Items:   items,
	}, nil
}

// CancelProcessInstance delegates to the per-test callback and panics on unexpected cancel calls.
func (s stubProcessInstanceAPI) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	if s.cancelProcessInstance == nil {
		panic("unexpected call")
	}
	return s.cancelProcessInstance(ctx, key, opts...)
}

// DeleteProcessInstance delegates to the per-test callback and panics on unexpected delete calls.
func (s stubProcessInstanceAPI) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	if s.deleteProcessInstance == nil {
		panic("unexpected call")
	}
	return s.deleteProcessInstance(ctx, key, opts...)
}

// GetProcessInstanceStateByKey panics when a facade test accidentally checks process state.
func (stubProcessInstanceAPI) GetProcessInstanceStateByKey(context.Context, string, ...services.CallOption) (d.State, d.ProcessInstance, error) {
	panic("unexpected call")
}

// WaitForProcessInstanceState panics when a facade test accidentally waits for one process instance.
func (stubProcessInstanceAPI) WaitForProcessInstanceState(context.Context, string, d.States, ...services.CallOption) (d.StateResponse, d.ProcessInstance, error) {
	panic("unexpected call")
}

// WaitForProcessInstanceExpectation delegates to the per-test callback and panics on unexpected single expectation waits.
func (s stubProcessInstanceAPI) WaitForProcessInstanceExpectation(ctx context.Context, key string, request d.ProcessInstanceExpectationRequest, opts ...services.CallOption) (d.ProcessInstanceExpectationResponse, d.ProcessInstance, error) {
	if s.waitForProcessInstanceExpectation == nil {
		panic("unexpected call")
	}
	return s.waitForProcessInstanceExpectation(ctx, key, request, opts...)
}

// Ancestry delegates the legacy traversal tuple used by older dry-run helpers.
func (s stubProcessInstanceAPI) Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
	if s.ancestry == nil {
		panic("unexpected call")
	}
	return s.ancestry(ctx, startKey, opts...)
}

// Descendants delegates the legacy traversal tuple used to expand a resolved
// root into actionable process instance keys.
func (s stubProcessInstanceAPI) Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	if s.descendants == nil {
		panic("unexpected call")
	}
	return s.descendants(ctx, rootKey, opts...)
}

// Family panics when a test unexpectedly takes the legacy full-family path.
func (stubProcessInstanceAPI) Family(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	panic("unexpected call")
}

// AncestryResult delegates structured traversal when supplied, or builds it
// from the legacy tuple callback to keep migration tests compact.
func (s stubProcessInstanceAPI) AncestryResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.ancestryResult != nil {
		return s.ancestryResult(ctx, startKey, opts...)
	}
	if s.ancestry == nil {
		panic("unexpected call")
	}
	return pitraversal.BuildAncestryResult(ctx, s, startKey, opts...)
}

// DescendantsResult delegates structured traversal when supplied, or builds it
// from the legacy tuple callback for tests that cover compatibility behavior.
func (s stubProcessInstanceAPI) DescendantsResult(ctx context.Context, rootKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.descendantsResult != nil {
		return s.descendantsResult(ctx, rootKey, opts...)
	}
	if s.descendants == nil {
		panic("unexpected call")
	}
	return pitraversal.BuildDescendantsResult(ctx, s, rootKey, opts...)
}

// FamilyResult delegates a structured family traversal when supplied, otherwise
// composes one from the legacy ancestry and descendants callbacks.
func (s stubProcessInstanceAPI) FamilyResult(ctx context.Context, startKey string, opts ...services.CallOption) (pitraversal.Result, error) {
	if s.familyResult != nil {
		return s.familyResult(ctx, startKey, opts...)
	}
	return pitraversal.BuildFamilyResult(ctx, s, startKey, opts...)
}

// GetProcessInstances panics when a facade test accidentally performs bulk lookup.
func (stubProcessInstanceAPI) GetProcessInstances(context.Context, typex.Keys, int, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

// WaitForProcessInstancesState panics when a facade test accidentally waits for bulk state changes.
func (stubProcessInstanceAPI) WaitForProcessInstancesState(context.Context, typex.Keys, d.States, int, ...services.CallOption) (d.StateResponses, error) {
	panic("unexpected call")
}

// WaitForProcessInstancesExpectation delegates to the per-test callback and panics on unexpected bulk expectation waits.
func (s stubProcessInstanceAPI) WaitForProcessInstancesExpectation(ctx context.Context, keys typex.Keys, request d.ProcessInstanceExpectationRequest, workers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
	if s.waitForProcessInstancesExpectation == nil {
		panic("unexpected call")
	}
	return s.waitForProcessInstancesExpectation(ctx, keys, request, workers, opts...)
}

var _ pisvc.API = stubProcessInstanceAPI{}
