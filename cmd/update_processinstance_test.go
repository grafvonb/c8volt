// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestUpdatePICommand_SubmitsV88UpdateAndConfirmsVariables(t *testing.T) {
	var sawUpdate bool
	var sawConfirmation bool
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			sawUpdate = true
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCalls++
			w.Header().Set("Content-Type", "application/json")
			if searchCalls == 1 {
				_, _ = w.Write([]byte(emptyVariableSearchResponse()))
				return
			}
			sawConfirmation = true
			_, _ = w.Write([]byte(variableSearchResponse(`{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}`)))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
	)

	require.True(t, sawUpdate)
	require.True(t, sawConfirmation)
	envelope := requireUpdateProcessInstanceEnvelope(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "update process-instance", envelope["command"])
	item := firstUpdateResultItem(t, envelope)
	require.Equal(t, "2251799813711967", item["key"])
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, true, item["mutationAccepted"])
	require.Equal(t, "confirmed", item["confirmationStatus"])
}

// Protects the existing `update pi --vars` request path after adding resolve commands.
func TestUpdatePICommand_RegressionVarsUsesVariableMutationAndConfirmation(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			if len(requests) == 1 {
				_, _ = w.Write([]byte(emptyVariableSearchResponse()))
				return
			}
			_, _ = w.Write([]byte(variableSearchResponse(`{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}`)))
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
		"--auto-confirm",
	)

	require.Equal(t, []string{
		"POST /v2/variables/search",
		"PUT /v2/element-instances/2251799813711967/variables",
		"POST /v2/variables/search",
	}, requests)
	require.Contains(t, output, "updated process-instance 2251799813711967: confirmed")
	require.Contains(t, output, "updated: 1 (confirmed/submitted: 1, failed: 0)")
}

func TestUpdatePICommand_VarsFileSubmitsAndConfirmsVariables(t *testing.T) {
	var sawUpdate bool
	var sawConfirmation bool
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			sawUpdate = true
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			searchCalls++
			w.Header().Set("Content-Type", "application/json")
			if searchCalls == 1 {
				_, _ = w.Write([]byte(emptyVariableSearchResponse()))
				return
			}
			sawConfirmation = true
			_, _ = w.Write([]byte(variableSearchResponse(`{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}`)))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	varsPath := t.TempDir() + "/vars.json"
	require.NoError(t, os.WriteFile(varsPath, []byte(`{"foo":"bar"}`), 0o600))

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"update", "pi",
		"--key", "2251799813711967",
		"--vars-file", varsPath,
	)

	require.True(t, sawUpdate)
	require.True(t, sawConfirmation)
	envelope := requireUpdateProcessInstanceEnvelope(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	item := firstUpdateResultItem(t, envelope)
	require.Equal(t, "confirmed", item["status"])
	require.Equal(t, "confirmed", item["confirmationStatus"])
}

func TestUpdateProcessInstanceCommand_MultipleRepeatedKeysApplyOneVarsPayloadToEachUniqueKey(t *testing.T) {
	var mu sync.Mutex
	updates := map[string]int{}
	confirmations := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables",
			"/v2/element-instances/2251799813711968/variables":
			require.Equal(t, http.MethodPut, r.Method)
			key := keyFromElementInstanceVariablesPath(t, r.URL.Path)
			mu.Lock()
			updates[key]++
			mu.Unlock()
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			mu.Lock()
			confirmations++
			searchCall := confirmations
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if searchCall <= 2 {
				_, _ = w.Write([]byte(emptyVariableSearchResponse()))
				return
			}
			_, _ = w.Write([]byte(variableSearchResponse(`{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}`, `{"name":"foo","value":"\"bar\"","variableKey":"902","processInstanceKey":"2251799813711968","scopeKey":"2251799813711968","tenantId":"<default>"}`)))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"update", "process-instance",
		"--key", "2251799813711967",
		"--key", "2251799813711968",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
	)

	mu.Lock()
	gotUpdates := map[string]int{}
	for key, count := range updates {
		gotUpdates[key] = count
	}
	gotConfirmations := confirmations
	mu.Unlock()
	require.Equal(t, map[string]int{
		"2251799813711967": 1,
		"2251799813711968": 1,
	}, gotUpdates)
	require.Equal(t, 4, gotConfirmations)
	envelope := requireUpdateProcessInstanceEnvelope(t, stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	requireUpdateResultKeys(t, envelope, "2251799813711967", "2251799813711968")
}

