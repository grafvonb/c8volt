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
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestNewGetUserTaskSearchRequest_MapsVariableFiltersAndPropagatesErrors pins
// construction independently of Cobra execution and prevents parser failures
// from being silently dropped by future call sites.
func TestNewGetUserTaskSearchRequest_MapsVariableFiltersAndPropagatesErrors(t *testing.T) {
	previousPIKey, previousAssignee := flagGetUserTaskPIKey, flagGetUserTaskAssignee
	previousBatchSize, previousLimit := flagGetUserTaskBatchSize, flagGetUserTaskLimit
	previousExists, previousVars, previousLikes := flagGetUserTaskVarExists, flagGetUserTaskVars, flagGetUserTaskVarLikes
	t.Cleanup(func() {
		flagGetUserTaskPIKey, flagGetUserTaskAssignee = previousPIKey, previousAssignee
		flagGetUserTaskBatchSize, flagGetUserTaskLimit = previousBatchSize, previousLimit
		flagGetUserTaskVarExists, flagGetUserTaskVars, flagGetUserTaskVarLikes = previousExists, previousVars, previousLikes
	})

	flagGetUserTaskPIKey = " 2251799813711967 "
	flagGetUserTaskAssignee = " alice "
	flagGetUserTaskBatchSize = 25
	flagGetUserTaskLimit = 10
	flagGetUserTaskVarExists = []string{"payload"}
	flagGetUserTaskVars = []string{`status="approved"`}
	flagGetUserTaskVarLikes = []string{"email=*@example.com"}

	request, err := newGetUserTaskSearchRequest()
	require.NoError(t, err)
	require.Equal(t, "2251799813711967", request.ProcessInstanceKey)
	require.Equal(t, "alice", request.Assignee)
	require.Equal(t, int32(25), request.BatchSize)
	require.Equal(t, int32(10), request.Limit)
	require.Len(t, request.VariableFilters.Clauses, 3)
	require.Equal(t, []string{"payload", "status", "email"}, []string{
		request.VariableFilters.Clauses[0].Name,
		request.VariableFilters.Clauses[1].Name,
		request.VariableFilters.Clauses[2].Name,
	})

	flagGetUserTaskVars = []string{"status"}
	request, err = newGetUserTaskSearchRequest()
	require.ErrorContains(t, err, "must use name=value syntax")
	require.Empty(t, request)
}

// TestGetUserTaskCommand_SearchBuildsAllPredicates verifies normalized state,
// exact string predicates, selector keys, tenant scope, bounds, and AND composition.
func TestGetUserTaskCommand_SearchBuildsAllPredicates(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(1, false, "", "2251799815391233")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "tenant-a", "--json", "get", "ut",
		"--pi-key", "2251799813711967", "--pd-key", "2251799813689000",
		"--bpmn-process-id", "invoice", "--element-id", "approve_invoice",
		"--state", "created", "--assignee", "AliceCase", "--candidate-user", "BobCase",
		"--candidate-group", "AccountingCase", "--batch-size", "2", "--limit", "2")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Contains(t, stdout, `"total": 1`)

	got := requests.Snapshot()
	require.Len(t, got, 1)
	filter := requireJSONMap(t, got[0]["filter"])
	require.Equal(t, "2251799813711967", filter["processInstanceKey"])
	require.Equal(t, "2251799813689000", filter["processDefinitionKey"])
	require.Equal(t, "invoice", filter["processDefinitionId"])
	require.Equal(t, "approve_invoice", filter["elementId"])
	require.Equal(t, "CREATED", jsonFilterValue(t, filter["state"]))
	require.Equal(t, "AliceCase", jsonFilterValue(t, filter["assignee"]))
	require.Equal(t, "BobCase", jsonFilterValue(t, filter["candidateUser"]))
	require.Equal(t, "AccountingCase", jsonFilterValue(t, filter["candidateGroup"]))
	require.Equal(t, "tenant-a", jsonFilterValue(t, filter["tenantId"]))
	require.Equal(t, float64(2), jsonPageLimit(t, got[0]))
}

