// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestGetProcessDefinitionSelectionFlagsRemainSearchFilters ensures existing
// process-definition selectors still map into the facade filter unchanged.
func TestGetProcessDefinitionSelectionFlagsRemainSearchFilters(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	flagGetPDKey = "2251799813685255"
	flagGetPDBpmnProcessId = "invoice"
	flagGetPDProcessVersion = 3
	flagGetPDProcessVersionTag = "stable"
	flagGetPDLatest = true

	filter := populatePDSearchFilterOpts()

	require.Equal(t, process.ProcessDefinitionFilter{
		Key:               "2251799813685255",
		BpmnProcessId:     "invoice",
		ProcessVersion:    3,
		ProcessVersionTag: "stable",
	}, filter)
}

// TestGetProcessDefinitionNonWatchMachineModesStayCompatible keeps finite
// process-definition output modes unchanged while watch validation is added.
func TestGetProcessDefinitionNonWatchMachineModesStayCompatible(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		wantStdout   string
		wantNoStdout string
		serve        func(*testing.T, *[]map[string]any) *httptest.Server
	}{
		{
			name:       "json",
			args:       []string{"--json", "get", "process-definition"},
			wantStdout: `"outcome": "succeeded"`,
		},
		{
			name:       "keys only",
			args:       []string{"--keys-only", "get", "process-definition"},
			wantStdout: "2251799813685255\n",
		},
		{
			name:       "xml",
			args:       []string{"get", "process-definition", "--key", "2251799813685255", "--xml"},
			wantStdout: "<definitions id=\"invoice\"/>",
			serve: func(t *testing.T, requests *[]map[string]any) *httptest.Server {
				t.Helper()
				return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if serveVersionCheckTopology(w, r, "8.8") {
						return
					}

					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, "/v2/process-definitions/2251799813685255/xml", r.URL.Path)
					*requests = append(*requests, map[string]any{
						"method": r.Method,
						"path":   r.URL.Path,
					})
					w.Header().Set("Content-Type", "application/xml")
					_, _ = w.Write([]byte("<definitions id=\"invoice\"/>"))
				}))
			},
		},
		{
			name:       "quiet",
			args:       []string{"--quiet", "get", "process-definition"},
			wantStdout: "2251799813685255",
		},
		{
			name:       "automation",
			args:       []string{"--automation", "get", "process-definition"},
			wantStdout: "found: 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []map[string]any
			serve := tt.serve
			if serve == nil {
				serve = func(t *testing.T, requests *[]map[string]any) *httptest.Server {
					t.Helper()
					return newProcessDefinitionSearchServerResponses(t, "8.8", requests,
						`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
					)
				}
			}
			srv := serve(t, &requests)
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			args := append([]string{"--config", cfgPath}, tt.args...)

			stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)

			require.Len(t, requests, 1)
			require.Empty(t, stderr)
			require.Contains(t, stdout, tt.wantStdout)
			if tt.wantNoStdout != "" {
				require.NotContains(t, stdout, tt.wantNoStdout)
			}
		})
	}
}

// TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle verifies ordinary
// process-definition paths remain independent from watch timing and repaint
// behavior, even if stale watch interval state would be invalid for watch mode.
func TestGetProcessDefinitionBaseDispatchSkipsWatchLifecycle(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		wantRequest      string
		responseType     string
		responseBody     string
		wantStdout       string
		wantNoStdout     string
		wantRequestCount int
	}{
		{
			name:             "list",
			args:             []string{"get", "process-definition"},
			wantRequest:      "POST /v2/process-definitions/search",
			responseType:     "application/json",
			responseBody:     `{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
			wantStdout:       "2251799813685255 tenant invoice v3/stable\nfound: 1\n",
			wantRequestCount: 1,
		},
		{
			name:             "key lookup",
			args:             []string{"get", "process-definition", "--key", "2251799813685255"},
			wantRequest:      "GET /v2/process-definitions/2251799813685255",
			responseType:     "application/json",
			responseBody:     `{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}`,
			wantStdout:       "2251799813685255 tenant invoice v3/stable\n",
			wantNoStdout:     "found:",
			wantRequestCount: 1,
		},
		{
			name:             "xml lookup",
			args:             []string{"get", "process-definition", "--key", "2251799813685255", "--xml"},
			wantRequest:      "GET /v2/process-definitions/2251799813685255/xml",
			responseType:     "application/xml",
			responseBody:     "<definitions id=\"invoice\"/>",
			wantStdout:       "<definitions id=\"invoice\"/>",
			wantRequestCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.8") {
					return
				}

				request := r.Method + " " + r.URL.Path
				requests = append(requests, request)
				require.Equal(t, tt.wantRequest, request)
				w.Header().Set("Content-Type", tt.responseType)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			args := append([]string{"--config", cfgPath}, tt.args...)

			stdout, stderr := executeRootForProcessDefinitionBaseDispatchTest(t, args...)

			require.Equal(t, tt.wantRequestCount, len(requests))
			require.Equal(t, tt.wantStdout, stdout)
			if tt.wantNoStdout != "" {
				require.NotContains(t, stdout, tt.wantNoStdout)
			}
			requireNoProcessDefinitionWatchLifecycleOutput(t, stdout, stderr)
		})
	}
}

