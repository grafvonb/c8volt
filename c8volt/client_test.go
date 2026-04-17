package c8volt

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

func TestNew_V89IsAdvertisedButNotYetRuntimeSupported(t *testing.T) {
	t.Parallel()

	cfg := config.New()
	cfg.App.CamundaVersion = toolx.V89
	cfg.APIs.Camunda.BaseURL = "http://localhost:8080/v2"

	cli, err := New(
		WithConfig(cfg),
		WithHTTPClient(&http.Client{}),
		WithLogger(slog.Default()),
	)

	require.Error(t, err)
	require.Nil(t, cli)
	require.ErrorIs(t, err, services.ErrUnknownAPIVersion)
	require.Contains(t, err.Error(), "\"8.9\"")
	require.Contains(t, err.Error(), toolx.ImplementedCamundaVersionsString())
}