// TestGetUserTaskCommand_SearchDefaultsAndStates verifies empty implicit stdin
// selects discovery, all omits the state predicate, and all nine states normalize.
func TestGetUserTaskCommand_SearchDefaultsAndStates(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "found: 0\n", stdout)
	filter := requireJSONMap(t, requests.Snapshot()[0]["filter"])
	require.NotContains(t, filter, "state")

	stdout, stderr, err = runGetUserTaskCommand(t, configPath, "", "get", "ut", "--state", "ALL")
	require.NoError(t, err, stderr)
	require.Equal(t, "found: 0\n", stdout)
	filter = requireJSONMap(t, requests.Snapshot()[1]["filter"])
	require.NotContains(t, filter, "state")

	states := []string{"ASSIGNING", "CANCELED", "CANCELING", "COMPLETED", "COMPLETING", "CREATED", "CREATING", "FAILED", "UPDATING"}
	for index, state := range states {
		_, stderr, err = runGetUserTaskCommand(t, configPath, "", "get", "ut", "--state", strings.ToLower(state))
		require.NoError(t, err, stderr)
		filter = requireJSONMap(t, requests.Snapshot()[index+2]["filter"])
		require.Equal(t, state, jsonFilterValue(t, filter["state"]))
	}
}

// TestGetUserTaskCommand_SearchEmptyModes verifies completed empty discovery
// renders once without synthetic rows or extra requests in every output mode.
func TestGetUserTaskCommand_SearchEmptyModes(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "human", args: []string{"get", "ut"}, want: "found: 0\n"},
		{name: "keys", args: []string{"--keys-only", "get", "ut"}, want: ""},
		{name: "quiet human", args: []string{"--quiet", "get", "ut"}, want: ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := len(requests.Snapshot())
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, test.want, stdout)
			require.Len(t, requests.Snapshot(), before+1)
		})
	}

	before := len(requests.Snapshot())
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	var envelope struct {
		Outcome string `json:"outcome"`
		Payload struct {
			Total int64 `json:"total"`
			Items []any `json:"items"`
		} `json:"payload"`
	}
	decoder := json.NewDecoder(strings.NewReader(stdout))
	require.NoError(t, decoder.Decode(&envelope))
	require.Equal(t, "succeeded", envelope.Outcome)
	require.Zero(t, envelope.Payload.Total)
	require.NotNil(t, envelope.Payload.Items)
	require.Empty(t, envelope.Payload.Items)
	var extra any
	require.Error(t, decoder.Decode(&extra), "JSON output must contain exactly one envelope")
	require.Len(t, requests.Snapshot(), before+1)
}

// TestGetUserTaskCommand_SearchRejectsInvalidInputBeforeRequests verifies state,
// selector, bound, and total-mode conflicts fail before backend discovery.
func TestGetUserTaskCommand_SearchRejectsInvalidInputBeforeRequests(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "assigned", args: []string{"get", "ut", "--state", "assigned"}, want: "invalid value for --state"},
		{name: "unknown state", args: []string{"get", "ut", "--state", "waiting"}, want: "invalid value for --state"},
		{name: "bad pi key", args: []string{"get", "ut", "--pi-key", "123"}, want: "--pi-key value"},
		{name: "bad pd key", args: []string{"get", "ut", "--pd-key", "123"}, want: "--pd-key value"},
		{name: "zero batch", args: []string{"get", "ut", "--batch-size", "0"}, want: "invalid value for --batch-size"},
		{name: "zero limit", args: []string{"get", "ut", "--limit", "0"}, want: "--limit must be positive"},
		{name: "total limit", args: []string{"get", "ut", "--total", "--limit", "1"}, want: "--total cannot be combined with --limit"},
		{name: "total json", args: []string{"--json", "get", "ut", "--total"}, want: "--total cannot be combined with --json"},
		{name: "total keys", args: []string{"--keys-only", "get", "ut", "--total"}, want: "--total cannot be combined with --keys-only"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := len(requests.Snapshot())
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, test.want)
			require.Len(t, requests.Snapshot(), before)
		})
	}
}