// TestGetProcessDefinitionLatestSearchPreservesSelectionRequest preserves the existing command contract with advisory-aware fixtures.
func TestGetProcessDefinitionLatestSearchPreservesSelectionRequest(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.8") {
			return
		}

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionLatestSearchPreservesSelectionRequestHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.NoError(t, err, string(output))
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "invoice", filter["processDefinitionId"])
	require.Equal(t, float64(3), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Equal(t, true, filter["isLatestVersion"])
}

// TestGetProcessDefinitionBroadLatestUsesPagedCanonicalDiscovery verifies
// broad latest listings use the command's paged collection path with batch size.
func TestGetProcessDefinitionBroadLatestUsesPagedCanonicalDiscovery(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	flagGetPDLatest = true
	flagGetPDBatchSize = 2

	cmd := &cobra.Command{Use: "process-definition"}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetContext(context.Background())
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cli := processDefinitionPagingActivityAPI{
		searchProcessDefinitionsPages: func(_ context.Context, request process.ProcessDefinitionSearchRequest, visitor process.ProcessDefinitionSearchPageVisitor, opts ...options.FacadeOption) (process.ProcessDefinitionSearchPagesResult, error) {
			require.True(t, request.Latest)
			require.Equal(t, int32(2), request.Page.Size)
			require.Equal(t, process.ProcessDefinitionFilter{}, request.Filter)
			require.NotNil(t, visitor)
			action, err := visitor(process.ProcessDefinitionSearchPageStep{
				Page: process.ProcessDefinitionPage{
					Request: request.Page,
					Items: []process.ProcessDefinition{
						{Key: "tenant-a-order", TenantId: "tenant-a", BpmnProcessId: "order", ProcessVersion: 3},
					},
				},
				CumulativeCount: 1,
			})
			require.NoError(t, err)
			require.Equal(t, process.ProcessDefinitionSearchPageActionContinue, action)
			return process.ProcessDefinitionSearchPagesResult{
				Items: []process.ProcessDefinition{
					{Key: "default-order", TenantId: "<default>", BpmnProcessId: "order", ProcessVersion: 4},
					{Key: "tenant-a-order", TenantId: "tenant-a", BpmnProcessId: "order", ProcessVersion: 3},
				},
				Pages: 1,
			}, nil
		},
	}

	runSearchProcessDefinitions(cmd, cli, slog.Default(), true, process.ProcessDefinitionFilter{})

	require.Empty(t, stderr.String())
	require.Equal(t, []string{"default-order", "tenant-a-order"}, processDefinitionRenderedKeys(t, stdout.String()))
	require.Contains(t, stdout.String(), "found: 2")
}

// TestGetProcessDefinitionLatestSearchPageSizeInvarianceAcrossTenants checks
// CLI latest discovery keeps one canonical sequence across discovery page sizes.
func TestGetProcessDefinitionLatestSearchPageSizeInvarianceAcrossTenants(t *testing.T) {
	backendItems := []map[string]any{
		{"processDefinitionKey": "tenant-b-invoice-v11", "processDefinitionId": "invoice", "name": "invoice", "version": 11, "tenantId": "tenant-b"},
		{"processDefinitionKey": "2", "processDefinitionId": "invoice", "name": "invoice", "version": 10, "tenantId": "tenant-a"},
		{"processDefinitionKey": "tenant-a-invoice-v9", "processDefinitionId": "invoice", "name": "invoice", "version": 9, "tenantId": "tenant-a"},
		{"processDefinitionKey": "default-order-v10", "processDefinitionId": "order", "name": "order", "version": 10, "tenantId": "<default>"},
		{"processDefinitionKey": "tenant-a-Invoice-v10", "processDefinitionId": "Invoice", "name": "Invoice", "version": 10, "tenantId": "tenant-a"},
		{"processDefinitionKey": "10", "processDefinitionId": "invoice", "name": "invoice", "version": 10, "tenantId": "tenant-a"},
	}
	wantKeys := []string{
		"default-order-v10",
		"tenant-a-Invoice-v10",
		"10",
		"tenant-b-invoice-v11",
	}

	for _, pageSize := range []int{1, 2, 1000} {
		t.Run(strconv.Itoa(pageSize), func(t *testing.T) {
			var requests []map[string]any
			servedItems := 0
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.9") {
					return
				}

				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
				var request map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&request))
				requests = append(requests, request)

				filter := requireJSONObject(t, request["filter"])
				require.Equal(t, true, filter["isLatestVersion"])
				require.NotContains(t, filter, "tenantId")
				page := requireJSONObject(t, request["page"])
				require.Equal(t, float64(pageSize), page["limit"])

				start := servedItems
				end := min(start+pageSize, len(backendItems))
				servedItems = end
				responsePage := map[string]any{
					"totalItems":        len(backendItems),
					"hasMoreTotalItems": end < len(backendItems),
				}
				if end < len(backendItems) {
					responsePage["endCursor"] = strconv.Itoa(end)
				}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{
					"items": backendItems[start:end],
					"page":  responsePage,
				}))
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeRawTestConfig(t, `
app:
  camunda_version: "8.9"
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: "`+srv.URL+`"
`)

			stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
				"--config", cfgPath,
				"--all-tenants",
				"--keys-only",
				"get", "process-definition",
				"--latest",
				"--batch-size", strconv.Itoa(pageSize),
			)

			require.Empty(t, stderr)
			require.Equal(t, strings.Join(wantKeys, "\n")+"\n", stdout)
			require.Equal(t, (len(backendItems)+pageSize-1)/pageSize, len(requests))
		})
	}
}

// TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnostic preserves the existing command contract with advisory-aware fixtures.
func TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnostic(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.8") {
			return
		}

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		writeEmptyProcessDefinitionSearchResponse(w)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnosticHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Len(t, requests, 1)
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "missing-process", filter["processDefinitionId"])
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[missing-process]")
}

// TestGetProcessDefinitionBpmnSelectorVisiblePreservesListing preserves the existing command contract with advisory-aware fixtures.
func TestGetProcessDefinitionBpmnSelectorVisiblePreservesListing(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, "8.8") {
			return
		}

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"order-process","name":"Order Process","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	output := executeRootForTest(t,
		"--config", cfgPath,
		"get", "process-definition",
		"--bpmn-process-id", "order-process",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
	)

	require.Len(t, requests, 1)
	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, float64(3), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Contains(t, output, "2251799813685255")
	require.Contains(t, output, "tenant order-process v3/stable")
}

// TestGetProcessDefinitionSearchRendersCanonicalOrderForTenantScopes verifies
// tenant-filtered and all-tenant command listings preserve the service-owned
// process-definition collection order through human rendering.
func TestGetProcessDefinitionSearchRendersCanonicalOrderForTenantScopes(t *testing.T) {
	tests := []struct {
		name           string
		configPath     func(*testing.T, string) string
		args           func(string) []string
		responses      []string
		wantKeys       []string
		assertRequests func(*testing.T, []map[string]any)
	}{
		{
			name: "configured tenant filter",
			configPath: func(t *testing.T, baseURL string) string {
				t.Helper()
				return writeTestConfigForVersion(t, baseURL, "8.9")
			},
			args: func(cfgPath string) []string {
				return []string{"--config", cfgPath, "--tenant", "tenant-a", "get", "process-definition"}
			},
			responses: []string{
				`{"items":[{"processDefinitionKey":"tenant-a-payment-v1","processDefinitionId":"payment","name":"payment","version":1,"tenantId":"tenant-a"},{"processDefinitionKey":"tenant-a-invoice-v9","processDefinitionId":"invoice","name":"invoice","version":9,"tenantId":"tenant-a"}],"page":{"totalItems":5,"hasMoreTotalItems":true,"endCursor":"pd-page-2"}}`,
				`{"items":[{"processDefinitionKey":"2","processDefinitionId":"invoice","name":"invoice","version":10,"tenantId":"tenant-a"},{"processDefinitionKey":"tenant-a-Invoice-v1","processDefinitionId":"Invoice","name":"Invoice","version":1,"tenantId":"tenant-a"},{"processDefinitionKey":"10","processDefinitionId":"invoice","name":"invoice","version":10,"tenantId":"tenant-a"}],"page":{"totalItems":5,"hasMoreTotalItems":false}}`,
			},
			wantKeys: []string{
				"tenant-a-Invoice-v1",
				"10",
				"2",
				"tenant-a-invoice-v9",
				"tenant-a-payment-v1",
			},
			assertRequests: func(t *testing.T, requests []map[string]any) {
				t.Helper()
				require.Len(t, requests, 2)
				for _, request := range requests {
					filter := requireJSONObject(t, request["filter"])
					require.Equal(t, "tenant-a", filter["tenantId"])
				}
			},
		},
		{
			name: "all tenants clears configured tenant filter",
			configPath: func(t *testing.T, baseURL string) string {
				t.Helper()
				return writeRawTestConfig(t, `
app:
  camunda_version: "8.9"
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: "`+baseURL+`"
`)
			},
			args: func(cfgPath string) []string {
				return []string{"--config", cfgPath, "--all-tenants", "get", "process-definition"}
			},
			responses: []string{
				`{"items":[{"processDefinitionKey":"tenant-b-invoice-v1","processDefinitionId":"invoice","name":"invoice","version":1,"tenantId":"tenant-b"},{"processDefinitionKey":"tenant-a-payment-v1","processDefinitionId":"payment","name":"payment","version":1,"tenantId":"tenant-a"},{"processDefinitionKey":"tenant-a-invoice-v9","processDefinitionId":"invoice","name":"invoice","version":9,"tenantId":"tenant-a"}],"page":{"totalItems":7,"hasMoreTotalItems":true,"endCursor":"pd-page-2"}}`,
				`{"items":[{"processDefinitionKey":"2","processDefinitionId":"invoice","name":"invoice","version":10,"tenantId":"tenant-a"},{"processDefinitionKey":"default-invoice-v10","processDefinitionId":"invoice","name":"invoice","version":10,"tenantId":"<default>"},{"processDefinitionKey":"tenant-a-Invoice-v1","processDefinitionId":"Invoice","name":"Invoice","version":1,"tenantId":"tenant-a"},{"processDefinitionKey":"10","processDefinitionId":"invoice","name":"invoice","version":10,"tenantId":"tenant-a"}],"page":{"totalItems":7,"hasMoreTotalItems":false}}`,
			},
			wantKeys: []string{
				"default-invoice-v10",
				"tenant-a-Invoice-v1",
				"10",
				"2",
				"tenant-a-invoice-v9",
				"tenant-a-payment-v1",
				"tenant-b-invoice-v1",
			},
			assertRequests: func(t *testing.T, requests []map[string]any) {
				t.Helper()
				require.Len(t, requests, 2)
				for _, request := range requests {
					filter := requireJSONObject(t, request["filter"])
					require.NotContains(t, filter, "tenantId")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []map[string]any
			srv := newProcessDefinitionSearchServerResponses(t, "8.9", &requests, tt.responses...)
			t.Cleanup(srv.Close)
			cfgPath := tt.configPath(t, srv.URL)

			stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, tt.args(cfgPath)...)

			require.Empty(t, stderr)
			require.Equal(t, tt.wantKeys, processDefinitionRenderedKeys(t, stdout))
			require.Contains(t, stdout, "found:")
			tt.assertRequests(t, requests)
		})
	}
}

