// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
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
		"./c8volt walk pi --key <process-instance-key>",
	}, nil)
	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"walk", "process-instance"}, []string{
		"By default, walk shows the full process-instance family as an ASCII tree",
		"returns the partial tree plus a warning",
		"./c8volt walk pi --key <process-instance-key> --with-incidents",
		"./c8volt walk pi --key <process-instance-key> --with-vars",
		"./c8volt walk pi --key <process-instance-key> --with-elements",
		"./c8volt walk pi --key <process-instance-key> --with-elements --with-listeners",
		"./c8volt walk pi --key <process-instance-key> --flat",
	}, nil)
	require.Contains(t, output, "--flat")
	require.Contains(t, output, "--with-elements")
	require.Contains(t, output, "--with-listeners")
	require.Contains(t, output, "--incident-message-limit int")
	require.Contains(t, output, "--incident-state string")
	require.Contains(t, output, "incident state scope for --with-incidents: active, pending, resolved, migrated, unknown, all")
	require.NotContains(t, output, "--tree")
	require.NotContains(t, output, "--family")
}

// TestWalkProcessInstanceCommand_RegressionPreservesReadOnlyTraversalContract keeps walk metadata and flag-state reset behavior stable.
func TestWalkProcessInstanceCommand_RegressionPreservesReadOnlyTraversalContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	flagWalkPIWithElements = true
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	capability := commandCapabilityForCommand(walkProcessInstanceCmd)

	require.Equal(t, "walk process-instance", capability.Path)
	require.Equal(t, CommandMutationReadOnly, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportUnsupported, capability.AutomationSupport)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "string",
		Required:    true,
		Repeated:    false,
		Description: "start walking from this process instance key",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-incidents",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "show incident keys, states, and messages for keyed process-instance walks",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "with-vars",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "show process-instance-scope variables for keyed process-instance walks",
	})
	require.False(t, flagWalkPIWithElements)
}

// TestActivityItemsFromTraversal_AttachesElementsPreservingTraversalOrder verifies shared walk activity items can carry element enrichment without changing selection order.
func TestActivityItemsFromTraversal_AttachesElementsPreservingTraversalOrder(t *testing.T) {
	result := process.TraversalResult{
		Keys: []string{"root", "child", "missing"},
		Chain: map[string]process.ProcessInstance{
			"root":  {Key: "root"},
			"child": {Key: "child"},
		},
	}
	elements := process.ElementEnrichedProcessInstances{
		Items: []process.ElementEnrichedProcessInstance{
			{
				Item: process.ProcessInstance{Key: "child"},
				Elements: []process.ProcessInstanceElement{{
					ElementInstanceKey: "element-child",
					ProcessInstanceKey: "child",
				}},
			},
			{
				Item:     process.ProcessInstance{Key: "root"},
				Elements: []process.ProcessInstanceElement{},
			},
		},
	}

	items := activityItemsFromTraversal(result, process.IncidentEnrichedTraversalResult{}, process.VariableEnrichedProcessInstances{}, elements, false)

	require.Len(t, items, 2)
	require.Equal(t, "root", items[0].Item.Key)
	require.Empty(t, items[0].Elements)
	require.NotNil(t, items[0].Elements)
	require.Equal(t, "child", items[1].Item.Key)
	require.Equal(t, "element-child", items[1].Elements[0].ElementInstanceKey)
}

// TestWalkActivityView_RendersElementsAsProcessInstanceDetails verifies element detail rows stay separate from child process-instance tree rows.
func TestWalkActivityView_RendersElementsAsProcessInstanceDetails(t *testing.T) {
	cmd := &cobra.Command{Use: "process-instance"}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	result := process.TraversalResult{
		RootKey: "root",
		Keys:    []string{"root", "child"},
		Edges:   map[string][]string{"root": {"child"}},
	}
	items := []processInstanceActivityItem{
		{
			Item: process.ProcessInstance{Key: "root", TenantId: "tenant", BpmnProcessId: "demo", ProcessVersion: 1, State: process.StateActive},
			Elements: []process.ProcessInstanceElement{{
				ElementInstanceKey: "element-root",
				Type:               "SERVICE_TASK",
				ElementId:          "task-a",
				State:              "ACTIVE",
				StartDate:          "2026-07-15T10:12:01Z",
				ProcessInstanceKey: "root",
			}},
		},
		{
			Item: process.ProcessInstance{Key: "child", TenantId: "tenant", BpmnProcessId: "demo", ProcessVersion: 1, State: process.StateActive},
		},
	}

	require.NoError(t, walkActivityView(cmd, walkPIModeFamily, result, items))

	output := buf.String()
	require.Contains(t, output, "root tenant demo v1 ACTIVE")
	require.Contains(t, output, "├─ elements:\n│  └─ element-root SERVICE_TASK task-a ACTIVE")
	require.Contains(t, output, "└─ child tenant demo v1 ACTIVE")
	require.Less(t, strings.Index(output, "elements:"), strings.Index(output, "child tenant"))
}

// TestWalkProcessInstanceCommand_WithElementsFamilyHumanOutputShowsRuntimeElements verifies the default family walk enriches each walked owner without changing tree order.
func TestWalkProcessInstanceCommand_WithElementsFamilyHumanOutputShowsRuntimeElements(t *testing.T) {
	var requests []string
	var elementFilters []map[string]any

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			elementFilters = append(elementFilters, filter)
			switch filter["processInstanceKey"] {
			case "123":
				_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
					walkedElementInstanceFixture("element-root", "123", "root-task", false, ""),
				)))
			case "124":
				_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
					walkedElementInstanceFixture("element-child", "124", "child-task", false, ""),
				)))
			default:
				t.Fatalf("unexpected element filter: %v", filter)
			}
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
		"--with-elements",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"GET /v2/process-instances/123",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/element-instances/search",
		"POST /v2/element-instances/search",
	}, requests)
	require.Len(t, elementFilters, 2)
	require.Equal(t, "123", elementFilters[0]["processInstanceKey"])
	require.Equal(t, "124", elementFilters[1]["processInstanceKey"])
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ elements:")
	require.Contains(t, output, "element-root SERVICE_TASK root-task")
	require.Contains(t, output, "└─ 124 tenant demo v3 ACTIVE")
	require.Contains(t, output, "   └─ elements:")
	require.Contains(t, output, "element-child SERVICE_TASK child-task")
	require.Less(t, strings.Index(output, "123 tenant demo"), strings.Index(output, "element-root"))
	require.Less(t, strings.Index(output, "element-root"), strings.Index(output, "124 tenant demo"))
	require.Less(t, strings.Index(output, "124 tenant demo"), strings.Index(output, "element-child"))
}

