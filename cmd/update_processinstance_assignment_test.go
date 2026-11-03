// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDAssignmentCommand proves dispatch bypasses variable parsing and
// uses exact keys and exact scalar values with one submission per distinct key.
func TestBusinessIDAssignmentCommand(t *testing.T) {
	var mu sync.Mutex
	reads, writes := 0, 0
	value := " Order,42 "
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == "GET":
			reads++
			w.Header().Set("Content-Type", "application/json")
			bid := ""
			if writes > 0 {
				bid = value
			}
			fmt.Fprintf(w, `{"processInstanceKey":"2251799813711967","state":"ACTIVE","businessId":%q}`, bid)
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/business-id-assignment"):
			require.Equal(t, "/v2/process-instances/2251799813711967/business-id-assignment", r.URL.Path)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"businessId": value}, body)
			writes++
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()
	stdout, _, err := runProcessInstanceOutcomeCommand(t, server, "--auto-confirm", "--json", "update", "pi", "--key", "2251799813711967", "--key", "2251799813711967", "--set-bid", value)
	require.NoError(t, err, stdout)
	envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
	require.Equal(t, "succeeded", envelope["outcome"])
	item := envelope["payload"].(map[string]any)["items"].([]any)[0].(map[string]any)
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, value, item["requestedBusinessId"])
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, writes)
	require.Equal(t, 2, reads)
}

// TestBusinessIDAssignmentInput validates the whole request before any target
// read or write, including late stdin errors and old adapter capability guards.
func TestBusinessIDAssignmentInput(t *testing.T) {
	for _, tc := range []struct {
		name, version, input string
		args                 []string
	}{
		{name: "missing keys", args: []string{}},
		{name: "blank", args: []string{"--key", assignmentKey, "--set-bid", "  "}},
		{name: "empty", args: []string{"--key", assignmentKey, "--set-bid", ""}},
		{name: "too long", args: []string{"--key", assignmentKey, "--set-bid", strings.Repeat("😀", 129)}},
		{name: "bid selector", args: []string{"--key", assignmentKey, "--bid", "lookup"}},
		{name: "variables", args: []string{"--key", assignmentKey, "--set-vars", `{"businessId":"variable"}`}},
		{name: "variable file", args: []string{"--key", assignmentKey, "--set-vars-file", "missing.json"}},
		{name: "limit", args: []string{"--key", assignmentKey, "--limit", "1"}},
		{name: "batch", args: []string{"--key", assignmentKey, "--batch-size", "1"}},
		{name: "workers", args: []string{"--key", assignmentKey, "--workers", "-1"}},
		{name: "invalid key", args: []string{"--key", assignmentKey, "--key", "bad"}},
		{name: "trailing stdin", args: []string{"-"}, input: assignmentKey + "\nbad\n"},
		{name: "old 88", version: "8.8", args: []string{"--key", assignmentKey}},
		{name: "old 89", version: "8.9", args: []string{"--key", assignmentKey}},
		{name: "JSON verbose", args: []string{"--key", assignmentKey, "--verbose"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			version := tc.version
			if version == "" {
				version = "8.10"
			}
			server, c := assignmentServer(t, "success", version)
			cfg := assignmentConfig(t, server, version, "json", "info")
			args := []string{"--config", cfg, "--auto-confirm", "--json", "update", "pi", "--set-bid", "Order"}
			args = append(args, tc.args...)
			stdout, _, err := assignmentCommand(t, args, tc.input, "configured", false, false, nil)
			require.Error(t, err)
			envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
			require.NotEqual(t, "succeeded", envelope["outcome"])
			c.mu.Lock()
			defer c.mu.Unlock()
			require.Zero(t, c.writes)
			require.Zero(t, c.reads)
		})
	}
}

