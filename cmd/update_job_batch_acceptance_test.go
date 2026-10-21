// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// completionAcceptancePage scripts backend evidence independently of the coordinator.
type completionAcceptancePage struct {
	rows   []string
	total  int
	cursor string
	status int
}

// completionAcceptanceServer records actual adapter traffic. The mutex protects only
// bookkeeping, so concurrent HTTP requests really overlap rather than serialize.
type completionAcceptanceServer struct {
	mu                      sync.Mutex
	pages                   []completionAcceptancePage
	behavior                map[string]string
	calls                   []string
	offsets                 []int
	cursors                 []string
	reads, writes           map[string]int
	discovery, active, peak int
	expectedPages           int
	gate                    chan struct{}
	gateSize                int
	cancelReady             chan struct{}
	cancelOnce              sync.Once
	interruptPreparation    bool
	nextListener            string
	processInstanceKeys     map[string]string
	extraFilters            map[string]string
}

// newCompletionAcceptanceServer rejects every endpoint outside discovery, lookup
// and completion, making hidden repair/history/process mutations test failures.
func newCompletionAcceptanceServer(t *testing.T, pages []completionAcceptancePage, behavior map[string]string) (*completionAcceptanceServer, *httptest.Server) {
	t.Helper()
	f := &completionAcceptanceServer{pages: pages, behavior: behavior, expectedPages: len(pages), reads: map[string]int{}, writes: map[string]int{}, cancelReady: make(chan struct{})}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, w, r) }))
	t.Cleanup(server.Close)
	return f, server
}

