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

// TestOpsRepairProcessInstanceSearchTenantContextUsesDiscoverySemantics verifies
// process-instance repair search reports tenant filtering separately from
// resolved target evidence.
func TestOpsRepairProcessInstanceSearchTenantContextUsesDiscoverySemantics(t *testing.T) {
	cmd := &cobra.Command{}
	cfg := &config.Config{App: config.App{Tenant: "tenant-a"}}
	result := ops.RepairResult{
		Request: ops.RepairRequest{DiscoveryMode: ops.RepairDiscoveryModeSearch},
		FrozenSet: ops.RepairFrozenSet{
			TenantEvidence: process.TenantEvidence{
				ResolvedTenantIDs: []string{"tenant-a", "tenant-b"},
				Targets: []process.TenantEvidenceTarget{
					{Key: "2251799813685249", TenantID: "tenant-b"},
					{Key: "2251799813685250", TenantID: "tenant-a"},
				},
			},
		},
	}

	got := attachOpsRepairResultTenantContext(cmd, cfg, result)

	require.NotNil(t, got.Report.TenantContext)
	require.Equal(t, tenant.ContextModeDiscovery, got.Report.TenantContext.Mode)
	require.Equal(t, tenant.ContextFilterNamed, got.Report.TenantContext.Filter)
	require.True(t, got.Report.TenantContext.CrossTenant)
	require.Equal(t, tenant.ContextWarningMultipleTenants, got.Report.TenantContext.Warnings[0].Code)
	require.Equal(t, "tenant-a", got.Report.TenantID)
}

// TestOpsRepairProcessInstanceHelpDocumentsSelectionShape verifies the target-specific key and incident selector contract.
func TestOpsRepairProcessInstanceHelpDocumentsSelectionShape(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	output := executeRootForProcessInstanceTest(t, "ops", "repair", "process-instance", "--help")

	assertHelpOutputContainsAll(t, output,
		"Repair incidents selected by process instances",
		"Aliases:",
		"pi",
		"--key strings",
		"--direct-incidents-only",
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
		"./c8volt ops repair process-instance --key <process-instance-key> --dry-run",
		"./c8volt ops repair process-instance --direct-incidents-only --bpmn-process-id <bpmn-process-id> --limit 5 --dry-run",
	)
	require.NotContains(t, output, "--incidents-only")
}

// TestOpsRepairProcessInstanceDryRunWritesMarkdownReport verifies report writing keeps dry-run mutation-free.
func TestOpsRepairProcessInstanceDryRunWritesMarkdownReport(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	reportFile := filepath.Join(t.TempDir(), "repair.md")
	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--dry-run", "--report-file", reportFile}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "report: written "+reportFile)
	report := readReportFile(t, reportFile)
	require.Contains(t, report, "# Repair Process Instance Audit Report")
	require.Contains(t, report, "- Command: ops repair process-instance")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- Tenant: -")
	require.Contains(t, report, "- Tenant Context: selection scope: explicit resource keys; tenant filter not applied")
	require.Contains(t, report, "- Resource Tenant: <default>")
	require.Contains(t, report, "- Unknown Target Tenants: 0")
	require.Contains(t, report, "- Cross Tenant: false")
	require.Contains(t, report, "- Outcome: planned")
	require.Contains(t, report, "## Fixed Targets")
	require.Contains(t, report, "  - 2251799813685251")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "GET /v2/process-instances/2251799813685251")
	require.Contains(t, gotRequests, "POST /v2/process-instances/2251799813685251/incidents/search")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairProcessInstanceVarsFileDryRunShowsVariableScopes verifies file-backed variables are planned per process-instance scope.