// TestWalkProcessInstanceCommand_WithListenersFamilyHumanOutputNestsListenerRows verifies default family traversal keeps listener details inside element blocks.
func TestWalkProcessInstanceCommand_WithListenersFamilyHumanOutputNestsListenerRows(t *testing.T) {
	var requests []string
	srv := newWalkProcessInstanceWithListenersServer(t, &requests, map[string]string{
		"123": walkedElementInstancesSearchJSON(t,
			walkedElementInstanceFixture("element-root", "123", "root-task", false, ""),
		),
		"124": walkedElementInstancesSearchJSON(t,
			walkedElementInstanceFixture("element-child", "124", "child-task", false, ""),
		),
	}, map[string][]string{
		"123": {
			walkedJobSearchJSON(t, map[string]any{"jobKey": "job-exec-root", "kind": "EXECUTION_LISTENER", "listenerEventType": "START", "type": "audit-start", "state": "CREATED", "retries": 3, "worker": "worker-a", "processInstanceKey": "123", "elementInstanceKey": "element-root", "elementId": "root-task", "tenantId": "tenant"}),
			walkedJobSearchJSON(t),
		},
		"124": {
			walkedJobSearchJSON(t),
			walkedJobSearchJSON(t, map[string]any{"jobKey": "job-task-child", "kind": "TASK_LISTENER", "listenerEventType": "COMPLETING", "type": "audit-task", "state": "FAILED", "retries": 0, "processInstanceKey": "124", "elementInstanceKey": "element-child", "elementId": "child-task", "tenantId": "tenant", "errorCode": "LISTENER_FAILED", "errorMessage": "worker failed"}),
		},
	})
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"walk", "process-instance",
		"--key", "123",
		"--with-elements",
		"--with-listeners",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"GET /v2/process-instances/123",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/search",
		"POST /v2/element-instances/search",
		"POST /v2/jobs/search",
		"POST /v2/jobs/search",
		"POST /v2/element-instances/search",
		"POST /v2/jobs/search",
		"POST /v2/jobs/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ elements:\n│  └─ element-root SERVICE_TASK root-task ACTIVE")
	require.Contains(t, output, "│     └─ listeners:\n│        └─ job-exec-root EXECUTION_LISTENER lsnr:START CREATED tp:audit-start r:3 worker:worker-a")
	require.Contains(t, output, "└─ 124 tenant demo v3 ACTIVE")
	require.Contains(t, output, "   └─ elements:\n      └─ element-child SERVICE_TASK child-task ACTIVE")
	require.Contains(t, output, "         └─ listeners:\n            └─ job-task-child TASK_LISTENER lsnr:COMPLETING FAILED tp:audit-task r:0")
	require.Contains(t, output, "ec:LISTENER_FAILED")
	require.Less(t, strings.Index(output, "element-root"), strings.Index(output, "job-exec-root"))
	require.Less(t, strings.Index(output, "job-exec-root"), strings.Index(output, "124 tenant demo"))
	require.Less(t, strings.Index(output, "element-child"), strings.Index(output, "job-task-child"))
}

// TestWalkProcessInstanceCommand_WithElementsKeepsEmptyOwnersVisible verifies a walked row with no elements does not gain placeholder detail rows.
func TestWalkProcessInstanceCommand_WithElementsKeepsEmptyOwnersVisible(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t)))
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
		"--with-elements",
	)

	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.NotContains(t, output, "elements:")
	require.NotContains(t, output, "element-")
}

// TestWalkProcessInstanceCommand_WithElementsRendersIncidentMarkers keeps element incident markers aligned with get pi output.
func TestWalkProcessInstanceCommand_WithElementsRendersIncidentMarkers(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
				walkedElementInstanceFixture("element-with-key", "123", "service-a", true, "incident-777"),
				walkedElementInstanceFixture("element-with-marker", "123", "service-b", true, ""),
			)))
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
		"--with-elements",
	)

	require.Contains(t, output, "└─ elements:")
	require.Contains(t, output, "element-with-key")
	require.Contains(t, output, "inc!:incident-777")
	require.Contains(t, output, "element-with-marker")
	require.Contains(t, output, "inc!")
}

// TestWalkProcessInstanceCommand_ChildrenWithElementsPreservesDescendantSelection verifies child traversal order is reused for owner-specific element lookup.
func TestWalkProcessInstanceCommand_ChildrenWithElementsPreservesDescendantSelection(t *testing.T) {
	var elementFilters []map[string]any

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("125", "124", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"125"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			elementFilters = append(elementFilters, filter)
			key, _ := filter["processInstanceKey"].(string)
			_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
				walkedElementInstanceFixture("element-"+key, key, "task-"+key, false, ""),
			)))
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
		"--with-elements",
	)

	require.Len(t, elementFilters, 3)
	require.Equal(t, "123", elementFilters[0]["processInstanceKey"])
	require.Equal(t, "124", elementFilters[1]["processInstanceKey"])
	require.Equal(t, "125", elementFilters[2]["processInstanceKey"])
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-123 SERVICE_TASK task-123 ACTIVE")
	require.Contains(t, output, "124 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-124 SERVICE_TASK task-124 ACTIVE")
	require.Contains(t, output, "125 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-125 SERVICE_TASK task-125 ACTIVE")
	require.Less(t, strings.Index(output, "123 tenant demo"), strings.Index(output, "element-123"))
	require.Less(t, strings.Index(output, "element-123"), strings.Index(output, "124 tenant demo"))
	require.Less(t, strings.Index(output, "124 tenant demo"), strings.Index(output, "element-124"))
	require.Less(t, strings.Index(output, "element-124"), strings.Index(output, "125 tenant demo"))
	require.Less(t, strings.Index(output, "125 tenant demo"), strings.Index(output, "element-125"))
}

