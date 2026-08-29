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
