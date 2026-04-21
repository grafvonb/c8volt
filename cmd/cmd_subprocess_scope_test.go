package cmd

import (
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestExecute_ExplicitEmptyAuthModeDoesNotFallBackToLowerPrecedence(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `auth:
  mode: oauth2
  oauth2:
    token_url: http://token.example.test
    client_id: client
    client_secret: secret
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
`)

	output, err := testx.RunCmdSubprocess(t, "TestExecute_ExplicitEmptyAuthModeDoesNotFallBackToLowerPrecedenceHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_AUTH_MODE":   "",
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), `mode: invalid value ""`)
}

func TestExecute_ExplicitEmptyAuthModeDoesNotFallBackToLowerPrecedenceHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "show", "--validate"}

	Execute()
}
