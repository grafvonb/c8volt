package config

import (
	"testing"
	"time"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

func TestAppNormalize_DefaultsProcessInstancePageSize(t *testing.T) {
	t.Parallel()

	app := &App{}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, int32(consts.MaxPISearchSize), app.ProcessInstancePageSize)
}

func TestAppNormalize_DefaultsMissingCamundaVersionToCurrentVersion(t *testing.T) {
	t.Parallel()

	app := &App{}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, toolx.CurrentCamundaVersion, app.CamundaVersion)
}

func TestAppNormalize_PreservesPositiveProcessInstancePageSize(t *testing.T) {
	t.Parallel()

	app := &App{ProcessInstancePageSize: 250}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, int32(250), app.ProcessInstancePageSize)
}

func TestAppNormalize_DefaultTenantForV87(t *testing.T) {
	t.Parallel()

	app := &App{CamundaVersion: toolx.V87}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, DefaultTenant, app.Tenant)
}

func TestAppNormalize_DoesNotForceDefaultTenantForV88(t *testing.T) {
	t.Parallel()

	app := &App{CamundaVersion: toolx.V88}

	err := app.Normalize()

	require.NoError(t, err)
	require.Empty(t, app.Tenant)
}

func TestAppNormalize_DoesNotForceDefaultTenantForV89AuditOnlyConfig(t *testing.T) {
	t.Parallel()

	app := &App{CamundaVersion: toolx.V89}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, toolx.V89, app.CamundaVersion)
	require.Empty(t, app.Tenant)
}

func TestAppNormalize_PreservesExplicitTenantForV87(t *testing.T) {
	t.Parallel()

	app := &App{CamundaVersion: toolx.V87, Tenant: "tenant-a"}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, "tenant-a", app.Tenant)
}

func TestAppNormalize_RejectsUnsupportedCamundaVersion(t *testing.T) {
	t.Parallel()

	app := &App{CamundaVersion: "9.9"}

	err := app.Normalize()

	require.ErrorContains(t, err, "version: unknown Camunda version: 9.9")
}

func TestAppNormalizeWithConfiguredKeys_PreservesExplicitEmptyTenantForV87(t *testing.T) {
	t.Parallel()

	app := &App{CamundaVersion: toolx.V87}

	err := app.normalizeWithConfiguredKeys(func(key string) bool {
		return key == "app.tenant"
	})

	require.NoError(t, err)
	require.Empty(t, app.Tenant)
}

func TestAppNormalizeWithConfiguredKeys_PreservesExplicitEmptyTenantForV89(t *testing.T) {
	t.Parallel()

	app := &App{CamundaVersion: toolx.V89}

	err := app.normalizeWithConfiguredKeys(func(key string) bool {
		return key == "app.tenant"
	})

	require.NoError(t, err)
	require.Equal(t, toolx.V89, app.CamundaVersion)
	require.Empty(t, app.Tenant)
}

func TestAppNormalize_PreservesExplicitBackoffTimeout(t *testing.T) {
	t.Parallel()

	app := &App{
		Backoff: BackoffConfig{
			Timeout: 45 * time.Second,
		},
	}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, 45*time.Second, app.Backoff.Timeout)
}

func TestAppValidate_RejectsInvalidExplicitBackoffAndPageSize(t *testing.T) {
	t.Parallel()

	app := &App{
		ProcessInstancePageSize: 0,
		Backoff: BackoffConfig{
			Timeout:    0,
			MaxRetries: -1,
		},
	}

	err := app.Validate()

	require.ErrorContains(t, err, "process_instance_page_size must be greater than 0")
	require.ErrorContains(t, err, "max_retries must be non-negative")
	require.ErrorContains(t, err, "timeout must be a positive duration")
}
