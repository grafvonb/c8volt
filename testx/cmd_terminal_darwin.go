// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build darwin

package testx

import (
	"bytes"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

const darwinIOCParmMask = 0x1fff

type darwinCmdTerminalAllocator struct {
	open  func(string, int, uint32) (int, error)
	ioctl func(int, uintptr, uintptr) error
}

// newCmdTerminalAllocator selects the Darwin PTY implementation.
func newCmdTerminalAllocator() cmdTerminalAllocator {
	return darwinCmdTerminalAllocator{}
}

// UnsupportedReason reports that Darwin provides the required PTY facility.
func (darwinCmdTerminalAllocator) UnsupportedReason() string {
	return ""
}

// Allocate opens a Darwin PTY pair and configures canonical input without echo.
func (a darwinCmdTerminalAllocator) Allocate() (allocation cmdTerminalAllocation, err error) {
	open := a.open
	if open == nil {
		open = unix.Open
	}
	ioctl := a.ioctl
	if ioctl == nil {
		ioctl = darwinCmdTerminalIoctl
	}

	masterFD, err := open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("open PTY master: %w", err)
	}
	master := os.NewFile(uintptr(masterFD), "/dev/ptmx")
	if master == nil {
		_ = unix.Close(masterFD)
		return cmdTerminalAllocation{}, fmt.Errorf("wrap PTY master descriptor")
	}
	defer func() {
		if err != nil {
			_ = master.Close()
		}
	}()

	slaveNameSize := int(uintptr(unix.TIOCPTYGNAME)>>16) & darwinIOCParmMask
	slaveName := make([]byte, slaveNameSize)
	if err = ioctl(masterFD, uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&slaveName[0]))); err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("resolve PTY slave name: %w", err)
	}
	nul := bytes.IndexByte(slaveName, 0)
	if nul < 0 {
		return cmdTerminalAllocation{}, fmt.Errorf("resolve PTY slave name: response is not NUL-terminated")
	}
	slavePath := string(slaveName[:nul])

	if err = ioctl(masterFD, uintptr(unix.TIOCPTYGRANT), 0); err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("grant PTY slave: %w", err)
	}
	if err = ioctl(masterFD, uintptr(unix.TIOCPTYUNLK), 0); err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("unlock PTY slave: %w", err)
	}

	slaveFD, err := open(slavePath, unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("open PTY slave %q: %w", slavePath, err)
	}
	slave := os.NewFile(uintptr(slaveFD), slavePath)
	if slave == nil {
		_ = unix.Close(slaveFD)
		return cmdTerminalAllocation{}, fmt.Errorf("wrap PTY slave descriptor")
	}
	defer func() {
		if err != nil {
			_ = slave.Close()
		}
	}()

	settings, err := unix.IoctlGetTermios(slaveFD, unix.TIOCGETA)
	if err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("read PTY slave settings: %w", err)
	}
	settings.Lflag |= unix.ICANON
	settings.Lflag &^= unix.ECHO | unix.ECHONL
	if err = unix.IoctlSetTermios(slaveFD, unix.TIOCSETA, settings); err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("configure PTY slave: %w", err)
	}

	return cmdTerminalAllocation{
		master:       master,
		slave:        slave,
		canonicalEOF: settings.Cc[unix.VEOF],
	}, nil
}

// darwinCmdTerminalIoctl invokes an ioctl whose argument may be a pointer or zero.
func darwinCmdTerminalIoctl(fd int, request uintptr, argument uintptr) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), request, argument)
	if errno != 0 {
		return errno
	}
	return nil
}
