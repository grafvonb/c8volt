// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const processDefinitionWatchTerminalModeEnv = "C8VOLT_PD_WATCH_TERMINAL_MODE"

// TestGetProcessDefinitionSuspendedTerminal exercises real terminal stdin with
// separate result/diagnostic streams for empty reads and full HTTP watch refreshes.
func TestGetProcessDefinitionSuspendedTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" && os.Getenv(testx.CmdSubprocessNameEnv) == t.Name() {
		require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
		var args []string
		require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_PD_ARGS")), &args))
		root := Root()
		resetCommandTreeFlags(root)
		resetGetProcessDefinitionCommandGlobals()
		root.SetOut(os.Stdout)
		command, _, err := root.Find([]string{"get", "process-definition"})
		require.NoError(t, err)
		capture, err := os.Create(os.Getenv("C8VOLT_TEST_PD_STDERR"))
		require.NoError(t, err)
		writer := io.MultiWriter(os.Stderr, capture)
		if os.Getenv(processDefinitionWatchTerminalModeEnv) == "configured" {
			command.SetErr(writer)
		} else {
			root.SetErr(writer)
			command.SetErr(nil)
		}
		sleeps := 0
		processDefinitionWatchSleep = func(context.Context, time.Duration) error {
			sleeps++
			if sleeps == 5 {
				return context.Canceled
			}
			return nil
		}
		root.SetArgs(args)
		require.NoError(t, root.Execute())
		require.NoError(t, capture.Close())
		os.Exit(0)
	}
	for _, destination := range []string{"configured", "inherited"} {
		for _, flags := range [][]string{nil, {"--json"}, {"--keys-only"}, {"--quiet"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}, {"--automation"}, {"--auto-confirm"}, {"--watch"}, {"--watch", "--key", "123"}} {
			t.Run(destination+"/"+strings.Join(flags, " "), func(t *testing.T) {
				watch := slices.Contains(flags, "--watch")
				fixture := &definitionStatisticsFixture{version: "8.10", empty: !watch, population: func(refresh int32) (int, int) {
					population := [5][2]int{{2, 3}, {2, 4}, {1, 5}, {1, 0}, {1, 0}}[refresh-1]
					return population[0], population[1]
				}}
				srv := fixture.server(t)
				args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "get", "process-definition", "--stat"}, flags...)
				capture := filepath.Join(t.TempDir(), "stderr")
				result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
					ScopeTestName: "TestGetProcessDefinitionSuspendedTerminal",
					Env:           map[string]string{"C8VOLT_TEST_PD_ARGS": marshalStringSliceForEnv(t, args), "C8VOLT_TEST_PD_STDERR": capture, processDefinitionWatchTerminalModeEnv: destination, "TERM": "xterm-256color", "NO_COLOR": ""},
				})
				if !result.Supported {
					t.Skip(result.UnsupportedReason)
				}
				require.NoError(t, result.Err, result.Stderr)
				captured, err := os.ReadFile(capture)
				require.NoError(t, err)
				require.Equal(t, string(captured), result.Stderr)
				if watch {
					require.EqualValues(t, 5, fixture.definitions.Load())
					require.EqualValues(t, 30, fixture.counts.Load())
					frames := strings.Split(result.Stdout, processDefinitionWatchRepaintControlSequenceForTest)[1:]
					require.Len(t, frames, 5)
					for i, tail := range []string{"[ac:2 cp:- cx:- inc:- su:3]", "[ac:2 cp:- cx:- inc:- su:4] *", "[ac:1 cp:- cx:- inc:- su:5] *", "[ac:1 cp:- cx:- inc:- su:-] *", "[ac:1 cp:- cx:- inc:- su:-]"} {
						require.Equal(t, "123 tenant invoice v1 "+tail+"\nfound: 1\n", frames[i])
					}
					require.NotContains(t, result.Stdout, "\x1b[1m", "terminal stdin must not style redirected stdout")
					require.Equal(t, 3, strings.Count(result.Stderr, "INFO 1 updated, 0 new, 0 removed\n"))
					require.Equal(t, 2, strings.Count(result.Stderr, "INFO 0 updated, 0 new, 0 removed\n"))
					require.Empty(t, watchDiagnosticsWithoutSummaries(result.Stderr))
					return
				}
				require.Empty(t, result.Stderr, "empty read must not prompt or emit diagnostics")
				require.EqualValues(t, 1, fixture.definitions.Load())
				require.Zero(t, fixture.counts.Load())
				switch {
				case slices.Contains(flags, "--json"):
					decoder := json.NewDecoder(strings.NewReader(result.Stdout))
					var envelope map[string]any
					require.NoError(t, decoder.Decode(&envelope))
					require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
					require.Equal(t, "succeeded", envelope["outcome"])
					payload := requireJSONObject(t, envelope["payload"])
					require.Empty(t, payload, "preserve omitted empty fields")
				case slices.Contains(flags, "--keys-only"):
					require.Empty(t, result.Stdout)
				case slices.Contains(flags, "--quiet"):
					require.Empty(t, result.Stdout)
				default:
					require.Equal(t, "found: 0\n", result.Stdout)
				}
			})
		}
	}
}