// TestGetProcessDefinitionSearchVerboseProgress defines the process-definition progress contract for broad listing.
func TestGetProcessDefinitionSearchVerboseProgress(t *testing.T) {
	var requests []map[string]any
	srv := newProcessDefinitionSearchServerResponses(t, "8.9", &requests,
		`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"},{"processDefinitionKey":"2251799813685256","processDefinitionId":"payment","name":"payment","version":2,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"pd-page-2"}}`,
		`{"items":[{"processDefinitionKey":"2251799813685257","processDefinitionId":"shipping","name":"shipping","version":1,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
		"--config", cfgPath,
		"--verbose",
		"--auto-confirm",
		"get", "process-definition",
		"--batch-size", "2",
	)

	require.Len(t, requests, 2)
	firstPage := requireJSONObject(t, requests[0]["page"])
	require.Equal(t, float64(2), firstPage["limit"])
	require.Equal(t, float64(0), firstPage["from"])
	secondPage := requireJSONObject(t, requests[1]["page"])
	require.Equal(t, "pd-page-2", secondPage["after"])
	require.Contains(t, stderr, "process-definition search scope: matched at least 3 process definitions; page size: 2; discovery pages: at least 2")
	require.Contains(t, stderr, "discovering process definitions, page 1/~2, 2 seen")
	require.Contains(t, stderr, "discovering process definitions, page 2/2, 3 seen")
	require.Contains(t, stdout, "2251799813685255")
	require.Contains(t, stdout, "2251799813685256")
	require.Contains(t, stdout, "2251799813685257")
	require.Contains(t, stdout, "found: 3")
}

// TestGetProcessDefinitionSearchMachineOutputStaysProgressFree protects process-definition JSON/key streams during progress rollout.
func TestGetProcessDefinitionSearchMachineOutputStaysProgressFree(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var requests []map[string]any
		srv := newProcessDefinitionSearchServerResponses(t, "8.9", &requests,
			`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--json",
			"get", "process-definition",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process definitions")
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
		payload := requireJSONObject(t, envelope["payload"])
		items := payload["items"].([]any)
		require.Len(t, items, 1)
	})

	t.Run("keys only", func(t *testing.T) {
		var requests []map[string]any
		srv := newProcessDefinitionSearchServerResponses(t, "8.9", &requests,
			`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"},{"processDefinitionKey":"2251799813685256","processDefinitionId":"payment","name":"payment","version":2,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t,
			"--config", cfgPath,
			"--keys-only",
			"get", "process-definition",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.Equal(t, "2251799813685255\n2251799813685256\n", stdout)
		require.NotContains(t, stdout, "scope:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process definitions")
	})
}

func TestGetProcessDefinitionXMLOutputRemainsKeyOnlyDisplayMode(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*process.ProcessDefinitionFilter)
		assert func(*testing.T, error)
	}{
		{
			name: "key only accepted",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
			},
			assert: func(t *testing.T, err error) {
				require.NoError(t, err)
			},
		},
		{
			name:  "missing key",
			setup: func(*process.ProcessDefinitionFilter) {},
			assert: func(t *testing.T, err error) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "xml output requires --key")
			},
		},
		{
			name: "bpmn process id",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				filter.BpmnProcessId = "invoice"
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--bpmn-process-id"),
		},
		{
			name: "process version",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDProcessVersion = 3
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--pd-version"),
		},
		{
			name: "process version tag",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				filter.ProcessVersionTag = "stable"
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--pd-version-tag"),
		},
		{
			name: "latest",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDLatest = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--latest"),
		},
		{
			name: "stat",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagGetPDWithStat = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--stat"),
		},
		{
			name: "json",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagViewAsJson = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--json"),
		},
		{
			name: "keys only",
			setup: func(filter *process.ProcessDefinitionFilter) {
				filter.Key = "2251799813685255"
				flagViewKeysOnly = true
			},
			assert: requireXMLDisplayModeIncompatibleFlag("--keys-only"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)

			var filter process.ProcessDefinitionFilter
			tt.setup(&filter)
			tt.assert(t, validateProcessDefinitionXMLFlags(filter))
		})
	}
}

func requireXMLDisplayModeIncompatibleFlag(flag string) func(*testing.T, error) {
	return func(t *testing.T, err error) {
		t.Helper()
		require.Error(t, err)
		require.Contains(t, err.Error(), "xml output only supports --key")
		require.Contains(t, err.Error(), flag)
	}
}

func TestSearchProcessDefinitionsWithPagingStatUsesCommandActivity(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	flagGetPDWithStat = true

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	cli := processDefinitionPagingActivityAPI{
		searchProcessDefinitionsPages: func(ctx context.Context, request process.ProcessDefinitionSearchRequest, visitor process.ProcessDefinitionSearchPageVisitor, opts ...options.FacadeOption) (process.ProcessDefinitionSearchPagesResult, error) {
			require.Equal(t, int32(1000), request.Page.Size)
			require.True(t, options.ApplyFacadeOptions(opts).Stat)
			return process.ProcessDefinitionSearchPagesResult{
				Items: []process.ProcessDefinition{{
					Key:           "2251799813685255",
					BpmnProcessId: "order-process",
					TenantId:      "<default>",
				}},
				Pages: 1,
			}, nil
		},
	}

	result, err := searchProcessDefinitionsWithPaging(cmd, cli, process.ProcessDefinitionFilter{})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, []activitysink.Start{{
		Message:    "loading process-definition statistics",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.Starts())
	require.Equal(t, 1, sink.Stopped())
}

type processDefinitionPagingActivityAPI struct {
	c8volt.API
	searchProcessDefinitionsPages func(context.Context, process.ProcessDefinitionSearchRequest, process.ProcessDefinitionSearchPageVisitor, ...options.FacadeOption) (process.ProcessDefinitionSearchPagesResult, error)
}

func (a processDefinitionPagingActivityAPI) SearchProcessDefinitionsPages(ctx context.Context, request process.ProcessDefinitionSearchRequest, visitor process.ProcessDefinitionSearchPageVisitor, opts ...options.FacadeOption) (process.ProcessDefinitionSearchPagesResult, error) {
	return a.searchProcessDefinitionsPages(ctx, request, visitor, opts...)
}

func resetGetProcessDefinitionCommandGlobals() {
	flagGetPDKey = ""
	flagGetPDBpmnProcessId = ""
	flagGetPDProcessVersion = 0
	flagGetPDProcessVersionTag = ""
	flagGetPDLatest = false
	flagGetPDWithStat = false
	flagGetPDAsXML = false
	flagGetPDBatchSize = 0
	flagGetPDWatch = false
	flagGetPDWatchInterval = defaultGetPDWatchInterval.String()
	flagGetPDWatchTimeout = 0
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagQuiet = false
	flagVerbose = false
	flagDebug = false
	flagCmdAutomation = false
	flagAllTenants = false
}

// marshalStringSliceForEnv keeps subprocess argument fixtures shell-safe.
func marshalStringSliceForEnv(t *testing.T, items []string) string {
	t.Helper()

	data, err := json.Marshal(items)
	require.NoError(t, err)
	return string(data)
}

// processDefinitionRenderedKeys extracts process-definition keys from compact
// human listing output while ignoring the trailing found summary line.
func processDefinitionRenderedKeys(t *testing.T, output string) []string {
	t.Helper()

	keys := []string{}
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "found:") {
			continue
		}
		fields := strings.Fields(line)
		require.NotEmpty(t, fields, "expected process-definition row")
		keys = append(keys, fields[0])
	}
	return keys
}

func TestGetProcessDefinitionLatestSearchPreservesSelectionRequestHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--json",
		"get", "process-definition",
		"--bpmn-process-id", "invoice",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
		"--latest",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetProcessDefinitionBpmnSelectorMissingFailsWithExplicitDiagnosticHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"get", "process-definition",
		"--bpmn-process-id", "missing-process",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func executeRootForProcessDefinitionTestWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return stdout.String(), stderr.String()
}

// executeRootForProcessDefinitionBaseDispatchTest keeps watch-only globals
// hostile so base dispatch tests fail if ordinary paths validate watch state.
func executeRootForProcessDefinitionBaseDispatchTest(t *testing.T, args ...string) (string, string) {
	t.Helper()

	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	flagGetPDWatchInterval = "not-a-duration"

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return stdout.String(), stderr.String()
}

// requireNoProcessDefinitionWatchLifecycleOutput keeps watch repaint, retry, and
// stop status text out of ordinary process-definition command output.
func requireNoProcessDefinitionWatchLifecycleOutput(t *testing.T, stdout, stderr string) {
	t.Helper()

	require.NotContains(t, stdout, processDefinitionWatchRepaintControlSequenceForTest)
	require.NotContains(t, stdout, "process-definition watch")
	require.NotContains(t, stdout, "watch stopped")
	require.NotContains(t, stderr, "process-definition watch")
	require.NotContains(t, stderr, "watch stopped")
	require.Empty(t, stderr)
}

// newProcessDefinitionSearchServerResponses keeps fixture setup aligned with the invocation's advisory version check.
func newProcessDefinitionSearchServerResponses(t *testing.T, version string, requests *[]map[string]any, responses ...string) *httptest.Server {
	t.Helper()

	served := 0
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if serveVersionCheckTopology(w, r, version) {
			return
		}

		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		var request map[string]any
		require.NoError(t, json.Unmarshal(body, &request))
		*requests = append(*requests, request)
		require.Less(t, served, len(responses), "unexpected extra process-definition search request")

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(responses[served]))
		served++
	}))
}

func countProcessDefinitionSearchRequests(items []string) int {
	count := 0
	for _, item := range items {
		if strings.HasPrefix(item, "POST /v2/process-definitions/search ") {
			count++
		}
	}
	return count
}

// TestGetProcessDefinitionLifecycle exercises real command reads and output modes
// against a single expected fake request, so lifecycle cannot add discovery or mutations.
// Adapter tests cover the full value vocabulary; mixed-row and mode tests cover
// unfamiliar/DELETED values, while this table checks both read paths and omission.
func TestGetProcessDefinitionLifecycle(t *testing.T) {
	for _, lookup := range []bool{true, false} {
		for _, state := range []string{"DRAINING", ""} {
			for _, mode := range []string{"human", "json"} {
				t.Run(strconv.FormatBool(lookup)+"/"+state+"/"+mode, func(t *testing.T) {
					item := map[string]any{"processDefinitionKey": "123", "processDefinitionId": "orders", "version": 2, "tenantId": "tenant"}
					if state != "" {
						item["state"] = state
					}
					var payload any = item
					method, path := http.MethodGet, "/v2/process-definitions/123"
					args := []string{"get", "pd"}
					if lookup {
						args = append(args, "--key", "123")
					} else {
						method, path = http.MethodPost, "/v2/process-definitions/search"
						payload = map[string]any{"items": []any{item}, "page": map[string]any{"totalItems": 1, "hasMoreTotalItems": false}}
					}
					if mode == "json" {
						args = append(args, "--json")
					}
					var requests atomic.Int32
					srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if serveVersionCheckTopology(w, r, "8.10") {
							return
						}

						requests.Add(1)
						require.Equal(t, method, r.Method)
						require.Equal(t, path, r.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						require.NoError(t, json.NewEncoder(w).Encode(payload))
					}))
					t.Cleanup(srv.Close)
					cfg := writeTestConfigForVersion(t, srv.URL, "8.10")
					stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, append([]string{"--config", cfg}, args...)...)
					require.EqualValues(t, 1, requests.Load())
					require.Empty(t, stderr)
					if mode == "json" {
						decoder := json.NewDecoder(strings.NewReader(stdout))
						var envelope map[string]any
						require.NoError(t, decoder.Decode(&envelope))
						require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
						require.Equal(t, "succeeded", envelope["outcome"])
						got := requireJSONObject(t, envelope["payload"])
						if !lookup {
							got = requireJSONObject(t, requireJSONItems(t, got["items"], 1)[0])
						}
						require.Equal(t, "123", got["key"])
						if state == "" {
							require.NotContains(t, got, "state")
						} else {
							require.Equal(t, state, got["state"])
						}
					} else {
						want := "123 tenant orders v2"
						if state != "" {
							want += " " + state
						}
						// Existing human result renderers remove trailing row whitespace.
						want = strings.TrimRight(want, " ")
						want += "\n"
						if !lookup {
							want += "found: 1\n"
						}
						require.Equal(t, want, stdout)
					}
				})
			}
		}
	}
}

// TestGetProcessDefinitionLifecycleModes keeps quiet/machine precedence and empty
// payload conventions stable with lifecycle present, using only one search request.
func TestGetProcessDefinitionLifecycleModes(t *testing.T) {
	for _, empty := range []bool{false, true} {
		for _, mode := range []struct {
			name              string
			flags             []string
			json, keys, quiet bool
		}{
			{"human", nil, false, false, false},
			{"quiet", []string{"--quiet"}, false, false, true},
			{"json", []string{"--json"}, true, false, false},
			{"keys", []string{"--keys-only"}, false, true, false},
			{"quiet json", []string{"--quiet", "--json"}, true, false, true},
			{"quiet keys", []string{"--quiet", "--keys-only"}, false, true, true},
			{"automation", []string{"--automation"}, false, false, false},
			{"auto-confirm", []string{"--auto-confirm"}, false, false, false},
		} {
			t.Run(strconv.FormatBool(empty)+"/"+mode.name, func(t *testing.T) {
				body := `{"items":[{"processDefinitionKey":"123","processDefinitionId":"orders","version":2,"state":"DELETED"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`
				if empty {
					body = `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
				}
				var requests []map[string]any
				srv := newProcessDefinitionSearchServerResponses(t, "8.10", &requests, body)
				t.Cleanup(srv.Close)
				cfg := writeTestConfigForVersion(t, srv.URL, "8.10")
				args := append([]string{"--config", cfg, "get", "pd"}, mode.flags...)
				stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)
				require.Len(t, requests, 1)
				require.Empty(t, stderr)
				if mode.json {
					decoder := json.NewDecoder(strings.NewReader(stdout))
					var envelope map[string]any
					require.NoError(t, decoder.Decode(&envelope))
					require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
					require.Equal(t, "succeeded", envelope["outcome"])
					payload := requireJSONObject(t, envelope["payload"])
					if empty {
						require.Empty(t, payload, "preserve omitted empty fields")
					} else {
						item := requireJSONObject(t, requireJSONItems(t, payload["items"], 1)[0])
						require.Equal(t, "DELETED", item["state"])
					}
				} else {
					want := "123 orders v2 DELETED\nfound: 1\n"
					if mode.keys {
						want = "123\n"
					}
					if empty {
						want = "found: 0\n"
						if mode.keys || mode.quiet {
							want = ""
						}
					}
					require.Equal(t, want, stdout)
				}
			})
		}
	}
}

