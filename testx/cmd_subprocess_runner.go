// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"os"
	"os/exec"
	"regexp"
	"testing"
)

// CmdSubprocessNameEnv binds a helper subprocess to the exact test allowed to call Execute.
const CmdSubprocessNameEnv = "C8VOLT_TEST_HELPER_PROCESS_NAME"

func RunCmdSubprocess(t *testing.T, scopeTestName string, env map[string]string) ([]byte, error) {
	t.Helper()
	return RunCmdSubprocessInDir(t, scopeTestName, "", env)
}

func RunCmdSubprocessInDir(t *testing.T, scopeTestName string, dir string, env map[string]string) ([]byte, error) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(scopeTestName)+"$")
	if dir != "" {
		cmd.Dir = dir
	}
	mergedEnv := append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		CmdSubprocessNameEnv+"="+scopeTestName,
	)
	for k, v := range env {
		mergedEnv = append(mergedEnv, k+"="+v)
	}
	cmd.Env = mergedEnv

	return cmd.CombinedOutput()
}