// TestWalkProcessInstanceCommand_WithListenersChildrenParentAndFlatModes verifies listener enrichment preserves non-default walk ordering.
func TestWalkProcessInstanceCommand_WithListenersChildrenParentAndFlatModes(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantSep     string
		wantFirst   string
		wantSecond  string
		wantElement string
		wantJob     string
	}{
		{
			name:        "children",
			args:        []string{"walk", "process-instance", "--key", "123", "--children", "--with-elements", "--with-listeners"},
			wantSep:     " → \n124 tenant demo",
			wantFirst:   "123 tenant demo",
			wantSecond:  "124 tenant demo",
			wantElement: "element-124 SERVICE_TASK task-124 ACTIVE",
			wantJob:     "job-task-124 TASK_LISTENER lsnr:COMPLETING CREATED tp:audit-task r:1",
		},
		{
			name:        "parent",
			args:        []string{"walk", "process-instance", "--key", "124", "--parent", "--with-elements", "--with-listeners"},
			wantSep:     " ← \n123 tenant demo",
			wantFirst:   "124 tenant demo",
			wantSecond:  "123 tenant demo",
			wantElement: "element-124 SERVICE_TASK task-124 ACTIVE",
			wantJob:     "job-task-124 TASK_LISTENER lsnr:COMPLETING CREATED tp:audit-task r:1",
		},
		{
			name:        "flat",
			args:        []string{"walk", "process-instance", "--key", "123", "--flat", "--with-elements", "--with-listeners"},
			wantSep:     " ⇄ \n124 tenant demo",
			wantFirst:   "123 tenant demo",
			wantSecond:  "124 tenant demo",
			wantElement: "element-124 SERVICE_TASK task-124 ACTIVE",
			wantJob:     "job-task-124 TASK_LISTENER lsnr:COMPLETING CREATED tp:audit-task r:1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newWalkProcessInstanceWithListenersServer(t, &requests, map[string]string{
				"123": walkedElementInstancesSearchJSON(t,
					walkedElementInstanceFixture("element-123", "123", "task-123", false, ""),
				),
				"124": walkedElementInstancesSearchJSON(t,
					walkedElementInstanceFixture("element-124", "124", "task-124", false, ""),
				),
			}, map[string][]string{
				"123": {
					walkedJobSearchJSON(t, map[string]any{"jobKey": "job-exec-123", "kind": "EXECUTION_LISTENER", "listenerEventType": "START", "type": "audit-start", "state": "CREATED", "retries": 3, "processInstanceKey": "123", "elementInstanceKey": "element-123", "elementId": "task-123", "tenantId": "tenant"}),
					walkedJobSearchJSON(t),
				},
				"124": {
					walkedJobSearchJSON(t),
					walkedJobSearchJSON(t, map[string]any{"jobKey": "job-task-124", "kind": "TASK_LISTENER", "listenerEventType": "COMPLETING", "type": "audit-task", "state": "CREATED", "retries": 1, "processInstanceKey": "124", "elementInstanceKey": "element-124", "elementId": "task-124", "tenantId": "tenant"}),
				},
			})
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
			args := append([]string{"--config", cfgPath}, tt.args...)

			output := executeRootForProcessInstanceTest(t, args...)

			require.Contains(t, output, tt.wantSep)
			require.Contains(t, output, tt.wantElement)
			require.Contains(t, output, tt.wantJob)
			require.Contains(t, output, "listeners:")
			require.Less(t, strings.Index(output, tt.wantFirst), strings.Index(output, tt.wantSecond))
			require.Less(t, strings.Index(output, tt.wantElement), strings.Index(output, tt.wantJob))
		})
	}
}

// TestWalkProcessInstanceCommand_ParentWithElementsPreservesAncestryOrder verifies parent traversal enriches the selected row before ancestors.
func TestWalkProcessInstanceCommand_ParentWithElementsPreservesAncestryOrder(t *testing.T) {
	var elementFilters []map[string]any

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/125":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("125", "124", false)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/124":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("124", "123", false)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			elementFilters = append(elementFilters, filter)
			key, _ := filter["processInstanceKey"].(string)
			_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
				walkedElementInstanceFixture("element-"+key, key, "task-"+key, false, ""),
			)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"walk", "process-instance",
		"--key", "125",
		"--parent",
		"--with-elements",
	)

	require.Len(t, elementFilters, 3)
	require.Equal(t, "125", elementFilters[0]["processInstanceKey"])
	require.Equal(t, "124", elementFilters[1]["processInstanceKey"])
	require.Equal(t, "123", elementFilters[2]["processInstanceKey"])
	require.Contains(t, output, "125 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-125 SERVICE_TASK task-125 ACTIVE")
	require.Contains(t, output, "124 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-124 SERVICE_TASK task-124 ACTIVE")
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-123 SERVICE_TASK task-123 ACTIVE")
	require.Less(t, strings.Index(output, "125 tenant demo"), strings.Index(output, "element-125"))
	require.Less(t, strings.Index(output, "element-125"), strings.Index(output, "124 tenant demo"))
	require.Less(t, strings.Index(output, "124 tenant demo"), strings.Index(output, "element-124"))
	require.Less(t, strings.Index(output, "element-124"), strings.Index(output, "123 tenant demo"))
	require.Less(t, strings.Index(output, "123 tenant demo"), strings.Index(output, "element-123"))
}

// TestWalkProcessInstanceCommand_FlatWithElementsPreservesPathSeparators verifies flat family rendering keeps relationship separators around detail sections.
func TestWalkProcessInstanceCommand_FlatWithElementsPreservesPathSeparators(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			key, _ := filter["processInstanceKey"].(string)
			_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
				walkedElementInstanceFixture("element-"+key, key, "task-"+key, false, ""),
			)))
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
		"--flat",
		"--with-elements",
	)

	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-123 SERVICE_TASK task-123 ACTIVE")
	require.Contains(t, output, " ⇄ \n124 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-124 SERVICE_TASK task-124 ACTIVE")
	require.Less(t, strings.Index(output, "element-123"), strings.Index(output, " ⇄ \n124 tenant demo"))
	require.Less(t, strings.Index(output, "124 tenant demo"), strings.Index(output, "element-124"))
}

// TestWalkProcessInstanceCommand_DefaultFamilyWithoutElementsDoesNotSearchElements proves default traversal does not run element lookup.
func TestWalkProcessInstanceCommand_DefaultFamilyWithoutElementsDoesNotSearchElements(t *testing.T) {
	var requests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			t.Fatalf("element lookup should not run without --with-elements: %s %s", r.Method, r.URL.Path)
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
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"GET /v2/process-instances/123",
		"POST /v2/process-instances/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.NotContains(t, output, "elements:")
}

// TestWalkProcessInstanceCommand_WithElementsPartialTraversalPreservesWarning keeps missing-ancestor diagnostics visible after element enrichment.
func TestWalkProcessInstanceCommand_WithElementsPartialTraversalPreservesWarning(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "999", false)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/999":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
				walkedElementInstanceFixture("element-123", "123", "task-123", false, ""),
			)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--verbose",
		"walk", "process-instance",
		"--key", "123",
		"--with-elements",
	)

	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "element-123 SERVICE_TASK task-123 ACTIVE")
	require.Contains(t, output, "one or more parent process instances were not found")
	require.Contains(t, output, "missing ancestor keys: 999")
}

