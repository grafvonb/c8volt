// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDOperationsSlowEmpty proves sparse BID discovery exhausts before a prompt or enrichment.
func TestBusinessIDOperationsSlowEmpty(t *testing.T) {
	calls := &testx.SafeSlice[string]{}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.10") {
			return
		}
		calls.Append(r.URL.Path)
		require.Equal(t, "/v2/process-instances/search", r.URL.Path)
		var body struct {
			Filter map[string]any
			Page   map[string]any
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, " ORDER,零*? ", body.Filter["businessId"])
		w.Header().Set("Content-Type", "application/json")
		if body.Page["after"] == nil {
			fmt.Fprint(w, `{"items":[],"page":{"endCursor":"next","hasMoreTotalItems":true}}`)
		} else {
			require.Equal(t, "next", body.Page["after"])
			fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
		}
	}))
	defer server.Close()
	out, stderr, err := runBusinessIDCommand(t, []string{"--config", runBusinessIDConfig(t, server.URL, "plain-time", "info"), "ops", "analyse", "slow-pi", "--bid", " ORDER,零*? "}, "configured", false)
	require.NoError(t, err, stderr+out)
	require.Equal(t, "process instances: 0\n", out)
	require.NotContains(t, stderr, "Continue slow analysis")
	require.Equal(t, []string{"/v2/process-instances/search", "/v2/process-instances/search"}, calls.Snapshot())
}

// businessIDSlowFixture serves only owned discovery and enrichment endpoints and captures exact request order.
type businessIDSlowFixture struct {
	calls   testx.SafeSlice[string]
	filters testx.SafeSlice[map[string]any]
	empty   bool
	failure string
}

// serve supplies duplicate and sparse pages with a final distinct collection.
func (f *businessIDSlowFixture) serve(t *testing.T, w http.ResponseWriter, r *http.Request) {
	if serveVersionCheckTopology(w, r, "8.10") {
		return
	}
	f.calls.Append(r.URL.Path)
	w.Header().Set("Content-Type", "application/json")
	var body struct {
		Filter map[string]any
		Page   map[string]any
	}
	require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
	switch r.URL.Path {
	case "/v2/process-instances/search":
		f.filters.Append(body.Filter)
		switch body.Page["after"] {
		case nil:
			fmt.Fprint(w, `{"items":[],"page":{"endCursor":"first","hasMoreTotalItems":true}}`)
		case "first":
			if f.empty {
				fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
				return
			}
			row := map[string]any{"processInstanceKey": "2251799813685101", "state": "COMPLETED", "businessId": "observed", "tenantId": "tenant-a", "startDate": "2026-01-01T00:00:00Z", "endDate": "2026-01-01T00:01:00Z"}
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": []any{row, row}, "page": map[string]any{"totalItems": 10, "hasMoreTotalItems": true, "endCursor": "sparse"}}))
		case "sparse":
			if f.failure == "discovery" {
				w.WriteHeader(400)
				fmt.Fprint(w, `{"detail":"POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret"}`)
				return
			}
			fmt.Fprint(w, `{"items":[],"page":{"totalItems":10,"hasMoreTotalItems":true,"endCursor":"last"}}`)
		case "last":
			fmt.Fprint(w, `{"items":[{"processInstanceKey":"2251799813685102","state":"COMPLETED","tenantId":"tenant-a"},{"processInstanceKey":"2251799813685103","state":"COMPLETED","tenantId":"tenant-a"}],"page":{"totalItems":4,"hasMoreTotalItems":false}}`)
		default:
			t.Errorf("unexpected cursor %v", body.Page)
			http.Error(w, "unexpected", 500)
		}
	case "/v2/element-instances/search":
		require.Len(t, f.filters.Snapshot(), 4, "enrichment must follow complete discovery")
		key := body.Filter["processInstanceKey"]
		if f.failure == "enrichment" && key != "2251799813685101" {
			w.Header().Set("Set-Cookie", "secret-cookie")
			w.WriteHeader(400)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret; POST https://private.invalid/path?token=secret-token body-only-secret resource-key-secret"}`)
			return
		}
		fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
	case "/v2/jobs/search":
		fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
	default:
		t.Errorf("unexpected endpoint %s", r.URL.Path)
		http.Error(w, "unexpected", 500)
	}
}

