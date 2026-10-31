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
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// definitionStatisticsFixture serves only test-owned definition/count reads.
// Lifecycle and count populations are independent observations; atomic counters
// also let watch tests select each refresh without contacting an engine.
type definitionStatisticsFixture struct {
	definitions atomic.Int32
	counts      atomic.Int32
	population  func(int32) (int, int)
	lifecycle   func(int32, string) string
	keys        []string
	completed   int
	canceled    int
	incidents   int
	countStatus int
	empty       bool
	version     string
}

// server rejects all unexpected requests, including mutations, and returns
// exact totals so output tests do not depend on traversal or a running engine.
func (f *definitionStatisticsFixture) server(t *testing.T) *httptest.Server {
	t.Helper()
	keys := f.keys
	if len(keys) == 0 {
		keys = []string{"123"}
	}
	srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, f.version) {
			return
		}

		w.Header().Set("Content-Type", "application/json")
		definition := func(read int32, key string) map[string]any {
			item := map[string]any{"processDefinitionKey": key, "processDefinitionId": "invoice", "version": 1, "tenantId": "tenant"}
			if f.lifecycle != nil {
				if state := f.lifecycle(read, key); state != "" {
					item["state"] = state
				}
			}
			return item
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /v2/process-definitions/123":
			require.NoError(t, json.NewEncoder(w).Encode(definition(f.definitions.Add(1), "123")))
		case "POST /v2/process-definitions/search":
			read := f.definitions.Add(1)
			if f.empty {
				_, _ = io.WriteString(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
			} else {
				items := make([]map[string]any, 0, len(keys))
				for _, key := range keys {
					items = append(items, definition(read, key))
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items), "hasMoreTotalItems": false}}))
			}
		case "POST /v2/process-instances/search":
			f.counts.Add(1)
			var body struct {
				Filter map[string]any `json:"filter"`
				Page   map[string]any `json:"page"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Contains(t, keys, body.Filter["processDefinitionKey"])
			require.Equal(t, float64(1), body.Page["limit"])
			if f.countStatus != 0 {
				w.WriteHeader(f.countStatus)
				_, _ = io.WriteString(w, `{"message":"statistics unavailable"}`)
				return
			}
			active, suspended := f.population(f.definitions.Load())
			count := 0
			switch state := body.Filter["state"].(type) {
			case string:
				switch state {
				case "ACTIVE":
					count = active
				case "SUSPENDED":
					require.Equal(t, "8.10", f.version, "older versions must not query suspension")
					count = suspended
				case "COMPLETED":
					count = f.completed
				case "TERMINATED":
					count = f.canceled
				default:
					t.Errorf("unexpected state %q", state)
				}
			case map[string]any:
				require.Equal(t, "8.10", f.version)
				require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, state["$in"])
				count = active + suspended
			default:
				require.Equal(t, true, body.Filter["hasIncident"])
				count = f.incidents
			}
			fmt.Fprintf(w, `{"items":[],"page":{"totalItems":%d,"hasMoreTotalItems":false}}`, count)
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestGetProcessDefinitionSuspendedStatisticsCommand checks real lookup/list
// dispatch, public JSON omission rules and unchanged finite output precedence.
func TestGetProcessDefinitionSuspendedStatisticsCommand(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, key := range []bool{false, true} {
			for _, suspended := range []int{0, 3} {
				for _, mode := range [][]string{nil, {"--json"}, {"--keys-only"}, {"--quiet"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}} {
					t.Run(fmt.Sprintf("%s/key=%t/suspended=%d/%s", version, key, suspended, strings.Join(mode, " ")), func(t *testing.T) {
						fixture := &definitionStatisticsFixture{version: version, population: func(int32) (int, int) { return 2, suspended }}
						srv := fixture.server(t)
						args := []string{"--config", writeTestConfigForVersion(t, srv.URL, version), "get", "process-definition", "--stat"}
						if key {
							args = append(args, "--key", "123")
						}
						args = append(args, mode...)
						stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)
						require.Empty(t, stderr)
						require.EqualValues(t, 1, fixture.definitions.Load())
						wantCounts := 4
						if version == "8.10" {
							wantCounts = 6
						}
						require.EqualValues(t, wantCounts, fixture.counts.Load())
						switch {
						case strings.Contains(strings.Join(mode, " "), "--json"):
							decoder := json.NewDecoder(strings.NewReader(stdout))
							var envelope map[string]any
							require.NoError(t, decoder.Decode(&envelope))
							require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
							require.Equal(t, "succeeded", envelope["outcome"])
							payload := requireJSONObject(t, envelope["payload"])
							if !key {
								payload = requireJSONObject(t, requireJSONItems(t, payload["items"], 1)[0])
							}
							require.NotContains(t, payload, "state", "counts cannot supply missing lifecycle")
							stats := requireJSONObject(t, payload["statistics"])
							require.Equal(t, float64(2), stats["active"])
							require.NotContains(t, stats, "unfinished")
							if version == "8.10" {
								require.Equal(t, true, stats["suspendedCountSupported"])
								if suspended == 0 {
									require.NotContains(t, stats, "suspended")
								} else {
									require.Equal(t, float64(suspended), stats["suspended"])
								}
							} else {
								require.NotContains(t, stats, "suspendedCountSupported")
								require.NotContains(t, stats, "suspended")
							}
						case strings.Contains(strings.Join(mode, " "), "--keys-only"):
							require.Equal(t, "123\n", stdout)
						default:
							want := "123 tenant invoice v1 [ac:2 cp:- cx:- inc:-"
							if version == "8.10" {
								want += " su:" + zeroAsMinus(int64(suspended))
							}
							want += "]\n"
							if !key {
								want += "found: 1\n"
							}
							require.Equal(t, want, stdout)
						}
					})
				}
			}
		}
	}
}

// TestGetProcessDefinitionSuspendedNoStat preserves the no-stat read boundary
// for both ordinary dispatch and actual watch commands against an HTTP fake.
func TestGetProcessDefinitionSuspendedNoStat(t *testing.T) {
	for _, watch := range []bool{false, true} {
		for _, key := range []bool{false, true} {
			t.Run(fmt.Sprintf("watch=%t/key=%t", watch, key), func(t *testing.T) {
				fixture := &definitionStatisticsFixture{version: "8.10", population: func(int32) (int, int) { return 2, 3 }}
				srv := fixture.server(t)
				args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "get", "process-definition"}
				if key {
					args = append(args, "--key", "123")
				}
				if watch {
					args = append(args, "--watch")
					previous := processDefinitionWatchSleep
					processDefinitionWatchSleep = func(context.Context, time.Duration) error { return context.Canceled }
					t.Cleanup(func() { processDefinitionWatchSleep = previous })
				}
				stdout, _ := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)
				require.Contains(t, stdout, "123 tenant invoice v1\n")
				require.NotContains(t, stdout, "[ac:")
				require.EqualValues(t, 1, fixture.definitions.Load())
				require.Zero(t, fixture.counts.Load())
			})
		}
	}
}

// TestGetProcessDefinitionLifecycleStatisticsCommand covers draining with suspended
// work through both read paths and output formats. Adapter tests cover other count
// populations; mixed-row tests cover unfamiliar and omitted lifecycle values.
func TestGetProcessDefinitionLifecycleStatisticsCommand(t *testing.T) {
	for _, key := range []bool{false, true} {
		for _, jsonOutput := range []bool{false, true} {
			t.Run(fmt.Sprintf("key=%t/json=%t", key, jsonOutput), func(t *testing.T) {
				fixture := &definitionStatisticsFixture{
					version: "8.10", completed: 7, canceled: 9, incidents: 1,
					lifecycle:  func(int32, string) string { return "DRAINING" },
					population: func(int32) (int, int) { return 0, 2 },
				}
				srv := fixture.server(t)
				args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "get", "pd", "--stat"}
				if key {
					args = append(args, "--key", "123")
				}
				if jsonOutput {
					args = append(args, "--json")
				}
				stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)
				require.Empty(t, stderr)
				require.EqualValues(t, 1, fixture.definitions.Load())
				require.EqualValues(t, 6, fixture.counts.Load())
				if !jsonOutput {
					want := "123 tenant invoice v1 DRAINING [ac:- cp:7 cx:9 inc:1 su:2]\n"
					if !key {
						want += "found: 1\n"
					}
					require.Equal(t, want, stdout)
					return
				}
				decoder := json.NewDecoder(strings.NewReader(stdout))
				var envelope map[string]any
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
				require.Equal(t, "succeeded", envelope["outcome"])
				payload := requireJSONObject(t, envelope["payload"])
				if !key {
					payload = requireJSONObject(t, requireJSONItems(t, payload["items"], 1)[0])
				}
				require.Equal(t, "DRAINING", payload["state"])
				require.Equal(t, map[string]any{
					"active": float64(0), "suspended": float64(2), "completed": float64(7),
					"canceled": float64(9), "incidents": float64(1),
					"incidentCountSupported": true, "suspendedCountSupported": true,
				}, requireJSONObject(t, payload["statistics"]))
			})
		}
	}
}

// TestGetProcessDefinitionLifecycleStatisticsMixedRows proves that missing and
// unfamiliar state neither shifts statistics nor changes per-definition reads.
func TestGetProcessDefinitionLifecycleStatisticsMixedRows(t *testing.T) {
	for _, jsonOutput := range []bool{false, true} {
		t.Run(fmt.Sprintf("json=%t", jsonOutput), func(t *testing.T) {
			states := map[string]string{"123": "DRAINING", "124": "", "125": "future-State"}
			fixture := &definitionStatisticsFixture{
				version: "8.10", keys: []string{"123", "124", "125"},
				population: func(int32) (int, int) { return 0, 2 },
				lifecycle:  func(_ int32, key string) string { return states[key] },
			}
			srv := fixture.server(t)
			args := []string{"--config", writeTestConfigForVersion(t, srv.URL, "8.10"), "get", "pd", "--stat"}
			if jsonOutput {
				args = append(args, "--json")
			}
			stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)
			require.Empty(t, stderr)
			require.EqualValues(t, 1, fixture.definitions.Load())
			require.EqualValues(t, 18, fixture.counts.Load())
			if !jsonOutput {
				require.Equal(t, "123 tenant invoice v1 DRAINING     [ac:- cp:- cx:- inc:- su:2]\n124 tenant invoice v1              [ac:- cp:- cx:- inc:- su:2]\n125 tenant invoice v1 future-State [ac:- cp:- cx:- inc:- su:2]\nfound: 3\n", stdout)
				return
			}
			decoder := json.NewDecoder(strings.NewReader(stdout))
			var envelope map[string]any
			require.NoError(t, decoder.Decode(&envelope))
			require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
			require.Equal(t, "succeeded", envelope["outcome"])
			for i, item := range requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 3) {
				pd := requireJSONObject(t, item)
				require.Equal(t, fixture.keys[i], pd["key"])
				state := states[fixture.keys[i]]
				if state == "" {
					require.NotContains(t, pd, "state")
				} else {
					require.Equal(t, state, pd["state"])
				}
				require.Equal(t, float64(2), requireJSONObject(t, pd["statistics"])["suspended"])
			}
		})
	}
}

// TestGetProcessDefinitionLifecycleStatisticsFailure keeps failed enrichment a
// failed single envelope, never a successful lifecycle-only or partial-stat row.
func TestGetProcessDefinitionLifecycleStatisticsFailure(t *testing.T) {
	for _, key := range []bool{false, true} {
		t.Run(fmt.Sprintf("key=%t", key), func(t *testing.T) {
			fixture := &definitionStatisticsFixture{
				version: "8.10", countStatus: http.StatusForbidden,
				lifecycle: func(int32, string) string { return "DRAINING" },
			}
			srv := fixture.server(t)
			args := []string{"get", "pd", "--stat"}
			if key {
				args = append(args, "--key", "123")
			}
			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, processDefinitionErrorEnvelopeHelper, map[string]string{
				"C8VOLT_TEST_CONFIG":      writeTestConfigForVersion(t, srv.URL, "8.10"),
				"C8VOLT_TEST_OUTPUT_MODE": "json", "C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, args),
			})
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Empty(t, stderr)
			decoder := json.NewDecoder(strings.NewReader(stdout))
			var envelope map[string]any
			require.NoError(t, decoder.Decode(&envelope))
			require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
			require.Equal(t, "failed", envelope["outcome"])
			require.Equal(t, "local_precondition", envelope["class"])
			require.NotContains(t, envelope, "payload")
			require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "statistics unavailable")
			require.EqualValues(t, 1, fixture.definitions.Load())
			require.EqualValues(t, 1, fixture.counts.Load())
		})
	}
}
