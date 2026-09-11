// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsRepairIncidentKeyTenantContextUsesExplicitSemantics verifies direct
// incident repair does not present the configured tenant as a local filter.
func TestOpsRepairIncidentKeyTenantContextUsesExplicitSemantics(t *testing.T) {
	cmd := &cobra.Command{}
	cfg := &config.Config{App: config.App{Tenant: "tenant-a"}}
	result := ops.RepairResult{
		Request: ops.RepairRequest{DiscoveryMode: ops.RepairDiscoveryModeKeyed},
		FrozenSet: ops.RepairFrozenSet{
			TenantEvidence: process.TenantEvidence{
				ResolvedTenantIDs: []string{"tenant-b"},
				Targets:           []process.TenantEvidenceTarget{{Key: "2251799813685249", TenantID: "tenant-b"}},
			},
		},
	}

	got := attachOpsRepairResultTenantContext(cmd, cfg, result)

	require.NotNil(t, got.Report.TenantContext)
	require.Equal(t, tenant.ContextModeExplicitKeys, got.Report.TenantContext.Mode)
	require.Equal(t, tenant.ContextFilterNotApplied, got.Report.TenantContext.Filter)
	require.Equal(t, []string{"tenant-b"}, got.Report.TenantContext.ResolvedTenantIDs)
	require.Empty(t, got.Report.TenantID)
}

func TestOpsRepairIncidentHelpDocumentsExplicitKeyShape(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	output := executeRootForProcessInstanceTest(t, "ops", "repair", "incident", "--help")

	assertHelpOutputContainsAll(t, output,
		"Repair incidents by key",
		"Aliases:",
		"inc",
		"--key strings",
		"--state string",
		"--error-type string",
		"--element-id string",
		"--element-instance-key string",
		"--batch-size int32",
		"--limit int32",
		"--retries int32",
		"--job-timeout string",
		"--vars string",
		"--vars-file string",
		"--report-file string",
		"--report-format string",
		"--dry-run",
		"--no-wait",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"./c8volt ops repair incident --key <incident-key> --dry-run",
		"./c8volt ops repair incident --key <incident-key> --vars '{\"hasIncident\":false}' --report-file repair-incident.md",
	)

	parentOutput := executeRootForProcessInstanceTest(t, "ops", "repair", "--help")
	require.NotContains(t, parentOutput, "--key strings")
	require.NotContains(t, parentOutput, "--retries int32")
	require.NotContains(t, output, "--flow-node-id")
	require.NotContains(t, output, "--fni-key")
}

func TestOpsRepairIncidentRejectsLegacyFlowNodeFilterFlags(t *testing.T) {
	tests := []string{"--flow-node-id", "--fni-key"}
	for _, flag := range tests {
		t.Run(flag, func(t *testing.T) {
			output, err := executeRootExpectErrorForTest(t, "ops", "repair", "incident", flag, "legacy-value")
			require.Error(t, err)
			if output == "" {
				output = err.Error()
			}
			require.Contains(t, output, "unknown flag: "+flag)
		})
	}
}

