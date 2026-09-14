// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const (
	userTaskTerminalArgsEnv       = "C8VOLT_USER_TASK_TERMINAL_ARGS"
	userTaskTerminalConfiguredEnv = "C8VOLT_USER_TASK_TERMINAL_CONFIGURED_STDERR"
)

// TestGetUserTaskPagingTerminal verifies real-terminal paging decisions, stream
// routing, prompt suppression, limit handling, and terminal stdin key policy.
func TestGetUserTaskPagingTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runGetUserTaskPagingTerminalHelper(t)
		os.Exit(0)
	}

	firstPrompt := userTaskTerminalPrompt(1, 1)
	secondPrompt := userTaskTerminalPrompt(1, 2)
	threePages := []string{
		userTaskSearchResponse(3, true, "cursor-a", "2251799815391233"),
		userTaskSearchResponse(3, true, "cursor-b", "2251799815391234"),
		userTaskSearchResponse(3, false, "", "2251799815391235"),
	}
	tests := []struct {
		name             string
		responses        []string
		args             []string
		exchanges        []testx.CmdTerminalExchange
		configuredStderr bool
		wantStdout       string
		wantStderr       string
		wantStderrPart   string
		wantRequests     int
		wantExitCode     int
	}{
		{
			name:      "yes yes uses configured stderr and reaches completion",
			responses: threePages,
			args:      []string{"--keys-only", "get", "ut", "--batch-size", "1"},
			exchanges: []testx.CmdTerminalExchange{
				{Prompt: firstPrompt, Response: "y"},
				{Prompt: secondPrompt, Response: "yes"},
			},
			configuredStderr: true,
			wantStdout:       "2251799815391233\n2251799815391234\n2251799815391235\n",
			wantStderr:       firstPrompt + secondPrompt,
			wantRequests:     3,
		},
		{
			name:         "decline uses inherited stderr and stops",
			responses:    threePages,
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1"},
			exchanges:    []testx.CmdTerminalExchange{{Prompt: firstPrompt, Response: "n"}},
			wantStdout:   "2251799815391233\n",
			wantStderr:   firstPrompt,
			wantRequests: 1,
		},
		{
			name:         "unexpected answer stops",
			responses:    threePages,
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1"},
			exchanges:    []testx.CmdTerminalExchange{{Prompt: firstPrompt, Response: "later"}},
			wantStdout:   "2251799815391233\n",
			wantStderr:   firstPrompt,
			wantRequests: 1,
		},
		{
			name:         "EOF stops",
			responses:    threePages,
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1"},
			exchanges:    []testx.CmdTerminalExchange{{Prompt: firstPrompt, EndOfInput: true}},
			wantStdout:   "2251799815391233\n",
			wantStderr:   firstPrompt,
			wantRequests: 1,
		},
		{
			name: "limit prompts before boundary and not after",
			responses: []string{
				userTaskSearchResponse(3, true, "cursor-a", "2251799815391233"),
				userTaskSearchResponse(3, true, "cursor-b", "2251799815391234"),
			},
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1", "--limit", "2"},
			exchanges:    []testx.CmdTerminalExchange{{Prompt: firstPrompt, Response: "yes"}},
			wantStdout:   "2251799815391233\n2251799815391234\n",
			wantStderr:   firstPrompt,
			wantRequests: 2,
		},
		{
			name:         "completed empty search is prompt free",
			responses:    []string{userTaskSearchResponse(0, false, "")},
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1"},
			wantStdout:   "",
			wantStderr:   "",
			wantRequests: 1,
		},
		{
			name: "sparse page is prompt free",
			responses: []string{
				userTaskSearchResponse(1, false, "cursor-a"),
				userTaskSearchResponse(1, false, "", "2251799815391233"),
			},
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1"},
			wantStdout:   "2251799815391233\n",
			wantStderr:   "",
			wantRequests: 2,
		},
		{
			name:         "json is prompt free",
			responses:    threePages,
			args:         []string{"--json", "get", "ut", "--batch-size", "1"},
			wantRequests: 3,
		},
		{
			name:         "automation is prompt free",
			responses:    threePages,
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1", "--automation"},
			wantStdout:   "2251799815391233\n2251799815391234\n2251799815391235\n",
			wantRequests: 3,
		},
		{
			name:         "auto confirm is prompt free",
			responses:    threePages,
			args:         []string{"--keys-only", "get", "ut", "--batch-size", "1", "--auto-confirm"},
			wantStdout:   "2251799815391233\n2251799815391234\n2251799815391235\n",
			wantRequests: 3,
		},
		{
			name:         "terminal stdin does not infer keys or block search",
			responses:    []string{userTaskSearchResponse(0, false, "")},
			args:         []string{"--keys-only", "get", "ut"},
			wantStdout:   "",
			wantStderr:   "",
			wantRequests: 1,
		},
		{
			name:           "explicit dash rejects terminal stdin",
			args:           []string{"--keys-only", "get", "ut", "-"},
			wantStderrPart: "invalid flag value: '-' requires piped/redirected stdin",
			wantRequests:   0,
			wantExitCode:   exitcode.InvalidArgs,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(index int, _ map[string]any) string {
				require.Less(t, index, len(tt.responses))
				return tt.responses[index]
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			configuredPath := filepath.Join(t.TempDir(), "configured-stderr.txt")
			args := append([]string{"--config", configPath, "--tenant", "tenant-a"}, tt.args...)
			encodedArgs, err := json.Marshal(args)
			require.NoError(t, err)

			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestGetUserTaskPagingTerminal",
				Env: map[string]string{
					userTaskTerminalArgsEnv:       string(encodedArgs),
					userTaskTerminalConfiguredEnv: configuredTerminalStderrPath(tt.configuredStderr, configuredPath),
				},
				Exchanges: tt.exchanges,
				Timeout:   3 * time.Second,
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.True(t, result.Supported, result.UnsupportedReason)
			if tt.wantExitCode == 0 {
				require.NoError(t, result.Err, result.Stderr)
			} else {
				var exitErr *exec.ExitError
				require.ErrorAs(t, result.Err, &exitErr)
				require.Equal(t, tt.wantExitCode, exitErr.ExitCode())
			}
			if tt.name == "json is prompt free" {
				require.Contains(t, result.Stdout, `"outcome": "succeeded"`)
				require.Contains(t, result.Stdout, `"total": 3`)
			} else {
				require.Equal(t, tt.wantStdout, result.Stdout)
			}
			if tt.wantStderrPart != "" {
				require.Contains(t, result.Stderr, tt.wantStderrPart)
			} else {
				require.Equal(t, tt.wantStderr, result.Stderr)
			}
			require.Len(t, requests.snapshot(t), tt.wantRequests)
			require.Equal(t, len(tt.exchanges), strings.Count(result.Stderr, "Continue? [y/N]: "))
			require.NotContains(t, result.Stdout, "Fetched")
			require.NotContains(t, result.Stdout, "Continue?")
			if tt.configuredStderr {
				configured, readErr := os.ReadFile(configuredPath)
				require.NoError(t, readErr)
				require.Equal(t, tt.wantStderr, string(configured))
			}
		})
	}
}

