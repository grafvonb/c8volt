// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestOpsPurgeAllProcessDefinitionsUnfinishedDiagnosticLogging catches direct stderr
// bypasses, duplicate errors, raw HTTP disclosure and dropped partial results.
func TestOpsPurgeAllProcessDefinitionsUnfinishedDiagnosticLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for i, tc := range []struct {
			name                                                   string
			flags                                                  []string
			configDebug, debug, machine, quiet, partial, preflight bool
		}{
			{name: "normal"}, {name: "verbose", flags: []string{"--verbose"}}, {name: "quiet", flags: []string{"--quiet"}, quiet: true},
			{name: "debug", flags: []string{"--debug"}, debug: true}, {name: "configured debug", configDebug: true, debug: true},
			{name: "quiet overrides debug", flags: []string{"--quiet", "--debug"}, configDebug: true, quiet: true},
			{name: "JSON", flags: []string{"--json"}, machine: true}, {name: "partial JSON", flags: []string{"--json", "--quiet"}, machine: true, quiet: true, partial: true},
			{name: "partial verbose", flags: []string{"--verbose"}, partial: true}, {name: "preflight", preflight: true},
		} {
			t.Run(format+"/"+tc.name, func(t *testing.T) {
				var mutations atomic.Int32
				srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/v2/process-definitions/search" {
						fmt.Fprintf(w, `{"items":[%s,%s,%s],"page":{"totalItems":3}}`, opsAllProcessDefinitionsPurgeDefinitionJSON("2251799813685123", "invoice", 1), opsAllProcessDefinitionsPurgeDefinitionJSON("2251799813685456", "invoice", 2), opsAllProcessDefinitionsPurgeDefinitionJSON("2251799813685789", "invoice", 3))
						return
					}
					if r.URL.Path == "/v2/process-instances/search" && !tc.preflight {
						io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
						return
					}
					if strings.HasPrefix(r.URL.Path, "/v2/process-definitions/") {
						io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON(strings.TrimPrefix(r.URL.Path, "/v2/process-definitions/"), "invoice", 1))
						return
					}
					if strings.HasPrefix(r.URL.Path, "/v2/resources/") && strings.HasSuffix(r.URL.Path, "/deletion") {
						mutations.Add(1)
						if tc.partial && strings.Contains(r.URL.Path, "/2251799813685123/") {
							io.WriteString(w, `{"resourceKey":"2251799813685123","batchOperation":null}`)
							return
						}
					} else if !tc.preflight {
						t.Errorf("unexpected request %s", r.URL.Path)
					}
					w.WriteHeader(400)
					io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret resource-secret-2251799813685456 token=password"}`)
				}))
				t.Cleanup(srv.Close)
				args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, format, tc.configDebug), "ops", "purge", "all-process-definitions", "--no-wait", "--auto-confirm", "--workers", "1"}
				args = append(args, tc.flags...)
				destination := "inherited"
				if i%2 == 1 {
					destination = "configured"
				}
				stdout, stderr, err := runDefinitionDeleteTerminal(t, args, destination, nil)
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				require.Equal(t, 2, exitErr.ExitCode())
				records := bulkCompletionDiagnosticRecords(t, stderr, format)
				errors, debugs := []string{}, []string{}
				for _, record := range records {
					switch record[0] {
					case "ERROR":
						errors = append(errors, record[1])
					case "DEBUG":
						debugs = append(debugs, record[1])
					case "INFO":
						require.False(t, tc.quiet)
					case "WARN":
						require.False(t, tc.quiet)
					default:
						t.Errorf("unexpected severity %s", record[0])
					}
				}
				if tc.machine {
					require.Empty(t, errors)
					decoder := json.NewDecoder(strings.NewReader(stdout))
					var envelope map[string]any
					require.NoError(t, decoder.Decode(&envelope))
					require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
					require.Equal(t, "invalid_input", envelope["class"])
					require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "body-only-secret")
					payload := requireJSONObject(t, envelope["payload"])
					deletion := requireJSONObject(t, payload["deletion"])
					items, _ := deletion["items"].([]any)
					if tc.preflight {
						require.Empty(t, items)
					} else {
						require.Len(t, items, 3)
					}
					ok := 0
					for _, item := range items {
						if requireJSONObject(t, item)["ok"] == true {
							ok++
						}
					}
					want := 0
					if tc.partial {
						want = 1
					}
					require.Equal(t, want, ok)
				} else {
					require.Empty(t, stdout)
					require.Len(t, errors, 1)
					stage := "process-definition purge"
					success := 0
					if tc.partial {
						success = 1
					}
					if tc.preflight {
						stage = "process-definition purge"
					}
					require.Equal(t, fmt.Sprintf("%s: selected 3, accepted %d; outcome unconfirmed; invalid input", stage, success), errors[0])
				}
				if tc.preflight {
					require.Zero(t, mutations.Load())
				} else {
					require.Equal(t, int32(3), mutations.Load())
				}
				for _, bad := range []string{"body-only-secret", "private.invalid", "resource-secret-2251799813685456", "token=password"} {
					require.NotContains(t, stderr, bad)
				}
				if tc.debug {
					require.NotEmpty(t, debugs)
					require.Contains(t, strings.Join(debugs, "\n"), "POST /v2/resources/")
					require.Contains(t, strings.Join(debugs, "\n"), "status=400")
				} else {
					require.Empty(t, debugs)
					for _, bad := range []string{srv.URL, "POST", "/v2/resources/"} {
						require.NotContains(t, stderr, bad)
					}
				}
			})
		}
	}
}