// TestOpsRepairIncidentDryRunWritesJSONReport verifies dry-run reports write structured audit data without mutation.
func TestOpsRepairIncidentDryRunWritesJSONReport(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	reportFile := filepath.Join(t.TempDir(), "repair.json")
	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"--json", "ops", "repair", "incident", "--state", "active", "--limit", "2", "--dry-run", "--report-file", reportFile, "--report-format", "json", "--automation"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), `"reportFile": "`+reportFile+`"`)
	require.Contains(t, string(output), `"reportFormat": "json"`)
	require.Contains(t, string(output), `"jobKeys": [`)
	require.Contains(t, string(output), `"retryUpdateStatus": "not_applicable"`)
	require.Contains(t, string(output), `"elementId": "task-a"`)
	require.Contains(t, string(output), `"elementInstanceKey": "2251799813685300"`)
	require.NotContains(t, string(output), "flowNode")
	var report map[string]any
	reportData := readReportFile(t, reportFile)
	require.Contains(t, reportData, `"elementId": "task-a"`)
	require.Contains(t, reportData, `"elementInstanceKey": "2251799813685300"`)
	require.NotContains(t, reportData, "flowNode")
	require.NoError(t, json.Unmarshal([]byte(reportData), &report))
	require.Equal(t, "ops.repair.v1", report["schemaVersion"])
	require.Equal(t, "ops repair incident", report["commandName"])
	require.Equal(t, "planned", report["outcome"])
	require.Equal(t, true, report["dryRun"])
	require.Equal(t, "8.9", report["camundaVersion"])
	require.NotContains(t, report, "tenantId")
	tenantContext := requireJSONObject(t, report["tenantContext"])
	require.Equal(t, "discovery", tenantContext["mode"])
	require.Equal(t, "none", tenantContext["filter"])
	require.Equal(t, []any{"<default>"}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(0), tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
	require.Equal(t, "unfiltered_selection", requireJSONObject(t, requireJSONItems(t, tenantContext["warnings"], 1)[0])["code"])
	require.Len(t, requireJSONObject(t, report["frozenSet"])["incidentKeys"], 2)
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "POST /v2/incidents/search")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairIncidentWorkerControlsReachJSONAndReport verifies CLI worker flags survive request construction.
func TestOpsRepairIncidentWorkerControlsReachJSONAndReport(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	reportFile := filepath.Join(t.TempDir(), "repair-controls.json")
	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{
			"--json",
			"ops", "repair", "incident",
			"--state", "active",
			"--limit", "2",
			"--workers", "3",
			"--fail-fast",
			"--no-worker-limit",
			"--dry-run",
			"--report-file", reportFile,
			"--report-format", "json",
			"--automation",
		}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), `"workers": 3`)
	require.Contains(t, string(output), `"failFast": true`)
	require.Contains(t, string(output), `"noWorkerLimit": true`)
	reportData := readReportFile(t, reportFile)
	require.Contains(t, reportData, `"failFast": true`)
	require.Contains(t, reportData, `"noWorkerLimit": true`)
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/search")
}

// TestOpsRepairIncidentWritesReportForFailureAfterDiscovery verifies post-discovery failures keep audit output.
func TestOpsRepairIncidentWritesReportForFailureAfterDiscovery(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	reportFile := filepath.Join(t.TempDir(), "repair-failed.json")
	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentFailingResolutionServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{
			"ops", "repair", "incident",
			"--key", "2251799813685249",
			"--no-wait",
			"--report-file", reportFile,
			"--report-format", "json",
		}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "ops repair incident:")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportFile)), &report))
	require.Equal(t, "failed", report["outcome"])
	require.NotContains(t, report, "tenantId")
	tenantContext := requireJSONObject(t, report["tenantContext"])
	require.Equal(t, "explicit_keys", tenantContext["mode"])
	require.Equal(t, "not_applied", tenantContext["filter"])
	require.Equal(t, []any{"<default>"}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(0), tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
	require.NotContains(t, tenantContext, "warnings")
	require.Len(t, report["errors"], 1)
	require.Len(t, requireJSONObject(t, report["frozenSet"])["incidentKeys"], 1)
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/2251799813685249/resolution")
}

// TestOpsRepairIncidentVarsDryRunShowsVariableScopes verifies repair variable flags reuse update-pi parsing and appear in dry-run output.
func TestOpsRepairIncidentVarsDryRunShowsVariableScopes(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--vars", `{"approved":true}`, "--dry-run", "--verbose"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair preview: 1 active incident(s) would be resolved; 1 related job(s), 1 variable scope(s) would be updated")
	require.Contains(t, string(output), "variable scope 2251799813685251: names=approved status=planned dependents=2251799813685249")
	require.Contains(t, string(output), "incident 2251799813685249: vars=planned")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PUT /v2/element-instances/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairIncidentRejectsInvalidVars verifies malformed repair variable JSON fails before remote mutation.