func TestUpdateProcessInstanceCommand_StdinKeysMergeAndDeduplicateWithFlagKeys(t *testing.T) {
	var mu sync.Mutex
	updates := map[string]int{}
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables",
			"/v2/element-instances/2251799813711968/variables":
			key := keyFromElementInstanceVariablesPath(t, r.URL.Path)
			mu.Lock()
			updates[key]++
			mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			mu.Lock()
			searchCalls++
			searchCall := searchCalls
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			if searchCall <= 2 {
				_, _ = w.Write([]byte(emptyVariableSearchResponse()))
				return
			}
			_, _ = w.Write([]byte(variableSearchResponse(`{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}`, `{"name":"foo","value":"\"bar\"","variableKey":"902","processInstanceKey":"2251799813711968","scopeKey":"2251799813711968","tenantId":"<default>"}`)))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout := executeRootForProcessInstanceTestWithStdin(t,
		"2251799813711967\n2251799813711968\n2251799813711968\n",
		"--config", cfgPath,
		"update", "pi",
		"--key", "2251799813711967",
		"-",
		"--vars", `{"foo":"bar"}`,
	)

	mu.Lock()
	gotUpdates := map[string]int{}
	for key, count := range updates {
		gotUpdates[key] = count
	}
	mu.Unlock()
	require.Equal(t, map[string]int{
		"2251799813711967": 1,
		"2251799813711968": 1,
	}, gotUpdates)
	require.Contains(t, stdout, "updated process-instance 2251799813711967: confirmed")
	require.Contains(t, stdout, "updated process-instance 2251799813711968: confirmed")
	require.Contains(t, stdout, "updated: 2")
}

func TestUpdateProcessInstanceCommand_NoWaitReturnsSubmittedAfterPreflightOnly(t *testing.T) {
	var sawUpdate bool
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			sawUpdate = true
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, map[string]any{"foo": "bar"}, body["variables"])
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			searchCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
		"--no-wait",
	)

	require.True(t, sawUpdate)
	require.Equal(t, 1, searchCalls)
	require.Contains(t, output, "updated process-instance 2251799813711967: submitted")
	require.Contains(t, output, "updated: 1 (confirmed/submitted: 1, failed: 0)")
}

func TestUpdateProcessInstanceCommand_NoWaitJSONReportsSubmittedResultsAfterPreflightOnly(t *testing.T) {
	var sawUpdate bool
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			require.Equal(t, http.MethodPut, r.Method)
			sawUpdate = true
			w.WriteHeader(http.StatusNoContent)
		case "/v2/variables/search":
			searchCalls++
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
		"--no-wait",
	)

	require.True(t, sawUpdate)
	require.Equal(t, 1, searchCalls)
	envelope := requireUpdateProcessInstanceEnvelope(t, stdout)
	require.Equal(t, string(OutcomeAccepted), envelope["outcome"])
	require.Equal(t, "update process-instance", envelope["command"])
	item := firstUpdateResultItem(t, envelope)
	require.Equal(t, "2251799813711967", item["key"])
	require.Equal(t, "submitted", item["status"])
	require.Equal(t, true, item["mutationAccepted"])
	require.Equal(t, "skipped", item["confirmationStatus"])
}

