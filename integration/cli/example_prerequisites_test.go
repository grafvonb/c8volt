// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cli_test

import (
	"io"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/pflag"
)

const exampleExpectationFixtureRequired = "expect example requires lifecycle or incident setup beyond the active user-task fixture"

// The example fixture creates active user tasks without incidents. Examples
// are independent commands, not a script that performs preceding mutations.
func exampleExpectationNeedsFixture(commandPath string, args []string) bool {
	if commandPath != "expect process-instance" {
		return false
	}
	flags := pflag.NewFlagSet("example prerequisites", pflag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.ParseErrorsWhitelist.UnknownFlags = true
	statesRaw := flags.StringSliceP("state", "s", nil, "")
	incidentRaw := flags.String("incident", "", "")
	if err := flags.Parse(args); err != nil {
		return false // Preserve CLI validation failures instead of masking them as setup gaps.
	}
	states, err := process.ParseStates(*statesRaw)
	if err != nil {
		return false
	}
	incident := false
	if flags.Changed("incident") {
		value, err := process.ParseIncidentExpectation(*incidentRaw)
		if err != nil {
			return false
		}
		incident = value.Bool()
	}
	// States are alternatives, whereas the incident requirement is an AND predicate.
	return incident || (len(states) > 0 && !states.Contains(process.StateActive))
}

func exampleExecutionMode(args []string, commandPath string, destructive bool, haveSeed bool) (string, bool) {
	if len(args) == 0 {
		return "empty command", false
	}
	if isLocalOnlyExample(commandPath, args) {
		return "local", true
	}
	if !haveSeed && requiresDisposableProfile(commandPath) {
		return "requires selected disposable profile", false
	}
	if exampleExpectationNeedsFixture(commandPath, args) {
		return exampleExpectationFixtureRequired, false
	}
	if destructive {
		if !haveSeed {
			return "requires selected disposable profile", false
		}
		if hasArg(args, "--dry-run") || hasArg(args, "--help") {
			return "disposable-dry-run", true
		}
		return "mutating example requires a dedicated disposable scenario before execution", false
	}
	if requiresDisposableProfile(commandPath) {
		if !haveSeed {
			return "requires selected disposable profile", false
		}
		return "disposable-profile", true
	}
	return "local", true
}

func isLocalOnlyExample(commandPath string, args []string) bool {
	if hasArg(args, "--help") {
		return true
	}
	switch commandPath {
	case "version", "capabilities", "config template", "embed list":
		return true
	default:
		return false
	}
}

func requiresDisposableProfile(commandPath string) bool {
	if commandPath == "" {
		return true
	}
	switch commandPath {
	case "version", "capabilities", "config template", "embed list":
		return false
	default:
		return true
	}
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want || strings.HasPrefix(arg, want+"=") {
			return true
		}
	}
	return false
}

// Exercise the same execution gate used before any example subprocess starts.
func TestExampleExpectationPrerequisites(t *testing.T) {
	for _, tc := range []struct {
		name    string
		path    string
		args    []string
		blocked bool
	}{
		{"delete follow-up", "expect process-instance", []string{"expect", "process-instance", "--key", "123", "--state", "absent"}, true},
		{"suspended", "expect process-instance", []string{"--state", "suspended"}, true},
		{"completed", "expect process-instance", []string{"--state=completed"}, true},
		{"canceled", "expect process-instance", []string{"-s", "canceled"}, true},
		{"cancelled alias", "expect process-instance", []string{"--state", "cancelled"}, true},
		{"terminated", "expect process-instance", []string{"--state", "terminated"}, true},
		{"terminal alternatives", "expect process-instance", []string{"--state", "completed", "--state", "absent"}, true},
		{"comma alternatives", "expect process-instance", []string{"--state=completed,absent"}, true},
		{"incident", "expect process-instance", []string{"--incident", "true"}, true},
		{"active with incident", "expect process-instance", []string{"--state", "active", "--incident=true"}, true},
		{"active", "expect process-instance", []string{"--state", "active"}, false},
		{"mixed case", "expect process-instance", []string{"-s", "AcTiVe"}, false},
		{"active without incident", "expect process-instance", []string{"--state", "active", "--incident", "false"}, false},
		{"no incident", "expect process-instance", []string{"--incident=false"}, false},
		{"active alternative", "expect process-instance", []string{"--state=completed,active"}, false},
		{"repeated active alternative", "expect process-instance", []string{"--state", "completed", "--state", "active"}, false},
		{"invalid state remains failure", "expect process-instance", []string{"--state", "typo"}, false},
		{"invalid incident remains failure", "expect process-instance", []string{"--state", "absent", "--incident", "typo"}, false},
		{"missing state value remains failure", "expect process-instance", []string{"--state"}, false},
		{"help needs no fixture", "expect process-instance", []string{"--help", "--state", "absent"}, false},
		{"ordinary read unchanged", "get process-instance", []string{"--state", "completed"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mode, runnable := exampleExecutionMode(tc.args, tc.path, false, true)
			if runnable == tc.blocked {
				t.Fatalf("runnable=%v, want %v (mode=%q)", runnable, !tc.blocked, mode)
			}
			if tc.blocked && mode != exampleExpectationFixtureRequired {
				t.Fatalf("unexpected blocked reason: %q", mode)
			}
		})
	}
	if mode, runnable := exampleExecutionMode([]string{"--state", "active"}, "expect process-instance", false, false); runnable || mode != "requires selected disposable profile" {
		t.Fatalf("missing profile gate changed: %q, %v", mode, runnable)
	}
	if _, runnable := exampleExecutionMode([]string{"--key", "123"}, "delete process-instance", true, true); runnable {
		t.Fatal("mutating example gate bypassed")
	}
}
