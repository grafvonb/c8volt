// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// jobTaskBusinessIDFixture owns all observations and rejects unrelated parent reads and mutations.
func jobTaskBusinessIDFixture(t *testing.T, version string, empty bool, completionStatus ...int) (*httptest.Server, *testx.SafeSlice[string]) {
	t.Helper()
	requests := &testx.SafeSlice[string]{}
	srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		rows := []map[string]any{}
		for i, key := range []string{"2251799813685101", "2251799813685102", "2251799813685103"} {
			row := map[string]any{"jobKey": key, "userTaskKey": key, "state": "CREATED", "retries": 1, "kind": "BPMN_ELEMENT", "type": "work", "elementId": "task", "elementInstanceKey": "500", "processInstanceKey": "900", "processDefinitionKey": "800", "processDefinitionId": "order", "tenantId": "<default>"}
			if i == 0 {
				row["businessId"] = "old,\"ID\"\n"
			}
			if i == 2 {
				row["businessId"] = ""
			}
			rows = append(rows, row)
		}
		page := func(items []map[string]any) {
			require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items), "hasMoreTotalItems": false}}))
		}
		switch {
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/completion") && len(completionStatus) > 0:
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.NotContains(t, body, "businessId")
			status := completionStatus[0]
			if strings.Contains(r.URL.Path, "/2251799813685101/") {
				status = http.StatusNoContent
			}
			w.WriteHeader(status)
			if status != http.StatusNoContent {
				fmt.Fprint(w, `{"title":"Conflict","detail":"POST https://private.invalid/secret body-only-secret resource-key-secret leaseToken"}`)
			}
		case r.Method == "POST" && r.URL.Path == "/v2/jobs/search":
			var body struct {
				Filter map[string]any `json:"filter"`
			}
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			if empty {
				page([]map[string]any{})
				return
			}
			if key, ok := body.Filter["jobKey"].(string); ok {
				for _, row := range rows {
					if row["jobKey"] == key {
						page([]map[string]any{row})
						return
					}
				}
			}
			page(rows)
		case r.Method == "POST" && r.URL.Path == "/v2/user-tasks/search":
			if empty {
				page([]map[string]any{})
				return
			}
			page(rows)
		case r.Method == "POST" && strings.HasSuffix(r.URL.Path, "/effective-variables/search"):
			page([]map[string]any{})
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/v2/user-tasks/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/user-tasks/")
			for _, row := range rows {
				if row["userTaskKey"] == key {
					require.NoError(t, json.NewEncoder(w).Encode(row))
					return
				}
			}
			t.Errorf("unexpected key %s", key)
			w.WriteHeader(400)
		default:
			t.Errorf("unexpected read or mutation: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(400)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, requests
}

// decodeJobTaskBusinessIDEnvelope rejects duplicate envelopes and retains arbitrary existing payload shapes.
func decodeJobTaskBusinessIDEnvelope(t *testing.T, out string) map[string]any {
	t.Helper()
	var envelope map[string]any
	dec := json.NewDecoder(strings.NewReader(out))
	require.NoError(t, dec.Decode(&envelope))
	require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
	require.Equal(t, "succeeded", envelope["outcome"])
	return requireJSONObject(t, envelope["payload"])
}

// assertJobTaskBusinessID preserves nil, empty and opaque historical facts without a parent lookup.
func assertJobTaskBusinessID(t *testing.T, item map[string]any, version string) {
	t.Helper()
	if version != "8.10" {
		require.NotContains(t, item, "businessId")
		return
	}
	switch item["key"] {
	case "2251799813685101":
		require.Equal(t, "old,\"ID\"\n", item["businessId"])
	case "2251799813685102":
		require.NotContains(t, item, "businessId")
	case "2251799813685103":
		require.Equal(t, "", item["businessId"])
	default:
		t.Fatalf("unexpected item %v", item)
	}
}

// TestJobTaskBusinessIDQueries exercises native single/search/enriched paths and keeps query results on stdout.
func TestJobTaskBusinessIDQueries(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, resource := range []string{"job", "ut"} {
			for _, selection := range []string{"single", "list", "vars"} {
				if selection == "vars" && resource == "job" {
					continue
				}
				for _, mode := range []string{"human", "json", "keys", "quiet-json"} {
					if selection == "vars" && mode == "keys" {
						continue
					}
					t.Run(fmt.Sprintf("%s/%s/%s/%s", version, resource, selection, mode), func(t *testing.T) {
						srv, requests := jobTaskBusinessIDFixture(t, version, false)
						args := []string{"--config", testx.WriteTestConfigForVersion(t, srv.URL, version), "get", resource}
						count := 3
						if selection == "single" {
							args = append(args, "--key", "2251799813685101")
							count = 1
						}
						if selection == "vars" {
							args = append(args, "--with-vars")
						}
						if strings.Contains(mode, "json") {
							args = append(args, "--json")
						}
						if mode == "keys" {
							args = append(args, "--keys-only")
						}
						if mode == "quiet-json" {
							args = append(args, "--quiet")
						}
						out, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
						require.NoError(t, err, stderr)
						require.Empty(t, stderr)
						switch mode {
						case "keys":
							want := "2251799813685101\n"
							if count == 3 {
								want += "2251799813685102\n2251799813685103\n"
							}
							require.Equal(t, want, out)
						case "human":
							if version == "8.10" {
								require.Contains(t, out, "business-id:"+strconv.Quote("old,\"ID\"\n"))
								if count == 3 {
									require.Contains(t, out, `business-id:""`)
								}
							} else {
								require.NotContains(t, out, "business-id:")
							}
						default:
							payload := decodeJobTaskBusinessIDEnvelope(t, out)
							if selection == "single" && resource == "job" {
								assertJobTaskBusinessID(t, payload, version)
							} else {
								items := payload["items"].([]any)
								require.Len(t, items, count)
								for _, raw := range items {
									item := requireJSONObject(t, raw)
									if selection == "vars" {
										require.Equal(t, []any{}, item["variables"])
										item = requireJSONObject(t, item["item"])
									}
									assertJobTaskBusinessID(t, item, version)
								}
							}
						}
						reads := 1
						if selection == "vars" {
							reads += 3
						}
						require.Len(t, requests.Snapshot(), reads)
					})
				}
			}
		}
	}
}

