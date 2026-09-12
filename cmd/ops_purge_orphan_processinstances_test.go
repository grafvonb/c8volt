// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
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
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsPurgeOrphanProcessInstancesUnfilteredTenantContext verifies orphan
// purge keeps empty discovery unfiltered and never reports it as default tenant.
func TestOpsPurgeOrphanProcessInstancesUnfilteredTenantContext(t *testing.T) {
	cmd := &cobra.Command{}
	result := ops.OrphanPurgeResult{
		DeletionPlan: ops.DeletionPlan{
			TenantEvidence: process.TenantEvidence{
				ResolvedTenantIDs: []string{"tenant-b"},
				Targets:           []process.TenantEvidenceTarget{{Key: opsOrphanProcessKey, TenantID: "tenant-b"}},
			},
		},
	}

	got := attachOpsPurgeOrphanProcessInstancesResultTenantContext(cmd, &config.Config{}, result)

	require.NotNil(t, got.Report.TenantContext)
	require.Equal(t, tenant.ContextModeDiscovery, got.Report.TenantContext.Mode)
	require.Equal(t, tenant.ContextFilterNone, got.Report.TenantContext.Filter)
	require.Empty(t, got.Report.TenantContext.ConfiguredTenantID)
	require.Equal(t, []string{"tenant-b"}, got.Report.TenantContext.ResolvedTenantIDs)
	require.Equal(t, tenant.ContextWarningUnfilteredSelection, got.Report.TenantContext.Warnings[0].Code)
}

const (
	opsOrphanChildKey   = "2251799813685250"
	opsOrphanParentKey  = "2251799813685249"
	opsOrphanProcessKey = "2251799813685248"
)

func TestOpsPurgeOrphanProcessInstancesHelpDocumentsSafeAutomationPreview(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "ops", "purge", "orphan-process-instances", "--help")

	assertHelpOutputContainsAll(t, output,
		"Delete orphan child process instances whose parents are missing",
		"./c8volt ops purge orphan-process-instances --dry-run",
		"./c8volt ops purge orphan-process-instances --state completed --limit 25 --report-file orphan-purge.md",
	)
	assertHelpOutputOmitsAll(t, output,
		"./c8volt ops purge orphan-process-instances --automation --json\n",
	)
}

func TestOpsPurgeOrphanProcessInstancesActivityWrapsDryRunDiscovery(t *testing.T) {
	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	_, err := purgeOrphanProcessInstancesWithCommandActivity(cmd, ops.OrphanPurgeRequest{DryRun: true, BatchSize: 25}, func() (ops.OrphanPurgeResult, error) {
		return ops.OrphanPurgeResult{}, nil
	})

	require.NoError(t, err)
	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"discovering orphan process-instance candidates"}, msgs)
}

func TestOpsPurgeOrphanProcessInstancesDryRunHidesCandidateKeysWithoutDelete(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, true)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--state", "active",
	)

	require.Contains(t, output, "dry run: purge orphan process-instances")
	require.Contains(t, output, "candidate orphan process instances: 1")
	require.NotContains(t, output, "candidate keys:")
	require.Contains(t, output, "delete preview: 1 orphan candidate(s), 1 affected process instance(s) across 1 root(s) would be deleted")
	require.NotContains(t, output, "dependency expansion:")
	require.NotContains(t, output, "one or more parent process instances were not found")
	require.NotContains(t, output, "no deletion request submitted")
	require.Contains(t, output, "outcome: planned; no changes applied; use --verbose to list process-instance keys")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

func TestOpsPurgeOrphanProcessInstancesDryRunVerboseReportsCandidateKeys(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, true)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"--verbose",
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--state", "active",
	)

	require.Contains(t, output, "candidate orphan process instances: 1")
	require.Contains(t, output, "candidate keys: "+opsOrphanChildKey)
	require.NotContains(t, output, "one or more parent process instances were not found")
	require.NotContains(t, output, "use --verbose to list process-instance keys")
	require.NotContains(t, strings.Join(requests.Snapshot(), "\n"), "/deletion")
}

