package cmd

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRootCommand_V89SupportMessagingIsUpdated(t *testing.T) {
	t.Parallel()

	root := Root()

	require.Contains(t, root.Long, "Camunda 8.7, 8.8, and 8.9")
	require.NotContains(t, root.Long, "version 8.9 is recognized by config normalization")
	require.Contains(t, root.PersistentFlags().Lookup("camunda-version").Usage, toolx.SupportedCamundaVersionsString())
}

// Verifies NewCli maps missing bootstrap context to the shared local-precondition error class.
func TestNewCliNormalizesMissingBootstrapContext(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	_, _, _, err := NewCli(cmd)
	require.Error(t, err)
	require.Equal(t, ferrors.ClassLocalPrecondition, ferrors.Classify(err))
}

// Verifies NewCli now constructs a v8.9-backed client instead of rejecting the version at bootstrap.
func TestNewCliConstructsSupportedV89Client(t *testing.T) {
	cfg := &config.Config{
		App: config.App{
			CamundaVersion: toolx.V89,
		},
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL: "http://127.0.0.1:1",
			},
		},
		HTTP: config.HTTP{
			Timeout: "30s",
		},
	}
	ctx := cfg.ToContext(context.Background())

	httpSvc, err := httpc.New(cfg, nil)
	require.NoError(t, err)

	cmd := &cobra.Command{}
	cmd.SetContext(httpSvc.ToContext(ctx))

	_, _, cli, err := NewCli(cmd)
	require.NoError(t, err)
	require.NotNil(t, cli)
}

// Verifies execute-time config validation failures use the shared failure model and exit behavior.
func TestExecute_ConfigValidationFailureUsesSharedFailureModel(t *testing.T) {
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	content := `app:
  camunda_version: "9.9"
auth:
  mode: none
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	output, err := testx.RunCmdSubprocess(t, "TestExecute_ConfigValidationFailureUsesSharedFailureModelHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed: normalize config:")
}

// Verifies v8.9 command execution now proceeds past bootstrap and surfaces downstream runtime failures normally.
func TestExecute_V89RuntimeFailuresUseSharedFailureModel(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.9")

	output, err := testx.RunCmdSubprocess(t, "TestExecute_V89RuntimeFailuresUseSharedFailureModelHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "internal error")
	require.Contains(t, string(output), "get cluster topology")
}

// Verifies an explicit --config path wins over default search-path config discovery.
func TestExecute_ConfigFlagOverridesDefaultSearchPath(t *testing.T) {
	dir := t.TempDir()
	defaultCfgPath := filepath.Join(dir, "config.yaml")
	explicitCfgPath := filepath.Join(dir, "explicit.yaml")

	defaultCfg := `app:
  tenant: default-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://default.example.test
`
	explicitCfg := `app:
  tenant: explicit-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://explicit.example.test
`
	require.NoError(t, os.WriteFile(defaultCfgPath, []byte(defaultCfg), 0o600))
	require.NoError(t, os.WriteFile(explicitCfgPath, []byte(explicitCfg), 0o600))

	output, err := testx.RunCmdSubprocessInDir(t, "TestExecute_ConfigFlagOverridesDefaultSearchPathHelper", dir, map[string]string{
		"C8VOLT_TEST_CONFIG": explicitCfgPath,
	})
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "tenant: explicit-tenant")
	require.NotContains(t, string(output), "tenant: default-tenant")
}

// Verifies bootstrap normalization maps command-validation sentinels to invalid-input classification.
func TestNormalizeBootstrapErrorMapsCommandValidationToInvalidInput(t *testing.T) {
	err := normalizeBootstrapError(invalidFlagValuef("resource lookup requires a non-empty --id"))

	require.Error(t, err)
	require.Equal(t, ferrors.ClassInvalidInput, ferrors.Classify(err))
}

// Verifies bootstrap normalization keeps already-normalized shared failures stable.
func TestNormalizeBootstrapErrorPreservesNormalizedFailures(t *testing.T) {
	t.Parallel()

	err := ferrors.WrapClass(ferrors.ErrLocalPrecondition, errors.New("loading configuration: missing base url"))
	got := normalizeBootstrapError(err)

	require.Equal(t, "local precondition failed: loading configuration: missing base url", got.Error())
	require.Equal(t, ferrors.ClassLocalPrecondition, ferrors.Classify(got))
	require.Equal(t, exitcode.Error, ferrors.ExitCode(got))
}

// Verifies shared command-validation sentinels normalize to the invalid-input class.
func TestNormalizeCommandErrorMapsSharedValidationSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
	}{
		{name: "invalid flag value", err: invalidFlagValuef("boom")},
		{name: "forbidden flag combination", err: forbiddenFlagCombinationf("boom")},
		{name: "missing dependent flags", err: missingDependentFlagsf("boom")},
		{name: "mutually exclusive flags", err: mutuallyExclusiveFlagsf("boom")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, ferrors.ClassInvalidInput, ferrors.Classify(normalizeCommandError(tt.err)))
		})
	}
}

// Verifies command normalization does not restack an already-normalized shared failure.
func TestNormalizeCommandErrorPreservesNormalizedFailures(t *testing.T) {
	t.Parallel()

	err := ferrors.WrapClass(ferrors.ErrInvalidInput, errors.New("invalid flag value: boom"))
	got := normalizeCommandError(err)

	require.Equal(t, "invalid input: invalid flag value: boom", got.Error())
	require.Equal(t, ferrors.ClassInvalidInput, ferrors.Classify(got))
	require.Equal(t, exitcode.InvalidArgs, ferrors.ExitCode(got))
}

// Verifies bootstrap normalization rule-table mappings for known and fallback error cases.
func TestNormalizeBootstrapErrorMapsSharedBootstrapRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		err       error
		wantClass ferrors.Class
	}{
		{
			name:      "command validation keeps invalid input",
			err:       ErrInvalidFlagValue,
			wantClass: ferrors.ClassInvalidInput,
		},
		{
			name:      "missing config becomes local precondition",
			err:       config.ErrNoConfigInContext,
			wantClass: ferrors.ClassLocalPrecondition,
		},
		{
			name:      "http service bootstrap becomes local precondition",
			err:       httpc.ErrNoHttpServiceInContext,
			wantClass: ferrors.ClassLocalPrecondition,
		},
		{
			name:      "missing profile becomes invalid input",
			err:       config.ErrProfileNotFound,
			wantClass: ferrors.ClassInvalidInput,
		},
		{
			name:      "unmapped errors fall back to internal",
			err:       errors.New("boom"),
			wantClass: ferrors.ClassInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, tt.wantClass, ferrors.Classify(normalizeBootstrapError(tt.err)))
		})
	}
}

// Helper-process entrypoint for config-validation failure normalization coverage.
func TestExecute_ConfigValidationFailureUsesSharedFailureModelHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "topology"}

	Execute()
}

// Helper-process entrypoint for v8.9 runtime-failure normalization coverage.
func TestExecute_V89RuntimeFailuresUseSharedFailureModelHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "topology"}

	Execute()
}

// Helper-process entrypoint for explicit-config precedence over default search-path config.
func TestExecute_ConfigFlagOverridesDefaultSearchPathHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "show"}

	Execute()
}
