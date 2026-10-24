// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// completionRequest captures only native completion traffic; every other route fails.
type completionRequest struct {
	Key  string
	Body map[string]any
}

// newUserTaskCompletionServer permits one native preflight read and completion only.
func newUserTaskCompletionServer(t *testing.T, respond func(http.ResponseWriter, *http.Request, string)) (*httptest.Server, *testx.SafeSlice[completionRequest]) {
	t.Helper()
	server, calls, _ := newUserTaskCompletionPreflightServer(t, nil, respond)
	return server, calls
}

// newUserTaskCompletionPreflightServer rejects repeated lookups, polling and unrelated operations.
func newUserTaskCompletionPreflightServer(t *testing.T, lookup, respond func(http.ResponseWriter, *http.Request, string)) (*httptest.Server, *testx.SafeSlice[completionRequest], *testx.SafeSlice[string]) {
	t.Helper()
	calls := &testx.SafeSlice[completionRequest]{}
	reads := &testx.SafeSlice[string]{}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/v2/user-tasks/")
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/user-tasks/") && !strings.Contains(key, "/") {
			for _, previous := range reads.Snapshot() {
				if previous == key {
					t.Errorf("repeated task lookup: %s", key)
				}
			}
			reads.Append(key)
			if lookup != nil {
				lookup(w, r, key)
			} else {
				writeCompletionTask(w, key, "CREATED", "alice")
			}
			return
		}
		if r.Method != http.MethodPost || !strings.HasPrefix(r.URL.Path, "/v2/user-tasks/") || !strings.HasSuffix(r.URL.Path, "/completion") {
			t.Errorf("forbidden completion workflow request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		key = strings.TrimSuffix(key, "/completion")
		require.Contains(t, reads.Snapshot(), key, "completion must follow its preflight lookup")
		var body map[string]any
		decoder := json.NewDecoder(r.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&body); err != nil {
			t.Errorf("decode completion: %v", err)
			w.WriteHeader(400)
			return
		}
		calls.Append(completionRequest{key, body})
		if respond != nil {
			respond(w, r, key)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)
	return server, calls, reads
}

// writeCompletionTask supplies the minimal native task state used by all supported adapters.
func writeCompletionTask(w http.ResponseWriter, key, state, assignee string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"userTaskKey": key, "state": state, "assignee": assignee, "processInstanceKey": "2251799813685240"})
}

// decodeUserTaskCompletionEnvelope requires exactly one JSON value, including
// failures whose partial evidence must not be followed by a second renderer.
func decodeUserTaskCompletionEnvelope(t *testing.T, stdout string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
	require.Equal(t, "update user-task", envelope["command"])
	return envelope
}