func TestOpsRepairIncidentRejectsInvalidVars(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--vars", "{"}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "--vars must be a valid JSON object")
}

func TestOpsRepairIncidentExplicitKeyNoWaitRepairsThroughServices(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--no-wait"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair incidents")
	require.Contains(t, string(output), "candidate incidents: 1")
	require.Contains(t, string(output), "outcome: repaired")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "GET /v2/incidents/2251799813685249")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/2251799813685252")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/2251799813685249/resolution")
}

func TestOpsRepairIncidentStdinDryRunUsesFixedKeys(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "-", "--dry-run"}),
	}, "2251799813685250\n")

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "dry run: repair incidents")
	require.Contains(t, string(output), "candidate incidents: 1")
	require.Contains(t, string(output), "repair preview: 1 active incident(s) would be resolved; 0 related job(s), 0 variable scope(s) would be updated")
	require.NotContains(t, string(output), "incidents without related jobs")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairIncidentFilterDryRunDiscoversFixedTargets verifies search-mode repair plans filtered incidents without mutation.
func TestOpsRepairIncidentFilterDryRunDiscoversFixedTargets(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--state", "active", "--error-type", "io_mapping_error", "--limit", "2", "--dry-run", "--verbose"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "dry run: repair incidents")
	require.Contains(t, string(output), `selection filters: {state=active, errorType="IO_MAPPING_ERROR"}`)
	require.Contains(t, string(output), "candidate incidents: 2")
	require.Contains(t, string(output), "incident keys: 2251799813685249, 2251799813685250")
	require.Contains(t, strings.Join(requests.Snapshot(), "\n"), "POST /v2/incidents/search")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "GET /v2/incidents/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PATCH /v2/jobs/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairIncidentSearchPreflightsBeforeMutation verifies search repair plans and fixes keys before mutation.
func TestOpsRepairIncidentSearchPreflightsBeforeMutation(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--state", "active", "--limit", "2", "--no-wait"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair incidents")
	requireRequestCount(t, requests.Snapshot(), "POST /v2/incidents/search", 1)
	requireRequestBefore(t, requests.Snapshot(), "POST /v2/incidents/search", "GET /v2/incidents/2251799813685249")
	requireRequestBefore(t, requests.Snapshot(), "GET /v2/incidents/2251799813685249", "POST /v2/incidents/2251799813685249/resolution")
}

// TestOpsRepairIncidentAutoConfirmReportsTenantScopeBeforeWork verifies keyed
// and search repairs publish selection and frozen evidence before variable
// updates without prompting or changing the selected backend targets.
func TestOpsRepairIncidentAutoConfirmReportsTenantScopeBeforeWork(t *testing.T) {
	tests := []struct {
		name                  string
		args                  []string
		selection             string
		notSelection          string
		wantSearchRequests    int
		wantIncidentGets      int
		wantVariableMutations int
		wantResolutions       int
	}{
		{
			name: "keyed tenant filter not applied",
			args: []string{
				"--tenant", "tenant-a",
				"ops", "repair", "incident",
				"--key", "2251799813685249",
				"--vars", `{"approved":true}`,
				"--auto-confirm",
				"--no-wait",
			},
			selection:             "selection scope: explicit resource keys; tenant filter not applied",
			notSelection:          "selection scope: tenant-a only",
			wantIncidentGets:      1,
			wantVariableMutations: 1,
			wantResolutions:       1,
		},
		{
			name: "named search",
			args: []string{
				"--tenant", "tenant-a",
				"ops", "repair", "incident",
				"--state", "active",
				"--limit", "2",
				"--vars", `{"approved":true}`,
				"--auto-confirm",
				"--no-wait",
			},
			selection:             "selection scope: tenant-a only",
			wantSearchRequests:    1,
			wantVariableMutations: 2,
			wantResolutions:       2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetOpsRepairIncidentFlagState()
			t.Cleanup(resetOpsRepairIncidentFlagState)

			var requests testx.SafeSlice[string]
			backend := newOpsRepairIncidentServer(t, &requests)
			t.Cleanup(backend.Close)
			output := &opsTenantTimingOutput{}
			proxy, observations := newOpsTenantTimingProxy(t, backend.URL, output, func(r *http.Request) bool {
				return r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/variables")
			})
			t.Cleanup(proxy.Close)

			args := append([]string{"--config", writeTestConfigForVersion(t, proxy.URL, "8.9")}, tt.args...)
			promptCount, err := executeRootForOpsTenantTiming(t, output, resetOpsRepairIncidentFlagState, args...)
			require.NoError(t, err, output.String())
			firstRequest, firstMutation := observations.snapshot()
			require.Contains(t, firstRequest, tt.selection)
			if tt.notSelection != "" {
				require.NotContains(t, firstRequest, tt.notSelection)
			}
			require.Contains(t, firstMutation, "affected tenants: <default>")
			require.Zero(t, promptCount)
			snapshot := requests.Snapshot()
			requireRequestCount(t, snapshot, "POST /v2/incidents/search", tt.wantSearchRequests)
			requireRequestCount(t, snapshot, "GET /v2/incidents/", tt.wantIncidentGets)
			requireRequestCount(t, snapshot, "PUT /v2/element-instances/", tt.wantVariableMutations)
			requireRequestCount(t, snapshot, "/resolution", tt.wantResolutions)
		})
	}
}