// TestGetUserTaskCommand_SearchTraversesSparsePagesAndHonorsLimit verifies exact
// totals across cursor pages, sparse intermediate pages, and nonfinal limits on every adapter.
func TestGetUserTaskCommand_SearchTraversesSparsePagesAndHonorsLimit(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, sparse := range []bool{false, true} {
			for _, limited := range []bool{false, true} {
				for _, jsonOutput := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/sparse=%t/limited=%t/json=%t", version, sparse, limited, jsonOutput), func(t *testing.T) {
						responses := []string{
							userTaskSearchResponse(3, false, "cursor-a", "2251799815391233"),
							userTaskSearchResponse(3, false, "cursor-b", "2251799815391234"),
							userTaskSearchResponse(3, false, "cursor-c", "2251799815391235"),
						}
						if sparse {
							responses = append(responses[:1], append([]string{userTaskSearchResponse(3, false, "cursor-sparse")}, responses[1:]...)...)
						}
						server, requests := newGetUserTaskSearchServer(t, func(index int, _ map[string]any) string {
							require.Less(t, index, len(responses))
							return responses[index]
						})
						configPath := testx.WriteTestConfigForVersion(t, server.URL, version)
						args := []string{"--keys-only", "get", "ut", "--batch-size", "1"}
						if jsonOutput {
							args[0] = "--json"
						}
						wantCount := 3
						wantRequests := len(responses)
						if limited {
							args = append(args, "--limit", "2")
							wantCount = 2
							wantRequests--
						}
						stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", args...)
						require.NoError(t, err, stderr)
						require.Empty(t, stderr)
						if jsonOutput {
							requireSucceededUserTaskEnvelope(t, stdout, int64(wantCount))
						} else {
							want := "2251799815391233\n2251799815391234\n"
							if !limited {
								want += "2251799815391235\n"
							}
							require.Equal(t, want, stdout)
						}
						got := requests.Snapshot()
						require.Len(t, got, wantRequests)
						for i := 1; i < len(got); i++ {
							var previous struct {
								Page struct {
									EndCursor string `json:"endCursor"`
								} `json:"page"`
							}
							require.NoError(t, json.Unmarshal([]byte(responses[i-1]), &previous))
							require.Equal(t, previous.Page.EndCursor, jsonPageAfter(t, got[i]))
						}
					})
				}
			}
		}
	}
}