// TestOpsPurgeOrphanProcessInstancesDryRunNoTargetsReportsNoOp verifies an
// empty validated scope remains explicit in audit data without invented tenants.
func TestOpsPurgeOrphanProcessInstancesDryRunNoTargetsReportsNoOp(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, false)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge-empty.json")

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Contains(t, output, "candidate orphan process instances: 0")
	require.Contains(t, output, "delete preview: skipped (no orphan process-instance targets)")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.NotContains(t, output, "use --verbose to list process-instance keys")
	snapshot := requests.Snapshot()
	require.Len(t, snapshot, 1)
	require.True(t, strings.HasPrefix(snapshot[0], "POST /v2/process-instances/search "))
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "planned", report["outcome"])
	tenantContext := requireJSONObject(t, report["tenantContext"])
	require.Equal(t, "discovery", tenantContext["mode"])
	require.Equal(t, "none", tenantContext["filter"])
	require.Equal(t, []any{}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(0), tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
	require.Equal(t, "unfiltered_selection", requireJSONObject(t, requireJSONItems(t, tenantContext["warnings"], 1)[0])["code"])
}

func TestOpsPurgeOrphanProcessInstancesDryRunAppliesCompatibleFilters(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, false)
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--bpmn-process-id", "order-process",
		"--state", "active",
		"--batch-size", "25",
		"--limit", "1",
	)

	require.Contains(t, output, "candidate orphan process instances: 0")
	request := decodeCapturedPISearchRequest(t, strings.TrimPrefix(requests.Snapshot()[0], "POST /v2/process-instances/search "))
	filter := request["filter"].(map[string]any)
	page := request["page"].(map[string]any)
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, "ACTIVE", filter["state"])
	require.Equal(t, float64(25), page["limit"])
	require.Contains(t, filter, "parentProcessInstanceKey")
}

func TestOpsPurgeOrphanProcessInstancesAutoConfirmDeletesCandidateKeys(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
	)

	require.Contains(t, output, "purge orphan process-instances")
	require.Contains(t, output, "candidate orphan process instances: 1")
	require.Contains(t, output, "delete plan: planned")
	require.NotContains(t, output, "one or more parent process instances were not found")
	require.Contains(t, output, "deletion: submitted 1 process-instance tree (--no-wait)")
	require.NotContains(t, output, "deletion confirmation:")
	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "elapsed:")
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
	require.NotContains(t, strings.Join(deleted.Snapshot(), "\n"), opsOrphanParentKey)
}

// TestOpsPurgeOrphanProcessInstancesInteractiveTenantContext verifies complete
// tenant context is visible at acceptance or decline, duplicate scope events
// stay silent, and the planned orphan key is reused for confirmed deletion.
func TestOpsPurgeOrphanProcessInstancesInteractiveTenantContext(t *testing.T) {
	for _, tt := range []struct {
		name    string
		decline bool
	}{
		{name: "accepted"},
		{name: "declined", decline: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
			t.Cleanup(srv.Close)
			promptPath := filepath.Join(t.TempDir(), "prompt.txt")
			promptOutputPath := filepath.Join(t.TempDir(), "prompt-output.txt")
			env := map[string]string{
				"C8VOLT_TEST_CONFIG":                     writeTestConfigForVersion(t, srv.URL, "8.9"),
				"C8VOLT_TEST_ORPHAN_PURGE_PROMPT":        promptPath,
				"C8VOLT_TEST_ORPHAN_PURGE_PROMPT_OUTPUT": promptOutputPath,
			}
			if tt.decline {
				env["C8VOLT_TEST_ORPHAN_PURGE_DECLINE"] = "1"
			}

			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeOrphanProcessInstancesInteractiveHelper", env)
			if tt.decline {
				require.Error(t, err, stderr)
			} else {
				require.NoError(t, err, stderr)
			}
			promptOutput := readReportFile(t, promptOutputPath)
			require.Contains(t, readReportFile(t, promptPath), "orphan purge: 1 orphan candidate(s)")
			require.Contains(t, promptOutput, "selection scope: unfiltered across accessible tenants")
			require.Contains(t, promptOutput, "affected tenants: tenant")
			require.Less(t, strings.Index(promptOutput, "selection scope: unfiltered across accessible tenants"), strings.Index(promptOutput, "affected tenants: tenant"))
			combined := stdout + stderr
			require.Equal(t, 1, strings.Count(combined, "selection scope: unfiltered across accessible tenants"), combined)
			require.Equal(t, 1, strings.Count(combined, "affected tenants: tenant"), combined)
			wantSearchRequests := 4
			if tt.decline {
				wantSearchRequests = 2
			}
			snapshot := requests.Snapshot()
			require.Equal(t, wantSearchRequests, countRequestPrefixes(snapshot, "POST /v2/process-instances/search "), snapshot)
			require.Equal(t, 1, strings.Count(strings.Join(snapshot, "\n"), `"$exists":true`), snapshot)
			if tt.decline {
				require.Empty(t, deleted.Snapshot())
				return
			}
			require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
		})
	}
}