// TestOpsRepairIncidentInteractiveTenantContext verifies keyed and search
// plans expose complete tenant context at confirmation, suppress repeated
// warning/context output, preserve repair counts, and never mutate on decline.
func TestOpsRepairIncidentInteractiveTenantContext(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		args                  []string
		selection             string
		warning               string
		decline               bool
		wantSearchRequests    int
		wantIncidentGets      int
		wantVariableMutations int
		wantResolutions       int
	}{
		{
			name: "keyed accepted",
			args: []string{
				"ops", "repair", "incident",
				"--key", "2251799813685249",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:             "selection scope: explicit resource keys; tenant filter not applied",
			wantIncidentGets:      2,
			wantVariableMutations: 1,
			wantResolutions:       1,
		},
		{
			name: "keyed declined",
			args: []string{
				"ops", "repair", "incident",
				"--key", "2251799813685249",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:        "selection scope: explicit resource keys; tenant filter not applied",
			decline:          true,
			wantIncidentGets: 1,
		},
		{
			name: "search accepted",
			args: []string{
				"--tenant", "",
				"ops", "repair", "incident",
				"--state", "active",
				"--limit", "2",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:             "selection scope: unfiltered across accessible tenants",
			warning:               `--tenant "" overrides the configured tenant filter; selection is unfiltered`,
			wantSearchRequests:    1,
			wantIncidentGets:      2,
			wantVariableMutations: 2,
			wantResolutions:       2,
		},
		{
			name: "search declined",
			args: []string{
				"--tenant", "",
				"ops", "repair", "incident",
				"--state", "active",
				"--limit", "2",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:          "selection scope: unfiltered across accessible tenants",
			warning:            `--tenant "" overrides the configured tenant filter; selection is unfiltered`,
			decline:            true,
			wantSearchRequests: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newOpsRepairIncidentServer(t, &requests)
			t.Cleanup(srv.Close)
			promptPath := filepath.Join(t.TempDir(), "prompt.txt")
			promptOutputPath := filepath.Join(t.TempDir(), "prompt-output.txt")
			cfgPath := writeRawTestConfig(t, fmt.Sprintf(`app:
  camunda_version: "8.9"
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: %s
`, srv.URL))
			env := map[string]string{
				"C8VOLT_TEST_CONFIG":                    cfgPath,
				"C8VOLT_TEST_OPS_REPAIR_INC_PROMPT":     promptPath,
				"C8VOLT_TEST_OPS_REPAIR_INC_PROMPT_OUT": promptOutputPath,
				"C8VOLT_TEST_OPS_REPAIR_INC_ARGS":       marshalOpsRepairIncidentArgsForEnv(t, tt.args),
			}
			if tt.decline {
				env["C8VOLT_TEST_OPS_REPAIR_INC_DECLINE"] = "1"
			}

			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsRepairIncidentCommandHelper", env)
			if tt.decline {
				require.Error(t, err, stderr)
			} else {
				require.NoError(t, err, stderr)
			}
			promptOutput := readReportFile(t, promptOutputPath)
			require.Contains(t, readReportFile(t, promptPath), "incident repair:")
			require.Contains(t, promptOutput, tt.selection)
			require.Contains(t, promptOutput, "affected tenants: <default>")
			require.Less(t, strings.Index(promptOutput, tt.selection), strings.Index(promptOutput, "affected tenants: <default>"))
			combined := stdout + stderr
			require.Equal(t, 1, strings.Count(combined, tt.selection), combined)
			require.Equal(t, 1, strings.Count(combined, "affected tenants: <default>"), combined)
			if tt.warning != "" {
				require.Contains(t, promptOutput, tt.warning)
				require.Less(t, strings.Index(promptOutput, tt.warning), strings.Index(promptOutput, tt.selection))
				require.Equal(t, 1, strings.Count(combined, tt.warning), combined)
			}
			snapshot := requests.Snapshot()
			requireRequestCount(t, snapshot, "POST /v2/incidents/search", tt.wantSearchRequests)
			requireRequestCount(t, snapshot, "GET /v2/incidents/", tt.wantIncidentGets)
			requireRequestCount(t, snapshot, "PUT /v2/element-instances/", tt.wantVariableMutations)
			requireRequestCount(t, snapshot, "/resolution", tt.wantResolutions)
		})
	}
}

// TestOpsRepairIncidentRejectsKeyedSearchMode verifies mixed key and filter selection fails before remote mutation.
func TestOpsRepairIncidentRejectsKeyedSearchMode(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "2251799813685249", "--state", "active"}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "--key cannot be combined with search filters")
	require.NotContains(t, string(output), "Usage:")
}