// TestGetProcessDefinitionLifecycleMixedRows preserves aligned mixed availability
// and unchanged legacy rows/JSON for both older adapters through command execution.
func TestGetProcessDefinitionLifecycleMixedRows(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, asJSON := range []bool{false, true} {
			t.Run(version+"/"+strconv.FormatBool(asJSON), func(t *testing.T) {
				var requests []map[string]any
				srv := newProcessDefinitionSearchServerResponses(t, version, &requests,
					`{"items":[{"processDefinitionKey":"1","processDefinitionId":"orders","version":3,"state":"ACTIVE"},{"processDefinitionKey":"2","processDefinitionId":"orders","version":2},{"processDefinitionKey":"3","processDefinitionId":"orders","version":1,"state":"FUTURE"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`)
				t.Cleanup(srv.Close)
				cfg := writeTestConfigForVersion(t, srv.URL, version)
				args := []string{"--config", cfg, "get", "pd"}
				if asJSON {
					args = append(args, "--json")
				}
				stdout, stderr := executeRootForProcessDefinitionTestWithSeparateOutputs(t, args...)
				require.Empty(t, stderr)
				require.Len(t, requests, 1)
				if asJSON {
					decoder := json.NewDecoder(strings.NewReader(stdout))
					var envelope map[string]any
					require.NoError(t, decoder.Decode(&envelope))
					require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
					require.Equal(t, "succeeded", envelope["outcome"])
					items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 3)
					for i, raw := range items {
						item := requireJSONObject(t, raw)
						if version != "8.10" || i == 1 {
							require.NotContains(t, item, "state")
						} else {
							require.Equal(t, []string{"ACTIVE", "", "FUTURE"}[i], item["state"])
						}
					}
				} else if version == "8.10" {
					require.Equal(t, "1 orders v3 ACTIVE\n2 orders v2\n3 orders v1 FUTURE\nfound: 3\n", stdout)
				} else {
					require.Equal(t, "1 orders v3\n2 orders v2\n3 orders v1\nfound: 3\n", stdout)
				}
			})
		}
	}
}

