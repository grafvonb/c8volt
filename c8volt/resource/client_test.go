package resource

import (
	"context"
	"log/slog"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_GetResource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := &stubResourceAPI{
		get: func(_ context.Context, key string, opts ...services.CallOption) (d.Resource, error) {
			cfg := services.ApplyCallOptions(opts)
			assert.Equal(t, "resource-1", key)
			assert.True(t, cfg.Verbose)
			return d.Resource{
				ID:         "demo-process",
				Key:        "resource-1",
				Name:       "demo.bpmn",
				TenantId:   "tenant-a",
				Version:    7,
				VersionTag: "v7",
			}, nil
		},
	}

	cli := New(api, nil, slog.Default())
	got, err := cli.GetResource(ctx, "resource-1", options.WithVerbose())

	require.NoError(t, err)
	assert.Equal(t, Resource{
		ID:         "demo-process",
		Key:        "resource-1",
		Name:       "demo.bpmn",
		TenantId:   "tenant-a",
		Version:    7,
		VersionTag: "v7",
	}, got)
}

func TestClient_GetResource_MapsDomainErrors(t *testing.T) {
	t.Parallel()

	api := &stubResourceAPI{
		get: func(context.Context, string, ...services.CallOption) (d.Resource, error) {
			return d.Resource{}, d.ErrNotFound
		},
	}

	cli := New(api, nil, slog.Default())
	_, err := cli.GetResource(context.Background(), "missing")

	require.Error(t, err)
	assert.ErrorIs(t, err, ferr.ErrNotFound)
}

