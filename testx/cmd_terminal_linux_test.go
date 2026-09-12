// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build linux

package testx

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func TestLinuxCmdTerminalAllocator(t *testing.T) {
	allocator := newCmdTerminalAllocator()
	require.Empty(t, allocator.UnsupportedReason())

	allocation, err := allocator.Allocate()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, allocation.master.Close())
		require.NoError(t, allocation.slave.Close())
	})

	require.NotNil(t, allocation.master)
	require.NotNil(t, allocation.slave)
	require.True(t, term.IsTerminal(int(allocation.slave.Fd())))

	settings, err := unix.IoctlGetTermios(int(allocation.slave.Fd()), unix.TCGETS)
	require.NoError(t, err)
	require.NotZero(t, settings.Lflag&unix.ICANON, "slave input must remain canonical")
	require.Zero(t, settings.Lflag&unix.ECHO, "slave input must not echo")
	require.Zero(t, settings.Lflag&unix.ECHONL, "slave newlines must not echo")
	require.Equal(t, settings.Cc[unix.VEOF], allocation.canonicalEOF)
	require.NotZero(t, allocation.canonicalEOF)
}

func TestLinuxCmdTerminalAllocatorClosesMasterWhenSlaveOpenFails(t *testing.T) {
	var masterFD int
	openCalls := 0
	allocator := linuxCmdTerminalAllocator{
		open: func(path string, flags int, mode uint32) (int, error) {
			openCalls++
			if openCalls == 2 {
				return -1, errors.New("injected slave open failure")
			}
			fd, err := unix.Open(path, flags, mode)
			masterFD = fd
			return fd, err
		},
	}

	allocation, err := allocator.Allocate()
	require.Equal(t, cmdTerminalAllocation{}, allocation)
	require.ErrorContains(t, err, "injected slave open failure")
	require.Equal(t, 2, openCalls)

	_, descriptorErr := unix.FcntlInt(uintptr(masterFD), unix.F_GETFD, 0)
	require.ErrorIs(t, descriptorErr, unix.EBADF)
}
