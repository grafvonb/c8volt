// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestDeployCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "18s")

	cfg := resolveCommandConfigForTest(t, deployCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "41s"))
	})

	require.Equal(t, 41*time.Second, cfg.App.Backoff.Timeout)
}

func TestDeployHelp_DocumentsWaitContractsAndFollowUp(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"deploy"}, []string{
		"Deploy BPMN resources to Camunda",
		"`deploy process-definition`",
		"`embed deploy`",
		"./c8volt deploy process-definition --file ./fixtures/processdefinitions/<embedded-process>.bpmn --run",
		"./c8volt embed deploy --all --run",
	}, nil)
	require.Contains(t, output, "process-definition")

	output = assertCommandHelpOutput(t, []string{"deploy", "process-definition"}, []string{
		"By default c8volt waits for deployment confirmation",
		"Use --run to start one process instance",
		"does not accept --all-tenants because it creates resources in one concrete tenant",
		"./c8volt deploy process-definition --file ./fixtures/processdefinitions/<embedded-process>.bpmn --run",
	}, nil)
	require.Contains(t, output, "--run")
	require.NotContains(t, output, "--expected-status")
}

// TestDeployProcessDefinitionSemanticProgressVerboseItemsReplaceMilestones
// verifies deployment progress writes one verbose line per process definition
// and suppresses paced aggregate duplicates.
func TestDeployProcessDefinitionSemanticProgressVerboseItemsReplaceMilestones(t *testing.T) {
	resetDeployCommandStateForTest()
	t.Cleanup(resetDeployCommandStateForTest)
	prevVerbose := flagVerbose
	flagVerbose = true
	t.Cleanup(func() { flagVerbose = prevVerbose })
	now := time.Date(2026, 9, 1, 6, 31, 0, 0, time.UTC)
	processDefinitionDeploySemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { processDefinitionDeploySemanticProgressNow = time.Now })

	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	progress := newProcessDefinitionDeploySemanticProgress(cmd)

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportProcessDefinitionDeployCompletionEvent(progress, "pd-1", 2, foptions.CompletionDispositionSubmitted, "")
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportProcessDefinitionDeployCompletionEvent(progress, "pd-2", 2, foptions.CompletionDispositionConfirmed, "")
	progress.Close()

	output := stderr.String()
	require.Contains(t, output, "pd-1 submitted (deploying process definitions, 1/2 process definition(s))")
	require.Contains(t, output, "pd-2 deployed (deploying process definitions, 2/2 process definition(s))")
	require.Equal(t, 2, strings.Count(output, "deploying process definitions"))
	require.NotContains(t, output, "\ndeploying process definitions, 1/2")
	require.NotContains(t, output, "\ndeploying process definitions, 2/2")
}

// TestDeployProcessDefinitionSemanticProgressModeGate verifies deployment
// completion progress preserves machine stdout and quiet/automation contracts.
func TestDeployProcessDefinitionSemanticProgressModeGate(t *testing.T) {
	assertOpsCompletionProgressModeGate(t, opsCompletionProgressModeGateCase{
		Configure: func(cmd *cobra.Command) (func(ops.ProgressEvent), func()) {
			progress := newProcessDefinitionDeploySemanticProgress(cmd)
			return progress.Report, progress.Close
		},
		Event: func(disposition ops.CompletionDisposition, detail string) ops.ProgressEvent {
			return ops.ProgressEvent{
				Kind: ops.ProgressEventKindCompletion,
				Completion: &ops.CompletionProgress{
					Phase:         processDefinitionDeployCompletionPhase,
					CoreResource:  "process definition(s)",
					Total:         1,
					Identity:      "pd-1",
					Disposition:   disposition,
					FailureDetail: detail,
				},
			}
		},
		QuietWarning: "pd-1 failed: request rejected (deploying process definitions, 1/1 process definition(s), 1 failed)",
	})
}

func TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_TENANT", "env-tenant")

	var sawDeploy bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.Equal(t, "flag-tenant", r.FormValue("tenantId"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tenantId":"flag-tenant"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `active_profile: dev
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
profiles:
  dev:
    app:
      tenant: profile-tenant
`)
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfigHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":    cfgPath,
		"C8VOLT_TEST_BPMN_PATH": bpmnPath,
	})
	require.NoError(t, err, string(output))
	require.True(t, sawDeploy)
}

// TestDeployProcessDefinitionCommand_AllTenantsRejectsBeforeInputOrRequest
// proves deployment cannot use an all-tenants scope and fails before file or
// stdin resources are inspected.
func TestDeployProcessDefinitionCommand_AllTenantsRejectsBeforeInputOrRequest(t *testing.T) {
	tests := []struct {
		name  string
		args  []string
		stdin string
	}{
		{
			name: "nonexistent file",
			args: []string{"--all-tenants", "deploy", "process-definition", "--file", t.TempDir() + string(os.PathSeparator) + "missing.bpmn"},
		},
		{
			name:  "stdin file",
			args:  []string{"deploy", "process-definition", "--all-tenants", "--file", "-"},
			stdin: validDeployProcessDefinitionBPMN(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.Method + " " + r.URL.Path)
				t.Fatalf("deploy process-definition must reject --all-tenants before request work: %s %s", r.Method, r.URL.Path)
			}))
			t.Cleanup(srv.Close)
			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, tt.args...)

			output, err := testx.RunCmdSubprocessWithStdin(t, "TestDeployProcessDefinitionCommand_AllTenantsRejectsBeforeInputOrRequestHelper", map[string]string{
				"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, args),
			}, tt.stdin)
			assertAllTenantsConcreteDestinationSubprocessFailure(t, output, err, "deploy process-definition")
			require.Empty(t, requests.Snapshot())
			require.NotContains(t, string(output), "validating files with process definition")
			require.NotContains(t, string(output), "collecting process definition")
			require.NotContains(t, string(output), "creation target:")
			require.NotContains(t, string(output), "pd deploy done")
		})
	}
}

func TestDeployProcessDefinitionCommand_RunFallsBackToBPMNIDForV87(t *testing.T) {
	var sawDeploy bool
	var sawRun bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tenantId":"<default>"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			sawRun = true
			defer r.Body.Close()
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.Equal(t, "order-process", body["processDefinitionId"])
			require.Equal(t, "<default>", body["tenantId"])
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionVersion":1,"tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_RunFallsBackToBPMNIDForV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG":    cfgPath,
		"C8VOLT_TEST_BPMN_PATH": bpmnPath,
	})
	require.NoError(t, err, string(output))

	require.True(t, sawDeploy)
	require.True(t, sawRun)
}

func TestDeployProcessDefinitionCommand_V89NoWait(t *testing.T) {
	var sawDeploy bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.Equal(t, "order-process.bpmn", r.MultipartForm.File["resources"][0].Filename)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deploymentKey":"deployment-123","tenantId":"<default>","deployments":[{"processDefinition":{"processDefinitionId":"order-process","processDefinitionKey":"2251799813685255","processDefinitionVersion":3,"resourceName":"order-process.bpmn","tenantId":"<default>"}}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_V89NoWaitHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":    cfgPath,
		"C8VOLT_TEST_BPMN_PATH": bpmnPath,
	})

	require.NoError(t, err, string(output))
	require.True(t, sawDeploy)
	var got map[string]any
	start := bytes.IndexByte(output, '{')
	require.NotEqual(t, -1, start)
	require.NoError(t, json.NewDecoder(bytes.NewReader(output[start:])).Decode(&got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "deploy process-definition", got["command"])
	payload, ok := got["payload"].([]any)
	require.True(t, ok)
	require.Len(t, payload, 1)
}

func TestDeployProcessDefinitionCommand_CreationContextPrecedesDeploymentRequest(t *testing.T) {
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
			resetDeployCommandStateForTest()
			stdout := &bytes.Buffer{}
			stderr := &bytes.Buffer{}
			var sawDeploy bool
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
					sawDeploy = true
					require.Contains(t, stderr.String(), tt.wantLine)
					require.NoError(t, r.ParseMultipartForm(1<<20))
					require.Equal(t, tt.wantTenant, r.FormValue("tenantId"))
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"deploymentKey":"deployment-123","tenantId":"` + tt.wantTenant + `","deployments":[{"processDefinition":{"processDefinitionId":"order-process","processDefinitionKey":"2251799813685255","processDefinitionVersion":3,"resourceName":"order-process.bpmn","tenantId":"` + tt.wantTenant + `"}}]}`))
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeRawTestConfig(t, tt.configYAML(srv.URL))
			bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

			root := Root()
			resetCommandTreeFlags(root)
			root.SetOut(stdout)
			root.SetErr(stderr)
			root.SetArgs([]string{"--config", cfgPath, "deploy", "process-definition", "--file", bpmnPath, "--no-wait"})

			_, err := root.ExecuteC()
			require.NoError(t, err)
			require.True(t, sawDeploy)
			require.Empty(t, stdout.String())
			require.Contains(t, stderr.String(), tt.wantLine)
			require.Less(t, strings.Index(stderr.String(), tt.wantLine), strings.Index(stderr.String(), "pd deploy done"))
		})
	}
}