func TestOpsRepairProcessInstanceVarsFileDryRunShowsVariableScopes(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	varsFile := t.TempDir() + "/repair-vars.json"
	require.NoError(t, os.WriteFile(varsFile, []byte(`{"customerTier":"gold"}`), 0o600))

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--vars-file", varsFile, "--dry-run", "--verbose"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair preview: 1 active incident(s) would be resolved; 1 related job(s), 1 variable scope(s) would be updated")
	require.Contains(t, string(output), "variable scope 2251799813685251: names=customerTier status=planned dependents=2251799813685249")
	require.Contains(t, string(output), "process-instance 2251799813685251 incident 2251799813685249: vars=planned")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "PUT /v2/element-instances/")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/resolution")
}

// TestOpsRepairProcessInstanceExplicitKeyNoWaitRepairsDiscoveredIncidents verifies keyed PI repair routes to incident repair.
func TestOpsRepairProcessInstanceExplicitKeyNoWaitRepairsDiscoveredIncidents(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--no-wait"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair process-instance incidents")
	require.Contains(t, string(output), "selected process instances: 1")
	require.Contains(t, string(output), "repairable process instances: 1")
	require.Contains(t, string(output), "active incidents: 1")
	require.Contains(t, string(output), "outcome: repaired")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "GET /v2/process-instances/2251799813685251")
	require.Contains(t, gotRequests, "POST /v2/process-instances/2251799813685251/incidents/search")
	require.Contains(t, gotRequests, "PATCH /v2/jobs/2251799813685252")
	require.Contains(t, gotRequests, "POST /v2/incidents/2251799813685249/resolution")
}

// TestOpsRepairProcessInstanceDirectKeysReportNonIncidentTargets verifies mixed direct PI keys stay non-fatal and visible.
func TestOpsRepairProcessInstanceDirectKeysReportNonIncidentTargets(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--key", "2251799813685255", "--dry-run", "--verbose"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "selected process instances: 2")
	require.Contains(t, string(output), "repairable process instances: 1")
	require.Contains(t, string(output), "active incidents: 1")
	require.Contains(t, string(output), "skipped process instances: 1 without active incidents")
	require.Contains(t, string(output), "skipped process-instance keys: 2251799813685255")
	require.Contains(t, string(output), "outcome: planned; no changes applied")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "GET /v2/process-instances/2251799813685255")
	require.Contains(t, gotRequests, "POST /v2/process-instances/2251799813685255/incidents/search")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairProcessInstanceDirectKeyWithoutIncidentNoOps verifies no-target preflight exits cleanly.
func TestOpsRepairProcessInstanceDirectKeyWithoutIncidentNoOps(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--key", "2251799813685255"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair process-instance incidents")
	require.Contains(t, string(output), "selected process instances: 1")
	require.Contains(t, string(output), "repairable process instances: 0")
	require.Contains(t, string(output), "active incidents: 0")
	require.Contains(t, string(output), "skipped process instances: 1 without active incidents")
	require.Contains(t, string(output), "repair plan: skipped")
	require.Contains(t, string(output), "outcome: planned; no changes applied")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "GET /v2/process-instances/2251799813685255")
	require.Contains(t, gotRequests, "POST /v2/process-instances/2251799813685255/incidents/search")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairProcessInstanceSearchPreflightsBeforeMutation verifies filtered PI repair plans and fixes keys before mutation.
func TestOpsRepairProcessInstanceSearchPreflightsBeforeMutation(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--state", "active", "--no-wait"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "repair process-instance incidents")
	requireRequestCount(t, requests.Snapshot(), "POST /v2/process-instances/search", 1)
	requireRequestBefore(t, requests.Snapshot(), "POST /v2/process-instances/search", "GET /v2/process-instances/2251799813685251")
	requireRequestBefore(t, requests.Snapshot(), "GET /v2/process-instances/2251799813685251", "PATCH /v2/jobs/2251799813685252")
}

