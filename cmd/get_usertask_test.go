// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const getUserTaskCommandHelper = "TestGetUserTaskCommandHelper"

// TestGetUserTaskCommand_KeyInputsAndAliases verifies canonical and alias
// dispatch, flag-before-stdin order, trimming, stable deduplication, and native GETs.
func TestGetUserTaskCommand_KeyInputsAndAliases(t *testing.T) {
	server, requests := newGetUserTaskCommandServer(t)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	for _, name := range []string{"user-task", "user-tasks", "ut", "uts"} {
		stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", name, "--key", "2251799815391233")
		require.NoError(t, err, stderr)
		require.Empty(t, stderr)
		require.Contains(t, stdout, "2251799815391233 tenant-a approve_invoice CREATED pi:2251799813711967 ei:2251799815391200 pd:2251799813689000 assignee:alice")
		require.Contains(t, stdout, "found: 1\n")
	}

	before := requests.Load()
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "\n 2251799815391234 \n2251799815391233\n\n", "--keys-only", "get", "user-task", "-k", "2251799815391233,2251799815391233")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "2251799815391233\n2251799815391234\n", stdout)
	require.Equal(t, int32(2), requests.Load()-before)

	stdout, stderr, err = runGetUserTaskCommand(t, configPath, "2251799815391234\n", "--keys-only", "get", "user-task")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "2251799815391234\n", stdout)

	stdout, stderr, err = runGetUserTaskCommand(t, configPath, "2251799815391234\n", "--keys-only", "get", "user-task", "-")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "2251799815391234\n", stdout)
}

// TestGetUserTaskCommand_MachineOutput pins the one-collection JSON shape and
// key-only stdout with stderr captured independently.
func TestGetUserTaskCommand_MachineOutput(t *testing.T) {
	server, _ := newGetUserTaskCommandServer(t)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut", "-k", "2251799815391233")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, "succeeded", envelope["outcome"])
	require.Equal(t, "get user-task", envelope["command"])
	payload := envelope["payload"].(map[string]any)
	require.Equal(t, float64(1), payload["total"])
	items := payload["items"].([]any)
	require.Len(t, items, 1)
	item := items[0].(map[string]any)
	require.Equal(t, []any{"bob"}, item["candidateUsers"])
	require.Equal(t, []any{"accounting"}, item["candidateGroups"])
}

// TestGetUserTaskCommand_VersionTenantDenialAndQuietMatrix verifies the keyed
// execution boundary across supported versions, V87 rejection, backend denial,
// authorized foreign-tenant metadata, and quiet machine modes.
func TestGetUserTaskCommand_VersionTenantDenialAndQuietMatrix(t *testing.T) {
	server, requests := newGetUserTaskCommandServer(t)
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		t.Run(version, func(t *testing.T) {
			configPath := testx.WriteTestConfigForVersion(t, server.URL, version)
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "configured-tenant", "--quiet", "--json", "get", "ut", "-k", "2251799815391233")
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			var envelope map[string]any
			require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
			item := envelope["payload"].(map[string]any)["items"].([]any)[0].(map[string]any)
			require.Equal(t, "tenant-a", item["tenantId"])
		})
	}

	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--quiet", "--keys-only", "get", "ut", "-k", "2251799815391233")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "2251799815391233\n", stdout)

	stdout, stderr, err = runGetUserTaskCommand(t, configPath, "", "get", "ut", "-k", "2251799815391288")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "forbidden")

	before := requests.Load()
	v87ConfigPath := testx.WriteTestConfigForVersion(t, server.URL, "8.7")
	stdout, stderr, err = runGetUserTaskCommand(t, v87ConfigPath, "", "get", "ut", "-k", "2251799815391233")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "unsupported")
	require.Equal(t, before, requests.Load(), "V87 native rejection must not issue a task request")
}

