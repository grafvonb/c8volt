package cmd

import (
	"context"
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

func TestNewCliNormalizesMissingBootstrapContext(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())

	_, _, _, err := NewCli(cmd)
	require.Error(t, err)
	require.Equal(t, ferrors.ClassLocalPrecondition, ferrors.Classify(err))
}

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

func TestNormalizeBootstrapErrorMapsCommandValidationToInvalidInput(t *testing.T) {
	err := normalizeBootstrapError(invalidFlagValuef("resource lookup requires a non-empty --id"))

	require.Error(t, err)
	require.Equal(t, ferrors.ClassInvalidInput, ferrors.Classify(err))
}

func TestExecute_ConfigValidationFailureUsesSharedFailureModelHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "topology"}

	Execute()
}
