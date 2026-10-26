// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const suspendKeyA = "2251799813685249"
const suspendKeyB = "2251799813685250"
const suspendKeyC = "2251799813685251"

// suspensionFixture rejects hierarchy/discovery calls and records exact-key writes and reads.
// Scenarios are configured before a subprocess starts; request evidence is mutex protected.
type suspensionFixture struct {
	server        *httptest.Server
	mu            sync.Mutex
	calls         []string
	writes, reads map[string]int
	reject        map[string]int
	states        map[string][]string
	readStatus    map[string]int
}

// newSuspensionFixture serves only the generated single-instance endpoints on a test-owned server.
func newSuspensionFixture(t *testing.T) *suspensionFixture {
	t.Helper()
	f := &suspensionFixture{writes: map[string]int{}, reads: map[string]int{}, reject: map[string]int{}, states: map[string][]string{}, readStatus: map[string]int{}}
	f.server = testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
		switch {
		case r.Method == "POST" && strings.HasSuffix(key, "/suspension"):
			key = strings.TrimSuffix(key, "/suspension")
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Empty(t, body)
			f.writes[key]++
			if status := f.reject[key]; status != 0 {
				w.Header().Set("Set-Cookie", "session=suspension-cookie-secret")
				w.WriteHeader(status)
				fmt.Fprintf(w, `{"title":"Rejected","status":%d,"detail":"POST https://private.invalid/suspend?token=suspension-token-secret body-only-secret cursor=private-cursor resource=%s"}`, status, key)
				return
			}
			w.WriteHeader(204)
		case r.Method == "GET" && !strings.Contains(key, "/"):
			require.Positive(t, f.writes[key], "no preflight reads")
			f.reads[key]++
			if status := f.readStatus[key]; status != 0 {
				w.WriteHeader(status)
				fmt.Fprint(w, `{"detail":"read denied"}`)
				return
			}
			state := "SUSPENDED"
			if states := f.states[key]; len(states) > 0 {
				i := f.reads[key] - 1
				if i >= len(states) {
					i = len(states) - 1
				}
				state = states[i]
			}
			fmt.Fprintf(w, `{"processInstanceKey":%q,"state":%q,"suspendedDate":"2026-10-25T12:00:00Z"}`, key, state)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

// snapshot copies evidence so terminal callbacks can safely inspect it before approval.
func (f *suspensionFixture) snapshot() ([]string, map[string]int, map[string]int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	writes, reads := map[string]int{}, map[string]int{}
	for k, v := range f.writes {
		writes[k] = v
	}
	for k, v := range f.reads {
		reads[k] = v
	}
	return append([]string{}, f.calls...), writes, reads
}

// TestSuspendProcessInstanceSelection proves flag/stdin order, deduplication, exact parent/child
// scope, no tenant discovery and authoritative no-wait versus observed confirmation.
func TestSuspendProcessInstanceSelection(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		t.Run(fmt.Sprint(noWait), func(t *testing.T) {
			f := newSuspensionFixture(t)
			f.states[suspendKeyA] = []string{"ACTIVE", "SUSPENDED"}
			args := []string{"--tenant", "unrelated", "--json", "--auto-confirm", "suspend", "pi", "-", "--key", suspendKeyA + "," + suspendKeyB, "-k", " " + suspendKeyA + " ", "--workers", "1"}
			if noWait {
				args = append(args, "--no-wait")
			}
			out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), suspendKeyB+"\n"+suspendKeyC+"\n", args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			env := requireSingleJSONObjectDocument(t, out)
			wantOutcome, wantStatus := "succeeded", "confirmed"
			if noWait {
				wantOutcome, wantStatus = "accepted", "accepted"
			}
			require.Equal(t, wantOutcome, env["outcome"])
			require.Equal(t, "suspend process-instance", env["command"])
			require.NotContains(t, env, "tenantContext")
			items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 3)
			for i, key := range []string{suspendKeyA, suspendKeyB, suspendKeyC} {
				item := requireJSONObject(t, items[i])
				require.Equal(t, key, item["key"])
				require.Equal(t, wantStatus, item["status"])
				require.Equal(t, "accepted", item["submission"])
				require.Equal(t, float64(204), item["statusCode"])
				require.Equal(t, !noWait, item["hasObservation"])
				if noWait {
					require.NotContains(t, item, "suspendedDate")
					require.Equal(t, "not_requested", item["confirmation"])
				} else {
					require.Equal(t, "SUSPENDED", item["lastObservedState"])
					require.Equal(t, "confirmed", item["confirmation"])
				}
			}
			calls, writes, reads := f.snapshot()
			require.Equal(t, map[string]int{suspendKeyA: 1, suspendKeyB: 1, suspendKeyC: 1}, writes)
			if noWait {
				require.Empty(t, reads)
				require.Len(t, calls, 3)
			} else {
				require.Equal(t, map[string]int{suspendKeyA: 2, suspendKeyB: 1, suspendKeyC: 1}, reads)
				require.Len(t, calls, 7)
			}
		})
	}
}