func TestUpdateProcessInstanceCommand_FullNameAndAliasBehaveIdenticallyForSingleKey(t *testing.T) {
	for _, leaf := range []string{"process-instance", "pi"} {
		t.Run(leaf, func(t *testing.T) {
			var requestedPath string
			searchCalls := 0
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/v2/element-instances/2251799813711967/variables":
					require.Equal(t, http.MethodPut, r.Method)
					requestedPath = r.URL.Path
					w.WriteHeader(http.StatusNoContent)
				case "/v2/variables/search":
					searchCalls++
					w.Header().Set("Content-Type", "application/json")
					if searchCalls == 1 {
						_, _ = w.Write([]byte(emptyVariableSearchResponse()))
						return
					}
					_, _ = w.Write([]byte(variableSearchResponse(`{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}`)))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
				"--config", cfgPath,
				"update", leaf,
				"--key", "2251799813711967",
				"--vars", `{"foo":"bar"}`,
			)

			require.Equal(t, "/v2/element-instances/2251799813711967/variables", requestedPath)
			require.Empty(t, stdout)
			require.Contains(t, stderr, "updated process-instance 2251799813711967: confirmed")
			require.Contains(t, stderr, "updated: 1")
		})
	}
}

func TestUpdateProcessInstanceVariablePlan_ClassifiesRequestedAndUntouchedVariables(t *testing.T) {
	plan := newProcessInstanceVariableUpdatePlan("2251799813687231", []process.ProcessInstanceVariable{
		{Name: "isActive", Value: "true", ProcessInstanceKey: "2251799813687231", ScopeKey: "2251799813687231"},
		{Name: "same", Value: `"gold"`, ProcessInstanceKey: "2251799813687231", ScopeKey: "2251799813687231"},
		{Name: "businessId", Value: "1334283", ProcessInstanceKey: "2251799813687231", ScopeKey: "2251799813687231"},
		{Name: "canRun", Value: "true", ProcessInstanceKey: "2251799813687231", ScopeKey: "2251799813687231"},
		{Name: "elementLocal", Value: `"ignored"`, ProcessInstanceKey: "2251799813687231", ScopeKey: "element-1"},
		{Name: "wrongOwner", Value: `"ignored"`, ProcessInstanceKey: "999", ScopeKey: "999"},
	}, map[string]any{
		"isActive": false,
		"message":  "hello",
		"same":     "gold",
	})

	require.Equal(t, "2251799813687231", plan.ProcessInstanceKey)
	require.Equal(t, []processInstanceVariablePlannedValue{{Name: "message", Value: "hello"}}, plan.Additions)
	require.Equal(t, []processInstanceVariablePlannedChange{{Name: "isActive", Before: true, After: false}}, plan.Changes)
	require.Equal(t, []processInstanceVariablePlannedValue{{Name: "same", Value: "gold"}}, plan.UnchangedRequested)
	require.Equal(t, []processInstanceVariablePlannedValue{
		{Name: "businessId", Value: float64(1334283)},
		{Name: "canRun", Value: true},
	}, plan.Untouched)
}

func TestUpdateProcessInstanceVariableDryRun_HumanOutputUsesCompactPlanSyntax(t *testing.T) {
	prevVerbose := flagVerbose
	flagVerbose = false
	t.Cleanup(func() { flagVerbose = prevVerbose })

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	preview := newProcessInstanceVariableUpdatePreview(types.Keys{"2251799813687231"}, []processInstanceVariableUpdatePlan{{
		ProcessInstanceKey: "2251799813687231",
		Additions:          []processInstanceVariablePlannedValue{{Name: "message", Value: "hello"}},
		Changes:            []processInstanceVariablePlannedChange{{Name: "isActive", Before: true, After: false}},
		Untouched: []processInstanceVariablePlannedValue{
			{Name: "businessId", Value: float64(1334283)},
			{Name: "canRun", Value: true},
		},
	}})

	require.NoError(t, renderUpdateProcessInstanceVariablePreview(cmd, preview))

	output := buf.String()
	require.Contains(t, output, "dry run: update process-instance variables: 1 process instance(s), 1 change(s), 1 addition(s), 0 unchanged, 2 untouched; no changes applied")
	require.Contains(t, output, `2251799813687231: ~ isActive: true -> false; + message: "hello"; = businessId: 1334283, canRun: true`)
	require.NotContains(t, output, "selected process instances")
	require.NotContains(t, output, "variables to add")
	require.NotContains(t, output, "variables to change")
	require.NotContains(t, output, "\n\n")
}

