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
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Verifies NewCli maps missing bootstrap context to the shared local-precondition error class.
func TestNewCliNormalizesMissingBootstrapContext(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	_, _, _, err := NewCli(cmd)
	require.Error(t, err)
	require.Equal(t, ferrors.ClassLocalPrecondition, ferrors.Classify(err))
}

// Verifies NewCli maps unsupported client construction to the shared unsupported error class.
func TestNewCliNormalizesUnsupportedClientConstruction(t *testing.T) {
	cfg := &config.Config{
		App: config.App{
			CamundaVersion: toolx.V89,
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

	_, _, _, err = NewCli(cmd)
	require.Error(t, err)
	require.Equal(t, ferrors.ClassUnsupported, ferrors.Classify(err))
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

	cmd := exec.Command(os.Args[0], "-test.run=TestExecute_ConfigValidationFailureUsesSharedFailureModelHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed: normalize config:")
}

// Verifies unsupported API versions are surfaced through the shared unsupported-capability failure model.
func TestExecute_UnsupportedVersionUsesSharedFailureModel(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.9")

	cmd := exec.Command(os.Args[0], "-test.run=TestExecute_UnsupportedVersionUsesSharedFailureModelHelper")
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+cfgPath,
	)

	output, err := cmd.CombinedOutput()
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "unknown API version")
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

	cmd := exec.Command(os.Args[0], "-test.run=TestExecute_ConfigFlagOverridesDefaultSearchPathHelper")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		"C8VOLT_TEST_CONFIG="+explicitCfgPath,
	)

	output, err := cmd.CombinedOutput()
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

// Helper-process entrypoint for unsupported-version failure normalization coverage.
func TestExecute_UnsupportedVersionUsesSharedFailureModelHelper(t *testing.T) {
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