// TestSuspendProcessInstanceFailFast keeps accepted work and keyed unscheduled slots; the
// command passes worker controls to the existing service rather than scheduling itself.
func TestSuspendProcessInstanceFailFast(t *testing.T) {
	f := newSuspensionFixture(t)
	f.reject[suspendKeyB] = 409
	out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", "--json", "--automation", "suspend", "pi", "--key", suspendKeyA+","+suspendKeyB+","+suspendKeyC, "--workers", "1", "--no-worker-limit", "--fail-fast")
	assertUserTaskCompletionExit(t, err, 6)
	require.Empty(t, stderr)
	items := requireJSONItems(t, requireJSONObject(t, requireSingleJSONObjectDocument(t, out)["payload"])["items"], 3)
	for i, want := range []string{"confirmed", "failed", "not_attempted"} {
		require.Equal(t, want, requireJSONObject(t, items[i])["status"])
	}
	require.Equal(t, suspendKeyC, requireJSONObject(t, items[2])["key"])
	require.Equal(t, "not_submitted", requireJSONObject(t, items[2])["submission"])
	calls, writes, reads := f.snapshot()
	require.Len(t, calls, 3)
	require.Equal(t, map[string]int{suspendKeyA: 1, suspendKeyB: 1}, writes)
	require.Equal(t, map[string]int{suspendKeyA: 1}, reads)
}

// TestSuspendProcessInstanceRejectedKeysAndExitOverride emits zero stdout bytes for wholly
// rejected execution while preserving the standard optional zero-exit behavior.
func TestSuspendProcessInstanceRejectedKeysAndExitOverride(t *testing.T) {
	for _, generic := range []bool{false, true} {
		f := newSuspensionFixture(t)
		f.reject[suspendKeyA] = 409
		f.reject[suspendKeyB] = 409
		args := []string{"--keys-only", "--quiet", "--auto-confirm", "suspend", "pi", "--key", suspendKeyA + "," + suspendKeyB}
		if generic {
			args = append(args, "--no-err-codes")
		}
		out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", args...)
		code := 6
		if generic {
			code = 0
		}
		assertUserTaskCompletionExit(t, err, code)
		require.Empty(t, out)
		require.Equal(t, [][2]string{{"ERROR", "suspend process instances: rejected: 2"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
		calls, _, reads := f.snapshot()
		require.Len(t, calls, 2)
		require.Empty(t, reads)
	}
}

// TestSuspendProcessInstanceWorkers uses paired request barriers to verify the CLI forwards
// explicit workers and no-worker-limit without overriding service scheduling policy.
func TestSuspendProcessInstanceWorkers(t *testing.T) {
	keys := []string{suspendKeyA, suspendKeyB, suspendKeyC, "2251799813685252"}
	for _, workers := range []int{1, 2, 0} {
		t.Run(fmt.Sprint(workers), func(t *testing.T) {
			want := workers
			if want == 0 {
				want = 4
			}
			var mu sync.Mutex
			active, peak, started := 0, 0, 0
			gates := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{}), make(chan struct{})}
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "POST", r.Method)
				require.Contains(t, r.URL.Path, "/suspension")
				mu.Lock()
				active++
				started++
				if active > peak {
					peak = active
				}
				index := (started - 1) / want
				if started%want == 0 {
					close(gates[index])
				}
				mu.Unlock()
				select {
				case <-gates[index]:
				case <-time.After(5 * time.Second):
					t.Error("worker bound did not permit the required group")
					w.WriteHeader(400)
					return
				}
				mu.Lock()
				active--
				mu.Unlock()
				w.WriteHeader(204)
			}))
			defer server.Close()
			args := []string{"--automation", "--json", "suspend", "pi", "--key", strings.Join(keys, ","), "--no-wait", "--no-worker-limit"}
			if workers > 0 {
				args = append(args, "--workers", fmt.Sprint(workers))
			}
			out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, server.URL, "plain-time", "info"), "", args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, "accepted", requireSingleJSONObjectDocument(t, out)["outcome"])
			mu.Lock()
			defer mu.Unlock()
			require.Equal(t, want, peak)
			require.Equal(t, 4, started)
			require.Zero(t, active)
		})
	}
}