// TestJobBusinessIDPlans routes scalar and frozen-batch preview identities through configured INFO and exact machine payloads.
func TestJobBusinessIDPlans(t *testing.T) {
	for _, selection := range []string{"retry", "retry-noop", "scalar-complete", "batch"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, destination := range []string{"configured", "inherited"} {
				for _, mode := range []string{"info", "warn", "quiet", "json"} {
					t.Run(selection+"/"+format+"/"+destination+"/"+mode, func(t *testing.T) {
						srv, requests := jobTaskBusinessIDFixture(t, "8.10", false)
						level := "info"
						if mode == "warn" {
							level = "warn"
						}
						args := []string{"--config", runBusinessIDConfig(t, srv.URL, format, level), "update", "job", "--dry-run"}
						switch selection {
						case "retry":
							args = append(args, "--key", "2251799813685101", "--retries", "3")
						case "retry-noop":
							args = append(args, "--key", "2251799813685101", "--retries", "1")
						case "scalar-complete":
							args = append(args, "--key", "2251799813685101", "--complete")
						case "batch":
							args = append(args, "--type", "work", "--complete")
						}
						if mode == "json" {
							args = append(args, "--json")
						} else {
							args = append(args, "--verbose")
						}
						if mode == "quiet" {
							args = append(args, "--quiet")
						}
						out, stderr, err := runBusinessIDCommand(t, args, destination, false)
						require.NoError(t, err, stderr)
						if mode == "json" {
							require.Empty(t, stderr)
							payload := decodeJobTaskBusinessIDEnvelope(t, out)
							if selection == "batch" {
								items := payload["candidates"].([]any)
								require.Len(t, items, 3)
								for _, item := range items {
									assertJobTaskBusinessID(t, requireJSONObject(t, item), "8.10")
								}
							} else {
								assertJobTaskBusinessID(t, requireJSONObject(t, payload["current"]), "8.10")
							}
						} else {
							require.Empty(t, out)
							records := bulkCompletionDiagnosticRecords(t, stderr, format)
							if mode != "info" {
								require.Empty(t, records)
							} else {
								require.NotEmpty(t, records)
								ids := 0
								for _, record := range records {
									require.Equal(t, "INFO", record[0])
									if strings.Contains(record[1], "business-id:") {
										ids++
										require.True(t, strings.Contains(record[1], `business-id:"old,\"ID\"\n"`) || strings.Contains(record[1], `business-id:""`))
									}
								}
								want := 1
								if selection == "batch" {
									want = 2
								}
								require.Equal(t, want, ids)
							}
						}
						require.Equal(t, []string{"POST /v2/jobs/search"}, requests.Snapshot())
					})
				}
			}
		}
	}
}