// TestOpsPurgeAllProcessDefinitionsSuspendedForceExecution exercises the full CLI,
// facade and v810 adapters against an isolated sparse union discovery fixture.
func TestOpsPurgeAllProcessDefinitionsSuspendedForceExecution(t *testing.T) {
	for _, scenario := range []string{"accepted", "preview", "cancel failure", "drain failure", "history failure", "additional work failure", "confirm", "abort"} {
		t.Run(scenario, func(t *testing.T) {
			var canceled, history, definitions, verifications atomic.Int32
			var events testx.SafeSlice[string]
			srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				state := "SUSPENDED"
				if canceled.Load() > 0 {
					state = "TERMINATED"
				}
				row := opsAllProcessDefinitionsPurgeProcessInstanceJSON("2251799813685321", "2251799813685321", "2251799813685123", state)
				switch r.URL.Path {
				case "/v2/process-definitions/2251799813685123":
					io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON("2251799813685123", "invoice", 1))
				case "/v2/process-instances/2251799813685321":
					io.WriteString(w, row)
				case "/v2/process-instances/search":
					var body struct {
						Filter map[string]any `json:"filter"`
						Page   struct {
							Limit int    `json:"limit"`
							After string `json:"after"`
							From  int    `json:"from"`
						} `json:"page"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					if _, ok := body.Filter["parentProcessInstanceKey"]; ok {
						io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
						return
					}
					require.Equal(t, "2251799813685123", body.Filter["processDefinitionKey"])
					if body.Page.Limit == 1 {
						count := 0
						if union, ok := body.Filter["state"].(map[string]any); ok {
							require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, union["$in"])
							if canceled.Load() > 0 {
								events.Append("verify union")
								attempt := verifications.Add(1)
								if scenario == "additional work failure" && attempt == 1 {
									io.WriteString(w, `{"items":[],"page":{"totalItems":1}}`)
									return
								}
								if scenario == "drain failure" || scenario == "additional work failure" {
									w.WriteHeader(400)
									io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
									return
								}
							} else {
								count = 1
							}
						} else if body.Filter["state"] == "SUSPENDED" && canceled.Load() == 0 {
							count = 1
						}
						fmt.Fprintf(w, `{"items":[],"page":{"totalItems":%d}}`, count)
						return
					}
					union, ok := body.Filter["state"].(map[string]any)
					require.True(t, ok)
					require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, union["$in"])
					if body.Page.After == "" {
						io.WriteString(w, `{"items":[],"page":{"totalItems":1,"endCursor":"sparse"}}`)
						return
					}
					require.Equal(t, "sparse", body.Page.After)
					require.Zero(t, body.Page.From)
					fmt.Fprintf(w, `{"items":[%s],"page":{"totalItems":1}}`, row)
				case "/v2/process-instances/2251799813685321/cancellation":
					events.Append("cancel")
					canceled.Add(1)
					if scenario == "cancel failure" {
						w.WriteHeader(400)
						io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
						return
					}
					w.WriteHeader(202)
					io.WriteString(w, `{}`)
				case "/v2/process-instances/2251799813685321/deletion":
					events.Append("history")
					history.Add(1)
					if scenario == "history failure" {
						w.WriteHeader(400)
						io.WriteString(w, `{"detail":"POST https://private.invalid/secret body-only-secret"}`)
						return
					}
					w.WriteHeader(200)
					io.WriteString(w, `{}`)
				case "/v2/resources/2251799813685123/deletion":
					events.Append("definition")
					definitions.Add(1)
					io.WriteString(w, `{"resourceKey":"2251799813685123","batchOperation":null}`)
				default:
					t.Errorf("unexpected request (including resume or child cancellation): %s %s", r.Method, r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			t.Cleanup(srv.Close)
			args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, "plain-time", false), "ops", "purge", "all-process-definitions", "--key", "2251799813685123", "--force", "--no-wait", "--json"}
			if scenario == "preview" {
				args = append(args, "--dry-run")
			}
			var exchanges []testx.CmdTerminalExchange
			prompt := "process-definition purge: 1 candidate process definition(s), 1 affected process instance(s) will be deleted.\nDo you want to proceed?"
			if scenario == "confirm" || scenario == "abort" {
				answer := "y"
				if scenario == "abort" {
					answer = "n"
				}
				exchanges = []testx.CmdTerminalExchange{{Prompt: prompt, Response: answer}}
			} else {
				args = append(args, "--auto-confirm")
			}
			stdout, stderr, err := runDefinitionDeleteTerminal(t, args, "configured", exchanges)
			if len(exchanges) > 0 {
				require.Equal(t, 1, strings.Count(stderr, prompt))
				require.NotContains(t, stdout, prompt)
			}
			if strings.HasSuffix(scenario, "failure") || scenario == "abort" {
				require.Error(t, err)
				require.Zero(t, definitions.Load())
			} else {
				require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
			}
			decoder := json.NewDecoder(strings.NewReader(stdout))
			var envelope map[string]any
			require.NoError(t, decoder.Decode(&envelope))
			require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
			require.NotContains(t, stderr, "body-only-secret")
			require.NotContains(t, stderr, "private.invalid")
			require.NotContains(t, stderr, "POST")
			if scenario == "abort" {
				require.Zero(t, canceled.Load())
				require.Zero(t, history.Load())
				require.Zero(t, definitions.Load())
				require.Equal(t, "confirmation_failed", requireJSONObject(t, requireJSONObject(t, envelope["payload"])["deletion"])["status"])
				return
			}
			if scenario == "preview" {
				require.Equal(t, "succeeded", envelope["outcome"])
				require.Zero(t, canceled.Load())
				require.Zero(t, history.Load())
				require.Zero(t, definitions.Load())
				payload := requireJSONObject(t, envelope["payload"])
				plan := requireJSONObject(t, payload["deletePlan"])
				item := requireJSONObject(t, plan["items"].([]any)[0])
				require.Equal(t, []any{"2251799813685321"}, item["unfinishedProcessInstanceKeys"])
				require.NotContains(t, item, "activeProcessInstanceKeys")
			} else {
				require.Equal(t, int32(1), canceled.Load())
				switch scenario {
				case "accepted", "confirm":
					require.Equal(t, "succeeded", envelope["outcome"])
					require.Equal(t, []string{"cancel", "verify union", "history", "definition"}, events.Snapshot())
				case "cancel failure", "drain failure", "additional work failure":
					require.Zero(t, history.Load())
				case "history failure":
					require.Equal(t, int32(1), history.Load())
				}
			}
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsUnfinishedCommand verifies blockers and previews
// through actual v810 union requests, retaining mode precedence and zero mutations.
func TestOpsPurgeAllProcessDefinitionsUnfinishedCommand(t *testing.T) {
	for _, population := range []struct {
		name                     string
		active, suspended, union int
	}{
		{"suspended", 0, 3, 3}, {"mixed", 2, 3, 5}, {"transition", 0, 0, 1},
	} {
		for _, flags := range [][]string{nil, {"--auto-confirm"}, {"--auto-confirm", "--json"}, {"--automation", "--json"}, {"--json"}, {"--quiet"}, {"--verbose"}, {"--dry-run"}, {"--dry-run", "--json"}, {"--dry-run", "--quiet", "--json"}} {
			t.Run(population.name+"/"+strings.Join(flags, " "), func(t *testing.T) {
				var reads, mutations atomic.Int32
				srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					switch {
					case r.URL.Path == "/v2/process-definitions/2251799813685123":
						io.WriteString(w, opsAllProcessDefinitionsPurgeDefinitionJSON("2251799813685123", "invoice", 1))
					case r.URL.Path == "/v2/process-instances/search":
						reads.Add(1)
						var body struct {
							Filter map[string]any `json:"filter"`
						}
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, "2251799813685123", body.Filter["processDefinitionKey"])
						count := 0
						switch state := body.Filter["state"].(type) {
						case string:
							if state == "ACTIVE" {
								count = population.active
							}
							if state == "SUSPENDED" {
								count = population.suspended
							}
						case map[string]any:
							require.Equal(t, []any{"ACTIVE", "SUSPENDED"}, state["$in"])
							count = population.union
						}
						fmt.Fprintf(w, `{"items":[],"page":{"totalItems":%d,"hasMoreTotalItems":false}}`, count)
					default:
						mutations.Add(1)
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
						w.WriteHeader(500)
					}
				}))
				t.Cleanup(srv.Close)
				args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, "plain-time", false), "ops", "purge", "all-process-definitions", "--key", "2251799813685123", "--no-wait"}
				args = append(args, flags...)
				extension := ".md"
				if strings.Contains(strings.Join(flags, " "), "--json") {
					extension = ".json"
				}
				reportPath := filepath.Join(t.TempDir(), "purge"+extension)
				args = append(args, "--report-file", reportPath)
				stdout, stderr, err := runDefinitionDeleteTerminal(t, args, "configured", nil)
				dry := strings.Contains(strings.Join(flags, " "), "--dry-run")
				machine := strings.Contains(strings.Join(flags, " "), "--json")
				if dry {
					require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
				} else {
					var exitErr *exec.ExitError
					require.ErrorAs(t, err, &exitErr)
					require.Equal(t, 1, exitErr.ExitCode())
				}
				report := readReportFile(t, reportPath)
				if extension == ".json" {
					var data map[string]any
					require.NoError(t, json.Unmarshal([]byte(report), &data))
					require.Equal(t, "ops.all-process-definitions.v1", data["schemaVersion"])
					plan := requireJSONObject(t, data["deletePlan"])
					require.Equal(t, float64(population.union), plan["unfinishedProcessInstanceCount"])
					require.Equal(t, true, plan["suspendedCountSupported"])
				} else {
					for label, count := range map[string]int{"Active": population.active, "Suspended": population.suspended, "Unfinished": population.union, "Affected": population.union} {
						require.Contains(t, report, fmt.Sprintf("- %s Process Instances: %d\n", label, count))
					}
					if dry && !strings.Contains(strings.Join(flags, " "), "--quiet") {
						require.Contains(t, stderr, fmt.Sprintf("unfinished workload: %d unfinished process instance(s) (active %d, suspended %d)", population.union, population.active, population.suspended))
					}
				}
				require.Zero(t, mutations.Load())
				require.Equal(t, int32(6), reads.Load(), "stdout=%s stderr=%s", stdout, stderr)
				require.NotContains(t, stderr, "Do you want to proceed?")
				records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
				if machine {
					decoder := json.NewDecoder(strings.NewReader(stdout))
					var envelope map[string]any
					require.NoError(t, decoder.Decode(&envelope))
					require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
					if dry {
						require.Equal(t, "succeeded", envelope["outcome"])
						payload := requireJSONObject(t, envelope["payload"])
						totals := requireJSONObject(t, payload["deletePlan"])
						require.Equal(t, float64(population.union), totals["unfinishedProcessInstanceCount"])
						if population.active == 0 {
							require.NotContains(t, totals, "activeProcessInstanceCount")
						} else {
							require.Equal(t, float64(population.active), totals["activeProcessInstanceCount"])
						}
						require.Equal(t, true, totals["suspendedCountSupported"])
						require.NotContains(t, requireJSONObject(t, payload["deletion"]), "submitted")
					} else {
						require.Equal(t, "local_precondition", envelope["class"])
					}
					require.Empty(t, records)
				} else {
					require.Empty(t, stdout)
					errors := []string{}
					for _, record := range records {
						if record[0] == "ERROR" {
							errors = append(errors, record[1])
						}
					}
					if dry {
						require.Empty(t, errors)
					} else {
						require.Len(t, errors, 1)
						require.Contains(t, errors[0], fmt.Sprintf("%d unfinished process instance(s) (active %d, suspended %d) require --force", population.union, population.active, population.suspended))
					}
					if strings.Contains(strings.Join(flags, " "), "--quiet") {
						require.Len(t, records, 1)
					}
				}
			})
		}
	}
}

// TestOpsPurgeAllProcessDefinitionsEmptyTerminal verifies prompt-free completion
// with real terminal stdin, one discovery request, and exact empty mode contracts.
func TestOpsPurgeAllProcessDefinitionsEmptyTerminal(t *testing.T) {
	for _, dry := range []bool{false, true} {
		for _, flags := range [][]string{nil, {"--json"}, {"--keys-only"}, {"--quiet"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}, {"--auto-confirm", "--no-wait"}, {"--automation", "--json", "--no-wait"}} {
			t.Run(fmt.Sprint(dry)+strings.Join(flags, " "), func(t *testing.T) {
				var reads atomic.Int32
				srv := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
					reads.Add(1)
					w.Header().Set("Content-Type", "application/json")
					io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
				}))
				t.Cleanup(srv.Close)
				args := []string{"--config", definitionDeleteLogConfig(t, srv.URL, "plain-time", false), "ops", "purge", "all-process-definitions", "--force"}
				args = append(args, flags...)
				if dry {
					args = append(args, "--dry-run")
				}
				stdout, stderr, err := runDefinitionDeleteTerminal(t, args, "inherited", nil)
				require.NoError(t, err, stdout+stderr)
				require.EqualValues(t, 1, reads.Load())
				require.NotContains(t, stderr, "Do you want to proceed?")
				joined := strings.Join(flags, " ")
				if strings.Contains(joined, "--json") {
					dec := json.NewDecoder(strings.NewReader(stdout))
					var envelope map[string]any
					require.NoError(t, dec.Decode(&envelope))
					require.ErrorIs(t, dec.Decode(new(any)), io.EOF)
					require.Equal(t, "succeeded", envelope["outcome"])
					payload := requireJSONObject(t, envelope["payload"])
					require.Equal(t, "planned", payload["outcome"])
					require.NotContains(t, requireJSONObject(t, payload["deletion"]), "submitted")
					require.NotContains(t, requireJSONObject(t, payload["deletePlan"]), "suspendedCountSupported")
				} else {
					require.Empty(t, stdout)
				}
				records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
				if strings.Contains(joined, "--json") || strings.Contains(joined, "--quiet") {
					require.Empty(t, records)
					return
				}
				var messages []string
				for _, record := range records {
					require.Contains(t, []string{"INFO", "WARN"}, record[0])
					messages = append(messages, record[1])
				}
				header := "purge all process definitions"
				plan := "delete plan: skipped"
				if dry {
					header = "dry run: " + header
					plan = "delete preview: skipped (no matching process definitions)"
				}
				// Duration is measured at execution; all other empty-result wording is exact.
				for i, msg := range messages {
					if strings.HasPrefix(msg, "outcome:") {
						messages[i] = strings.Split(msg, "; elapsed:")[0]
					}
				}
				want := []string{}
				if !strings.Contains(joined, "--keys-only") {
					work := "process-definition purge: process-definition purge dry run will discover matching process definitions and validate delete impact only; no changes will be applied"
					if !dry && strings.Contains(joined, "--auto-confirm") {
						work = "process-definition purge is destructive: process-definition purge will discover matching process definitions, validate delete impact, and delete confirmed definitions"
					}
					want = append(want, "selection scope: unfiltered across accessible tenants", "process-definition purge scope: all-process-definitions purge matched an unknown number of process definitions; page size: 1000", work)
				}
				want = append(want, header, "selection filters: none", "candidate process definitions: 0", plan, "outcome: planned; no changes applied")
				require.Equal(t, want, messages)
			})
		}
	}
}

// TestOpsPurgeAllProcessDefinitionsUnfinishedReportKeys retains suspended keys
// without expanded family evidence and deduplicates overlapping family keys.
func TestOpsPurgeAllProcessDefinitionsUnfinishedReportKeys(t *testing.T) {
	plan := ops.AllProcessDefinitionsPurgeDeletePlan{RequiresForce: true, Items: []resource.DeleteProcessDefinitionPlanItem{
		{Key: "pd-a", UnfinishedProcessInstanceCount: 2, UnfinishedProcessInstanceKeys: []string{"active", "suspended"}, ActiveProcessInstanceKeys: []string{"active"}},
		{Key: "pd-b", UnfinishedProcessInstanceCount: 1, UnfinishedProcessInstanceKeys: []string{"child"}, CancellationPlan: process.DryRunPIKeyExpansion{Collected: typex.Keys{"root", "child", "suspended"}}},
	}}
	require.ElementsMatch(t, []string{"active", "suspended", "child", "root"}, allProcessDefinitionsPurgeAffectedProcessInstanceKeys(plan))
	require.ElementsMatch(t, []string{"active", "suspended", "child"}, allProcessDefinitionsPurgeBlockedProcessInstanceKeys(plan))
	require.Equal(t, map[string]int64{"pd-a": 2, "pd-b": 3}, allProcessDefinitionsPurgeAffectedCountByProcessDefinitionKey(plan))
	plan.RequiresForce = false
	require.Empty(t, allProcessDefinitionsPurgeBlockedProcessInstanceKeys(plan))
}
