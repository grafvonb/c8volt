// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const scopeElementC = "2251799813689004"

// elementUpdateFixture keeps all planning and mutation evidence on a test-owned server.
// Callbacks are installed before command execution; the mutex serializes worker requests.
type elementUpdateFixture struct {
	server              *httptest.Server
	mu                  sync.Mutex
	calls               []string
	writes              map[string]map[string]any
	reads               map[string]int
	local               map[string][]string
	lookupFailure       string
	mutationFailure     func(http.ResponseWriter, string) bool
	confirmationFailure bool
}

// newElementUpdateFixture serves exact identities and authoritative local pages only.
func newElementUpdateFixture(t *testing.T) *elementUpdateFixture {
	t.Helper()
	f := &elementUpdateFixture{writes: map[string]map[string]any{}, reads: map[string]int{}, local: map[string][]string{}}
	f.server = testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/v2/element-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/element-instances/")
			if key == f.lookupFailure {
				w.WriteHeader(404)
				fmt.Fprint(w, `{}`)
				return
			}
			fmt.Fprint(w, scopeElementJSON(key))
		case r.URL.Path == "/v2/variables/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			key := filter["scopeKey"].(string)
			require.Equal(t, scopeOwner, filter["processInstanceKey"])
			require.NotContains(t, filter, "tenantId")
			require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
			f.reads[key]++
			rows := f.local[key]
			if values, ok := f.writes[key]; ok {
				if f.confirmationFailure {
					w.WriteHeader(400)
					fmt.Fprint(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
					return
				}
				rows = nil
				for name, value := range values {
					encoded, err := json.Marshal(value)
					require.NoError(t, err)
					rows = append(rows, scopeVariableJSON(name, string(encoded), key, false))
				}
				// Preserve unchanged/untouched locals during confirmation as the real store would.
				for _, raw := range f.local[key] {
					var row map[string]any
					require.NoError(t, json.Unmarshal([]byte(raw), &row))
					if _, changed := values[row["name"].(string)]; !changed {
						rows = append(rows, raw)
					}
				}
			}
			fmt.Fprint(w, scopePage(rows, len(rows), false))
		case r.Method == "PUT" && strings.HasSuffix(r.URL.Path, "/variables"):
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/element-instances/"), "/variables")
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, true, body["local"])
			require.NotContains(t, f.writes, key, "accepted destinations must never be resubmitted")
			if f.mutationFailure != nil && f.mutationFailure(w, key) {
				return
			}
			f.writes[key] = requireJSONObject(t, body["variables"])
			w.WriteHeader(204)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

// snapshot copies request evidence after the command exits without racing handlers.
func (f *elementUpdateFixture) snapshot() ([]string, map[string]map[string]any, map[string]int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := append([]string{}, f.calls...)
	writes := map[string]map[string]any{}
	reads := map[string]int{}
	for key, value := range f.writes {
		writes[key] = value
	}
	for key, value := range f.reads {
		reads[key] = value
	}
	return calls, writes, reads
}

// TestElementScopeVariablesUpdateExact proves ordered deduplication, independent owner/scope,
// frozen mutation-only values and normal/no-wait confirmation across supported adapters.
func TestElementScopeVariablesUpdateExact(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, noWait := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/noWait=%t", version, noWait), func(t *testing.T) {
				f := newElementUpdateFixture(t)
				f.local[scopeElementA] = []string{scopeVariableJSON("same", "1", scopeElementA, false), scopeVariableJSON("change", "false", scopeElementA, false), scopeVariableJSON("untouched", "2", scopeElementA, false)}
				f.local[scopeElementB] = []string{scopeVariableJSON("same", "1", scopeElementB, false), scopeVariableJSON("change", "true", scopeElementB, false), scopeVariableJSON("added", "null", scopeElementB, false)}
				args := []string{"--tenant", "other-tenant", "--json", "--auto-confirm", "update", "ei", "-", "--key", scopeElementA + "," + scopeElementB, "-k", scopeElementA, "--vars", `{"same":1,"change":true,"added":null}`, "--workers", "1"}
				if noWait {
					args = append(args, "--no-wait")
				}
				out, stderr, err := runUpdateUserTaskCommand(t, testx.WriteTestConfigForVersion(t, f.server.URL, version), scopeElementB+"\n"+scopeElementA+"\n", args...)
				require.NoError(t, err, stderr)
				require.Empty(t, stderr)
				env := requireSingleJSONObjectDocument(t, out)
				outcome := "succeeded"
				status := "confirmed"
				reads := 2
				if noWait {
					outcome = "accepted"
					status = "submitted"
					reads = 1
				}
				require.Equal(t, outcome, env["outcome"])
				require.Equal(t, "update element", env["command"])
				items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 2)
				first := requireJSONObject(t, items[0])
				second := requireJSONObject(t, items[1])
				require.Equal(t, scopeElementA, first["key"])
				require.Equal(t, status, first["status"])
				require.Equal(t, true, first["mutationAccepted"])
				require.Equal(t, scopeOwner, first["processInstanceKey"])
				require.Equal(t, scopeElementA, first["scopeKey"])
				require.Equal(t, "accepted", first["submissionStatus"])
				require.Equal(t, "unchanged", second["status"])
				calls, writes, readCounts := f.snapshot()
				require.Equal(t, map[string]map[string]any{scopeElementA: {"change": true, "added": nil}}, writes)
				require.Equal(t, map[string]int{scopeElementA: reads, scopeElementB: 1}, readCounts)
				require.Len(t, calls, 3+reads+1)
				require.Equal(t, []string{"GET /v2/element-instances/" + scopeElementA, "POST /v2/variables/search", "GET /v2/element-instances/" + scopeElementB, "POST /v2/variables/search"}, calls[:4])
			})
		}
	}
}