// newBusinessIDSlowFixture keeps each subprocess backed by its own isolated HTTP server.
func newBusinessIDSlowFixture(t *testing.T, empty bool, failure string) (*businessIDSlowFixture, *httptest.Server) {
	t.Helper()
	f := &businessIDSlowFixture{empty: empty, failure: failure}
	s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { f.serve(t, w, r) }))
	t.Cleanup(s.Close)
	return f, s
}

// businessIDSlowArgs supplies a configured logger, named discovery tenant and opaque scalar selector.
func businessIDSlowArgs(t *testing.T, s *httptest.Server, format, level string, flags ...string) []string {
	t.Helper()
	return append([]string{"--config", runBusinessIDConfig(t, s.URL, format, level), "--tenant", "tenant-a", "ops", "analyse", "slow-pi", "--bid", " ORDER,零*? ", "--batch-size", "2"}, flags...)
}

// TestBusinessIDOperationsSlowModes covers real-terminal empty completion, output precedence and native predicates.
func TestBusinessIDOperationsSlowModes(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, mode := range []string{"human", "json", "quiet-json", "keys", "quiet-keys", "quiet", "automation", "auto-confirm", "listeners", "limit", "definition", "pd-key", "full"} {
			t.Run(fmt.Sprintf("empty=%t/%s", empty, mode), func(t *testing.T) {
				f, s := newBusinessIDSlowFixture(t, empty, "")
				flags := []string{"--batch-size", "2"}
				if !empty {
					flags = append(flags, "--auto-confirm")
				}
				switch mode {
				case "json":
					flags = append(flags, "--json")
				case "quiet-json":
					flags = append(flags, "--quiet", "--json", "--keys-only")
				case "keys":
					flags = append(flags, "--keys-only")
				case "quiet-keys":
					flags = append(flags, "--quiet", "--keys-only")
				case "quiet":
					flags = append(flags, "--quiet")
				case "automation":
					flags = append(flags, "--automation", "--json")
				case "auto-confirm":
					flags = append(flags, "--auto-confirm")
				case "listeners":
					flags = append(flags, "--with-listeners", "--json")
				case "limit":
					flags = append(flags, "--limit", "2", "--json")
				case "definition":
					flags = append(flags, "--bpmn-process-id", "order", "--state", "completed", "--no-incidents-only", "--start-date-after", "2026-01-01", "--end-date-before", "2026-12-01", "--json")
				case "pd-key":
					flags = append(flags, "--pd-key", "9001", "--json")
				case "full":
					flags = append(flags, "--with-full-timeline", "--json")
				}
				out, stderr, err := runBusinessIDCommand(t, businessIDSlowArgs(t, s, "plain-time", "info", flags...), "configured", false)
				require.NoError(t, err, stderr+out)
				require.NotContains(t, stderr, "Continue slow analysis")
				count := 3
				if empty {
					count = 0
				}
				if mode == "limit" && !empty {
					count = 2
				}
				for _, filter := range f.filters.Snapshot() {
					require.Equal(t, " ORDER,零*? ", filter["businessId"])
					require.Equal(t, "tenant-a", filter["tenantId"])
					if mode == "definition" {
						require.Equal(t, "order", filter["processDefinitionId"])
						require.Equal(t, false, filter["hasIncident"])
						require.Contains(t, filter, "startDate")
						require.Contains(t, filter, "endDate")
					}
					if mode == "pd-key" {
						require.Equal(t, "9001", filter["processDefinitionKey"])
					}
				}
				switch mode {
				case "human", "auto-confirm", "quiet":
					if empty && mode == "quiet" {
						require.Empty(t, out)
					} else {
						require.Contains(t, out, fmt.Sprintf("process instances: %d\n", count))
					}
				case "keys", "quiet-keys":
					if empty {
						require.Empty(t, out)
					} else {
						require.Equal(t, "2251799813685101\n2251799813685102\n2251799813685103\n", out)
					}
					require.Empty(t, stderr)
				default:
					payload := decodeJobTaskBusinessIDEnvelope(t, out)
					require.EqualValues(t, count, payload["count"])
					request := requireJSONObject(t, payload["request"])
					wantMode := "process_instance_search"
					if mode == "definition" || mode == "pd-key" {
						wantMode = "process_definition_search"
					}
					require.Equal(t, wantMode, request["selectionMode"])
					require.Equal(t, " ORDER,零*? ", requireJSONObject(t, request["processInstanceFilters"])["businessId"])
					require.Len(t, payload["items"], count)
					if empty {
						require.Equal(t, []any{}, payload["items"])
						preflight := requireJSONObject(t, payload["preflightScope"])
						require.NotContains(t, preflight, "pageCount")
						require.Equal(t, "unknown", preflight["pageCountKind"])
						consequence := requireJSONObject(t, preflight["consequenceSummary"])
						require.NotContains(t, consequence, "confirmationText")
					}
					selection := requireJSONObject(t, payload["selection"])
					completion := "exhausted"
					if mode == "limit" && !empty {
						completion = "limit_reached"
					}
					require.Equal(t, completion, selection["completion"])
					if count > 0 {
						items := payload["items"].([]any)
						require.Equal(t, "observed", requireJSONObject(t, items[0])["businessId"])
						require.NotContains(t, requireJSONObject(t, items[1]), "businessId")
					}
					require.Empty(t, stderr)
				}
				searches := 4
				if empty {
					searches = 2
				}
				require.Len(t, f.filters.Snapshot(), searches)
				enrichments := count
				if mode == "listeners" {
					enrichments *= 3
				}
				require.Len(t, f.calls.Snapshot(), searches+enrichments)
			})
		}
	}
}

