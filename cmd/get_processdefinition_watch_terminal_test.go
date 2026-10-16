// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

const processDefinitionWatchTerminalModeEnv = "C8VOLT_PD_WATCH_TERMINAL_MODE"

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