// TestWalkProcessInstanceCommand_WithElementsJSONOutputPreservesTraversalMetadata keeps scripted walk metadata and per-item element arrays together.
func TestWalkProcessInstanceCommand_WithElementsJSONOutputPreservesTraversalMetadata(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			switch filter["processInstanceKey"] {
			case "123":
				_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
					walkedElementInstanceFixture("element-root", "123", "root-task", false, ""),
				)))
			case "124":
				_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t)))
			default:
				t.Fatalf("unexpected element filter: %v", filter)
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
		"--key", "123",
		"--with-elements",
	)

	payload := requireWalkProcessInstanceJSONPayload(t, output)
	require.Equal(t, "family", payload["mode"])
	require.Equal(t, "complete", payload["outcome"])
	require.Equal(t, "123", payload["rootKey"])
	requireJSONItems(t, payload["keys"], 2)
	edges := requireJSONObject(t, payload["edges"])
	requireJSONItems(t, edges["123"], 1)

	items := requireJSONItems(t, payload["items"], 2)
	root := requireJSONObject(t, items[0])
	require.Equal(t, "123", requireJSONObject(t, root["item"])["key"])
	rootElements := requireJSONItems(t, root["elements"], 1)
	require.Equal(t, "element-root", requireJSONObject(t, rootElements[0])["elementInstanceKey"])
	require.Equal(t, "root-task", requireJSONObject(t, rootElements[0])["elementId"])

	child := requireJSONObject(t, items[1])
	require.Equal(t, "124", requireJSONObject(t, child["item"])["key"])
	require.Empty(t, requireJSONItems(t, child["elements"], 0))
}

// TestWalkProcessInstanceCommand_WithListenersJSONOutputPreservesEmptyArraysAndOmitsUnmatchedJobs verifies traversal JSON carries requested listener arrays under elements.
func TestWalkProcessInstanceCommand_WithListenersJSONOutputPreservesEmptyArraysAndOmitsUnmatchedJobs(t *testing.T) {
	var requests []string
	srv := newWalkProcessInstanceWithListenersServer(t, &requests, map[string]string{
		"123": walkedElementInstancesSearchJSON(t,
			walkedElementInstanceFixture("element-root", "123", "root-task", false, ""),
			walkedElementInstanceFixture("element-empty", "123", "empty-task", false, ""),
		),
		"124": walkedElementInstancesSearchJSON(t),
	}, map[string][]string{
		"123": {
			walkedJobSearchJSON(t,
				map[string]any{"jobKey": "job-exec-root", "kind": "EXECUTION_LISTENER", "listenerEventType": "START", "type": "audit-start", "state": "CREATED", "retries": 3, "processInstanceKey": "123", "elementInstanceKey": "element-root", "elementId": "root-task", "tenantId": "tenant"},
				map[string]any{"jobKey": "job-unmatched", "kind": "EXECUTION_LISTENER", "listenerEventType": "END", "type": "audit-end", "state": "CREATED", "retries": 3, "processInstanceKey": "123", "elementInstanceKey": "element-missing", "elementId": "missing", "tenantId": "tenant"},
			),
			walkedJobSearchJSON(t),
		},
		"124": {
			walkedJobSearchJSON(t),
			walkedJobSearchJSON(t),
		},
	})
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"walk", "process-instance",
		"--key", "123",
		"--with-elements",
		"--with-listeners",
	)

	require.Contains(t, strings.Join(requests, ","), "POST /v2/jobs/search")
	payload := requireWalkProcessInstanceJSONPayload(t, output)
	require.Equal(t, "family", payload["mode"])
	items := requireJSONItems(t, payload["items"], 2)
	root := requireJSONObject(t, items[0])
	elements := requireJSONItems(t, root["elements"], 2)
	elementsByKey := map[string]map[string]any{}
	for _, rawElement := range elements {
		element := requireJSONObject(t, rawElement)
		key, _ := element["elementInstanceKey"].(string)
		elementsByKey[key] = element
	}
	firstListeners := requireJSONItems(t, elementsByKey["element-root"]["listeners"], 1)
	require.Equal(t, "job-exec-root", requireJSONObject(t, firstListeners[0])["jobKey"])
	require.Empty(t, requireJSONItems(t, elementsByKey["element-empty"]["listeners"], 0))
	child := requireJSONObject(t, items[1])
	require.Empty(t, requireJSONItems(t, child["elements"], 0))
	require.NotContains(t, output, "job-unmatched")
}

// TestWalkProcessInstanceCommand_WithElementsWithoutListenersSkipsListenerLookup keeps existing element-only walks free of job search calls.
func TestWalkProcessInstanceCommand_WithElementsWithoutListenersSkipsListenerLookup(t *testing.T) {
	var requests []string
	srv := newWalkProcessInstanceWithListenersServer(t, &requests, map[string]string{
		"123": walkedElementInstancesSearchJSON(t,
			walkedElementInstanceFixture("element-root", "123", "root-task", false, ""),
		),
		"124": walkedElementInstancesSearchJSON(t),
	}, nil)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"walk", "process-instance",
		"--key", "123",
		"--with-elements",
	)

	require.NotContains(t, strings.Join(requests, ","), "/v2/jobs/search")
	require.NotContains(t, output, `"listeners"`)
}

// TestWalkProcessInstanceCommand_WithListenersValidation rejects invalid listener combinations before remote work.
func TestWalkProcessInstanceCommand_WithListenersValidation(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "missing elements",
			helper: "TestWalkProcessInstanceCommand_WithListenersWithoutElementsHelper",
			want:   "--with-listeners requires --with-elements",
		},
		{
			name:   "keys only",
			helper: "TestWalkProcessInstanceCommand_WithListenersWithKeysOnlyHelper",
			want:   "--with-listeners cannot be combined with --keys-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, tt.helper, map[string]string{
				"C8VOLT_TEST_CONFIG": cfgPath,
			})
			require.Error(t, err)
			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), tt.want)
			require.NotContains(t, string(output), "127.0.0.1:1")
		})
	}
}

