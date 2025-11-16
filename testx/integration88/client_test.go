//go:build integration

package integration88_test

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func newITClient(t *testing.T) (context.Context, c8volt.API, *config.Config, *slog.Logger) {
	t.Helper()

	cfg := testx.ITConfigFromEnv(t)
	cfg.Auth = config.Auth{
		Mode: "none",
	}

	require.NoError(t, cfg.Normalize())
	require.NoError(t, cfg.Validate())

	log := testx.Logger(t)
	ctx := testx.ITCtx(t, 30*time.Second)
	httpClient := testx.ITHttpClient(t, ctx, cfg, log)

	api, err := c8volt.New(
		c8volt.WithConfig(cfg),
		c8volt.WithHTTPClient(httpClient),
		c8volt.WithLogger(log),
	)
	require.NoError(t, err)

	return ctx, api, cfg, log
}

func deployTestProcessDefinitions(t *testing.T, ctx context.Context, api c8volt.API, cfg *config.Config, log *slog.Logger) {
	t.Helper()

	all, err := embedded.List()
	require.NoError(t, err)

	var toDeploy []string
	for _, d := range all {
		if strings.Contains(d, cfg.App.CamundaVersion.FilePrefix()) {
			toDeploy = append(toDeploy, d)
		}
	}

	var units []resource.DeploymentUnitData
	for _, f := range toDeploy {
		data, err := fs.ReadFile(embedded.FS, f)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("read embedded %q: %w", f, err))
		}
		log.Debug(fmt.Sprintf("deploying embedded resource(s) %q to tenant %s", f, cfg.App.ViewTenant()))
		units = append(units, resource.DeploymentUnitData{Name: f, Data: data})
	}

	pdds, err := api.DeployProcessDefinition(ctx, cfg.App.Tenant, units)
	require.NoError(t, err)
	require.Equal(t, 5, len(pdds))
}
