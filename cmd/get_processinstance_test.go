// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

type fakeCommandActivitySink struct {
	mu      sync.Mutex
	started int
	stopped int
	msgs    []string
}

func (s *fakeCommandActivitySink) StartActivity(msg string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.started++
	s.msgs = append(s.msgs, msg)
}

func (s *fakeCommandActivitySink) StopActivity() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopped++
}

func TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "get", "process-instance", "--help")

	require.Contains(t, output, "Use this command to inspect workflow instances")
	require.Contains(t, output, "Use --total when you only need the numeric count")
	require.Contains(t, output, "Direct --key lookups stay strict")
	require.Contains(t, output, "JSON mode consumes remaining pages")
	require.Contains(t, output, "./c8volt get pi --state active --total")
	require.Contains(t, output, "./c8volt get pi --key 2251799813711967 --json")
	require.Contains(t, output, "capped backend totals stay lower bounds")
	require.Contains(t, output, "--auto-confirm")
	require.Contains(t, output, "--batch-size int32")
	require.Contains(t, output, "number of process instances to fetch per page")
	require.Contains(t, output, "--limit int32")
	require.Contains(t, output, "maximum number of matching process instances to return or process across all pages")
	require.NotContains(t, output, "--count")
}

// Verifies search-mode get process-instance sends the expected filter and pagination request shape.
func TestGetProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	page := decodeCapturedPISearchPage(t, requests)

	require.Equal(t, "ACTIVE", filter["state"])
	require.EqualValues(t, 5, page["limit"])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get process-instance", got["command"])
}

func TestGetProcessInstanceJSONWithAge_AddsMetaField(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--with-age",
	)

	require.NotEmpty(t, requests)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	meta, ok := payload["meta"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, meta["withAge"])
}

func TestGetProcessInstanceTotalOutput(t *testing.T) {
	t.Run("reported total prints only the numeric count without fetching later pages", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Zero(t, promptCalls)
		require.Equal(t, "3\n", stdout)
		require.Empty(t, stderr)
	})

	t.Run("zero matches still print zero only", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Equal(t, "0\n", stdout)
		require.Empty(t, stderr)
	})
}

func TestGetProcessInstanceTotalValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "key lookup stays on the strict single-resource path",
			helper: "TestGetProcessInstanceTotalWithKeyHelper",
			want:   "--total cannot be combined with --key",
		},
		{
			name:   "json output is rejected",
			helper: "TestGetProcessInstanceTotalWithJSONHelper",
			want:   "--total cannot be combined with --json",
		},
		{
			name:   "keys-only output is rejected",
			helper: "TestGetProcessInstanceTotalWithKeysOnlyHelper",
			want:   "--total cannot be combined with --keys-only",
		},
		{
			name:   "with-age output is rejected",
			helper: "TestGetProcessInstanceTotalWithAgeHelper",
			want:   "--total cannot be combined with --with-age",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

func TestGetProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "removed count flag is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper",
			want:   "unknown flag: --count",
		},
		{
			name:   "non-positive limit is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsInvalidLimitHelper",
			want:   "--limit must be positive integer",
		},
		{
			name:   "limit cannot be combined with key",
			helper: "TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper",
			want:   "--limit cannot be combined with --key",
		},
		{
			name:   "limit cannot be combined with total",
			helper: "TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper",
			want:   "--total cannot be combined with --limit",
		},
		{
			name:   "invalid batch size is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper",
			want:   "invalid value for --batch-size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, tt.want)
		})
	}
}

func TestApplyPISearchResultFilters_OrphanChildrenUseCommandActivity(t *testing.T) {
	prevOrphanOnly := flagGetPIOrphanChildrenOnly
	t.Cleanup(func() {
		flagGetPIOrphanChildrenOnly = prevOrphanOnly
	})
	flagGetPIOrphanChildrenOnly = true

	sink := &fakeCommandActivitySink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	cliFilterCalls := 0
	cli := stubProcessAPI{filterOrphanParent: func(ctx context.Context, items []process.ProcessInstance, opts ...options.FacadeOption) ([]process.ProcessInstance, error) {
		cliFilterCalls++
		return items[:1], nil
	}}

	pis := process.ProcessInstances{
		Total: 2,
		Items: []process.ProcessInstance{
			{Key: "123", ParentKey: "456"},
			{Key: "124", ParentKey: "457"},
		},
	}

	got, err := applyPISearchResultFilters(cmd, cli, pis)
	require.NoError(t, err)
	require.Equal(t, 1, cliFilterCalls)
	require.Len(t, got.Items, 1)
	require.EqualValues(t, 1, got.Total)

	sink.mu.Lock()
	defer sink.mu.Unlock()
	require.Equal(t, 1, sink.started)
	require.Equal(t, 1, sink.stopped)
	require.Equal(t, []string{"checking orphan parents for 2 process instance(s)"}, sink.msgs)
}

