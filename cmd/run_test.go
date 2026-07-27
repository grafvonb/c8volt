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
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestRunCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "22s")

	cfg := resolveCommandConfigForTest(t, runCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "44s"))
	})

	require.Equal(t, 44*time.Second, cfg.App.Backoff.Timeout)
}

func TestRunHelp_DocumentsWaitAndVerificationRouting(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"run"}, []string{
		"Start process instances",
		"waits until created instances are observable",
		"./c8volt run process-instance --bpmn-process-id <bpmn-process-id>",
	}, nil)

	require.Contains(t, output, "process-instance")

	output = assertCommandHelpOutput(t, []string{"run", "process-instance"}, []string{
		"Run by BPMN process ID",
		"waits until created instances are observable",
		"./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --count 3 --workers 2",
		"./c8volt run process-instance --bpmn-process-id <bpmn-process-id> --keys-only | ./c8volt expect process-instance --state completed -",
	}, nil)
	require.Contains(t, output, "--no-wait")
}

func TestRunProcessInstanceCommand_RegressionPreservesSelectorAndWorkerContract(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)

	capability := commandCapabilityForCommand(runProcessInstanceCmd)

	require.Equal(t, "run process-instance", capability.Path)
	require.Equal(t, CommandMutationStateChanging, capability.Mutation)
	require.Equal(t, ContractSupportFull, capability.ContractSupport)
	require.Equal(t, AutomationSupportFull, capability.AutomationSupport)
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "BPMN process ID(s) to run process instance for (mutually exclusive with --pd-key). Runs latest version unless --pd-version is specified",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "pd-key",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "specific process definition key(s) to run process instance for (mutually exclusive with --bpmn-process-id)",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "count",
		Shorthand:   "n",
		Type:        "int",
		Required:    false,
		Repeated:    false,
		Description: "number of instances to start for a single process definition",
	})
	require.Contains(t, capability.Flags, FlagContract{
		Name:        "no-wait",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return after creation is accepted",
	})
}

// Verifies run commands consume the profile selected by the root flag for tenant and API URL resolution.
func TestRunProcessInstanceCommand_ProfileFlagSelectsProfileTenantAndBaseURL(t *testing.T) {
	baseSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("base profile server should not be used: %s %s", r.Method, r.URL.Path)
	}))
	t.Cleanup(baseSrv.Close)

	prodSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"profile-tenant","version":1}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			require.Equal(t, http.MethodPost, r.Method)
			defer r.Body.Close()
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "profile-tenant", body["tenantId"])
			require.Equal(t, "order-process", body["processDefinitionId"])
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionVersion":1,"tenantId":"profile-tenant","variables":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(prodSrv.Close)

	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+baseSrv.URL+`
profiles:
  prod:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: `+prodSrv.URL+`
`)

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_ProfileFlagSelectsProfileTenantAndBaseURLHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err, string(output))
}

// Verifies run process-instance rejects mutually exclusive definition selectors.
func TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlags(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlagsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "flags --pd-key and --bpmn-process-id are mutually exclusive")
}

func TestRunProcessInstanceCommand_JSONInvalidInputUsesEnvelope(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_JSONInvalidInputUsesEnvelopeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())

	var got map[string]any
	require.NoError(t, json.Unmarshal(output, &got))
	require.Equal(t, string(OutcomeInvalid), got["outcome"])
	require.Equal(t, "run process-instance", got["command"])
}

// Verifies run process-instance maps HTTP 409 responses to the conflict exit code.
func TestRunProcessInstanceCommand_ConflictUsesConflictExitCode(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			http.Error(w, "already exists", http.StatusConflict)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_ConflictUsesConflictExitCodeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Conflict, exitErr.ExitCode())
	require.Contains(t, string(output), "conflict")
	require.Contains(t, string(output), "running process instance(s)")
}

func TestRunProcessInstanceCommand_V89NoWait(t *testing.T) {
	var sawRun bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			sawRun = true
			defer r.Body.Close()
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "order-process", body["processDefinitionId"])
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	)

	require.True(t, sawRun)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "run process-instance", got["command"])
	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	require.EqualValues(t, 1, payload["total"])
	require.Contains(t, stderr, "INFO")
}

func TestRunProcessInstanceCommand_VarsPayloadRemainsCreationInput(t *testing.T) {
	var sawRun bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			sawRun = true
			defer r.Body.Close()
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "order-process", body["processDefinitionId"])
			require.Equal(t, map[string]any{"customerId": "1234", "priority": float64(2)}, body["variables"])
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","tenantId":"<default>","variables":{"customerId":"1234","priority":2}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--vars", `{"customerId":"1234","priority":2}`,
		"--no-wait",
	)

	require.True(t, sawRun)
	require.Contains(t, output, "2251799813711967")
}

