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
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// definitionStatisticsFixture serves only test-owned definition/count reads.
// Atomic counters also let watch tests select a population for each refresh.
type definitionStatisticsFixture struct {
	definitions atomic.Int32
	counts      atomic.Int32
	population  func(int32) (int, int)
	empty       bool
	version     string
}

// server rejects all unexpected requests, including mutations, and returns
// exact totals so output tests do not depend on traversal or a running engine.
func (f *definitionStatisticsFixture) server(t *testing.T) *httptest.Server {
	t.Helper()
	srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		definition := `{"processDefinitionKey":"123","processDefinitionId":"invoice","version":1,"tenantId":"tenant"}`
		switch r.Method + " " + r.URL.Path {
		case "GET /v2/process-definitions/123":
			f.definitions.Add(1)
			_, _ = io.WriteString(w, definition)
		case "POST /v2/process-definitions/search":
			f.definitions.Add(1)
			if f.empty {
				_, _ = io.WriteString(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
			} else {
				fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, definition)
			}
		case "POST /v2/process-instances/search":
			f.counts.Add(1)
			var body struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "123", body.Filter["processDefinitionKey"])
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
				case "COMPLETED", "TERMINATED":
				default:
					t.Errorf("unexpected state %q", state)
				}
			case map[string]any:
				require.Equal(t, "8.10", f.version)
				require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, state["$in"])
				count = active + suspended
			default:
				require.Equal(t, true, body.Filter["hasIncident"])
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