// TestOpsPurgeOrphanProcessInstancesAutoConfirmReportsTenantScopeBeforeWork
// verifies unfiltered selection is visible before discovery and affected
// evidence is visible before deletion without changing the frozen root target.
func TestOpsPurgeOrphanProcessInstancesAutoConfirmReportsTenantScopeBeforeWork(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	backend := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(backend.Close)
	output := &opsTenantTimingOutput{}
	proxy, observations := newOpsTenantTimingProxy(t, backend.URL, output, func(r *http.Request) bool {
		return r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/deletion")
	})
	t.Cleanup(proxy.Close)
	reset := func() {
		resetProcessInstanceCommandGlobals()
		flagOpsPurgeOrphanReportFile = ""
		flagOpsPurgeOrphanReportFormat = ""
	}

	promptCount, err := executeRootForOpsTenantTiming(t, output, reset,
		"--config", writeTestConfigForVersion(t, proxy.URL, "8.9"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
	)
	require.NoError(t, err, output.String())
	firstRequest, firstMutation := observations.snapshot()
	require.Contains(t, firstRequest, "selection scope: unfiltered across accessible tenants")
	require.Contains(t, firstMutation, "affected tenants: tenant")
	require.Zero(t, promptCount)
	require.Equal(t, 3, countRequestPrefixes(requests.Snapshot(), "POST /v2/process-instances/search "))
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesAutoConfirmNoTargetsSkipsDelete(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, false, "TERMINATED")
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
	)

	require.Contains(t, output, "candidate orphan process instances: 0")
	require.Contains(t, output, "delete plan: skipped")
	require.Contains(t, output, "outcome: planned; no targets deleted")
	require.Empty(t, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesAutomationDeletesWithoutAutoConfirm(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--automation",
		"ops", "purge", "orphan-process-instances",
		"--no-wait",
	)

	require.Contains(t, output, "deletion: submitted 1 process-instance tree (--no-wait)")
	require.NotContains(t, output, "deletion confirmation:")
	require.Contains(t, output, "outcome: deleted")
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesAutomationJSONUsesEnvelope(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--automation",
		"--json",
		"ops", "purge", "orphan-process-instances",
		"--no-wait",
	)

	require.NotContains(t, stderr, "purge orphan process-instances\n")
	require.NotContains(t, stderr, "report: written")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops purge orphan-process-instances", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "deleted", payload["outcome"])
	require.Equal(t, true, payload["deleteRequested"])
	require.NotContains(t, stdout, "purge orphan process-instances\n")
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesWritesMarkdownReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServer(t, &requests, true)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge.md")

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.8"),
		"ops", "purge", "orphan-process-instances",
		"--dry-run",
		"--report-file", reportPath,
	)

	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: planned; no changes applied"))
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Purge Orphan Process Instances Audit Report")
	require.Contains(t, report, "- Command: ops purge orphan-process-instances")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- No Wait: false")
	require.Contains(t, report, "- Outcome: planned")
	require.Contains(t, report, "- Camunda Version: 8.8")
	require.Contains(t, report, "- Profile: default")
	require.Contains(t, report, "  - "+opsOrphanChildKey)
}