func TestUpdateProcessInstanceVariableDryRun_JSONIgnoresVerboseForStableShape(t *testing.T) {
	prevJSON := flagViewAsJson
	prevVerbose := flagVerbose
	flagViewAsJson = true
	flagVerbose = false
	t.Cleanup(func() {
		flagViewAsJson = prevJSON
		flagVerbose = prevVerbose
	})

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	preview := newProcessInstanceVariableUpdatePreview(types.Keys{"2251799813687231"}, []processInstanceVariableUpdatePlan{{
		ProcessInstanceKey: "2251799813687231",
		Changes:            []processInstanceVariablePlannedChange{{Name: "buba", Before: "was here 3", After: "was here 2"}},
	}})

	require.NoError(t, renderUpdateProcessInstanceVariablePreview(cmd, preview))
	payload := requireDryRunEnvelopePayload(t, buf.String())
	items := requireJSONItems(t, payload["processInstances"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813687231", item["processInstanceKey"])
	require.Equal(t, float64(1), payload["variableChangeCount"])

	defaultOutput := buf.String()
	buf.Reset()
	flagVerbose = true
	require.NoError(t, renderUpdateProcessInstanceVariablePreview(cmd, preview))
	require.JSONEq(t, defaultOutput, buf.String())
}

func TestUpdateProcessInstanceVariableDryRun_NoPlannedChangesStaysCompact(t *testing.T) {
	prevVerbose := flagVerbose
	flagVerbose = false
	t.Cleanup(func() { flagVerbose = prevVerbose })

	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	preview := newProcessInstanceVariableUpdatePreview(types.Keys{"2251799813687231"}, []processInstanceVariableUpdatePlan{{
		ProcessInstanceKey: "2251799813687231",
		UnchangedRequested: []processInstanceVariablePlannedValue{
			{Name: "isActive", Value: true},
			{Name: "message", Value: "hello"},
		},
		Untouched: []processInstanceVariablePlannedValue{
			{Name: "businessId", Value: float64(1334283)},
			{Name: "canRun", Value: true},
		},
	}})

	require.False(t, preview.HasPlannedChanges())
	require.Equal(t, 0, preview.UpdateCount)
	require.NoError(t, renderUpdateProcessInstanceVariablePreview(cmd, preview))

	output := buf.String()
	require.Contains(t, output, "dry run: update process-instance variables: nothing to update (2 requested value(s) already match visible variables); no changes applied")
	require.NotContains(t, output, "process instances to update")
	require.NotContains(t, output, "variables unchanged by request")
	require.NotContains(t, output, "isActive")
	require.NotContains(t, output, "businessId")
	require.NotContains(t, output, "variables left untouched")
}

func TestFormatProcessInstanceVariableUpdatePlan_RendersUnchangedWithoutArrow(t *testing.T) {
	plan := processInstanceVariableUpdatePlan{
		ProcessInstanceKey: "2251799813687231",
		UnchangedRequested: []processInstanceVariablePlannedValue{
			{Name: "hasIncident", Value: false},
		},
	}

	require.Equal(t, "~ hasIncident: false (unchanged)", formatProcessInstanceVariableUpdatePlan(plan))
}

func TestUpdateProcessInstanceCommand_NoPlannedChangesSkipsPromptAndMutation(t *testing.T) {
	var sawSearch bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/element-instances/2251799813711967/variables":
			t.Fatalf("no-op update must not submit mutation")
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			sawSearch = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"name":"foo","value":"\"bar\"","variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799813711967","tenantId":"<default>"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("no-op update must not prompt for confirmation")
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"update", "pi",
		"--key", "2251799813711967",
		"--vars", `{"foo":"bar"}`,
	)

	require.True(t, sawSearch)
	require.Contains(t, output, "plan: update process-instance variables")
	require.Contains(t, output, "nothing to update")
	require.Contains(t, output, "no confirmation required")
	require.NotContains(t, output, "updated process-instance")
}

func TestUpdateProcessInstanceCommand_JSONMutationRequiresExplicitConfirmation(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestUpdateProcessInstanceCommand_JSONMutationRequiresExplicitConfirmationHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, []string{
			"--json",
			"update", "pi",
			"--key", "2251799813711967",
			"--vars", `{"foo":"bar"}`,
		}),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.NotContains(t, string(output), "plan: update process-instance variables")
	require.NotContains(t, string(output), "Do you want to proceed?")

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(output, &envelope))
	require.Equal(t, string(OutcomeInvalid), envelope["outcome"])
	require.Equal(t, "invalid_input", envelope["class"])
	detail := requireJSONObject(t, envelope["detail"])
	require.Contains(t, detail["message"], "--json update pi requires --dry-run, --auto-confirm, or --automation")
}