// TestWalkProcessInstanceCommand_WithListenersUnsupportedV87 verifies listener lookup failures surface through the normal command error path.
func TestWalkProcessInstanceCommand_WithListenersUnsupportedV87(t *testing.T) {
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch searchCalls {
		case 0:
			searchCalls++
			_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}],"total":1}`))
		default:
			searchCalls++
			_, _ = w.Write([]byte(`{"items":[],"total":0}`))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_WithListenersUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, 2, searchCalls)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "element search requires Camunda 8.8 or newer")
	require.NotContains(t, string(output), "tenant demo v3")
	require.NotContains(t, string(output), "└─ elements:")
}

// TestWalkProcessInstanceCommand_WithVarsIncidentsAndElementsOutputShowsGroupedSections verifies combined enrichment uses stable section and JSON fields.
func TestWalkProcessInstanceCommand_WithVarsIncidentsAndElementsOutputShowsGroupedSections(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			_, _ = w.Write([]byte(`{"items":[{"elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"Root job failed","errorType":"JOB_NO_RETRIES","incidentKey":"incident-1","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/variables/search":
			_, _ = w.Write([]byte(`{"items":[{"name":"businessKey","value":"2234809392328","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			_, _ = w.Write([]byte(walkedElementInstancesSearchJSON(t,
				walkedElementInstanceFixture("element-root", "123", "root-task", true, "incident-1"),
			)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	humanOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"walk", "process-instance",
		"--key", "123",
		"--children",
		"--with-vars",
		"--with-incidents",
		"--with-elements",
	)

	require.Contains(t, humanOutput, "├─ vars:\n│  └─ businessKey=2234809392328")
	require.Contains(t, humanOutput, "├─ incidents:\n│  └─ incident-1 JOB_NO_RETRIES ACTIVE j:n/a e:task-a ei:element-123 m:Root job failed")
	require.Contains(t, humanOutput, "└─ elements:\n   └─ element-root SERVICE_TASK root-task ACTIVE")
	require.Less(t, strings.Index(humanOutput, "├─ vars:"), strings.Index(humanOutput, "├─ incidents:"))
	require.Less(t, strings.Index(humanOutput, "├─ incidents:"), strings.Index(humanOutput, "└─ elements:"))

	jsonOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"walk", "process-instance",
		"--key", "123",
		"--children",
		"--with-vars",
		"--with-incidents",
		"--with-elements",
	)

	payload := requireWalkProcessInstanceJSONPayload(t, jsonOutput)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	require.Equal(t, "123", requireJSONObject(t, first["item"])["key"])
	require.Equal(t, "businessKey", requireJSONObject(t, requireJSONItems(t, first["variables"], 1)[0])["name"])
	require.Equal(t, "incident-1", requireJSONObject(t, requireJSONItems(t, first["incidents"], 1)[0])["incidentKey"])
	require.Equal(t, "element-root", requireJSONObject(t, requireJSONItems(t, first["elements"], 1)[0])["elementInstanceKey"])
}

// TestWalkProcessInstanceCommand_RejectsKeysOnlyWithElements rejects output modes that cannot carry element details before remote work.
func TestWalkProcessInstanceCommand_RejectsKeysOnlyWithElements(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_RejectsKeysOnlyWithElementsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "--with-elements cannot be combined with --keys-only")
	require.NotContains(t, string(output), "127.0.0.1:1")
}

// TestWalkProcessInstanceCommand_WithElementsUnsupportedV87 preserves the reused element-service unsupported-version boundary.
func TestWalkProcessInstanceCommand_WithElementsUnsupportedV87(t *testing.T) {
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		request := decodeCapturedPISearchRequest(t, string(body))
		filter, _ := request["filter"].(map[string]any)

		w.Header().Set("Content-Type", "application/json")
		switch {
		case searchCalls == 0:
			require.NotContains(t, filter, "parentKey")
			searchCalls++
			_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}],"total":1}`))
		case filter["parentKey"] == float64(123):
			searchCalls++
			_, _ = w.Write([]byte(`{"items":[],"total":0}`))
		default:
			t.Fatalf("unexpected search body: %s", string(body))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_WithElementsUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, 2, searchCalls)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "element search requires Camunda 8.8 or newer")
	require.NotContains(t, string(output), "tenant demo v3")
	require.NotContains(t, string(output), "└─ elements:")
}

// TestWalkProcessInstanceCommand_WithElementsLookupFailureDoesNotRenderPartialTraversal fails before showing partial enriched output.
func TestWalkProcessInstanceCommand_WithElementsLookupFailureDoesNotRenderPartialTraversal(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"title":"element lookup failed","status":500,"detail":"element lookup failed"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_WithElementsLookupFailureDoesNotRenderPartialTraversalHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	require.Contains(t, string(output), "element lookup failed")
	require.NotContains(t, string(output), "tenant demo v3")
	require.NotContains(t, string(output), "└─ elements:")
}

// TestWalkProcessInstanceCommand_RejectsWithIncidentsWithoutKey keeps incident enrichment scoped to keyed walks.
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

func TestWalkIncidentLines_RenderGroupedIncidentDetails(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	}
	prevLimit := flagGetPIIncidentMessageLimit
	flagGetPIIncidentMessageLimit = 0
	t.Cleanup(func() {
		relativeDayNow = prevNow
		flagGetPIIncidentMessageLimit = prevLimit
	})

	var out strings.Builder
	writeIncidentLines(&out, "  ", []incident.ProcessInstanceIncidentDetail{{
		IncidentKey:        "incident-1",
		CreationTime:       "2026-05-06T09:29:42.711Z",
		ErrorMessage:       "Root job failed",
		ElementId:          "task-a",
		ElementInstanceKey: "element-123",
		State:              "ACTIVE",
		ErrorType:          "JOB_NO_RETRIES",
		JobKey:             "job-123",
	}})

	require.Equal(t, "\n  └─ incident-1 JOB_NO_RETRIES ACTIVE j:job-123 2026-05-06T09:29:42.711 (4 days ago) e:task-a ei:element-123 m:Root job failed", out.String())
	require.NotContains(t, out.String(), "incident incident-1:")
	require.NotContains(t, out.String(), "fn:")
	require.NotContains(t, out.String(), "fni:")
}

// TestWalkProcessInstanceCommand_WithIncidentsChildrenHumanOutputShowsIncident renders incident keys under child-walk rows.
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
		"--incident-message-limit", "7",
	)

	require.Equal(t, []string{"/v2/process-instances/123/incidents/search"}, incidentRequests)
	require.Contains(t, output, "123")
	require.Contains(t, output, "inc!")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-1 JOB_NO_RETRIES ACTIVE j:n/a m:Root jo...")
	require.NotContains(t, output, "Root job failed")
}

