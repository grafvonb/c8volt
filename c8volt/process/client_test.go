package process

import (
	"context"
	"log/slog"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

	cli := New(pdAPI, stubProcessInstanceAPI{}, slog.Default())
	xml, err := cli.GetProcessDefinitionXML(ctx, "2251799813685255", options.WithVerbose(), options.WithStat())

	require.NoError(t, err)
	assert.Equal(t, "<definitions id=\"order-process\"/>", xml)
}

func TestClient_SearchProcessInstances_MapsDateBoundsToDomainFilter(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
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
			}, filter)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return []d.ProcessInstance{}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	_, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		BpmnProcessId:        "order-process",
		ProcessDefinitionKey: "2251799813685255",
		StartDateAfter:       "2026-01-01",
		StartDateBefore:      "2026-01-31",
		EndDateAfter:         "2026-02-01",
		EndDateBefore:        "2026-02-28",
		State:                StateCompleted,
		ParentKey:            "12345",
	}, 25, options.WithVerbose())

	require.NoError(t, err)
}

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

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	_, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		StartDateAfter:  "2026-03-11",
		StartDateBefore: "2026-04-03",
		EndDateAfter:    "2026-02-09",
		EndDateBefore:   "2026-03-27",
	}, 10, options.WithVerbose())

	require.NoError(t, err)
}

func TestClient_SearchProcessInstancesPage_MapsPagingMetadata(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	piAPI := stubProcessInstanceAPI{
		searchForProcessInstancesPage: func(_ context.Context, filter d.ProcessInstanceFilter, page d.ProcessInstancePageRequest, opts ...services.CallOption) (d.ProcessInstancePage, error) {
			assert.Equal(t, d.ProcessInstanceFilter{BpmnProcessId: "order-process"}, filter)
			assert.Equal(t, d.ProcessInstancePageRequest{From: 25, Size: 10}, page)
			assert.True(t, services.ApplyCallOptions(opts).Verbose)
			return d.ProcessInstancePage{
				Request:       page,
				OverflowState: d.ProcessInstanceOverflowStateIndeterminate,
				Items: []d.ProcessInstance{
					{Key: "2251799813711967", BpmnProcessId: "order-process"},
				},
			}, nil
		},
	}

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{From: 25, Size: 10}, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, ProcessInstancePageRequest{From: 25, Size: 10}, page.Request)
	assert.Equal(t, ProcessInstanceOverflowStateIndeterminate, page.OverflowState)
	require.Len(t, page.Items, 1)
	assert.Equal(t, "2251799813711967", page.Items[0].Key)
}

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

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	page, err := cli.SearchProcessInstancesPage(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, ProcessInstancePageRequest{Size: 2})

	require.NoError(t, err)
	assert.Equal(t, ProcessInstanceOverflowStateNoMore, page.OverflowState)
	require.Len(t, page.Items, 1)
}

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

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	items, err := cli.SearchProcessInstances(ctx, ProcessInstanceFilter{
		BpmnProcessId: "order-process",
	}, 2, options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, int32(2), items.Total)
	require.Len(t, items.Items, 2)
	assert.Equal(t, "2251799813711967", items.Items[0].Key)
	assert.Equal(t, "2251799813711968", items.Items[1].Key)
}

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

	cli := New(&stubProcessDefinitionAPI{}, piAPI, slog.Default())
	roots, collected, err := cli.DryRunCancelOrDeleteGetPIKeys(ctx, typex.Keys{"c1", "c2", "c3"})

	require.NoError(t, err)
	assert.Equal(t, typex.Keys{"r1", "r2"}, roots)
	assert.Equal(t, typex.Keys{"r1", "c1", "c2", "r2", "c3"}, collected)
}

type stubProcessDefinitionAPI struct {
	getProcessDefinitionXML func(ctx context.Context, key string, opts ...services.CallOption) (string, error)
}

func (s *stubProcessDefinitionAPI) SearchProcessDefinitions(context.Context, d.ProcessDefinitionFilter, int32, ...services.CallOption) ([]d.ProcessDefinition, error) {
	panic("unexpected call")
}

func (s *stubProcessDefinitionAPI) SearchProcessDefinitionsLatest(context.Context, d.ProcessDefinitionFilter, ...services.CallOption) ([]d.ProcessDefinition, error) {
	panic("unexpected call")
}

func (s *stubProcessDefinitionAPI) GetProcessDefinition(context.Context, string, ...services.CallOption) (d.ProcessDefinition, error) {
	panic("unexpected call")
}

func (s *stubProcessDefinitionAPI) GetProcessDefinitionXML(ctx context.Context, key string, opts ...services.CallOption) (string, error) {
	if s.getProcessDefinitionXML == nil {
		panic("unexpected call")
	}
	return s.getProcessDefinitionXML(ctx, key, opts...)
}

var _ pdsvc.API = (*stubProcessDefinitionAPI)(nil)

type stubProcessInstanceAPI struct {
	searchForProcessInstances     func(context.Context, d.ProcessInstanceFilter, int32, ...services.CallOption) ([]d.ProcessInstance, error)
	searchForProcessInstancesPage func(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
	ancestry                      func(context.Context, string, ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error)
	descendants                   func(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error)
}

func (stubProcessInstanceAPI) CreateProcessInstance(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) GetProcessInstance(context.Context, string, ...services.CallOption) (d.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) FilterProcessInstanceWithOrphanParent(context.Context, []d.ProcessInstance, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

func (s stubProcessInstanceAPI) SearchForProcessInstances(ctx context.Context, filter d.ProcessInstanceFilter, size int32, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	if s.searchForProcessInstances == nil {
		panic("unexpected call")
	}
	return s.searchForProcessInstances(ctx, filter, size, opts...)
}

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

func (stubProcessInstanceAPI) CancelProcessInstance(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) DeleteProcessInstance(context.Context, string, ...services.CallOption) (d.DeleteResponse, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) GetProcessInstanceStateByKey(context.Context, string, ...services.CallOption) (d.State, d.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) WaitForProcessInstanceState(context.Context, string, d.States, ...services.CallOption) (d.StateResponse, d.ProcessInstance, error) {
	panic("unexpected call")
}

func (s stubProcessInstanceAPI) Ancestry(ctx context.Context, startKey string, opts ...services.CallOption) (string, []string, map[string]d.ProcessInstance, error) {
	if s.ancestry == nil {
		panic("unexpected call")
	}
	return s.ancestry(ctx, startKey, opts...)
}

func (s stubProcessInstanceAPI) Descendants(ctx context.Context, rootKey string, opts ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	if s.descendants == nil {
		panic("unexpected call")
	}
	return s.descendants(ctx, rootKey, opts...)
}

func (stubProcessInstanceAPI) Family(context.Context, string, ...services.CallOption) ([]string, map[string][]string, map[string]d.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) GetProcessInstances(context.Context, typex.Keys, int, ...services.CallOption) ([]d.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessInstanceAPI) WaitForProcessInstancesState(context.Context, typex.Keys, d.States, int, ...services.CallOption) (d.StateResponses, error) {
	panic("unexpected call")
}

var _ pisvc.API = stubProcessInstanceAPI{}