// TestOpsPurgeOrphanProcessInstancesWritesJSONReport verifies submitted audit
// output retains the complete unfiltered tenant scope and frozen target facts.
func TestOpsPurgeOrphanProcessInstancesWritesJSONReport(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge.json")
	require.NoError(t, os.WriteFile(reportPath, []byte("old report"), 0o600))

	output := executeRootForProcessInstanceTest(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: deleted"))
	require.NotContains(t, readReportFile(t, reportPath), "old report")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "ops.orphan-process-instances.v1", report["schemaVersion"])
	require.Equal(t, "ops purge orphan-process-instances", report["commandName"])
	require.Equal(t, "deleted", report["outcome"])
	require.Equal(t, true, report["deleteRequested"])
	require.Equal(t, true, report["noWait"])
	require.NotContains(t, report, "dryRun")
	require.Equal(t, "8.9", report["camundaVersion"])
	require.NotContains(t, report, "tenantId")
	tenantContext := requireJSONObject(t, report["tenantContext"])
	require.Equal(t, "discovery", tenantContext["mode"])
	require.Equal(t, "none", tenantContext["filter"])
	require.Equal(t, []any{"tenant"}, tenantContext["resolvedTenantIds"])
	require.Equal(t, float64(0), tenantContext["unknownTargetCount"])
	require.Equal(t, false, tenantContext["crossTenant"])
	require.Equal(t, "unfiltered_selection", requireJSONObject(t, requireJSONItems(t, tenantContext["warnings"], 1)[0])["code"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(1), discovery["count"])
	keys := discovery["keys"].([]any)
	require.Equal(t, opsOrphanChildKey, keys[0])
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["noWait"])
}

func TestOpsPurgeOrphanProcessInstancesExistingReportFailsBeforePreflight(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge.md")
	const existingReport = "existing report"
	require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeOrphanProcessInstancesAbortPreservesExistingReportHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_REPORT": reportPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "report file already exists: "+reportPath)
	require.NotContains(t, string(output), "aborted by user")
	require.NotContains(t, string(output), "write audit report")
	require.Equal(t, existingReport, readReportFile(t, reportPath))
	require.Empty(t, requests.Snapshot())
	require.Empty(t, deleted.Snapshot())
}

func TestOpsPurgeOrphanProcessInstancesWritesReportAfterPostDiscoveryFailure(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "ACTIVE")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-purge-failed.json")

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeOrphanProcessInstancesWritesReportAfterPostDiscoveryFailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_REPORT": reportPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "refusing to delete orphan process-instance scope")
	require.Empty(t, deleted.Snapshot())
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "failed", report["outcome"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(1), discovery["count"])
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "blocked", deletion["status"])
	require.NotEmpty(t, report["errors"])
}

// TestOpsPurgeOrphanProcessInstancesProgressContractPendingT066 defines orphan
// candidate discovery, parent checking, delete planning, deletion, and report progress.
func TestOpsPurgeOrphanProcessInstancesProgressContractPendingT066(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "orphan-progress.json")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", writeTestConfigForVersion(t, srv.URL, "8.9"),
		"--verbose",
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--no-wait",
		"--batch-size", "1",
		"--report-file", reportPath,
		"--report-format", "json",
	)

	require.Contains(t, stderr, "orphan purge scope: orphan-process-instances purge matched 1 process instance; page size: 1; discovery pages: 1")
	require.Contains(t, stderr, "discovering orphan process-instance candidates, page 1/1, 1 seen")
	require.Contains(t, stderr, "checking orphan process-instance parents 1/1 process instance(s)")
	require.Contains(t, stderr, "planning orphan process-instance delete scope 1/1 process instance(s)")
	require.Contains(t, stderr, opsOrphanChildKey+" submitted (deletion process-instance trees, 1/1 process-instance tree(s), affected process instances: 1)")
	require.NotContains(t, stderr, "/v2/")
	require.NotContains(t, stderr, "cursor")
	require.NotContains(t, stdout, "orphan purge scope:")
	require.NotContains(t, stdout, "discovering orphan process-instance candidates")
	require.Contains(t, stderr, "report: written "+reportPath)
	require.Contains(t, stderr, "outcome: deleted")
	require.Equal(t, []string{"/v2/process-instances/" + opsOrphanChildKey + "/deletion"}, deleted.Snapshot())

	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "deleted", report["outcome"])
	require.Equal(t, true, report["deleteRequested"])
}

