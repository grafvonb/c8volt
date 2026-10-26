// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// suspensionTreeFixture adds discovery to the exact-key fake without weakening its
// rejection of unexpected requests. Its embedded mutex protects all observations.
type suspensionTreeFixture struct {
	*suspensionFixture
	observed        map[string]string
	children        map[string][]string
	discovery       []string
	discoveryStatus map[string]int
}

// newSuspensionTreeFixture serves a nested chain; callers may configure states or
// failures before execution, and terminal callbacks change them under the mutex.
func newSuspensionTreeFixture(t *testing.T) *suspensionTreeFixture {
	t.Helper()
	base := newSuspensionFixture(t)
	f := &suspensionTreeFixture{
		suspensionFixture: base,
		observed:          map[string]string{suspendKeyA: "ACTIVE", suspendKeyB: "ACTIVE", suspendKeyC: "ACTIVE"},
		children:          map[string][]string{suspendKeyA: {suspendKeyB}, suspendKeyB: {suspendKeyC}},
		discoveryStatus:   map[string]int{},
	}
	original := base.server.Config.Handler
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
		search := r.Method == "POST" && key == "search"
		seed := r.Method == "GET" && !strings.Contains(key, "/") && f.writes[key] == 0
		if !search && !seed {
			f.mu.Unlock()
			original.ServeHTTP(w, r)
			return
		}
		defer f.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		if search {
			var body struct {
				Filter map[string]json.RawMessage `json:"filter"`
				Page   struct {
					Limit int `json:"limit"`
				} `json:"page"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Len(t, body.Filter, 1, "only parent selection; no tenant/state pruning")
			require.NoError(t, json.Unmarshal(body.Filter["parentProcessInstanceKey"], &key))
			require.Equal(t, 1000, body.Page.Limit)
		}
		f.discovery = append(f.discovery, r.Method+" "+key)
		if status := f.discoveryStatus[key]; status != 0 {
			w.WriteHeader(status)
			fmt.Fprintf(w, `{"detail":"POST https://private.invalid/tree?token=suspension-token-secret body-only-secret cursor=private-cursor resource=%s"}`, key)
			return
		}
		state, ok := f.observed[key]
		require.True(t, ok, "no ancestor or unrelated reads: %s", key)
		if seed {
			fmt.Fprintf(w, `{"processInstanceKey":%q,"parentProcessInstanceKey":"2251799813685248","state":%q}`, key, state)
			return
		}
		items := []map[string]string{}
		for _, child := range f.children[key] {
			items = append(items, map[string]string{"processInstanceKey": child, "parentProcessInstanceKey": key, "state": f.observed[child]})
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]int{"totalItems": len(items)}}))
	}))
	t.Cleanup(server.Close)
	base.server = server
	return f
}

// discoverySnapshot separates planning reads from post-submission evidence.
func (f *suspensionTreeFixture) discoverySnapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.discovery...)
}

// TestSuspendProcessInstanceTreeSelection verifies alias/flag/stdin overlap, mixed
// ancestors, subtree-only scope, frozen observations, and no confirmation in no-wait.
func TestSuspendProcessInstanceTreeSelection(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		for _, childFirst := range []bool{false, true} {
			t.Run(fmt.Sprintf("noWait=%t/childFirst=%t", noWait, childFirst), func(t *testing.T) {
				f := newSuspensionTreeFixture(t)
				f.observed[suspendKeyA] = "SUSPENDED"
				f.observed[suspendKeyB] = "COMPLETED"
				seeds := suspendKeyA + "," + suspendKeyB
				if childFirst {
					seeds = suspendKeyB + "," + suspendKeyA
				}
				args := []string{"--tenant", "unrelated", "suspend", "pi", "--tree", "--key", seeds, "--key", suspendKeyA, "-", "--json", "--auto-confirm", "--workers", "1"}
				if noWait {
					args = append(args, "--no-wait")
				}
				out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), suspendKeyB+"\n", args...)
				require.NoError(t, err, stderr)
				require.Empty(t, stderr)
				env := requireSingleJSONObjectDocument(t, out)
				outcome := "succeeded"
				if noWait {
					outcome = "accepted"
				}
				require.Equal(t, outcome, env["outcome"])
				payload := requireJSONObject(t, env["payload"])
				plan := requireJSONObject(t, payload["plan"])
				require.Equal(t, []any{suspendKeyC}, plan["targetKeys"])
				requested := []any{suspendKeyA, suspendKeyB}
				if childFirst {
					requested = []any{suspendKeyB, suspendKeyA}
				}
				require.Equal(t, requested, plan["requestedKeys"])
				instances := requireJSONItems(t, plan["instances"], 3)
				classifications := map[string]any{}
				for _, v := range instances {
					item := requireJSONObject(t, v)
					classifications[item["key"].(string)] = item["classification"]
				}
				require.Equal(t, map[string]any{suspendKeyA: "already_suspended", suspendKeyB: "excluded", suspendKeyC: "eligible"}, classifications)
				item := requireJSONObject(t, requireJSONItems(t, payload["items"], 1)[0])
				require.Equal(t, suspendKeyC, item["key"])
				require.Equal(t, "accepted", item["submission"])
				require.Equal(t, !noWait, item["hasObservation"])
				_, writes, reads := f.snapshot()
				require.Equal(t, map[string]int{suspendKeyC: 1}, writes)
				if noWait {
					require.Empty(t, reads)
				} else {
					require.Equal(t, writes, reads)
				}
				discovery := f.discoverySnapshot()
				if childFirst {
					require.Len(t, discovery, 7)
				} else {
					require.Equal(t, []string{"GET " + suspendKeyA, "POST " + suspendKeyA, "POST " + suspendKeyB, "POST " + suspendKeyC}, discovery)
				}
			})
		}
	}
}

// newSuspensionModeFixture lets existing execution matrices also cover tree
// dispatch with flat explicit seeds. The exact-key fake still rejects discovery.
func newSuspensionModeFixture(t *testing.T, tree bool) *suspensionFixture {
	t.Helper()
	if !tree {
		return newSuspensionFixture(t)
	}
	f := newSuspensionTreeFixture(t)
	f.children = map[string][]string{}
	return f.suspensionFixture
}

// suspensionModeArgs opts existing matrices into tree dispatch without changing input.
func suspensionModeArgs(args []string, tree bool) []string {
	if tree {
		return append(args, "--tree")
	}
	return args
}

// suspensionTreeTestScope states expected flat-fixture counts independently of production views.
func suspensionTreeTestScope(n int) string {
	return fmt.Sprintf("selected process instances: %d; process instances in scope: %d; active targets: %d; already suspended: 0; excluded: 0", n, n, n)
}

// suspensionTreeTestApproval is the flat-fixture approval expectation.
func suspensionTreeTestApproval(n int) string {
	return fmt.Sprintf("plan: suspend %d process instance(s); %d selected; %d in scope; 0 already suspended; 0 excluded", n, n, n)
}

// suspensionExecutionTestHandler adds flat planning responses around the existing
// worker/cancellation barriers, rejecting any discovery once execution begins.
func suspensionExecutionTestHandler(t *testing.T, tree bool, keys []string, next http.Handler) http.Handler {
	t.Helper()
	var mu sync.Mutex
	executing := false
	discovered := []string{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !tree {
			next.ServeHTTP(w, r)
			return
		}
		mu.Lock()
		if strings.HasSuffix(r.URL.Path, "/suspension") {
			executing = true
			require.Equal(t, keys, discovered, "resolve all seeds before execution")
		}
		planning := !executing
		mu.Unlock()
		if !planning {
			require.NotEqual(t, "/v2/process-instances/search", r.URL.Path, "never rediscover approved scope")
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			require.Contains(t, keys, key)
			mu.Lock()
			discovered = append(discovered, key)
			mu.Unlock()
			fmt.Fprintf(w, `{"processInstanceKey":%q,"state":"ACTIVE"}`, key)
		} else {
			require.Equal(t, "POST", r.Method)
			require.Equal(t, "/v2/process-instances/search", r.URL.Path)
			fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
		}
	})
}

// requireSuspensionModePlan keeps frozen planning facts present in partial and
// interrupted tree results while protecting the exact-key payload's original shape.
func requireSuspensionModePlan(t *testing.T, payload map[string]any, tree bool, keys []string) {
	t.Helper()
	if !tree {
		require.NotContains(t, payload, "plan")
		return
	}
	plan := requireJSONObject(t, payload["plan"])
	expected := make([]any, len(keys))
	for i, key := range keys {
		expected[i] = key
	}
	require.Equal(t, expected, plan["requestedKeys"])
	require.Equal(t, expected, plan["targetKeys"])
	instances := requireJSONItems(t, plan["instances"], len(keys))
	for i, v := range instances {
		item := requireJSONObject(t, v)
		require.Equal(t, keys[i], item["key"])
		require.Equal(t, "ACTIVE", item["observedState"])
		require.Equal(t, "eligible", item["classification"])
	}
}