// TestOpsRepairProcessInstanceAutoConfirmReportsTenantScopeBeforeWork verifies
// keyed and search paths report selection and affected evidence before the
// first variable update while preserving target discovery and prompt behavior.
func TestOpsRepairProcessInstanceAutoConfirmReportsTenantScopeBeforeWork(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		selection          string
		notSelection       string
		wantSearchRequests int
		wantProcessGets    int
	}{
		{
			name: "keyed tenant filter not applied",
			args: []string{
				"--tenant", "tenant-a",
				"ops", "repair", "process-instance",
				"--key", "2251799813685251",
				"--vars", `{"approved":true}`,
				"--auto-confirm",
				"--no-wait",
			},
			selection:       "selection scope: explicit resource keys; tenant filter not applied",
			notSelection:    "selection scope: tenant-a only",
			wantProcessGets: 1,
		},
		{
			name: "unfiltered search",
			args: []string{
				"ops", "repair", "process-instance",
				"--state", "active",
				"--vars", `{"approved":true}`,
				"--auto-confirm",
				"--no-wait",
			},
			selection:          "selection scope: unfiltered across accessible tenants",
			wantSearchRequests: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetOpsRepairProcessInstanceFlagState()
			t.Cleanup(resetOpsRepairProcessInstanceFlagState)

			var requests testx.SafeSlice[string]
			backend := newOpsRepairProcessInstanceServer(t, &requests)
			t.Cleanup(backend.Close)
			output := &opsTenantTimingOutput{}
			proxy, observations := newOpsTenantTimingProxy(t, backend.URL, output, func(r *http.Request) bool {
				return r.Method == http.MethodPut && strings.HasSuffix(r.URL.Path, "/variables")
			})
			t.Cleanup(proxy.Close)

			args := append([]string{"--config", writeTestConfigForVersion(t, proxy.URL, "8.9")}, tt.args...)
			promptCount, err := executeRootForOpsTenantTiming(t, output, resetOpsRepairProcessInstanceFlagState, args...)
			require.NoError(t, err, output.String())
			firstRequest, firstMutation := observations.snapshot()
			require.Contains(t, firstRequest, tt.selection)
			if tt.notSelection != "" {
				require.NotContains(t, firstRequest, tt.notSelection)
			}
			require.Contains(t, firstMutation, "affected tenants: <default>")
			require.Zero(t, promptCount)
			snapshot := requests.Snapshot()
			requireRequestCount(t, snapshot, "POST /v2/process-instances/search", tt.wantSearchRequests)
			requireRequestCount(t, snapshot, "GET /v2/process-instances/", tt.wantProcessGets)
			requireRequestCount(t, snapshot, "/incidents/search", 1)
			requireRequestCount(t, snapshot, "PUT /v2/element-instances/2251799813685251/variables", 1)
			requireRequestCount(t, snapshot, "/resolution", 1)
		})
	}
}

