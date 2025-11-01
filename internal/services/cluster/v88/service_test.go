package v88

import (
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/testx"
	"github.com/stretchr/testify/require"
)

func Test_Internal_Cluster_v88_GetClusterTopology_OK(t *testing.T) {
	ctx := testx.ITCtx(t, 20*time.Second)
	cfg := testx.TestConfig(t)
	log := testx.Logger(t)

	fs := testx.NewFakeServer(t)
	httpClient := fs.FS.Client()
	cfg.APIs.Camunda.BaseURL = fs.BaseURL + "/v2"

	svc, err := New(cfg, httpClient, log)
	require.NoError(t, err)

	topology, err := svc.GetClusterTopology(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, topology)

	t.Logf("success: got cluster topology")
	testx.LogJson(t, topology)
}