// TestBusinessIDOperationsSlowConfirmation proves delayed preflight routes plain prompts and preserves decline/EOF.
func TestBusinessIDOperationsSlowConfirmation(t *testing.T) {
	for _, destination := range []string{"configured", "inherited"} {
		for _, answer := range []string{"y", "n", "\x04"} {
			t.Run(destination+"/"+answer, func(t *testing.T) {
				f, s := newBusinessIDSlowFixture(t, false, "")
				prompt := "Continue slow analysis for 3 process instances? [y/N]: "
				out, stderr, err := runBusinessIDCommand(t, businessIDSlowArgs(t, s, "plain-time", "info"), destination, false, testx.CmdTerminalExchange{Prompt: prompt, Response: answer, EndOfInput: answer == "\x04", BeforeResponse: func(out, stderr string) {
					require.Empty(t, out)
					require.Len(t, f.calls.Snapshot(), 4)
					require.Len(t, f.filters.Snapshot(), 4)
					require.True(t, strings.HasSuffix(stderr, prompt))
				}})
				require.Equal(t, 1, strings.Count(stderr, prompt), stderr)
				if answer == "y" {
					require.NoError(t, err, stderr)
					require.Contains(t, out, "process instances: 3\n")
					require.Len(t, f.calls.Snapshot(), 7)
				} else {
					require.Error(t, err)
					require.Empty(t, out)
					require.Len(t, f.calls.Snapshot(), 4)
					require.NotContains(t, stderr, "process instances: 0")
				}
			})
		}
	}
}

// TestBusinessIDOperationsSlowFailureLogging verifies aggregate diagnostics, partial evidence and independent redacted DEBUG.
func TestBusinessIDOperationsSlowFailureLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn", "debug-flag", "debug-config", "json", "quiet-json", "override"} {
				t.Run(format+"/"+destination+"/"+mode, func(t *testing.T) {
					f, s := newBusinessIDSlowFixture(t, false, "enrichment")
					level := "info"
					flags := []string{"--auto-confirm"}
					switch mode {
					case "verbose":
						flags = append(flags, "--verbose")
					case "quiet":
						flags = append(flags, "--quiet")
					case "warn":
						level = "warn"
					case "debug-flag":
						flags = append(flags, "--debug")
					case "debug-config":
						level = "debug"
					case "json":
						flags = append(flags, "--json")
					case "quiet-json":
						flags = append(flags, "--quiet", "--json")
					case "override":
						flags = append(flags, "--no-err-codes", "--json")
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDSlowArgs(t, s, format, level, flags...), destination, false)
					exit := 2
					if mode == "override" {
						exit = 0
					}
					assertRunBusinessIDExit(t, err, exit)
					require.Len(t, f.calls.Snapshot(), 7, "both failed resources are attempted once")
					machine := mode == "json" || mode == "quiet-json" || mode == "override"
					if machine {
						var envelope map[string]any
						decoder := json.NewDecoder(strings.NewReader(out))
						require.NoError(t, decoder.Decode(&envelope))
						require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
						require.Equal(t, "invalid_input", envelope["class"])
						require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
						payload := requireJSONObject(t, envelope["payload"])
						require.EqualValues(t, 1, payload["count"])
						require.Len(t, payload["items"], 1)
						require.Len(t, requireJSONObject(t, payload["selection"])["keys"], 3)
						require.Empty(t, stderr)
					} else {
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						errorMessages, debugs := []string{}, []string{}
						for _, record := range records {
							switch record[0] {
							case "ERROR":
								errorMessages = append(errorMessages, record[1])
							case "DEBUG":
								debugs = append(debugs, record[1])
							case "WARN":
								require.Equal(t, "slow analysis is expensive: load runtime element timelines for the selected instances", record[1])
							case "INFO":
								require.NotContains(t, []string{"quiet", "warn"}, mode)
							}
						}
						require.Equal(t, []string{"analyse slow process instances: selected 3, analysed 1; invalid input"}, errorMessages)
						require.Contains(t, out, "2251799813685101")
						require.NotContains(t, out, "2251799813685102")
						for _, secret := range []string{"private.invalid", "secret-token", "secret-cookie", "body-only-secret", "resource-key-secret"} {
							require.NotContains(t, stderr+out, secret)
						}
						if strings.HasPrefix(mode, "debug-") {
							require.NotEmpty(t, debugs)
							require.Contains(t, stderr, "POST /v2/")
						} else {
							require.Empty(t, debugs)
							require.NotContains(t, stderr+out, "POST ")
						}
						require.NotContains(t, out, "DEBUG")
						require.NotContains(t, out, "ERROR")
					}
				})
			}
		}
	}
}