// TestOpsRepairProcessInstanceInteractiveTenantContext verifies keyed and
// search plans expose complete tenant context at confirmation, suppress
// repeated warning/context output, preserve repair counts, and never mutate
// on decline.
func TestOpsRepairProcessInstanceInteractiveTenantContext(t *testing.T) {
	for _, tt := range []struct {
		name               string
		args               []string
		selection          string
		warning            string
		decline            bool
		wantSearchRequests int
		wantProcessGets    int
		wantIncidentSearch int
		wantMutations      int
	}{
		{
			name: "keyed accepted",
			args: []string{
				"ops", "repair", "process-instance",
				"--key", "2251799813685251",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:          "selection scope: explicit resource keys; tenant filter not applied",
			wantProcessGets:    2,
			wantIncidentSearch: 2,
			wantMutations:      1,
		},
		{
			name: "keyed declined",
			args: []string{
				"ops", "repair", "process-instance",
				"--key", "2251799813685251",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:          "selection scope: explicit resource keys; tenant filter not applied",
			decline:            true,
			wantProcessGets:    1,
			wantIncidentSearch: 1,
		},
		{
			name: "search accepted",
			args: []string{
				"--tenant", "",
				"ops", "repair", "process-instance",
				"--state", "active",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:          "selection scope: unfiltered across accessible tenants",
			warning:            `--tenant "" overrides the configured tenant filter; selection is unfiltered`,
			wantSearchRequests: 1,
			wantProcessGets:    1,
			wantIncidentSearch: 2,
			wantMutations:      1,
		},
		{
			name: "search declined",
			args: []string{
				"--tenant", "",
				"ops", "repair", "process-instance",
				"--state", "active",
				"--vars", `{"approved":true}`,
				"--no-wait",
			},
			selection:          "selection scope: unfiltered across accessible tenants",
			warning:            `--tenant "" overrides the configured tenant filter; selection is unfiltered`,
			decline:            true,
			wantSearchRequests: 1,
			wantIncidentSearch: 1,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newOpsRepairProcessInstanceServer(t, &requests)
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
				"C8VOLT_TEST_CONFIG":                   cfgPath,
				"C8VOLT_TEST_OPS_REPAIR_PI_PROMPT":     promptPath,
				"C8VOLT_TEST_OPS_REPAIR_PI_PROMPT_OUT": promptOutputPath,
				"C8VOLT_TEST_OPS_REPAIR_PI_ARGS":       marshalOpsRepairProcessInstanceArgsForEnv(t, tt.args),
			}
			if tt.decline {
				env["C8VOLT_TEST_OPS_REPAIR_PI_DECLINE"] = "1"
			}

			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsRepairProcessInstanceCommandHelper", env)
			if tt.decline {
				require.Error(t, err, stderr)
			} else {
				require.NoError(t, err, stderr)
			}
			promptOutput := readReportFile(t, promptOutputPath)
			require.Contains(t, readReportFile(t, promptPath), "process-instance repair:")
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
			requireRequestCount(t, snapshot, "POST /v2/process-instances/search", tt.wantSearchRequests)
			requireRequestCount(t, snapshot, "GET /v2/process-instances/", tt.wantProcessGets)
			requireRequestCount(t, snapshot, "/incidents/search", tt.wantIncidentSearch)
			requireRequestCount(t, snapshot, "PATCH /v2/jobs/", tt.wantMutations)
			requireRequestCount(t, snapshot, "PUT /v2/element-instances/", tt.wantMutations)
			requireRequestCount(t, snapshot, "/resolution", tt.wantMutations)
		})
	}
}

// TestOpsRepairProcessInstanceBareDryRunSearchesIncidents verifies no explicit selector means the default incident-bearing search.
func TestOpsRepairProcessInstanceBareDryRunSearchesIncidents(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "--dry-run"}),
	})

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "dry run: repair process-instance incidents")
	require.Contains(t, string(output), "selection filters: {hasIncident=true}")
	require.Contains(t, string(output), "selected process instances: 1")
	require.Contains(t, string(output), "repairable process instances: 1")
	require.Contains(t, string(output), "active incidents: 1")
	require.Contains(t, string(output), "repair preview: 1 active incident(s) would be resolved; 1 related job(s), 0 variable scope(s) would be updated")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.Contains(t, gotRequests, "POST /v2/process-instances/search")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairProcessInstanceStdinDryRunUsesDiscoveredIncidents verifies stdin PI keys discover incidents without mutation in dry-run.
func TestOpsRepairProcessInstanceStdinDryRunUsesDiscoveredIncidents(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocessWithStdin(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, []string{"ops", "repair", "process-instance", "-", "--dry-run"}),
	}, "2251799813685253\n")

	require.NoError(t, err, string(output))
	require.Contains(t, string(output), "dry run: repair process-instance incidents")
	require.Contains(t, string(output), "selected process instances: 1")
	require.Contains(t, string(output), "repairable process instances: 1")
	require.Contains(t, string(output), "active incidents: 1")
	require.Contains(t, string(output), "repair preview: 1 active incident(s) would be resolved; 0 related job(s), 0 variable scope(s) would be updated")
	require.NotContains(t, string(output), "incidents without related jobs")
	gotRequests := strings.Join(requests.Snapshot(), "\n")
	require.NotContains(t, gotRequests, "PATCH /v2/jobs/")
	require.NotContains(t, gotRequests, "/resolution")
}