// runGetUserTaskPagingTerminalHelper executes one isolated command scenario
// with terminal stdin and either leaf-configured or inherited stderr.
func runGetUserTaskPagingTerminalHelper(t *testing.T) {
	require.Equal(t, "TestGetUserTaskPagingTerminal", os.Getenv(testx.CmdSubprocessNameEnv))
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv(userTaskTerminalArgsEnv)), &args))

	root := Root()
	resetCommandTreeFlags(root)
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	getUserTaskCmd.SetErr(nil)
	if configuredPath := os.Getenv(userTaskTerminalConfiguredEnv); configuredPath != "" {
		configured, err := os.Create(configuredPath)
		require.NoError(t, err)
		defer configured.Close()
		getUserTaskCmd.SetErr(io.MultiWriter(os.Stderr, configured))
	}
	root.SetArgs(args)
	_ = root.Execute()
}

// userTaskTerminalPrompt returns the exact shared default-no paging prompt.
func userTaskTerminalPrompt(pageCount, loaded int) string {
	return fmt.Sprintf("Fetched %d user task(s) on this page (%d loaded). More matching user tasks remain.\nContinue? [y/N]: ", pageCount, loaded)
}

// configuredTerminalStderrPath selects a child-only configured destination.
func configuredTerminalStderrPath(configured bool, path string) string {
	if configured {
		return path
	}
	return ""
}
