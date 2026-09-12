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
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
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
		"Use a BPMN process ID",
		"waits until created instances are observable",
		"--all-tenants is not supported because",
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

// TestRunProcessInstanceCommand_AllTenantsRejectsBeforeInputValidationOrRequest
// proves process-instance creation rejects all-tenants before variable parsing,
// selector validation, activity, or creation requests can run.
func TestRunProcessInstanceCommand_AllTenantsRejectsBeforeInputValidationOrRequest(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "invalid vars",
			args: []string{"run", "process-instance", "--all-tenants", "--bpmn-process-id", "order-process", "--vars", "{"},
		},
		{
			name: "bpmn selector",
			args: []string{"--all-tenants", "run", "process-instance", "--bpmn-process-id", "order-process"},
		},
		{
			name: "direct process definition key",
			args: []string{"run", "process-instance", "--pd-key", "9001", "--all-tenants"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.Method + " " + r.URL.Path)
				t.Fatalf("run process-instance must reject --all-tenants before request work: %s %s", r.Method, r.URL.Path)
			}))
			t.Cleanup(srv.Close)
			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, tt.args...)

			output, err := testx.RunCmdSubprocess(t, "TestRunProcessInstanceCommand_AllTenantsRejectsBeforeInputValidationOrRequestHelper", map[string]string{
				"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, args),
			})
			assertAllTenantsConcreteDestinationSubprocessFailure(t, output, err, "run process-instance")
			require.Empty(t, requests.Snapshot())
			require.NotContains(t, string(output), "parsing --vars JSON")
			require.NotContains(t, string(output), "creation target:")
			require.NotContains(t, string(output), "running process instance")
		})
	}
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

// TestRunProcessInstanceCommand_V810NoWaitJSONUsesNativeCreation verifies V810 creation preserves the accepted JSON envelope.
func TestRunProcessInstanceCommand_V810NoWaitJSONUsesNativeCreation(t *testing.T) {
	var requests []string
	var createBody map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"<default>","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&createBody))
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","state":"ACTIVE","tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.10")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--no-wait",
	)

	require.Equal(t, []string{"POST /v2/process-definitions/search", "POST /v2/process-instances"}, requests)
	require.Equal(t, "order-process", createBody["processDefinitionId"])
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "run process-instance", got["command"])
	payload := requireJSONObject(t, got["payload"])
	require.EqualValues(t, 1, payload["total"])
	require.NotContains(t, stdout, "found:")
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

// Helper-process entrypoint for all-tenants process-instance run rejection.
func TestRunProcessInstanceCommand_AllTenantsRejectsBeforeInputValidationOrRequestHelper(t *testing.T) {
	executeRootHelperFromArgsEnv(t, "C8VOLT_TEST_ROOT_ARGS")
}