func TestGetProcessInstanceKeyLookup_UsesGeneratedLookupEndpoint(t *testing.T) {
	response := `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`

	t.Run("explicit flag tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"--tenant", "tenant-a",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("environment tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTestWithEnv(t,
			[]string{"C8VOLT_APP_TENANT=tenant-a"},
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("profile tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
profiles:
  dev:
    app:
      tenant: tenant-a
    apis:
      camunda_api:
        base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"--profile", "dev",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("base config tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  tenant: tenant-a
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})
}

func TestGetProcessInstanceSearch_V87StillSupportsTenantScopedSearch(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"<default>"}]}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	require.Equal(t, "<default>", filter["tenantId"])
	require.Equal(t, "ACTIVE", filter["state"])
	require.Contains(t, output, `"tenantId": "<default>"`)
}

func TestGetProcessInstanceCommand_V89KeyLookupUsesNativeSearchPath(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/2251799813711967", r.URL.Path)
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
	)

	require.Equal(t, []string{"/v2/process-instances/2251799813711967"}, requests)
	require.Contains(t, output, `"key": "2251799813711967"`)
}

// Verifies get process-instance date filters map to expected API query fields and invalid combinations are rejected.
func TestGetProcessInstanceDateFilterScaffold(t *testing.T) {
	t.Run("start date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			require.NotContains(t, filter["startDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
				"--start-date-before", "2026-01-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-01-31T23:59:59.999999999Z")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("end date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--end-date-after", "2026-02-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")
			require.NotContains(t, filter["endDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range composed with state filter", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--state", "completed",
				"--end-date-after", "2026-02-01",
				"--end-date-before", "2026-03-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			require.Equal(t, "COMPLETED", filter["state"])
			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-03-31T23:59:59.999999999Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("invalid date command coverage", func(t *testing.T) {
		t.Run("invalid start-date format exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidDateFormatHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
		})

		t.Run("invalid start-date range exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidStartDateRangeHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid range for --start-date-after and --start-date-before: "2026-02-01" is later than "2026-01-31"`)
		})

		t.Run("date filters are rejected for direct key lookup", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceDateFiltersWithKeyHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
		})
	})
}

func TestGetProcessInstanceRelativeDayFilterScaffold(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	t.Run("start-day range search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "7",
			"--start-date-newer-days", "30",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-03-11T00:00:00Z")
		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-03T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})

	t.Run("end-day upper bound search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--state", "completed",
			"--end-date-newer-days", "14",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		require.Equal(t, "COMPLETED", filter["state"])
		requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-03-27T00:00:00Z")
		requireCapturedPISearchDateExists(t, filter, "endDate")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

func TestGetProcessInstanceRelativeDayValidation(t *testing.T) {
	t.Run("negative relative-day values exit through shared invalid-input path", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceNegativeRelativeDayHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "invalid value for --start-date-older-days: -2, expected non-negative integer")
	})

	t.Run("mixed absolute and relative start-date filters are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "start-date absolute and relative day filters cannot be combined")
	})

	t.Run("invalid derived relative-day ranges are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidRelativeDayRangeHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, `invalid range for --start-date-newer-days and --start-date-older-days: "2026-04-03" is later than "2026-03-11"`)
	})

	t.Run("local-day derivation honors the configured day boundary override", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTestWithEnv(t,
			[]string{testRelativeDayNowEnv + "=2026-04-10T00:30:00+02:00"},
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "0",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-10T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

func TestPopulatePISearchFilterOpts_DerivesRelativeDayBounds(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	flagGetPIStartAfterDays = 7
	flagGetPIStartBeforeDays = 30
	flagGetPIEndAfterDays = 14
	flagGetPIEndBeforeDays = 1

	filter := populatePISearchFilterOpts()

	require.Equal(t, "2026-03-11", filter.StartDateAfter)
	require.Equal(t, "2026-04-03", filter.StartDateBefore)
	require.Equal(t, "2026-04-09", filter.EndDateAfter)
	require.Equal(t, "2026-03-27", filter.EndDateBefore)
}

func TestPopulatePISearchFilterOpts_TranslatesSupportedPresenceFlags(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIChildrenOnly = true
	flagGetPIIncidentsOnly = true

	filter := populatePISearchFilterOpts()

	require.Equal(t, new(true), filter.HasParent)
	require.Equal(t, new(true), filter.HasIncident)
}

func TestValidatePISearchFlags_RejectsMixedAbsoluteAndRelativeInputs(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartDateAfter = "2026-04-03"
	flagGetPIStartBeforeDays = 7

	err := validatePISearchFlags()

	require.Error(t, err)
	require.Contains(t, err.Error(), "start-date absolute and relative day filters cannot be combined")
}

func TestHasPISearchFilterFlags_WithRelativeDaysOnly(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartAfterDays = 72

	require.True(t, hasPISearchFilterFlags())
}

func TestResolvePISearchSize(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := getProcessInstanceCmd
	resetPISearchBatchSizeFlag(t, cmd)

	t.Run("uses shared config default when batch-size flag is unchanged", func(t *testing.T) {
		resetPISearchBatchSizeFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(250), resolvePISearchSize(cmd, cfg))
	})

	t.Run("uses batch-size override when the flag is changed", func(t *testing.T) {
		resetPISearchBatchSizeFlag(t, cmd)
		require.NoError(t, cmd.Flags().Set("batch-size", "125"))
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(125), resolvePISearchSize(cmd, cfg))
	})

	t.Run("falls back to repository default for invalid config values", func(t *testing.T) {
		resetProcessInstanceCommandGlobals()
		resetPISearchBatchSizeFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 0

		require.Equal(t, int32(consts.MaxPISearchSize), resolvePISearchSize(cmd, cfg))
	})
}

