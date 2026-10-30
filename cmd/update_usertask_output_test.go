// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// updateUserTaskOutputCapture records remote work independently of rendering.
type updateUserTaskOutputCapture struct {
	mu            sync.Mutex
	initialValue  bool
	failingScope  string
	mutatedScopes map[string]bool
	mutations     int
	taskReads     int
	variableReads int
	totalRequests int
}

// TestUpdateUserTaskOutputModes exercises dry-run, no-op, confirmed, and
// accepted execution through the command path across human and machine modes.
func TestUpdateUserTaskOutputModes(t *testing.T) {
	tests := []struct {
		name         string
		initialValue bool
		args         []string
		wantOutput   string
		wantOutcome  string
		wantKeys     string
		keysMode     bool
		humanMode    bool
		wantMutates  int
		wantReads    int
	}{
		{
			name:      "dry run human",
			humanMode: true,
			args:      []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--dry-run"},
			wantOutput: "INFO selection scope: explicit resource keys; tenant filter not applied\n" +
				"INFO affected tenants: tenant-a\n" +
				"INFO dry run: update user-task variables: 1 user task(s), 1 change(s), 0 addition(s), 0 unchanged, 0 untouched; no changes applied\n" +
				"INFO 101: ~ approved (inherited, element:201): false -> true\n",
			wantReads: 1,
		},
		{
			name:      "dry run keys",
			args:      []string{"--keys-only", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--dry-run"},
			wantKeys:  "101\n",
			keysMode:  true,
			wantReads: 1,
		},
		{
			name:        "dry run quiet json wins over keys",
			args:        []string{"--quiet", "--keys-only", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--dry-run", "--no-wait"},
			wantOutcome: "succeeded",
			wantReads:   1,
		},
		{
			name:         "no-op human",
			humanMode:    true,
			initialValue: true,
			args:         []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantOutput: "INFO selection scope: explicit resource keys; tenant filter not applied\n" +
				"INFO affected tenants: tenant-a\n" +
				"INFO plan: update user-task variables: nothing to update (1 requested value(s) already match visible variables); no confirmation required\n",
			wantReads: 1,
		},
		{
			name:         "no-op no-wait json remains succeeded",
			initialValue: true,
			args:         []string{"--automation", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait"},
			wantOutcome:  "succeeded",
			wantReads:    1,
		},
		{
			name:         "no-op keys is zero bytes",
			initialValue: true,
			args:         []string{"--keys-only", "update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantKeys:     "",
			keysMode:     true,
			wantReads:    1,
		},
		{
			name:      "confirmed human",
			humanMode: true,
			args:      []string{"--auto-confirm", "update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantOutput: "INFO updated user-task 101: confirmed\n" +
				"INFO updated: 1 (confirmed/submitted: 1, unchanged: 0, failed: 0, skipped: 0)\n",
			wantMutates: 1,
			wantReads:   2,
		},
		{
			name:        "accepted quiet keys",
			args:        []string{"--quiet", "--keys-only", "--automation", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait"},
			wantKeys:    "101\n",
			keysMode:    true,
			wantMutates: 1,
			wantReads:   1,
		},
		{
			name:        "accepted json",
			args:        []string{"--automation", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait"},
			wantOutcome: "accepted",
			wantMutates: 1,
			wantReads:   1,
		},
		{
			name:        "quiet human suppresses result",
			humanMode:   true,
			args:        []string{"--quiet", "--auto-confirm", "update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantOutput:  "",
			wantMutates: 1,
			wantReads:   2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capture := newUpdateUserTaskOutputServer(t, "8.10", test.initialValue, "")
			configPath := writeTestConfigForVersion(t, server.URL, "8.10")
			stdout, stderr, err := runUpdateUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err)
			switch {
			case test.wantOutcome != "":
				require.Empty(t, stderr)
				envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
				require.Equal(t, test.wantOutcome, envelope["outcome"])
			case test.keysMode:
				require.Empty(t, stderr)
				require.Equal(t, test.wantKeys, stdout)
			case test.humanMode:
				require.Empty(t, stdout)
				require.Equal(t, test.wantOutput, normalizeUpdateUserTaskHumanOutput(stderr))
			}

			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Equal(t, test.wantMutates, capture.mutations)
			require.Equal(t, 1, capture.taskReads)
			require.Equal(t, test.wantReads, capture.variableReads)
			require.Equal(t, 1+test.wantReads+test.wantMutates, capture.totalRequests)
		})
	}
}

// TestUpdateUserTaskDestinationMetadataFlowsThroughCommand proves planning
// supplies every destination kind without adding preview requests or writes.
func TestUpdateUserTaskDestinationMetadataFlowsThroughCommand(t *testing.T) {
	server, capture := newUpdateUserTaskOutputServer(t, "8.10", false, "")
	configPath := writeTestConfigForVersion(t, server.URL, "8.10")
	stdout, stderr, err := runUpdateUserTaskCommand(t, configPath, "",
		"update", "ut", "--key", "103", "--vars", `{"approved":true,"middle":2,"local":true,"new":"value"}`, "--dry-run",
	)
	require.NoError(t, err)
	require.Empty(t, stdout)
	normalized := normalizeUpdateUserTaskHumanOutput(stderr)
	require.Contains(t, normalized, "~ approved (inherited, pi:301): false -> true")
	require.Contains(t, normalized, "~ middle (inherited, element:203): 1 -> 2")
	require.Contains(t, normalized, "~ local (local, element:501103): false -> true")
	require.Contains(t, normalized, "+ new (local, element:501103): \"value\"")

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 1, capture.taskReads)
	require.Equal(t, 1, capture.variableReads)
	require.Zero(t, capture.mutations)
	require.Equal(t, 2, capture.totalRequests)
}

// TestUpdateUserTaskBulkDestinationPreviewsThroughCommand proves compact and
// verbose dry-run previews preserve typed destination order without extra work.
func TestUpdateUserTaskBulkDestinationPreviewsThroughCommand(t *testing.T) {
	tests := []struct {
		name             string
		keys             string
		variables        string
		verbose          bool
		wantSummary      string
		wantScopes       string
		wantDetails      []string
		unwantedDetails  []string
		wantTaskReads    int
		wantVariableRead int
	}{
		{
			name:             "compact mixed scopes deduplicate inherited destinations",
			keys:             "103,104",
			variables:        `{"approved":true,"middle":2,"local":true,"new":"value"}`,
			wantSummary:      "INFO dry run: update user-task variables: 2 user task(s), 6 change(s), 2 addition(s), 0 unchanged, 0 untouched; no changes applied\n",
			wantScopes:       "INFO inherited target scopes: pi:301, element:203\n",
			unwantedDetails:  []string{"INFO 103:", "INFO 104:"},
			wantTaskReads:    2,
			wantVariableRead: 2,
		},
		{
			name:        "verbose mixed scopes render every planned write",
			keys:        "103,104",
			variables:   `{"approved":true,"middle":2,"local":true,"new":"value"}`,
			verbose:     true,
			wantSummary: "INFO dry run: update user-task variables: 2 user task(s), 6 change(s), 2 addition(s), 0 unchanged, 0 untouched; no changes applied\n",
			wantScopes:  "INFO inherited target scopes: pi:301, element:203\n",
			wantDetails: []string{
				`INFO 103: ~ approved (inherited, pi:301): false -> true; ~ local (local, element:501103): false -> true; ~ middle (inherited, element:203): 1 -> 2; + new (local, element:501103): "value"`,
				`INFO 104: ~ approved (inherited, pi:301): false -> true; ~ local (local, element:501104): false -> true; ~ middle (inherited, element:203): 1 -> 2; + new (local, element:501104): "value"`,
			},
			wantTaskReads:    2,
			wantVariableRead: 2,
		},
		{
			name:             "compact local-only bulk omits inherited targets",
			keys:             "105,106",
			variables:        `{"local":true,"new":"value"}`,
			wantSummary:      "INFO dry run: update user-task variables: 2 user task(s), 2 change(s), 2 addition(s), 0 unchanged, 0 untouched; no changes applied\n",
			unwantedDetails:  []string{"inherited target scopes:", "INFO 105:", "INFO 106:"},
			wantTaskReads:    2,
			wantVariableRead: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capture := newUpdateUserTaskOutputServer(t, "8.10", false, "")
			configPath := writeTestConfigForVersion(t, server.URL, "8.10")
			args := []string{"update", "ut", "--key", test.keys, "--vars", test.variables, "--dry-run"}
			if test.verbose {
				args = append([]string{"--verbose"}, args...)
			}
			stdout, stderr, err := runUpdateUserTaskCommand(t, configPath, "", args...)
			require.NoError(t, err)
			require.Empty(t, stdout)
			normalized := normalizeUpdateUserTaskHumanOutput(stderr)
			require.Contains(t, normalized, test.wantSummary)
			if test.wantScopes != "" {
				require.Contains(t, normalized, test.wantScopes)
			}
			for _, detail := range test.wantDetails {
				require.Contains(t, normalized, detail)
			}
			for _, detail := range test.unwantedDetails {
				require.NotContains(t, normalized, detail)
			}

			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Equal(t, test.wantTaskReads, capture.taskReads)
			require.Equal(t, test.wantVariableRead, capture.variableReads)
			require.Zero(t, capture.mutations)
			require.Equal(t, test.wantTaskReads+test.wantVariableRead, capture.totalRequests)
		})
	}
}

// normalizeUpdateUserTaskHumanOutput removes logger timestamps while retaining
// severity, wording, ordering, and stream-routing assertions.
func normalizeUpdateUserTaskHumanOutput(output string) string {
	return regexp.MustCompile(`(?m)^\d{2}:\d{2}:\d{2}\.\d{3} (INFO|WARN) `).ReplaceAllString(output, `$1 `)
}

// TestUpdateUserTaskPartialFailureJSON renders accepted and failed task facts
// once in the normalized error envelope and exits with a failure code.
func TestUpdateUserTaskPartialFailureJSON(t *testing.T) {
	server, capture := newUpdateUserTaskOutputServer(t, "8.10", false, "202")
	configPath := writeTestConfigForVersion(t, server.URL, "8.10")
	stdout, _, err := runUpdateUserTaskCommand(t, configPath, "",
		"--automation", "--json", "update", "ut",
		"--key", "101,102", "--vars", `{"approved":true}`,
		"--no-wait", "--workers", "1",
	)
	require.Error(t, err)

	envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
	require.Equal(t, "failed", envelope["outcome"])
	require.NotNil(t, envelope["detail"])
	items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 2)
	require.Equal(t, "submitted", requireJSONObject(t, items[0])["status"])
	require.Equal(t, "mutation_failed", requireJSONObject(t, items[1])["status"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 2, capture.mutations)
	require.Equal(t, 2, capture.taskReads)
	require.Equal(t, 2, capture.variableReads, "no-wait performs planning reads only")
	require.Equal(t, 6, capture.totalRequests)
}

// decodeSingleUserTaskUpdateEnvelope requires exactly one JSON value and EOF.
func decodeSingleUserTaskUpdateEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewBufferString(output))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	return envelope
}

// newUpdateUserTaskOutputServer provides deterministic independent scopes so
// command output tests can observe no-op, acceptance, confirmation, and partial failure.
func newUpdateUserTaskOutputServer(t *testing.T, version string, initialValue bool, failingScope string) (*httptest.Server, *updateUserTaskOutputCapture) {
	t.Helper()
	capture := &updateUserTaskOutputCapture{
		initialValue:  initialValue,
		failingScope:  failingScope,
		mutatedScopes: make(map[string]bool),
	}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if serveVersionCheckTopology(writer, request, version) {
			return
		}

		capture.mu.Lock()
		capture.totalRequests++
		capture.mu.Unlock()
		switch {
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2/user-tasks/"):
			key := strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/")
			capture.mu.Lock()
			capture.taskReads++
			capture.mu.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"userTaskKey":%q,"state":"CREATED","elementId":"approve","processInstanceKey":"301","elementInstanceKey":%q,"processDefinitionKey":"401","tenantId":"tenant-a"}`, key, "501"+key)
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/effective-variables/search"):
			key := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/"), "/effective-variables/search")
			scope := updateUserTaskOutputScope(key)
			capture.mu.Lock()
			capture.variableReads++
			value := capture.initialValue || capture.mutatedScopes[scope]
			capture.mu.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			if key == "103" || key == "104" {
				_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":"false","variableKey":"601301","processInstanceKey":"301","scopeKey":"301","tenantId":"tenant-a"},{"name":"middle","value":"1","variableKey":"602203","processInstanceKey":"301","scopeKey":"203","tenantId":"tenant-a"},{"name":"local","value":"false","variableKey":%q,"processInstanceKey":"301","scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`, "603"+key, "501"+key)
				return
			}
			if key == "105" || key == "106" {
				_, _ = fmt.Fprintf(writer, `{"items":[{"name":"local","value":"false","variableKey":%q,"processInstanceKey":"301","scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, "603"+key, "501"+key)
				return
			}
			_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":%q,"variableKey":%q,"processInstanceKey":"301","scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, fmt.Sprint(value), "601"+key, scope)
		case request.Method == http.MethodPut && strings.HasPrefix(request.URL.Path, "/v2/element-instances/"):
			scope := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/element-instances/"), "/variables")
			capture.mu.Lock()
			capture.mutations++
			if scope != capture.failingScope {
				capture.mutatedScopes[scope] = true
			}
			capture.mu.Unlock()
			if scope == failingScope {
				http.Error(writer, `{"message":"mutation denied"}`, http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, capture
}

// updateUserTaskOutputScope maps selected task fixtures to distinct inherited scopes.
func updateUserTaskOutputScope(key string) string {
	if key == "102" {
		return "202"
	}
	return "201"
}