func TestUpdateProcessInstanceCommand_JSONRejectsVerbose(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestUpdateProcessInstanceCommand_JSONRejectsVerboseHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
		"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, []string{
			"--json",
			"--verbose",
			"update", "pi",
			"--key", "2251799813711967",
			"--vars", `{"foo":"bar"}`,
			"--dry-run",
		}),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.NotContains(t, string(output), "dry run: update process-instance variables")
	require.NotContains(t, string(output), "Do you want to proceed?")

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(output, &envelope))
	require.Equal(t, string(OutcomeInvalid), envelope["outcome"])
	require.Equal(t, "invalid_input", envelope["class"])
	detail := requireJSONObject(t, envelope["detail"])
	require.Contains(t, detail["message"], "--json cannot be combined with --verbose for update pi")
}

func TestUpdateProcessInstanceCommand_InvalidVarsFailBeforeMutation(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	malformedVarsFile := t.TempDir() + "/malformed-vars.json"
	require.NoError(t, os.WriteFile(malformedVarsFile, []byte(`{"foo":`), 0o600))

	tests := []struct {
		name       string
		helperName string
		args       []string
		want       string
	}{
		{
			name:       "missing vars",
			helperName: "TestUpdateProcessInstanceCommand_MissingVarsHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967"},
			want:       "--vars or --vars-file is required and must be a JSON object",
		},
		{
			name:       "mutually exclusive vars and vars file",
			helperName: "TestUpdateProcessInstanceCommand_MutuallyExclusiveVarsHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars", `{"foo":"bar"}`, "--vars-file", malformedVarsFile},
			want:       "--vars cannot be combined with --vars-file",
		},
		{
			name:       "vars file flag missing argument",
			helperName: "TestUpdateProcessInstanceCommand_VarsFileMissingArgumentHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars", `{"foo":"bar"}`, "--vars-file"},
			want:       "flag needs an argument: --vars-file",
		},
		{
			name:       "malformed json",
			helperName: "TestUpdateProcessInstanceCommand_MalformedVarsHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars", `{"foo":`},
			want:       "--vars must be a valid JSON object",
		},
		{
			name:       "non object json",
			helperName: "TestUpdateProcessInstanceCommand_NonObjectVarsHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars", `["foo"]`},
			want:       "--vars must be a valid JSON object",
		},
		{
			name:       "missing vars file",
			helperName: "TestUpdateProcessInstanceCommand_MissingVarsFileHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars-file", t.TempDir() + "/missing.json"},
			want:       "--vars-file could not be read",
		},
		{
			name:       "malformed vars file",
			helperName: "TestUpdateProcessInstanceCommand_MalformedVarsFileHelper",
			args:       []string{"update", "pi", "--key", "2251799813711967", "--vars-file", malformedVarsFile},
			want:       "--vars-file must be a valid JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, tt.helperName, map[string]string{
				"C8VOLT_TEST_CONFIG":      cfgPath,
				"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, tt.args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), tt.want)
			require.NotContains(t, string(output), "Usage:")
		})
	}
}