// serve keeps scripted search responses stable even when completing a job would
// create a new listener; any rediscovery or replacement key is therefore visible.
func (f *completionAcceptanceServer) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/test/cancel" {
		select {
		case <-f.cancelReady:
			w.WriteHeader(204)
		case <-r.Context().Done():
		}
		return
	}
	if r.Method == "POST" && r.URL.Path == "/v2/jobs/search" {
		var body struct {
			Filter map[string]any `json:"filter"`
			Page   struct {
				From  int    `json:"from"`
				After string `json:"after"`
			} `json:"page"`
			Sort []map[string]any `json:"sort"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode search: %v", err)
			w.WriteHeader(400)
			return
		}
		if key, ok := body.Filter["jobKey"].(string); ok {
			// Generated nullable fields may encode null, which adds no predicate.
			for name, value := range body.Filter {
				if name != "jobKey" && value != nil {
					t.Errorf("key-only lookup contains %s: %v", name, value)
				}
			}
			f.mu.Lock()
			f.calls = append(f.calls, "read:"+key)
			f.reads[key]++
			count, submitted, behavior := f.reads[key], f.writes[key], f.behavior[key]
			f.mu.Unlock()
			row := bulkCompletionRow(key, "CREATED")
			switch behavior {
			case "completed":
				row = bulkCompletionRow(key, "COMPLETED")
			case "canceled":
				row = bulkCompletionRow(key, "CANCELED")
			case "error thrown":
				row = bulkCompletionRow(key, "ERROR_THROWN")
			case "unknown":
				row = bulkCompletionRow(key, "FUTURE_STATE")
			case "filter changed":
				row = strings.ReplaceAll(row, `"cleanup"`, `"other"`)
			case "tenant changed":
				row = strings.ReplaceAll(row, `"tenant-a"`, `"tenant-b"`)
			case "missing":
				row = ""
			case "unreadable":
				w.WriteHeader(400)
				io.WriteString(w, `{"title":"lookup unavailable"}`)
				return
			}
			if submitted > 0 {
				if strings.HasSuffix(behavior, "completed") {
					row = bulkCompletionRow(key, "COMPLETED")
				}
				if strings.HasSuffix(behavior, "missing") {
					row = ""
				}
				if behavior == "transport delayed" {
					row = ""
					if count >= 3 {
						row = bulkCompletionRow(key, "COMPLETED")
					}
				}
			}
			if processKey := f.processInstanceKeys[key]; processKey != "" {
				row = strings.ReplaceAll(row, `"2251799813711000"`, strconv.Quote(processKey))
			}
			total := 1
			if row == "" {
				total = 0
			}
			fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":%d}}`, row, total)
			return
		}
		if body.Filter["tenantId"] != "tenant-a" || body.Filter["type"] != "cleanup" {
			t.Errorf("unexpected discovery filter: %v", body.Filter)
		}
		for name, want := range f.extraFilters {
			if body.Filter[name] != want {
				t.Errorf("unexpected %s filter: got %v, want %s", name, body.Filter[name], want)
			}
		}
		if len(body.Sort) != 1 || body.Sort[0]["field"] != "jobKey" || body.Sort[0]["order"] != "ASC" {
			t.Errorf("unstable discovery sort: %v", body.Sort)
		}
		f.mu.Lock()
		index := f.discovery
		f.discovery++
		f.calls = append(f.calls, "search:"+strconv.Itoa(index))
		f.offsets = append(f.offsets, body.Page.From)
		f.cursors = append(f.cursors, body.Page.After)
		mutated := len(f.writes) != 0
		nextListener := f.nextListener
		f.mu.Unlock()
		if mutated {
			t.Error("discovery continued after a mutation (new sequential listener could enter scope)")
			if nextListener != "" {
				fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, bulkCompletionRow(nextListener, "CREATED"))
				return
			}
		}
		if f.interruptPreparation && index == 1 {
			f.cancelOnce.Do(func() { close(f.cancelReady) })
			<-r.Context().Done()
			return
		}
		if index >= len(f.pages) {
			t.Errorf("unexpected discovery page %d", index)
			w.WriteHeader(400)
			return
		}
		page := f.pages[index]
		if page.status != 0 {
			w.WriteHeader(page.status)
			io.WriteString(w, `{"title":"discovery unavailable"}`)
			return
		}
		fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":%d,"endCursor":%q}}`, strings.Join(page.rows, ","), page.total, page.cursor)
		return
	}
	if r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/v2/jobs/") && strings.HasSuffix(r.URL.Path, "/completion") {
		key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/jobs/"), "/completion")
		f.mu.Lock()
		f.calls = append(f.calls, "submit:"+key)
		f.writes[key]++
		f.active++
		f.peak = max(f.peak, f.active)
		if f.discovery != f.expectedPages {
			t.Errorf("mutation before complete preparation: got %d pages, want %d", f.discovery, f.expectedPages)
		}
		if f.writes[key] != 1 {
			t.Errorf("replayed mutation for %s", key)
		}
		if f.gate != nil && f.active == f.gateSize {
			select {
			case <-f.gate:
			default:
				close(f.gate)
			}
		}
		behavior := f.behavior[key]
		if behavior == "spawn listener" {
			f.nextListener = completionAcceptanceKey(9999)
		}
		f.mu.Unlock()
		defer func() { f.mu.Lock(); f.active--; f.mu.Unlock() }()
		if f.gate != nil {
			select {
			case <-f.gate:
			case <-time.After(5 * time.Second):
				t.Error("workers did not overlap at HTTP boundary")
			}
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Errorf("decode completion: %v", err)
		}
		if variables, ok := payload["variables"].(map[string]any); !ok || len(variables) != 0 {
			t.Errorf("listener payload: %v", payload)
		}
		if len(payload) != 1 {
			t.Errorf("unexpected optional completion fields: %v", payload)
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"detail":"unset optional completion fields must be omitted"}`)
			return
		}
		if behavior == "interrupt" {
			f.cancelOnce.Do(func() { close(f.cancelReady) })
			<-r.Context().Done()
			return
		}
		if strings.HasPrefix(behavior, "transport") {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("hijack: %v", err)
				return
			}
			_ = conn.Close() // Lose the response after receiving the mutation, as a real uncertain dispatch.
			return
		}
		status := 204
		if strings.HasPrefix(behavior, "404") {
			status = 404
		}
		if strings.HasPrefix(behavior, "409") {
			status = 409
		}
		if behavior == "rejected" {
			status = 429
		}
		w.WriteHeader(status)
		if status != 204 {
			fmt.Fprintf(w, `{"title":"completion rejected","status":%d}`, status)
		}
		return
	}
	t.Errorf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
	w.WriteHeader(400)
}

// completionAcceptanceKey stays above floating-point exact integer range to detect
// accidental numeric conversion while retaining valid signed-int64 string keys.
func completionAcceptanceKey(i int) string { return strconv.FormatInt(9007199254741000+int64(i), 10) }

// completionAcceptancePages partitions raw rows, including duplicates/nonmatches,
// so request offsets can be checked independently of selected candidate counts.
func completionAcceptancePages(rows []string, size int) []completionAcceptancePage {
	if len(rows) == 0 {
		return []completionAcceptancePage{{}}
	}
	var pages []completionAcceptancePage
	for start := 0; start < len(rows); start += size {
		pages = append(pages, completionAcceptancePage{rows: rows[start:min(start+size, len(rows))], total: len(rows)})
	}
	return pages
}

// runCompletionAcceptance executes the real command in an isolated process so
// nonzero exits and stdout/stderr are observed without suppression or flag leakage.
func runCompletionAcceptance(t *testing.T, server *httptest.Server, version string, flags []string, stdin string, interrupt bool) (string, string, error) {
	t.Helper()
	args := []string{"--config", writeTestConfigForVersion(t, server.URL, version), "--tenant", "tenant-a", "update", "job", "--complete", "--workers", "4", "--no-worker-limit", "--backoff-max-retries", "3", "--backoff-timeout", "2s"}
	args = append(args, flags...)
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	env := map[string]string{"C8VOLT_BATCH_ARGS": string(encoded)}
	if interrupt {
		env["C8VOLT_ACCEPTANCE_CANCEL_URL"] = server.URL + "/test/cancel"
	}
	return testx.RunCmdSubprocessInDirWithSeparateOutputs(t, "TestBulkCompletionAcceptanceHelper", "", env, stdin)
}

// TestBulkCompletionAcceptanceHelper uses a server handshake to cancel the actual
// command context during a known request, avoiding timing-based cancellation races.
func TestBulkCompletionAcceptanceHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_BATCH_ARGS")), &args))
	root := Root()
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	root.SetArgs(args)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if url := os.Getenv("C8VOLT_ACCEPTANCE_CANCEL_URL"); url != "" {
		go func() {
			client := &http.Client{Timeout: 10 * time.Second}
			response, err := client.Get(url)
			if err == nil {
				_ = response.Body.Close()
			}
			cancel()
		}()
	}
	if _, err := root.ExecuteContextC(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

// requireCompletionAcceptanceFailure proves a true process failure while retaining
// exactly one envelope with the command's full partial payload.
func requireCompletionAcceptanceFailure(t *testing.T, stdout string, err error) map[string]any {
	t.Helper()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.NotZero(t, exitErr.ExitCode())
	envelope := decodeBulkCompletionEnvelope(t, stdout)
	require.Contains(t, []string{"failed", "invalid"}, envelope["outcome"])
	require.NotEmpty(t, envelope["class"])
	return requireJSONObject(t, envelope["payload"])
}

// requireCompletionAcceptancePartition verifies count conservation and frozen
// order through the JSON wire contract, including unstarted and failed targets.
func requireCompletionAcceptancePartition(t *testing.T, payload map[string]any, keys []string) []any {
	t.Helper()
	require.NotContains(t, payload, "candidates")
	outcomes, ok := payload["outcomes"].([]any)
	require.True(t, ok)
	require.Len(t, outcomes, len(keys))
	counts := requireJSONObject(t, payload["counts"])
	require.Len(t, counts, 5)
	actual := map[string]int{"accepted": 0, "completed": 0, "unsatisfied": 0, "notStarted": 0}
	for i, raw := range outcomes {
		out := requireJSONObject(t, raw)
		require.Equal(t, keys[i], requireJSONObject(t, out["candidate"])["key"])
		disposition := out["disposition"].(string)
		if disposition == "not_started" {
			disposition = "notStarted"
		}
		_, valid := actual[disposition]
		require.True(t, valid, disposition)
		actual[disposition]++
	}
	require.Equal(t, float64(len(keys)), counts["selected"])
	for name, count := range actual {
		require.Equal(t, float64(count), counts[name], name)
	}
	return outcomes
}

// TestBulkCompletionAcceptanceScale proves 1,000 unique mixed outcomes, raw paging,
// scope exclusion, one-shot dispatch and real bounded HTTP concurrency on each adapter.
func TestBulkCompletionAcceptanceScale(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		t.Run(version, func(t *testing.T) {
			var rows, keys []string
			behaviors := map[string]string{}
			// An entire sparse page and a nonmatching type must not consume the target count.
			for i := 0; i < 100; i++ {
				rows = append(rows, strings.ReplaceAll(bulkCompletionRow(completionAcceptanceKey(2000+i), "CREATED"), `"tenant-a"`, `"tenant-b"`))
			}
			rows = append(rows, strings.ReplaceAll(bulkCompletionRow(completionAcceptanceKey(3000), "CREATED"), `"cleanup"`, `"other"`))
			kinds := []string{"completed", "canceled", "filter changed", "missing", "unreadable", "409 completed", "transport missing", "accepted", "404 completed", "accepted"}
			for i := 0; i < 1000; i++ {
				key := completionAcceptanceKey(i)
				keys = append(keys, key)
				rows = append(rows, bulkCompletionRow(key, "CREATED"))
				behaviors[key] = kinds[i%len(kinds)]
			}
			rows = append(rows, rows[101]) // Duplicate on a later page cannot replay completion.
			previewFixture, previewServer := newCompletionAcceptanceServer(t, completionAcceptancePages(rows, 100), behaviors)
			preview, previewStderr, previewErr := runCompletionAcceptance(t, previewServer, version, []string{"--type", "cleanup", "--batch-size", "100", "--dry-run", "--json"}, "", false)
			require.NoError(t, previewErr, previewStderr)
			previewEnvelope := decodeBulkCompletionEnvelope(t, preview)
			require.Equal(t, "succeeded", previewEnvelope["outcome"])
			previewPayload := requireJSONObject(t, previewEnvelope["payload"])
			require.Equal(t, "preview", previewPayload["status"])
			require.NotContains(t, previewPayload, "outcomes")
			candidates := previewPayload["candidates"].([]any)
			require.Len(t, candidates, 1000)
			for i, candidate := range candidates {
				item := requireJSONObject(t, candidate)
				require.Equal(t, keys[i], item["key"])
				require.Equal(t, "tenant-a", item["tenantId"])
				require.Equal(t, "2251799813711000", item["processInstanceKey"])
			}
			require.Equal(t, 12, previewFixture.discovery)
			require.Empty(t, previewFixture.reads)
			require.Empty(t, previewFixture.writes)
			require.Empty(t, previewStderr)
			f, server := newCompletionAcceptanceServer(t, completionAcceptancePages(rows, 100), behaviors)
			f.gate = make(chan struct{})
			f.gateSize = 4
			stdout, stderr, err := runCompletionAcceptance(t, server, version, []string{"--type", "cleanup", "--batch-size", "100", "--auto-confirm", "--json", "--no-wait"}, "", false)
			payload := requireCompletionAcceptanceFailure(t, stdout, err)
			outcomes := requireCompletionAcceptancePartition(t, payload, keys)
			require.Equal(t, "finished", payload["status"])
			require.Equal(t, map[string]any{"selected": float64(1000), "accepted": float64(200), "completed": float64(300), "unsatisfied": float64(500), "notStarted": float64(0)}, payload["counts"])
			require.Empty(t, stderr)
			f.mu.Lock()
			defer f.mu.Unlock()
			require.Equal(t, 12, f.discovery)
			require.Equal(t, 4, f.peak)
			require.Len(t, f.writes, 500)
			require.Len(t, f.reads, 1000)
			totalReads := 0
			for _, count := range f.reads {
				totalReads += count
			}
			require.Equal(t, 1300, totalReads)
			for i, offset := range f.offsets {
				require.Equal(t, i*100, offset)
			}
			for i, key := range keys {
				out := requireJSONObject(t, outcomes[i])
				submission := requireJSONObject(t, out["submission"])
				observation := requireJSONObject(t, out["observation"])
				wantWrites := 0
				wantReads := 1
				if i%10 >= 5 {
					wantWrites = 1
				}
				if i%10 == 5 || i%10 == 6 || i%10 == 8 {
					wantReads = 2
				}
				require.Equal(t, wantWrites, f.writes[key], key)
				require.Equal(t, wantReads, f.reads[key], key)
				if i%10 == 6 {
					require.Equal(t, "uncertain", submission["status"])
					require.Equal(t, "missing", observation["status"])
				}
				if i%10 == 7 || i%10 == 9 {
					require.Equal(t, "accepted", submission["status"])
					require.Equal(t, "skipped", observation["status"])
				}
			}
			require.Equal(t, "search:11", f.calls[11])
			require.True(t, strings.HasPrefix(f.calls[12], "read:"))
			t.Logf("selected=1000 raw=%d pages=%d lookups=%d submissions=%d peak=%d", len(rows), f.discovery, totalReads, len(f.writes), f.peak)
		})
	}
}

// TestBulkCompletionAcceptancePaging proves sparse cursor/offset traversal, unique
// limits and incomplete evidence at the command boundary, with zero writes on failure.
func TestBulkCompletionAcceptancePaging(t *testing.T) {
	a, b, c := completionAcceptanceKey(0), completionAcceptanceKey(1), completionAcceptanceKey(2)
	rowA, rowB, rowC := bulkCompletionRow(a, "CREATED"), bulkCompletionRow(b, "CREATED"), bulkCompletionRow(c, "CREATED")
	foreign := strings.ReplaceAll(bulkCompletionRow(completionAcceptanceKey(99), "CREATED"), `"tenant-a"`, `"tenant-b"`)
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, tc := range []struct {
			name    string
			pages   []completionAcceptancePage
			limit   int
			keys    []string
			failed  bool
			offsets []int
			cursors []string
		}{
			{name: "sparse cursor", pages: []completionAcceptancePage{{total: 1, cursor: "next"}, {rows: []string{rowA}, total: 1}}, keys: []string{a}, offsets: []int{0, 0}, cursors: []string{"", "next"}},
			{name: "sparse duplicate unique limit", pages: []completionAcceptancePage{{rows: []string{foreign, foreign}, total: 8}, {rows: []string{rowA, rowA}, total: 8}, {rows: []string{rowB, rowC}, total: 8}}, limit: 2, keys: []string{a, b}, offsets: []int{0, 2, 4}, cursors: []string{"", "", ""}},
			{name: "contradictory empty", pages: []completionAcceptancePage{{rows: []string{rowA, rowB}, total: 3}, {total: 3}}, failed: true, offsets: []int{0, 2}},
			{name: "cyclic cursor", pages: []completionAcceptancePage{{total: 1, cursor: "same"}, {total: 1, cursor: "same"}}, failed: true, offsets: []int{0, 0}},
			{name: "discovery error", pages: []completionAcceptancePage{{rows: []string{rowA, rowB}, total: 3}, {status: 400}}, failed: true, offsets: []int{0, 2}},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				f, server := newCompletionAcceptanceServer(t, tc.pages, nil)
				flags := []string{"--type", "cleanup", "--batch-size", "2", "--auto-confirm", "--json"}
				if tc.limit > 0 {
					flags = append(flags, "--limit", strconv.Itoa(tc.limit))
				}
				stdout, stderr, err := runCompletionAcceptance(t, server, version, flags, "", false)
				require.Empty(t, stderr)
				if tc.failed {
					payload := requireCompletionAcceptanceFailure(t, stdout, err)
					require.Equal(t, "preparation_failed", payload["status"])
					require.Contains(t, payload, "candidates")
					require.NotContains(t, payload, "outcomes")
					require.Empty(t, f.writes)
					require.Empty(t, f.reads)
				} else {
					require.NoError(t, err, stdout)
					payload := requireJSONObject(t, decodeBulkCompletionEnvelope(t, stdout)["payload"])
					requireCompletionAcceptancePartition(t, payload, tc.keys)
					require.Len(t, f.writes, len(tc.keys))
					if tc.limit > 0 {
						require.Equal(t, "limit_reached", payload["completionReason"])
						require.Equal(t, true, payload["limited"])
					}
				}
				require.Equal(t, tc.offsets, f.offsets)
				if tc.cursors != nil {
					require.Equal(t, tc.cursors, f.cursors)
				}
			})
		}
	}
}

// TestBulkCompletionAcceptanceReconciliation verifies independent submission and
// observation evidence, bounded polling and no replay after an actual lost response.
func TestBulkCompletionAcceptanceReconciliation(t *testing.T) {
	key := completionAcceptanceKey(0)
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, tc := range []struct {
			behavior, submission, observation, disposition string
			reads                                          int
			noWait                                         bool
		}{
			{"404 completed", "rejected", "completed", "completed", 2, true},
			{"404 missing", "rejected", "missing", "unsatisfied", 2, true},
			{"409 completed", "rejected", "completed", "completed", 2, true},
			{"409 missing", "rejected", "missing", "unsatisfied", 2, true},
			{"transport completed", "uncertain", "completed", "completed", 2, true},
			{"transport missing", "uncertain", "missing", "unsatisfied", 2, true},
			{"transport delayed", "uncertain", "completed", "completed", 3, false},
			{"transport delayed", "uncertain", "missing", "unsatisfied", 2, true},
			{"transport missing", "uncertain", "missing", "unsatisfied", 4, false},
			{"rejected", "rejected", "not_completed", "unsatisfied", 1, false},
		} {
			t.Run(fmt.Sprintf("%s/%s/no-wait=%t", version, tc.behavior, tc.noWait), func(t *testing.T) {
				f, server := newCompletionAcceptanceServer(t, completionAcceptancePages([]string{bulkCompletionRow(key, "CREATED")}, 2), map[string]string{key: tc.behavior})
				flags := []string{"--type", "cleanup", "--auto-confirm", "--json"}
				if tc.noWait {
					flags = append(flags, "--no-wait")
				}
				stdout, stderr, err := runCompletionAcceptance(t, server, version, flags, "", false)
				require.Empty(t, stderr)
				var payload map[string]any
				if tc.disposition == "unsatisfied" {
					payload = requireCompletionAcceptanceFailure(t, stdout, err)
				} else {
					require.NoError(t, err, stdout)
					envelope := decodeBulkCompletionEnvelope(t, stdout)
					require.Equal(t, "succeeded", envelope["outcome"])
					payload = requireJSONObject(t, envelope["payload"])
				}
				out := requireJSONObject(t, requireCompletionAcceptancePartition(t, payload, []string{key})[0])
				submission := requireJSONObject(t, out["submission"])
				observation := requireJSONObject(t, out["observation"])
				require.Equal(t, tc.submission, submission["status"])
				require.Equal(t, tc.observation, observation["status"])
				require.Equal(t, tc.disposition, out["disposition"])
				if tc.submission == "uncertain" {
					require.NotContains(t, submission, "httpStatus")
				}
				if strings.HasPrefix(tc.behavior, "404") {
					require.Equal(t, float64(404), submission["httpStatus"])
				}
				if strings.HasPrefix(tc.behavior, "409") {
					require.Equal(t, float64(409), submission["httpStatus"])
				}
				require.Equal(t, tc.reads, f.reads[key])
				require.Equal(t, map[string]int{key: 1}, f.writes)
			})
		}
	}
}

// TestBulkCompletionAcceptanceStops preserves all frozen rows after fail-fast or
// caller cancellation, including an accepted row and a dispatch whose response was lost.
func TestBulkCompletionAcceptanceStops(t *testing.T) {
	var rows, keys []string
	for i := 0; i < 6; i++ {
		key := completionAcceptanceKey(i)
		keys = append(keys, key)
		rows = append(rows, bulkCompletionRow(key, "CREATED"))
	}
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, scenario := range []string{"fail fast", "interrupt execution", "interrupt preparation"} {
			t.Run(version+"/"+scenario, func(t *testing.T) {
				behavior := map[string]string{keys[1]: "rejected"}
				interrupt := scenario != "fail fast"
				if scenario == "interrupt execution" {
					behavior[keys[1]] = "interrupt"
				}
				f, server := newCompletionAcceptanceServer(t, completionAcceptancePages(rows, 2), behavior)
				f.interruptPreparation = scenario == "interrupt preparation"
				flags := []string{"--type", "cleanup", "--batch-size", "2", "--workers", "1", "--auto-confirm", "--json", "--no-wait"}
				if scenario == "fail fast" {
					flags = append(flags, "--fail-fast")
				}
				stdout, stderr, err := runCompletionAcceptance(t, server, version, flags, "", interrupt)
				require.Empty(t, stderr)
				payload := requireCompletionAcceptanceFailure(t, stdout, err)
				if scenario == "interrupt preparation" {
					require.Equal(t, "preparation_failed", payload["status"])
					require.Len(t, payload["candidates"], 2)
					require.NotContains(t, payload, "outcomes")
					require.Empty(t, f.writes)
					require.Empty(t, f.reads)
					return
				}
				outcomes := requireCompletionAcceptancePartition(t, payload, keys)
				require.Equal(t, map[string]any{"selected": float64(6), "accepted": float64(1), "completed": float64(0), "unsatisfied": float64(1), "notStarted": float64(4)}, payload["counts"])
				require.Equal(t, map[string]int{keys[0]: 1, keys[1]: 1}, f.writes)
				reason := "fail_fast"
				if interrupt {
					reason = "interrupted"
					require.Equal(t, "interrupted", payload["status"])
					require.Equal(t, "uncertain", requireJSONObject(t, requireJSONObject(t, outcomes[1])["submission"])["status"])
				}
				for _, raw := range outcomes[2:] {
					out := requireJSONObject(t, raw)
					require.Equal(t, reason, out["reason"])
					require.Equal(t, "not_submitted", requireJSONObject(t, out["submission"])["status"])
				}
			})
		}
	}
}

// TestBulkCompletionAcceptanceGuards exercises final-state, tenant and listener
// safety at the real HTTP boundary, with no fallback to unrelated mutations.
func TestBulkCompletionAcceptanceGuards(t *testing.T) {
	key := completionAcceptanceKey(0)
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, tc := range []struct {
			behavior, reason string
			variables        bool
		}{
			{behavior: "tenant changed", reason: "scope_changed"}, {behavior: "filter changed", reason: "scope_changed"},
			{behavior: "canceled", reason: "other_terminal"}, {behavior: "error thrown", reason: "other_terminal"},
			{behavior: "unknown", reason: "unsupported_state"}, {behavior: "missing", reason: "missing"}, {behavior: "unreadable", reason: "unreadable"},
			{behavior: "variables", variables: true},
		} {
			t.Run(version+"/"+tc.behavior, func(t *testing.T) {
				f, server := newCompletionAcceptanceServer(t, completionAcceptancePages([]string{bulkCompletionRow(key, "CREATED")}, 2), map[string]string{key: tc.behavior})
				flags := []string{"--type", "cleanup", "--auto-confirm", "--json"}
				if tc.variables {
					flags = append(flags, "--vars", `{"approved":true}`)
				}
				stdout, _, err := runCompletionAcceptance(t, server, version, flags, "", false)
				payload := requireCompletionAcceptanceFailure(t, stdout, err)
				require.Empty(t, f.writes)
				if tc.variables {
					require.Equal(t, "preparation_failed", payload["status"])
					require.Empty(t, f.reads)
				} else {
					out := requireJSONObject(t, requireCompletionAcceptancePartition(t, payload, []string{key})[0])
					require.Equal(t, tc.reason, out["reason"])
				}
			})
		}
	}
}

// TestBulkCompletionAcceptanceOutputMatrix checks real execution and preview
// streams across adapters, including authoritative empty search/stdin and partial discovery.
func TestBulkCompletionAcceptanceOutputMatrix(t *testing.T) {
	key := completionAcceptanceKey(0)
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, scenario := range []string{"nonempty", "empty search", "empty stdin", "incomplete"} {
			for _, mode := range []struct {
				name                                  string
				flags                                 []string
				preview, json, quiet, verbose, noWait bool
			}{
				{name: "human", flags: []string{"--auto-confirm"}},
				{name: "verbose", flags: []string{"--auto-confirm", "--verbose"}, verbose: true},
				{name: "quiet", flags: []string{"--auto-confirm", "--quiet"}, quiet: true},
				{name: "JSON automation", flags: []string{"--automation", "--json"}, json: true},
				{name: "JSON quiet no wait", flags: []string{"--auto-confirm", "--quiet", "--json", "--no-wait"}, json: true, quiet: true, noWait: true},
				{name: "human no wait", flags: []string{"--auto-confirm", "--no-wait"}, noWait: true},
				{name: "preview", flags: []string{"--dry-run"}, preview: true},
				{name: "preview verbose", flags: []string{"--dry-run", "--verbose"}, preview: true, verbose: true},
				{name: "preview quiet", flags: []string{"--dry-run", "--quiet"}, preview: true, quiet: true},
				{name: "preview quiet JSON", flags: []string{"--dry-run", "--quiet", "--json"}, preview: true, quiet: true, json: true},
			} {
				t.Run(version+"/"+scenario+"/"+mode.name, func(t *testing.T) {
					pages := completionAcceptancePages(nil, 1)
					if scenario == "nonempty" {
						pages = completionAcceptancePages([]string{bulkCompletionRow(key, "CREATED")}, 1)
					}
					if scenario == "incomplete" {
						pages = []completionAcceptancePage{{rows: []string{bulkCompletionRow(key, "CREATED")}, total: 2}, {status: 400}}
					}
					f, server := newCompletionAcceptanceServer(t, pages, nil)
					flags := []string{"--type", "cleanup", "--batch-size", "1"}
					if scenario == "empty stdin" {
						flags = []string{"-"}
						f.expectedPages = 0
					}
					stdout, stderr, err := runCompletionAcceptance(t, server, version, append(flags, mode.flags...), "", false)
					if !mode.json {
						stdout, stderr = assertBulkCompletionHumanLogs(t, stdout, stderr)
					}
					if scenario == "incomplete" {
						var exitErr *exec.ExitError
						require.ErrorAs(t, err, &exitErr)
					} else {
						require.NoError(t, err, stdout, stderr)
					}
					if scenario == "incomplete" && !mode.json {
						require.Contains(t, stderr, "job completion:")
					} else {
						require.Empty(t, stderr)
					}
					require.NotContains(t, stdout, "Do you want to proceed?")
					require.NotContains(t, stderr, "Do you want to proceed?")
					require.NotContains(t, stdout, "next page")
					wantCount := 0
					if scenario == "nonempty" || scenario == "incomplete" {
						wantCount = 1
					}
					wantWrites := 0
					if scenario == "nonempty" && !mode.preview {
						wantWrites = 1
					}
					require.Len(t, f.writes, wantWrites)
					require.Len(t, f.reads, wantWrites)
					expectedPages := len(pages)
					if scenario == "empty stdin" {
						expectedPages = 0
					}
					require.Equal(t, expectedPages, f.discovery)
					if mode.json {
						envelope := decodeBulkCompletionEnvelope(t, stdout)
						payload := requireJSONObject(t, envelope["payload"])
						expectedOutcome := "succeeded"
						if wantWrites > 0 && mode.noWait {
							expectedOutcome = "accepted"
						}
						if scenario == "incomplete" {
							require.Contains(t, []string{"failed", "invalid"}, envelope["outcome"])
						} else {
							require.Equal(t, expectedOutcome, envelope["outcome"])
						}
						collection, omitted := "outcomes", "candidates"
						status := "finished"
						if mode.preview {
							collection, omitted, status = "candidates", "outcomes", "preview"
						}
						if scenario == "incomplete" {
							collection, omitted, status = "candidates", "outcomes", "preparation_failed"
						}
						require.Equal(t, status, payload["status"])
						require.NotContains(t, payload, omitted)
						values, ok := payload[collection].([]any)
						require.True(t, ok, "collection must be an array, never null")
						require.Len(t, values, wantCount)
						counts := requireJSONObject(t, payload["counts"])
						require.Len(t, counts, 5)
						require.Equal(t, float64(wantCount), counts["selected"])
						for _, name := range []string{"accepted", "completed", "unsatisfied", "notStarted"} {
							_, ok := counts[name].(float64)
							require.True(t, ok, name)
						}
					} else if mode.quiet {
						require.Empty(t, stdout)
					} else if strings.HasPrefix(scenario, "empty") {
						label := "job completion\n"
						if mode.preview {
							label = "dry run: job completion\n"
						}
						scope := "filters: type=cleanup\nconfigured tenant: none\n--tenant \"tenant-a\" sets the tenant filter\nselection scope: tenant-a only\n"
						if scenario == "empty stdin" {
							scope = "selection: explicit keys\nselection scope: explicit resource keys; tenant filter not applied\n"
						}
						want := label + scope + "selected job(s): 0\n"
						if mode.verbose && scenario != "empty stdin" {
							want += "discovery: pages 1; batch size 1\n"
						}
						if mode.preview {
							want += jobCompletionListenerConsequence + "\noutcome: preview; no changes applied\n"
						} else if scenario == "empty stdin" {
							want += "outcome: no selected jobs; no changes applied\n"
						} else {
							want += "outcome: no matching jobs; no changes applied\n"
						}
						require.Equal(t, want, stdout)
					} else if scenario == "incomplete" {
						require.Contains(t, stdout, "preparation incomplete: 1 observed candidate job(s); no changes applied")
					} else {
						require.Contains(t, stdout, "selected job(s): 1\n")
						if mode.preview {
							require.Contains(t, stdout, jobCompletionListenerConsequence)
						} else {
							require.Contains(t, stdout, "accepted 1; observed completed 0; unsatisfied 0; not started 0")
						}
						if mode.verbose {
							require.Contains(t, stdout, key)
						} else {
							require.NotContains(t, stdout, key)
						}
						require.NotContains(t, stdout, "POST")
						require.NotContains(t, stdout, "cursor")
					}
				})
			}
		}
	}
}

// TestBulkCompletionAcceptanceUnsupported preserves retired-version rejection even
// when stdin is empty; no backend request or fabricated successful no-op is allowed.
func TestBulkCompletionAcceptanceUnsupported(t *testing.T) {
	for _, flags := range [][]string{{"--type", "cleanup", "--auto-confirm", "--json"}, {"-", "--auto-confirm", "--json"}, {"-", "--dry-run", "--json"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			f, server := newCompletionAcceptanceServer(t, nil, nil)
			stdout, stderr, err := runCompletionAcceptance(t, server, "8.7", flags, "", false)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, "unknown Camunda version: 8.7")
			require.Empty(t, f.calls)
		})
	}
}

// TestBulkCompletionAcceptanceSequentialListener makes completion create a new
// backend listener and verifies the invocation neither discovers nor mutates it.
func TestBulkCompletionAcceptanceSequentialListener(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		t.Run(version, func(t *testing.T) {
			key := completionAcceptanceKey(0)
			f, server := newCompletionAcceptanceServer(t, completionAcceptancePages([]string{bulkCompletionRow(key, "CREATED")}, 1), map[string]string{key: "spawn listener"})
			stdout, stderr, err := runCompletionAcceptance(t, server, version, []string{"--type", "cleanup", "--auto-confirm", "--json"}, "", false)
			require.NoError(t, err, stdout, stderr)
			require.Empty(t, stderr)
			payload := requireJSONObject(t, decodeBulkCompletionEnvelope(t, stdout)["payload"])
			requireCompletionAcceptancePartition(t, payload, []string{key})
			require.Equal(t, completionAcceptanceKey(9999), f.nextListener)
			require.Equal(t, []string{"search:0", "read:" + key, "submit:" + key}, f.calls)
			require.Zero(t, f.reads[f.nextListener])
			require.Zero(t, f.writes[f.nextListener])
			require.NotContains(t, stdout, f.nextListener)
		})
	}
}