// TestElementScopeVariablesUpdateInput rejects every locally detectable error before requests.
func TestElementScopeVariablesUpdateInput(t *testing.T) {
	file := filepath.Join(t.TempDir(), "vars.json")
	require.NoError(t, os.WriteFile(file, []byte(`{"x":null}`), 0600))
	cases := [][]string{
		{"--key", scopeElementA}, {"--vars", "{}"}, {"--key", "bad", "--vars", "{}"},
		{"--key", scopeElementA + ",", "--vars", "{}"}, {"--key", scopeElementA, "--vars", "null"},
		{"--key", scopeElementA, "--vars", "[]"}, {"--key", scopeElementA, "--vars", "true"},
		{"--key", scopeElementA, "--vars", "{"}, {"--key", scopeElementA, "--vars", "{} {}"},
		{"--key", scopeElementA, "--vars", "{}", "--vars-file", file},
		{"--key", scopeElementA, "--vars-file", file + "-missing"},
		{"--key", scopeElementA, "--vars", "{}", "--workers", "0"},
		{"--key", scopeElementA, "--vars", "{}", "--json"},
		{"--key", scopeElementA, "--vars", "{}", "--json", "--verbose", "--dry-run"},
		{"-", "--vars", "{}"}, {"unexpected", "--vars", "{}"},
	}
	for _, flags := range cases {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			f := newElementUpdateFixture(t)
			_, _, err := runUpdateUserTaskCommand(t, testx.WriteTestConfig(t, f.server.URL), "", append([]string{"update", "element"}, flags...)...)
			require.Error(t, err)
			calls, _, _ := f.snapshot()
			require.Empty(t, calls)
		})
	}
	for _, payload := range []string{`{"x":null}`, `[]`, `null`, `{`} {
		t.Run("file/"+payload, func(t *testing.T) {
			f := newElementUpdateFixture(t)
			require.NoError(t, os.WriteFile(file, []byte(payload), 0600))
			out, stderr, err := runUpdateUserTaskCommand(t, testx.WriteTestConfig(t, f.server.URL), "", "--json", "update", "element", "--key", scopeElementA, "--vars-file", file, "--dry-run")
			calls, _, _ := f.snapshot()
			if payload == `{"x":null}` {
				require.NoError(t, err, stderr)
				require.Len(t, calls, 2)
				require.Equal(t, "succeeded", requireSingleJSONObjectDocument(t, out)["outcome"])
			} else {
				require.Error(t, err)
				require.Empty(t, calls)
			}
		})
	}
}

// TestElementScopeVariablesUpdatePlanningFailure prevents all writes on a late missing target.
func TestElementScopeVariablesUpdatePlanningFailure(t *testing.T) {
	f := newElementUpdateFixture(t)
	f.lookupFailure = scopeElementB
	out, stderr, err := runUpdateUserTaskCommand(t, testx.WriteTestConfig(t, f.server.URL), "", "--json", "--auto-confirm", "update", "element", "--key", scopeElementA+","+scopeElementB, "--vars", `{"x":1}`)
	require.Error(t, err)
	require.Empty(t, stderr)
	env := requireSingleJSONObjectDocument(t, out)
	require.Equal(t, "not_found", env["class"])
	calls, writes, reads := f.snapshot()
	require.Len(t, calls, 3)
	require.Empty(t, writes)
	require.Equal(t, map[string]int{scopeElementA: 1}, reads)
}

// TestElementScopeVariablesUpdateUnsupported preserves bootstrap failure before any request.
func TestElementScopeVariablesUpdateUnsupported(t *testing.T) {
	f := newElementUpdateFixture(t)
	out, stderr, err := runUpdateUserTaskCommand(t, testx.WriteTestConfigForVersion(t, f.server.URL, "8.7"), "", "--json", "update", "element", "--key", scopeElementA, "--vars", "{}", "--dry-run")
	assertUserTaskCompletionExit(t, err, 1)
	require.Empty(t, out)
	require.Contains(t, stderr, "8.7")
	calls, _, _ := f.snapshot()
	require.Empty(t, calls)
}