// TestBusinessIDAssignmentEmptyMatrix verifies deliberate empty input has one
// successful envelope or zero key bytes, no prompt and no backend target traffic.
func TestBusinessIDAssignmentEmptyMatrix(t *testing.T) {
	for _, mode := range []string{"human", "json", "keys", "quiet", "quiet-json", "quiet-keys"} {
		for _, dry := range []bool{false, true} {
			for _, noWait := range []bool{false, true} {
				for _, confirm := range []string{"--auto-confirm", "--automation"} {
					t.Run(fmt.Sprintf("%s/dry=%t/noWait=%t/%s", mode, dry, noWait, confirm), func(t *testing.T) {
						server, c := assignmentServer(t, "success", "8.10")
						cfg := assignmentConfig(t, server, "8.10", "json", "info")
						args := []string{"--config", cfg, confirm, "update", "pi", "-", "--set-bid", "Order"}
						if strings.Contains(mode, "json") {
							args = append(args, "--json")
						}
						if strings.Contains(mode, "keys") {
							args = append(args, "--keys-only")
						}
						if strings.Contains(mode, "quiet") {
							args = append(args, "--quiet")
						}
						if dry {
							args = append(args, "--dry-run")
						}
						if noWait {
							args = append(args, "--no-wait")
						}
						stdout, stderr, err := assignmentCommand(t, args, " \n\n", "configured", false, false, nil)
						require.NoError(t, err, stderr)
						if strings.Contains(mode, "json") {
							envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
							require.Equal(t, "succeeded", envelope["outcome"])
							payload := requireJSONObject(t, envelope["payload"])
							require.Equal(t, true, payload["ready"])
							require.Equal(t, []any{}, payload["items"])
							require.Equal(t, []any{}, payload["requestedKeys"])
						} else {
							require.Empty(t, stdout)
						}
						records := bulkCompletionDiagnosticRecords(t, stderr, "json")
						if mode == "human" {
							label := "plan"
							if dry {
								label = "dry run"
							}
							require.Equal(t, [][2]string{{"INFO", label + ": business-ID assignment: nothing to update"}}, records)
						} else {
							require.Empty(t, records)
						}
						c.mu.Lock()
						defer c.mu.Unlock()
						require.Zero(t, c.reads)
						require.Zero(t, c.writes)
					})
				}
			}
		}
	}
}

// TestBusinessIDAssignmentStdinAndKeys checks full-input deduplication, preview
// key meaning, and accepted key output without diagnostics on stdout.
func TestBusinessIDAssignmentStdinAndKeys(t *testing.T) {
	for _, dry := range []bool{false, true} {
		t.Run(fmt.Sprint(dry), func(t *testing.T) {
			server, c := assignmentServer(t, "success", "8.10")
			cfg := assignmentConfig(t, server, "8.10", "json", "info")
			args := []string{"--config", cfg, "--auto-confirm", "--keys-only", "update", "pi", "-", "--key", assignmentKey, "--set-bid", "Order"}
			if dry {
				args = append(args, "--dry-run")
			}
			stdout, stderr, err := assignmentCommand(t, args, assignmentOtherKey+"\n"+assignmentKey+"\n", "inherited", false, false, nil)
			require.NoError(t, err, stderr)
			require.Equal(t, assignmentKey+"\n"+assignmentOtherKey+"\n", stdout)
			require.Empty(t, stderr)
			c.mu.Lock()
			defer c.mu.Unlock()
			if dry {
				require.Zero(t, c.writes)
				require.Equal(t, 2, c.reads)
			} else {
				require.Equal(t, 2, c.writes)
				require.Equal(t, 4, c.reads)
			}
		})
	}
}

// TestBusinessIDAssignmentTerminal proves prompt routing on actual terminal
// stdin, including redirected stdout, decline, EOF and prompt-free empty input.
func TestBusinessIDAssignmentTerminal(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, tc := range []struct {
			name, response                string
			eof, empty, dry, auto, noWait bool
			writes                        int
		}{
			{name: "accept", response: "yes", writes: 1},
			{name: "decline", response: "no"},
			{name: "EOF", eof: true},
			{name: "empty", empty: true},
			{name: "empty no-wait", empty: true, noWait: true},
			{name: "empty dry", empty: true, dry: true},
			{name: "dry", dry: true},
			{name: "auto-confirm", auto: true, writes: 1},
		} {
			t.Run(destination+"/"+tc.name, func(t *testing.T) {
				server, c := assignmentServer(t, "success", "8.10")
				cfg := assignmentConfig(t, server, "8.10", "plain-time", "info")
				args := []string{"--config", cfg, "update", "pi", "--set-bid", "Order"}
				if tc.empty {
					args = append(args, "-")
				} else {
					args = append(args, "--key", assignmentKey)
				}
				if tc.dry {
					args = append(args, "--dry-run")
				}
				if tc.noWait {
					args = append(args, "--no-wait")
				}
				if tc.auto {
					args = append(args, "--auto-confirm")
				}
				var exchanges []testx.CmdTerminalExchange
				prompt := "You are about to assign a business ID to 1 process instance(s).\nDo you want to proceed? [y/N]: "
				if tc.empty {
					exchanges = []testx.CmdTerminalExchange{{Prompt: assignmentReady, EndOfInput: true}}
				} else if !tc.dry && !tc.auto {
					exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: tc.response, EndOfInput: tc.eof}}
				}
				stdout, stderr, err := assignmentCommand(t, args, "", destination, true, tc.empty, exchanges)
				if tc.name == "decline" || tc.eof {
					require.Error(t, err)
				} else {
					require.NoError(t, err, stderr)
				}
				require.Empty(t, stdout)
				if tc.empty || tc.dry || tc.auto {
					require.NotContains(t, stderr, "Do you want to proceed?")
				} else {
					require.Equal(t, 1, strings.Count(stderr, prompt))
				}
				c.mu.Lock()
				defer c.mu.Unlock()
				require.Equal(t, tc.writes, c.writes)
				reads := 1 + tc.writes
				if tc.empty {
					reads = 0
				}
				require.Equal(t, reads, c.reads)
			})
		}
	}
}

