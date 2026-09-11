// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// DefaultCmdTerminalTimeout bounds a terminal subprocess when a caller does not
// provide a shorter scenario-specific deadline.
const DefaultCmdTerminalTimeout = 10 * time.Second

// CmdTerminalExchange describes one prompt that must be completely observed on
// stderr before the runner supplies either a line response or canonical EOF.
type CmdTerminalExchange struct {
	Prompt     string
	Response   string
	EndOfInput bool
}

// CmdTerminalRunRequest describes an exactly selected helper subprocess whose
// stdin is a terminal while stdout and stderr remain independently captured.
type CmdTerminalRunRequest struct {
	ScopeTestName string
	Dir           string
	Env           map[string]string
	Exchanges     []CmdTerminalExchange
	Timeout       time.Duration
}

// CmdTerminalRunResult preserves independent output streams and reports
// unsupported platforms separately from allocation or subprocess failures.
type CmdTerminalRunResult struct {
	Stdout            string
	Stderr            string
	Err               error
	Supported         bool
	UnsupportedReason string
}

// CmdTerminalRunner executes a bounded helper subprocess and guarantees that a
// timed-out child is killed, reaped, and detached from all PTY descriptors.
type CmdTerminalRunner interface {
	Run(t *testing.T, request CmdTerminalRunRequest) CmdTerminalRunResult
}

type cmdTerminalSubprocessRunner struct {
	allocator cmdTerminalAllocator
}

// NewCmdTerminalRunner returns the platform-aware terminal subprocess runner.
func NewCmdTerminalRunner() CmdTerminalRunner {
	return cmdTerminalSubprocessRunner{allocator: newCmdTerminalAllocator()}
}

// Run executes one exactly selected test helper with terminal stdin and independently captured output streams.
func (r cmdTerminalSubprocessRunner) Run(t *testing.T, request CmdTerminalRunRequest) (result CmdTerminalRunResult) {
	t.Helper()

	allocator := r.allocator
	if allocator == nil {
		allocator = newCmdTerminalAllocator()
	}
	if reason := allocator.UnsupportedReason(); reason != "" {
		result.UnsupportedReason = reason
		return result
	}
	result.Supported = true

	allocation, err := allocator.Allocate()
	if err != nil {
		result.Err = fmt.Errorf("allocate terminal stdin: %w", err)
		return result
	}
	if allocation.master == nil || allocation.slave == nil {
		closeCmdTerminalAllocation(allocation)
		result.Err = errors.New("allocate terminal stdin: allocator returned an incomplete PTY pair")
		return result
	}

	var stdout cmdTerminalCapture
	var stderr cmdTerminalCapture
	cmd := exec.Command(os.Args[0], "-test.run=^"+regexp.QuoteMeta(request.ScopeTestName)+"$")
	cmd.Dir = request.Dir
	cmd.Stdin = allocation.slave
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Env = cmdTerminalEnvironment(request.ScopeTestName, request.Env)

	if err = cmd.Start(); err != nil {
		closeCmdTerminalAllocation(allocation)
		result.Err = fmt.Errorf("start terminal subprocess: %w", err)
		return result
	}

	// The child owns its duplicated slave descriptor after Start. Closing the
	// parent copy lets child exit and terminal EOF propagate without a hidden
	// extra slave reference keeping the PTY alive.
	_ = allocation.slave.Close()

	drainDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(io.Discard, allocation.master)
		close(drainDone)
	}()

	var childErr error
	childDone := make(chan struct{})
	go func() {
		childErr = cmd.Wait()
		close(childDone)
	}()

	defer func() {
		select {
		case <-childDone:
		default:
			_ = cmd.Process.Kill()
			<-childDone
		}
		_ = allocation.master.Close()
		<-drainDone
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
	}()

	timeout := request.Timeout
	if timeout <= 0 {
		timeout = DefaultCmdTerminalTimeout
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	searchFrom := 0
	for _, exchange := range request.Exchanges {
		if exchange.Prompt == "" {
			result.Err = errors.New("terminal exchange prompt must not be empty")
			return result
		}
		searchFrom, err = stderr.waitFor(exchange.Prompt, searchFrom, deadline.C, childDone)
		if err != nil {
			result.Err = cmdTerminalWaitError(err, exchange.Prompt, timeout, childErr)
			return result
		}

		var input []byte
		if exchange.EndOfInput {
			input = []byte{allocation.canonicalEOF}
		} else {
			input = []byte(exchange.Response + "\n")
		}
		if _, err = allocation.master.Write(input); err != nil {
			result.Err = fmt.Errorf("write terminal response for prompt %q: %w", exchange.Prompt, err)
			return result
		}
	}

	select {
	case <-childDone:
		result.Err = childErr
	case <-deadline.C:
		result.Err = fmt.Errorf("terminal subprocess timed out after %s", timeout)
	}
	return result
}

