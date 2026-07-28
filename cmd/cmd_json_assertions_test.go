// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func requireJSONObject(t *testing.T, value any) map[string]any {
	t.Helper()

	got, ok := value.(map[string]any)
	require.True(t, ok, "expected JSON object")
	return got
}

func requireJSONItems(t *testing.T, value any, wantLen int) []any {
	t.Helper()

	items, ok := value.([]any)
	require.True(t, ok, "expected JSON array")
	require.Len(t, items, wantLen)
	return items
}

// TestPagedSearchMachineOutputCleanliness verifies paged search progress never pollutes machine stdout.
func TestPagedSearchMachineOutputCleanliness(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStdout func(t *testing.T, stdout string)
	}{
		{
			name: "json",
			args: []string{"--json", "get", "job", "--batch-size", "2"},
			wantStdout: func(t *testing.T, stdout string) {
				t.Helper()
				var envelope map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
				payload := requireJSONObject(t, envelope["payload"])
				requireJSONItems(t, payload["items"], 3)
				require.NotContains(t, stdout, "page size:")
				require.NotContains(t, stdout, "found:")
			},
		},
		{
			name: "keys-only",
			args: []string{"--keys-only", "--auto-confirm", "get", "job", "--batch-size", "2"},
			wantStdout: func(t *testing.T, stdout string) {
				t.Helper()
				require.Equal(t, "2251799813711967\n2251799813711968\n2251799813711969\n", stdout)
			},
		},
		{
			name: "quiet verbose",
			args: []string{"--quiet", "--verbose", "--auto-confirm", "get", "job", "--batch-size", "2"},
			wantStdout: func(t *testing.T, stdout string) {
				t.Helper()
				require.Contains(t, stdout, "2251799813711967")
				require.Contains(t, stdout, "found: 3")
				require.NotContains(t, stdout, "page size:")
			},
		},
		{
			name: "automation json verbose",
			args: []string{"--automation", "--json", "--verbose", "get", "job", "--batch-size", "2"},
			wantStdout: func(t *testing.T, stdout string) {
				t.Helper()
				var envelope map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
				require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
				require.NotContains(t, stdout, "page size:")
			},
		},
		{
			name: "no-indicator json verbose",
			args: []string{"--no-indicator", "--json", "--verbose", "get", "job", "--batch-size", "2"},
			wantStdout: func(t *testing.T, stdout string) {
				t.Helper()
				var envelope map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
				require.Equal(t, "get job", envelope["command"])
				require.NotContains(t, stdout, "page size:")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodies []map[string]any
			srv := newJobSearchServerResponses(t, &bodies,
				`{"items":[{"jobKey":"2251799813711967","state":"FAILED","retries":0},{"jobKey":"2251799813711968","state":"FAILED","retries":1}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
				`{"items":[{"jobKey":"2251799813711969","state":"FAILED","retries":2}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
			)
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
			args := append([]string{"--config", cfgPath}, tt.args...)

			stdout, stderr := executeRootForJobWithSeparateOutputs(t, args...)

			require.Len(t, bodies, 2)
			require.Empty(t, stderr)
			tt.wantStdout(t, stdout)
		})
	}
}

// TestPagedProcessInstanceJSONAndKeysOnlyOutputCleanliness verifies process-instance paging keeps machine streams clean.
func TestPagedProcessInstanceJSONAndKeysOnlyOutputCleanliness(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		responses  []string
		wantStdout func(t *testing.T, stdout string)
	}{
		{
			name: "json",
			args: []string{"--json", "get", "process-instance", "--batch-size", "1"},
			responses: []string{
				`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"cursor-1"}}`,
				`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:01:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false,"startCursor":"cursor-1"}}`,
			},
			wantStdout: func(t *testing.T, stdout string) {
				t.Helper()
				var envelope map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
				payload := requireJSONObject(t, envelope["payload"])
				requireJSONItems(t, payload["items"], 2)
			},
		},
		{
			name: "keys-only",
			args: []string{"--keys-only", "get", "process-instance"},
			responses: []string{
				`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:01:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
			},
			wantStdout: func(t *testing.T, stdout string) {
				t.Helper()
				require.Equal(t, "123\n124\n", stdout)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests, tt.responses...)
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			args := append([]string{"--config", cfgPath}, tt.args...)

			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)

			require.NotEmpty(t, requests)
			require.Empty(t, stderr)
			require.NotContains(t, stdout, "process-instance search scope")
			require.NotContains(t, stdout, "discovering process instances")
			require.NotContains(t, stdout, "searching process instances")
			tt.wantStdout(t, stdout)
		})
	}
}

// executeRootForJobWithSeparateOutputs runs the root command and captures stdout and stderr independently.
func executeRootForJobWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	resetGetJobFlagState()
	resetUpdateJobFlagState()
	t.Cleanup(func() {
		resetGetJobFlagState()
		resetUpdateJobFlagState()
	})

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetJobFlagState()
	resetUpdateJobFlagState()

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return stdout.String(), stderr.String()
}
