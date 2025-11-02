package cluster_test

import (
	"net/http"
	"testing"

	"log/slog"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/cluster"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{},
	}
}

func TestFactory_V87(t *testing.T) {
	cfg := testConfig()
	cfg.App.CamundaVersion = toolx.V87
	svc, err := cluster.New(cfg, &http.Client{}, slog.Default())
	require.NoError(t, err)
	require.NotNil(t, svc)
}

func TestFactory_V88(t *testing.T) {
	cfg := testConfig()
	cfg.App.CamundaVersion = toolx.V88
	svc, err := cluster.New(cfg, &http.Client{}, slog.Default())
	require.NoError(t, err)
	require.NotNil(t, svc)
}

func TestFactory_Unknown(t *testing.T) {
	cfg := testConfig()
	cfg.App.CamundaVersion = "v0"
	svc, err := cluster.New(cfg, &http.Client{}, slog.Default())
	require.Error(t, err)
	require.Nil(t, svc)
	require.Contains(t, err.Error(), "unknown API version")
}
