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

const resumeKeyA = "2251799813685249"
const resumeKeyB = "2251799813685250"
const resumeKeyC = "2251799813685251"

// resumptionFixture rejects hierarchy/discovery calls and records exact-key writes and reads.
// Scenarios are configured before a subprocess starts; request evidence is mutex protected.
type resumptionFixture struct {
	server        *httptest.Server
	mu            sync.Mutex
	calls         []string
	writes, reads map[string]int
	reject        map[string]int
	states        map[string][]string
	readStatus    map[string]int
}

// newResumptionFixture serves only the generated single-instance endpoints on a test-owned server.
func newResumptionFixture(t *testing.T) *resumptionFixture {
	t.Helper()
	f := &resumptionFixture{writes: map[string]int{}, reads: map[string]int{}, reject: map[string]int{}, states: map[string][]string{}, readStatus: map[string]int{}}
	f.server = testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.calls = append(f.calls, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
		switch {
		case r.Method == "POST" && strings.HasSuffix(key, "/resumption"):
			key = strings.TrimSuffix(key, "/resumption")
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Empty(t, body)
			f.writes[key]++
			if status := f.reject[key]; status != 0 {
				w.Header().Set("Set-Cookie", "session=resumption-cookie-secret")
				w.WriteHeader(status)
				fmt.Fprintf(w, `{"title":"Rejected","status":%d,"detail":"POST https://private.invalid/resume?token=resumption-token-secret body-only-secret cursor=private-cursor resource=%s"}`, status, key)
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
			state := "ACTIVE"
			if states := f.states[key]; len(states) > 0 {
				i := f.reads[key] - 1
				if i >= len(states) {
					i = len(states) - 1
				}
				state = states[i]
			}
			date := ""
			if state == "SUSPENDED" {
				date = `,"suspendedDate":"2026-10-25T12:00:00Z"`
			}
			fmt.Fprintf(w, `{"processInstanceKey":%q,"state":%q%s}`, key, state, date)
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}

// snapshot copies evidence so terminal callbacks can safely inspect it before approval.
func (f *resumptionFixture) snapshot() ([]string, map[string]int, map[string]int) {
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

// TestResumeProcessInstanceSelection proves flag/stdin order, deduplication, exact parent/child
// scope, no tenant discovery and authoritative no-wait versus observed confirmation.
func TestResumeProcessInstanceSelection(t *testing.T) {
	for _, noWait := range []bool{false, true} {
		t.Run(fmt.Sprint(noWait), func(t *testing.T) {
			f := newResumptionFixture(t)
			f.states[resumeKeyA] = []string{"SUSPENDED", "ACTIVE"}
			args := []string{"--tenant", "unrelated", "--json", "--auto-confirm", "resume", "pi", "-", "--key", resumeKeyA + "," + resumeKeyB, "-k", " " + resumeKeyA + " ", "--workers", "1"}
			if noWait {
				args = append(args, "--no-wait")
			}
			out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), resumeKeyB+"\n"+resumeKeyC+"\n", args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			env := requireSingleJSONObjectDocument(t, out)
			wantOutcome, wantStatus := "succeeded", "confirmed"
			if noWait {
				wantOutcome, wantStatus = "accepted", "accepted"
			}
			require.Equal(t, wantOutcome, env["outcome"])
			require.Equal(t, "resume process-instance", env["command"])
			require.NotContains(t, env, "tenantContext")
			items := requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 3)
			for i, key := range []string{resumeKeyA, resumeKeyB, resumeKeyC} {
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
					require.Equal(t, "ACTIVE", item["lastObservedState"])
					require.NotContains(t, item, "suspendedDate")
					require.Equal(t, "confirmed", item["confirmation"])
				}
			}
			calls, writes, reads := f.snapshot()
			require.Equal(t, map[string]int{resumeKeyA: 1, resumeKeyB: 1, resumeKeyC: 1}, writes)
			if noWait {
				require.Empty(t, reads)
				require.Len(t, calls, 3)
			} else {
				require.Equal(t, map[string]int{resumeKeyA: 2, resumeKeyB: 1, resumeKeyC: 1}, reads)
				require.Len(t, calls, 7)
			}
		})
	}
}

// TestResumeProcessInstanceFailFast keeps accepted work and keyed unscheduled slots; the
// command passes worker controls to the existing service rather than scheduling itself.
func TestResumeProcessInstanceFailFast(t *testing.T) {
	f := newResumptionFixture(t)
	f.reject[resumeKeyB] = 409
	out, stderr, err := runUpdateUserTaskCommand(t, completionLogConfig(t, f.server.URL, "plain-time", "info"), "", "--json", "--automation", "resume", "pi", "--key", resumeKeyA+","+resumeKeyB+","+resumeKeyC, "--workers", "1", "--no-worker-limit", "--fail-fast")
	assertUserTaskCompletionExit(t, err, 6)
	require.Empty(t, stderr)
	items := requireJSONItems(t, requireJSONObject(t, requireSingleJSONObjectDocument(t, out)["payload"])["items"], 3)
	for i, want := range []string{"confirmed", "failed", "not_attempted"} {
		require.Equal(t, want, requireJSONObject(t, items[i])["status"])
	}
	require.Equal(t, resumeKeyC, requireJSONObject(t, items[2])["key"])
	require.Equal(t, "not_submitted", requireJSONObject(t, items[2])["submission"])
	calls, writes, reads := f.snapshot()
	require.Len(t, calls, 3)
	require.Equal(t, map[string]int{resumeKeyA: 1, resumeKeyB: 1}, writes)
	require.Equal(t, map[string]int{resumeKeyA: 1}, reads)
}

// TestResumeProcessInstanceRejectedKeysAndExitOverride emits zero stdout bytes for wholly
// rejected execution while preserving the standard optional zero-exit behavior.
func TestResumeProcessInstanceRejectedKeysAndExitOverride(t *testing.T) {
	for _, generic := range []bool{false, true} {
		f := newResumptionFixture(t)
		f.reject[resumeKeyA] = 409
		f.reject[resumeKeyB] = 409
		args := []string{"--keys-only", "--quiet", "--auto-confirm", "resume", "pi", "--key", resumeKeyA + "," + resumeKeyB}
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
		require.Equal(t, [][2]string{{"ERROR", "resume process instances: rejected: 2"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
		calls, _, reads := f.snapshot()
		require.Len(t, calls, 2)
		require.Empty(t, reads)
	}
}

// TestResumeProcessInstanceWorkers uses paired request barriers to verify the CLI forwards
// explicit workers and no-worker-limit without overriding service scheduling policy.
func TestResumeProcessInstanceWorkers(t *testing.T) {
	keys := []string{resumeKeyA, resumeKeyB, resumeKeyC, "2251799813685252"}
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
				require.Contains(t, r.URL.Path, "/resumption")
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
			args := []string{"--automation", "--json", "resume", "pi", "--key", strings.Join(keys, ","), "--no-wait", "--no-worker-limit"}
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

// TestResumeProcessInstanceCancellation cancels a real command while an accepted target's
// confirmation is in flight, preserving that evidence and the unscheduled target without resubmission.
func TestResumeProcessInstanceCancellation(t *testing.T) {
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
		require.Equal(t, "/v2/process-instances/"+resumeKeyA, r.URL.Path)
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
		args := []string{"--config", completionLogConfig(t, server.URL, "plain-time", "info"), "--auto-confirm", "--" + mode, "resume", "pi", "--workers", "1", "--key", resumeKeyA + "," + resumeKeyB}
		encoded, err := json.Marshal(args)
		require.NoError(t, err)
		out, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestResumeProcessInstanceCancellationProcess", "", map[string]string{"C8VOLT_RESUME_ARGS": string(encoded), "C8VOLT_RESUME_CANCEL_URL": control.URL}, "")
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
		require.Equal(t, resumeKeyB, last["key"])
	}
	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, []string{"POST /v2/process-instances/" + resumeKeyA + "/resumption", "GET /v2/process-instances/" + resumeKeyA}, calls)
}

// TestResumeProcessInstanceCancellationProcess waits for a test-owned control handshake
// before canceling the command context; configuration always points at isolated servers.
func TestResumeProcessInstanceCancellationProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_RESUME_ARGS")), &args))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		response, err := client.Get(os.Getenv("C8VOLT_RESUME_CANCEL_URL"))
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

// TestResumeProcessInstanceConfiguredBound proves configured confirmation timeout and its
// flag override are effective even when the next poll would otherwise occur much later.
func TestResumeProcessInstanceConfiguredBound(t *testing.T) {
	for _, override := range []bool{false, true} {
		t.Run(fmt.Sprint(override), func(t *testing.T) {
			f := newResumptionFixture(t)
			f.states[resumeKeyA] = []string{"SUSPENDED"}
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
			args := []string{"--json", "--auto-confirm", "resume", "pi", "--key", resumeKeyA}
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
			require.Equal(t, "SUSPENDED", item["lastObservedState"])
			calls, _, _ := f.snapshot()
			require.Len(t, calls, 2)
		})
	}
}

// TestResumeProcessInstanceFreshObservations proves ACTIVE and immediate COMPLETED
// confirmation, followed by independent get/expect reads with no stale suspension date.
func TestResumeProcessInstanceFreshObservations(t *testing.T) {
	for _, state := range []string{"ACTIVE", "COMPLETED"} {
		for _, mode := range []string{"json", "human", "keys"} {
			t.Run(state+"/"+mode, func(t *testing.T) {
				var mu sync.Mutex
				submitted, reads, writes := false, 0, 0
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					mu.Lock()
					defer mu.Unlock()
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.Method == "POST" && r.URL.Path == "/v2/process-instances/"+resumeKeyA+"/resumption":
						require.False(t, submitted, "observation must never resubmit")
						submitted = true
						writes++
						w.WriteHeader(204)
					case r.Method == "GET" && r.URL.Path == "/v2/process-instances/"+resumeKeyA:
						reads++
						if !submitted {
							fmt.Fprintf(w, `{"processInstanceKey":%q,"state":"SUSPENDED","suspendedDate":"2026-10-25T12:00:00Z"}`, resumeKeyA)
						} else {
							fmt.Fprintf(w, `{"processInstanceKey":%q,"state":%q}`, resumeKeyA, state)
						}
					default:
						t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
						w.WriteHeader(400)
					}
				}))
				defer server.Close()
				cfg := completionLogConfig(t, server.URL, "plain-time", "info")
				get := func(wantState string, wantDate bool) {
					out, stderr, err := runUpdateUserTaskCommand(t, cfg, "", "get", "pi", "--key", resumeKeyA, "--json")
					require.NoError(t, err, stderr)
					require.Empty(t, stderr)
					env := requireSingleJSONObjectDocument(t, out)
					require.Equal(t, "succeeded", env["outcome"])
					item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 1)[0])
					require.Equal(t, wantState, item["state"])
					if wantDate {
						require.Equal(t, "2026-10-25T12:00:00Z", item["suspendedDate"])
					} else {
						require.NotContains(t, item, "suspendedDate")
					}
				}
				get("SUSPENDED", true)
				args := []string{"resume", "pi", "--key", resumeKeyA, "--auto-confirm"}
				if mode == "json" {
					args = append(args, "--json")
				}
				if mode == "keys" {
					args = append(args, "--keys-only")
				}
				out, stderr, err := runUpdateUserTaskCommand(t, cfg, "", args...)
				require.NoError(t, err, stderr)
				switch mode {
				case "json":
					require.Empty(t, stderr)
					env := requireSingleJSONObjectDocument(t, out)
					require.Equal(t, "succeeded", env["outcome"])
					item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 1)[0])
					require.Equal(t, "accepted", item["submission"])
					require.Equal(t, "confirmed", item["confirmation"])
					require.Equal(t, "confirmed", item["status"])
					require.Equal(t, state, item["lastObservedState"])
					require.Equal(t, true, item["hasObservation"])
					require.NotContains(t, item, "suspendedDate")
				case "keys":
					require.Empty(t, stderr)
					require.Equal(t, resumeKeyA+"\n", out)
				default:
					require.Empty(t, out)
					require.Equal(t, [][2]string{{"INFO", "resumption: resumed 1/1 process instance(s)"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
				}
				get(state, false)
				out, stderr, err = runUpdateUserTaskCommand(t, cfg, "", "expect", "pi", "--key", resumeKeyA, "--state", strings.ToLower(state), "--json")
				require.NoError(t, err, stderr)
				require.Empty(t, stderr)
				env := requireSingleJSONObjectDocument(t, out)
				require.Equal(t, "succeeded", env["outcome"])
				item := requireJSONObject(t, requireJSONItems(t, requireJSONObject(t, env["payload"])["items"], 1)[0])
				require.Equal(t, state, item["state"])
				mu.Lock()
				defer mu.Unlock()
				require.Equal(t, 1, writes)
				require.Equal(t, 4, reads)
			})
		}
	}
}
