package testx

import (
	"os"
	"os/exec"
	"testing"
)

func RunCmdSubprocess(t *testing.T, scopeTestName string, env map[string]string) ([]byte, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run="+scopeTestName)
	mergedEnv := append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	for k, v := range env {
		mergedEnv = append(mergedEnv, k+"="+v)
	}
	cmd.Env = mergedEnv

	return cmd.CombinedOutput()
}
