// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

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

// TestAppNormalize_DefaultCamundaVersionRemainsV88 verifies V810 support does
// not alter the existing no-configuration default.
func TestAppNormalize_DefaultCamundaVersionRemainsV88(t *testing.T) {
	t.Parallel()

	app := &App{}

	err := app.Normalize()

	require.NoError(t, err)
	require.Equal(t, toolx.V88, app.CamundaVersion)
}

func TestAppNormalize_DefaultsTimezoneOffsetOutputToFalse(t *testing.T) {
	t.Parallel()

	app := &App{}

	err := app.Normalize()

	require.NoError(t, err)
	require.False(t, app.ShowTimezoneOffset)
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

// TestAppNormalize_AcceptsV810Aliases verifies configuration normalization
// routes all stable 8.10 aliases to the single canonical V810 identity.
func TestAppNormalize_AcceptsV810Aliases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input toolx.CamundaVersion
	}{
		{name: "canonical", input: "8.10"},
		{name: "numeric", input: "810"},
		{name: "compact with prefix", input: "v810"},
		{name: "dotted with prefix", input: "v8.10"},
		{name: "trim and case", input: " V8.10 "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app := &App{CamundaVersion: tt.input}

			err := app.Normalize()

			require.NoError(t, err)
			require.Equal(t, toolx.V810, app.CamundaVersion)
			require.Empty(t, app.Tenant)
		})
	}
}

// TestAppNormalize_RejectsV810SourceTags verifies Camunda source-release tags
// stay out of the operator-facing version identity contract.
func TestAppNormalize_RejectsV810SourceTags(t *testing.T) {
	t.Parallel()

	tests := []toolx.CamundaVersion{
		"8.10-alpha4",
		"8.10.0-alpha4",
		"8.10.0",
		"v8.10.0-alpha4",
	}

	for _, input := range tests {
		t.Run(string(input), func(t *testing.T) {
			t.Parallel()

			app := &App{CamundaVersion: input}

			err := app.Normalize()

			require.ErrorContains(t, err, "version: unknown Camunda version")
		})
	}
}

func TestAppTargetTenant_DefaultsEmptyTenant(t *testing.T) {
	t.Parallel()

	app := &App{}

	require.Equal(t, DefaultTenant, app.TargetTenant())
	require.Empty(t, app.Tenant)
}

func TestAppTargetTenant_PreservesConfiguredTenant(t *testing.T) {
	t.Parallel()

	app := &App{Tenant: "tenant-a"}

	require.Equal(t, "tenant-a", app.TargetTenant())
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