// Verifies run process-instance renders the creation tenant before the backend creation request.
func TestRunProcessInstanceCommand_CreationContextPrecedesCreateRequest(t *testing.T) {
	tests := []struct {
		name       string
		configYAML func(baseURL string) string
		wantLine   string
		wantTenant string
	}{
		{
			name: "named target",
			configYAML: func(baseURL string) string {
				return `app:
  camunda_version: "8.9"
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: ` + baseURL + `
`
			},
			wantLine:   "creation target: tenant-a",
			wantTenant: "tenant-a",
		},
		{
			name: "default target",
			configYAML: func(baseURL string) string {
				return `app:
  camunda_version: "8.9"
auth:
  mode: none
apis:
  camunda_api:
    base_url: ` + baseURL + `
`
			},
			wantLine:   "creation target: default tenant",
			wantTenant: "<default>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			var sawRun bool
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, http.MethodPost, r.Method)
				require.Equal(t, "/v2/process-instances", r.URL.Path)
				sawRun = true
				require.Contains(t, stderr.String(), tt.wantLine)
				defer r.Body.Close()
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, "2251799813685255", body["processDefinitionKey"])
				require.Equal(t, tt.wantTenant, body["tenantId"])
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"processDefinitionKey":"2251799813685255","processInstanceKey":"2251799813711967","tenantId":"` + tt.wantTenant + `","variables":{}}`))
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeRawTestConfig(t, tt.configYAML(srv.URL))
			root := Root()
			resetCommandTreeFlags(root)
			resetDeployCommandContextForTest(root)
			root.SetOut(stdout)
			root.SetErr(stderr)
			root.SetArgs([]string{"--config", cfgPath, "run", "process-instance", "--pd-key", "2251799813685255", "--no-wait"})

			_, err := root.ExecuteC()
			require.NoError(t, err)
			require.True(t, sawRun)
			require.Contains(t, stdout.String(), "2251799813711967")
			require.Contains(t, stderr.String(), tt.wantLine)
			require.NotContains(t, stderr.String(), "Proceed?")
		})
	}
}

// Verifies run process-instance JSON results include creation context without reshaping the payload.
func TestRunProcessInstanceCommand_JSONEnvelopeIncludesCreationContext(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	var sawRun bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances", r.URL.Path)
		sawRun = true
		defer r.Body.Close()
		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "tenant-a", body["tenantId"])
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
	resetDeployCommandContextForTest(Root())

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"run", "process-instance",
		"--pd-key", "2251799813685255",
		"--no-wait",
	)

	require.True(t, sawRun)
	require.NotContains(t, stderr, "creation target:")
	require.NotContains(t, stdout, "creation target:")
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "run process-instance", got["command"])
	tenantContext := requireJSONObject(t, got["tenantContext"])
	require.Equal(t, "creation", tenantContext["mode"])
	require.Equal(t, "not_applicable", tenantContext["filter"])
	require.Equal(t, "tenant-a", tenantContext["targetTenantId"])
	require.Equal(t, []any{"tenant-a"}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(0), tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
	require.Empty(t, tenantContext["warnings"])
	payload := requireJSONObject(t, got["payload"])
	require.EqualValues(t, 1, payload["total"])
	items, ok := payload["items"].([]any)
	require.True(t, ok)
	require.Len(t, items, 1)
}

// Verifies quiet and keys-only run output never mixes tenant context into protected streams.
func TestRunProcessInstanceCommand_ProtectedModesSuppressCreationContext(t *testing.T) {
	tests := []struct {
		name       string
		argsPrefix []string
		wantStdout string
		exact      bool
	}{
		{
			name:       "quiet",
			argsPrefix: []string{"--quiet"},
			wantStdout: "2251799813711967",
		},
		{
			name:       "keys only",
			argsPrefix: []string{},
			wantStdout: "2251799813711967\n",
			exact:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)
			resetDeployCommandContextForTest(Root())
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
			args := append([]string{"--config", cfgPath}, tt.argsPrefix...)
			args = append(args, "run", "process-instance", "--pd-key", "2251799813685255", "--no-wait")
			if tt.name == "keys only" {
				args = append(args, "--keys-only")
			}

			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)

			if tt.exact {
				require.Equal(t, tt.wantStdout, stdout)
			} else {
				require.Contains(t, stdout, tt.wantStdout)
			}
			require.NotContains(t, stdout, "creation target:")
			require.NotContains(t, stderr, "creation target:")
		})
	}
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

// TestRunProcessInstanceCommand_V810KeysOnlyKeepsActivityOffStdout verifies V810 keys-only output remains pipeline-safe while activity is routed away from stdout.
func TestRunProcessInstanceCommand_V810KeysOnlyKeepsActivityOffStdout(t *testing.T) {
	srv := newRunProcessInstanceObservedStateServer(t, "COMPLETED")
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.10")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"run", "process-instance",
		"--bpmn-process-id", "order-process",
		"--keys-only",
	)

	require.Equal(t, "2251799813711967\n", stdout)
	require.NotContains(t, stdout, "COMPLETED")
	require.NotContains(t, stdout, "waiting for pi")
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
		case "/v2/process-instances/2251799813685248":
			require.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(processInstanceJSON("2251799813685248")))
		case "/v2/process-instances/2251799813685249":
			require.Equal(t, http.MethodGet, r.Method)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(processInstanceJSON("2251799813685249")))
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
	)

	require.Contains(t, stdout, "found: 2")
	require.Contains(t, stderr, "starting process instances 2/2 process instance(s)")
	require.NotContains(t, stderr, "waiting for pi 2251799813685248")
	require.NotContains(t, stderr, "waiting for pi 2251799813685249")
	require.NotContains(t, stderr, "created; pd")
	require.NotContains(t, stderr, "create requested")
	require.Len(t, created.Snapshot(), 2)
}

// TestRunProcessInstanceBulkProgressUsesWorkflowImportance verifies explicit-count run progress outranks nested create and confirmation activity.
func TestRunProcessInstanceBulkProgressUsesWorkflowImportance(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	printExplicitLargeWorkProgressEvent(cmd, ops.ProgressEvent{
		Kind: ops.ProgressEventKindFrozenScope,
		FrozenScope: &ops.FrozenScopeProgress{
			Phase:        "starting process instances",
			CoreResource: "process instance(s)",
			Done:         2,
			Total:        5,
		},
	})

	require.Equal(t, []activitysink.Update{{
		Message:    "starting process instances 2/5 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

// TestRunProcessInstanceBulkStartCompletionUsesSemanticWorkflowActivity verifies
// explicit-count starts are eligible for exact semantic completion aggregation.
func TestRunProcessInstanceBulkStartCompletionUsesSemanticWorkflowActivity(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	reporter := newRunProcessInstanceSemanticProgressReporter(cmd, 2)
	defer reporter.Close()
	opts := appendRunProcessInstanceProgressOption(cmd, nil, reporter)
	progress := options.ApplyFacadeOptions(opts).Progress
	require.NotNil(t, progress)

	recordHTTPFallbackActivity(cmd.Context(), "creating process instance request")
	affected := 1
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindCompletion,
		Completion: &options.CompletionProgress{
			Phase:            "create",
			CoreResource:     "process instance(s)",
			Total:            2,
			Identity:         "2251799813711967",
			Disposition:      options.CompletionDispositionConfirmed,
			AffectedResource: "process instances",
			AffectedCount:    &affected,
		},
	})
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "starting process instances",
			CoreResource: "process instance(s)",
			Done:         1,
			Total:        2,
		},
	})
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindCompletion,
		Completion: &options.CompletionProgress{
			Phase:            "create",
			CoreResource:     "process instance(s)",
			Total:            2,
			Identity:         "2251799813711968",
			Disposition:      options.CompletionDispositionConfirmed,
			AffectedResource: "process instances",
			AffectedCount:    &affected,
		},
	})
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "starting process instances",
			CoreResource: "process instance(s)",
			Done:         2,
			Total:        2,
		},
	})

	require.Equal(t, []activitysink.Update{
		{
			Message:    "starting process instances, 1/2 process instance(s), process instances: 1",
			Importance: logging.ActivityImportanceWorkflow,
		},
		{
			Message:    "starting process instances, 2/2 process instance(s), process instances: 2",
			Importance: logging.ActivityImportanceWorkflow,
		},
	}, sink.PriorityUpdates())
	require.Equal(t, opsSemanticProgressAggregate{
		Completed:     2,
		Failed:        0,
		Total:         2,
		Affected:      2,
		AffectedValid: true,
	}, reporter.Aggregate())
}

// TestRunProcessInstanceDefaultStartMilestonesAndFinalFlush verifies explicit
// count starts use default semantic milestones and flush only once on close.
func TestRunProcessInstanceDefaultStartMilestonesAndFinalFlush(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	now := time.Date(2026, 9, 1, 7, 6, 0, 0, time.UTC)
	runProcessInstanceSemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { runProcessInstanceSemanticProgressNow = time.Now })

	cmd, stderr := newSemanticProgressStderrCommand()
	reporter := newRunProcessInstanceSemanticProgressReporter(cmd, 2)
	opts := appendRunProcessInstanceProgressOption(cmd, nil, reporter)
	progress := options.ApplyFacadeOptions(opts).Progress

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportRunProcessInstanceCompletionEvent(progress, "pi-1", 2, options.CompletionDispositionConfirmed, "", ptrInt(1))
	reportRunProcessInstanceCompletionEvent(progress, "pi-2", 2, options.CompletionDispositionConfirmed, "", ptrInt(1))
	reporter.Close()
	reporter.Close()

	output := stderr.String()
	require.Equal(t, 1, strings.Count(output, "starting process instances, 1/2 process instance(s), process instances: 1"))
	require.Equal(t, 1, strings.Count(output, "starting process instances, 2/2 process instance(s), process instances: 2"))
	require.NotContains(t, output, "pi-1 started")
}

// TestRunProcessInstanceVerboseLifecycleVocabulary verifies bulk-start progress
// maps submitted, started, and failed wording in the command layer.
func TestRunProcessInstanceVerboseLifecycleVocabulary(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagVerbose = true

	cmd, stderr := newSemanticProgressStderrCommand()
	reporter := newRunProcessInstanceSemanticProgressReporter(cmd, 3)
	opts := appendRunProcessInstanceProgressOption(cmd, nil, reporter)
	progress := options.ApplyFacadeOptions(opts).Progress
	defer reporter.Close()

	reportRunProcessInstanceCompletionEvent(progress, "pi-1", 3, options.CompletionDispositionSubmitted, "", ptrInt(1))
	reportRunProcessInstanceCompletionEvent(progress, "pi-2", 3, options.CompletionDispositionConfirmed, "", ptrInt(1))
	reportRunProcessInstanceCompletionEvent(progress, "pi-3", 3, options.CompletionDispositionFailed, "start rejected", ptrInt(0))
	reporter.Close()

	output := stderr.String()
	require.Contains(t, output, "pi-1 submitted (starting process instances, 1/3 process instance(s), process instances: 1)")
	require.Contains(t, output, "pi-2 started (starting process instances, 2/3 process instance(s), process instances: 2)")
	require.Contains(t, output, "pi-3 failed: start rejected (starting process instances, 3/3 process instance(s), 1 failed, process instances: 2)")
}

// reportRunProcessInstanceCompletionEvent sends one facade-level bulk-start
// completion fact through the configured run command progress callback.
func reportRunProcessInstanceCompletionEvent(progress func(options.ProgressEvent), identity string, total int, disposition options.CompletionDisposition, detail string, affected *int) {
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindCompletion,
		Completion: &options.CompletionProgress{
			Phase:         "create",
			CoreResource:  "process instance(s)",
			Total:         total,
			Identity:      identity,
			Disposition:   disposition,
			FailureDetail: detail,
			AffectedCount: affected,
		},
	})
}

// TestExplicitLargeWorkSharedAdapterIgnoresCompletionFacts documents that
// walk-style callers remain frozen-scope progress only until a finite semantic
// completion boundary is explicitly added for that command family.
func TestExplicitLargeWorkSharedAdapterIgnoresCompletionFacts(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	opts := appendExplicitLargeWorkProgressOption(cmd, nil)
	progress := options.ApplyFacadeOptions(opts).Progress
	require.NotNil(t, progress)

	affected := 1
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindCompletion,
		Completion: &options.CompletionProgress{
			Phase:         "walking process-instance family",
			Total:         2,
			Identity:      "child",
			Disposition:   options.CompletionDispositionConfirmed,
			AffectedCount: &affected,
		},
	})
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "walking process-instance family",
			CoreResource: "process instance(s)",
			Done:         2,
			Total:        2,
		},
	})

	require.Equal(t, []activitysink.Update{{
		Message:    "walking process-instance family 2/2 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
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