func TestGetProcessInstancePagingFlow(t *testing.T) {
	t.Run("limit truncates results across pages and stops without continuation prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"126","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--state", "active",
			"--batch-size", "2",
			"--limit", "3",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: yes, next step: limit-reached")
		require.Contains(t, output, "detail: stopped after reaching limit of 3 process instance(s)")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
		require.NotContains(t, output, "126")
	})

	t.Run("uses shared config default and prompts before the next page", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prompts := []string{}
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			prompts = append(prompts, prompt)
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 1000, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Len(t, prompts, 1)
		require.Contains(t, prompts[0], "More matching process instances remain")
		require.Contains(t, output, "page size: 1000, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 1000, current page: 1, total so far: 3, more matches: no, next step: complete")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("batch-size override and auto-confirm fetch every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	})

	t.Run("short n controls per-page batch size", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--state", "active",
			"-n", "4",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.EqualValues(t, 4, pages[0]["limit"])
		require.Contains(t, output, "page size: 4, current page: 1, total so far: 1, more matches: no, next step: complete")
		require.Contains(t, output, "123")
	})

	t.Run("batch-size and limit remain independent when limit is smaller", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"126","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":6,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--state", "active",
			"--batch-size", "4",
			"--limit", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.EqualValues(t, 4, pages[0]["limit"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 4, current page: 2, total so far: 2, more matches: yes, next step: limit-reached")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.NotContains(t, output, "125")
		require.NotContains(t, output, "126")
	})

	t.Run("json mode fetches every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, `"outcome": "succeeded"`)
		require.Contains(t, output, `"total": 3`)
		require.Contains(t, output, `"key": "123"`)
		require.Contains(t, output, `"key": "124"`)
		require.Contains(t, output, `"key": "125"`)
	})

	t.Run("automation mode fetches every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--automation",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("automation json mode keeps stdout machine-readable", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--automation",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.Zero(t, promptCalls)
		require.Contains(t, stdout, `"outcome": "succeeded"`)
		require.Contains(t, stdout, `"total": 3`)
		require.NotContains(t, stdout, "page size:")
		require.Empty(t, stderr)
	})

	t.Run("automation json mode keeps stdout machine-readable even with debug logs", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--debug",
			"--automation",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.Zero(t, promptCalls)
		require.Contains(t, stdout, `"outcome": "succeeded"`)
		require.Contains(t, stdout, `"total": 3`)
		require.NotContains(t, stdout, "DEBUG")
		require.NotContains(t, stdout, "config loaded")
		require.NotEmpty(t, stderr)
		require.Contains(t, stderr, "DEBUG")
	})

	t.Run("declined continuation reports partial completion summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			return localPreconditionError(ErrCmdAborted)
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
		require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
	})

	t.Run("indeterminate overflow stops with warning summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
		require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
	})

	t.Run("v87 fallback keeps final filtered results even when the request stays broad", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/v1/process-instances/search", r.URL.Path)

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant","parentKey":456,"incident":true},{"key":124,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant","incident":false}]}`))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--json",
			"get", "process-instance",
			"--children-only",
			"--incidents-only",
		)

		filter := decodeCapturedPISearchFilter(t, requests)
		require.NotContains(t, filter, "parentKey")
		require.NotContains(t, filter, "hasIncident")
		require.Contains(t, output, `"total": 1`)
		require.Contains(t, output, `"key": "123"`)
		require.NotContains(t, output, `"key": "124"`)
	})

	t.Run("orphan-child filtering stays on follow-up lookups for supported versions", func(t *testing.T) {
		for _, version := range []string{"8.8", "8.9"} {
			t.Run(version, func(t *testing.T) {
				var searchRequests []string
				var getPaths []string
				call := 0
				srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					call++
					w.Header().Set("Content-Type", "application/json")
					if call == 1 {
						require.Equal(t, http.MethodPost, r.Method)
						require.Equal(t, "/v2/process-instances/search", r.URL.Path)
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						searchRequests = append(searchRequests, string(body))
						_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","parentProcessInstanceKey":"456","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
						return
					}
					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, "/v2/process-instances/456", r.URL.Path)
					getPaths = append(getPaths, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"message":"not found"}`))
				}))
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, version)

				output := executeRootForProcessInstanceTest(t,
					"--config", cfgPath,
					"--tenant", "tenant",
					"--json",
					"get", "process-instance",
					"--orphan-children-only",
				)

				filters := decodeCapturedPISearchRequests(t, searchRequests)
				require.Len(t, filters, 1)

				topLevelFilter, ok := filters[0]["filter"].(map[string]any)
				require.True(t, ok)
				require.NotContains(t, topLevelFilter, "parentProcessInstanceKey")
				require.NotContains(t, topLevelFilter, "processInstanceKey")
				require.Equal(t, []string{"/v2/process-instances/456"}, getPaths)

				require.Contains(t, output, `"total": 1`)
				require.Contains(t, output, `"key": "123"`)
			})
		}
	})

	t.Run("supported filters keep paging summaries aligned with server-filtered pages", func(t *testing.T) {
		for _, version := range []string{"8.8", "8.9"} {
			t.Run(version, func(t *testing.T) {
				var requests []string
				srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
					`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"456"},{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"457"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
					`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"458"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
				)
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, version)
				prompts := []string{}
				prevConfirm := confirmCmdOrAbortFn
				confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
					prompts = append(prompts, prompt)
					return nil
				}
				t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

				output := executeRootForProcessInstanceTest(t,
					"--config", cfgPath,
					"--tenant", "tenant",
					"--verbose",
					"get", "process-instance",
					"--children-only",
					"--incidents-only",
					"--batch-size", "2",
				)

				pages := decodeCapturedPISearchPages(t, requests)
				decoded := decodeCapturedPISearchRequests(t, requests)
				require.Len(t, pages, 2)
				require.Len(t, decoded, 2)
				require.EqualValues(t, 2, pages[0]["limit"])
				require.EqualValues(t, 0, pages[0]["from"])
				require.EqualValues(t, 2, pages[1]["from"])
				filter, ok := decoded[0]["filter"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, filter["hasIncident"])

				parentFilter, ok := filter["parentProcessInstanceKey"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, parentFilter["$exists"])

				require.Len(t, prompts, 1)
				require.Contains(t, prompts[0], "Fetched 2 process instance(s) on this page (2 total so far)")
				require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
				require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
			})
		}
	})
}

