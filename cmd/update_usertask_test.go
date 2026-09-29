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
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

const updateUserTaskCommandHelper = "TestUpdateUserTaskCommandHelper"

type updateUserTaskCommandCapture struct {
	mu            sync.Mutex
	requests      int
	mutations     int
	mutationPaths []string
	mutationBody  []map[string]any
	taskReads     []string
	variableReads []string
}

// TestUpdateUserTaskCommand_PlansSubmitsAndConfirmsScopeLocalVariables proves
// the command delegates its frozen scope plan through the composed workflow.
func TestUpdateUserTaskCommand_PlansSubmitsAndConfirmsScopeLocalVariables(t *testing.T) {
	server, capture := newUpdateUserTaskCommandServer(t)
	configPath := writeTestConfigForVersion(t, server.URL, "8.10")

	stdout, stderr, err := runUpdateUserTaskCommand(t, configPath, "",
		"--automation", "--json", "update", "ut",
		"--key", "2251799815391233", "--vars", `{"approved":true}`,
	)
	require.NoError(t, err)
	require.Empty(t, stderr)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, "succeeded", envelope["outcome"])
	require.Equal(t, "update user-task", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799815391233", item["key"])
	require.Equal(t, "confirmed", item["status"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 1, capture.mutations)
	require.Equal(t, []string{"/v2/element-instances/2251799815391100/variables"}, capture.mutationPaths)
	require.Equal(t, true, capture.mutationBody[0]["local"])
	require.Equal(t, map[string]any{"approved": true}, capture.mutationBody[0]["variables"])
}

// TestUpdateUserTaskPayloadParserMatchesProcessInstanceContract protects the
// shared inline parser and preserves null property values.
func TestUpdateUserTaskPayloadParserMatchesProcessInstanceContract(t *testing.T) {
	got, err := parseUpdateVariables(`{"approved":null,"count":2}`, "--vars")
	require.NoError(t, err)
	require.Equal(t, map[string]any{"approved": nil, "count": float64(2)}, got)

	_, err = parseUpdateProcessInstanceVariables(`["not-an-object"]`, "--vars")
	require.Error(t, err)
	require.Contains(t, err.Error(), "must be a valid JSON object")
}

// TestUpdateUserTaskCommandHelper executes isolated command paths because
// production command errors terminate the process.
func TestUpdateUserTaskCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_UPDATE_USER_TASK_ARGS")), &args); err != nil {
		t.Fatalf("decode helper args: %v", err)
	}
	os.Args = append([]string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...)
	Execute()
	os.Exit(0)
}

// runUpdateUserTaskCommand captures result and control streams independently.
func runUpdateUserTaskCommand(t *testing.T, configPath, stdin string, args ...string) (string, string, error) {
	t.Helper()
	encoded, err := json.Marshal(args)
	require.NoError(t, err)
	return testx.RunCmdSubprocessInDirWithSeparateOutputs(t, updateUserTaskCommandHelper, "", map[string]string{
		"C8VOLT_TEST_CONFIG":                configPath,
		"C8VOLT_TEST_UPDATE_USER_TASK_ARGS": string(encoded),
	}, stdin)
}

// newUpdateUserTaskCommandServer exposes the exact task, effective-variable,
// and scope-write routes used by the command workflow.
func newUpdateUserTaskCommandServer(t *testing.T) (*httptest.Server, *updateUserTaskCommandCapture) {
	t.Helper()
	capture := new(updateUserTaskCommandCapture)
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		capture.mu.Lock()
		capture.requests++
		capture.mu.Unlock()
		switch {
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2/user-tasks/"):
			key := strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/")
			capture.mu.Lock()
			capture.taskReads = append(capture.taskReads, key)
			capture.mu.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"userTaskKey":%q,"state":"CREATED","elementId":"approve_invoice","processInstanceKey":"2251799813711967","elementInstanceKey":"2251799815391200","processDefinitionKey":"2251799813689000","tenantId":"tenant-a"}`, key)
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/effective-variables/search"):
			key := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/"), "/effective-variables/search")
			capture.mu.Lock()
			capture.variableReads = append(capture.variableReads, key)
			mutated := capture.mutations > 0
			capture.mu.Unlock()
			value := "false"
			if mutated {
				value = "true"
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":%q,"variableKey":"901","processInstanceKey":"2251799813711967","scopeKey":"2251799815391100","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, value)
		case request.Method == http.MethodPut && strings.HasPrefix(request.URL.Path, "/v2/element-instances/"):
			var body map[string]any
			require.NoError(t, json.NewDecoder(request.Body).Decode(&body))
			capture.mu.Lock()
			capture.mutations++
			capture.mutationPaths = append(capture.mutationPaths, request.URL.Path)
			capture.mutationBody = append(capture.mutationBody, body)
			capture.mu.Unlock()
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, capture
}