// TestOpsPurgeOrphanProcessInstancesDefaultDeletionMilestonesOmitUnknownAffected
// verifies orphan purge milestones omit affected counts when the completion
// scope cannot prove every per-root delta.
func TestOpsPurgeOrphanProcessInstancesDefaultDeletionMilestonesOmitUnknownAffected(t *testing.T) {
	resetSemanticProgressModeFlags(t)
	now := time.Date(2026, 9, 1, 7, 1, 0, 0, time.UTC)
	opsProcessInstancePurgeSemanticProgressNow = func() time.Time { return now }
	t.Cleanup(func() { opsProcessInstancePurgeSemanticProgressNow = time.Now })

	cmd, stderr := newSemanticProgressStderrCommand()
	request := ops.OrphanPurgeRequest{}
	progress := configureOpsPurgeOrphanProcessInstancesProgress(cmd, &request)
	defer progress.Close()

	reportOpsProcessInstancePurgeCompletionEvent(request.Progress, "orphan-root-1", 3, ops.CompletionDispositionConfirmed, "", nil)
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportOpsProcessInstancePurgeCompletionEvent(request.Progress, "orphan-root-2", 3, ops.CompletionDispositionConfirmed, "", nil)
	reportOpsProcessInstancePurgeCompletionEvent(request.Progress, "orphan-root-3", 3, ops.CompletionDispositionConfirmed, "", nil)
	progress.Close()

	output := stderr.String()
	require.Contains(t, output, "deletion process-instance trees, 2/3 process-instance tree(s)")
	require.Contains(t, output, "deletion process-instance trees, 3/3 process-instance tree(s)")
	require.NotContains(t, output, "affected process instances:")
}

// TestOpsPurgeOrphanProcessInstancesSemanticProgressModeGate verifies orphan
// purge deletion progress keeps protected modes silent except quiet failures.
func TestOpsPurgeOrphanProcessInstancesSemanticProgressModeGate(t *testing.T) {
	assertOpsCompletionProgressModeGate(t, opsCompletionProgressModeGateCase{
		Configure: func(cmd *cobra.Command) (func(ops.ProgressEvent), func()) {
			request := ops.OrphanPurgeRequest{}
			progress := configureOpsPurgeOrphanProcessInstancesProgress(cmd, &request)
			return request.Progress, progress.Close
		},
		Event: func(disposition ops.CompletionDisposition, detail string) ops.ProgressEvent {
			return ops.ProgressEvent{
				Kind: ops.ProgressEventKindCompletion,
				Completion: &ops.CompletionProgress{
					Phase:            "delete",
					CoreResource:     "process-instance tree(s)",
					Total:            1,
					Identity:         "orphan-root-1",
					Disposition:      disposition,
					FailureDetail:    detail,
					AffectedResource: "affected process instances",
					AffectedCount:    ptrInt(1),
				},
			}
		},
		QuietWarning: "orphan-root-1 failed: request rejected (deletion process-instance trees, 1/1 process-instance tree(s), 1 failed, affected process instances: 1)",
	})
}