// TestBusinessIDOperationsSlowDiscoveryFailure preserves incomplete selection and avoids a false empty report or prompt.
func TestBusinessIDOperationsSlowDiscoveryFailure(t *testing.T) {
	for _, mode := range []string{"human", "json"} {
		t.Run(mode, func(t *testing.T) {
			f, s := newBusinessIDSlowFixture(t, false, "discovery")
			flags := []string{}
			if mode == "json" {
				flags = append(flags, "--json")
			}
			out, stderr, err := runBusinessIDCommand(t, businessIDSlowArgs(t, s, "plain-time", "info", flags...), "configured", false)
			assertRunBusinessIDExit(t, err, 2)
			require.Len(t, f.calls.Snapshot(), 3)
			require.NotContains(t, stderr, "Continue slow analysis")
			if mode == "json" {
				var envelope map[string]any
				decoder := json.NewDecoder(strings.NewReader(out))
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
				payload := requireJSONObject(t, envelope["payload"])
				selection := requireJSONObject(t, payload["selection"])
				require.Equal(t, "incomplete", selection["completion"])
				require.Len(t, selection["keys"], 1)
				require.Empty(t, stderr)
			} else {
				require.Empty(t, out)
				require.Equal(t, [][2]string{{"ERROR", "analyse slow process instances: selected 1, analysed 0; invalid input"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
				require.NotContains(t, stderr, "private.invalid")
			}
		})
	}
}

// TestBusinessIDOperationsSlowInputGates rejects old versions and key/search conflicts without native requests.
func TestBusinessIDOperationsSlowInputGates(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, mode := range []string{"bid", "empty", "key", "stdin", "definitions", "limit"} {
			if version == "8.10" && mode == "bid" {
				continue
			}
			t.Run(version+"/"+mode, func(t *testing.T) {
				calls := &testx.SafeSlice[string]{}
				s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, version) {
						return
					}
					calls.Append(r.URL.Path)
					http.Error(w, "unexpected", 500)
				}))
				defer s.Close()
				flags := []string{"--bid", "ID", "--json"}
				switch mode {
				case "empty":
					flags = []string{"--bid=", "--json"}
				case "key":
					flags = append(flags, "--key", "2251799813685101")
				case "stdin":
					flags = append(flags, "-")
				case "definitions":
					flags = append(flags, "--bpmn-process-id", "order", "--pd-key", "9001")
				case "limit":
					flags = append(flags, "--limit", "0")
				}
				args := append([]string{"--config", testx.WriteTestConfigForVersion(t, s.URL, version), "ops", "analyse", "slow-pi"}, flags...)
				out, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
				require.Error(t, err)
				require.Empty(t, calls.Snapshot())
				require.Empty(t, stderr)
				var envelope map[string]any
				decoder := json.NewDecoder(strings.NewReader(out))
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
				require.NotEqual(t, "succeeded", envelope["outcome"])
				if mode == "bid" {
					require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "8.10")
				}
			})
		}
	}
}

