package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestConfigHelp_ExplainsEffectiveConfigurationWorkflow(t *testing.T) {
	output := executeRootForTest(t, "config", "--help")

	require.Contains(t, output, "Inspect and validate c8volt configuration")
	require.Contains(t, output, "effective settings c8volt will use")
	require.Contains(t, output, "`config show`")
	require.Contains(t, output, "./c8volt config show")
	require.Contains(t, output, "./c8volt config show --template")
}

func TestConfigShowHelp_ExplainsEffectiveConfigExamples(t *testing.T) {
	output := executeRootForTest(t, "config", "show", "--help")

	require.Contains(t, output, "Show the effective configuration with sensitive values sanitized")
	require.Contains(t, output, "flag > env > profile > base config > default")
	require.Contains(t, output, "./c8volt --config ./config.yaml config show --validate")
	require.Contains(t, output, "C8VOLT_AUTH_MODE=oauth2 ./c8volt --config ./config.yaml config show --validate")
}

// Verifies config show surfaces invalid effective configuration through the shared failure model.
func TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfig(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfigHelper", nil)
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
	t.Setenv("C8VOLT_HTTP_TIMEOUT", "18s")

	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
http:
  timeout: 30s
profiles:
  dev:
    app:
      tenant: profile-dev
    apis:
      camunda_api:
        base_url: http://dev.example.test
    http:
      timeout: 9s
  prod:
    app:
      tenant: profile-prod
    apis:
      camunda_api:
        base_url: http://prod.example.test
    http:
      timeout: 7s
`)

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
	require.NoError(t, root.PersistentFlags().Set("profile", "prod"))
	require.NoError(t, root.PersistentFlags().Set("tenant", "flag-tenant"))
	require.NoError(t, root.PersistentFlags().Set("timeout", "45s"))

	v := viper.New()
	bindings, err := initViper(v, root)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)

	require.Equal(t, "prod", cfg.ActiveProfile)
	require.Equal(t, "flag-tenant", cfg.App.Tenant)
	require.Equal(t, "45s", cfg.HTTP.Timeout)
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

func TestRetrieveAndNormalizeConfig_EnvironmentTenantWinsWhenRootFlagIsUnset(t *testing.T) {
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
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`)

	cfg := resolveCommandConfigForTest(t, getCmd, cfgPath, nil)

	require.Equal(t, "dev", cfg.ActiveProfile)
	require.Equal(t, "env-tenant", cfg.App.Tenant)
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestRetrieveAndNormalizeConfig_ProfileTenantWinsWhenFlagAndEnvAreUnset(t *testing.T) {
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
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`)

	cfg := resolveCommandConfigForTest(t, walkCmd, cfgPath, nil)

	require.Equal(t, "dev", cfg.ActiveProfile)
	require.Equal(t, "profile-tenant", cfg.App.Tenant)
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestRetrieveAndNormalizeConfig_BaseConfigTenantWinsWhenNoHigherPrecedenceSourceExists(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
`)

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))

	v := viper.New()
	bindings, err := initViper(v, root)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)

	require.Empty(t, cfg.ActiveProfile)
	require.Equal(t, "base-tenant", cfg.App.Tenant)
	require.Equal(t, "http://base.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

// Verifies the critical baseline settings resolve identically across the audited command surface.
func TestRetrieveAndNormalizeConfig_CriticalBaselineSettingsStayAlignedAcrossCommands(t *testing.T) {
	t.Setenv("C8VOLT_ACTIVE_PROFILE", "ignored-env-profile")
	t.Setenv("C8VOLT_AUTH_MODE", "oauth2")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_ID", "env-client")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_SECRET", "env-secret")
	t.Setenv("C8VOLT_AUTH_OAUTH2_SCOPES_CAMUNDA_API", "env-scope")

	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: cookie
  oauth2:
    token_url: http://token.example.test
    client_id: base-client
    client_secret: base-secret
    scopes:
      camunda_api: profile-scope
apis:
  camunda_api:
    base_url: http://base.example.test
    require_scope: true
profiles:
  dev:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`)

	testCases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "config-show", cmd: configShowCmd},
		{name: "get", cmd: getCmd},
		{name: "cancel", cmd: cancelCmd},
		{name: "delete", cmd: deleteCmd},
		{name: "deploy", cmd: deployCmd},
		{name: "expect", cmd: expectCmd},
		{name: "run", cmd: runCmd},
		{name: "walk", cmd: walkCmd},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := resolveCommandConfigForTest(t, tc.cmd, cfgPath, func(cmd *cobra.Command) {
				require.NoError(t, Root().PersistentFlags().Set("tenant", "flag-tenant"))
			})

			require.Equal(t, "dev", cfg.ActiveProfile)
			require.Equal(t, "flag-tenant", cfg.App.Tenant)
			require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
			require.Equal(t, config.ModeOAuth2, cfg.Auth.Mode)
			require.Equal(t, "env-client", cfg.Auth.OAuth2.ClientID)
			require.Equal(t, "env-secret", cfg.Auth.OAuth2.ClientSecret)
			require.Equal(t, "env-scope", cfg.Auth.OAuth2.Scope(config.CamundaApiKeyConst))
		})
	}
}

func TestExecute_ProfileFlagMissingProfileUsesInvalidInputFailureModel(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `active_profile: dev
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
profiles:
  dev:
    app:
      tenant: dev-tenant
`)

	output, err := testx.RunCmdSubprocess(t, "TestExecute_ProfileFlagMissingProfileUsesInvalidInputFailureModelHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `profile not found: "missing"`)
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

func TestExecute_ProfileFlagMissingProfileUsesInvalidInputFailureModelHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--profile", "missing", "config", "show"}

	Execute()
}

func writeBackoffPrecedenceConfig(t *testing.T) string {
	t.Helper()

	return writeRawTestConfig(t, `active_profile: dev
app:
  backoff:
    timeout: 12s
    max_retries: 2
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
profiles:
  dev:
    app:
      backoff:
        timeout: 9s
        max_retries: 4
`)
}

func resolveCommandConfigForTest(t *testing.T, cmd *cobra.Command, cfgPath string, configure func(*cobra.Command)) *config.Config {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
	require.NoError(t, root.PersistentFlags().Set("profile", "dev"))
	if configure != nil {
		configure(cmd)
	}

	v := viper.New()
	bindings, err := initViper(v, cmd)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)
	return cfg
}

func writeRawTestConfig(t *testing.T, content string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(strings.TrimLeft(content, "\n")), 0o600))
	return cfgPath
}
