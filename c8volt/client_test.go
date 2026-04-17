package c8volt

import (
	"context"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

func TestNew_V89WiresSupportedRuntime(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cfg.App.CamundaVersion = toolx.V89
	cfg.APIs.Camunda.BaseURL = "http://localhost:8080/v2"

	cli, err := New(
		WithConfig(cfg),
		WithHTTPClient(&http.Client{}),
		WithLogger(slog.Default()),
	)

	require.NoError(t, err)
	require.NotNil(t, cli)

	got, err := cli.SearchProcessInstances(context.Background(), process.ProcessInstanceFilter{}, 1)
	require.Error(t, err)
	require.Empty(t, got.Items)

	_, err = cli.GetResource(context.Background(), "resource-id-123")
	require.Error(t, err)
}
