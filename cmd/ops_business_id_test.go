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
	"strconv"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

const opsBusinessIDOpaque = "old,\"ID\"|<tag>&*_[link](url)`~~strike~~\\\r\nnext"

// newOpsBusinessIDDefinitionFixture supplies completed discovery for both the
// older active-state and v810 unfinished-union planning contracts.
func newOpsBusinessIDDefinitionFixture(t *testing.T, version string, requests *testx.SafeSlice[string]) *httptest.Server {
	return testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		definition := opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 1)
		switch {
		case r.URL.Path == "/v2/process-definitions/search":
			fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, definition)
		case r.URL.Path == "/v2/process-definitions/"+opsAllProcessDefinitionsPurgePDKeyA:
			fmt.Fprint(w, definition)
		case r.URL.Path == "/v2/process-instances/search":
			var body struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if body.Filter["parentProcessInstanceKey"] != nil || body.Filter["state"] == "SUSPENDED" {
				fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
				return
			}
			fmt.Fprintf(w, `{"items":[%s,%s],"page":{"totalItems":2}}`, opsAllProcessDefinitionsPurgeProcessInstanceJSON(opsAllProcessDefinitionsPurgeRootA, opsAllProcessDefinitionsPurgeRootA, opsAllProcessDefinitionsPurgePDKeyA, "ACTIVE"), opsAllProcessDefinitionsPurgeProcessInstanceJSON(opsAllProcessDefinitionsPurgeRootB, opsAllProcessDefinitionsPurgeRootB, opsAllProcessDefinitionsPurgePDKeyA, "ACTIVE"))
		case strings.HasPrefix(r.URL.Path, "/v2/process-instances/") && r.Method == "GET":
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			require.Contains(t, []string{opsAllProcessDefinitionsPurgeRootA, opsAllProcessDefinitionsPurgeRootB}, key)
			fmt.Fprint(w, opsAllProcessDefinitionsPurgeProcessInstanceJSON(key, key, opsAllProcessDefinitionsPurgePDKeyA, "ACTIVE"))
		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
}

// opsBusinessIDServer adds native identity to existing test-owned service fixtures.
// It refuses mutations during preview tests and never proxies to a real endpoint.
func opsBusinessIDServer(t *testing.T, backend *httptest.Server, version string, ids map[string]*string) *httptest.Server {
	t.Helper()
	t.Cleanup(backend.Close)
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}
		if r.Method != http.MethodGet && !(r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/search")) {
			t.Errorf("unexpected mutation: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
			return
		}
		recorder := httptest.NewRecorder()
		backend.Config.Handler.ServeHTTP(recorder, r)
		for k, vs := range recorder.Header() {
			for _, v := range vs {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(recorder.Code)
		var data any
		if json.Unmarshal(recorder.Body.Bytes(), &data) != nil {
			_, _ = w.Write(recorder.Body.Bytes())
			return
		}
		var decorate func(any)
		decorate = func(v any) {
			switch v := v.(type) {
			case []any:
				for _, item := range v {
					decorate(item)
				}
			case map[string]any:
				if key, ok := v["processInstanceKey"].(string); ok && v["state"] != nil {
					if id := ids[key]; id != nil {
						v["businessId"] = *id
					}
				}
				for _, item := range v {
					decorate(item)
				}
			}
		}
		decorate(data)
		require.NoError(t, json.NewEncoder(w).Encode(data))
	}))
	t.Cleanup(server.Close)
	return server
}