// TestOpsRepairProcessInstanceRejectsInvalidSelection verifies local validation catches ambiguous or unsafe inputs.
func TestOpsRepairProcessInstanceRejectsInvalidSelection(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid key",
			args: []string{"ops", "repair", "process-instance", "--key", "bad-key"},
			want: `process-instance key "bad-key" is not a valid key`,
		},
		{
			name: "keyed plus filter",
			args: []string{"ops", "repair", "process-instance", "--key", "2251799813685251", "--state", "active"},
			want: "--key cannot be combined with process-instance search filters",
		},
		{
			name: "invalid limit",
			args: []string{"ops", "repair", "process-instance", "--state", "active", "--limit", "0"},
			want: "--limit must be positive integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestOpsRepairProcessInstanceCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":             writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.9"),
				"C8VOLT_TEST_OPS_REPAIR_PI_ARGS": marshalOpsRepairProcessInstanceArgsForEnv(t, tt.args),
			})

			require.Error(t, err)
			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), tt.want)
			require.NotContains(t, string(output), "Usage:")
		})
	}
}

// TestOpsRepairProcessInstanceProgressContractPendingT068 defines
// process-instance search preflight, confirmation, incident lookup, repair
// counters, and stdout-safe progress.
func TestOpsRepairProcessInstanceProgressContractPendingT068(t *testing.T) {
	resetOpsRepairProcessInstanceFlagState()
	t.Cleanup(resetOpsRepairProcessInstanceFlagState)

	var requests testx.SafeSlice[string]
	srv := newOpsRepairProcessInstanceServer(t, &requests)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "process-instance-repair-progress.json")

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
		require.False(t, autoConfirm)
		require.Contains(t, prompt, "process-instance repair: 1 repairable process instance(s)")
		require.Contains(t, prompt, "1 active incident(s)")
		return nil
	}

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--verbose",
		"ops", "repair", "process-instance",
		"--state", "active",
		"--limit", "1",
		"--batch-size", "1",
		"--no-wait",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Contains(t, stderr, "process-instance repair scope: matched 1 process instance; page size: 1; discovery pages: 1")
	require.Contains(t, stderr, "discovering repair process instances, page 1/1, 1 seen")
	require.Contains(t, stderr, "loading process-instance repair incidents 1/1 process instance(s)")
	require.Contains(t, stderr, "planning process-instance repair scope 1/1 process instance(s)")
	require.Contains(t, stderr, "repairing incidents 1/1 incident(s)")
	require.NotContains(t, stderr, "/v2/")
	require.NotContains(t, stderr, "cursor")
	require.NotContains(t, stdout, "process-instance repair scope:")
	require.NotContains(t, stdout, "discovering repair process instances")
	require.NotContains(t, stdout, "planning process-instance repair scope")
	require.Contains(t, stderr, "report: written "+reportPath)
	require.Contains(t, stderr, "outcome: repaired")
}

// TestOpsRepairProcessInstanceQuietProgressShowsOnlyFailureWarning verifies
// process-instance repair keeps quiet mode free of success chatter while still
// exposing the immediate failure warning.
func TestOpsRepairProcessInstanceQuietProgressShowsOnlyFailureWarning(t *testing.T) {
	resetSemanticProgressModeFlags(t)
	flagQuiet = true
	now := time.Date(2026, 9, 1, 7, 4, 0, 0, time.UTC)
	opsRepairSemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { opsRepairSemanticProgressNow = time.Now })

	cmd, stderr := newSemanticProgressStderrCommand()
	request := ops.RepairRequest{}
	progress := configureOpsRepairProgress(cmd, &request)
	defer progress.Close()

	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportOpsRepairCompletionEvent(request.Progress, "incident-1", 2, ops.CompletionDispositionConfirmed, "")
	reportOpsRepairCompletionEvent(request.Progress, "incident-2", 2, ops.CompletionDispositionFailed, "retry exhausted")
	progress.Close()

	output := stderr.String()
	require.Contains(t, output, "incident-2 failed: retry exhausted (repairing incidents, 2/2 incident(s), 1 failed)")
	require.NotContains(t, output, "incident-1 repaired")
	require.NotContains(t, output, "\nrepairing incidents,")
}