func TestUpdateProcessInstanceCommand_MissingTargetsFailBeforeMutation(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	t.Run("no key", func(t *testing.T) {
		output, err := testx.RunCmdSubprocess(t, "TestUpdateProcessInstanceCommand_MissingKeyHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":      cfgPath,
			"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, []string{"update", "pi", "--vars", `{"foo":"bar"}`}),
		})
		require.Error(t, err)

		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.Error, exitErr.ExitCode())
		require.Contains(t, string(output), "local precondition failed")
		require.Contains(t, string(output), "no process instance keys provided or found to update")
	})

	t.Run("empty stdin", func(t *testing.T) {
		output, err := testx.RunCmdSubprocessWithStdin(t, "TestUpdateProcessInstanceCommand_EmptyStdinHelper", map[string]string{
			"C8VOLT_TEST_CONFIG":      cfgPath,
			"C8VOLT_TEST_UPDATE_ARGS": marshalUpdateArgsForEnv(t, []string{"update", "pi", "-", "--vars", `{"foo":"bar"}`}),
		}, "\n")
		require.Error(t, err)

		exitErr, ok := err.(*exec.ExitError)
		require.True(t, ok)
		require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
		require.Contains(t, string(output), "invalid input")
		require.Contains(t, string(output), "stdin contained no keys")
	})
}

func requireUpdateProcessInstanceEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	return envelope
}

func firstUpdateResultItem(t *testing.T, envelope map[string]any) map[string]any {
	t.Helper()

	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item, ok := items[0].(map[string]any)
	require.True(t, ok)
	return item
}

func requireUpdateResultKeys(t *testing.T, envelope map[string]any, want ...string) {
	t.Helper()

	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, len(want))
	got := make([]string, 0, len(items))
	for _, raw := range items {
		item, ok := raw.(map[string]any)
		require.True(t, ok)
		got = append(got, item["key"].(string))
		require.Equal(t, "confirmed", item["status"])
		require.Equal(t, true, item["mutationAccepted"])
		require.Equal(t, "confirmed", item["confirmationStatus"])
	}
	sort.Strings(got)
	sort.Strings(want)
	require.Equal(t, want, got)
}

func TestUpdateProcessInstanceCommand_MissingVarsHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_MalformedVarsHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_MutuallyExclusiveVarsHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_VarsFileMissingArgumentHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_NonObjectVarsHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_MissingVarsFileHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_MalformedVarsFileHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_JSONMutationRequiresExplicitConfirmationHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_JSONRejectsVerboseHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_MissingKeyHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func TestUpdateProcessInstanceCommand_EmptyStdinHelper(t *testing.T) {
	runUpdateProcessInstanceHelperFromEnv(t)
}

func runUpdateProcessInstanceHelperFromEnv(t *testing.T) {
	t.Helper()
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var updateArgs []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_UPDATE_ARGS")), &updateArgs); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, updateArgs...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func marshalUpdateArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

func keyFromElementInstanceVariablesPath(t *testing.T, path string) string {
	t.Helper()

	const prefix = "/v2/element-instances/"
	const suffix = "/variables"
	require.True(t, len(path) > len(prefix)+len(suffix))
	require.Equal(t, prefix, path[:len(prefix)])
	require.Equal(t, suffix, path[len(path)-len(suffix):])
	return path[len(prefix) : len(path)-len(suffix)]
}

func emptyVariableSearchResponse() string {
	return `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
}

func variableSearchResponse(items ...string) string {
	return `{"items":[` + strings.Join(items, ",") + `],"page":{"totalItems":` + strconv.Itoa(len(items)) + `,"hasMoreTotalItems":false}}`
}

func executeRootForProcessInstanceTestWithStdin(t *testing.T, stdin string, args ...string) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	require.NoError(t, err)
	_, err = writer.WriteString(stdin)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	prevStdin := os.Stdin
	os.Stdin = reader
	t.Cleanup(func() {
		os.Stdin = prevStdin
		_ = reader.Close()
	})

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm

	_, err = root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}
