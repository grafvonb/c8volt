// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestUpdateUserTaskCompletionCommandWorkers uses request barriers to prove
// bounded parallelism and explicit-worker precedence without timing sleeps.
func TestUpdateUserTaskCompletionCommandWorkers(t *testing.T) {
	keys := []string{completionKeyA, completionKeyB, "2251799813685251", "2251799813685252"}
	for _, workers := range []int{1, 2} {
		t.Run(fmt.Sprint(workers), func(t *testing.T) {
			var mu sync.Mutex
			active, peak, started := 0, 0, 0
			gates := []chan struct{}{make(chan struct{}), make(chan struct{})}
			server, calls := newUserTaskCompletionServer(t, "8.10", func(w http.ResponseWriter, r *http.Request, key string) {
				mu.Lock()
				active++
				started++
				if active > peak {
					peak = active
				}
				index := started - 1
				if workers == 2 && started%2 == 0 {
					close(gates[index/2])
				}
				mu.Unlock()
				if workers == 2 {
					select {
					case <-gates[index/2]:
					case <-time.After(5 * time.Second):
						t.Error("worker bound did not allow paired requests")
						w.WriteHeader(500)
						return
					}
				}
				mu.Lock()
				active--
				mu.Unlock()
				w.WriteHeader(204)
			})
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", "--automation", "--json", "update", "ut", "--complete", "--key", strings.Join(keys, ","), "--workers", fmt.Sprint(workers), "--no-worker-limit")
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Len(t, calls.Snapshot(), 4)
			mu.Lock()
			require.Equal(t, workers, peak)
			require.Zero(t, active)
			mu.Unlock()
			rows := requireJSONItems(t, requireJSONObject(t, decodeUserTaskCompletionEnvelope(t, stdout)["payload"])["items"], 4)
			for i, key := range keys {
				require.Equal(t, key, requireJSONObject(t, rows[i])["key"])
			}
			if workers == 1 {
				for i, call := range calls.Snapshot() {
					require.Equal(t, keys[i], call.Key)
				}
			}
		})
	}
}

// TestUpdateUserTaskCompletionCommandFailFast proves in-flight acceptance survives
// failure, and separately proves serial fail-fast leaves queued slots unattempted.
func TestUpdateUserTaskCompletionCommandFailFast(t *testing.T) {
	for _, parallel := range []bool{false, true} {
		t.Run(fmt.Sprint(parallel), func(t *testing.T) {
			bothStarted := make(chan struct{})
			var started atomic.Int32
			server, calls := newUserTaskCompletionServer(t, "8.10", func(w http.ResponseWriter, r *http.Request, key string) {
				if parallel {
					if started.Add(1) == 2 {
						close(bothStarted)
					}
					select {
					case <-bothStarted:
					case <-time.After(5 * time.Second):
						t.Error("in-flight request missing")
						return
					}
				}
				if key == completionKeyA {
					w.Header().Set("Content-Type", "application/problem+json")
					w.WriteHeader(409)
					fmt.Fprint(w, `{"detail":"rejected"}`)
					return
				}
				w.WriteHeader(204)
			})
			workers := "1"
			keys := completionKeyA + "," + completionKeyB + ",2251799813685251"
			count := 3
			if parallel {
				workers = "2"
				keys = completionKeyA + "," + completionKeyB
				count = 2
			}
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", "--auto-confirm", "--json", "update", "ut", "--complete", "--key", keys, "--workers", workers, "--fail-fast")
			assertUserTaskCompletionExit(t, err, 6)
			require.Empty(t, stderr)
			rows := requireJSONItems(t, requireJSONObject(t, decodeUserTaskCompletionEnvelope(t, stdout)["payload"])["items"], count)
			require.Equal(t, "failed", requireJSONObject(t, rows[0])["status"])
			if parallel {
				require.Len(t, calls.Snapshot(), 2)
				require.Equal(t, "accepted", requireJSONObject(t, rows[1])["status"])
				require.Equal(t, "accepted", requireJSONObject(t, rows[1])["submission"])
			} else {
				require.Len(t, calls.Snapshot(), 1)
				for _, row := range rows[1:] {
					require.Equal(t, "not_attempted", requireJSONObject(t, row)["status"])
					require.Equal(t, "not_submitted", requireJSONObject(t, row)["submission"])
				}
			}
		})
	}
}

// TestUpdateUserTaskCompletionCommandCancellation handshakes on a known in-flight
// request before canceling the real command context; accepted and queued evidence
// must survive. A second server carries test control, never production traffic.
func TestUpdateUserTaskCompletionCommandCancellation(t *testing.T) {
	ready := make(chan struct{})
	var once sync.Once
	server, calls := newUserTaskCompletionServer(t, "8.10", func(w http.ResponseWriter, r *http.Request, key string) {
		if key == completionKeyA {
			w.WriteHeader(204)
			return
		}
		once.Do(func() { close(ready) })
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
			t.Error("command context was not canceled")
		}
	})
	control := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}

		select {
		case <-ready:
			w.WriteHeader(204)
		case <-r.Context().Done():
		}
	}))
	defer control.Close()
	args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.10"), "--auto-confirm", "--json", "update", "ut", "--complete", "--workers", "1", "--key", completionKeyA + "," + completionKeyB + ",2251799813685251"}
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestUpdateUserTaskCompletionCancellationProcess", "", map[string]string{"C8VOLT_COMPLETION_ARGS": string(encoded), "C8VOLT_COMPLETION_CANCEL_URL": control.URL}, "")
	assertUserTaskCompletionExit(t, err, 1)
	require.Empty(t, stderr)
	require.Len(t, calls.Snapshot(), 2)
	envelope := decodeUserTaskCompletionEnvelope(t, stdout)
	require.Equal(t, "local_precondition", envelope["class"])
	rows := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 3)
	for i, status := range []string{"accepted", "canceled", "not_attempted"} {
		require.Equal(t, status, requireJSONObject(t, rows[i])["status"])
	}
	require.Equal(t, "unknown", requireJSONObject(t, rows[1])["submission"])
	require.Equal(t, "not_submitted", requireJSONObject(t, rows[2])["submission"])
}

// TestUpdateUserTaskCompletionCancellationProcess supplies cancellation only after
// the isolated control handshake proves the native operation has started.
func TestUpdateUserTaskCompletionCancellationProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_COMPLETION_ARGS")), &args))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		client := &http.Client{Timeout: 10 * time.Second}
		response, err := client.Get(os.Getenv("C8VOLT_COMPLETION_CANCEL_URL"))
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