// Verifies normal run output shows the state observed by creation confirmation.
func TestRunProcessInstanceCommand_NormalOutputRendersObservedState(t *testing.T) {
	srv := newRunProcessInstanceObservedStateServer(t, "COMPLETED")
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
	)

	require.Contains(t, stdout, "2251799813711967")
	require.Contains(t, stdout, "order-process")
	require.Contains(t, stdout, "COMPLETED")
	require.Contains(t, stdout, "found: 1")
	require.NotContains(t, stdout, "(today)")
	require.NotContains(t, stdout, `"outcome"`)
	require.Contains(t, stderr, "waiting for pi 2251799813711967")
}

// Verifies JSON run output keeps the full command envelope and includes the observed state.
func TestRunProcessInstanceCommand_JSONEnvelopeIncludesObservedState(t *testing.T) {
	srv := newRunProcessInstanceObservedStateServer(t, "COMPLETED")
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
	)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "run process-instance", got["command"])
	payload := requireJSONObject(t, got["payload"])
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813711967", item["key"])
	require.Equal(t, "COMPLETED", item["state"])
}

func TestRunProcessInstanceCommand_JSONNoWaitOmitsUnobservedState(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","state":"ACTIVE","tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	payload := requireJSONObject(t, got["payload"])
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "2251799813711967", item["key"])
	require.NotContains(t, item, "state")
}

// Verifies keys-only run output stays suitable for strict downstream expect pipelines.
func TestRunProcessInstanceCommand_KeysOnlyOutputsOnlyCreatedKeys(t *testing.T) {
	srv := newRunProcessInstanceObservedStateServer(t, "COMPLETED")
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--keys-only",
	)

	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	require.Equal(t, []string{"2251799813711967"}, lines)
	require.NotContains(t, stdout, "COMPLETED")
	require.NotContains(t, stdout, "found:")
	require.NotContains(t, stdout, `"outcome"`)
	require.Contains(t, stderr, "waiting for pi 2251799813711967")
}

// TestRunProcessInstanceCommand_VerboseBulkProgressRendersCounters verifies explicit --count progress is routed to stderr.
func TestRunProcessInstanceCommand_VerboseBulkProgressRendersCounters(t *testing.T) {
	var created testx.SafeSlice[string]
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v2/process-instances":
			require.Equal(t, http.MethodPost, r.Method)
			created.Append(r.URL.Path)
			key := "2251799813685248"
			if len(created.Snapshot()) == 2 {
				key = "2251799813685249"
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(processInstanceCreationJSON(key)))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--verbose",
		"run", "process-instance",
		"--pd-key", "pd-88",
		"--count", "2",
		"--workers", "1",
		"--no-wait",
	)

	require.Contains(t, stdout, "found: 2")
	require.Contains(t, stderr, "starting process instances 2/2 process instance(s)")
	require.Len(t, created.Snapshot(), 2)
}

func TestRunProcessInstanceResultSortsCountOutputByStartDateAndKey(t *testing.T) {
	items := []process.ProcessInstance{
		{Key: "30", StartDate: "2026-05-23T18:16:52.711Z"},
		{Key: "10", StartDate: "2026-05-23T18:16:52.705Z"},
		{Key: "20", StartDate: "2026-05-23T18:16:52.705Z"},
		{Key: "5", StartDate: ""},
	}

	sortRunProcessInstancesForOutput(items)

	require.Equal(t, []process.ProcessInstance{
		{Key: "10", StartDate: "2026-05-23T18:16:52.705Z"},
		{Key: "20", StartDate: "2026-05-23T18:16:52.705Z"},
		{Key: "30", StartDate: "2026-05-23T18:16:52.711Z"},
		{Key: "5", StartDate: ""},
	}, items)
}

func TestRunProcessInstanceCommand_DefaultOutputDoesNotEmitMachineEnvelope(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","state":"ACTIVE","tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	)

	require.NotContains(t, output, `"outcome"`)
	require.NotContains(t, output, `"command"`)
	require.Contains(t, output, "2251799813711967 <default> order-process v3 s:")
	require.NotContains(t, output, "ACTIVE")
}

