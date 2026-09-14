// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
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
		require.Contains(t, stdout, "2251799815391233 tenant-a approve_invoice CREATED name:Approve invoice assignee:alice invoice pi:2251799813711967 ei:2251799815391200 pd:2251799813689000")
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
		_, _ = fmt.Fprintf(writer, `{"userTaskKey":%q,"state":%q,"name":%q,"elementId":%q,"assignee":%q,"candidateUsers":["bob"],"candidateGroups":["accounting"],"processInstanceKey":%q,"elementInstanceKey":"2251799815391200","processDefinitionKey":"2251799813689000","processDefinitionId":"invoice","tenantId":"tenant-a"}`, key, state, name, elementID, assignee, piKey)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}