func TestDeployProcessDefinitionCommand_JSONEnvelopeIncludesCreationContext(t *testing.T) {
	resetDeployCommandStateForTest()
	var sawDeploy bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.Equal(t, "tenant-a", r.FormValue("tenantId"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deploymentKey":"deployment-123","tenantId":"tenant-a","deployments":[{"processDefinition":{"processDefinitionId":"order-process","processDefinitionKey":"2251799813685255","processDefinitionVersion":3,"resourceName":"order-process.bpmn","tenantId":"tenant-a"}}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
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
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	stdout, stderr := executeRootWithSeparateOutputsForTest(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"deploy", "process-definition",
		"--file", bpmnPath,
		"--no-wait",
	)

	require.True(t, sawDeploy)
	require.NotContains(t, stderr, "creation target:")
	require.NotContains(t, stdout, "creation target:")
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	tenantContext := requireJSONObject(t, got["tenantContext"])
	require.Equal(t, "creation", tenantContext["mode"])
	require.Equal(t, "not_applicable", tenantContext["filter"])
	require.Equal(t, "tenant-a", tenantContext["targetTenantId"])
	require.Equal(t, []any{"tenant-a"}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(0), tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
	requireJSONItems(t, got["payload"], 1)
	require.Empty(t, tenantContext["warnings"])
}

func TestDeployProcessDefinitionCommand_QuietSuppressesCreationContext(t *testing.T) {
	resetDeployCommandStateForTest()
	var sawDeploy bool
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deploymentKey":"deployment-123","tenantId":"<default>","deployments":[{"processDefinition":{"processDefinitionId":"order-process","processDefinitionKey":"2251799813685255","processDefinitionVersion":3,"resourceName":"order-process.bpmn","tenantId":"<default>"}}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	bpmnPath := writeTempFile(t, "order-process.bpmn", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`))

	stdout, stderr := executeRootWithSeparateOutputsForTest(t,
		"--config", cfgPath,
		"--quiet",
		"deploy", "process-definition",
		"--file", bpmnPath,
		"--no-wait",
	)

	require.True(t, sawDeploy)
	require.Empty(t, stdout)
	require.NotContains(t, stderr, "creation target:")
	require.NotContains(t, stderr, "pd deploy done")
}

func TestDeployProcessDefinitionCommand_RunFallsBackToBPMNIDForV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"deploy", "process-definition",
		"--file", os.Getenv("C8VOLT_TEST_BPMN_PATH"),
		"--run",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeployProcessDefinitionCommand_TenantFlagOverridesEnvProfileAndConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--tenant", "flag-tenant",
		"deploy", "process-definition",
		"--file", os.Getenv("C8VOLT_TEST_BPMN_PATH"),
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeployProcessDefinitionCommand_V89NoWaitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--automation",
		"--json",
		"deploy", "process-definition",
		"--file", os.Getenv("C8VOLT_TEST_BPMN_PATH"),
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func resetDeployCommandStateForTest() {
	flagDeployPDFiles = nil
	flagDeployPDWithRun = false
	flagNoWait = false
	flagForce = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagQuiet = false
	resetDeployCommandContextForTest(Root())
}

func resetDeployCommandContextForTest(cmd *cobra.Command) {
	if cmd == nil {
		return
	}
	cmd.SetContext(context.Background())
	for _, child := range cmd.Commands() {
		resetDeployCommandContextForTest(child)
	}
}

// reportProcessDefinitionDeployCompletionEvent sends one facade-level
// deployment completion fact through the command adapter.
func reportProcessDefinitionDeployCompletionEvent(progress *processDefinitionDeploySemanticProgress, identity string, total int, disposition foptions.CompletionDisposition, detail string) {
	progress.FacadeProgress(foptions.ProgressEvent{
		Kind: foptions.ProgressEventKindCompletion,
		Completion: &foptions.CompletionProgress{
			Phase:         processDefinitionDeployCompletionPhase,
			CoreResource:  "process definition(s)",
			Total:         total,
			Identity:      identity,
			Disposition:   disposition,
			FailureDetail: detail,
		},
	})
}

// Helper-process entrypoint for all-tenants deployment rejection.
func TestDeployProcessDefinitionCommand_AllTenantsRejectsBeforeInputOrRequestHelper(t *testing.T) {
	executeRootHelperFromArgsEnv(t, "C8VOLT_TEST_ROOT_ARGS")
}

// Verifies deploy process-definition rejects multiple stdin markers in --file arguments.
func TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFile(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFileHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "only one '-' (stdin) allowed")
}

func writeTempFile(t *testing.T, name string, data []byte) string {
	t.Helper()
	path := t.TempDir() + string(os.PathSeparator) + name
	require.NoError(t, os.WriteFile(path, data, 0o600))
	return path
}

// validDeployProcessDefinitionBPMN returns stdin content that would deploy
// successfully if all-tenants rejection failed to run first.
func validDeployProcessDefinitionBPMN() string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<bpmn:definitions xmlns:bpmn="http://www.omg.org/spec/BPMN/20100524/MODEL">
  <bpmn:process id="order-process" isExecutable="true" />
</bpmn:definitions>`
}

// marshalRootArgsForEnv serializes helper-process command arguments without
// relying on shell quoting.
func marshalRootArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

// executeRootHelperFromArgsEnv runs the real Execute path for helper-process
// tests so invalid-input exit codes are asserted through process status.
func executeRootHelperFromArgsEnv(t *testing.T, envName string) {
	t.Helper()
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv(envName)), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}
	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = append([]string{"c8volt"}, args...)

	Execute()
}

// assertAllTenantsConcreteDestinationSubprocessFailure pins the shared error
// class and text used by all concrete-destination rejection tests.
func assertAllTenantsConcreteDestinationSubprocessFailure(t *testing.T, output []byte, err error, commandPath string) {
	t.Helper()
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "--all-tenants cannot be used with "+commandPath+"; this command requires a concrete destination tenant")
	require.NotContains(t, string(output), "Usage:")
	require.NotContains(t, string(output), "Examples:")
}

// Helper-process entrypoint for repeated-stdin-file validation.
func TestDeployProcessDefinitionCommand_RejectsRepeatedStdinFileHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "deploy", "process-definition", "--file", "-", "--file", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
