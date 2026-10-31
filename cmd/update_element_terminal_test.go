// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const elementUpdatePrompt = "You are about to update 1 requested variable value(s) on 1 element(s).\nDo you want to proceed? [y/N]: "

// TestElementScopeVariablesUpdateTerminal proves prompt routing/default/EOF against actual terminal stdin,
// with redirected stdout and both command-configured and inherited stderr writers.
func TestElementScopeVariablesUpdateTerminal(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, tc := range []struct {
			name, response                 string
			eof, noop, empty, dry, approve bool
			flags                          []string
		}{
			{name: "approve", response: "yes", approve: true}, {name: "decline", response: "no"}, {name: "default"}, {name: "EOF", eof: true},
			{name: "keys", response: "y", approve: true, flags: []string{"--keys-only"}},
			{name: "quiet keys", response: "y", approve: true, flags: []string{"--keys-only", "--quiet"}},
			{name: "no-wait", response: "y", approve: true, flags: []string{"--no-wait"}},
			{name: "auto-confirm", approve: true, flags: []string{"--auto-confirm"}},
			{name: "automation", approve: true, flags: []string{"--automation"}},
			{name: "JSON", approve: true, flags: []string{"--json", "--auto-confirm"}},
			{name: "unchanged", noop: true}, {name: "empty object", empty: true}, {name: "dry-run", dry: true},
			{name: "noop no-wait", noop: true, flags: []string{"--no-wait"}},
			{name: "noop keys", noop: true, flags: []string{"--keys-only"}},
			{name: "noop quiet", noop: true, flags: []string{"--quiet"}},
			{name: "dry JSON", dry: true, flags: []string{"--json"}},
			{name: "noop JSON", noop: true, flags: []string{"--json", "--auto-confirm"}},
		} {
			t.Run(destination+"/"+tc.name, func(t *testing.T) {
				f := newElementUpdateFixture(t, "8.10")
				if tc.noop {
					f.local[scopeElementA] = []string{scopeVariableJSON("x", "true", scopeElementA, false)}
				}
				payload := `{"x":true}`
				if tc.empty {
					payload = "{}"
				}
				args := append([]string{"--config", completionLogConfig(t, f.server.URL, "plain-time", "info"), "--no-indicator", "update", "element", "--key", scopeElementA, "--vars", payload}, tc.flags...)
				if tc.dry {
					args = append(args, "--dry-run")
				}
				flags := strings.Join(tc.flags, " ")
				prompt := !tc.noop && !tc.empty && !tc.dry && !strings.Contains(flags, "--auto")
				var exchanges []testx.CmdTerminalExchange
				if prompt {
					exchanges = []testx.CmdTerminalExchange{{Prompt: elementUpdatePrompt, Response: tc.response, EndOfInput: tc.eof, BeforeResponse: func(out, stderr string) {
						calls, writes, reads := f.snapshot()
						require.Len(t, calls, 2)
						require.Empty(t, writes)
						require.Equal(t, 1, reads[scopeElementA])
						require.Empty(t, out)
					}}}
				}
				out, stderr, err := runElementScopeTerminal(t, args, destination, exchanges)
				abort := prompt && !tc.approve
				if abort {
					assertUserTaskCompletionExit(t, err, 1)
				} else {
					require.NoError(t, err, stderr)
				}
				if prompt {
					require.Equal(t, 1, strings.Count(stderr, elementUpdatePrompt))
					stderr = strings.Replace(stderr, elementUpdatePrompt, "", 1)
				} else {
					require.NotContains(t, stderr, "proceed?")
				}
				require.NotContains(t, out, "proceed?")
				require.NotContains(t, out, "[y/N]")
				records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
				errors := 0
				for _, r := range records {
					if r[0] == "ERROR" {
						errors++
						require.Contains(t, r[1], "aborted")
					}
				}
				if abort {
					require.Equal(t, 1, errors)
				} else {
					require.Zero(t, errors)
				}
				calls, writes, reads := f.snapshot()
				wantWrites, wantReads := 0, 1
				if tc.approve {
					wantWrites = 1
					if !strings.Contains(flags, "--no-wait") {
						wantReads = 2
					}
				}
				require.Len(t, writes, wantWrites)
				require.Equal(t, map[string]int{scopeElementA: wantReads}, reads)
				require.Len(t, calls, 1+wantWrites+wantReads)
				switch {
				case strings.Contains(flags, "--json"):
					env := requireSingleJSONObjectDocument(t, out)
					require.Equal(t, "succeeded", env["outcome"])
					require.Empty(t, stderr)
				case strings.Contains(flags, "--keys-only"):
					expected := ""
					if tc.approve {
						expected = scopeElementA + "\n"
					}
					require.Equal(t, expected, out)
				default:
					require.Empty(t, out)
				}
			})
		}
	}
}

// TestElementScopeVariablesUpdateTerminalPlanningFailure never offers approval for a partial plan.
func TestElementScopeVariablesUpdateTerminalPlanningFailure(t *testing.T) {
	f := newElementUpdateFixture(t, "8.10")
	f.lookupFailure = scopeElementB
	args := []string{"--config", completionLogConfig(t, f.server.URL, "plain-time", "info"), "--no-indicator", "update", "element", "--key", scopeElementA + "," + scopeElementB, "--vars", `{"x":true}`}
	out, stderr, err := runElementScopeTerminal(t, args, "configured", nil)
	assertUserTaskCompletionExit(t, err, 3)
	require.Empty(t, out)
	require.NotContains(t, stderr, "proceed?")
	require.Equal(t, [][2]string{{"ERROR", "plan element variable update: not found"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
	calls, writes, _ := f.snapshot()
	require.Len(t, calls, 3)
	require.Empty(t, writes)
}

// TestElementScopeVariablesUpdateMetadata keeps help, alias, capability and supported controls discoverable.
func TestElementScopeVariablesUpdateMetadata(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	cap := commandCapabilityForCommand(updateElementCmd)
	require.Equal(t, "update element", cap.Path)
	require.Equal(t, CommandMutationStateChanging, cap.Mutation)
	require.Equal(t, ContractSupportFull, cap.ContractSupport)
	require.Equal(t, AutomationSupportFull, cap.AutomationSupport)
	require.Contains(t, cap.Aliases, "ei")
	for _, name := range []string{"key", "vars", "vars-file", "dry-run", "workers", "no-worker-limit", "fail-fast", "no-wait"} {
		require.NotNil(t, updateElementCmd.Flags().Lookup(name))
	}
	require.Len(t, cap.OutputModes, 3)
	require.Contains(t, updateCmd.Long, "element-local")
	for _, text := range []string{"element:<key>", "no-wait", "non-null", "--automation", "Parent", "Partial failures", "empty object"} {
		require.Contains(t, updateElementCmd.Long, text)
	}
	for _, cmd := range []string{"update", "update element"} {
		_, ok := findCommandCapability(capabilityDocumentForRoot(root).Commands, cmd)
		require.True(t, ok, fmt.Sprintf("missing %s", cmd))
	}
}
