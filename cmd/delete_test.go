// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestDeleteCommands_RegressionPreservesCleanupContracts(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	piCapability := commandCapabilityForCommand(deleteProcessInstanceCmd)
	require.Equal(t, "delete process-instance", piCapability.Path)
	require.Equal(t, CommandMutationStateChanging, piCapability.Mutation)
	require.Equal(t, ContractSupportFull, piCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, piCapability.AutomationSupport)
	require.Contains(t, piCapability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process instance key(s) to delete; repeat or combine with stdin '-'",
	})
	require.Contains(t, piCapability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview delete scope without submitting deletion or cancel-before-delete requests",
	})
	require.Contains(t, piCapability.Flags, FlagContract{
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "force cancellation of the process instance(s), prior to deletion",
	})

	pdCapability := commandCapabilityForCommand(deleteProcessDefinitionCmd)
	require.Equal(t, "delete process-definition", pdCapability.Path)
	require.Equal(t, CommandMutationStateChanging, pdCapability.Mutation)
	require.Equal(t, ContractSupportFull, pdCapability.ContractSupport)
	require.Equal(t, AutomationSupportFull, pdCapability.AutomationSupport)
	require.Contains(t, pdCapability.Flags, FlagContract{
		Name:        "key",
		Shorthand:   "k",
		Type:        "stringSlice",
		Required:    false,
		Repeated:    true,
		Description: "process definition key(s) to delete",
	})
	require.Contains(t, pdCapability.Flags, FlagContract{
		Name:        "bpmn-process-id",
		Shorthand:   "b",
		Type:        "string",
		Required:    false,
		Repeated:    false,
		Description: "BPMN process ID of the process definition (all versions) to delete",
	})
	require.Contains(t, pdCapability.Flags, FlagContract{
		Name:        "dry-run",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "preview process-definition delete impact without submitting deletion or cancellation requests",
	})
	require.Contains(t, pdCapability.Flags, FlagContract{
		Name:        "force",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "force cancellation of the process instance(s), prior to deletion",
	})
}

// TestDeleteProcessDefinitionHelp_DocumentsTenantContract verifies destructive
// definition help separates selector discovery from explicit admin keys.
func TestDeleteProcessDefinitionHelp_DocumentsTenantContract(t *testing.T) {
	output := executeRootForTest(t, "delete", "process-definition", "--help")

	require.Contains(t, output, "Tenant contract:")
	require.Contains(t, output, "--tenant scopes BPMN selector discovery")
	require.Contains(t, output, "Explicit --key and stdin process-definition keys are backend-authorized admin input")
	require.Contains(t, output, "existing impact, confirmation, force, and wait safety checks still apply")
}

// TestDeleteCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig verifies command-local timeout precedence.
func TestDeleteCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "24s")

	cfg := resolveCommandConfigForTest(t, deleteCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "46s"))
	})

	require.Equal(t, 46*time.Second, cfg.App.Backoff.Timeout)
}

// TestDeleteHelp_DocumentsDestructiveConfirmationPaths verifies delete help explains confirmation paths.
func TestDeleteHelp_DocumentsDestructiveConfirmationPaths(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"delete"}, []string{
		"Delete process instances or process definitions",
		"--auto-confirm",
		"show verification examples",
		"./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --auto-confirm",
	}, nil)
	require.Contains(t, output, "process-instance")
	require.Contains(t, output, "process-definition")

	output = assertCommandHelpOutput(t, []string{"delete", "process-instance"}, []string{
		"validates the complete affected tree before submitting any delete request",
		"the whole delete batch is refused before mutation",
		"Use --force to cancel the affected scope first",
		"Use --auto-confirm for unattended destructive runs",
		"process instance key(s) to delete; repeat or combine with stdin '-'",
		"number of process instances to inspect per discovery page; does not cap total frozen scope",
		"maximum number of matching process instances to freeze for deletion across all pages; omit to continue through all matches",
		"./c8volt delete process-instance --state terminated --batch-size 250 --limit 5 --dry-run",
		"./c8volt delete process-instance --bpmn-process-id <bpmn-process-id> --state terminated --batch-size 250 --limit 5 --dry-run",
	}, []string{"--count"})
	require.Contains(t, output, "--force")
	require.Contains(t, output, "--batch-size int32")
	require.Contains(t, output, "--limit int32")

	output = assertCommandHelpOutput(t, []string{"delete", "process-definition"}, []string{
		"Delete process definition resources from Camunda",
		"checks delete impact without changing anything",
		"requires the full process-definition history deletion capability, currently Camunda 8.9 or newer",
		"associated history",
		"c8volt delete process-instance --bpmn-process-id <bpmn-process-id>",
		"Use --dry-run to preview process-definition delete impact without submitting deletion or cancellation requests",
		"./c8volt delete process-definition --key <process-definition-key> --dry-run",
		"./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --dry-run",
		"Use --auto-confirm for unattended destructive runs",
		"./c8volt delete process-definition --bpmn-process-id <bpmn-process-id> --latest --auto-confirm",
	}, nil)
	require.NotContains(t, output, "--allow-inconsistent")
}

