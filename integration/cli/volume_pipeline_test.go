// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeVolumeStdinKeys(t *testing.T, name string, keys []string) string {
	t.Helper()
	path := filepath.Join(suite.workDir, "data", sanitizeEvidenceName(name)+".keys")
	data := strings.Join(keys, "\n")
	if data != "" {
		data += "\n"
	}
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatalf("write stdin keys: %v", err)
	}
	return path
}

func runC8VoltWithInput(t *testing.T, scenarioName string, input string, args ...string) commandResult {
	t.Helper()
	if err := rejectExplicitConfigArgs(args); err != nil {
		t.Fatalf("%v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultCommandTimeout)
	defer cancel()

	started := time.Now().UTC()
	cmd := exec.CommandContext(ctx, suite.binPath, args...)
	cmd.Dir = suite.workDir
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	cmd.Stdin = strings.NewReader(input)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	finished := time.Now().UTC()

	exitCode := 0
	if err != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}

	result := commandResult{
		Args:       append([]string(nil), args...),
		StdoutPath: writeLogFile(t, scenarioName, "stdout", stdout.String()),
		StderrPath: writeLogFile(t, scenarioName, "stderr", stderr.String()),
		ExitCode:   exitCode,
		StartedAt:  started,
		FinishedAt: finished,
		Stdout:     stdout.String(),
		Stderr:     stderr.String(),
		Err:        err,
	}
	logCommandResult(t, scenarioName, result)
	return result
}