// TestOpsRepairIncidentSearchValidation verifies local batch and limit guardrails are enforced before CLI initialization.
func TestOpsRepairIncidentSearchValidation(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid batch size",
			args: []string{"ops", "repair", "incident", "--batch-size", "0"},
			want: "invalid value for --batch-size",
		},
		{
			name: "invalid limit",
			args: []string{"ops", "repair", "incident", "--limit", "0"},
			want: "--limit must be positive integer",
		},
		{
			name: "report format requires report file",
			args: []string{"ops", "repair", "incident", "--report-format", "json", "--dry-run"},
			want: "--report-format requires --report-file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
				"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, tt.args),
			})

			require.Error(t, err)
			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), tt.want)
		})
	}
}

func TestOpsRepairIncidentInvalidKeyFailsBeforeMutation(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairIncidentCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_INC_ARGS": marshalOpsRepairIncidentArgsForEnv(t, []string{"ops", "repair", "incident", "--key", "bad-key"}),
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `incident key "bad-key" is not a valid key`)
	require.NotContains(t, string(output), "Usage:")
}

// TestOpsRepairIncidentProgressContractPendingT068 defines incident-search
// preflight, frozen confirmation, repair counters, and stdout-safe progress.
func TestOpsRepairIncidentProgressContractPendingT068(t *testing.T) {
	resetOpsRepairIncidentFlagState()
	t.Cleanup(resetOpsRepairIncidentFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairIncidentServer(t, &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "incident-repair-progress.json")

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.False(t, autoConfirm)
		require.Contains(t, prompt, "incident repair: 2 active incident(s)")
		require.Contains(t, prompt, "1 related job(s)")
		return nil
	}

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--verbose",
		"ops", "repair", "incident",
		"--state", "active",
		"--limit", "2",
		"--batch-size", "2",
		"--no-wait",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Contains(t, stderr, "incident repair scope: matched 2 incidents; page size: 2; discovery pages: 1")
	require.Contains(t, stderr, "discovering repair incidents, page 1/1, 2 seen")
	require.Contains(t, stderr, "planning incident repair scope 2/2 incident(s)")
	require.Contains(t, stderr, "repairing incidents 2/2 incident(s)")
	require.NotContains(t, stderr, "/v2/")
	require.NotContains(t, stderr, "cursor")
	require.NotContains(t, stdout, "incident repair scope:")
	require.NotContains(t, stdout, "discovering repair incidents")
	require.NotContains(t, stdout, "planning incident repair scope")
	require.Contains(t, stderr, "report: written "+reportPath)
	require.Contains(t, stderr, "outcome: repaired")

	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "repaired", report["outcome"])
	require.Len(t, requireJSONObject(t, report["frozenSet"])["incidentKeys"], 2)
}