// TestSuspendProcessInstanceCancellation cancels a real command while an accepted target's
// confirmation is in flight, preserving that evidence and the unscheduled target without resubmission.
func TestSuspendProcessInstanceCancellation(t *testing.T) {
	ready := make(chan struct{})
	var once sync.Once
	var mu sync.Mutex
	calls := []string{}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		calls = append(calls, r.Method+" "+r.URL.Path)
		mu.Unlock()
		if r.Method == "POST" {
			w.WriteHeader(204)
			return
		}
		require.Equal(t, "/v2/process-instances/"+suspendKeyA, r.URL.Path)
		once.Do(func() { close(ready) })
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			t.Error("confirmation did not observe cancellation")
		}
	}))
	defer server.Close()
	control := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-ready:
			w.WriteHeader(204)
		case <-r.Context().Done():
		}
	}))
	defer control.Close()
	for _, mode := range []string{"json"} {
		args := []string{"--config", completionLogConfig(t, server.URL, "plain-time", "info"), "--auto-confirm", "--" + mode, "suspend", "pi", "--workers", "1", "--key", suspendKeyA + "," + suspendKeyB}
		encoded, err := json.Marshal(args)
		require.NoError(t, err)
		out, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestSuspendProcessInstanceCancellationProcess", "", map[string]string{"C8VOLT_SUSPEND_ARGS": string(encoded), "C8VOLT_SUSPEND_CANCEL_URL": control.URL}, "")
		assertUserTaskCompletionExit(t, err, 1)
		require.Empty(t, stderr)
		env := requireSingleJSONObjectDocument(t, out)
		require.Equal(t, "failed", env["outcome"])
		items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 2)
		first, last := requireJSONObject(t, items[0]), requireJSONObject(t, items[1])
		require.Equal(t, "accepted", first["submission"])
		require.Equal(t, "unconfirmed", first["confirmation"])
		require.Equal(t, "interrupted", first["status"])
		require.Equal(t, float64(204), first["statusCode"])
		require.Equal(t, false, first["hasObservation"])
		require.Equal(t, "not_attempted", last["status"])
		require.Equal(t, "not_submitted", last["submission"])
		require.Equal(t, suspendKeyB, last["key"])
	}
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"POST /v2/process-instances/" + suspendKeyA + "/suspension", "GET /v2/process-instances/" + suspendKeyA}, calls)
}

// TestSuspendProcessInstanceCancellationProcess waits for a test-owned control handshake
// before canceling the command context; configuration always points at isolated servers.
func TestSuspendProcessInstanceCancellationProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_SUSPEND_ARGS")), &args))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		response, err := client.Get(os.Getenv("C8VOLT_SUSPEND_CANCEL_URL"))
		if err == nil {
			_ = response.Body.Close()
		}
		cancel()
	}()
	root := Root()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SetArgs(args)
	require.NoError(t, root.ExecuteContext(ctx))
	os.Exit(0)
}

// TestSuspendProcessInstanceConfiguredBound proves configured confirmation timeout and its
// flag override are effective even when the next poll would otherwise occur much later.
func TestSuspendProcessInstanceConfiguredBound(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(fmt.Sprint(override), func(t *testing.T) {
			f := newSuspensionFixture(t)
			f.states[suspendKeyA] = []string{"ACTIVE"}
			path := completionLogConfig(t, f.server.URL, "plain-time", "info")
			data, err := os.ReadFile(path)
			require.NoError(t, err)
			// Insert under the existing app mapping to avoid duplicate YAML keys.
			timeout := "40ms"
			if override {
				timeout = "10s"
			}
			data = []byte(strings.Replace(string(data), "app:\n", "app:\n  backoff:\n    timeout: "+timeout+"\n    initial_delay: 2s\n", 1))
			require.NoError(t, os.WriteFile(path, data, 0600))
			args := []string{"--json", "--auto-confirm", "suspend", "pi", "--key", suspendKeyA}
			if override {
				args = append(args, "--backoff-timeout", "40ms")
			}
			out, stderr, err := runUpdateUserTaskCommand(t, path, "", args...)
			assertUserTaskCompletionExit(t, err, 4)
			require.Empty(t, stderr)
			env := requireSingleJSONObjectDocument(t, out)
			require.Equal(t, "timeout", env["class"])
			item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 1)[0])
			require.Equal(t, "timed_out", item["status"])
			require.Equal(t, "accepted", item["submission"])
			require.Equal(t, "ACTIVE", item["lastObservedState"])
			calls, _, _ := f.snapshot()
			require.Len(t, calls, 2)
		})
	}
}
