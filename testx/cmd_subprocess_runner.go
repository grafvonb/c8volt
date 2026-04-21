package testx

import (
	"os"
	"os/exec"
	"testing"
)

func RunCmdSubprocess(t *testing.T, scopeTestName string, env map[string]string) ([]byte, error) {
	t.Helper()
	return RunCmdSubprocessInDir(t, scopeTestName, "", env)
}

func RunCmdSubprocessInDir(t *testing.T, scopeTestName string, dir string, env map[string]string) ([]byte, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run="+scopeTestName)
	if dir != "" {
		cmd.Dir = dir
	}
	mergedEnv := append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	for k, v := range env {
		mergedEnv = append(mergedEnv, k+"="+v)
	}
	cmd.Env = mergedEnv

	return cmd.CombinedOutput()
}
