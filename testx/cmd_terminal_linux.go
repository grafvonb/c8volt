// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build linux

package testx

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

type linuxCmdTerminalAllocator struct {
	open func(string, int, uint32) (int, error)
}

func newCmdTerminalAllocator() cmdTerminalAllocator {
	return linuxCmdTerminalAllocator{}
}

func (linuxCmdTerminalAllocator) UnsupportedReason() string {
	return ""
}

func (a linuxCmdTerminalAllocator) Allocate() (allocation cmdTerminalAllocation, err error) {
	open := a.open
	if open == nil {
		open = unix.Open
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

	if err = unix.IoctlSetPointerInt(masterFD, unix.TIOCSPTLCK, 0); err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("unlock PTY slave: %w", err)
	}
	ptyNumber, err := unix.IoctlGetInt(masterFD, unix.TIOCGPTN)
	if err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("resolve PTY slave number: %w", err)
	}

	slavePath := "/dev/pts/" + strconv.Itoa(ptyNumber)
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

	settings, err := unix.IoctlGetTermios(slaveFD, unix.TCGETS)
	if err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("read PTY slave settings: %w", err)
	}
	settings.Lflag |= unix.ICANON
	settings.Lflag &^= unix.ECHO | unix.ECHONL
	if err = unix.IoctlSetTermios(slaveFD, unix.TCSETS, settings); err != nil {
		return cmdTerminalAllocation{}, fmt.Errorf("configure PTY slave: %w", err)
	}

	return cmdTerminalAllocation{
		master:       master,
		slave:        slave,
		canonicalEOF: settings.Cc[unix.VEOF],
	}, nil
}