// newRunProcessInstanceObservedStateServer returns a v8.9 fixture that confirms creation through a keyed lookup.
func newRunProcessInstanceObservedStateServer(t *testing.T, observedState string) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","tenantId":"<default>","variables":{}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","tenantId":"<default>","state":"` + observedState + `","startDate":"2026-05-23T12:00:00Z","hasIncident":false,"tags":[]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func TestRunProcessInstanceBpmnSelectorPartialMultiIDFailsBeforeCreate(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)

		body := decodeRunProcessDefinitionSearchBody(t, r)
		filter := requireJSONObject(t, body["filter"])
		w.Header().Set("Content-Type", "application/json")
		switch filter["processDefinitionId"] {
		case "order-process":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "missing-process":
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected process definition filter: %v", filter)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceBpmnSelectorPartialMultiIDFailsBeforeCreateHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, []string{"POST /v2/process-definitions/search", "POST /v2/process-definitions/search"}, requests)
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[missing-process]")
	require.NotContains(t, string(output), "bpmnProcessId:")
}

func TestRunProcessInstanceBpmnSelectorMultipleMissingDiagnostics(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceBpmnSelectorMultipleMissingDiagnosticsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, []string{"POST /v2/process-definitions/search", "POST /v2/process-definitions/search"}, requests)
	require.Contains(t, string(output), "no visible process definitions match the provided selector(s)")
	require.Contains(t, string(output), "[missing-a]")
	require.Contains(t, string(output), "[missing-b]")
	require.NotContains(t, string(output), "bpmnProcessId:")
}

func TestRunProcessInstanceBpmnSelectorAllVisiblePreservesCreate(t *testing.T) {
	var requests []string
	var pdSearchBodies []map[string]any
	var createBodies []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v2/process-definitions/search":
			body := decodeRunProcessDefinitionSearchBody(t, r)
			pdSearchBodies = append(pdSearchBodies, body)
			filter := requireJSONObject(t, body["filter"])
			bpmnID, ok := filter["processDefinitionId"].(string)
			require.True(t, ok)
			require.True(t, filter["isLatestVersion"].(bool))
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"` + bpmnID + `","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			createBodies = append(createBodies, body)
			bpmnID, ok := body["processDefinitionId"].(string)
			require.True(t, ok)
			_, _ = w.Write([]byte(`{"processDefinitionId":"` + bpmnID + `","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, _ := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--bpmn-process-id", "invoice-process",
		"--no-wait",
	)

	require.Equal(t, []string{
		"POST /v2/process-definitions/search",
		"POST /v2/process-definitions/search",
		"POST /v2/process-instances",
		"POST /v2/process-instances",
	}, requests)
	require.Len(t, pdSearchBodies, 2)
	require.Len(t, createBodies, 2)
	require.Equal(t, "order-process", createBodies[0]["processDefinitionId"])
	require.Equal(t, "invoice-process", createBodies[1]["processDefinitionId"])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	payload := requireJSONObject(t, got["payload"])
	require.EqualValues(t, 2, payload["total"])
}

func TestRunProcessInstanceBpmnSelectorVersionUsesExactSearch(t *testing.T) {
	var pdSearchBody map[string]any
	var createBody map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v2/process-definitions/search":
			pdSearchBody = decodeRunProcessDefinitionSearchBody(t, r)
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":7}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&createBody))
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":7,"processInstanceKey":"2251799813711967","tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	_ = executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--pd-version", "7",
		"--no-wait",
	)

	filter := requireJSONObject(t, pdSearchBody["filter"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, float64(7), filter["version"])
	require.NotContains(t, filter, "isLatestVersion")
	require.Equal(t, "order-process", createBody["processDefinitionId"])
	require.Equal(t, float64(7), createBody["processDefinitionVersion"])
}

func decodeRunProcessDefinitionSearchBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()

	defer r.Body.Close()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(body, &got))
	return got
}

// Helper-process entrypoint for mutually-exclusive definition-flag validation.
func TestRunProcessInstanceCommand_RejectsMutuallyExclusiveDefinitionFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "run", "process-instance", "--pd-key", "2251799813685255", "--bpmn-process-id", "order-process"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestRunProcessInstanceBpmnSelectorPartialMultiIDFailsBeforeCreateHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--bpmn-process-id", "missing-process",
		"--auto-confirm",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestRunProcessInstanceBpmnSelectorMultipleMissingDiagnosticsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"run", "process-instance",
		"--bpmn-process-id", "missing-a",
		"--bpmn-process-id", "missing-b",
		"--auto-confirm",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestRunProcessInstanceCommand_JSONInvalidInputUsesEnvelopeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "run", "process-instance", "--pd-key", "2251799813685255", "--bpmn-process-id", "order-process"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for conflict exit-code mapping validation.
func TestRunProcessInstanceCommand_ConflictUsesConflictExitCodeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "run", "process-instance", "--bpmn-process-id", "order-process", "--no-wait"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestRunProcessInstanceCommand_ProfileFlagSelectsProfileTenantAndBaseURLHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	flagRunPIProcessDefinitionKey = nil
	flagRunPIProcessDefinitionBpmnProcessIds = nil

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--profile", "prod",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
