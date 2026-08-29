// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// requireSingleJSONObjectDocument decodes one complete JSON object and rejects
// any trailing document that would break machine-output consumers.
func requireSingleJSONObjectDocument(t *testing.T, output string) map[string]any {
	t.Helper()

	decoder := json.NewDecoder(strings.NewReader(output))
	var got map[string]any
	require.NoError(t, decoder.Decode(&got))
	var extra any
	require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
	return got
}

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

// TestTenantContextMachineOutputCleanlinessAcrossFamilies verifies tenant
// context keeps JSON, quiet, and key streams within their machine contracts.
func TestTenantContextMachineOutputCleanlinessAcrossFamilies(t *testing.T) {
	t.Run("run JSON is one document with root tenant context", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/v2/process-instances", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processDefinitionKey":"2251799813685255","processInstanceKey":"2251799813711967","tenantId":"tenant-a","variables":{}}`))
		}))
		t.Cleanup(srv.Close)
		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: "8.9"
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--automation",
			"--json",
			"run", "process-instance",
			"--pd-key", "2251799813685255",
			"--no-wait",
		)

		require.NotContains(t, stderr, "Create in tenant:")
		require.NotContains(t, stdout, "Create in tenant:")
		envelope := requireSingleJSONObjectDocument(t, stdout)
		tenantContext := requireJSONObject(t, envelope["tenantContext"])
		require.Equal(t, "creation", tenantContext["mode"])
		require.Equal(t, "not_applicable", tenantContext["filter"])
		require.Equal(t, "tenant-a", tenantContext["targetTenantId"])
		require.Equal(t, []any{"tenant-a"}, tenantContext["resolvedTenantIds"])
		payload := requireJSONObject(t, envelope["payload"])
		require.NotContains(t, payload, "tenantContext")
		requireJSONItems(t, payload["items"], 1)
	})

	t.Run("run quiet suppresses tenant context text", func(t *testing.T) {
		srv := newTenantContextMachineRunServer(t)
		t.Cleanup(srv.Close)

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"--quiet",
			"run", "process-instance",
			"--pd-key", "2251799813685255",
			"--no-wait",
		)

		require.Contains(t, stdout, "2251799813711967")
		require.NotContains(t, stdout, "Create in tenant:")
		require.NotContains(t, stderr, "Create in tenant:")
		require.NotContains(t, stdout, "tenantContext")
	})

	t.Run("run keys-only stdout is exact", func(t *testing.T) {
		srv := newTenantContextMachineRunServer(t)
		t.Cleanup(srv.Close)

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
			"run", "process-instance",
			"--pd-key", "2251799813685255",
			"--no-wait",
			"--keys-only",
		)

		require.Equal(t, "2251799813711967\n", stdout)
		require.NotContains(t, stderr, "Create in tenant:")
	})
}

// newTenantContextMachineRunServer returns a minimal process-instance creation
// endpoint for protected output-mode contract tests.
func newTenantContextMachineRunServer(t *testing.T) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"processDefinitionKey":"2251799813685255","processInstanceKey":"2251799813711967","tenantId":"<default>","variables":{}}`))
	}))
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