func TestPIContinuationHelpers(t *testing.T) {
	t.Run("auto-confirm chooses auto-continue for overflow", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 50},
			OverflowState: process.ProcessInstanceOverflowStateHasMore,
			Items:         []process.ProcessInstance{{Key: "1"}, {Key: "2"}},
		}

		summary := newPIProgressSummary(page, 2, true)

		require.Equal(t, processInstanceContinuationAutoContinue, summary.ContinuationState)
		require.Equal(t, 50, int(summary.PageSize))
		require.Equal(t, 2, summary.CurrentPageCount)
		require.Equal(t, 2, summary.CumulativeCount)
	})

	t.Run("indeterminate overflow stops with warning", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 25},
			OverflowState: process.ProcessInstanceOverflowStateIndeterminate,
		}

		summary := newPIProgressSummary(page, 0, false)

		require.Equal(t, processInstanceContinuationWarningStop, summary.ContinuationState)
	})
}

func decodeSingleRequestJSON(t *testing.T, requests []string) map[string]any {
	t.Helper()

	require.Len(t, requests, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(requests[0]), &got))
	return got
}

func newIPv4Server(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, handler)
}

func executeRootForProcessInstanceTest(t *testing.T, args ...string) string {
	t.Helper()

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

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}

func executeRootForProcessInstanceWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return stdout.String(), stderr.String()
}

