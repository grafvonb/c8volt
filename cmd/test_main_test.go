// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	guardCmdSubprocessHelpers(os.Args)
	os.Exit(m.Run())
}

// guardCmdSubprocessHelpers prevents leaked helper-process env from turning a broad test run into os.Exit.
func guardCmdSubprocessHelpers(args []string) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	helperName := os.Getenv(testx.CmdSubprocessNameEnv)
	if helperName != "" && testRunArgMatchesExactHelper(args, helperName) {
		return
	}
	_ = os.Unsetenv("GO_WANT_HELPER_PROCESS")
	_ = os.Unsetenv(testx.CmdSubprocessNameEnv)
}

func testRunArgMatchesExactHelper(args []string, helperName string) bool {
	want := "^" + regexp.QuoteMeta(helperName) + "$"
	for i, arg := range args {
		if strings.HasPrefix(arg, "-test.run=") {
			return strings.TrimPrefix(arg, "-test.run=") == want
		}
		if arg == "-test.run" && i+1 < len(args) {
			return args[i+1] == want
		}
	}
	return false
}

// TestGuardCmdSubprocessHelpersKeepsExactHelper preserves the intended subprocess path that validates real exit codes.
func TestGuardCmdSubprocessHelpersKeepsExactHelper(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv(testx.CmdSubprocessNameEnv, "TestExactHelper")

	guardCmdSubprocessHelpers([]string{"cmd.test", "-test.run=^TestExactHelper$"})

	require.Equal(t, "1", os.Getenv("GO_WANT_HELPER_PROCESS"))
	require.Equal(t, "TestExactHelper", os.Getenv(testx.CmdSubprocessNameEnv))
}

// TestGuardCmdSubprocessHelpersClearsLeakedEnv protects broad package runs from helper tests calling os.Exit.
func TestGuardCmdSubprocessHelpersClearsLeakedEnv(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	t.Setenv(testx.CmdSubprocessNameEnv, "TestExactHelper")

	guardCmdSubprocessHelpers([]string{"cmd.test", "-test.run=TestExactHelper"})

	require.Empty(t, os.Getenv("GO_WANT_HELPER_PROCESS"))
	require.Empty(t, os.Getenv(testx.CmdSubprocessNameEnv))
}