// TestOpsBusinessIDPurgeCommands verifies saved files, per-resource identity,
// older-version absence, configured log routing and read-free rendering.
func TestOpsBusinessIDPurgeCommands(t *testing.T) {
	for _, workflow := range []string{"incident", "orphan", "definitions", "repair"} {
		for _, version := range []string{"8.8", "8.9", "8.10"} {
			if workflow == "definitions" && version == "8.8" {
				continue
			} // Existing capability gate rejects history purge before discovery.
			for _, mode := range []string{"verbose", "json", "quiet", "warn"} {
				t.Run(workflow+"/"+version+"/"+mode, func(t *testing.T) {
					var requests testx.SafeSlice[string]
					var backend *httptest.Server
					var command []string
					ids := map[string]*string{}
					switch workflow {
					case "incident":
						backend = newOpsIncidentPurgeServer(t, version, &requests, nil, true)
						command = []string{"ops", "purge", "process-instances-with-incidents"}
						ids[opsIncidentPurgeRootKey], ids[opsIncidentPurgeChildKey] = toolx.Ptr("new"), toolx.Ptr(opsBusinessIDOpaque)
					case "orphan":
						backend = newOpsOrphanPurgeServerWithState(t, &requests, nil, true, "COMPLETED")
						command = []string{"ops", "purge", "orphan-process-instances"}
						ids[opsOrphanChildKey] = toolx.Ptr(opsBusinessIDOpaque)
					case "definitions":
						backend = newOpsBusinessIDDefinitionFixture(t, version, &requests)
						command = []string{"ops", "purge", "all-process-definitions", "--force"}
						ids[opsAllProcessDefinitionsPurgeRootA], ids[opsAllProcessDefinitionsPurgeRootB] = toolx.Ptr("new"), toolx.Ptr(opsBusinessIDOpaque)
					case "repair":
						backend = newOpsRepairProcessInstanceServer(t, &requests)
						command = []string{"ops", "repair", "pi", "--key", "2251799813685251"}
						ids["2251799813685251"] = toolx.Ptr(opsBusinessIDOpaque)
					}
					srv := opsBusinessIDServer(t, backend, version, ids)
					config := writeTestConfigForVersion(t, srv.URL, version)
					// The version helper uses a plain logger; explicit overrides exercise structured output too.
					format := "plain-time"
					if version == "8.10" {
						format = "json"
					}
					level := "info"
					if mode == "warn" {
						level = "warn"
					}
					configData, err := os.ReadFile(config)
					require.NoError(t, err)
					configData = append(configData, []byte(fmt.Sprintf("\nlog:\n  level: %s\n  format: %s\n", level, format))...)
					require.NoError(t, os.WriteFile(config, configData, 0600))
					for _, extension := range []string{"json", "md"} {
						path := filepath.Join(t.TempDir(), "audit."+extension)
						args := append([]string{"--config", config}, command...)
						args = append(args, "--dry-run", "--report-file", path)
						if mode != "warn" {
							args = append(args, "--"+mode)
						}
						start := len(requests.Snapshot())
						stdout, stderr, err := runBusinessIDCommand(t, args, "configured", false)
						require.NoError(t, err, stdout+stderr)
						data, err := os.ReadFile(path)
						require.NoError(t, err)
						expected := map[string]*string{}
						for key, id := range ids {
							if version == "8.10" {
								expected[key] = id
							} else {
								expected[key] = nil
							}
						}
						if extension == "json" {
							var payload any
							require.NoError(t, json.Unmarshal(data, &payload))
							if workflow != "repair" {
								require.Positive(t, assertOpsBusinessIDObservations(t, payload, expected))
							}
						} else if workflow != "repair" && version == "8.10" {
							require.Contains(t, string(data), `\|\<tag\>`)
						}
						if workflow == "repair" {
							require.NotContains(t, string(data), "businessId")
							require.NotContains(t, string(data), "bid:")
						}
						if mode == "json" {
							payload := decodeJobTaskBusinessIDEnvelope(t, stdout)
							if workflow != "repair" {
								require.Positive(t, assertOpsBusinessIDObservations(t, payload, expected))
							}
						} else {
							require.Empty(t, stdout)
						}
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						var messages []string
						for _, record := range records {
							require.Equal(t, "INFO", record[0])
							messages = append(messages, record[1])
						}
						if mode != "verbose" {
							require.Empty(t, messages)
						} else if workflow != "repair" && version == "8.10" {
							require.Contains(t, strings.Join(messages, "\n"), "bid:"+strconv.Quote(opsBusinessIDOpaque))
						}
						wantReads := map[string]int{"incident": 5, "orphan": 6, "definitions": 15, "repair": 2}[workflow]
						if workflow == "definitions" && version == "8.9" {
							wantReads = 13
						}
						require.Equal(t, wantReads, len(requests.Snapshot())-start)
					}
				})
			}
		}
	}
}