// TestGetUserTaskCommand_SearchVariablesEnrichOnlySelectedTasks verifies search
// filters and limits select tasks before complete per-task variable retrieval in
// both incremental human and collected JSON execution.
func TestGetUserTaskCommand_SearchVariablesEnrichOnlySelectedTasks(t *testing.T) {
	const (
		firstKey    = "2251799815391233"
		secondKey   = "2251799815391234"
		excludedKey = "2251799815391235"
	)
	variable := func(name, value string) userTaskVariableFixtureValue {
		return userTaskVariableFixtureValue{
			Name: name, Value: value, VariableKey: "901", ProcessInstanceKey: "2251799813711967",
			ScopeKey: "2251799815391200", TenantID: "tenant-a",
		}
	}
	for _, test := range []struct {
		name       string
		args       []string
		wantSearch int
		wantTotal  int64
	}{
		{name: "incremental partial page limit", args: []string{"--tenant", "tenant-a", "get", "ut", "--assignee", "alice", "--batch-size", "3", "--limit", "2", "--with-vars"}, wantSearch: 1, wantTotal: 2},
		{name: "collected json partial page limit", args: []string{"--tenant", "tenant-a", "--json", "get", "ut", "--assignee", "alice", "--batch-size", "3", "--limit", "2", "--with-vars"}, wantSearch: 1, wantTotal: 2},
		{name: "sparse incremental pages", args: []string{"get", "ut", "--batch-size", "1", "--with-vars"}, wantSearch: 2, wantTotal: 1},
		{name: "sparse collected pages", args: []string{"--json", "get", "ut", "--batch-size", "1", "--with-vars"}, wantSearch: 2, wantTotal: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskVariablesServer(t, getUserTaskVariablesFixture{
				SearchRespond: func(index int, _ map[string]any) string {
					if strings.Contains(test.name, "sparse") {
						if index == 0 {
							return userTaskSearchResponse(1, false, "cursor-sparse")
						}
						return userTaskSearchResponse(1, false, "", firstKey)
					}
					return userTaskSearchResponse(3, false, "cursor-unused", firstKey, secondKey, excludedKey)
				},
				VariablePages: map[string][]userTaskVariablePageFixture{
					firstKey: {
						{Total: 2, Items: []userTaskVariableFixtureValue{variable("alpha", "1")}},
						{Total: 2, Items: []userTaskVariableFixtureValue{variable("omega", "2")}},
					},
					secondKey:   {{Total: 1, Items: []userTaskVariableFixtureValue{variable("beta", "3")}}},
					excludedKey: {{Total: 1, Items: []userTaskVariableFixtureValue{variable("excluded", "4")}}},
				},
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			if strings.Contains(test.name, "json") || strings.Contains(test.name, "collected") {
				requireSucceededUserTaskEnvelope(t, stdout, test.wantTotal)
				require.Contains(t, stdout, `"variables"`)
			} else {
				require.Contains(t, stdout, "alpha=1")
				require.Contains(t, stdout, "omega=2")
				if test.wantTotal == 2 {
					require.Contains(t, stdout, "beta=3")
				} else {
					require.NotContains(t, stdout, "beta=3")
				}
				require.Equal(t, 1, strings.Count(stdout, fmt.Sprintf("found: %d\n", test.wantTotal)), "final summary must not re-enrich or rerender tasks")
			}

			_, searches, _ := requests.snapshot()
			require.Len(t, searches, test.wantSearch)
			if !strings.Contains(test.name, "sparse") {
				filter := requireJSONMap(t, searches[0]["filter"])
				require.Equal(t, "alice", jsonFilterValue(t, filter["assignee"]))
				require.Equal(t, "tenant-a", jsonFilterValue(t, filter["tenantId"]))
			}
			require.Equal(t, 2, requests.variableRequestCount(firstKey), "variable pagination must complete independently of the task limit")
			if test.wantTotal == 2 {
				require.Equal(t, 1, requests.variableRequestCount(secondKey))
			} else {
				require.Zero(t, requests.variableRequestCount(secondKey))
			}
			require.Zero(t, requests.variableRequestCount(excludedKey), "trimmed tasks must never be enriched")
		})
	}
}

// TestGetUserTaskCommand_FilteredDisplayPreservesSelection verifies native
// local filtering selects independently of effective-variable display while
// limits and sparse pages bound enrichment to the tasks actually rendered.
func TestGetUserTaskCommand_FilteredDisplayPreservesSelection(t *testing.T) {
	t.Run("same selected identities with and without display", func(t *testing.T) {
		for _, withVariables := range []bool{false, true} {
			t.Run(fmt.Sprintf("with-vars=%t", withVariables), func(t *testing.T) {
				server, requests := newGetUserTaskVariablesServer(t, filteredUserTaskVariablesFixture())
				configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
				args := []string{"get", "ut", "--var", `status="approved"`}
				if withVariables {
					args = append(args, "--with-vars")
				}

				stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", args...)
				require.NoError(t, err, stderr)
				require.Empty(t, stderr)
				require.Equal(t, 1, strings.Count(stdout, userTaskLocalMatchKey))
				require.Equal(t, 1, strings.Count(stdout, userTaskShadowedMatchKey))
				require.NotContains(t, stdout, userTaskParentOnlyKey)
				require.Equal(t, 1, strings.Count(stdout, "found: 2\n"), "incremental rendering must emit one final summary")
				require.Equal(t, 1, requests.searchRequestCount())
				require.Equal(t, [][]any{{
					map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}},
				}}, requests.localVariablePredicates())
				if withVariables {
					require.Equal(t, 1, requests.variableRequestCount(userTaskLocalMatchKey))
					require.Equal(t, 1, requests.variableRequestCount(userTaskShadowedMatchKey))
					require.Contains(t, stdout, `region="eu"`)
					require.Contains(t, stdout, `status="approved-local"`)
				} else {
					require.Zero(t, requests.totalVariableRequestCount(), "the native filter must not trigger effective-variable reads")
				}
				require.Zero(t, requests.variableRequestCount(userTaskParentOnlyKey))
			})
		}
	})

	t.Run("within-page limit excludes enrichment", func(t *testing.T) {
		fixture := filteredUserTaskVariablesFixture()
		fixture.SearchRespond = func(_ int, _ map[string]any) string {
			return userTaskSearchResponse(3, false, "", userTaskLocalMatchKey, userTaskShadowedMatchKey, userTaskParentOnlyKey)
		}
		server, requests := newGetUserTaskVariablesServer(t, fixture)
		configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

		stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut", "--var", `status="approved"`, "--limit", "1", "--with-vars")
		require.NoError(t, err, stderr)
		require.Empty(t, stderr)
		require.Contains(t, stdout, userTaskLocalMatchKey)
		require.NotContains(t, stdout, userTaskShadowedMatchKey)
		require.NotContains(t, stdout, userTaskParentOnlyKey)
		require.Equal(t, 1, requests.variableRequestCount(userTaskLocalMatchKey))
		require.Zero(t, requests.variableRequestCount(userTaskShadowedMatchKey))
		require.Zero(t, requests.variableRequestCount(userTaskParentOnlyKey))
	})

	t.Run("sparse pages retain the filter and enrich once", func(t *testing.T) {
		fixture := filteredUserTaskVariablesFixture()
		fixture.SearchRespond = func(index int, _ map[string]any) string {
			if index == 0 {
				return userTaskSearchResponse(1, false, "cursor-sparse")
			}
			return userTaskSearchResponse(1, false, "", userTaskLocalMatchKey)
		}
		server, requests := newGetUserTaskVariablesServer(t, fixture)
		configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

		stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut", "--var", `status="approved"`, "--batch-size", "1", "--with-vars", "--auto-confirm")
		require.NoError(t, err, stderr)
		require.Empty(t, stderr)
		require.Equal(t, 1, strings.Count(stdout, userTaskLocalMatchKey))
		require.Equal(t, 2, requests.searchRequestCount())
		require.Equal(t, 1, requests.variableRequestCount(userTaskLocalMatchKey))
		require.Zero(t, requests.variableRequestCount(userTaskShadowedMatchKey))
		require.Zero(t, requests.variableRequestCount(userTaskParentOnlyKey))
		require.Equal(t, 2, len(requests.localVariablePredicates()))
		for _, clauses := range requests.localVariablePredicates() {
			require.Equal(t, []any{
				map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}},
			}, clauses)
		}
	})
}