// TestUpdateUserTaskCompletionCommandSelection exercises actual dispatch, all
// aliases, key sources and supported adapters without remote selection reads.
func TestUpdateUserTaskCompletionCommandSelection(t *testing.T) {
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
			{"uts", []string{"-"}, "\n" + completionKeyB + "\n" + completionKeyA + "\n" + completionKeyB + "\n", []string{completionKeyB, completionKeyA}},
			{"ut", []string{"-", "--key", completionKeyA}, completionKeyB + "\n" + completionKeyA + "\n", []string{completionKeyA, completionKeyB}},
			{"ut", []string{"-", "--key", completionKeyA}, "\n", []string{completionKeyA}},
		} {
			t.Run(version+"/"+tc.alias+"/"+strings.Join(tc.flags, " "), func(t *testing.T) {
				server, calls := newUserTaskCompletionServer(t, nil)
				args := append([]string{"--automation", "--json", "--tenant", "different-tenant", "update", tc.alias, "--complete", "--workers", "1"}, tc.flags...)
				stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, version), tc.stdin, args...)
				require.NoError(t, err, stderr)
				require.Empty(t, stderr)
				envelope := decodeUserTaskCompletionEnvelope(t, stdout)
				require.Equal(t, "accepted", envelope["outcome"])
				payload := requireJSONObject(t, envelope["payload"])
				keys := []any{}
				for _, key := range tc.keys {
					keys = append(keys, key)
				}
				require.Equal(t, keys, payload["requestedKeys"])
				require.Equal(t, false, payload["variablesProvided"])
				rows := calls.Snapshot()
				require.Len(t, rows, len(tc.keys))
				for i, row := range rows {
					require.Equal(t, tc.keys[i], row.Key)
					require.Equal(t, map[string]any{"action": nil, "variables": nil}, row.Body)
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

// TestUpdateUserTaskCompletionCommandVariables proves omitted, empty and matching
// values still submit, with identical typed data on every native request.
func TestUpdateUserTaskCompletionCommandVariables(t *testing.T) {
	nested := `{"approved":true,"count":9007199254740993,"nested":{"array":[null,"x",false,0.123456789012345678901,{"negative":-9007199254740993}]}}`
	file := filepath.Join(t.TempDir(), "variables.json")
	require.NoError(t, os.WriteFile(file, []byte(nested), 0600))
	for _, tc := range []struct {
		name  string
		flags []string
		want  map[string]any
	}{
		{"omitted", nil, nil}, {"empty", []string{"--vars", "{}"}, map[string]any{}},
		{"matching", []string{"--vars", `{"approved":true}`}, map[string]any{"approved": true}},
		{"inline", []string{"--vars", nested}, map[string]any{"approved": true, "count": json.Number("9007199254740993"), "nested": map[string]any{"array": []any{nil, "x", false, json.Number("0.123456789012345678901"), map[string]any{"negative": json.Number("-9007199254740993")}}}}},
		{"file", []string{"--vars-file", file}, map[string]any{"approved": true, "count": json.Number("9007199254740993"), "nested": map[string]any{"array": []any{nil, "x", false, json.Number("0.123456789012345678901"), map[string]any{"negative": json.Number("-9007199254740993")}}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := newUserTaskCompletionServer(t, nil)
			args := append([]string{"update", "ut", "--complete", "--key", completionKeyA + "," + completionKeyB, "--auto-confirm", "--json"}, tc.flags...)
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			payload := requireJSONObject(t, decodeUserTaskCompletionEnvelope(t, stdout)["payload"])
			require.Equal(t, tc.want != nil, payload["variablesProvided"])
			require.Equal(t, float64(len(tc.want)), payload["variableCount"])
			require.Len(t, calls.Snapshot(), 2)
			for _, call := range calls.Snapshot() {
				if tc.want == nil {
					require.Nil(t, call.Body["variables"])
				} else {
					require.Equal(t, tc.want, call.Body["variables"])
				}
			}
		})
	}
}

// TestUpdateUserTaskCompletionCommandInvalidInputs ensures whole-input validation
// prevents every backend request, including invalid late tokens and payloads.
func TestUpdateUserTaskCompletionCommandInvalidInputs(t *testing.T) {
	empty := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(empty, nil, 0600))
	bad := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(bad, []byte(`{} {}`), 0600))
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
		{"short key", []string{"--key", "123"}, ""},
		{"zero workers", []string{"--key", completionKeyA, "--workers", "0"}, ""},
		{"negative workers", []string{"--key", completionKeyA, "--workers", "-1"}, ""},
		{"no wait", []string{"--key", completionKeyA, "--no-wait"}, ""},
		{"sources", []string{"--key", completionKeyA, "--vars", "{}", "--vars-file", bad}, ""},
		{"missing file", []string{"--key", completionKeyA, "--vars-file", filepath.Join(t.TempDir(), "missing")}, ""},
		{"empty file", []string{"--key", completionKeyA, "--vars-file", empty}, ""},
		{"bad file", []string{"--key", completionKeyA, "--vars-file", bad}, ""},
		{"malformed", []string{"--key", completionKeyA, "--vars", "{"}, ""},
		{"array", []string{"--key", completionKeyA, "--vars", "[]"}, ""},
		{"null", []string{"--key", completionKeyA, "--vars", "null"}, ""},
		{"scalar", []string{"--key", completionKeyA, "--vars", "42"}, ""},
		{"trailing", []string{"--key", completionKeyA, "--vars", "{} {}"}, ""},
		{"empty payload", []string{"--key", completionKeyA, "--vars="}, ""},
		{"verbose json", []string{"--key", completionKeyA, "--verbose"}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := newUserTaskCompletionServer(t, nil)
			args := append([]string{"--json", "--auto-confirm", "update", "ut", "--complete"}, tc.flags...)
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), tc.stdin, args...)
			assertUserTaskCompletionExit(t, err, 2)
			require.Empty(t, stderr)
			envelope := decodeUserTaskCompletionEnvelope(t, stdout)
			require.Equal(t, "invalid", envelope["outcome"])
			require.Equal(t, "invalid_input", envelope["class"])
			require.Empty(t, envelope["payload"])
			require.Empty(t, calls.Snapshot())
		})
	}
}

// TestUpdateUserTaskCompletionCommandGuardrails retains explicit false variable
// dispatch, JSON confirmation requirements and generic unsupported-version errors.
func TestUpdateUserTaskCompletionCommandGuardrails(t *testing.T) {
	for _, tc := range []struct {
		name, version, class string
		flags                []string
		code                 int
	}{
		{"json needs confirmation", "8.10", "invalid_input", []string{"--complete", "--json"}, 2},
		{"false requires variables", "8.10", "invalid_input", []string{"--complete=false", "--auto-confirm", "--json"}, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server, calls := newUserTaskCompletionServer(t, nil)
			args := append([]string{"update", "ut", "--key", completionKeyA}, tc.flags...)
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, tc.version), "", args...)
			assertUserTaskCompletionExit(t, err, tc.code)
			require.Empty(t, stderr)
			require.Equal(t, tc.class, decodeUserTaskCompletionEnvelope(t, stdout)["class"])
			require.Empty(t, calls.Snapshot())
		})
	}
	// Unknown versions fail in the existing bootstrap before output configuration.
	serverUnsupported, callsUnsupported := newUserTaskCompletionServer(t, nil)
	unsupportedOut, unsupportedErr, runErr := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, serverUnsupported.URL, "8.7"), "", "update", "ut", "--complete", "--key", completionKeyA, "--auto-confirm")
	assertUserTaskCompletionExit(t, runErr, 1)
	require.Empty(t, unsupportedOut)
	require.Regexp(t, `^\d{4}/\d{2}/\d{2} .* ERROR local precondition failed: normalize config: app: version: unknown Camunda version: 8.7\n$`, unsupportedErr)
	require.Empty(t, callsUnsupported.Snapshot())
	server, capture := newUpdateUserTaskCommandServer(t)
	stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", "--automation", "--json", "update", "ut", "--complete=false", "--key", "2251799815391233", "--vars", `{"approved":true}`, "--no-wait")
	require.NoError(t, err, stderr)
	envelope := decodeUserTaskCompletionEnvelope(t, stdout)
	require.Equal(t, "accepted", envelope["outcome"])
	require.NotContains(t, requireJSONObject(t, envelope["payload"]), "operation")
	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 1, capture.mutations)
}
