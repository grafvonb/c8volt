// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	baseSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("base/profile server should not be used: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(baseSrv.Close)

	searchCalls := 0
	envSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		"Inspect ancestry, descendants",
		"./c8volt walk pi --key 2251799813711967 --family --tree",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"walk", "process-instance"}, []string{
		"Choose --parent for ancestry, --children for descendants, and --family",
		"returns the partial tree plus a warning",
		"./c8volt walk pi --key 2251799813711967 --family --tree",
	}, nil)
	require.Contains(t, output, "--tree")
}

func TestWalkProcessInstanceCommand_RejectsWithIncidentsWithoutKey(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--config", cfgPath, "walk", "process-instance", "--with-incidents"})
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	_, err := root.ExecuteC()
	require.Error(t, err)
	require.Contains(t, err.Error(), `required flag(s) "key" not set`)
	require.NotContains(t, buf.String(), "127.0.0.1:1")
}

func TestWalkProcessInstanceCommand_WithIncidentsChildrenHumanOutputShowsIncident(t *testing.T) {
	var incidentRequests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `"parentProcessInstanceKey":"123"`)
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			incidentRequests = append(incidentRequests, r.URL.Path)
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123", "Root job failed")))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"walk", "process-instance",
		"--key", "123",
		"--children",
		"--with-incidents",
	)

	require.Equal(t, []string{"/v2/process-instances/123/incidents/search"}, incidentRequests)
	require.Contains(t, output, "123")
	require.Contains(t, output, "inc!")
	require.Contains(t, output, "  incident: Root job failed")
}

func TestWalkProcessInstanceCommand_WithIncidentsFamilyHumanOutputShowsMultipleIncidents(t *testing.T) {
	var incidentRequests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", true))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			incidentRequests = append(incidentRequests, r.URL.Path)
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123", "Root failed")))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/124/incidents/search":
			incidentRequests = append(incidentRequests, r.URL.Path)
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "124", "Child failed", "Child timed out")))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"walk", "process-instance",
		"--key", "123",
		"--family",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"/v2/process-instances/123/incidents/search",
		"/v2/process-instances/124/incidents/search",
	}, incidentRequests)
	require.Contains(t, output, "123")
	require.Contains(t, output, "124")
	require.Contains(t, output, "  incident: Root failed")
	require.Contains(t, output, "  incident: Child failed")
	require.Contains(t, output, "  incident: Child timed out")
}

func TestWalkProcessInstanceCommand_WithIncidentsParentHumanOutputOmitsIncidentLinesWhenNoneReturned(t *testing.T) {
	var incidentRequests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/124":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("124", "123", false)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/124/incidents/search":
			incidentRequests = append(incidentRequests, r.URL.Path)
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "124")))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			incidentRequests = append(incidentRequests, r.URL.Path)
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123")))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"walk", "process-instance",
		"--key", "124",
		"--parent",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"/v2/process-instances/124/incidents/search",
		"/v2/process-instances/123/incidents/search",
	}, incidentRequests)
	require.Contains(t, output, "124")
	require.Contains(t, output, "123")
	require.NotContains(t, output, "  incident:")
}

func TestWalkProcessInstanceCommand_WithIncidentsJSONOutputShowsIncidentDetails(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `"parentProcessInstanceKey":"123"`)
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123", "Root job failed")))
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
		"--key", "123",
		"--children",
		"--with-incidents",
	)

	payload := requireWalkProcessInstanceJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	item := requireJSONObject(t, first["item"])
	require.Equal(t, "123", item["key"])

	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, "incident-1", incident["incidentKey"])
	require.Equal(t, "123", incident["processInstanceKey"])
	require.Equal(t, "Root job failed", incident["errorMessage"])
}

