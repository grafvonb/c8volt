// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// assignmentRequest captures native intent; no other operation is allowed.
type assignmentRequest struct {
	Key  string
	Body map[string]any
}

// newUserTaskAssignmentServer supplies unassigned native lookup results for assignment
// preparation and records only mutations. Dedicated preflight tests count every request.
func newUserTaskAssignmentServer(t *testing.T, operation string, respond func(http.ResponseWriter, *http.Request, string)) (*httptest.Server, *testx.SafeSlice[assignmentRequest]) {
	t.Helper()
	calls := &testx.SafeSlice[assignmentRequest]{}
	method, suffix := http.MethodPost, "/assignment"
	if operation == "unassign" {
		method, suffix = http.MethodDelete, "/assignee"
	}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		if operation == "assign" && r.Method == http.MethodGet {
			key := strings.TrimPrefix(r.URL.Path, "/v2/user-tasks/")
			if len(key) != 16 {
				t.Errorf("unexpected assignment lookup: %s", r.URL.Path)
				w.WriteHeader(400)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"userTaskKey": key, "processInstanceKey": completionKeyB, "state": "CREATED", "assignee": ""}))
			return
		}
		key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/user-tasks/"), suffix)
		if r.Method != method || r.URL.Path != "/v2/user-tasks/"+key+suffix || len(key) != 16 {
			t.Errorf("forbidden assignment request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
			return
		}
		var body map[string]any
		if operation == "assign" {
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Len(t, body, 3)
			require.Contains(t, body, "assignee")
			require.Contains(t, body, "allowOverride")
			require.Nil(t, body["action"])
		} else {
			data, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Empty(t, data)
		}
		calls.Append(assignmentRequest{key, body})
		if respond != nil {
			respond(w, r, key)
		} else {
			w.WriteHeader(204)
		}
	}))
	t.Cleanup(server.Close)
	return server, calls
}

// assignmentFlags chooses the native mode while preserving an opaque identity.
func assignmentFlags(operation string) []string {
	if operation == "unassign" {
		return []string{"--unassign"}
	}
	return []string{"--assign", " alice\nopaque "}
}

// TestUpdateUserTaskAssignmentCommandSelection checks aliases, all input sources,
// supported adapters, tenant bypass and explicit false/true replacement permission.
func TestUpdateUserTaskAssignmentCommandSelection(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		for _, version := range []string{"8.8", "8.9", "8.10"} {
			for _, tc := range []struct {
				alias string
				flags []string
				stdin string
				keys  []string
			}{
				{"user-task", []string{"-k", completionKeyA}, "", []string{completionKeyA}},
				{"user-tasks", []string{"--key", completionKeyA, "--key", completionKeyB, "--key", completionKeyA}, "", []string{completionKeyA, completionKeyB}},
				{"ut", []string{"--key", completionKeyB + "," + completionKeyA + "," + completionKeyB}, "", []string{completionKeyB, completionKeyA}},
				{"uts", []string{"-"}, "\n" + completionKeyB + "\n" + completionKeyA + "\n", []string{completionKeyB, completionKeyA}},
				{"ut", []string{"-", "--key", completionKeyA}, completionKeyB + "\n" + completionKeyA + "\n", []string{completionKeyA, completionKeyB}},
				{"ut", []string{"-", "--key", completionKeyA}, "\n", []string{completionKeyA}},
			} {
				t.Run(operation+"/"+version+"/"+tc.alias+strings.Join(tc.flags, " "), func(t *testing.T) {
					server, calls := newUserTaskAssignmentServer(t, operation, nil)
					args := append([]string{"--automation", "--json", "--tenant", "different-tenant", "update", tc.alias, "--workers", "1", "--no-wait=false", "--complete=false"}, assignmentFlags(operation)...)
					override := operation == "assign" && tc.alias == "uts"
					if override {
						args = append(args, "--force")
					} else if operation == "assign" {
						args = append(args, "--force=false", "--unassign=false")
					}
					args = append(args, tc.flags...)
					stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, version), tc.stdin, args...)
					require.NoError(t, err, stderr)
					require.Empty(t, stderr)
					envelope := decodeUserTaskCompletionEnvelope(t, stdout)
					require.Equal(t, "accepted", envelope["outcome"])
					payload := requireJSONObject(t, envelope["payload"])
					require.Equal(t, operation, payload["operation"])
					want := []any{}
					for _, key := range tc.keys {
						want = append(want, key)
					}
					require.Equal(t, want, payload["requestedKeys"])
					require.NotContains(t, payload, "variablesProvided")
					rows := calls.Snapshot()
					require.Len(t, rows, len(tc.keys))
					for i, row := range rows {
						require.Equal(t, tc.keys[i], row.Key)
						if operation == "assign" {
							require.Equal(t, map[string]any{"action": nil, "assignee": " alice\nopaque ", "allowOverride": override}, row.Body)
						}
					}
					if operation == "assign" {
						require.Equal(t, " alice\nopaque ", payload["assignee"])
						require.Equal(t, override, payload["allowOverride"])
					} else {
						require.NotContains(t, payload, "assignee")
						require.NotContains(t, payload, "allowOverride")
					}
					tenant := requireJSONObject(t, envelope["tenantContext"])
					require.Equal(t, "explicit_keys", tenant["mode"])
					require.Equal(t, "not_applied", tenant["filter"])
					require.Equal(t, "different-tenant", tenant["configuredTenantId"])
					require.Equal(t, []any{}, tenant["resolvedTenantIds"])
				})
			}
		}
	}
}