// TestGetUserTaskCommand_FilteredDisplayStopsBeforeUnreadPages verifies a
// real terminal decline cannot enrich tasks beyond the accepted filtered page.
func TestGetUserTaskCommand_FilteredDisplayStopsBeforeUnreadPages(t *testing.T) {
	fixture := filteredUserTaskVariablesFixture()
	fixture.SearchRespond = func(index int, _ map[string]any) string {
		if index == 0 {
			return userTaskSearchResponse(2, false, "cursor-next", userTaskLocalMatchKey)
		}
		return userTaskSearchResponse(2, false, "", userTaskShadowedMatchKey)
	}
	server, requests := newGetUserTaskVariablesServer(t, fixture)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	args, err := json.Marshal([]string{
		"--config", configPath, "--tenant", "tenant-a", "get", "ut", "--batch-size", "1",
		"--var", `status="approved"`, "--with-vars",
	})
	require.NoError(t, err)

	result := testx.NewCmdTerminalRunner().Run(t, testx.CmdTerminalRunRequest{
		ScopeTestName: "TestGetUserTaskPagingTerminal",
		Env: map[string]string{
			userTaskTerminalArgsEnv:       string(args),
			userTaskTerminalConfiguredEnv: "",
		},
		Exchanges: []testx.CmdTerminalExchange{{Prompt: userTaskTerminalPrompt(1, 1), Response: "no"}},
		Timeout:   3 * time.Second,
	})
	if !result.Supported {
		t.Skip(result.UnsupportedReason)
	}
	require.NoError(t, result.Err, result.Stderr)
	require.Contains(t, result.Stdout, userTaskLocalMatchKey)
	require.NotContains(t, result.Stdout, userTaskShadowedMatchKey)
	require.NotContains(t, result.Stdout, userTaskParentOnlyKey)
	require.Equal(t, userTaskTerminalPrompt(1, 1), result.Stderr)
	require.Equal(t, 1, requests.searchRequestCount())
	require.Equal(t, 1, requests.variableRequestCount(userTaskLocalMatchKey))
	require.Zero(t, requests.variableRequestCount(userTaskShadowedMatchKey))
	require.Zero(t, requests.variableRequestCount(userTaskParentOnlyKey))
	require.Equal(t, [][]any{{
		map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}},
	}}, requests.localVariablePredicates())
}

