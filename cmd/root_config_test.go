// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestRetrieveAndNormalizeConfig_BindsAutomationFlagAndEnvironment verifies that automation mode can be
// configured through environment variables, not only through CLI flags.
func TestRetrieveAndNormalizeConfig_BindsAutomationFlagAndEnvironment(t *testing.T) {
	t.Setenv("C8VOLT_APP_AUTOMATION", "true")

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	v := viper.New()
	bindings, err := initViper(v, root)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)
	require.True(t, cfg.App.Automation)
}

// TestRetrieveAndNormalizeConfig_VersionSpecificEmptyTenantSemantics verifies root
// config loading applies Camunda 8.7 default-tenant normalization without
// converting explicitly empty or newer-version discovery configuration.
func TestRetrieveAndNormalizeConfig_VersionSpecificEmptyTenantSemantics(t *testing.T) {
	tests := []struct {
		name       string
		configYAML string
		wantTenant string
	}{
		{
			name: "v87 omitted tenant defaults to default tenant",
			configYAML: `
app:
  camunda_version: "8.7"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			wantTenant: config.DefaultTenant,
		},
		{
			name: "v87 explicit empty tenant stays unfiltered",
			configYAML: `
app:
  camunda_version: "8.7"
  tenant: ""
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
		},
		{
			name: "v88 omitted tenant stays unfiltered",
			configYAML: `
app:
  camunda_version: "8.8"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
		},
		{
			name: "v89 omitted tenant stays unfiltered",
			configYAML: `
app:
  camunda_version: "8.9"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
		},
		{
			name: "v810 omitted tenant stays unfiltered",
			configYAML: `
app:
  camunda_version: "8.10"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := Root()
			resetCommandTreeFlags(root)
			t.Cleanup(func() {
				resetCommandTreeFlags(root)
			})

			cfgPath := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tt.configYAML), 0o600))
			require.NoError(t, root.PersistentFlags().Set("config", cfgPath))

			v := viper.New()
			bindings, err := initViper(v, root)
			require.NoError(t, err)

			cfg, err := retrieveAndNormalizeConfig(v, bindings)
			require.NoError(t, err)
			require.Equal(t, tt.wantTenant, cfg.App.Tenant)
		})
	}
}

// TestApplyAllTenantsOverride_ClearsConfiguredTenantSources verifies the
// command-line-only all-tenants selection runs after normal config precedence.
func TestApplyAllTenantsOverride_ClearsConfiguredTenantSources(t *testing.T) {
	tests := []struct {
		name           string
		configYAML     string
		envTenant      string
		allTenantsFlag *string
		wantTenant     string
		wantProvenance tenantOverrideProvenance
	}{
		{
			name: "base config tenant",
			configYAML: `
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			allTenantsFlag: stringPtr("true"),
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: "base-tenant",
				AllTenants:         true,
			},
		},
		{
			name: "profile tenant",
			configYAML: `
active_profile: dev
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`,
			allTenantsFlag: stringPtr("true"),
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: "profile-tenant",
				AllTenants:         true,
			},
		},
		{
			name:      "environment tenant",
			envTenant: "env-tenant",
			configYAML: `
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			allTenantsFlag: stringPtr("true"),
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: "env-tenant",
				AllTenants:         true,
			},
		},
		{
			name: "already empty configured tenant",
			configYAML: `
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			allTenantsFlag: stringPtr("true"),
		},
		{
			name: "explicit false leaves named tenant",
			configYAML: `
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			allTenantsFlag: stringPtr("false"),
			wantTenant:     "base-tenant",
		},
		{
			name: "absent flag leaves named tenant",
			configYAML: `
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			wantTenant: "base-tenant",
		},
		{
			name: "v87 default tenant after normalization",
			configYAML: `
app:
  camunda_version: "8.7"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			allTenantsFlag: stringPtr("true"),
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: config.DefaultTenant,
				AllTenants:         true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envTenant != "" {
				t.Setenv("C8VOLT_APP_TENANT", tt.envTenant)
			}
			root := Root()
			resetCommandTreeFlags(root)
			t.Cleanup(func() {
				resetCommandTreeFlags(root)
			})

			cfgPath := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tt.configYAML), 0o600))
			require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
			if tt.allTenantsFlag != nil {
				require.NoError(t, root.PersistentFlags().Set("all-tenants", *tt.allTenantsFlag))
			}

			v := viper.New()
			bindings, err := initViper(v, root)
			require.NoError(t, err)

			cfg, err := retrieveAndNormalizeConfig(v, bindings)
			require.NoError(t, err)
			provenance := applyAllTenantsOverride(cfg)

			require.Equal(t, tt.wantTenant, cfg.App.Tenant)
			require.Equal(t, tt.wantProvenance, provenance)
		})
	}
}

// TestTenantOverrideProvenanceFromConfigTracksExplicitTenantFlagTransitions
// verifies root config setup preserves the pre-flag tenant and explicit flag
// value without changing the resolved tenant used by commands.
func TestTenantOverrideProvenanceFromConfigTracksExplicitTenantFlagTransitions(t *testing.T) {
	tests := []struct {
		name           string
		configYAML     string
		tenantFlag     *string
		wantTenant     string
		wantProvenance tenantOverrideProvenance
	}{
		{
			name: "absent flag records no provenance",
			configYAML: `
app:
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			wantTenant: "tenant-a",
		},
		{
			name: "equal explicit flag preserves both values",
			configYAML: `
app:
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			tenantFlag: stringPtr("tenant-a"),
			wantTenant: "tenant-a",
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				ExplicitTenantID:   "tenant-a",
				Explicit:           true,
			},
		},
		{
			name: "named to empty keeps configured tenant",
			configYAML: `
app:
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			tenantFlag: stringPtr(""),
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				ExplicitTenantID:   "",
				Explicit:           true,
			},
		},
		{
			name: "named to different keeps explicit tenant",
			configYAML: `
app:
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			tenantFlag: stringPtr("tenant-b"),
			wantTenant: "tenant-b",
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: "tenant-a",
				ExplicitTenantID:   "tenant-b",
				Explicit:           true,
			},
		},
		{
			name: "empty to named records empty configured tenant",
			configYAML: `
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`,
			tenantFlag: stringPtr("tenant-a"),
			wantTenant: "tenant-a",
			wantProvenance: tenantOverrideProvenance{
				ConfiguredTenantID: "",
				ExplicitTenantID:   "tenant-a",
				Explicit:           true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := Root()
			resetCommandTreeFlags(root)
			t.Cleanup(func() {
				resetCommandTreeFlags(root)
			})

			cfgPath := filepath.Join(t.TempDir(), "config.yaml")
			require.NoError(t, os.WriteFile(cfgPath, []byte(tt.configYAML), 0o600))
			require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
			if tt.tenantFlag != nil {
				require.NoError(t, root.PersistentFlags().Set("tenant", *tt.tenantFlag))
			}

			v := viper.New()
			bindings, err := initViper(v, root)
			require.NoError(t, err)

			cfg, err := retrieveAndNormalizeConfig(v, bindings)
			require.NoError(t, err)
			require.Equal(t, tt.wantTenant, cfg.App.Tenant)
			require.Equal(t, tt.wantProvenance, tenantOverrideProvenanceFromConfig(v, bindings, cfg))
		})
	}
}

// TestAutomationModeEnabled_PrefersResolvedConfigContext ensures runtime decisions read the resolved
// config placed on the command context, even when the raw persistent flag value says otherwise.
func TestAutomationModeEnabled_PrefersResolvedConfigContext(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("automation", "false"))

	cfg := config.New()
	cfg.App.Automation = true
	root.SetContext(cfg.ToContext(context.Background()))

	require.True(t, automationModeEnabled(root))
}

// TestMissingConfigHint_PrefersLocalExampleConfigWhenPresent keeps the bootstrap error helpful for new
// users by pointing at a nearby config.example.yaml when one exists.
func TestMissingConfigHint_PrefersLocalExampleConfigWhenPresent(t *testing.T) {
	prevWD, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.example.yaml"), []byte("apis: {}\n"), 0o600))
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	got := missingConfigHint()
	require.Contains(t, got, `found "config.example.yaml" in the current directory`)
	require.Contains(t, got, "config show --validate")
}

// TestMissingConfigHint_FallsBackToTemplateAdviceWhenNoLocalExampleExists covers the no-local-example path,
// where the best recovery hint is to generate a template with config show --template.
func TestMissingConfigHint_FallsBackToTemplateAdviceWhenNoLocalExampleExists(t *testing.T) {
	prevWD, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	got := missingConfigHint()
	require.Contains(t, got, "config show --template")
	require.NotContains(t, got, "config.example.yaml")
}

// stringPtr keeps table-driven root-config tests readable when an explicit
// empty flag value must be distinguished from an absent flag.
func stringPtr(value string) *string {
	return &value
}
