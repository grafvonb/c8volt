package cmd

import (
	"context"
	"os"
	"os/exec"
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