func TestClient_DeleteProcessDefinition_UsesStructuredDryRunPlan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	api := &stubResourceAPI{
		delete: func(_ context.Context, resourceKey string, _ ...services.CallOption) error {
			assert.Equal(t, "pd-1", resourceKey)
			return nil
		},
	}

	var canceledKeys typex.Keys
	papi := stubProcessAPI{
		searchProcessInstances: func(_ context.Context, filter process.ProcessInstanceFilter, size int32, _ ...options.FacadeOption) (process.ProcessInstances, error) {
			assert.Equal(t, process.ProcessInstanceFilter{ProcessDefinitionKey: "pd-1", State: process.StateActive}, filter)
			assert.Contains(t, []int32{1, consts.MaxPISearchSize}, size)
			return process.ProcessInstances{Items: []process.ProcessInstance{{Key: "child-1"}}}, nil
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			assert.Equal(t, typex.Keys{"child-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"root-1"},
				Collected:        typex.Keys{"root-1", "child-1"},
				MissingAncestors: []process.MissingAncestor{{Key: "missing-1", StartKey: "child-1"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, _ ...options.FacadeOption) (process.CancelReports, error) {
			canceledKeys = append(typex.Keys(nil), keys...)
			assert.Equal(t, 1, wantedWorkers)
			return process.CancelReports{Items: []process.CancelReport{{Key: "root-1", Ok: true}}}, nil
		},
	}

	cli := New(api, papi, slog.Default())
	report, err := cli.DeleteProcessDefinition(ctx, "pd-1", options.WithForce(), options.WithAllowInconsistent())

	require.NoError(t, err)
	assert.True(t, report.Ok)
	assert.Equal(t, typex.Keys{"root-1"}, canceledKeys)
}

type stubResourceAPI struct {
	get    func(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error)
	delete func(ctx context.Context, resourceKey string, opts ...services.CallOption) error
}

func (s *stubResourceAPI) Deploy(context.Context, []d.DeploymentUnitData, ...services.CallOption) (d.Deployment, error) {
	panic("unexpected call")
}

func (s *stubResourceAPI) Delete(ctx context.Context, resourceKey string, opts ...services.CallOption) error {
	if s.delete == nil {
		panic("unexpected call")
	}
	return s.delete(ctx, resourceKey, opts...)
}

func (s *stubResourceAPI) Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error) {
	if s.get == nil {
		panic("unexpected call")
	}
	return s.get(ctx, resourceKey, opts...)
}

var _ rsvc.API = (*stubResourceAPI)(nil)

type stubProcessAPI struct {
	searchProcessInstances   func(context.Context, process.ProcessInstanceFilter, int32, ...options.FacadeOption) (process.ProcessInstances, error)
	dryRunCancelOrDeletePlan func(context.Context, typex.Keys, ...options.FacadeOption) (process.DryRunPIKeyExpansion, error)
	cancelProcessInstances   func(context.Context, typex.Keys, int, ...options.FacadeOption) (process.CancelReports, error)
}

func (stubProcessAPI) SearchProcessDefinitions(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessDefinitionsLatest(context.Context, process.ProcessDefinitionFilter, ...options.FacadeOption) (process.ProcessDefinitions, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessDefinition(context.Context, string, ...options.FacadeOption) (process.ProcessDefinition, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessDefinitionXML(context.Context, string, ...options.FacadeOption) (string, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstance(context.Context, process.ProcessInstanceData, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateProcessInstances(context.Context, []process.ProcessInstanceData, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) LookupProcessInstanceStateByKey(context.Context, string, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) SearchProcessInstancesPage(context.Context, process.ProcessInstanceFilter, process.ProcessInstancePageRequest, ...options.FacadeOption) (process.ProcessInstancePage, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) SearchProcessInstances(ctx context.Context, filter process.ProcessInstanceFilter, size int32, opts ...options.FacadeOption) (process.ProcessInstances, error) {
	if s.searchProcessInstances == nil {
		panic("unexpected call")
	}
	return s.searchProcessInstances(ctx, filter, size, opts...)
}

func (stubProcessAPI) CancelProcessInstance(context.Context, string, ...options.FacadeOption) (process.CancelReport, process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DeleteProcessInstance(context.Context, string, ...options.FacadeOption) (process.DeleteReport, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetDirectChildrenOfProcessInstance(context.Context, string, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) FilterProcessInstanceWithOrphanParent(context.Context, []process.ProcessInstance, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) WaitForProcessInstanceState(context.Context, string, process.States, ...options.FacadeOption) (process.StateReport, process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Ancestry(context.Context, string, ...options.FacadeOption) (string, []string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Descendants(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) Family(context.Context, string, ...options.FacadeOption) ([]string, map[string][]string, map[string]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (stubProcessAPI) AncestryResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DescendantsResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) FamilyResult(context.Context, string, ...options.FacadeOption) (process.TraversalResult, error) {
	panic("unexpected call")
}

func (stubProcessAPI) GetProcessInstances(context.Context, typex.Keys, int, ...options.FacadeOption) (process.ProcessInstances, error) {
	panic("unexpected call")
}

func (stubProcessAPI) CreateNProcessInstances(context.Context, process.ProcessInstanceData, int, int, ...options.FacadeOption) ([]process.ProcessInstance, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) CancelProcessInstances(ctx context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.CancelReports, error) {
	if s.cancelProcessInstances == nil {
		panic("unexpected call")
	}
	return s.cancelProcessInstances(ctx, keys, wantedWorkers, opts...)
}

func (stubProcessAPI) DeleteProcessInstances(context.Context, typex.Keys, int, ...options.FacadeOption) (process.DeleteReports, error) {
	panic("unexpected call")
}

func (stubProcessAPI) WaitForProcessInstancesState(context.Context, typex.Keys, process.States, int, ...options.FacadeOption) (process.StateReports, error) {
	panic("unexpected call")
}

func (stubProcessAPI) DryRunCancelOrDeleteGetPIKeys(context.Context, typex.Keys, ...options.FacadeOption) (typex.Keys, typex.Keys, error) {
	panic("unexpected call")
}

func (s stubProcessAPI) DryRunCancelOrDeletePlan(ctx context.Context, keys typex.Keys, opts ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
	if s.dryRunCancelOrDeletePlan == nil {
		panic("unexpected call")
	}
	return s.dryRunCancelOrDeletePlan(ctx, keys, opts...)
}

var _ process.API = stubProcessAPI{}
