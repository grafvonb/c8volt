package resource

import (
	"context"
	"log/slog"
	"testing"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	rsvc "github.com/grafvonb/c8volt/internal/services/resource"
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

type stubResourceAPI struct {
	get func(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error)
}

func (s *stubResourceAPI) Deploy(context.Context, []d.DeploymentUnitData, ...services.CallOption) (d.Deployment, error) {
	panic("unexpected call")
}

func (s *stubResourceAPI) Delete(context.Context, string, ...services.CallOption) error {
	panic("unexpected call")
}

func (s *stubResourceAPI) Get(ctx context.Context, resourceKey string, opts ...services.CallOption) (d.Resource, error) {
	if s.get == nil {
		panic("unexpected call")
	}
	return s.get(ctx, resourceKey, opts...)
}

var _ rsvc.API = (*stubResourceAPI)(nil)