// TestGetUserTaskCommand_RejectsInvalidInputBeforeReads covers malformed flag
// and stdin keys, explicit empty stdin, extra args, bounds, and every keyed conflict.
func TestGetUserTaskCommand_RejectsInvalidInputBeforeReads(t *testing.T) {
	server, requests := newGetUserTaskCommandServer(t)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	longLine := strings.Repeat("1", 10*1024*1024+1) + "\n"

	tests := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{name: "malformed flag key", args: []string{"get", "ut", "-k", "123"}, want: `--key value "123" is not a valid key`},
		{name: "malformed stdin key", stdin: "123\n", args: []string{"get", "ut", "-"}, want: "validating keys from stdin failed"},
		{name: "scanner failure", stdin: longLine, args: []string{"get", "ut", "-"}, want: "token too long"},
		{name: "empty explicit dash", args: []string{"get", "ut", "-"}, want: "stdin contained no keys"},
		{name: "extra positional arg", args: []string{"get", "ut", "unexpected"}, want: "unexpected args"},
		{name: "invalid batch zero", args: []string{"get", "ut", "-k", "2251799815391233", "--batch-size", "0"}, want: "invalid value for --batch-size"},
		{name: "invalid batch high", args: []string{"get", "ut", "-k", "2251799815391233", "--batch-size", "1001"}, want: "invalid value for --batch-size"},
		{name: "invalid workers", args: []string{"get", "ut", "-k", "2251799815391233", "--workers", "0"}, want: "--workers must be positive integer"},
		{name: "pi key conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--pi-key", "2251799813711967"}, want: "--key cannot be combined"},
		{name: "pd key conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--pd-key", "2251799813689000"}, want: "--key cannot be combined"},
		{name: "bpmn conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--bpmn-process-id", "invoice"}, want: "--key cannot be combined"},
		{name: "element conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--element-id", "approve_invoice"}, want: "--key cannot be combined"},
		{name: "explicit all conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--state", "all"}, want: "--key cannot be combined"},
		{name: "assignee conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--assignee", "alice"}, want: "--key cannot be combined"},
		{name: "candidate user conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--candidate-user", "bob"}, want: "--key cannot be combined"},
		{name: "candidate group conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--candidate-group", "accounting"}, want: "--key cannot be combined"},
		{name: "limit conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--limit", "1"}, want: "--key cannot be combined"},
		{name: "total conflict", args: []string{"get", "ut", "-k", "2251799815391233", "--total"}, want: "--key cannot be combined"},
		{name: "stdin filter conflict", stdin: "2251799815391233\n", args: []string{"get", "ut", "--assignee", "alice"}, want: "--key cannot be combined"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := requests.Load()
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, test.stdin, test.args...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, test.want)
			require.Equal(t, before, requests.Load(), "invalid input issued a native read")
		})
	}
}

// TestGetUserTaskCommand_MissingKeyFailsWithoutPartialSuccess ensures strict
// bulk lookup does not render an earlier successful task when a later key is missing.
func TestGetUserTaskCommand_MissingKeyFailsWithoutPartialSuccess(t *testing.T) {
	server, _ := newGetUserTaskCommandServer(t)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut", "-k", "2251799815391233,2251799815391299")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "not found")
	require.NotContains(t, stderr, "found: 1")

	stdout, stderr, err = runGetUserTaskCommand(t, configPath, "", "--json", "get", "ut", "-k", "2251799815391233,2251799815391299")
	require.Error(t, err)
	require.Empty(t, stderr)
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, "failed", envelope["outcome"])
	require.Nil(t, envelope["payload"])
	require.NotContains(t, stdout, `"items"`)
}

