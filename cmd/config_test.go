package cmd

import (
	"context"
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
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