// TestUpdateUserTaskAssignmentCommandInvalidInputs proves whole-input validation
// rejects late tokens and flag presence conflicts without requests or invented reports.
func TestUpdateUserTaskAssignmentCommandInvalidInputs(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		for _, tc := range []struct {
			name  string
			flags []string
			stdin string
		}{
			{"missing", nil, ""}, {"empty stdin", []string{"-"}, "\n"},
			{"late flag", []string{"--key", completionKeyA + ",bad"}, ""},
			{"late stdin", []string{"-", "--key", completionKeyA}, completionKeyB + "\nbad\n"},
			{"empty flag", []string{"--key", completionKeyA, "--key="}, ""},
			{"empty component", []string{"--key", completionKeyA + ","}, ""},
			{"workers zero", []string{"--key", completionKeyA, "--workers", "0"}, ""},
			{"workers negative", []string{"--key", completionKeyA, "--workers", "-1"}, ""},
			{"no wait", []string{"--key", completionKeyA, "--no-wait"}, ""},
			{"completion", []string{"--key", completionKeyA, "--complete"}, ""},
			{"vars", []string{"--key", completionKeyA, "--vars="}, ""},
			{"vars file", []string{"--key", completionKeyA, "--vars-file="}, ""},
			{"verbose json", []string{"--key", completionKeyA, "--verbose"}, ""},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				server, calls := newUserTaskAssignmentServer(t, operation, nil)
				args := append([]string{"--json", "--auto-confirm", "update", "ut"}, assignmentFlags(operation)...)
				args = append(args, tc.flags...)
				stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), tc.stdin, args...)
				assertUserTaskCompletionExit(t, err, 2)
				require.Empty(t, stderr)
				require.Empty(t, calls.Snapshot())
				envelope := decodeUserTaskCompletionEnvelope(t, stdout)
				require.Equal(t, "invalid_input", envelope["class"])
				require.NotContains(t, envelope, "payload")
			})
		}
	}
	for _, flags := range [][]string{{"--assign="}, {"--assign", " \t "}, {"--assign", "alice", "--unassign"}, {"--unassign", "--force=false"}, {"--force=false"}, {"--unassign=false", "--complete=false"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			server, calls := newUserTaskAssignmentServer(t, "assign", nil)
			args := append([]string{"--json", "--auto-confirm", "update", "ut", "--key", completionKeyA}, flags...)
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", args...)
			assertUserTaskCompletionExit(t, err, 2)
			require.Empty(t, stderr)
			require.Empty(t, calls.Snapshot())
			require.Equal(t, "invalid_input", decodeUserTaskCompletionEnvelope(t, stdout)["class"])
		})
	}
}

// assignmentModeFlag keeps action selection independent from output test matrices.
func assignmentModeFlag(operation string) string {
	if operation == "unassign" {
		return "--unassign"
	}
	return "--assign=alice"
}

// TestUpdateUserTaskAssignmentCommandGuardrails retains JSON confirmation guards,
// false-selector legacy dispatch and nonterminal prompt bypass without override.
func TestUpdateUserTaskAssignmentCommandGuardrails(t *testing.T) {
	for _, operation := range []string{"assign", "unassign"} {
		t.Run(operation, func(t *testing.T) {
			server, calls := newUserTaskAssignmentServer(t, operation, nil)
			path := writeTestConfigForVersion(t, server.URL, "8.10")
			stdout, stderr, err := runUpdateUserTaskCommand(t, path, "", "--json", "update", "ut", assignmentModeFlag(operation), "--key", completionKeyA)
			assertUserTaskCompletionExit(t, err, 2)
			require.Empty(t, stderr)
			require.Empty(t, calls.Snapshot())
			require.Equal(t, "invalid_input", decodeUserTaskCompletionEnvelope(t, stdout)["class"])
			// Pipe stdin retains the shared noninteractive behavior without auto-confirm.
			stdout, stderr, err = runUpdateUserTaskCommand(t, path, "", "--log-format", "plain-time", "update", "ut", assignmentModeFlag(operation), "--key", completionKeyA)
			require.NoError(t, err)
			require.Empty(t, stdout)
			require.NotContains(t, stderr, "Do you want to proceed?")
			require.Len(t, calls.Snapshot(), 1)
			if operation == "assign" {
				require.Equal(t, false, calls.Snapshot()[0].Body["allowOverride"])
			}
		})
	}
	server, capture := newUpdateUserTaskCommandServer(t)
	stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", "--automation", "--json", "update", "ut", "--unassign=false", "--complete=false", "--key", "2251799815391233", "--vars", `{"approved":true}`, "--no-wait")
	require.NoError(t, err, stderr)
	envelope := decodeUserTaskCompletionEnvelope(t, stdout)
	require.Equal(t, "accepted", envelope["outcome"])
	require.NotContains(t, requireJSONObject(t, envelope["payload"]), "operation")
	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 1, capture.mutations)
}