// TestProcessDefinitionWatchBoldCapabilityTerminal verifies the configured or
// inherited Cobra output destination, rather than terminal stdin alone, enables bold.
func TestProcessDefinitionWatchBoldCapabilityTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runProcessDefinitionWatchBoldCapabilityTerminalHelper(t)
		os.Exit(0)
	}

	tests := []struct {
		name    string
		mode    string
		term    string
		noColor string
		want    bool
	}{
		{name: "configured terminal output", mode: "configured", term: "xterm-256color", want: true},
		{name: "inherited terminal output", mode: "inherited", term: "xterm-256color", want: true},
		{name: "empty TERM", mode: "configured", want: false},
		{name: "dumb TERM", mode: "configured", term: "dumb", want: false},
		{name: "NO_COLOR", mode: "configured", term: "xterm-256color", noColor: "1", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
				ScopeTestName: "TestProcessDefinitionWatchBoldCapabilityTerminal",
				Env: map[string]string{
					processDefinitionWatchTerminalModeEnv: tt.mode,
					"TERM":                                tt.term,
					"NO_COLOR":                            tt.noColor,
				},
			})

			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.NoError(t, result.Err)
			require.Equal(t, fmt.Sprintf("enabled=%t\n", tt.want), result.Stdout)
			require.Empty(t, result.Stderr)
		})
	}
}

// TestProcessDefinitionWatchBoldCapabilityNonTerminal verifies buffers and
// regular files stay plain even when the environment otherwise allows styling.
func TestProcessDefinitionWatchBoldCapabilityNonTerminal(t *testing.T) {
	t.Setenv("TERM", "xterm-256color")
	t.Setenv("NO_COLOR", "")

	bufferCmd := &cobra.Command{Use: "process-definition"}
	bufferCmd.SetOut(&bytes.Buffer{})
	require.False(t, processDefinitionWatchBoldEnabled(bufferCmd))

	file, err := os.Create(filepath.Join(t.TempDir(), "watch-output.txt"))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, file.Close()) })
	fileCmd := &cobra.Command{Use: "process-definition"}
	fileCmd.SetOut(file)
	require.False(t, processDefinitionWatchBoldEnabled(fileCmd))
	require.False(t, processDefinitionWatchBoldEnabled(nil))
}

// runProcessDefinitionWatchBoldCapabilityTerminalHelper checks a real terminal
// descriptor while reporting the decision through separately captured stdout.
func runProcessDefinitionWatchBoldCapabilityTerminalHelper(t *testing.T) {
	require.Equal(t, "TestProcessDefinitionWatchBoldCapabilityTerminal", os.Getenv(testx.CmdSubprocessNameEnv))
	require.True(t, term.IsTerminal(int(os.Stdin.Fd())))

	cmd := &cobra.Command{Use: "process-definition"}
	switch os.Getenv(processDefinitionWatchTerminalModeEnv) {
	case "configured":
		cmd.SetOut(os.Stdin)
	case "inherited":
		parent := &cobra.Command{Use: "get"}
		parent.SetOut(os.Stdin)
		parent.AddCommand(cmd)
	default:
		t.Fatalf("unknown terminal mode %q", os.Getenv(processDefinitionWatchTerminalModeEnv))
	}

	fmt.Fprintf(os.Stdout, "enabled=%t\n", processDefinitionWatchBoldEnabled(cmd))
}