// TestGetUserTaskCommand_KeyedVariablesInputAndAliases verifies every keyed
// spelling enriches selected tasks after stable stdin/flag merging and deduplication.
func TestGetUserTaskCommand_KeyedVariablesInputAndAliases(t *testing.T) {
	const firstKey = "2251799815391233"
	const secondKey = "2251799815391234"
	onePage := func(name, value string) []userTaskVariablePageFixture {
		return []userTaskVariablePageFixture{{Total: 1, Items: []userTaskVariableFixtureValue{{
			Name: name, Value: value, VariableKey: "901", ProcessInstanceKey: "2251799813711967",
			ScopeKey: "2251799815391200", TenantID: "tenant-a",
		}}}}
	}

	for _, alias := range []string{"user-task", "user-tasks", "ut", "uts"} {
		t.Run(alias, func(t *testing.T) {
			server, requests := newGetUserTaskVariablesServer(t, getUserTaskVariablesFixture{
				VariablePages: map[string][]userTaskVariablePageFixture{firstKey: onePage("amount", "120")},
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", alias, "--key", firstKey, "--with-vars")
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Equal(t, ""+
				firstKey+" tenant-a approve_invoice CREATED pi:2251799813711967 ei:2251799815391200 pd:2251799813689000 assignee:alice\n"+
				"└─ vars:\n"+
				"   └─ amount=120\n"+
				"found: 1\n", stdout)
			require.Equal(t, 1, requests.variableRequestCount(firstKey))
		})
	}

	server, requests := newGetUserTaskVariablesServer(t, getUserTaskVariablesFixture{
		VariablePages: map[string][]userTaskVariablePageFixture{
			firstKey:  onePage("amount", "120"),
			secondKey: {{Total: 0, Items: []userTaskVariableFixtureValue{}}},
		},
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "\n"+secondKey+"\n"+firstKey+"\n", "get", "ut", "--workers", "1", "--key", firstKey+","+firstKey, "--with-vars", "-")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Less(t, strings.Index(stdout, firstKey+" tenant-a"), strings.Index(stdout, secondKey+" tenant-a"))
	require.Equal(t, 1, strings.Count(stdout, firstKey+" tenant-a"))
	require.Equal(t, 1, strings.Count(stdout, "amount=120"))
	require.Equal(t, 1, strings.Count(stdout, "vars:"), "the task without variables must not invent a subtree")
	taskKeys, _, _ := requests.snapshot()
	require.Equal(t, []string{firstKey, secondKey}, taskKeys)
	require.Equal(t, 1, requests.variableRequestCount(firstKey))
	require.Equal(t, 1, requests.variableRequestCount(secondKey))

	implicitServer, implicitRequests := newGetUserTaskVariablesServer(t, getUserTaskVariablesFixture{
		VariablePages: map[string][]userTaskVariablePageFixture{secondKey: onePage("status", `"done"`)},
	})
	implicitConfig := testx.WriteTestConfigForVersion(t, implicitServer.URL, "8.9")
	stdout, stderr, err = runGetUserTaskCommand(t, implicitConfig, secondKey+"\n", "get", "ut", "--with-vars")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Contains(t, stdout, `status="done"`)
	require.Equal(t, 1, implicitRequests.variableRequestCount(secondKey))
}

// TestGetUserTaskCommand_KeyedVariablesJSONPreservesEffectiveMetadata verifies
// complete variable paging, backend-selected scope, authorized tenant metadata,
// stable effective-name normalization, and a single enriched JSON envelope.
func TestGetUserTaskCommand_KeyedVariablesJSONPreservesEffectiveMetadata(t *testing.T) {
	const key = "2251799815391233"
	server, requests := newGetUserTaskVariablesServer(t, getUserTaskVariablesFixture{
		VariablePages: map[string][]userTaskVariablePageFixture{key: {
			{Total: 4, Items: []userTaskVariableFixtureValue{
				{Name: "shared", Value: `"local"`, VariableKey: "902", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-foreign"},
				{Name: "zeta", Value: "2", VariableKey: "903", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799813711967", TenantID: "tenant-foreign"},
			}},
			{Total: 4, Items: []userTaskVariableFixtureValue{
				{Name: "shared", Value: `"local"`, VariableKey: "902", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-foreign"},
				{Name: "alpha", Value: "1", VariableKey: "904", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799813711967", TenantID: "tenant-foreign"},
			}},
		}},
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "configured-tenant", "--json", "get", "ut", "--key", key, "--with-vars")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	payload := envelope["payload"].(map[string]any)
	require.Equal(t, float64(1), payload["total"])
	enriched := payload["items"].([]any)[0].(map[string]any)
	require.Equal(t, "tenant-a", enriched["item"].(map[string]any)["tenantId"])
	variables := enriched["variables"].([]any)
	require.Len(t, variables, 3)
	require.Equal(t, "alpha", variables[0].(map[string]any)["name"])
	require.Equal(t, "shared", variables[1].(map[string]any)["name"])
	require.Equal(t, "2251799815391200", variables[1].(map[string]any)["scopeKey"])
	require.Equal(t, "tenant-foreign", variables[1].(map[string]any)["tenantId"])
	require.Equal(t, "zeta", variables[2].(map[string]any)["name"])
	require.Equal(t, 2, requests.variableRequestCount(key))
}

// TestGetUserTaskCommand_KeyedVariableFailuresDoNotRenderPartialSuccess verifies
// denied or disappeared tasks and later variable-page failures retain strict errors.
func TestGetUserTaskCommand_KeyedVariableFailuresDoNotRenderPartialSuccess(t *testing.T) {
	const deniedKey = "2251799815391288"
	const missingKey = "2251799815391299"
	const failingKey = "2251799815391233"
	for _, test := range []struct {
		name          string
		fixture       getUserTaskVariablesFixture
		key           string
		want          string
		variableCalls int
	}{
		{name: "denied task", key: deniedKey, fixture: getUserTaskVariablesFixture{TaskStatuses: map[string]int{deniedKey: http.StatusForbidden}}, want: "forbidden"},
		{name: "disappeared task", key: missingKey, fixture: getUserTaskVariablesFixture{TaskStatuses: map[string]int{missingKey: http.StatusNotFound}}, want: "not found"},
		{name: "later variable page", key: failingKey, fixture: getUserTaskVariablesFixture{VariablePages: map[string][]userTaskVariablePageFixture{failingKey: {
			{Total: 2, Items: []userTaskVariableFixtureValue{{Name: "alpha", Value: "1", VariableKey: "901", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799813711967", TenantID: "tenant-a"}}},
			{Status: http.StatusInternalServerError},
		}}}, want: "internal server error", variableCalls: 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskVariablesServer(t, test.fixture)
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut", "--key", test.key, "--with-vars")
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, strings.ToLower(stderr), test.want)
			require.Equal(t, test.variableCalls, requests.variableRequestCount(test.key))
		})
	}
}

// TestGetUserTaskCommand_VariableModeGates verifies enrichment is skipped for
// absent opt-in, effective keys-only, total, and empty selections while quiet
// human and JSON-precedence executions still retrieve requested variables.
func TestGetUserTaskCommand_VariableModeGates(t *testing.T) {
	const key = "2251799815391233"
	newFixture := func() getUserTaskVariablesFixture {
		return getUserTaskVariablesFixture{
			SearchRespond: func(_ int, _ map[string]any) string { return userTaskSearchResponse(0, false, "") },
			VariablePages: map[string][]userTaskVariablePageFixture{key: {{
				Total: 1, Items: []userTaskVariableFixtureValue{{Name: "amount", Value: "120", VariableKey: "901", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-a"}},
			}}},
		}
	}
	for _, test := range []struct {
		name          string
		args          []string
		want          string
		wantEmpty     bool
		variableCalls int
	}{
		{name: "absent flag", args: []string{"get", "ut", "--key", key}, want: "found: 1\n"},
		{name: "keys only", args: []string{"--keys-only", "get", "ut", "--key", key, "--with-vars"}, want: key + "\n"},
		{name: "total", args: []string{"get", "ut", "--total", "--with-vars"}, want: "0\n"},
		{name: "empty search", args: []string{"get", "ut", "--with-vars"}, want: "found: 0\n"},
		{name: "quiet human", args: []string{"--quiet", "get", "ut", "--key", key, "--with-vars"}, wantEmpty: true, variableCalls: 1},
		{name: "json precedence", args: []string{"--json", "--keys-only", "get", "ut", "--key", key, "--with-vars"}, want: `"variables"`, variableCalls: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskVariablesServer(t, newFixture())
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			if test.wantEmpty {
				require.Empty(t, stdout)
			} else {
				require.Contains(t, stdout, test.want)
			}
			require.Equal(t, test.variableCalls, requests.variableRequestCount(key))
		})
	}
}

// TestGetUserTaskCommandHelper executes one isolated real command path because
// production command errors terminate the process.
func TestGetUserTaskCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_USER_TASK_ARGS")), &args); err != nil {
		t.Fatalf("decode helper args: %v", err)
	}
	flagGetUserTaskVarValueLimit = 0
	os.Args = append([]string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...)
	Execute()
	os.Exit(0)
}

// runGetUserTaskCommand captures result and control streams separately for one
// helper-process invocation.
func runGetUserTaskCommand(t *testing.T, configPath, stdin string, args ...string) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	return testx.RunCmdSubprocessInDirWithSeparateOutputs(t, getUserTaskCommandHelper, "", map[string]string{
		"C8VOLT_TEST_CONFIG":         configPath,
		"C8VOLT_TEST_USER_TASK_ARGS": string(encoded),
	}, stdin)
}

// newGetUserTaskCommandServer returns native tasks and counts every request so
// validation cases can prove they stop before backend access.
func newGetUserTaskCommandServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		key := strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/")
		if key == "2251799815391288" {
			http.Error(writer, `{"message":"denied"}`, http.StatusForbidden)
			return
		}
		if request.Method != http.MethodGet || key == request.URL.Path || key == "2251799815391299" {
			http.NotFound(writer, request)
			return
		}
		name, elementID, state, assignee, piKey := "Approve invoice", "approve_invoice", "CREATED", "alice", "2251799813711967"
		if key == "2251799815391234" {
			name, elementID, state, assignee, piKey = "", "archive_invoice", "COMPLETED", "", "2251799813711968"
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(writer, `{"userTaskKey":%q,"state":%q,"name":%q,"elementId":%q,"assignee":%q,"candidateUsers":["bob"],"candidateGroups":["accounting"],"processInstanceKey":%q,"elementInstanceKey":"2251799815391200","processDefinitionKey":"2251799813689000","processDefinitionId":"invoice","processDefinitionVersion":7,"tenantId":"tenant-a"}`, key, state, name, elementID, assignee, piKey)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}