var (
	errCmdTerminalWaitTimeout = errors.New("terminal prompt wait timed out")
	errCmdTerminalChildExited = errors.New("terminal subprocess exited before prompt")
)

// cmdTerminalWaitError adds the missing prompt and child outcome to a wait failure.
func cmdTerminalWaitError(waitErr error, prompt string, timeout time.Duration, childErr error) error {
	if errors.Is(waitErr, errCmdTerminalWaitTimeout) {
		return fmt.Errorf("terminal subprocess timed out after %s waiting for stderr prompt %q", timeout, prompt)
	}
	if childErr != nil {
		return fmt.Errorf("terminal subprocess exited before stderr prompt %q: %w", prompt, childErr)
	}
	return fmt.Errorf("terminal subprocess exited before stderr prompt %q", prompt)
}

// cmdTerminalEnvironment preserves the existing isolated helper-process markers while adding scenario values.
func cmdTerminalEnvironment(scopeTestName string, env map[string]string) []string {
	merged := append(os.Environ(),
		"GO_WANT_HELPER_PROCESS=1",
		CmdSubprocessNameEnv+"="+scopeTestName,
	)
	for key, value := range env {
		merged = append(merged, key+"="+value)
	}
	return merged
}

// closeCmdTerminalAllocation closes both sides of a PTY allocation during setup failure.
func closeCmdTerminalAllocation(allocation cmdTerminalAllocation) {
	if allocation.slave != nil {
		_ = allocation.slave.Close()
	}
	if allocation.master != nil {
		_ = allocation.master.Close()
	}
}

type cmdTerminalCapture struct {
	mu      sync.Mutex
	buffer  bytes.Buffer
	changed chan struct{}
}

// Write appends subprocess output and wakes prompt observers without exposing an unsynchronized bytes.Buffer.
func (c *cmdTerminalCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	written, err := c.buffer.Write(p)
	if c.changed == nil {
		c.changed = make(chan struct{}, 1)
	}
	changed := c.changed
	c.mu.Unlock()

	select {
	case changed <- struct{}{}:
	default:
	}
	return written, err
}

// String returns a stable snapshot of captured subprocess output.
func (c *cmdTerminalCapture) String() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buffer.String()
}

// waitFor observes a complete prompt after searchFrom without polling shared capture state.
func (c *cmdTerminalCapture) waitFor(prompt string, searchFrom int, deadline <-chan time.Time, childDone <-chan struct{}) (int, error) {
	for {
		contents, changed := c.snapshot()
		if index := strings.Index(contents[searchFrom:], prompt); index >= 0 {
			return searchFrom + index + len(prompt), nil
		}

		select {
		case <-changed:
		case <-deadline:
			return searchFrom, errCmdTerminalWaitTimeout
		case <-childDone:
			contents, _ = c.snapshot()
			if index := strings.Index(contents[searchFrom:], prompt); index >= 0 {
				return searchFrom + index + len(prompt), nil
			}
			return searchFrom, errCmdTerminalChildExited
		}
	}
}

// snapshot returns captured output and the notification channel protected by the same lock.
func (c *cmdTerminalCapture) snapshot() (string, <-chan struct{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.changed == nil {
		c.changed = make(chan struct{}, 1)
	}
	return c.buffer.String(), c.changed
}

// cmdTerminalAllocation is the minimal platform boundary: platform files open
// a PTY pair and provide the canonical-mode EOF control byte used by the runner.
type cmdTerminalAllocation struct {
	master       *os.File
	slave        *os.File
	canonicalEOF byte
}

// cmdTerminalAllocator reports unsupported targets without disguising runtime
// allocation failures on platforms where PTYs are expected to work.
type cmdTerminalAllocator interface {
	Allocate() (cmdTerminalAllocation, error)
	UnsupportedReason() string
}