// processDefinitionTerminalCaptureWriter forwards bytes to a real terminal and
// separately captured stdout without hiding the terminal's capability descriptor.
type processDefinitionTerminalCaptureWriter struct {
	io.Writer
	terminal *os.File
}

// Fd exposes the same terminal receiving every result write.
func (w processDefinitionTerminalCaptureWriter) Fd() uintptr { return w.terminal.Fd() }

// TestGetProcessDefinitionLifecycleWatchTerminal proves real-terminal styling
// and mixed-row alignment through HTTP-backed command execution.
func TestGetProcessDefinitionLifecycleWatchTerminal(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" && os.Getenv(testx.CmdSubprocessNameEnv) == t.Name() {
		require.True(t, term.IsTerminal(int(os.Stdin.Fd())))
		root := Root()
		resetCommandTreeFlags(root)
		resetGetProcessDefinitionCommandGlobals()
		root.SetOut(processDefinitionTerminalCaptureWriter{Writer: io.MultiWriter(os.Stdin, os.Stdout), terminal: os.Stdin})
		root.SetErr(os.Stderr)
		sleeps := 0
		processDefinitionWatchSleep = func(context.Context, time.Duration) error {
			sleeps++
			if sleeps == 5 {
				return context.Canceled
			}
			return nil
		}
		root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "pd", "--watch", "--stat"})
		require.NoError(t, root.Execute())
		os.Exit(0)
	}
	for _, noColor := range []string{"", "1"} {
		t.Run("NO_COLOR="+noColor, func(t *testing.T) {
			fixture := &definitionStatisticsFixture{version: "8.10", keys: []string{"123", "456"}, lifecycle: func(read int32, key string) string {
				if key == "456" {
					return ""
				}
				return []string{"ACTIVE", "DRAINING", "DELETED", "", ""}[read-1]
			}, population: func(read int32) (int, int) {
				if read >= 3 {
					return 1, 2
				}
				return 0, 2
			}}
			srv := fixture.server(t)
			result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{ScopeTestName: "TestGetProcessDefinitionLifecycleWatchTerminal", Env: map[string]string{"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.10"), "TERM": "xterm-256color", "NO_COLOR": noColor}})
			if !result.Supported {
				t.Skip(result.UnsupportedReason)
			}
			require.NoError(t, result.Err, result.Stderr)
			require.EqualValues(t, 5, fixture.definitions.Load())
			require.EqualValues(t, 60, fixture.counts.Load())
			frames := strings.Split(result.Stdout, processDefinitionWatchRepaintControlSequenceForTest)[1:]
			require.Len(t, frames, 5)
			for i, state := range []string{"ACTIVE", "DRAINING", "DELETED", "", ""} {
				plain := strings.ReplaceAll(strings.ReplaceAll(frames[i], processDefinitionWatchBoldStart, ""), processDefinitionWatchBoldReset, "")
				lines := strings.Split(strings.TrimSpace(plain), "\n")
				require.Len(t, lines, 3)
				require.Equal(t, "found: 2", lines[2])
				require.Equal(t, strings.Index(lines[0], "[ac:"), strings.Index(lines[1], "[ac:"), "mixed state columns must align")
				if state != "" {
					require.Contains(t, lines[0], state)
				} else {
					require.NotContains(t, lines[0], "DELETED")
				}
				require.NotContains(t, frames[i], "\x1b[1m\x1b[0m")
				if noColor != "" || i == 0 || i >= 3 {
					require.NotContains(t, frames[i], processDefinitionWatchBoldStart)
				} else {
					require.Contains(t, frames[i], processDefinitionWatchBoldStart+state+processDefinitionWatchBoldReset)
					require.NotContains(t, frames[i], "\x1b[1m2\x1b[0m", "unchanged suspended counts must stay plain")
					if i == 2 {
						require.Contains(t, frames[i], "ac:\x1b[1m1\x1b[0m")
					} else {
						require.NotContains(t, frames[i], "ac:\x1b")
					}
				}
				if i == 4 {
					require.NotContains(t, plain, " *")
				}
			}
			require.Equal(t, 5, strings.Count(result.Stderr, " new, "))
			require.NotContains(t, result.Stdout, "INFO")
		})
	}
}