// TestBusinessIDAssignmentEvidence keeps preview, acceptance, confirmation
// failures and fail-fast slots distinct through exactly one shared envelope.
func TestBusinessIDAssignmentEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, scenario, outcome, status, confirmation string
		flags                                         []string
		reads, writes                                 int
		failed                                        bool
	}{
		{name: "preview no-wait", scenario: "success", outcome: "succeeded", flags: []string{"--dry-run", "--no-wait"}, reads: 2},
		{name: "accepted", scenario: "success", outcome: "accepted", status: "accepted", confirmation: "skipped", flags: []string{"--no-wait"}, reads: 2, writes: 2},
		{name: "confirmed", scenario: "success", outcome: "succeeded", status: "confirmed", confirmation: "confirmed", reads: 4, writes: 2},
		{name: "accepted unconfirmed", scenario: "unconfirmed", outcome: "invalid", status: "accepted", confirmation: "unconfirmed", reads: 4, writes: 2, failed: true},
		{name: "fail fast", scenario: "reject", outcome: "invalid", status: "rejected", confirmation: "skipped", flags: []string{"--fail-fast", "--workers", "1"}, reads: 2, writes: 1, failed: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, c := assignmentServer(t, tc.scenario, "8.10")
			cfg := assignmentConfig(t, server, "8.10", "json", "info")
			args := []string{"--config", cfg, "--automation", "--json", "update", "pi", "--key", assignmentKey, "--key", assignmentOtherKey, "--set-bid", "Order"}
			args = append(args, tc.flags...)
			stdout, stderr, err := assignmentCommand(t, args, "", "configured", false, false, nil)
			if tc.failed {
				require.Error(t, err)
			} else {
				require.NoError(t, err, stderr)
			}
			require.Empty(t, stderr)
			envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
			require.Equal(t, tc.outcome, envelope["outcome"])
			if tc.failed {
				require.Equal(t, "invalid_input", envelope["class"])
			}
			payload := requireJSONObject(t, envelope["payload"])
			items := payload["items"].([]any)
			require.Len(t, items, 2)
			if tc.status != "" {
				first := requireJSONObject(t, items[0])
				require.Equal(t, tc.status, first["status"])
				require.Equal(t, tc.confirmation, first["confirmation"])
				if tc.name == "fail fast" {
					require.Equal(t, "not_submitted", requireJSONObject(t, items[1])["status"])
				}
			} else {
				require.Equal(t, true, payload["ready"])
				require.NotContains(t, requireJSONObject(t, items[0]), "status")
			}
			c.mu.Lock()
			defer c.mu.Unlock()
			require.Equal(t, tc.reads, c.reads)
			require.Equal(t, tc.writes, c.writes)
		})
	}
}

// TestBusinessIDAssignmentFrozenApproval proves changes to flag storage during
// approval cannot change the service-owned scope or assignment value.
func TestBusinessIDAssignmentFrozenApproval(t *testing.T) {
	server, c := assignmentServer(t, "success", "8.10")
	cfg := assignmentConfig(t, server, "8.10", "json", "info")
	previous := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = previous })
	approvals := 0
	confirmCmdOrAbortFn = func(_ io.Writer, _ bool, _ string) error {
		approvals++
		flagUpdatePISetBID = "changed after review"
		flagUpdatePIKeys = []string{assignmentOtherKey}
		return nil
	}
	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t, "--config", cfg, "--auto-confirm", "--json", "update", "pi", "--key", assignmentKey, "--set-bid", "original", "--no-wait")
	envelope := decodeSingleProcessInstanceOutcomeEnvelope(t, stdout)
	require.Equal(t, "accepted", envelope["outcome"])
	require.Equal(t, 1, approvals)
	c.mu.Lock()
	defer c.mu.Unlock()
	require.Equal(t, map[string]string{assignmentKey: "original"}, c.values)
	require.Equal(t, 1, c.reads)
	require.Equal(t, 1, c.writes)
}
