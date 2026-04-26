// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// Verifies walk commands consume env-overridden API base URLs during traversal requests.
func TestWalkProcessInstanceCommand_EnvBaseURLOverridesProfileAndBaseConfig(t *testing.T) {
	baseSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("base/profile server should not be used: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(baseSrv.Close)

	searchCalls := 0
	envSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813685255":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			w.Header().Set("Content-Type", "application/json")
			if searchCalls == 0 {
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813685256","parentProcessInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			} else {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			}
			searchCalls++
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813685256":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813685256","parentProcessInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(envSrv.Close)

	t.Setenv("C8VOLT_APIS_CAMUNDA_API_BASE_URL", envSrv.URL)

	cfgPath := writeRawTestConfig(t, `active_profile: dev
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+baseSrv.URL+`
profiles:
  dev:
    apis:
      camunda_api:
        base_url: `+baseSrv.URL+`
`)

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_EnvBaseURLOverridesProfileAndBaseConfigHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "2251799813685256")
}

func TestWalkHelp_DocumentsTraversalVerificationGuidance(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"walk"}, []string{
		"Inspect process-instance relationships",
		"parent/child structure",
		"./c8volt walk pi --key 2251799813711967 --family --tree",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"walk", "process-instance"}, []string{
		"understand ancestry or descendants",
		"Choose --parent for ancestry, --children for descendants, and --family",
		"returns the partial tree plus a warning",
		"./c8volt walk pi --key 2251799813711967 --family --tree",
	}, nil)
	require.Contains(t, output, "--tree")
}

func TestWalkProcessInstanceCommand_V89ChildrenTraversalUsesNativeSearchPath(t *testing.T) {
	var requests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813685255":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))
			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			switch {
			case parentKey == "2251799813685255":
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813685256","parentProcessInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case parentKey == "2251799813685256":
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"walk", "process-instance",
		"--key", "2251799813685255",
		"--children",
	)

	require.Len(t, requests, 2)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "walk process-instance", got["command"])
	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 2)
	require.Equal(t, "complete", payload["outcome"])
}

func TestWalkProcessInstanceCommand_PartialTraversalRendersWarningsAndJSONMetadata(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"123","parentProcessInstanceKey":"999","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/124":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"124","parentProcessInstanceKey":"123","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/999":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			w.Header().Set("Content-Type", "application/json")
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"124","parentProcessInstanceKey":"123","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	t.Run("parent list stays successful with warning", func(t *testing.T) {
		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"walk", "process-instance",
			"--key", "123",
			"--parent",
		)

		require.Contains(t, output, "123")
		require.Contains(t, output, "warning: one or more parent process instances were not found")
		require.Contains(t, output, "missing ancestor keys: 1 (use --verbose to list keys)")
		require.NotContains(t, output, "missing ancestor keys: 999")
	})

	t.Run("family tree renders resolved nodes with warning", func(t *testing.T) {
		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--verbose",
			"walk", "process-instance",
			"--key", "123",
			"--family",
			"--tree",
		)

		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "warning: one or more parent process instances were not found")
		require.Contains(t, output, "missing ancestor keys: 999")
	})

	t.Run("json output exposes partial traversal metadata", func(t *testing.T) {
		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"walk", "process-instance",
			"--key", "123",
			"--family",
		)

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.Equal(t, string(OutcomeSucceeded), got["outcome"])
		payload, ok := got["payload"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "partial", payload["outcome"])
		require.Equal(t, "one or more parent process instances were not found", payload["warning"])
		missing, ok := payload["missingAncestors"].([]any)
		require.True(t, ok)
		require.Len(t, missing, 1)
		items, ok := payload["items"].([]any)
		require.True(t, ok)
		require.Len(t, items, 2)
	})
}

func TestWalkProcessInstanceCommand_UsesEffectiveTenantForTraversalSearches(t *testing.T) {
	var requests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813685255":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))
			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			switch {
			case parentKey == "2251799813685255":
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813685256","parentProcessInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case parentKey == "2251799813685256":
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.9
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"--tenant", "tenant-a",
		"walk", "process-instance",
		"--key", "2251799813685255",
		"--children",
	)

	require.Len(t, requests, 2)
	for _, request := range requests {
		body := decodeCapturedPISearchRequest(t, request)
		filter, ok := body["filter"].(map[string]any)
		require.True(t, ok, "expected search request filter object")
		require.Equal(t, "tenant-a", filter["tenantId"])
	}
	require.Contains(t, output, `"tenantId": "tenant-a"`)
	require.NotContains(t, output, "base-tenant")
}

// Verifies walk process-instance rejects unsupported --mode values.
func TestWalkProcessInstanceCommand_RejectsInvalidMode(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_RejectsInvalidModeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "invalid --mode")
}

func TestWalkProcessInstanceCommand_FailureKeepsSingleRootDetail(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/2251799813685255", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_FailureKeepsSingleRootDetailHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.NotFound, exitErr.ExitCode())
	require.Contains(t, string(output), "resource not found")
	require.Contains(t, string(output), "ancestry")
	require.NotContains(t, string(output), "ancestry get")
	require.Contains(t, string(output), "get process instance")
	require.Less(t, strings.Index(string(output), "ancestry"), strings.Index(string(output), "get process instance"))
	require.NotContains(t, string(output), "fetching process instance with key")
	require.NotContains(t, string(output), "get 2251799813685255")
	require.NotContains(t, string(output), "missing ancestor keys")
	require.NotContains(t, string(output), "parent process instances were not found")
}

func TestWalkProcessInstanceCommand_DefaultOutputRemainsHumanReadable(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813685255":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813685255","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"walk", "process-instance",
		"--key", "2251799813685255",
		"--children",
	)

	require.Contains(t, output, "2251799813685255")
	require.NotContains(t, output, `"outcome"`)
	require.NotContains(t, output, `"command"`)
}

func TestWalkProcessInstanceCommand_RejectsAutomationMode(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_RejectsAutomationModeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "walk process-instance does not support --automation")
}

// Helper-process entrypoint for invalid walk-mode validation.
func TestWalkProcessInstanceCommand_RejectsInvalidModeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "walk", "process-instance", "--key", "2251799813685255", "--mode", "broken"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestWalkProcessInstanceCommand_EnvBaseURLOverridesProfileAndBaseConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "walk", "process-instance", "--key", "2251799813685255", "--children"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestWalkProcessInstanceCommand_FailureKeepsSingleRootDetailHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "walk", "process-instance", "--key", "2251799813685255", "--parent"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestWalkProcessInstanceCommand_RejectsAutomationModeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--automation", "walk", "process-instance", "--key", "2251799813685255", "--children"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