// TestJobTaskBusinessIDEmpty keeps successful no-op/query shapes and terminal completion prompt-free with no mutation or extra reads.
func TestJobTaskBusinessIDEmpty(t *testing.T) {
	for _, resource := range []string{"job", "ut", "completion", "preview"} {
		for _, mode := range []string{"human", "quiet", "json", "quiet-json", "keys", "quiet-keys"} {
			if (resource == "completion" || resource == "preview") && strings.Contains(mode, "keys") {
				continue
			}
			for _, control := range [][]string{nil, {"--auto-confirm"}, {"--automation"}, {"--no-wait"}} {
				if len(control) > 0 && control[0] == "--no-wait" && (resource == "job" || resource == "ut") {
					continue
				}
				t.Run(fmt.Sprintf("%s/%s/%v", resource, mode, control), func(t *testing.T) {
					srv, requests := jobTaskBusinessIDFixture(t, "8.10", true)
					args := []string{"--config", runBusinessIDConfig(t, srv.URL, "plain-time", "info")}
					mutation := resource == "completion" || resource == "preview"
					if mutation {
						args = append(args, "update", "job", "--complete", "--type", "work")
						if resource == "preview" {
							args = append(args, "--dry-run")
						}
					} else {
						args = append(args, "get", resource)
					}
					args = append(args, control...)
					if strings.Contains(mode, "json") {
						args = append(args, "--json")
						if mutation {
							args = append(args, "--auto-confirm")
						}
					}
					if strings.Contains(mode, "keys") {
						args = append(args, "--keys-only")
					}
					if strings.HasPrefix(mode, "quiet") {
						args = append(args, "--quiet")
					}
					out, stderr, err := runBusinessIDCommand(t, args, "configured", false)
					require.NoError(t, err, stderr)
					if strings.Contains(mode, "json") {
						require.Empty(t, stderr)
						payload := decodeJobTaskBusinessIDEnvelope(t, out)
						if mutation {
							counts := requireJSONObject(t, payload["counts"])
							require.Equal(t, float64(0), counts["selected"])
							require.Empty(t, payload["outcomes"])
							require.Empty(t, payload["candidates"])
						} else if resource == "ut" {
							require.Equal(t, map[string]any{"total": float64(0), "items": []any{}}, payload)
						} else {
							require.Empty(t, payload["items"])
						}
					} else if mode == "human" {
						if mutation {
							require.Empty(t, out)
							records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
							require.NotEmpty(t, records)
							for _, record := range records {
								require.Equal(t, "INFO", record[0])
							}
							require.Contains(t, records[len(records)-1][1], "no changes applied")
						} else {
							require.Empty(t, stderr)
							require.Equal(t, "found: 0\n", out)
						}
					} else {
						require.Empty(t, out)
						require.Empty(t, stderr)
					}
					path := "/v2/jobs/search"
					if resource == "ut" {
						path = "/v2/user-tasks/search"
					}
					require.Equal(t, []string{"POST " + path}, requests.Snapshot())
				})
			}
		}
	}
}