// TestBusinessIDOperationsSlowWriteFailure allows only one envelope attempt and retains the operation exit.
func TestBusinessIDOperationsSlowWriteFailure(t *testing.T) {
	for _, failure := range []string{"", "enrichment"} {
		t.Run(failure, func(t *testing.T) {
			_, s := newBusinessIDSlowFixture(t, false, failure)
			out, stderr, err := runBusinessIDCommand(t, businessIDSlowArgs(t, s, "plain-time", "info", "--json"), "configured", true)
			code := 1
			if failure != "" {
				code = 2
			}
			assertRunBusinessIDExit(t, err, code)
			require.Equal(t, "attempt\n", out)
			require.Equal(t, [][2]string{{"ERROR", "analyse slow process instances: could not write result"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
		})
	}
}

// TestBusinessIDOperationsSlowProgressLogging requires configured INFO progress and honors quiet/WARN in both formats.
func TestBusinessIDOperationsSlowProgressLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"configured", "inherited"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "warn"} {
				t.Run(format+"/"+destination+"/"+mode, func(t *testing.T) {
					_, s := newBusinessIDSlowFixture(t, false, "")
					level := "info"
					flags := []string{"--auto-confirm"}
					switch mode {
					case "verbose":
						flags = append(flags, "--verbose")
					case "quiet":
						flags = append(flags, "--quiet")
					case "warn":
						level = "warn"
					}
					out, stderr, err := runBusinessIDCommand(t, businessIDSlowArgs(t, s, format, level, flags...), destination, false)
					require.NoError(t, err, stderr+out)
					require.Contains(t, out, "process instances: 3\n")
					require.NotContains(t, out, "business ID")
					require.NotContains(t, out, "INFO")
					if mode == "quiet" {
						require.Empty(t, stderr)
						return
					}
					if mode == "warn" {
						require.Equal(t, [][2]string{{"WARN", "slow analysis is expensive: load runtime element timelines for the selected instances"}}, bulkCompletionDiagnosticRecords(t, stderr, format))
						return
					}
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					require.NotEmpty(t, records)
					found := false
					for _, record := range records {
						if record[0] == "WARN" {
							require.Equal(t, "slow analysis is expensive: load runtime element timelines for the selected instances", record[1])
							continue
						}
						require.Equal(t, "INFO", record[0])
						if strings.Contains(record[1], `business ID " ORDER,零*? "`) {
							found = true
						}
					}
					require.True(t, found, "preflight must retain opaque selector")
				})
			}
		}
	}
}

// TestBusinessIDOperationsSlowOpaqueAndFiltered preserves exact opaque input and distinguishes selected from visible results.
func TestBusinessIDOperationsSlowOpaqueAndFiltered(t *testing.T) {
	for _, bid := range []string{"000123", " \tref,零*?\n<tag>| ", strings.Repeat("界", 300)} {
		t.Run(bid[:min(12, len(bid))], func(t *testing.T) {
			for _, visible := range []bool{true, false} {
				f, s := newBusinessIDSlowFixture(t, false, "")
				flags := []string{"--json", "--dur-longer", "30s"}
				if !visible {
					flags = []string{"--json", "--element-id", "absent"}
				}
				args := businessIDSlowArgs(t, s, "plain-time", "info", flags...)
				for i := range args {
					if args[i] == "--bid" {
						args[i+1] = bid
						break
					}
				}
				out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
				require.NoError(t, err, stderr+out)
				require.Empty(t, stderr)
				payload := decodeJobTaskBusinessIDEnvelope(t, out)
				require.Len(t, requireJSONObject(t, payload["selection"])["keys"], 3)
				require.Equal(t, bid, requireJSONObject(t, requireJSONObject(t, payload["request"])["processInstanceFilters"])["businessId"])
				count := 0
				if visible {
					count = 1
				}
				require.EqualValues(t, count, payload["count"])
				require.Len(t, f.calls.Snapshot(), 7)
				for _, filter := range f.filters.Snapshot() {
					require.Equal(t, bid, filter["businessId"])
				}
			}
		})
	}
	// A newline in the criterion must remain escaped in the configured preflight logger.
	_, s := newBusinessIDSlowFixture(t, false, "")
	args := businessIDSlowArgs(t, s, "plain-time", "info", "--auto-confirm")
	for i := range args {
		if args[i] == "--bid" {
			args[i+1] = "line\nnext"
			break
		}
	}
	out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
	require.NoError(t, err, stderr+out)
	records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
	require.Contains(t, records[0][1], `business ID "line\nnext"`)
	require.NotContains(t, stderr, "line\nnext")
}