func TestWalkProcessInstanceCommand_WithVarsAndIncidentsChildrenHumanOutputShowsGroupedSections(t *testing.T) {
	var requests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
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
			_, _ = w.Write([]byte(`{"items":[{"elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"Root job failed","errorType":"JOB_NO_RETRIES","incidentKey":"incident-1","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/variables/search":
			require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
			_, _ = w.Write([]byte(`{"items":[{"name":"businessKey","value":"2234809392328","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"},{"name":"hasIncident","value":"true","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
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
		"--with-vars",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/variables/search",
	}, requests)
	require.Contains(t, output, "123")
	require.Contains(t, output, "├─ vars:")
	require.Contains(t, output, "│  ├─ businessKey=2234809392328")
	require.Contains(t, output, "│  └─ hasIncident=true")
	require.Contains(t, output, "└─ incidents:")
	require.Contains(t, output, "   └─ incident-1 JOB_NO_RETRIES ACTIVE j:n/a e:task-a ei:element-123 m:Root job failed")
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "└─ incidents:"))
}

func TestWalkProcessInstanceCommand_WithVarsOnlyDoesNotShowIncidentSectionForIncidentMarker(t *testing.T) {
	var requests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), `"parentProcessInstanceKey":"123"`)
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/variables/search":
			require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
			_, _ = w.Write([]byte(`{"items":[{"name":"hasIncident","value":"true","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case strings.Contains(r.URL.Path, "/incidents/search"):
			t.Fatalf("incident lookup should not run without --with-incidents: %s %s", r.Method, r.URL.Path)
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
		"--with-vars",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"POST /v2/process-instances/search",
		"POST /v2/variables/search",
	}, requests)
	require.Contains(t, output, "123")
	require.Contains(t, output, "└─ vars:")
	require.Contains(t, output, "   └─ hasIncident=true")
	require.NotContains(t, output, "incidents:")
	require.NotContains(t, output, "process instance is marked as having incidents")
}

// TestWalkProcessInstanceCommand_WithIncidentsFamilyHumanOutputShowsMultipleIncidents keeps incidents attached to their walked owners.
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
		"--flat",
		"--with-incidents",
	)

	require.ElementsMatch(t, []string{
		"/v2/process-instances/123/incidents/search",
		"/v2/process-instances/124/incidents/search",
	}, incidentRequests)
	require.Contains(t, output, "123")
	require.Contains(t, output, "124")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-1 JOB_NO_RETRIES ACTIVE j:n/a m:Root failed")
	require.Contains(t, output, "└─ incidents:\n   ├─ incident-1 JOB_NO_RETRIES ACTIVE j:n/a m:Child failed")
	require.Contains(t, output, "└─ incident-2 JOB_NO_RETRIES ACTIVE j:n/a m:Child timed out")
}

// TestWalkProcessInstanceCommand_WithIncidentsParentHumanOutputOmitsIncidentLinesWhenNoneReturned avoids implying missing details exist.
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

	require.ElementsMatch(t, []string{
		"/v2/process-instances/124/incidents/search",
		"/v2/process-instances/123/incidents/search",
	}, incidentRequests)
	require.Contains(t, output, "124")
	require.Contains(t, output, "123")
	require.NotContains(t, output, "  inc ")
}

// TestWalkProcessInstanceCommand_WithIncidentsJSONOutputShowsIncidentDetails preserves incident details in traversal JSON.
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
			_, _ = w.Write([]byte(`{"items":[{"elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"Root job failed","errorType":"JOB_NO_RETRIES","incidentKey":"incident-1","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
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
	require.Equal(t, "task-a", incident["elementId"])
	require.Equal(t, "element-123", incident["elementInstanceKey"])
	require.NotContains(t, output, "flowNode")
}

// TestWalkProcessInstanceCommand_WithIncidentsJSONOutputAssociatesMultipleKeys prevents cross-key incident leakage.
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

// TestWalkProcessInstanceCommand_WithIncidentsJSONOutputShowsEmptyIncidentCollection keeps empty enrichment explicit.
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

// TestWalkProcessInstanceCommand_WithIncidentsJSONOutputPreservesTraversalMetadata keeps walk metadata intact after enrichment.
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

// TestWalkProcessInstanceCommand_DefaultChildrenHumanOutputUnchangedWithoutWithIncidents protects the default path renderer.
func TestWalkProcessInstanceCommand_DefaultChildrenHumanOutputUnchangedWithoutWithIncidents(t *testing.T) {
	incidentRequests := 0

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
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSONWithParentElement("124", "123", "ei-parent", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case strings.Contains(r.URL.Path, "/incidents/search"):
			incidentRequests++
			t.Fatalf("incident lookup should not run without --with-incidents: %s %s", r.Method, r.URL.Path)
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
	)

	root := walkedProcessInstanceModel("123", "", true)
	child := walkedProcessInstanceModel("124", "123", false)
	require.Equal(t, oneLinePI(root)+" → \n"+oneLinePI(child)+"\n", output)
	require.Zero(t, incidentRequests)
	require.NotContains(t, output, "  inc ")
}

// TestWalkProcessInstanceCommand_DefaultJSONOutputUnchangedWithoutWithIncidents protects the default traversal JSON shape.
func TestWalkProcessInstanceCommand_DefaultJSONOutputUnchangedWithoutWithIncidents(t *testing.T) {
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
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSONWithParentElement("124", "123", "ei-parent", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case strings.Contains(r.URL.Path, "/incidents/search"):
			t.Fatalf("incident lookup should not run without --with-incidents: %s %s", r.Method, r.URL.Path)
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
	)

	payload := requireWalkProcessInstanceJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 2)
	first := requireJSONObject(t, items[0])
	require.Equal(t, "123", first["key"])
	require.NotContains(t, first, "incidents")
	second := requireJSONObject(t, items[1])
	require.Equal(t, "124", second["key"])
	require.Equal(t, "ei-parent", second["parentElementInstanceKey"])
	require.NotContains(t, second, "incidents")
	require.NotContains(t, second, "parentFlowNodeInstanceKey")
	require.NotContains(t, output, "parentFlowNodeInstanceKey")
}

// TestWalkProcessInstanceCommand_DefaultFamilyTreeLayoutUnchangedWithoutWithIncidents protects the plain tree renderer.
func TestWalkProcessInstanceCommand_DefaultFamilyTreeLayoutUnchangedWithoutWithIncidents(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t,
					walkedProcessInstanceJSON("124", "123", false),
					walkedProcessInstanceJSON("125", "123", false),
				)))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`),
				strings.Contains(string(body), `"parentProcessInstanceKey":"125"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case strings.Contains(r.URL.Path, "/incidents/search"):
			t.Fatalf("incident lookup should not run without --with-incidents: %s %s", r.Method, r.URL.Path)
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
	)

	root := walkedProcessInstanceModel("123", "", false)
	firstChild := walkedProcessInstanceModel("124", "123", false)
	secondChild := walkedProcessInstanceModel("125", "123", false)
	require.Equal(t, oneLinePI(root)+"\n"+"├─ "+oneLinePI(firstChild)+"\n"+"└─ "+oneLinePI(secondChild)+"\n", output)
	require.NotContains(t, output, "  inc ")
}