// TestGetUserTaskCommand_SearchLimitBoundaries verifies limits before, at, and
// after a backend page boundary stop with the exact required request count.
func TestGetUserTaskCommand_SearchLimitBoundaries(t *testing.T) {
	tests := []struct {
		name         string
		limit        string
		wantKeys     string
		wantRequests int
	}{
		{name: "before", limit: "1", wantKeys: "2251799815391233\n", wantRequests: 1},
		{name: "at", limit: "3", wantKeys: "2251799815391233\n2251799815391234\n2251799815391235\n", wantRequests: 1},
		{name: "after", limit: "5", wantKeys: "2251799815391233\n2251799815391234\n2251799815391235\n2251799815391236\n", wantRequests: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(index int, _ map[string]any) string {
				if index == 0 {
					return userTaskSearchResponse(4, false, "cursor-a", "2251799815391233", "2251799815391234", "2251799815391235")
				}
				return userTaskSearchResponse(4, false, "", "2251799815391236")
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--keys-only", "get", "ut", "--batch-size", "3", "--limit", test.limit)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, test.wantKeys, stdout)
			require.Len(t, requests.Snapshot(), test.wantRequests)
		})
	}
}

// TestGetUserTaskCommand_TotalUsesExactAndCappedTraversal verifies numeric zero,
// exact fast-path totals, capped fallback traversal, and quiet preservation.
func TestGetUserTaskCommand_TotalUsesExactAndCappedTraversal(t *testing.T) {
	tests := []struct {
		name      string
		responses []string
		want      string
	}{
		{name: "zero", responses: []string{userTaskSearchResponse(0, false, "")}, want: "0\n"},
		{name: "exact", responses: []string{userTaskSearchResponse(7, false, "cursor-unused")}, want: "7\n"},
		{name: "capped", responses: []string{
			userTaskSearchResponse(2, true, "cursor-a", "2251799815391233", "2251799815391234"),
			userTaskSearchResponse(3, false, "", "2251799815391235"),
		}, want: "3\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(index int, _ map[string]any) string {
				return test.responses[index]
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--quiet", "get", "ut", "--total", "--batch-size", "2")
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, test.want, stdout)
			require.Len(t, requests.Snapshot(), len(test.responses))
		})
	}
}

// TestGetUserTaskCommand_SearchSupportsEveryNativeVersion verifies command
// dispatch reaches each supported adapter and V87 rejects without a request.
func TestGetUserTaskCommand_SearchSupportsEveryNativeVersion(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(1, false, "", "2251799815391233")
	})
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		configPath := testx.WriteTestConfigForVersion(t, server.URL, version)
		stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "tenant-a", "--keys-only", "get", "ut", "--automation",
			"--pi-key", "2251799813711967", "--pd-key", "2251799813689000",
			"--bpmn-process-id", "invoice", "--element-id", "approve_invoice",
			"--state", "created", "--assignee", "alice", "--candidate-user", "bob", "--candidate-group", "accounting")
		require.NoError(t, err, stderr)
		require.Empty(t, stderr)
		require.Equal(t, "2251799815391233\n", stdout)
	}
	before := len(requests.Snapshot())
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.7")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "unsupported")
	require.Len(t, requests.Snapshot(), before)
}

