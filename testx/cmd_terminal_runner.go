// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package testx

import (
	"os"
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