// TestWalkProcessInstanceCommand_WithIncidentsFamilyTreeOutputShowsIncidentUnderMatchingNode preserves tree ownership and indentation.
func TestWalkProcessInstanceCommand_WithIncidentsFamilyTreeOutputShowsIncidentUnderMatchingNode(t *testing.T) {
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
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "124", "Child failed")))
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
		"--with-incidents",
	)

	require.Contains(t, output, "123")
	require.Contains(t, output, "├─ incidents:")
	require.Contains(t, output, "│  └─ incident-1 JOB_NO_RETRIES ACTIVE j:n/a m:Root failed")
	require.Contains(t, output, "└─ ")
	require.Contains(t, output, "124")
	require.Contains(t, output, "   └─ incidents:")
	require.Contains(t, output, "      └─ incident-1 JOB_NO_RETRIES ACTIVE j:n/a m:Child failed")
	require.Less(t, strings.Index(output, "123"), strings.Index(output, "Root failed"))
	require.Less(t, strings.Index(output, "124"), strings.Index(output, "Child failed"))
}

// TestWalkProcessInstanceCommand_WithIncidentsPartialTraversalPreservesWarning keeps traversal warnings visible after enrichment.
func TestWalkProcessInstanceCommand_WithIncidentsPartialTraversalPreservesWarning(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "999", false)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/999":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
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
		"--verbose",
		"walk", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	require.Contains(t, output, "123")
	require.Contains(t, output, "one or more parent process instances were not found")
	require.Contains(t, output, "missing ancestor keys: 999")
}

// TestWalkProcessInstanceCommand_RejectsKeysOnlyWithIncidents rejects output modes that cannot carry incident details.
func TestWalkProcessInstanceCommand_RejectsKeysOnlyWithIncidents(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_RejectsKeysOnlyWithIncidentsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "--with-incidents cannot be combined with --keys-only")
	require.NotContains(t, string(output), "127.0.0.1:1")
}

// TestWalkProcessInstanceCommand_WithIncidentsLookupFailureDoesNotRenderPartialTraversal fails before showing partial enriched output.
func TestWalkProcessInstanceCommand_WithIncidentsLookupFailureDoesNotRenderPartialTraversal(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"title":"incident lookup failed","status":500,"detail":"incident lookup failed"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_WithIncidentsLookupFailureDoesNotRenderPartialTraversalHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	require.Contains(t, string(output), "incident lookup failed")
	require.NotContains(t, string(output), "tenant demo v3")
	require.NotContains(t, string(output), "inc!")
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
		require.Contains(t, output, "one or more parent process instances were not found")
		require.Contains(t, output, "missing ancestor keys: 1 (use --verbose to list keys)")
		require.NotContains(t, output, "missing ancestor keys: 999")
	})

	t.Run("family tree renders resolved nodes with warning", func(t *testing.T) {
		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--verbose",
			"walk", "process-instance",
			"--key", "123",
		)

		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "one or more parent process instances were not found")
		require.Contains(t, output, "missing ancestor keys: 999")
	})

	t.Run("json output exposes partial traversal metadata", func(t *testing.T) {
		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"walk", "process-instance",
			"--key", "123",
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

func TestWalkProcessInstanceCommand_KeyTraversalOmitsSelectedTenant(t *testing.T) {
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
		require.NotContains(t, filter, "tenantId")
	}
	require.Contains(t, output, `"tenantId": "tenant-a"`)
	require.NotContains(t, output, "base-tenant")
}

// TestWalkProcessInstanceCommand_KeyTenantMismatchUsesAdminTraversal ensures
// explicit-key walks do not apply the selected tenant to backend traversal.
func TestWalkProcessInstanceCommand_KeyTenantMismatchUsesAdminTraversal(t *testing.T) {
	var searchRequests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+tenantAdminKeysProcessInstanceKey:
			_, _ = w.Write([]byte(tenantAdminKeysMismatchProcessInstanceJSON()))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			searchRequests = append(searchRequests, string(body))
			request := decodeCapturedPISearchRequest(t, string(body))
			filter, ok := request["filter"].(map[string]any)
			require.True(t, ok, "expected search request filter object")
			require.Equal(t, tenantAdminKeysProcessInstanceKey, filter["parentProcessInstanceKey"])
			require.NotContains(t, filter, "tenantId")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", tenantAdminKeysSelectedTenant,
		"--json",
		"walk", "process-instance",
		"--key", tenantAdminKeysProcessInstanceKey,
		"--children",
	)

	require.Len(t, searchRequests, 1)
	require.Contains(t, output, `"tenantId": "tenant-b"`)
	require.Contains(t, output, `"key": "`+tenantAdminKeysProcessInstanceKey+`"`)
}

// TestWalkProcessInstanceCommand_WithIncidentsUsesEffectiveTenantForIncidentSearches applies command tenant overrides to incident lookup.
func TestWalkProcessInstanceCommand_WithIncidentsUsesEffectiveTenantForIncidentSearches(t *testing.T) {
	var incidentRequests []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", true)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.NotContains(t, string(body), `"tenantId"`)
			_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/123/incidents/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			incidentRequests = append(incidentRequests, string(body))
			_, _ = w.Write([]byte(walkedIncidentDetailsJSON(t, "123", "Root failed")))
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
		"--tenant", "tenant-a",
		"walk", "process-instance",
		"--key", "123",
		"--children",
		"--with-incidents",
	)

	require.Contains(t, output, "incident-1 JOB_NO_RETRIES ACTIVE j:n/a m:Root failed")
	require.Len(t, incidentRequests, 1)
	body := decodeCapturedPISearchRequest(t, incidentRequests[0])
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok, "expected incident search request filter object")
	require.Equal(t, "tenant-a", filter["tenantId"])
	require.NotContains(t, filter, "processInstanceKey")
}

// TestWalkProcessInstanceCommand_WithIncidentsUnsupportedV87 preserves the tenant-safe version boundary.
func TestWalkProcessInstanceCommand_WithIncidentsUnsupportedV87(t *testing.T) {
	searchCalls := 0
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		request := decodeCapturedPISearchRequest(t, string(body))
		filter, _ := request["filter"].(map[string]any)

		w.Header().Set("Content-Type", "application/json")
		switch {
		case searchCalls == 0:
			require.NotContains(t, filter, "parentKey")
			searchCalls++
			_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant"}],"total":1}`))
		case filter["parentKey"] == float64(123):
			searchCalls++
			_, _ = w.Write([]byte(`{"items":[],"total":0}`))
		default:
			t.Fatalf("unexpected search body: %s", string(body))
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestWalkProcessInstanceCommand_WithIncidentsUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, 2, searchCalls)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "process-instance incident lookup is not tenant-safe in Camunda 8.7")
	require.NotContains(t, string(output), "  inc ")
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

