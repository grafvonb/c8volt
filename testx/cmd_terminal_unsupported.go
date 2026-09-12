// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build !linux && !darwin

package testx

import (
	"fmt"
	"runtime"
)

type unsupportedCmdTerminalAllocator struct{}

// newCmdTerminalAllocator selects the explicit unsupported-platform implementation.
func newCmdTerminalAllocator() cmdTerminalAllocator {
	return unsupportedCmdTerminalAllocator{}
}

// UnsupportedReason explains why terminal subprocess acceptance cannot run.
func (unsupportedCmdTerminalAllocator) UnsupportedReason() string {
	return fmt.Sprintf("terminal subprocesses are unsupported on %s", runtime.GOOS)
}

// Allocate returns the platform reason instead of attempting Unix-only operations.
func (a unsupportedCmdTerminalAllocator) Allocate() (cmdTerminalAllocation, error) {
	return cmdTerminalAllocation{}, fmt.Errorf("%s", a.UnsupportedReason())
}