// TestOpsBusinessIDReports covers every custom report containing observations,
// including nil, empty and independent child values, without altering the model.
func TestOpsBusinessIDReports(t *testing.T) {
	items := []process.ProcessInstance{
		{Key: "root", State: "COMPLETED", BusinessId: toolx.Ptr("new")},
		{Key: "child", State: "COMPLETED", BusinessId: toolx.Ptr(opsBusinessIDOpaque)},
		{Key: "absent", State: "COMPLETED"},
		{Key: "empty", State: "COMPLETED", BusinessId: toolx.Ptr("")},
	}
	expansion := process.DryRunPIKeyExpansion{SelectedFinalState: items, RequiresCancelBeforeDelete: items}
	for _, tc := range []struct {
		name     string
		markdown func() ([]byte, error)
		json     func() ([]byte, error)
	}{
		{"retention", func() ([]byte, error) {
			return renderOpsExecuteRetentionPolicyMarkdownReport(ops.RetentionAuditReport{DeletePlan: ops.RetentionDeletePlan{FinalStateItems: items, NonFinalAffectedItems: items, SkippedNonFinalRoots: items}}, nil)
		}, func() ([]byte, error) {
			return renderOpsExecuteRetentionPolicyJSONReport(ops.RetentionAuditReport{DeletePlan: ops.RetentionDeletePlan{FinalStateItems: items, NonFinalAffectedItems: items, SkippedNonFinalRoots: items}})
		}},
		{"incident", func() ([]byte, error) {
			return renderOpsPurgeProcessInstancesWithIncidentsMarkdownReport(ops.IncidentPurgeReport{DeletePlan: ops.IncidentPurgeDeletePlan{FinalStateItems: items, NonFinalAffectedItems: items}}, nil)
		}, func() ([]byte, error) {
			return renderOpsPurgeProcessInstancesWithIncidentsJSONReport(ops.IncidentPurgeReport{DeletePlan: ops.IncidentPurgeDeletePlan{FinalStateItems: items, NonFinalAffectedItems: items}})
		}},
		{"orphan", func() ([]byte, error) {
			return renderOpsPurgeOrphanProcessInstancesMarkdownReport(ops.OrphanPurgeReport{DeletionPlan: ops.DeletionPlan{DryRunPreview: expansion}}, nil)
		}, func() ([]byte, error) {
			return renderOpsPurgeOrphanProcessInstancesJSONReport(ops.OrphanPurgeReport{DeletionPlan: ops.DeletionPlan{DryRunPreview: expansion}})
		}},
		{"definitions", func() ([]byte, error) {
			return renderOpsPurgeAllProcessDefinitionsMarkdownReport(ops.AllProcessDefinitionsPurgeReport{DeletePlan: ops.AllProcessDefinitionsPurgeDeletePlan{Items: []resource.DeleteProcessDefinitionPlanItem{{Key: "definition", CancellationPlan: expansion}}}}, nil)
		}, func() ([]byte, error) {
			return renderOpsPurgeAllProcessDefinitionsJSONReport(ops.AllProcessDefinitionsPurgeReport{DeletePlan: ops.AllProcessDefinitionsPurgeDeletePlan{Items: []resource.DeleteProcessDefinitionPlanItem{{Key: "definition", CancellationPlan: expansion}}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.markdown()
			require.NoError(t, err)
			text := string(data)
			require.Contains(t, text, `key=root state=COMPLETED bid:new`)
			require.Contains(t, text, `key=empty state=COMPLETED bid:""`)
			require.Contains(t, text, "key=absent state=COMPLETED\n")
			require.Contains(t, text, `\|\<tag\>\&\*\_\[link\]\(url\)`)
			require.NotContains(t, text, "\nnext")
			require.NotContains(t, text, "<tag>")
			require.Contains(t, text, `\~\~strike\~\~`)
			data, err = tc.json()
			require.NoError(t, err)
			var payload any
			require.NoError(t, json.Unmarshal(data, &payload))
			assertOpsBusinessIDObservations(t, payload, map[string]*string{"root": toolx.Ptr("new"), "child": toolx.Ptr(opsBusinessIDOpaque), "absent": nil, "empty": toolx.Ptr("")})
		})
	}
}

// assertOpsBusinessIDObservations inspects only actual resource observations.
func assertOpsBusinessIDObservations(t *testing.T, value any, expected map[string]*string) int {
	t.Helper()
	count := 0
	switch value := value.(type) {
	case []any:
		for _, item := range value {
			count += assertOpsBusinessIDObservations(t, item, expected)
		}
	case map[string]any:
		if key, ok := value["key"].(string); ok {
			if want, exists := expected[key]; exists && value["state"] != nil {
				count++
				if want == nil {
					require.NotContains(t, value, "businessId")
				} else {
					require.Equal(t, *want, value["businessId"], "resource %s", key)
				}
			}
		}
		for _, item := range value {
			count += assertOpsBusinessIDObservations(t, item, expected)
		}
	}
	return count
}

// TestOpsBusinessIDRetentionCommand checks real saved reports and configured
// logging with terminal stdin, separately captured streams and owned HTTP reads.
func TestOpsBusinessIDRetentionCommand(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, mode := range []string{"verbose", "quiet", "warn", "json"} {
			for _, extension := range []string{"json", "md"} {
				t.Run(format+"/"+mode+"/"+extension, func(t *testing.T) {
					f := newObservedBusinessIDFixture(t)
					expected := map[string]*string{}
					for key, item := range f.instances {
						item["state"] = "COMPLETED"
						item["endDate"] = "2026-10-02T00:00:00Z"
						if id, ok := item["businessId"].(string); ok {
							expected[key] = toolx.Ptr(id)
						} else {
							expected[key] = nil
						}
					}
					f.instances["2251799813685102"]["businessId"] = opsBusinessIDOpaque
					expected["2251799813685102"] = toolx.Ptr(opsBusinessIDOpaque)
					level := "info"
					if mode == "warn" {
						level = "warn"
					}
					path := filepath.Join(t.TempDir(), "audit."+extension)
					args := []string{"--config", runBusinessIDConfig(t, f.server.URL, format, level), "ops", "execute", "retention-policy", "--retention-days", "1", "--dry-run", "--report-file", path}
					if mode != "warn" {
						args = append(args, "--"+mode)
					}
					destination := "configured"
					if format == "json" {
						destination = "inherited"
					}
					stdout, stderr, err := runBusinessIDCommand(t, args, destination, false)
					require.NoError(t, err)
					records := bulkCompletionDiagnosticRecords(t, stderr, format)
					if mode == "json" {
						payload := decodeJobTaskBusinessIDEnvelope(t, stdout)
						require.Positive(t, assertOpsBusinessIDObservations(t, payload, expected))
						require.Empty(t, records)
					} else {
						require.Empty(t, stdout)
						if mode == "verbose" {
							var messages []string
							for _, record := range records {
								require.Equal(t, "INFO", record[0])
								messages = append(messages, record[1])
							}
							require.Contains(t, strings.Join(messages, "\n"), "bid:"+strconv.Quote(opsBusinessIDOpaque))
						} else {
							require.Empty(t, records)
						}
					}
					data, err := os.ReadFile(path)
					require.NoError(t, err)
					if extension == "json" {
						var payload any
						require.NoError(t, json.Unmarshal(data, &payload))
						require.Positive(t, assertOpsBusinessIDObservations(t, payload, expected))
					} else {
						require.Contains(t, string(data), `bid:""`)
						require.Contains(t, string(data), `\|\<tag\>`)
					}
					require.Len(t, f.requests.Snapshot(), 13, "only discovery and existing traversal reads")
				})
			}
		}
	}
}