// TestGetProcessDefinitionLifecycleNotFound distinguishes an absent key from a
// returned DELETED item, retaining the failure exit and single JSON envelope.
func TestGetProcessDefinitionLifecycleNotFound(t *testing.T) {
	for _, mode := range []string{"human", "json"} {
		t.Run(mode, func(t *testing.T) {
			var requests atomic.Int32
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if serveVersionCheckTopology(w, r, "8.10") {
					return
				}

				requests.Add(1)
				require.Equal(t, http.MethodGet, r.Method)
				require.Equal(t, "/v2/process-definitions/123", r.URL.Path)
				w.WriteHeader(http.StatusNotFound)
			}))
			t.Cleanup(srv.Close)
			cfg := writeTestConfigForVersion(t, srv.URL, "8.10")
			stdout, stderr := runCommandErrorEnvelopeSubprocess(t, processDefinitionErrorEnvelopeHelper, "", map[string]string{
				"C8VOLT_TEST_CONFIG":       cfg,
				"C8VOLT_TEST_OUTPUT_MODE":  mode,
				"C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, []string{"--log-format", "json", "get", "pd", "--key", "123"}),
			}, "", exitcode.NotFound)
			require.EqualValues(t, 1, requests.Load())
			if mode == "json" {
				require.Empty(t, stderr)
				decoder := json.NewDecoder(strings.NewReader(stdout))
				var envelope map[string]any
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
				require.Equal(t, "failed", envelope["outcome"])
				require.Equal(t, "not_found", envelope["class"])
			} else {
				require.Empty(t, stdout)
				decoder := json.NewDecoder(strings.NewReader(stderr))
				var record map[string]any
				require.NoError(t, decoder.Decode(&record))
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
				require.Equal(t, "ERROR", record["level"])
				require.Equal(t, "get process definition: not found", record["msg"])
				require.NotContains(t, stderr, srv.URL)
				require.NotContains(t, stderr, "GET")
			}
		})
	}
}