// TestOpsRepairIncidentDefaultFailureWarnsAndFlushes verifies incident repair
// emits immediate failure evidence and one final aggregate after later progress.
func TestOpsRepairIncidentDefaultFailureWarnsAndFlushes(t *testing.T) {
	resetSemanticProgressModeFlags(t)
	now := time.Date(2026, 9, 1, 7, 3, 0, 0, time.UTC)
	opsRepairSemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { opsRepairSemanticProgressNow = time.Now })

	cmd, stderr := newSemanticProgressStderrCommand()
	request := ops.RepairRequest{}
	progress := configureOpsRepairProgress(cmd, &request)
	defer progress.Close()

	reportOpsRepairCompletionEvent(request.Progress, "incident-1", 2, ops.CompletionDispositionFailed, "job activation timed out")
	reportOpsRepairCompletionEvent(request.Progress, "incident-2", 2, ops.CompletionDispositionConfirmed, "")
	progress.Close()
	progress.Close()

	output := stderr.String()
	require.Equal(t, 1, strings.Count(output, "incident-1 failed: job activation timed out (repairing incidents, 1/2 incident(s), 1 failed)"))
	require.Equal(t, 1, strings.Count(output, "repairing incidents, 2/2 incident(s), 1 failed"))
}

// TestOpsRepairIncidentVerboseLifecycleVocabulary verifies repair progress
// maps submitted, repaired, and failed wording in the command layer.
func TestOpsRepairIncidentVerboseLifecycleVocabulary(t *testing.T) {
	resetSemanticProgressModeFlags(t)
	flagVerbose = true

	cmd, stderr := newSemanticProgressStderrCommand()
	request := ops.RepairRequest{}
	progress := configureOpsRepairProgress(cmd, &request)
	defer progress.Close()

	reportOpsRepairCompletionEvent(request.Progress, "incident-1", 3, ops.CompletionDispositionSubmitted, "")
	reportOpsRepairCompletionEvent(request.Progress, "incident-2", 3, ops.CompletionDispositionConfirmed, "")
	reportOpsRepairCompletionEvent(request.Progress, "incident-3", 3, ops.CompletionDispositionFailed, "retry exhausted")
	progress.Close()

	output := stderr.String()
	require.Contains(t, output, "incident-1 submitted (repairing incidents, 1/3 incident(s))")
	require.Contains(t, output, "incident-2 repaired (repairing incidents, 2/3 incident(s))")
	require.Contains(t, output, "incident-3 failed: retry exhausted (repairing incidents, 3/3 incident(s), 1 failed)")
}

// TestOpsRepairIncidentSemanticProgressModeGate verifies incident repair
// progress is stdout-safe in machine modes and quiet reports only failures.
func TestOpsRepairIncidentSemanticProgressModeGate(t *testing.T) {
	assertOpsCompletionProgressModeGate(t, opsCompletionProgressModeGateCase{
		Configure: func(cmd *cobra.Command) (func(ops.ProgressEvent), func()) {
			request := ops.RepairRequest{}
			progress := configureOpsRepairProgress(cmd, &request)
			return request.Progress, progress.Close
		},
		Event: func(disposition ops.CompletionDisposition, detail string) ops.ProgressEvent {
			return ops.ProgressEvent{
				Kind: ops.ProgressEventKindCompletion,
				Completion: &ops.CompletionProgress{
					Phase:         opsRepairCompletionPhase,
					CoreResource:  "incident(s)",
					Total:         1,
					Identity:      "incident-1",
					Disposition:   disposition,
					FailureDetail: detail,
				},
			}
		},
		QuietWarning: "incident-1 failed: request rejected (repairing incidents, 1/1 incident(s), 1 failed)",
	})
}

