package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// Verifies config show surfaces invalid effective configuration through the shared failure model.
func TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfig(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfigHelper")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "configuration is invalid")
}

// Verifies root persistent flags override env, profile, and base config for baseline settings.
func TestRetrieveAndNormalizeConfig_RootPersistentFlagsOverrideEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_ACTIVE_PROFILE", "dev")
	t.Setenv("C8VOLT_APP_TENANT", "env-tenant")

	cfgPath := writeRawTestConfig(t, `active_profile: base
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
      tenant: profile-dev
    apis:
      camunda_api:
        base_url: http://dev.example.test
  prod:
    app:
      tenant: profile-prod
    apis:
      camunda_api:
        base_url: http://prod.example.test
`)

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
	require.NoError(t, root.PersistentFlags().Set("profile", "prod"))
	require.NoError(t, root.PersistentFlags().Set("tenant", "flag-tenant"))

	v := viper.New()
	bindings, err := initViper(v, root)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)

	require.Equal(t, "prod", cfg.ActiveProfile)
	require.Equal(t, "flag-tenant", cfg.App.Tenant)
	require.Equal(t, "http://prod.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

// Verifies root bootstrap binds command-local backoff flags into the same effective-config resolver.
func TestRetrieveAndNormalizeConfig_UsesSharedResolverForCommandLocalBackoffFlags(t *testing.T) {
	cfgPath := t.TempDir() + "/config.yaml"
	content := `active_profile: dev
app:
  tenant: base-tenant
  backoff:
    timeout: 12s
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
http:
  timeout: 30s
profiles:
  dev:
    app:
      backoff:
        timeout: 9s
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
	require.NoError(t, root.PersistentFlags().Set("profile", "dev"))
	require.NoError(t, getCmd.PersistentFlags().Set("backoff-timeout", "45s"))

	v := viper.New()
	bindings, err := initViper(v, getCmd)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)

	require.Equal(t, 45*time.Second, cfg.App.Backoff.Timeout)
}

// Helper-process entrypoint for invalid effective-config failure-path validation.
func TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevValidate := flagShowConfigValidate
	prevTemplate := flagShowConfigTemplate
	t.Cleanup(func() {
		flagShowConfigValidate = prevValidate
		flagShowConfigTemplate = prevTemplate
	})

	cfg := config.New()
	flagShowConfigValidate = true
	flagShowConfigTemplate = false

	configShowCmd.SetContext(cfg.ToContext(context.Background()))
	configShowCmd.SetOut(os.Stdout)
	configShowCmd.SetErr(os.Stderr)
	configShowCmd.Run(configShowCmd, nil)
}

func writeRawTestConfig(t *testing.T, content string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))
	return cfgPath
}