// TestWalkProcessInstanceCommand_RejectsKeysOnlyWithIncidentsHelper exercises invalid key-only enrichment in a subprocess.
func TestWalkProcessInstanceCommand_RejectsKeysOnlyWithIncidentsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--keys-only", "walk", "process-instance", "--key", "123", "--with-incidents"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_RejectsKeysOnlyWithElementsHelper exercises invalid key-only element enrichment in a subprocess.
func TestWalkProcessInstanceCommand_RejectsKeysOnlyWithElementsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--keys-only", "walk", "process-instance", "--key", "123", "--with-elements"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_WithIncidentsLookupFailureDoesNotRenderPartialTraversalHelper exercises lookup failure in a subprocess.
func TestWalkProcessInstanceCommand_WithIncidentsLookupFailureDoesNotRenderPartialTraversalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "walk", "process-instance", "--key", "123", "--children", "--with-incidents"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_WithElementsLookupFailureDoesNotRenderPartialTraversalHelper exercises element lookup failure in a subprocess.
func TestWalkProcessInstanceCommand_WithElementsLookupFailureDoesNotRenderPartialTraversalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "walk", "process-instance", "--key", "123", "--children", "--with-elements"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_WithIncidentsUnsupportedV87Helper exercises unsupported v8.7 enrichment in a subprocess.
func TestWalkProcessInstanceCommand_WithIncidentsUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "walk", "process-instance", "--key", "123", "--children", "--with-incidents"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_WithElementsUnsupportedV87Helper exercises unsupported v8.7 element enrichment in a subprocess.
func TestWalkProcessInstanceCommand_WithElementsUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "walk", "process-instance", "--key", "123", "--children", "--with-elements"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_WithListenersWithoutElementsHelper exercises missing listener element context in a subprocess.
func TestWalkProcessInstanceCommand_WithListenersWithoutElementsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "walk", "process-instance", "--key", "123", "--with-listeners"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_WithListenersWithKeysOnlyHelper exercises invalid key-only listener enrichment in a subprocess.
func TestWalkProcessInstanceCommand_WithListenersWithKeysOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--keys-only", "walk", "process-instance", "--key", "123", "--with-elements", "--with-listeners"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestWalkProcessInstanceCommand_WithListenersUnsupportedV87Helper exercises unsupported listener enrichment in a subprocess.
func TestWalkProcessInstanceCommand_WithListenersUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "walk", "process-instance", "--key", "123", "--children", "--with-elements", "--with-listeners"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// walkedProcessInstanceModel builds the canonical process-instance fixture used by walk renderer assertions.
func walkedProcessInstanceModel(key, parentKey string, hasIncident bool) process.ProcessInstance {
	return process.ProcessInstance{
		Key:            key,
		ParentKey:      parentKey,
		TenantId:       "tenant",
		BpmnProcessId:  "demo",
		ProcessVersion: 3,
		StartDate:      "2026-03-23T18:00:00Z",
		State:          process.StateActive,
		Incident:       hasIncident,
	}
}

// walkedProcessInstanceJSON builds a v8.8/v8.9 process-instance response fixture.
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

func walkedProcessInstanceJSONWithParentElement(key, parentKey, parentElementKey string, hasIncident bool) string {
	raw := walkedProcessInstanceJSON(key, parentKey, hasIncident)
	if parentElementKey == "" {
		return raw
	}
	return strings.TrimSuffix(raw, "}") + `,"parentElementInstanceKey":"` + parentElementKey + `"}`
}

// walkedProcessInstanceSearchJSON wraps process-instance fixtures in the generated search response shape.
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

// walkedIncidentDetailsJSON builds incident search fixtures with stable incident keys.
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

// walkedElementInstanceFixture builds runtime element rows with stable process-instance ownership for walk enrichment tests.
func walkedElementInstanceFixture(elementInstanceKey, processInstanceKey, elementID string, hasIncident bool, incidentKey string) map[string]any {
	item := map[string]any{
		"elementInstanceKey":     elementInstanceKey,
		"elementId":              elementID,
		"elementName":            elementID,
		"type":                   "SERVICE_TASK",
		"state":                  "ACTIVE",
		"startDate":              "2026-07-15T10:12:04Z",
		"processInstanceKey":     processInstanceKey,
		"rootProcessInstanceKey": "123",
		"processDefinitionId":    "demo",
		"processDefinitionKey":   "9001",
		"tenantId":               "tenant",
		"hasIncident":            hasIncident,
	}
	if incidentKey != "" {
		item["incidentKey"] = incidentKey
	}
	return item
}

// walkedJobSearchJSON wraps runtime job fixtures in the generated search response shape.
func walkedJobSearchJSON(t *testing.T, items ...map[string]any) string {
	t.Helper()

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

// walkedElementInstancesSearchJSON wraps runtime element fixtures in the generated search response shape.
func walkedElementInstancesSearchJSON(t *testing.T, items ...map[string]any) string {
	t.Helper()

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

// newWalkProcessInstanceWithListenersServer serves traversal, element, and
// listener-job responses keyed by process instance for walk enrichment tests.
func newWalkProcessInstanceWithListenersServer(t *testing.T, requests *[]string, elementResponses map[string]string, jobResponses map[string][]string) *httptest.Server {
	t.Helper()

	jobCounts := map[string]int{}
	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*requests = append(*requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/123":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("123", "", false)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/124":
			_, _ = w.Write([]byte(walkedProcessInstanceJSON("124", "123", false)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			switch {
			case strings.Contains(string(body), `"parentProcessInstanceKey":"123"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t, walkedProcessInstanceJSON("124", "123", false))))
			case strings.Contains(string(body), `"parentProcessInstanceKey":"124"`):
				_, _ = w.Write([]byte(walkedProcessInstanceSearchJSON(t)))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodPost && r.URL.Path == "/v2/element-instances/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			key, _ := filter["processInstanceKey"].(string)
			response, ok := elementResponses[key]
			require.True(t, ok, "missing element response for process instance %s", key)
			_, _ = w.Write([]byte(response))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/jobs/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			key, _ := filter["processInstanceKey"].(string)
			responses, ok := jobResponses[key]
			require.True(t, ok, "missing listener response for process instance %s", key)
			responseIndex := jobCounts[key]
			require.Less(t, responseIndex, len(responses), "unexpected extra listener lookup for process instance %s", key)
			jobCounts[key] = responseIndex + 1
			_, _ = w.Write([]byte(responses[responseIndex]))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// requireWalkProcessInstanceJSONPayload unwraps the shared JSON envelope for walk command assertions.
func requireWalkProcessInstanceJSONPayload(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "walk process-instance", envelope["command"])
	return requireJSONObject(t, envelope["payload"])
}