// TestDeleteProcessDefinitionImpact_RenderForceImpact verifies the destructive
// prompt context includes cancellation impact before the user is asked to proceed.
func TestDeleteProcessDefinitionImpact_RenderForceImpact(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagForce = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	renderDeleteProcessDefinitionImpact(cmd, resource.DeleteProcessDefinitionPlan{
		Items: []resource.DeleteProcessDefinitionPlanItem{
			{
				Key:                        "pd-1",
				ActiveProcessInstanceCount: 2,
				CancellationPlan: process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-1"},
					Collected: typex.Keys{"root-1", "child-1"},
				},
			},
		},
	})

	output := buf.String()
	require.Contains(t, output, "deletion is irreversible")
	require.Contains(t, output, "delete impact check: 1 process definition(s); 2 active process instance(s) found; no changes made yet")
	require.Contains(t, output, "--force will cancel 1 root process instance(s), then delete 2 affected process instance(s), before deleting process definitions")
	require.NotContains(t, output, "WARNING:")
}

// TestDeleteProcessDefinitionImpact_RendersTenantWarningsBeforeImpact verifies
// compact destructive confirmation context keeps cross-tenant and unknown
// warnings prominent before process-definition impact details.
func TestDeleteProcessDefinitionImpact_RendersTenantWarningsBeforeImpact(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagForce = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	attachTenantContext(cmd, withTenantContextEvidence(newDiscoveryTenantContext(""), []string{"tenant-b", "tenant-a"}, 1))

	renderDeleteProcessDefinitionImpact(cmd, resource.DeleteProcessDefinitionPlan{
		Items: []resource.DeleteProcessDefinitionPlanItem{
			{
				Key:                        "pd-1",
				TenantId:                   "tenant-b",
				ActiveProcessInstanceCount: 2,
				CancellationPlan: process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"root-1"},
					Collected: typex.Keys{"root-1", "child-1"},
				},
			},
		},
	})

	output := buf.String()
	requireLineOrder(t, output,
		"selection scope: unfiltered across accessible tenants",
		"affected tenants: tenant-a, tenant-b",
		"WARNING: resources from multiple tenants will be affected: tenant-a, tenant-b",
		"WARNING: tenant metadata is unknown for 1 target",
		"delete impact check: 1 process definition(s); 2 active process instance(s) found; no changes made yet",
		"--force will cancel 1 root process instance(s), then delete 2 affected process instance(s), before deleting process definitions",
	)
}

// Verifies delete process-definition requires either --key or --bpmn-process-id as a target selector.
func TestDeleteProcessDefinitionCommand_RequiresTargetSelector(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionCommand_RequiresTargetSelectorHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "either --key, stdin keys, or --bpmn-process-id must be provided")
}

// TestDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector verifies stdin input counts as a delete target.
func TestDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector(t *testing.T) {
	var deleteBodies []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/batch-operations/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case "/v2/resources/2251799813692357/deletion":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			deleteBodies = append(deleteBodies, string(body))
			_, _ = w.Write([]byte(`{"resourceKey":"2251799813692357","batchOperation":{"batchOperationKey":"batch-1","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
		case "/v2/batch-operations/batch-1":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"batchOperationKey":"batch-1","batchOperationType":"DELETE_PROCESS_DEFINITION","state":"COMPLETED"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	}, "2251799813692357\n")
	require.NoError(t, err, string(output))
	require.NotContains(t, string(output), "either --key")
	require.Contains(t, string(output), "selection scope: explicit resource keys; tenant filter not applied")
	require.Contains(t, string(output), "WARNING: tenant metadata is unknown for 1 target")
	require.NotContains(t, string(output), "WARNING: resources from multiple tenants")
	require.Contains(t, string(output), "pd delete done; requested 1, ok 1, failed 0")
	body := decodeSingleRequestJSON(t, deleteBodies)
	require.Equal(t, true, body["deleteHistory"])
}

// TestDeleteProcessDefinitionCommand_RejectsUnsupportedFullHistoryVersionsBeforeMutation keeps
// unsupported process-definition history delete capability failures before any remote request.
func TestDeleteProcessDefinitionCommand_RejectsUnsupportedFullHistoryVersionsBeforeMutation(t *testing.T) {
	for _, version := range []toolx.CamundaVersion{toolx.V87, toolx.V88} {
		t.Run(version.String(), func(t *testing.T) {
			var called bool
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				http.Error(w, "unexpected request", http.StatusInternalServerError)
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, version.String())

			output, err := testx.RunCmdSubprocessWithStdin(t, "TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector", map[string]string{
				"C8VOLT_TEST_CONFIG": cfgPath,
			}, "2251799813692357\n")

			require.Error(t, err)
			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), "unsupported capability")
			require.Contains(t, string(output), "process-definition deletion requires the full process-definition history deletion capability, currently Camunda 8.9 or newer")
			require.Contains(t, string(output), "c8volt delete process-instance --bpmn-process-id")
			require.False(t, called)
		})
	}
}

// TestDeleteProcessDefinitionCommand_AcceptsFullHistoryCapabilityVersionsBeforeMutation
// proves V89 and V810 pass the local capability gate and submit the expected history delete.
func TestDeleteProcessDefinitionCommand_AcceptsFullHistoryCapabilityVersionsBeforeMutation(t *testing.T) {
	for _, version := range []toolx.CamundaVersion{toolx.V89, toolx.V810} {
		t.Run(version.String(), func(t *testing.T) {
			var deleteBodies []string
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/batch-operations/search":
					require.Equal(t, http.MethodPost, r.Method)
					_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				case "/v2/resources/2251799813692357/deletion":
					require.Equal(t, http.MethodPost, r.Method)
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					deleteBodies = append(deleteBodies, string(body))
					_, _ = w.Write([]byte(`{"resourceKey":"2251799813692357","batchOperation":{"batchOperationKey":"batch-1","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
				case "/v2/batch-operations/batch-1":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"batchOperationKey":"batch-1","batchOperationType":"DELETE_PROCESS_DEFINITION","state":"COMPLETED"}`))
				default:
					http.NotFound(w, r)
				}
			}))
			t.Cleanup(srv.Close)
			cfgPath := writeTestConfigForVersion(t, srv.URL, version.String())

			output, err := testx.RunCmdSubprocessWithStdin(t, "TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector", map[string]string{
				"C8VOLT_TEST_CONFIG": cfgPath,
			}, "2251799813692357\n")

			require.NoError(t, err, string(output))
			require.Contains(t, string(output), "pd delete done; requested 1, ok 1, failed 0")
			require.NotContains(t, string(output), `"outcome"`)
			require.Equal(t, []string{
				"POST /v2/batch-operations/search",
				"POST /v2/resources/2251799813692357/deletion",
				"GET /v2/batch-operations/batch-1",
				"GET /v2/process-definitions/2251799813692357",
			}, requests)
			body := decodeSingleRequestJSON(t, deleteBodies)
			require.Equal(t, true, body["deleteHistory"])
		})
	}
}

// TestDeleteProcessDefinitionCommand_BatchReadCheckBlocksBeforeMutation verifies
// missing batch read permission fails before any destructive resource deletion.
func TestDeleteProcessDefinitionCommand_BatchReadCheckBlocksBeforeMutation(t *testing.T) {
	var mu sync.Mutex
	deleteCalled := false
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/batch-operations/search":
			require.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"title":"FORBIDDEN","status":403,"detail":"Unauthorized to perform operation 'READ' on resource 'BATCH'"}`))
		case "/v2/resources/2251799813692357/deletion":
			mu.Lock()
			deleteCalled = true
			mu.Unlock()
			http.Error(w, "delete should not be called", http.StatusInternalServerError)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	}, "2251799813692357\n")

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "cannot confirm asynchronous history deletion because this identity cannot read Camunda batch operations")
	require.Contains(t, string(output), "READ")
	require.Contains(t, string(output), "BATCH")
	mu.Lock()
	defer mu.Unlock()
	require.False(t, deleteCalled)
}