// TestJobBusinessIDCompletionOutcomes preserves each candidate in accepted/partial-failure envelopes and verbose logged outcomes.
func TestJobBusinessIDCompletionOutcomes(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusConflict} {
		for _, mode := range []string{"human", "verbose", "quiet", "json", "quiet-json"} {
			for _, format := range []string{"plain-time", "json"} {
				t.Run(fmt.Sprintf("%d/%s/%s", status, mode, format), func(t *testing.T) {
					srv, requests := jobTaskBusinessIDFixture(t, "8.10", false, status)
					args := []string{"--config", runBusinessIDConfig(t, srv.URL, format, "info"), "update", "job", "--complete", "--type", "work", "--auto-confirm", "--no-wait", "--workers", "2"}
					if strings.Contains(mode, "json") {
						args = append(args, "--json")
					}
					if mode == "verbose" {
						args = append(args, "--verbose")
					}
					if strings.HasPrefix(mode, "quiet") {
						args = append(args, "--quiet")
					}
					out, stderr, err := runBusinessIDCommand(t, args, "inherited", false)
					exit := 0
					if status == http.StatusConflict {
						exit = 6
					}
					assertRunBusinessIDExit(t, err, exit)
					if strings.Contains(mode, "json") {
						dec := json.NewDecoder(strings.NewReader(out))
						var envelope map[string]any
						require.NoError(t, dec.Decode(&envelope))
						require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
						require.Empty(t, stderr)
						payload := requireJSONObject(t, envelope["payload"])
						outcomes := payload["outcomes"].([]any)
						require.Len(t, outcomes, 3)
						for _, raw := range outcomes {
							assertJobTaskBusinessID(t, requireJSONObject(t, requireJSONObject(t, raw)["candidate"]), "8.10")
						}
						counts := requireJSONObject(t, payload["counts"])
						accepted := float64(3)
						if status == http.StatusConflict {
							accepted = 1
							require.Equal(t, "conflict", envelope["class"])
							require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
						}
						require.Equal(t, accepted, counts["accepted"])
					} else {
						require.Empty(t, out)
						records := bulkCompletionDiagnosticRecords(t, stderr, format)
						errors, identities := 0, 0
						for _, record := range records {
							if record[0] == "ERROR" {
								errors++
							} else {
								require.Equal(t, "INFO", record[0])
								require.NotEqual(t, "quiet", mode)
							}
							if strings.Contains(record[1], "business-id:") {
								identities++
							}
						}
						wantErrors := 0
						if status == http.StatusConflict {
							wantErrors = 1
						}
						require.Equal(t, wantErrors, errors)
						if mode == "verbose" {
							require.Equal(t, 2, identities)
						} else {
							require.Zero(t, identities)
						}
						for _, secret := range []string{"https://private.invalid", "POST ", "body-only-secret", "resource-key-secret", "leaseToken"} {
							require.NotContains(t, out+stderr, secret)
						}
					}
					reads := 7
					if status == http.StatusConflict {
						reads += 2
					}
					require.Len(t, requests.Snapshot(), reads, "discovery, rechecks, submissions and existing conflict reconciliation only")
				})
			}
		}
	}
}

// TestJobBusinessIDLocalPlanSnapshot keeps copied plan observations independent without fetching enrichment.
func TestJobBusinessIDLocalPlanSnapshot(t *testing.T) {
	id := "old"
	source := job.Job{Key: "2251799813685101", BusinessId: &id}
	retry := buildUpdateJobPlan(source, job.UpdateRequest{Key: source.Key})
	complete := buildWorkerOutcomeUpdatePlan(source, job.UpdateRequest{Key: source.Key, WorkerOutcome: &job.WorkerOutcomeRequest{Mode: job.WorkerOutcomeCompletion}})
	id = "changed"
	require.Equal(t, "old", *retry.Current.BusinessId)
	require.Equal(t, "old", *complete.Current.BusinessId)
	absent := buildUpdateJobPlan(job.Job{Key: source.Key}, job.UpdateRequest{Key: source.Key})
	require.Nil(t, absent.Current.BusinessId)
}
