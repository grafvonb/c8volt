package resource_test

import (
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/resource"
	v87 "github.com/grafvonb/c8volt/internal/services/resource/v87"
	v88 "github.com/grafvonb/c8volt/internal/services/resource/v88"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

func testConfig() *config.Config {
	return &config.Config{
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "http://localhost:8080/v2",
			},
		},
	}
}

func TestFactory_SupportedVersions(t *testing.T) {
	tests := []struct {
		name    string
		version toolx.CamundaVersion
		assert  func(*testing.T, resource.API)
	}{
		{
			name:    "v87",
			version: toolx.V87,
			assert: func(t *testing.T, svc resource.API) {
				require.IsType(t, &v87.Service{}, svc)
			},
		},
		{
			name:    "v88",
			version: toolx.V88,
			assert: func(t *testing.T, svc resource.API) {
				require.IsType(t, &v88.Service{}, svc)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig()
			cfg.App.CamundaVersion = tt.version

			svc, err := resource.New(cfg, &http.Client{}, slog.Default())

			require.NoError(t, err)
			require.NotNil(t, svc)
			tt.assert(t, svc)
		})
	}
}

func TestFactory_UnknownVersion(t *testing.T) {
	cfg := testConfig()
	cfg.App.CamundaVersion = "v0"

	svc, err := resource.New(cfg, &http.Client{}, slog.Default())

	require.Error(t, err)
	require.Nil(t, svc)
	require.ErrorIs(t, err, services.ErrUnknownAPIVersion)
	require.Contains(t, err.Error(), "\"unknown\"")
	require.Contains(t, err.Error(), toolx.SupportedCamundaVersionsString())
}