// TestOpsPurgeOrphanProcessInstancesMachineProgressSafetyPendingT066 pins orphan
// purge progress silence for JSON, quiet, and automation modes.
func TestOpsPurgeOrphanProcessInstancesMachineProgressSafetyPendingT066(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{name: "json", args: []string{"--json", "ops", "purge", "orphan-process-instances", "--auto-confirm", "--no-wait", "--batch-size", "1"}},
		{name: "quiet", args: []string{"--quiet", "ops", "purge", "orphan-process-instances", "--auto-confirm", "--no-wait", "--batch-size", "1"}},
		{name: "automation", args: []string{"--automation", "ops", "purge", "orphan-process-instances", "--no-wait", "--batch-size", "1"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsOrphanPurgeServerWithState(t, &requests, &deleted, true, "TERMINATED")
			t.Cleanup(srv.Close)

			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, mode.args...)
			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
			require.NotContains(t, stdout, "orphan purge scope:")
			require.NotContains(t, stdout, "discovering orphan process-instance candidates")
			require.NotContains(t, stdout, "checking orphan process-instance parents")
			require.NotContains(t, stdout, "planning orphan process-instance delete scope")
			require.NotContains(t, stdout, "deletion process-instance trees")
			require.NotContains(t, stderr, "orphan purge scope:")
			require.NotContains(t, stderr, "discovering orphan process-instance candidates")
			require.NotContains(t, stderr, "checking orphan process-instance parents")
			require.NotContains(t, stderr, "planning orphan process-instance delete scope")
			require.NotContains(t, stderr, "deletion process-instance trees")
			if mode.name == "json" {
				var envelope map[string]any
				require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
			}
		})
	}
}

func TestOpsPurgeOrphanProcessInstancesWritesReportAfterPostDiscoveryFailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"ops", "purge", "orphan-process-instances",
		"--auto-confirm",
		"--report-file", os.Getenv("C8VOLT_TEST_REPORT"),
		"--report-format", "json",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestOpsPurgeOrphanProcessInstancesAbortPreservesExistingReportHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	confirmCmdOrAbortFn = func(_ io.Writer, _ bool, _ string) error {
		return localPreconditionError(ErrCmdAborted)
	}
	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"ops", "purge", "orphan-process-instances",
		"--report-file", os.Getenv("C8VOLT_TEST_REPORT"),
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestOpsPurgeOrphanProcessInstancesInteractiveHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var promptOutput bytes.Buffer
	confirmCmdOrAbortFn = func(_ io.Writer, autoConfirm bool, prompt string) error {
		if autoConfirm {
			return fmt.Errorf("unexpected auto-confirm prompt")
		}
		if err := os.WriteFile(os.Getenv("C8VOLT_TEST_ORPHAN_PURGE_PROMPT"), []byte(prompt), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(os.Getenv("C8VOLT_TEST_ORPHAN_PURGE_PROMPT_OUTPUT"), promptOutput.Bytes(), 0o600); err != nil {
			return err
		}
		if os.Getenv("C8VOLT_TEST_ORPHAN_PURGE_DECLINE") == "1" {
			return localPreconditionError(ErrCmdAborted)
		}
		return nil
	}
	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"ops", "purge", "orphan-process-instances",
		"--no-wait",
	})
	root.SetOut(os.Stdout)
	root.SetErr(io.MultiWriter(os.Stderr, &promptOutput))
	_ = root.Execute()
}

func readReportFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}

func newOpsOrphanPurgeServer(t *testing.T, requests *testx.SafeSlice[string], withOrphan bool) *httptest.Server {
	return newOpsOrphanPurgeServerWithState(t, requests, nil, withOrphan, "ACTIVE")
}

func newOpsOrphanPurgeServerWithState(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string], withOrphan bool, orphanState string) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			w.Header().Set("Content-Type", "application/json")
			if strings.Contains(string(body), opsOrphanChildKey) || !withOrphan {
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[` + opsOrphanProcessInstanceJSON(opsOrphanChildKey, opsOrphanParentKey, orphanState) + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsOrphanChildKey:
			requests.Append(r.Method + " " + r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(opsOrphanProcessInstanceJSON(opsOrphanChildKey, opsOrphanParentKey, orphanState)))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/"+opsOrphanParentKey:
			requests.Append(r.Method + " " + r.URL.Path)
			http.NotFound(w, r)
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/"+opsOrphanChildKey+"/deletion":
			if deleted != nil {
				deleted.Append(r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func opsOrphanProcessInstanceJSON(key string, parentKey string, state string) string {
	parent := ""
	if parentKey != "" {
		parent = `,"parentProcessInstanceKey":"` + parentKey + `","rootProcessInstanceKey":"` + opsOrphanProcessKey + `"`
	}
	return `{"processInstanceKey":"` + key + `","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"startDate":"2026-05-11T12:00:00Z","state":"` + state + `","tenantId":"tenant"` + parent + `}`
}