func TestWalkProcessInstanceCommand_WithIncidentsJSONOutputAssociatesMultipleKeys(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", true))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123", "Root failed")))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/124/incidents/search":
			_, _ = w.Write([]byte(`{"items":[
				{"incidentKey":"incident-child","processInstanceKey":"124","tenantId":"tenant","state":"ACTIVE","errorType":"JOB_NO_RETRIES","errorMessage":"Child failed"},
				{"incidentKey":"incident-wrong","processInstanceKey":"123","tenantId":"tenant","state":"ACTIVE","errorType":"JOB_NO_RETRIES","errorMessage":"Wrong key"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
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
		"--key", "123",
		"--family",
		"--with-incidents",
	)

	payload := requireWalkProcessInstanceJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 2)

	root := requireJSONObject(t, items[0])
	require.Equal(t, "123", requireJSONObject(t, root["item"])["key"])
	rootIncidents := requireJSONItems(t, root["incidents"], 1)
	require.Equal(t, "Root failed", requireJSONObject(t, rootIncidents[0])["errorMessage"])

	child := requireJSONObject(t, items[1])
	require.Equal(t, "124", requireJSONObject(t, child["item"])["key"])
	childIncidents := requireJSONItems(t, child["incidents"], 1)
	require.Equal(t, "Child failed", requireJSONObject(t, childIncidents[0])["errorMessage"])
}

func TestWalkProcessInstanceCommand_WithIncidentsJSONOutputShowsEmptyIncidentCollection(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123")))
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
		"--key", "123",
		"--children",
		"--with-incidents",
	)

	payload := requireWalkProcessInstanceJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	incidents := requireJSONItems(t, first["incidents"], 0)
	require.Empty(t, incidents)
}

func TestWalkProcessInstanceCommand_WithIncidentsJSONOutputPreservesTraversalMetadata(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "999", false)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/999":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", true))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123")))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/124/incidents/search":
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "124", "Child failed")))
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
		"--key", "123",
		"--family",
		"--with-incidents",
	)

	payload := requireWalkProcessInstanceJSONPayload(t, output)
	require.Equal(t, "family", payload["mode"])
	require.Equal(t, "partial", payload["outcome"])
	require.Equal(t, "123", payload["rootKey"])
	require.Equal(t, "one or more parent process instances were not found", payload["warning"])
	requireJSONItems(t, payload["keys"], 2)
	requireJSONItems(t, payload["items"], 2)
	missing := requireJSONItems(t, payload["missingAncestors"], 1)
	require.Equal(t, "999", requireJSONObject(t, missing[0])["Key"])
	edges := requireJSONObject(t, payload["edges"])
	requireJSONItems(t, edges["123"], 1)
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

func walkedProcessInstanceJSON(key, parentKey string, hasIncident bool) string {
	parent := ""
	if parentKey != "" {
		parent = `,"parentProcessInstanceKey":"` + parentKey + `"`
	}
	incident := "false"
	if hasIncident {
		incident = "true"
	}
	return `{"processInstanceKey":"` + key + `"` + parent + `,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","hasIncident":` + incident + `}`
}

func walkedProcessInstanceSearchJSON(t *testing.T, items ...string) string {
	t.Helper()

	rawItems := make([]json.RawMessage, len(items))
	for i, item := range items {
		rawItems[i] = json.RawMessage(item)
	}
	payload := map[string]any{
		"items": rawItems,
		"page": map[string]any{
			"totalItems":        len(rawItems),
			"hasMoreTotalItems": false,
		},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return string(raw)
}

func walkedIncidentDetailsJSON(t *testing.T, processInstanceKey string, messages ...string) string {
	t.Helper()

	items := make([]map[string]any, 0, len(messages))
	for i, message := range messages {
		items = append(items, map[string]any{
			"incidentKey":        fmt.Sprintf("incident-%d", i+1),
			"processInstanceKey": processInstanceKey,
			"tenantId":           "tenant",
			"state":              "ACTIVE",
			"errorType":          "JOB_NO_RETRIES",
			"errorMessage":       message,
		})
	}
	payload := map[string]any{
		"items": items,
		"page": map[string]any{
			"totalItems":        len(items),
			"hasMoreTotalItems": false,
		},
	}
	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	return string(raw)
}

func requireWalkProcessInstanceJSONPayload(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "walk process-instance", envelope["command"])
	return requireJSONObject(t, envelope["payload"])
}