// TestDeleteProcessDefinitionCommand_LatestSearchUsesEffectiveTenant verifies latest-definition lookup uses resolved tenant context.
func TestDeleteProcessDefinitionCommand_LatestSearchUsesEffectiveTenant(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.9
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionCommand_LatestSearchUsesEffectiveTenantHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[order-process]")

	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok, "expected search request filter object")
	require.Equal(t, "tenant-a", filter["tenantId"])
	require.Equal(t, true, filter["isLatestVersion"])
}

// TestDeleteProcessDefinitionCommand_KeyTenantMismatchUsesAdminScope verifies
// direct process-definition keys keep impact checks out of selected-tenant scope.
func TestDeleteProcessDefinitionCommand_KeyTenantMismatchUsesAdminScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	var statsRequests testx.SafeSlice[string]
	var requests testx.SafeSlice[string]
	srv := newTenantAdminProcessDefinitionDeleteServer(t, &requests, &statsRequests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	output := executeRootForTest(t,
		"--config", cfgPath,
		"--tenant", tenantAdminKeysSelectedTenant,
		"delete", "process-definition",
		"--key", tenantAdminKeysProcessDefinitionKey,
		"--auto-confirm",
		"--no-wait",
	)

	got := requests.Snapshot()
	require.Contains(t, got, "GET /v2/process-definitions/"+tenantAdminKeysProcessDefinitionKey)
	require.Contains(t, got, "POST /v2/resources/"+tenantAdminKeysProcessDefinitionKey+"/deletion")
	require.Equal(t, 0, countRequestPrefixes(got, "POST /v2/process-definitions/search"))
	require.NotEmpty(t, statsRequests.Snapshot())
	for _, request := range statsRequests.Snapshot() {
		filter := requireJSONObject(t, decodeSingleRequestJSON(t, []string{request})["filter"])
		require.NotContains(t, filter, "tenantId")
		require.Equal(t, tenantAdminKeysProcessDefinitionKey, stringFilterEqValue(t, filter["processDefinitionKey"]))
	}
	require.Contains(t, output, "selection scope: explicit resource keys; tenant filter not applied\n")
	require.Contains(t, output, "affected tenants: "+tenantAdminKeysReturnedTenant+"\n")
	require.NotContains(t, output, "selection scope: "+tenantAdminKeysSelectedTenant)
	require.Less(t,
		strings.Index(output, "selection scope: explicit resource keys; tenant filter not applied"),
		strings.Index(output, "delete impact check:"),
	)
	require.Contains(t, output, "tenant-b")
	require.Contains(t, output, "delete accepted")
}

// TestDeleteProcessDefinitionCommand_StdinTenantMismatchUsesAdminScope verifies
// stdin keys follow the same explicit admin-input path as direct --key values.
func TestDeleteProcessDefinitionCommand_StdinTenantMismatchUsesAdminScope(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	var statsRequests testx.SafeSlice[string]
	var requests testx.SafeSlice[string]
	srv := newTenantAdminProcessDefinitionDeleteServer(t, &requests, &statsRequests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	output, err := testx.RunCmdSubprocessWithStdin(t, "TestHelperDeleteProcessDefinitionCommand_StdinTenantMismatchUsesAdminScope", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	}, tenantAdminKeysProcessDefinitionKey+"\n")

	require.NoError(t, err, string(output))
	got := requests.Snapshot()
	require.Contains(t, got, "GET /v2/process-definitions/"+tenantAdminKeysProcessDefinitionKey)
	require.Contains(t, got, "POST /v2/resources/"+tenantAdminKeysProcessDefinitionKey+"/deletion")
	require.Equal(t, 0, countRequestPrefixes(got, "POST /v2/process-definitions/search"))
	require.NotEmpty(t, statsRequests.Snapshot())
	for _, request := range statsRequests.Snapshot() {
		filter := requireJSONObject(t, decodeSingleRequestJSON(t, []string{request})["filter"])
		require.NotContains(t, filter, "tenantId")
	}
	require.Contains(t, string(output), "tenant-b")
	require.Contains(t, string(output), "delete accepted")
}