// TestOpsRepairProcessInstanceSemanticProgressModeGate verifies
// process-instance-selected repair progress preserves protected output modes
// and still surfaces quiet failure warnings.
func TestOpsRepairProcessInstanceSemanticProgressModeGate(t *testing.T) {
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

// TestOpsRepairProcessInstanceMachineProgressSafetyPendingT068 pins
// process-instance repair progress silence for JSON, quiet, and automation
// modes.
func TestOpsRepairProcessInstanceMachineProgressSafetyPendingT068(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{name: "json", args: []string{"--json", "ops", "repair", "process-instance", "--state", "active", "--limit", "1", "--batch-size", "1", "--dry-run"}},
		{name: "quiet", args: []string{"--quiet", "ops", "repair", "process-instance", "--state", "active", "--limit", "1", "--batch-size", "1", "--dry-run"}},
		{name: "automation", args: []string{"--automation", "ops", "repair", "process-instance", "--state", "active", "--limit", "1", "--batch-size", "1", "--no-wait"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			resetOpsRepairProcessInstanceFlagState()
			t.Cleanup(resetOpsRepairProcessInstanceFlagState)

			var requests testx.SafeSlice[string]
			srv := newOpsRepairProcessInstanceServer(t, &requests)
			t.Cleanup(srv.Close)

			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, mode.args...)
			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
			for _, disallowed := range []string{
				"process-instance repair scope:",
				"discovering repair process instances",
				"loading process-instance repair incidents",
				"planning process-instance repair scope",
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

// TestOpsRepairProcessInstanceCommandHelper runs process-instance repair commands in a subprocess for exit-code assertions.
func TestOpsRepairProcessInstanceCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	cfgPath := os.Getenv("C8VOLT_TEST_CONFIG")
	args := unmarshalOpsRepairProcessInstanceArgsFromEnv(t)
	root := Root()
	resetCommandTreeFlags(root)
	resetOpsRepairProcessInstanceFlagState()
	var promptOutput bytes.Buffer
	errWriter := io.Writer(os.Stderr)
	if os.Getenv("C8VOLT_TEST_OPS_REPAIR_PI_PROMPT_OUT") != "" {
		errWriter = io.MultiWriter(os.Stderr, &promptOutput)
	}
	if promptPath := os.Getenv("C8VOLT_TEST_OPS_REPAIR_PI_PROMPT"); promptPath != "" {
		prevConfirm := confirmCmdOrAbortFn
		defer func() { confirmCmdOrAbortFn = prevConfirm }()
		confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
			if autoConfirm {
				return fmt.Errorf("unexpected auto-confirm prompt")
			}
			if err := os.WriteFile(promptPath, []byte(prompt), 0o600); err != nil {
				return err
			}
			if outputPath := os.Getenv("C8VOLT_TEST_OPS_REPAIR_PI_PROMPT_OUT"); outputPath != "" {
				if err := os.WriteFile(outputPath, promptOutput.Bytes(), 0o600); err != nil {
					return err
				}
			}
			if os.Getenv("C8VOLT_TEST_OPS_REPAIR_PI_DECLINE") == "1" {
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

// newOpsRepairProcessInstanceServer provides the minimal v2 endpoints used by process-instance repair tests.
func newOpsRepairProcessInstanceServer(t *testing.T, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()
	return testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813685251":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(opsRepairProcessInstanceJSON("2251799813685251")))
		case "/v2/process-instances/2251799813685253":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(opsRepairProcessInstanceJSON("2251799813685253")))
		case "/v2/process-instances/2251799813685255":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(opsRepairProcessInstanceJSONWithIncident("2251799813685255", false)))
		case "/v2/process-instances/2251799813685251/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[` + opsRepairIncidentJSON("2251799813685249", "2251799813685251", "2251799813685252", "ACTIVE") + `],"page":{"totalItems":1}}`))
		case "/v2/process-instances/2251799813685253/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[` + opsRepairIncidentJSON("2251799813685250", "2251799813685253", "", "ACTIVE") + `],"page":{"totalItems":1}}`))
		case "/v2/process-instances/2251799813685255/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0}}`))
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			payload, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Contains(t, string(payload), `"hasIncident":true`)
			_, _ = w.Write([]byte(`{"items":[` + opsRepairProcessInstanceJSON("2251799813685251") + `],"page":{"totalItems":1}}`))
		case "/v2/jobs/2251799813685252":
			require.Equal(t, http.MethodPatch, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/element-instances/2251799813685251/variables":
			require.Equal(t, http.MethodPut, r.Method)
			w.WriteHeader(http.StatusNoContent)
		case "/v2/incidents/2251799813685249/resolution":
			require.Equal(t, http.MethodPost, r.Method)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// opsRepairProcessInstanceJSON returns a compact process-instance API response for repair tests.
func opsRepairProcessInstanceJSON(key string) string {
	return opsRepairProcessInstanceJSONWithIncident(key, true)
}

// opsRepairProcessInstanceJSONWithIncident returns a compact process-instance API response with incident state.
func opsRepairProcessInstanceJSONWithIncident(key string, hasIncident bool) string {
	incidentValue := "false"
	if hasIncident {
		incidentValue = "true"
	}
	return `{"hasIncident":` + incidentValue + `,"processDefinitionId":"demo","processDefinitionKey":"2251799813685200","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":"` + key + `","rootProcessInstanceKey":"` + key + `","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"<default>"}`
}

// marshalOpsRepairProcessInstanceArgsForEnv serializes subprocess command arguments.
func marshalOpsRepairProcessInstanceArgsForEnv(t *testing.T, args []string) string {
	t.Helper()
	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

// unmarshalOpsRepairProcessInstanceArgsFromEnv deserializes subprocess command arguments.
func unmarshalOpsRepairProcessInstanceArgsFromEnv(t *testing.T) []string {
	t.Helper()
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_OPS_REPAIR_PI_ARGS")), &args))
	return args
}

// resetOpsRepairProcessInstanceFlagState restores shared command globals that process-instance repair uses.
func resetOpsRepairProcessInstanceFlagState() {
	flagOpsRepairProcessInstanceKeys = nil
	flagOpsRepairProcessInstanceRetries = 1
	flagOpsRepairProcessInstanceJobTimeoutRaw = ""
	flagOpsRepairProcessInstanceVars = ""
	flagOpsRepairProcessInstanceVarsFile = ""
	flagOpsRepairProcessInstanceReportFile = ""
	flagOpsRepairProcessInstanceReportFormat = ""
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
	flagGetPIState = "all"
	flagGetPIParentKey = ""
	flagGetPISize = consts.MaxPISearchSize
	flagGetPILimit = 0
	flagGetPIRootsOnly = false
	flagGetPIChildrenOnly = false
	flagGetPIOrphanChildrenOnly = false
	flagGetPIIncidentsOnly = false
	flagGetPIDirectIncidentsOnly = false
	flagGetPINoIncidentsOnly = false
	flagGetPIIncidentState = "active"
	flagGetPIIncidentErrorType = ""
	flagGetPIIncidentErrorMessage = ""
	flagGetPIIncidentMessageLimit = 0
	flagGetPIWithIncidents = false
	flagGetPIWithVars = false
	flagGetPIVarValueLimit = 0
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