// reportOpsRepairCompletionEvent sends one repair completion fact through the
// configured repair command progress callback.
func reportOpsRepairCompletionEvent(progress func(ops.ProgressEvent), identity string, total int, disposition ops.CompletionDisposition, detail string) {
	progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:         opsRepairCompletionPhase,
			CoreResource:  "incident(s)",
			Total:         total,
			Identity:      identity,
			Disposition:   disposition,
			FailureDetail: detail,
		},
	})
}

// TestOpsRepairIncidentMachineProgressSafetyPendingT068 pins repair progress
// silence for JSON, quiet, and automation modes.
func TestOpsRepairIncidentMachineProgressSafetyPendingT068(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{name: "json", args: []string{"--json", "ops", "repair", "incident", "--state", "active", "--limit", "2", "--batch-size", "1", "--dry-run"}},
		{name: "quiet", args: []string{"--quiet", "ops", "repair", "incident", "--state", "active", "--limit", "2", "--batch-size", "1", "--dry-run"}},
		{name: "automation", args: []string{"--automation", "ops", "repair", "incident", "--state", "active", "--limit", "2", "--batch-size", "1", "--no-wait"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			resetOpsRepairIncidentFlagState()
			t.Cleanup(resetOpsRepairIncidentFlagState)

			var requests testx.SafeSlice[string]
			srv := newOpsRepairIncidentServer(t, &requests)
			t.Cleanup(srv.Close)

			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, mode.args...)
			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
			for _, disallowed := range []string{
				"incident repair scope:",
				"discovering repair incidents",
				"planning incident repair scope",
				"repairing incidents",
			} {
				require.NotContains(t, stdout, disallowed)
				require.NotContains(t, stderr, disallowed)
			}
			if mode.name == "json" {
				var envelope map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
			}
		})
	}
}

func TestOpsRepairIncidentCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cfgPath := os.Getenv("C8VOLT_TEST_CONFIG")
	args := unmarshalOpsRepairIncidentArgsFromEnv(t)
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsRepairIncidentFlagState()
	var promptOutput bytes.Buffer
	errWriter := io.Writer(os.Stderr)
	if os.Getenv("C8VOLT_TEST_OPS_REPAIR_INC_PROMPT_OUT") != "" {
		errWriter = io.MultiWriter(os.Stderr, &promptOutput)
	}
	if promptPath := os.Getenv("C8VOLT_TEST_OPS_REPAIR_INC_PROMPT"); promptPath != "" {
		prevConfirm := confirmCmdOrAbortFn
		defer func() { confirmCmdOrAbortFn = prevConfirm }()
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			if autoConfirm {
				return fmt.Errorf("unexpected auto-confirm prompt")
			}
			if err := os.WriteFile(promptPath, []byte(prompt), 0o600); err != nil {
				return err
			}
			if outputPath := os.Getenv("C8VOLT_TEST_OPS_REPAIR_INC_PROMPT_OUT"); outputPath != "" {
				if err := os.WriteFile(outputPath, promptOutput.Bytes(), 0o600); err != nil {
					return err
				}
			}
			if os.Getenv("C8VOLT_TEST_OPS_REPAIR_INC_DECLINE") == "1" {
				return fmt.Errorf("confirmation declined")
			}
			return nil
		}
	}
	root.SetArgs(append([]string{"--config", cfgPath}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(errWriter)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

func newOpsRepairIncidentServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[` + opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE") + `,` + opsRepairIncidentJSON("2251799813685250", "2251799813685253", "", "ACTIVE") + `],"page":{"totalItems":2}}`))
		case "/v2/incidents/2251799813685249":
			_, _ = w.Write([]byte(opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE")))
		case "/v2/incidents/2251799813685250":
			_, _ = w.Write([]byte(opsRepairIncidentJSON("2251799813685250", "2251799813685253", "", "ACTIVE")))
		case "/v2/jobs/2251799813685252":
			require.Equal(t, http.MethodPatch, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/element-instances/2251799813685251/variables", "/v2/element-instances/2251799813685253/variables":
			require.Equal(t, http.MethodPut, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249/resolution", "/v2/incidents/2251799813685250/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsRepairIncidentFailingResolutionServer returns discovery data but fails after mutation begins.
func newOpsRepairIncidentFailingResolutionServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/incidents/2251799813685249":
			_, _ = w.Write([]byte(opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE")))
		case "/v2/jobs/2251799813685252":
			require.Equal(t, http.MethodPatch, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			http.Error(w, "resolution failed", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func opsRepairIncidentJSON(incidentKey string, processInstanceKey string, jobKey string, state string) string {
	job := ""
	if jobKey != "" {
		job = `,"jobKey":"` + jobKey + `"`
	}
	return `{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"2251799813685300","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"` + incidentKey + `","processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processInstanceKey":"` + processInstanceKey + `","rootProcessInstanceKey":"` + processInstanceKey + `","state":"` + state + `","tenantId":"<default>"` + job + `}`
}

func marshalOpsRepairIncidentArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

func unmarshalOpsRepairIncidentArgsFromEnv(t *testing.T) []string {
	t.Helper()
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_OPS_REPAIR_INC_ARGS")), &args))
	return args
}

func resetOpsRepairIncidentFlagState() {
	flagOpsRepairIncidentKeys = nil
	flagOpsRepairIncidentState = "active"
	flagOpsRepairIncidentErrorType = ""
	flagOpsRepairIncidentErrorMessage = ""
	flagOpsRepairIncidentPIKey = ""
	flagOpsRepairIncidentRootKey = ""
	flagOpsRepairIncidentPDKey = ""
	flagOpsRepairIncidentBpmnProcessID = ""
	flagOpsRepairIncidentElementID = ""
	flagOpsRepairIncidentElementInstanceKey = ""
	flagOpsRepairIncidentCreationTimeAfter = ""
	flagOpsRepairIncidentCreationTimeBefore = ""
	flagOpsRepairIncidentCreationTimeNewer = -1
	flagOpsRepairIncidentCreationTimeOlder = -1
	flagOpsRepairIncidentBatchSize = consts.MaxPISearchSize
	flagOpsRepairIncidentLimit = 0
	flagOpsRepairIncidentRetries = 1
	flagOpsRepairIncidentJobTimeoutRaw = ""
	flagOpsRepairIncidentVars = ""
	flagOpsRepairIncidentVarsFile = ""
	flagOpsRepairIncidentReportFile = ""
	flagOpsRepairIncidentReportFormat = ""
	flagDryRun = false
	flagNoWait = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagVerbose = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagCmdAutoConfirm = false
	flagCmdAutomation = false
}

// requireRequestBefore asserts that one captured request happened before another.
func requireRequestBefore(t *testing.T, requests []string, before string, after string) {
	t.Helper()
	beforeIndex := -1
	afterIndex := -1
	for i, request := range requests {
		if beforeIndex < 0 && strings.Contains(request, before) {
			beforeIndex = i
		}
		if afterIndex < 0 && strings.Contains(request, after) {
			afterIndex = i
		}
	}
	require.NotEqual(t, -1, beforeIndex, "missing request containing %q in %v", before, requests)
	require.NotEqual(t, -1, afterIndex, "missing request containing %q in %v", after, requests)
	require.Less(t, beforeIndex, afterIndex, "%q should happen before %q", before, after)
}

// requireRequestCount verifies preflight discovery is not repeated after confirmation.
func requireRequestCount(t *testing.T, requests []string, contains string, want int) {
	t.Helper()
	got := 0
	for _, request := range requests {
		if strings.Contains(request, contains) {
			got++
		}
	}
	require.Equal(t, want, got, "request count for %q in %v", contains, requests)
}
