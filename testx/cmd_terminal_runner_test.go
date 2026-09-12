// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

// TestCmdTerminalRunnerSequencesPrompts verifies real terminal stdin, prompt-paced answers, EOF, and separate output capture.
func TestCmdTerminalRunnerSequencesPrompts(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		runCmdTerminalSequenceHelper(t)
		os.Exit(0)
	}

	result := NewCmdTerminalRunner().Run(t, CmdTerminalRunRequest{
		ScopeTestName: "TestCmdTerminalRunnerSequencesPrompts",
		Env:           map[string]string{"C8VOLT_TERMINAL_RUNNER_VALUE": "ready"},
		Exchanges: []CmdTerminalExchange{
			{Prompt: "repeated prompt: ", Response: "alpha"},
			{Prompt: "repeated prompt: ", Response: "beta"},
			{Prompt: "eof prompt: ", EndOfInput: true},
		},
		Timeout: 3 * time.Second,
	})

	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.True(t, result.Supported, result.UnsupportedReason)
	require.NoError(t, result.Err)
	require.Equal(t, "terminal=true env=ready\nanswers=alpha,beta eof=true\n", result.Stdout)
	require.Equal(t, "repeated prompt: repeated prompt: eof prompt: ", result.Stderr)
}

// runCmdTerminalSequenceHelper is the isolated child side of the terminal sequencing test.
func runCmdTerminalSequenceHelper(t *testing.T) {
	require.Equal(t, "TestCmdTerminalRunnerSequencesPrompts", os.Getenv(CmdSubprocessNameEnv))
	fmt.Fprintf(os.Stdout, "terminal=%t env=%s\n", term.IsTerminal(int(os.Stdin.Fd())), os.Getenv("C8VOLT_TERMINAL_RUNNER_VALUE"))

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Fprint(os.Stderr, "repeated prompt: ")
	require.True(t, scanner.Scan())
	first := scanner.Text()
	fmt.Fprint(os.Stderr, "repeated prompt: ")
	require.True(t, scanner.Scan())
	second := scanner.Text()
	fmt.Fprint(os.Stderr, "eof prompt: ")
	require.False(t, scanner.Scan())
	require.NoError(t, scanner.Err())

	fmt.Fprintf(os.Stdout, "answers=%s,%s eof=true\n", first, second)
}

// TestCmdTerminalRunnerTimesOut verifies that an unresponsive child is bounded and reaped.
func TestCmdTerminalRunnerTimesOut(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		fmt.Fprint(os.Stderr, "waiting forever: ")
		time.Sleep(time.Minute)
		return
	}

	started := time.Now()
	result := NewCmdTerminalRunner().Run(t, CmdTerminalRunRequest{
		ScopeTestName: "TestCmdTerminalRunnerTimesOut",
		Timeout:       100 * time.Millisecond,
	})

	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.True(t, result.Supported, result.UnsupportedReason)
	require.ErrorContains(t, result.Err, "timed out")
	require.Less(t, time.Since(started), 3*time.Second)
	require.Equal(t, "waiting forever: ", result.Stderr)
}

// TestCmdTerminalRunnerReportsAllocatorFailure verifies supported-platform allocation errors are failures, not skips.
func TestCmdTerminalRunnerReportsAllocatorFailure(t *testing.T) {
	runner := cmdTerminalSubprocessRunner{
		allocator: stubCmdTerminalAllocator{err: errors.New("injected allocation failure")},
	}

	result := runner.Run(t, CmdTerminalRunRequest{ScopeTestName: t.Name()})

	require.True(t, result.Supported)
	require.Empty(t, result.UnsupportedReason)
	require.ErrorContains(t, result.Err, "injected allocation failure")
}

// TestCmdTerminalRunnerReportsUnsupportedPlatform verifies explicit unsupported targets return without starting a child.
func TestCmdTerminalRunnerReportsUnsupportedPlatform(t *testing.T) {
	runner := cmdTerminalSubprocessRunner{
		allocator: stubCmdTerminalAllocator{unsupportedReason: "no PTY implementation"},
	}

	result := runner.Run(t, CmdTerminalRunRequest{ScopeTestName: t.Name()})

	require.False(t, result.Supported)
	require.Equal(t, "no PTY implementation", result.UnsupportedReason)
	require.NoError(t, result.Err)
}

type stubCmdTerminalAllocator struct {
	allocation        cmdTerminalAllocation
	err               error
	unsupportedReason string
}

// Allocate returns the injected terminal allocation result for runner control-flow tests.
func (a stubCmdTerminalAllocator) Allocate() (cmdTerminalAllocation, error) {
	return a.allocation, a.err
}

// UnsupportedReason returns the injected platform support result for runner control-flow tests.
func (a stubCmdTerminalAllocator) UnsupportedReason() string {
	return a.unsupportedReason
}