func TestDeleteProcessDefinitionBpmnSelectorMissingFailsBeforeImpactPlanning(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			writeEmptyProcessDefinitionSearchResponse(w)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-definitions/"):
			t.Fatal("unexpected process-definition lookup before selector validation")
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			t.Fatal("unexpected delete impact planning before selector validation")
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/resources/"):
			t.Fatal("unexpected resource deletion before selector validation")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionBpmnSelectorMissingFailsBeforeImpactPlanningHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[missing-process]")
	require.NotContains(t, string(output), "no process definitions found to delete")
}

func TestDeleteProcessDefinitionBpmnSelectorVisiblePreservesPreviewAndDeletion(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"order-process","name":"Order Process","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/2251799813685255":
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(`{"processDefinitionKey":"2251799813685255","processDefinitionId":"order-process","name":"Order Process","version":3,"tenantId":"tenant","versionTag":"stable"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/resources/2251799813685255/deletion":
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(`{"resourceKey":"2251799813685255","batchOperation":{"batchOperationKey":"batch-2251799813685255","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionBpmnSelectorVisiblePreservesPreviewAndDeletionHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.NoError(t, err, string(output))
	require.Equal(t, []string{
		"POST /v2/process-definitions/search",
		"POST /v2/resources/2251799813685255/deletion",
	}, requests.Snapshot())
	require.Contains(t, string(output), "delete impact check: 1 process definition(s); process-instance state check skipped; no changes made yet")
	require.Contains(t, string(output), "pd 2251799813685255; delete accepted; batch batch-2251799813685255")
}

func TestDeleteProcessDefinitionDryRunStopsAfterImpactPreview(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"2251799813685255","processDefinitionId":"order-process","name":"Order Process","version":3,"tenantId":"tenant","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/batch-operations/search":
			t.Fatal("dry-run must not check batch-operation read access")
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/resources/"):
			t.Fatal("dry-run must not submit process-definition deletion")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionDryRunStopsAfterImpactPreviewHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.NoError(t, err, string(output))
	require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests.Snapshot())
	require.Contains(t, string(output), "dry run: delete process-definition")
	require.Contains(t, string(output), "delete impact check: 1 process definition(s); process-instance state check skipped; no changes made yet")
	require.NotContains(t, string(output), "Proceed with this deletion")
	require.NotContains(t, string(output), "delete accepted")
}

func TestDeleteProcessDefinitionCommand_RegressionPreservesSelectorPreflightForceAndNoWait(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			_, _ = w.Write([]byte(`{"items":[` + opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 3) + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/"+opsAllProcessDefinitionsPurgePDKeyA:
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 3)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/resources/"+opsAllProcessDefinitionsPurgePDKeyA+"/deletion":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			_, _ = w.Write([]byte(`{"resourceKey":"` + opsAllProcessDefinitionsPurgePDKeyA + `","batchOperation":{"batchOperationKey":"batch-` + opsAllProcessDefinitionsPurgePDKeyA + `","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")
	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionCommand_RegressionPreservesSelectorPreflightForceAndNoWaitHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "delete impact check: 1 process definition(s); no active process instances found; no changes made yet")
	require.Contains(t, string(output), "pd 2251799813685255 invoice v3/stable tenant; delete accepted; batch batch-2251799813685255")
	require.NotContains(t, string(output), "pd delete done; requested")

	got := requests.Snapshot()
	require.Equal(t, 1, countRequestPrefixes(got, "POST /v2/process-definitions/search "))
	require.Equal(t, 0, countRequestPrefixes(got, "POST /v2/batch-operations/search "))
	require.Equal(t, 0, countRequestPrefixes(got, "GET /v2/batch-operations/"))
	require.Equal(t, 1, countRequestPrefixes(got, "POST /v2/resources/"+opsAllProcessDefinitionsPurgePDKeyA+"/deletion "))

	searchBody := decodeSingleRequestJSON(t, requestBodyForPrefix(t, got, "POST /v2/process-definitions/search "))
	filter, ok := searchBody["filter"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "invoice", filter["processDefinitionId"])
	require.Equal(t, float64(3), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Equal(t, true, filter["isLatestVersion"])

	deleteBody := decodeSingleRequestJSON(t, requestBodyForPrefix(t, got, "POST /v2/resources/"+opsAllProcessDefinitionsPurgePDKeyA+"/deletion "))
	require.Equal(t, true, deleteBody["deleteHistory"])
}

func countRequestPrefixes(requests []string, prefix string) int {
	count := 0
	for _, request := range requests {
		if strings.HasPrefix(request, prefix) {
			count++
		}
	}
	return count
}

// requireLineOrder asserts that each expected fragment appears after the
// previous one in a combined command-output stream.
func requireLineOrder(t *testing.T, output string, expected ...string) {
	t.Helper()

	last := -1
	for _, want := range expected {
		idx := strings.Index(output, want)
		require.NotEqualf(t, -1, idx, "missing output fragment %q in:\n%s", want, output)
		require.Greaterf(t, idx, last, "output fragment %q appeared out of order in:\n%s", want, output)
		last = idx
	}
}

func requestBodyForPrefix(t *testing.T, requests []string, prefix string) []string {
	t.Helper()

	var matches []string
	for _, request := range requests {
		if strings.HasPrefix(request, prefix) {
			matches = append(matches, strings.TrimPrefix(request, prefix))
		}
	}
	return matches
}

func newTenantAdminProcessDefinitionDeleteServer(t *testing.T, requests *testx.SafeSlice[string], statsRequests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/"+tenantAdminKeysProcessDefinitionKey:
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(tenantAdminKeysMismatchProcessDefinitionJSON()))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			statsRequests.Append(string(body))
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/resources/"+tenantAdminKeysProcessDefinitionKey+"/deletion":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path)
			require.Equal(t, true, decodeSingleRequestJSON(t, []string{string(body)})["deleteHistory"])
			_, _ = w.Write([]byte(`{"resourceKey":"` + tenantAdminKeysProcessDefinitionKey + `","batchOperation":{"batchOperationKey":"batch-tenant-b","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// Runs a delete helper subprocess expected to fail and returns combined output with the exit code.
func executeDeleteProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
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

// Runs a delete helper subprocess and returns combined output with the underlying execution error.
func executeDeleteProcessInstanceSuccessHelper(t *testing.T, helperName string, cfgPath string) (string, error) {
	t.Helper()

	output, err := testx.RunCmdSubprocess(t, helperName, map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	})
	out := string(output)
	if err != nil {
		return out, err
	}
	return out, nil
}

// Helper-process entrypoint for delete process-definition target-selector validation.
func TestDeleteProcessDefinitionCommand_RequiresTargetSelectorHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-definition"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector is the helper-process entrypoint for stdin target validation.
func TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-definition", "--auto-confirm", "--no-state-check", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestHelperDeleteProcessDefinitionCommand_StdinTenantMismatchUsesAdminScope(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--tenant", tenantAdminKeysSelectedTenant,
		"delete", "process-definition",
		"--auto-confirm",
		"--no-wait",
		"-",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// TestDeleteProcessDefinitionCommand_LatestSearchUsesEffectiveTenantHelper is the helper-process entrypoint for tenant lookup validation.
func TestDeleteProcessDefinitionCommand_LatestSearchUsesEffectiveTenantHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant-a", "delete", "process-definition", "--bpmn-process-id", "order-process", "--latest", "--auto-confirm", "--no-wait"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeleteProcessDefinitionBpmnSelectorMissingFailsBeforeImpactPlanningHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"delete", "process-definition",
		"--bpmn-process-id", "missing-process",
		"--auto-confirm",
		"--no-state-check",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeleteProcessDefinitionBpmnSelectorVisiblePreservesPreviewAndDeletionHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"delete", "process-definition",
		"--bpmn-process-id", "order-process",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
		"--auto-confirm",
		"--no-state-check",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeleteProcessDefinitionDryRunStopsAfterImpactPreviewHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"delete", "process-definition",
		"--bpmn-process-id", "order-process",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
		"--dry-run",
		"--no-state-check",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeleteProcessDefinitionCommand_RegressionPreservesSelectorPreflightForceAndNoWaitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"delete", "process-definition",
		"--bpmn-process-id", "invoice",
		"--pd-version", "3",
		"--pd-version-tag", "stable",
		"--latest",
		"--force",
		"--no-wait",
		"--auto-confirm",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