// TestGetProcessDefinitionLifecycleLookupDiagnostics exercises bounded failures
// through configured loggers, preserving machine evidence and redacted DEBUG only.
func TestGetProcessDefinitionLifecycleLookupDiagnostics(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		for _, route := range []string{"inherited", "configured"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "json", "debug-flag", "debug-config"} {
				t.Run(format+"/"+route+"/"+mode, func(t *testing.T) {
					var requests atomic.Int32
					srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if serveVersionCheckTopology(w, r, "8.10") {
							return
						}

						requests.Add(1)
						require.Equal(t, "GET /v2/process-definitions/8675309", r.Method+" "+r.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(http.StatusNotFound)
						_, _ = w.Write([]byte(`{"message":"definition-body GET https://definition-secret.invalid/8675309","access_token":"definition-token-secret"}`))
					}))
					t.Cleanup(srv.Close)
					level := "info"
					if mode == "debug-config" {
						level = "debug"
					}
					cfg := writeRawTestConfig(t, fmt.Sprintf("app:\n  camunda_version: \"8.10\"\nauth:\n  mode: none\napis:\n  camunda_api:\n    base_url: %s\nlog:\n  level: %s\n  format: %s\n", srv.URL, level, format))
					args := []string{"get", "pd", "--key", "8675309"}
					if mode == "verbose" || mode == "quiet" {
						args = append(args, "--"+mode)
					}
					if mode == "debug-flag" {
						args = append(args, "--debug")
					}
					capture := filepath.Join(t.TempDir(), "diagnostics")
					stdout, stderr, runErr := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, processDefinitionErrorEnvelopeHelper, "", map[string]string{
						"C8VOLT_TEST_CONFIG": cfg, "C8VOLT_TEST_OUTPUT_MODE": mode,
						"C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, args),
						"C8VOLT_TEST_PD_LOG_DEST":  capture, "C8VOLT_TEST_PD_LOG_ROUTE": route,
					}, "")
					var exitErr *exec.ExitError
					require.ErrorAs(t, runErr, &exitErr, stdout+stderr)
					require.Equal(t, exitcode.NotFound, exitErr.ExitCode(), stdout+stderr)
					diagnostics, err := os.ReadFile(capture)
					require.NoError(t, err)
					require.Empty(t, stderr, "all diagnostics must honor the configured destination")
					require.EqualValues(t, 1, requests.Load())
					if mode == "json" {
						decoder := json.NewDecoder(strings.NewReader(stdout))
						var envelope map[string]any
						require.NoError(t, decoder.Decode(&envelope))
						require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
						require.Equal(t, "failed", envelope["outcome"])
						require.Equal(t, "not_found", envelope["class"])
						require.Contains(t, requireJSONObject(t, envelope["detail"])["message"], "definition-body")
						require.Empty(t, diagnostics)
						return
					}
					require.Empty(t, stdout)
					require.Equal(t, 1, strings.Count(string(diagnostics), "get process definition: not found"))
					require.NotContains(t, string(diagnostics), "definition-token-secret")
					if strings.HasPrefix(mode, "debug-") {
						require.Contains(t, string(diagnostics), "api #")
						require.Contains(t, string(diagnostics), "/v2/process-definitions/8675309")
					} else {
						for _, marker := range []string{"GET", srv.URL, "definition-secret.invalid", "definition-body"} {
							require.NotContains(t, string(diagnostics), marker)
						}
						if mode != "verbose" {
							require.NotContains(t, string(diagnostics), "8675309")
						}
						if format == "plain" {
							lines := strings.Split(strings.TrimSuffix(string(diagnostics), "\n"), "\n")
							var messages []string
							for _, line := range lines {
								require.Regexp(t, `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}[+-]\d{2}:\d{2} `, line)
								messages = append(messages, strings.SplitN(line, " ", 2)[1])
							}
							want := []string{"ERROR get process definition: not found"}
							if mode == "verbose" {
								want = append([]string{"INFO retrieving process definition"}, want...)
							}
							require.Equal(t, want, messages)
						} else {
							decoder := json.NewDecoder(bytes.NewReader(diagnostics))
							if mode == "verbose" {
								var info map[string]any
								require.NoError(t, decoder.Decode(&info))
								require.Equal(t, "INFO", info["level"])
								require.Equal(t, "retrieving process definition", info["msg"])
								require.Equal(t, "8675309", info["key"])
							}
							var record map[string]any
							require.NoError(t, decoder.Decode(&record))
							require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
							require.Equal(t, "ERROR", record["level"])
							require.Equal(t, "get process definition: not found", record["msg"])
						}
					}
				})
			}
		}
	}
}