func executeRootForProcessInstanceTestWithEnv(t *testing.T, env []string, args ...string) string {
	t.Helper()

	prevNow := relativeDayNow
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	for _, kv := range env {
		key, value, ok := strings.Cut(kv, "=")
		require.True(t, ok)
		prevValue, hadValue := os.LookupEnv(key)
		require.NoError(t, os.Setenv(key, value))
		t.Cleanup(func() {
			if hadValue {
				require.NoError(t, os.Setenv(key, prevValue))
				return
			}
			require.NoError(t, os.Unsetenv(key))
		})
	}
	applyRelativeDayNowOverrideFromEnv(t)

	return executeRootForProcessInstanceTest(t, args...)
}

func resetProcessInstanceCommandGlobals() {
	flagCancelPIKeys = nil
	flagDeletePIKeys = nil
	flagGetPIKeys = nil
	flagRunPIProcessDefinitionBpmnProcessIds = nil
	flagRunPIProcessDefinitionKey = nil
	flagRunPIProcessDefinitionVersion = 0
	flagRunPICount = 1
	flagRunPIVars = ""
	flagGetPIBpmnProcessID = ""
	flagGetPIProcessVersion = 0
	flagGetPIProcessVersionTag = ""
	flagGetPIProcessDefinitionKey = ""
	flagGetPIStartDateAfter = ""
	flagGetPIStartDateBefore = ""
	flagGetPIEndDateAfter = ""
	flagGetPIEndDateBefore = ""
	flagGetPIStartAfterDays = -1
	flagGetPIStartBeforeDays = -1
	flagGetPIEndAfterDays = -1
	flagGetPIEndBeforeDays = -1
	flagGetPIWithAge = false
	flagGetPITotal = false
	flagGetPIState = "all"
	flagGetPIParentKey = ""
	flagGetPISize = consts.MaxPISearchSize
	flagGetPILimit = 0
	flagGetPIRootsOnly = false
	flagGetPIChildrenOnly = false
	flagGetPIOrphanChildrenOnly = false
	flagGetPIIncidentsOnly = false
	flagGetPINoIncidentsOnly = false
	flagCmdAutoConfirm = false
	flagVerbose = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagNoWait = false
	flagForce = false
	flagNoStateCheck = false
	flagDryRun = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	confirmCmdOrAbortFn = confirmCmdOrAbort
}

func resetPISearchBatchSizeFlag(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	flag := cmd.Flags().Lookup("batch-size")
	require.NotNil(t, flag)
	require.NoError(t, flag.Value.Set("1000"))
	flag.Changed = false
}

func resetRootPersistentFlags(t *testing.T, root *cobra.Command) {
	t.Helper()

	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		require.NoError(t, flag.Value.Set(flag.DefValue))
		flag.Changed = false
	})
}

func executeProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
	t.Helper()

	output, err := testx.RunCmdSubprocess(t, helperName, map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

func TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--count", "2"}

	Execute()
}

func TestGetProcessInstanceCommand_RejectsInvalidLimitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--limit", "0"}

	Execute()
}

func TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--limit", "1"}

	Execute()
}

func TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--total", "--limit", "10"}

	Execute()
}

func TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--batch-size", "0"}

	Execute()
}

// Helper-process entrypoint for negative relative-day validation.
func TestGetProcessInstanceNegativeRelativeDayHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "-2"}

	Execute()
}

// Helper-process entrypoint for mixed absolute-plus-relative start-date validation.
func TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-04-03", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for invalid relative-day range validation.
func TestGetProcessInstanceInvalidRelativeDayRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "30", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for --total with --key validation.
func TestGetProcessInstanceTotalWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--total"}

	Execute()
}

// Helper-process entrypoint for --total with --json validation.
func TestGetProcessInstanceTotalWithJSONHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--json", "--total"}

	Execute()
}

// Helper-process entrypoint for --total with --keys-only validation.
func TestGetProcessInstanceTotalWithKeysOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--keys-only", "--total"}

	Execute()
}

// Helper-process entrypoint for --total with --with-age validation.
func TestGetProcessInstanceTotalWithAgeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--with-age", "--total"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestGetProcessInstanceInvalidDateFormatHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for invalid start-date range validation.
func TestGetProcessInstanceInvalidStartDateRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-01", "--start-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestGetProcessInstanceDateFiltersWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}