// TestGetUserTaskCommand_SearchUsesEffectiveTenant verifies configured,
// explicit override, and all-tenant discovery scopes reach the backend filter.
func TestGetUserTaskCommand_SearchUsesEffectiveTenant(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	config := fmt.Sprintf("app:\n  camunda_version: %q\n  tenant: %q\nauth:\n  mode: none\napis:\n  camunda_api:\n    base_url: %q\n", "8.9", "configured-tenant", server.URL)
	require.NoError(t, os.WriteFile(configPath, []byte(config), 0o600))

	_, stderr, err := runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	filter := requireJSONMap(t, requests.Snapshot()[0]["filter"])
	require.Equal(t, "configured-tenant", jsonFilterValue(t, filter["tenantId"]))

	_, stderr, err = runGetUserTaskCommand(t, configPath, "", "--tenant", "override-tenant", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	filter = requireJSONMap(t, requests.Snapshot()[1]["filter"])
	require.Equal(t, "override-tenant", jsonFilterValue(t, filter["tenantId"]))

	_, stderr, err = runGetUserTaskCommand(t, configPath, "", "--all-tenants", "--json", "get", "ut")
	require.NoError(t, err, stderr)
	filter = requireJSONMap(t, requests.Snapshot()[2]["filter"])
	require.NotContains(t, filter, "tenantId")
}

// TestGetUserTaskCommand_SearchRejectsMalformedBackendItems verifies a failed
// traversal cannot be rendered as a successful empty or partial collection.
func TestGetUserTaskCommand_SearchRejectsMalformedBackendItems(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return `{"items":[{"state":"CREATED","processInstanceKey":"2251799813711967"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut")
	require.Error(t, err)
	require.Empty(t, stderr)
	require.NotContains(t, stdout, `"outcome": "succeeded"`)
	require.Contains(t, stdout, `"outcome": "failed"`)
	require.Len(t, requests.Snapshot(), 1)
}

// TestGetUserTaskCommand_SearchReportsBackendFailure verifies HTTP failures
// retain command error classification and never emit a successful collection.
func TestGetUserTaskCommand_SearchReportsBackendFailure(t *testing.T) {
	var requests testx.AtomicCounter
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Inc()
		http.Error(writer, `{"message":"unavailable"}`, http.StatusServiceUnavailable)
	}))
	defer server.Close()
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "unavailable")
	require.Equal(t, int64(1), requests.Load())
}

// newGetUserTaskSearchServer captures generic generated-client request JSON and
// delegates deterministic page responses to each command scenario.
func newGetUserTaskSearchServer(t *testing.T, respond func(int, map[string]any) string) (*httptest.Server, *testx.SafeSlice[map[string]any]) {
	t.Helper()
	requests := new(testx.SafeSlice[map[string]any])
	var requestCount testx.AtomicCounter
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/v2/user-tasks/search" {
			http.NotFound(writer, request)
			return
		}
		var body map[string]any
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}
		index := int(requestCount.Inc() - 1)
		requests.Append(body)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(writer, respond(index, body))
	}))
	t.Cleanup(server.Close)
	return server, requests
}

// userTaskSearchResponse builds the stable response fields shared by supported versions.
func userTaskSearchResponse(total int64, capped bool, cursor string, keys ...string) string {
	items := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		items = append(items, map[string]any{
			"userTaskKey": key, "state": "CREATED", "name": "Approve invoice",
			"elementId": "approve_invoice", "assignee": "alice", "elementInstanceKey": "2251799815391200",
			"processInstanceKey": "2251799813711967", "processDefinitionKey": "2251799813689000",
			"processDefinitionId": "invoice", "tenantId": "tenant-a",
		})
	}
	payload := map[string]any{"items": items, "page": map[string]any{"totalItems": total, "hasMoreTotalItems": capped}}
	if cursor != "" {
		payload["page"].(map[string]any)["endCursor"] = cursor
	}
	encoded, _ := json.Marshal(payload)
	return string(encoded)
}

// requireJSONMap asserts and returns one decoded object.
func requireJSONMap(t *testing.T, value any) map[string]any {
	t.Helper()
	got, ok := value.(map[string]any)
	require.True(t, ok, "expected JSON object, got %#v", value)
	return got
}

// jsonFilterValue accepts scalar and generated equality-union JSON shapes.
func jsonFilterValue(t *testing.T, value any) string {
	t.Helper()
	if scalar, ok := value.(string); ok {
		return scalar
	}
	object := requireJSONMap(t, value)
	for _, key := range []string{"eq", "$eq"} {
		if scalar, ok := object[key].(string); ok {
			return scalar
		}
	}
	t.Fatalf("unsupported filter JSON %#v", value)
	return ""
}

// jsonPageLimit reads the limit from any supported generated pagination union.
func jsonPageLimit(t *testing.T, request map[string]any) float64 {
	t.Helper()
	page := requireJSONMap(t, request["page"])
	return page["limit"].(float64)
}

// jsonPageAfter reads a forward cursor from a captured request.
func jsonPageAfter(t *testing.T, request map[string]any) string {
	t.Helper()
	page := requireJSONMap(t, request["page"])
	return page["after"].(string)
}
