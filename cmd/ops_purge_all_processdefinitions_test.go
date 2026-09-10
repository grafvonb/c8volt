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
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestOpsPurgeAllProcessDefinitionsKeyTenantContextUsesExplicitSemantics
// verifies direct process-definition keys do not pretend the configured tenant
// is a discovery filter or legacy report tenant.
func TestOpsPurgeAllProcessDefinitionsKeyTenantContextUsesExplicitSemantics(t *testing.T) {
	cmd := &cobra.Command{}
	cfg := &config.Config{App: config.App{Tenant: "tenant-a"}}
	result := ops.AllProcessDefinitionsPurgeResult{
		Request: ops.AllProcessDefinitionsPurgeRequest{
			Selection: ops.ProcessDefinitionSelection{Key: "2251799813685249"},
		},
		DeletePlan: ops.AllProcessDefinitionsPurgeDeletePlan{
			TenantEvidence: process.TenantEvidence{
				ResolvedTenantIDs: []string{"tenant-b"},
				Targets:           []process.TenantEvidenceTarget{{Key: "2251799813685249", TenantID: "tenant-b"}},
			},
		},
	}

	got := attachOpsPurgeAllProcessDefinitionsResultTenantContext(cmd, cfg, result)

	require.NotNil(t, got.Report.TenantContext)
	require.Equal(t, tenant.ContextModeExplicitKeys, got.Report.TenantContext.Mode)
	require.Equal(t, tenant.ContextFilterNotApplied, got.Report.TenantContext.Filter)
	require.Equal(t, []string{"tenant-b"}, got.Report.TenantContext.ResolvedTenantIDs)
	require.Empty(t, got.Report.TenantID)
}

// TestOpsPurgeAllProcessDefinitionsDeletionMilestonesStaySeparateFromDiscovery
// verifies APD default milestones are driven by deletion completions while
// discovery progress remains on the existing discovery path.
func TestOpsPurgeAllProcessDefinitionsDeletionMilestonesStaySeparateFromDiscovery(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)
	now := time.Date(2026, 9, 1, 6, 32, 0, 0, time.UTC)
	opsPurgeAllProcessDefinitionsProgressNow = func() time.Time { return now }
	t.Cleanup(func() { opsPurgeAllProcessDefinitionsProgressNow = time.Now })

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	stderr := &bytes.Buffer{}
	cmd.SetErr(stderr)
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	deletionProgress := newOpsPurgeAllProcessDefinitionsProgressForCommand(cmd)
	defer deletionProgress.Close()

	request := ops.AllProcessDefinitionsPurgeRequest{}
	configureOpsPurgeAllProcessDefinitionsProgress(cmd, &request, deletionProgress)
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindPage,
		Page: &ops.PageProgress{
			Phase:       "discovering process definitions",
			CurrentPage: 1,
			Seen:        1,
			Selected:    1,
		},
	})

	totalDefinitions := 2
	request.Progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindStage,
		Stage: &ops.StageProgress{
			Phase:        processDefinitionDeleteCompletionPhase,
			CoreResource: "process definition(s)",
			Total:        &totalDefinitions,
		},
	})
	now = now.Add(opsDurableMilestoneMinimumElapsed)
	reportOpsPurgeAllProcessDefinitionsCompletionEvent(request.Progress, "pd-1", 2, ops.CompletionDispositionFailed, "delete rejected")
	reportOpsPurgeAllProcessDefinitionsCompletionEvent(request.Progress, "pd-2", 2, ops.CompletionDispositionConfirmed, "")
	deletionProgress.Close()

	output := stderr.String()
	require.Contains(t, sink.PriorityUpdates(), activitysink.Update{
		Message:    "discovering process definitions, page 1, 1 seen",
		Importance: logging.ActivityImportanceWorkflow,
	})
	require.Equal(t, 1, strings.Count(output, "pd-1 failed: delete rejected (deleting process definitions, 1/2 process definition(s), 1 failed)"))
	require.Equal(t, 1, strings.Count(output, "deleting process definitions, 2/2 process definition(s), 1 failed"))
	require.NotContains(t, output, "discovering process definitions, 2/2")
}

// TestOpsPurgeAllProcessDefinitionsSemanticProgressModeGate verifies APD
// deletion completion progress remains compatible with machine and quiet modes.
func TestOpsPurgeAllProcessDefinitionsSemanticProgressModeGate(t *testing.T) {
	assertOpsCompletionProgressModeGate(t, opsCompletionProgressModeGateCase{
		Configure: func(cmd *cobra.Command) (func(ops.ProgressEvent), func()) {
			deletionProgress := newOpsPurgeAllProcessDefinitionsProgressForCommand(cmd)
			request := ops.AllProcessDefinitionsPurgeRequest{}
			configureOpsPurgeAllProcessDefinitionsProgress(cmd, &request, deletionProgress)
			totalDefinitions := 1
			request.Progress(ops.ProgressEvent{
				Kind: ops.ProgressEventKindStage,
				Stage: &ops.StageProgress{
					Phase:        processDefinitionDeleteCompletionPhase,
					CoreResource: "process definition(s)",
					Total:        &totalDefinitions,
				},
			})
			return request.Progress, deletionProgress.Close
		},
		Event: func(disposition ops.CompletionDisposition, detail string) ops.ProgressEvent {
			return ops.ProgressEvent{
				Kind: ops.ProgressEventKindCompletion,
				Completion: &ops.CompletionProgress{
					Phase:         processDefinitionDeleteCompletionPhase,
					CoreResource:  "process definition(s)",
					Total:         1,
					Identity:      "pd-1",
					Disposition:   disposition,
					FailureDetail: detail,
				},
			}
		},
		QuietWarning: "pd-1 failed: request rejected (deleting process definitions, 1/1 process definition(s), 1 failed)",
	})
}

// TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape verifies the registered command, alias, and safe examples.
func TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	output := executeRootForTest(t, "ops", "purge", "all-process-definitions", "--help")

	assertHelpOutputContainsAll(t, output,
		"Purge all selected process definitions",
		"Forced cleanup enters the actual stages as needed",
		"cancelling process-instance root trees",
		"waiting for active process instances to drain",
		"deleting process-instance histories",
		"Cancellation and history deletion count unique root trees; definition deletion counts process definitions.",
		"one completion line per root or definition in the current stage",
		"Aliases:",
		"all-pds",
		"--key string",
		"--bpmn-process-id string",
		"--pd-version int32",
		"--pd-version-tag string",
		"--latest",
		"--batch-size int32",
		"--limit int32",
		"--dry-run",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"--no-wait",
		"--force",
		"--report-file string",
		"--report-format string",
		"./c8volt ops purge all-process-definitions --dry-run",
		"./c8volt ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --dry-run",
		"./c8volt ops purge all-process-definitions --bpmn-process-id <bpmn-process-id> --latest --force",
		"./c8volt ops purge all-process-definitions --key <process-definition-key> --force --report-file process-definition-purge.md",
	)
	assertHelpOutputOmitsAll(t, output,
		"purge-definitions",
		"delete-all",
		"./c8volt ops purge all-process-definitions --automation --json\n",
		"--xml",
		"--stat",
	)

	aliasOutput := executeRootForTest(t, "ops", "purge", "all-pds", "--help")
	require.Contains(t, aliasOutput, "Purge all selected process definitions")
}

// TestOpsPurgeAllProcessDefinitionsRejectsDisplayOnlyPDFlags keeps get-pd display flags out of the purge surface.
func TestOpsPurgeAllProcessDefinitionsRejectsDisplayOnlyPDFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "xml", args: []string{"--xml"}, want: "unknown flag: --xml"},
		{name: "stat", args: []string{"--stat"}, want: "unknown flag: --stat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeAllProcessDefinitionsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsInvalidFlagsUseInvalidInput verifies local flag validation before remote work.
func TestOpsPurgeAllProcessDefinitionsInvalidFlagsUseInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid key",
			args: []string{"--key", "not-a-key"},
			want: `process definition key "not-a-key" is not a valid key`,
		},
		{
			name: "zero explicit process definition version",
			args: []string{"--pd-version", "0"},
			want: "--pd-version must be positive integer",
		},
		{
			name: "negative process definition version",
			args: []string{"--pd-version", "-1"},
			want: "--pd-version must be positive integer",
		},
		{
			name: "invalid batch size",
			args: []string{"--batch-size", "0"},
			want: "invalid value for --batch-size: 0",
		},
		{
			name: "invalid limit",
			args: []string{"--limit", "0"},
			want: "--limit must be positive integer",
		},
		{
			name: "invalid worker count",
			args: []string{"--workers", "0"},
			want: "--workers must be positive integer",
		},
		{
			name: "report format without file",
			args: []string{"--report-format", "json"},
			want: "--report-format requires --report-file",
		},
		{
			name: "unsupported report format",
			args: []string{"--report-file", "purge.txt", "--report-format", "yaml"},
			want: `unsupported ops workflow report format "yaml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeAllProcessDefinitionsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsDryRunDiscoveryOutput verifies compact discovery output for dry-run previews.
func TestOpsPurgeAllProcessDefinitionsDryRunDiscoveryOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()))
	output := buf.String()

	require.Contains(t, output, "dry run: purge all process definitions")
	require.Contains(t, output, `selection filters: {bpmnProcessId="invoice", processVersion=3, processVersionTag="stable", latestOnly=true}`)
	require.Contains(t, output, "candidate process definitions: 1")
	require.NotContains(t, output, "discovery complete:")
	require.Contains(t, output, "candidate scope: latest matching process definitions")
	require.Contains(t, output, "duplicate candidate process definitions: 1")
	require.Contains(t, output, "delete preview: skipped (no matching process definitions)")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.NotContains(t, output, "candidate process-definition keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()))
	require.Contains(t, verbose.String(), "discovery complete: pages 2; batch size 25")
	require.Contains(t, verbose.String(), "candidate process-definition keys: 2251799813685255")
	require.Contains(t, verbose.String(), "candidate process-definition details: 2251799813685255 (bpmnProcessId=invoice, version=3, versionTag=stable)")
	require.Contains(t, verbose.String(), "duplicate candidate process-definition keys: 2251799813685255")
}

// TestOpsPurgeAllProcessDefinitionsUserLimitedDiscoveryOutput verifies user-limited APD discovery is visible in compact output.
func TestOpsPurgeAllProcessDefinitionsUserLimitedDiscoveryOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	result := sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()
	result.Discovery.DiscoveryScopeStatus = ops.DiscoveryScopeStatus{
		Limited:          true,
		Limit:            1,
		BatchSize:        25,
		Pages:            1,
		CandidatesSeen:   25,
		CandidatesFrozen: 1,
	}
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, result))

	require.Contains(t, buf.String(), "discovery user-limited: limit 1; pages 1; batch size 25")
}

// TestOpsPurgeAllProcessDefinitionsDryRunJSONDiscoveryData verifies machine output carries complete discovery fields.
func TestOpsPurgeAllProcessDefinitionsDryRunJSONDiscoveryData(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "all-process-definitions"}
	cmd.SetOut(&buf)
	setContractSupport(cmd, ContractSupportFull)
	flagViewAsJson = true
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope), buf.String())
	require.Equal(t, "succeeded", envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	discovery := requireJSONObject(t, payload["discovery"])
	require.Equal(t, "planned", discovery["status"])
	require.Equal(t, float64(1), discovery["candidateProcessDefinitionCount"])
	require.Equal(t, true, discovery["complete"])
	require.Equal(t, float64(25), discovery["batchSize"])
	require.Equal(t, float64(2), discovery["pages"])
	require.Equal(t, float64(1), discovery["candidatesFrozen"])
	require.Equal(t, true, discovery["latestOnly"])
	require.Len(t, discovery["candidateProcessDefinitionKeys"], 1)
	require.Len(t, discovery["candidateProcessDefinitions"], 1)
	require.Len(t, discovery["duplicateCandidateProcessDefinitionKeys"], 1)
	require.Len(t, discovery["notices"], 2)
	require.Equal(t, "skipped", requireJSONObject(t, payload["deletePlan"])["status"])
	require.Equal(t, "skipped", requireJSONObject(t, payload["deletion"])["status"])
}

// TestOpsPurgeAllProcessDefinitionsDryRunPlanOutput verifies compact delete-plan rendering after discovery.
func TestOpsPurgeAllProcessDefinitionsDryRunPlanOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))
	output := buf.String()

	require.Contains(t, output, "delete preview: 2 candidate process definition(s), 3 affected process instance(s) would be deleted")
	require.NotContains(t, output, "\nprocess definitions:\n")
	require.Contains(t, output, "invoice [v1: 3, v2/stable: 0]")
	require.Contains(t, output, "active-instance blocker: 3 active process instances require --force before deletion")
	require.Contains(t, output, "outcome: planned; no changes applied")
	require.NotContains(t, output, "candidate process-definition keys:")
	require.NotContains(t, output, "affected process-instance keys:")

	flagVerbose = true
	var verbose bytes.Buffer
	cmd = &cobra.Command{}
	cmd.SetOut(&verbose)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))
	require.Contains(t, verbose.String(), "candidate process-definition keys: pd-a, pd-b")
	require.NotContains(t, verbose.String(), "affected process-instance keys:")
	require.NotContains(t, verbose.String(), "blocked process-instance keys:")
}

// TestOpsPurgeAllProcessDefinitionsDryRunJSONPlanData verifies machine output carries complete delete-plan fields.
func TestOpsPurgeAllProcessDefinitionsDryRunJSONPlanData(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var buf bytes.Buffer
	cmd := &cobra.Command{Use: "all-process-definitions"}
	cmd.SetOut(&buf)
	setContractSupport(cmd, ContractSupportFull)
	flagViewAsJson = true
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope), buf.String())
	payload := requireJSONObject(t, envelope["payload"])
	plan := requireJSONObject(t, payload["deletePlan"])
	require.Equal(t, "planned", plan["status"])
	require.Len(t, plan["candidateProcessDefinitionKeys"], 2)
	require.Len(t, plan["duplicateCandidateProcessDefinitionKeys"], 1)
	require.Len(t, plan["items"], 2)
	require.Equal(t, float64(3), plan["affectedProcessInstanceCount"])
	require.Equal(t, float64(3), plan["activeProcessInstanceCount"])
	require.Equal(t, true, plan["requiresForce"])
	require.Equal(t, true, plan["requiresConfirmation"])
}

// TestOpsPurgeAllProcessDefinitionsJSONOutputIsDeterministic verifies dry-run machine output is stable and complete.
func TestOpsPurgeAllProcessDefinitionsJSONOutputIsDeterministic(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	render := func() string {
		var buf bytes.Buffer
		cmd := &cobra.Command{Use: "all-process-definitions"}
		cmd.SetOut(&buf)
		setContractSupport(cmd, ContractSupportFull)
		flagViewAsJson = true
		require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, sampleAllProcessDefinitionsPurgeDryRunPlanResult()))
		return buf.String()
	}

	first := render()
	second := render()
	require.Equal(t, first, second)
	require.NotContains(t, first, "dry run: purge all process definitions")

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(first), &envelope), first)
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "planned", payload["outcome"])
	require.Equal(t, "planned", requireJSONObject(t, payload["discovery"])["status"])
	require.Equal(t, "planned", requireJSONObject(t, payload["deletePlan"])["status"])
	require.Equal(t, "skipped", requireJSONObject(t, payload["deletion"])["status"])
}

// TestOpsPurgeAllProcessDefinitionsConfirmedDeletionUsesFrozenCandidates verifies prompted deletion submits only the planned scope.
func TestOpsPurgeAllProcessDefinitionsConfirmedDeletionUsesFrozenCandidates(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	promptPath := filepath.Join(t.TempDir(), "prompt.txt")

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_PROMPT": promptPath,
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--no-wait",
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, readReportFile(t, promptPath), "process-definition purge: 2 candidate process definition(s)")
	require.Contains(t, output, "deletion: submitted 2 process definitions (--no-wait)")
	require.NotContains(t, output, "deletion confirmation:")
	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "elapsed:")
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, deleted.Snapshot())
	require.Equal(t, 1, countOpsPurgeAllProcessDefinitionsRequests(requests.Snapshot(), "POST /v2/process-definitions/search "))
}

// TestOpsPurgeAllProcessDefinitionsPagedConfirmationReusesFrozenCandidates verifies confirmed APD mutation does not rediscover.
func TestOpsPurgeAllProcessDefinitionsPagedConfirmationReusesFrozenCandidates(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	promptPath := filepath.Join(t.TempDir(), "prompt.txt")

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":              writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_PROMPT": promptPath,
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--batch-size", "1",
			"--no-wait",
		}),
	})
	require.NoError(t, err, string(outputBytes))

	require.Contains(t, readReportFile(t, promptPath), "process-definition purge: 2 candidate process definition(s)")
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, deleted.Snapshot())
	require.Equal(t, 2, countOpsPurgeAllProcessDefinitionsRequests(requests.Snapshot(), "POST /v2/process-definitions/search "))
}

// TestOpsPurgeAllProcessDefinitionsVerboseDiscoveryProgress defines APD discovery progress routing.
func TestOpsPurgeAllProcessDefinitionsVerboseDiscoveryProgress(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"--verbose",
			"--auto-confirm",
			"ops", "purge", "all-process-definitions",
			"--batch-size", "1",
			"--dry-run",
		}),
	})
	require.NoError(t, err, stderr)
	require.Empty(t, deleted.Snapshot())

	require.Equal(t, 2, countProcessDefinitionSearchRequests(requests.Snapshot()))
	require.Contains(t, stderr, "process-definition purge scope: all-process-definitions purge matched at least 2 process definitions; page size: 1; discovery pages: at least 2")
	require.Contains(t, stderr, "discovering process definitions, page 1/~2, 1 seen")
	require.Contains(t, stderr, "discovering process definitions, page 2/2, 2 seen")
	require.NotContains(t, stdout, "process-definition purge scope:")
	require.NotContains(t, stdout, "discovering process definitions")
	require.Contains(t, stderr, "dry run: purge all process definitions")
	require.Contains(t, stderr, "candidate process definitions: 2")
}

// TestOpsPurgeAllProcessDefinitionsLimitJSONOutput verifies APD flags reach discovery and machine output.
func TestOpsPurgeAllProcessDefinitionsLimitJSONOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"--json",
			"ops", "purge", "all-process-definitions",
			"--batch-size", "2",
			"--limit", "1",
			"--dry-run",
		}),
	})
	require.NoError(t, err, stderr)
	require.Empty(t, deleted.Snapshot())

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
	payload := requireJSONObject(t, envelope["payload"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, float64(2), request["batchSize"])
	require.Equal(t, float64(1), request["limit"])
	discovery := requireJSONObject(t, payload["discovery"])
	require.NotContains(t, discovery, "complete")
	require.Equal(t, true, discovery["limited"])
	require.Equal(t, float64(1), discovery["limit"])
	require.Equal(t, float64(2), discovery["batchSize"])
	require.Equal(t, float64(1), discovery["pages"])
	require.Equal(t, float64(1), discovery["candidatesFrozen"])
	require.Len(t, discovery["candidateProcessDefinitionKeys"], 1)
	require.Equal(t, 1, countOpsPurgeAllProcessDefinitionsRequests(requests.Snapshot(), "POST /v2/process-definitions/search "))
}

// TestOpsPurgeAllProcessDefinitionsAutomationJSONExecutesWithoutAutoConfirm verifies automation confirms the supported destructive path.
func TestOpsPurgeAllProcessDefinitionsAutomationJSONExecutesWithoutAutoConfirm(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"--automation",
			"--json",
			"ops", "purge", "all-process-definitions",
			"--workers", "2",
			"--fail-fast",
			"--no-worker-limit",
			"--no-wait",
			"--force",
		}),
	})
	require.NoError(t, err, stderr)

	require.NotContains(t, stderr, "purge all process definitions")
	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "ops purge all-process-definitions", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "deleted", payload["outcome"])
	request := requireJSONObject(t, payload["request"])
	require.Equal(t, true, request["automation"])
	require.NotContains(t, request, "autoConfirm")
	require.Equal(t, float64(2), request["workers"])
	require.Equal(t, true, request["failFast"])
	require.Equal(t, true, request["noWorkerLimit"])
	require.Equal(t, true, request["noWait"])
	require.Equal(t, true, request["force"])
	deletion := requireJSONObject(t, payload["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["submitted"])
	require.Equal(t, true, deletion["noWait"])
	require.Len(t, deletion["submittedProcessDefinitionKeys"], 2)
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, deleted.Snapshot())
}

// TestOpsPurgeAllProcessDefinitionsForceCleanupActivityFollowsNestedStages
// exercises the real command, facade, and service chain while paused inside
// nested cleanup so old definition-only progress wiring cannot satisfy it.
func TestOpsPurgeAllProcessDefinitionsForceCleanupActivityFollowsNestedStages(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	state := newOpsPurgeAllProcessDefinitionsNestedState()
	var requests testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsNestedServer(t, state, &requests)
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		state.releaseCancel.Do(func() { close(state.cancelRelease) })
		state.releaseDrain.Do(func() { close(state.drainRelease) })
	})

	sink := &activitysink.Sink{}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd := opsPurgeAllProcessDefinitionsCmd
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetContext(newOpsPurgeAllProcessDefinitionsActivityCommandContext(t, srv.URL, sink, &stderr))
	flagCmdAutoConfirm = true
	flagForce = true
	flagNoWait = true
	flagWorkers = 1

	done := make(chan struct{})
	go func() {
		defer close(done)
		cmd.Run(cmd, nil)
	}()

	requireOpsPurgeAllProcessDefinitionsSignal(t, state.cancelStarted)
	requireOpsPurgeAllProcessDefinitionsActivityContains(t, sink, "cancelling process-instance root trees, 0/3 process-instance tree(s), affected scope: 3 process instance(s)")
	state.releaseCancel.Do(func() { close(state.cancelRelease) })

	require.Eventually(t, func() bool {
		select {
		case <-state.drainPolled:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond, "requests: %v stderr: %s", requests.Snapshot(), stderr.String())
	requireOpsPurgeAllProcessDefinitionsActivityContains(t, sink, "waiting for active process instances to drain")
	state.releaseDrain.Do(func() { close(state.drainRelease) })

	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)

	requireOpsPurgeAllProcessDefinitionsActivityContains(t, sink, "deleting process-instance histories, 0/3 process-instance tree(s)")
	requireOpsPurgeAllProcessDefinitionsActivityContains(t, sink, "deleting process definitions, 0/2 process definition(s)")
	require.ElementsMatch(t, []string{
		"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootA + "/cancellation",
		"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootShared + "/cancellation",
		"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootB + "/cancellation",
	}, state.cancelled.Snapshot())
	require.ElementsMatch(t, []string{
		"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootA + "/deletion",
		"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootShared + "/deletion",
		"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootB + "/deletion",
	}, state.deletedPI.Snapshot())
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, state.deletedPD.Snapshot())
	require.NotEmpty(t, requests.Snapshot())
	require.Eventually(t, func() bool {
		return sink.Started() > 0 && sink.Started() == sink.Stopped()
	}, 5*time.Second, 10*time.Millisecond, "activity starts: %v stops: %d", sink.Starts(), sink.Stopped())
}

// TestOpsPurgeAllProcessDefinitionsForceCleanupDefaultWarningsUseEnteredStages
// verifies nested force-cleanup failures produce one immediate warning with the
// stage aggregate that owns the failed item.
func TestOpsPurgeAllProcessDefinitionsForceCleanupDefaultWarningsUseEnteredStages(t *testing.T) {
	tests := []struct {
		name       string
		configure  func(*opsPurgeAllProcessDefinitionsNestedState)
		wantFailed string
		wantStage  string
	}{
		{
			name: "cancellation",
			configure: func(state *opsPurgeAllProcessDefinitionsNestedState) {
				state.failCancelKey = opsAllProcessDefinitionsPurgeRootA
			},
			wantFailed: opsAllProcessDefinitionsPurgeRootA + " failed:",
			wantStage:  "cancelling process-instance root trees, 1/3 process-instance tree(s), 1 failed",
		},
		{
			name: "history",
			configure: func(state *opsPurgeAllProcessDefinitionsNestedState) {
				state.failHistoryKey = opsAllProcessDefinitionsPurgeRootA
			},
			wantFailed: opsAllProcessDefinitionsPurgeRootA + " failed:",
			wantStage:  "deleting process-instance histories, 1/3 process-instance tree(s), 1 failed",
		},
		{
			name: "definition",
			configure: func(state *opsPurgeAllProcessDefinitionsNestedState) {
				state.failDefinitionKey = opsAllProcessDefinitionsPurgePDKeyA
			},
			wantFailed: opsAllProcessDefinitionsPurgePDKeyA + " failed:",
			wantStage:  "deleting process definitions, 1/2 process definition(s), 1 failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newOpsPurgeAllProcessDefinitionsNestedState()
			tt.configure(state)
			state.releaseDrain.Do(func() { close(state.drainRelease) })
			var requests testx.SafeSlice[string]
			srv := newOpsPurgeAllProcessDefinitionsNestedServer(t, state, &requests)
			t.Cleanup(srv.Close)

			_, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
				"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
					"ops", "purge", "all-process-definitions",
					"--auto-confirm",
					"--force",
					"--no-wait",
					"--workers", "1",
				}),
			})
			require.Error(t, err, stderr)
			require.Equal(t, 1, strings.Count(stderr, tt.wantFailed), stderr)
			require.Contains(t, stderr, tt.wantStage)
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsForceCleanupVerboseAndDebugPrintOneOutcome
// verifies diagnostic modes replace aggregate milestones with one item outcome
// for each nested stage completion and preserve no-wait wording.
func TestOpsPurgeAllProcessDefinitionsForceCleanupVerboseAndDebugPrintOneOutcome(t *testing.T) {
	tests := []struct {
		name           string
		flag           string
		noWait         bool
		cancelVerb     string
		mutationVerb   string
		deletionOutput string
	}{
		{name: "verbose no-wait", flag: "--verbose", noWait: true, cancelVerb: "submitted", mutationVerb: "submitted", deletionOutput: "deletion: submitted 2 process definitions (--no-wait)"},
		{name: "debug no-wait", flag: "--debug", noWait: true, cancelVerb: "submitted", mutationVerb: "submitted", deletionOutput: "deletion: submitted 2 process definitions (--no-wait)"},
		{name: "verbose confirmed", flag: "--verbose", cancelVerb: "cancelled", mutationVerb: "deleted", deletionOutput: "deletion: removed 2 process definitions"},
		{name: "debug confirmed", flag: "--debug", cancelVerb: "cancelled", mutationVerb: "deleted", deletionOutput: "deletion: removed 2 process definitions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newOpsPurgeAllProcessDefinitionsNestedState()
			state.releaseDrain.Do(func() { close(state.drainRelease) })
			var requests testx.SafeSlice[string]
			srv := newOpsPurgeAllProcessDefinitionsNestedServer(t, state, &requests)
			t.Cleanup(srv.Close)

			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":            writeTestConfigForVersion(t, srv.URL, "8.9"),
				"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, opsPurgeAllProcessDefinitionsNestedDiagnosticArgs(tt.flag, tt.noWait)),
			})
			require.NoError(t, err, stderr)

			require.Empty(t, stdout)
			require.Contains(t, stderr, tt.deletionOutput)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgeRootA+" "+tt.cancelVerb+" (cancelling process-instance root trees", 1)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgeRootShared+" "+tt.cancelVerb+" (cancelling process-instance root trees", 1)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgeRootB+" "+tt.cancelVerb+" (cancelling process-instance root trees", 1)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgeRootA+" "+tt.mutationVerb+" (deleting process-instance histories", 1)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgeRootShared+" "+tt.mutationVerb+" (deleting process-instance histories", 1)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgeRootB+" "+tt.mutationVerb+" (deleting process-instance histories", 1)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgePDKeyA+" "+tt.mutationVerb+" (deleting process definitions", 1)
			requireOpsPurgeAllProcessDefinitionsOutcomeCount(t, stderr, opsAllProcessDefinitionsPurgePDKeyB+" "+tt.mutationVerb+" (deleting process definitions", 1)
			require.NotContains(t, stderr, "stage progress:")
			require.NotContains(t, stderr, "cancelling process-instance root trees, 3/3 process-instance tree(s)\n")
			require.NotContains(t, stderr, "deleting process-instance histories, 3/3 process-instance tree(s)\n")
			require.NotContains(t, stderr, "deleting process definitions, 2/2 process definition(s)\n")
		})
	}
}

// opsPurgeAllProcessDefinitionsNestedDiagnosticArgs builds the root/command
// argument order needed for diagnostic progress mode assertions.
func opsPurgeAllProcessDefinitionsNestedDiagnosticArgs(modeFlag string, noWait bool) []string {
	args := []string{
		modeFlag,
		"ops", "purge", "all-process-definitions",
		"--auto-confirm",
		"--force",
		"--workers", "1",
	}
	if noWait {
		args = append(args, "--no-wait")
	}
	return args
}

// TestOpsPurgeAllProcessDefinitionsForceCleanupMachineModeCompatibility
// verifies real nested force-cleanup stage facts stay out of machine-mode
// output while final JSON envelopes and mutation results remain stable.
func TestOpsPurgeAllProcessDefinitionsForceCleanupMachineModeCompatibility(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "json",
			args: []string{
				"--json",
				"ops", "purge", "all-process-definitions",
				"--auto-confirm",
				"--force",
				"--no-wait",
				"--workers", "1",
			},
		},
		{
			name: "json verbose",
			args: []string{
				"--json",
				"--verbose",
				"ops", "purge", "all-process-definitions",
				"--auto-confirm",
				"--force",
				"--no-wait",
				"--workers", "1",
			},
		},
		{
			name: "automation json verbose",
			args: []string{
				"--automation",
				"--json",
				"--verbose",
				"ops", "purge", "all-process-definitions",
				"--force",
				"--no-wait",
				"--workers", "1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := newOpsPurgeAllProcessDefinitionsNestedState()
			state.releaseDrain.Do(func() { close(state.drainRelease) })
			var requests testx.SafeSlice[string]
			srv := newOpsPurgeAllProcessDefinitionsNestedServer(t, state, &requests)
			t.Cleanup(srv.Close)

			stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":            writeTestConfigForVersion(t, srv.URL, "8.9"),
				"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, tt.args),
			})
			require.NoError(t, err, stderr)
			assertOpsPurgeAllProcessDefinitionsNoStageProgress(t, stdout)
			assertOpsPurgeAllProcessDefinitionsNoStageProgress(t, stderr)

			var envelope map[string]any
			require.NoError(t, json.Unmarshal([]byte(stdout), &envelope), stdout)
			require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
			payload := requireJSONObject(t, envelope["payload"])
			require.Equal(t, "deleted", payload["outcome"])
			deletion := requireJSONObject(t, payload["deletion"])
			require.Equal(t, "submitted", deletion["status"])
			require.Equal(t, true, deletion["submitted"])
			require.Equal(t, true, deletion["noWait"])
			require.Len(t, deletion["submittedProcessDefinitionKeys"], 2)
			require.ElementsMatch(t, []string{
				"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootA + "/cancellation",
				"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootShared + "/cancellation",
				"/v2/process-instances/" + opsAllProcessDefinitionsPurgeRootB + "/cancellation",
			}, state.cancelled.Snapshot())
			require.ElementsMatch(t, []string{
				"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
				"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
			}, state.deletedPD.Snapshot())
			require.NotEmpty(t, requests.Snapshot())
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsQuietForceCleanupFailureCompatibility
// verifies quiet mode still surfaces the immediate failed item warning without
// adding successful progress chatter.
func TestOpsPurgeAllProcessDefinitionsQuietForceCleanupFailureCompatibility(t *testing.T) {
	state := newOpsPurgeAllProcessDefinitionsNestedState()
	state.failCancelKey = opsAllProcessDefinitionsPurgeRootA
	var requests testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsNestedServer(t, state, &requests)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"--quiet",
			"ops", "purge", "all-process-definitions",
			"--auto-confirm",
			"--force",
			"--no-wait",
			"--workers", "1",
		}),
	})
	require.Error(t, err, stderr)
	require.Empty(t, stdout)
	require.Contains(t, stderr, opsAllProcessDefinitionsPurgeRootA+" failed:")
	require.Contains(t, stderr, "cancelling process-instance root trees, 1/3 process-instance tree(s), 1 failed")
	require.NotContains(t, stderr, "waiting for active process instances to drain")
	require.NotContains(t, stderr, "deleting process-instance histories")
	require.NotContains(t, stderr, "deleting process definitions")
	require.NotContains(t, stderr, "stage progress:")
	require.NotEmpty(t, requests.Snapshot())
}

// TestOpsPurgeAllProcessDefinitionsDeclinedConfirmationCompatibility
// verifies the interactive safety prompt still aborts before mutation when the
// operator declines the frozen plan.
func TestOpsPurgeAllProcessDefinitionsDeclinedConfirmationCompatibility(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	promptPath := filepath.Join(t.TempDir(), "prompt.txt")

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":               writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_PROMPT":  promptPath,
		"C8VOLT_TEST_ALL_PD_PURGE_DECLINE": "1",
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS":    marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{"ops", "purge", "all-process-definitions"}),
	})
	require.Error(t, err, stderr)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "confirmation declined")
	require.Contains(t, readReportFile(t, promptPath), "process-definition purge: 2 candidate process definition(s), 0 affected process instance(s) will be deleted")
	assertOpsPurgeAllProcessDefinitionsNoStageProgress(t, stderr)
	require.Empty(t, deleted.Snapshot())
	require.Equal(t, 1, countOpsPurgeAllProcessDefinitionsRequests(requests.Snapshot(), "POST /v2/process-definitions/search "))
}

// TestOpsPurgeAllProcessDefinitionsEmptySelectionCompatibility verifies empty
// discovery still skips planning and mutation without introducing progress text.
func TestOpsPurgeAllProcessDefinitionsEmptySelectionCompatibility(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsEmptyServer(t, &requests, &deleted)
	t.Cleanup(srv.Close)

	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--dry-run",
		}),
	})
	require.NoError(t, err, stderr)
	output := stdout + stderr
	require.Contains(t, output, "dry run: purge all process definitions")
	require.Contains(t, output, "candidate process definitions: 0")
	require.Contains(t, output, "delete preview: skipped (no matching process definitions)")
	require.Contains(t, output, "outcome: planned; no changes applied")
	assertOpsPurgeAllProcessDefinitionsNoStageProgress(t, output)
	require.Empty(t, deleted.Snapshot())
	require.Equal(t, 1, countOpsPurgeAllProcessDefinitionsRequests(requests.Snapshot(), "POST /v2/process-definitions/search "))
}

// TestOpsPurgeAllProcessDefinitionsBlocksActiveInstancesBeforeMutation verifies post-planning blockers keep local-precondition exit behavior.
func TestOpsPurgeAllProcessDefinitionsBlocksActiveInstancesBeforeMutation(t *testing.T) {
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 3)
	t.Cleanup(srv.Close)

	output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
		}),
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "refusing to delete all-process-definitions purge scope")
	require.Contains(t, string(output), "active process instance")
	require.Empty(t, deleted.Snapshot())
}

// TestOpsPurgeAllProcessDefinitionsDeletionOutput verifies compact execution rendering.
func TestOpsPurgeAllProcessDefinitionsDeletionOutput(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	result := sampleAllProcessDefinitionsPurgeDeletedResult()
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	require.NoError(t, renderOpsPurgeAllProcessDefinitionsResult(cmd, result))
	output := buf.String()

	require.Contains(t, output, "purge all process definitions")
	require.Contains(t, output, "deletion: submitted 2 process definitions (--no-wait)")
	require.NotContains(t, output, "deletion confirmation:")
	require.Contains(t, output, "outcome: deleted")
}

// TestOpsPurgeAllProcessDefinitionsWritesMarkdownReport verifies the dry-run report includes complete audit sections.
func TestOpsPurgeAllProcessDefinitionsWritesMarkdownReport(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "all-pd-purge.md")

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--dry-run",
			"--report-file", reportPath,
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, output, "outcome: planned; no changes applied")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: planned; no changes applied"))
	require.Empty(t, deleted.Snapshot())
	report := readReportFile(t, reportPath)
	require.Contains(t, report, "# Purge All Process Definitions Audit Report")
	require.Contains(t, report, "- Command: ops purge all-process-definitions")
	require.Contains(t, report, "- Dry Run: true")
	require.Contains(t, report, "- Camunda Version: 8.9")
	require.Contains(t, report, "- Profile: default")
	require.Contains(t, report, "- Outcome: planned")
	require.Contains(t, report, "## Discovery")
	require.Contains(t, report, "- Completeness: discovery complete")
	require.Contains(t, report, "- Discovery Batch Size: 1000")
	require.Contains(t, report, "- Candidate Process-Definition Keys:")
	require.Contains(t, report, "  - "+opsAllProcessDefinitionsPurgePDKeyA)
	require.Contains(t, report, "## Delete Plan")
	require.Contains(t, report, "- Affected Process Instances: 0")
}

// TestOpsPurgeAllProcessDefinitionsWritesJSONReport verifies confirmed runs overwrite only after deletion submission.
func TestOpsPurgeAllProcessDefinitionsWritesJSONReport(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)
	var requests testx.SafeSlice[string]
	var deleted testx.SafeSlice[string]
	srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
	t.Cleanup(srv.Close)
	reportPath := filepath.Join(t.TempDir(), "all-pd-purge.json")
	require.NoError(t, os.WriteFile(reportPath, []byte("old report"), 0o600))

	outputBytes, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, "8.9"),
		"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
			"ops", "purge", "all-process-definitions",
			"--auto-confirm",
			"--no-wait",
			"--report-file", reportPath,
			"--report-format", "json",
		}),
	})
	require.NoError(t, err, string(outputBytes))
	output := string(outputBytes)

	require.Contains(t, output, "outcome: deleted")
	require.Contains(t, output, "report: written "+reportPath)
	require.Less(t, strings.Index(output, "report: written "+reportPath), strings.Index(output, "outcome: deleted"))
	require.NotContains(t, readReportFile(t, reportPath), "old report")
	var report map[string]any
	require.NoError(t, json.Unmarshal([]byte(readReportFile(t, reportPath)), &report))
	require.Equal(t, "ops.all-process-definitions.v1", report["schemaVersion"])
	require.Equal(t, "ops purge all-process-definitions", report["commandName"])
	require.Equal(t, "deleted", report["outcome"])
	require.Equal(t, true, report["noWait"])
	require.Equal(t, "8.9", report["camundaVersion"])
	discovery := requireJSONObject(t, report["discovery"])
	require.Equal(t, float64(2), discovery["candidateProcessDefinitionCount"])
	require.Len(t, discovery["candidateProcessDefinitionKeys"], 2)
	deletePlan := requireJSONObject(t, report["deletePlan"])
	require.Len(t, deletePlan["candidateProcessDefinitionKeys"], 2)
	deletion := requireJSONObject(t, report["deletion"])
	require.Equal(t, "submitted", deletion["status"])
	require.Equal(t, true, deletion["submitted"])
	require.ElementsMatch(t, []string{
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyA + "/deletion",
		"/v2/resources/" + opsAllProcessDefinitionsPurgePDKeyB + "/deletion",
	}, deleted.Snapshot())
}

// TestOpsPurgeAllProcessDefinitionsExistingReportPreservation verifies non-submitted paths never clobber reports.
func TestOpsPurgeAllProcessDefinitionsExistingReportPreservation(t *testing.T) {
	tests := []struct {
		name         string
		args         []string
		activeCount  int64
		want         string
		wantRequests bool
	}{
		{
			name: "dry run",
			args: []string{"ops", "purge", "all-process-definitions", "--dry-run"},
			want: "report file already exists:",
		},
		{
			name: "unconfirmed",
			args: []string{"ops", "purge", "all-process-definitions"},
			want: "report file already exists:",
		},
		{
			name:         "locally blocked",
			args:         []string{"ops", "purge", "all-process-definitions", "--auto-confirm"},
			activeCount:  3,
			want:         "write audit report: report file already exists:",
			wantRequests: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, tt.activeCount)
			t.Cleanup(srv.Close)
			reportPath := filepath.Join(t.TempDir(), "all-pd-purge.md")
			const existingReport = "existing report"
			require.NoError(t, os.WriteFile(reportPath, []byte(existingReport), 0o600))
			args := append([]string{}, tt.args...)
			args = append(args, "--report-file", reportPath)

			output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":            writeTestConfigForVersion(t, srv.URL, "8.9"),
				"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), tt.want)
			require.Equal(t, existingReport, readReportFile(t, reportPath))
			require.Empty(t, deleted.Snapshot())
			if tt.wantRequests {
				require.NotEmpty(t, requests.Snapshot())
			} else {
				require.Empty(t, requests.Snapshot())
			}
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsRejectsUnsupportedFullHistoryVersionsBeforePlanning
// keeps unsupported full-history delete capability failures before APD discovery.
func TestOpsPurgeAllProcessDefinitionsRejectsUnsupportedFullHistoryVersionsBeforePlanning(t *testing.T) {
	for _, version := range []toolx.CamundaVersion{toolx.V87, toolx.V88} {
		t.Run(version.String(), func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
			t.Cleanup(srv.Close)

			output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, version.String()),
				"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
					"ops", "purge", "all-process-definitions",
					"--dry-run",
				}),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.Error, exitErr.ExitCode())
			require.Contains(t, string(output), "unsupported capability")
			require.Contains(t, string(output), "all-process-definitions purge requires the full process-definition history deletion capability, currently Camunda 8.9 or newer")
			require.Empty(t, requests.Snapshot())
			require.Empty(t, deleted.Snapshot())
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsAcceptsFullHistoryCapabilityVersionsBeforeDiscovery
// proves V89 and V810 pass the APD capability gate and reach dry-run discovery.
func TestOpsPurgeAllProcessDefinitionsAcceptsFullHistoryCapabilityVersionsBeforeDiscovery(t *testing.T) {
	for _, version := range []toolx.CamundaVersion{toolx.V89, toolx.V810} {
		t.Run(version.String(), func(t *testing.T) {
			var requests testx.SafeSlice[string]
			var deleted testx.SafeSlice[string]
			srv := newOpsPurgeAllProcessDefinitionsServer(t, &requests, &deleted, 0)
			t.Cleanup(srv.Close)

			output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeAllProcessDefinitionsCommandHelper", map[string]string{
				"C8VOLT_TEST_CONFIG": writeTestConfigForVersion(t, srv.URL, version.String()),
				"C8VOLT_TEST_ALL_PD_PURGE_ARGS": marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t, []string{
					"ops", "purge", "all-process-definitions",
					"--dry-run",
				}),
			})

			require.NoError(t, err, string(output))
			require.Contains(t, string(output), "dry run: purge all process definitions")
			require.NotEmpty(t, requests.Snapshot())
			require.Empty(t, deleted.Snapshot())
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsCommandHelper runs all-process-definitions purge command subprocess cases.
func TestOpsPurgeAllProcessDefinitionsCommandHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_ALL_PD_PURGE_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetOpsPurgeAllProcessDefinitionsFlagState()
	if promptPath := os.Getenv("C8VOLT_TEST_ALL_PD_PURGE_PROMPT"); promptPath != "" {
		prevConfirm := confirmCmdOrAbortFn
		defer func() { confirmCmdOrAbortFn = prevConfirm }()
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			if autoConfirm {
				return fmt.Errorf("unexpected auto-confirm prompt")
			}
			if err := os.WriteFile(promptPath, []byte(prompt), 0o600); err != nil {
				return err
			}
			if os.Getenv("C8VOLT_TEST_ALL_PD_PURGE_DECLINE") == "1" {
				return fmt.Errorf("confirmation declined")
			}
			return nil
		}
	}
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
	os.Exit(0)
}

// executeOpsPurgeAllProcessDefinitionsExpectError runs all-process-definitions purge and returns Cobra parse/validation errors.
func executeOpsPurgeAllProcessDefinitionsExpectError(t *testing.T, args ...string) (string, error) {
	t.Helper()

	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(append([]string{"ops", "purge", "all-process-definitions"}, args...))
	resetCommandTreeFlags(root)
	resetOpsPurgeAllProcessDefinitionsFlagState()

	_, err := root.ExecuteC()
	if err != nil {
		return buf.String() + err.Error(), err
	}
	return buf.String(), nil
}

// reportOpsPurgeAllProcessDefinitionsCompletionEvent sends one ops-level APD
// deletion completion fact through the configured command progress callback.
func reportOpsPurgeAllProcessDefinitionsCompletionEvent(progress func(ops.ProgressEvent), identity string, total int, disposition ops.CompletionDisposition, detail string) {
	progress(ops.ProgressEvent{
		Kind: ops.ProgressEventKindCompletion,
		Completion: &ops.CompletionProgress{
			Phase:         processDefinitionDeleteCompletionPhase,
			CoreResource:  "process definition(s)",
			Total:         total,
			Identity:      identity,
			Disposition:   disposition,
			FailureDetail: detail,
		},
	})
}

// sampleAllProcessDefinitionsPurgeDeletedResult returns a successful no-wait deletion result for command rendering tests.
func sampleAllProcessDefinitionsPurgeDeletedResult() ops.AllProcessDefinitionsPurgeResult {
	result := sampleAllProcessDefinitionsPurgeDryRunPlanResult()
	result.Request.DryRun = false
	result.Outcome = ops.AllProcessDefinitionsPurgeOutcomeDeleted
	result.DeletePlan.RequiresForce = false
	result.Deletion = ops.AllProcessDefinitionsPurgeDeletionResult{
		Status:                         ops.WorkflowStepStatusSubmitted,
		SubmittedProcessDefinitionKeys: typex.Keys{"pd-a", "pd-b"},
		Items: []resource.DeleteReport{
			{Key: "pd-a", Ok: true, StatusCode: http.StatusOK, Status: "200 OK", DeleteHistory: true},
			{Key: "pd-b", Ok: true, StatusCode: http.StatusOK, Status: "200 OK", DeleteHistory: true},
		},
		Submitted: true,
		NoWait:    true,
	}
	return result
}

// sampleAllProcessDefinitionsPurgeDryRunPlanResult returns a planned purge result for command rendering tests.
func sampleAllProcessDefinitionsPurgeDryRunPlanResult() ops.AllProcessDefinitionsPurgeResult {
	result := sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult()
	result.Discovery.CandidateProcessDefinitionKeys = typex.Keys{"pd-a", "pd-b"}
	result.Discovery.CandidateProcessDefinitions = []process.ProcessDefinition{
		{Key: "pd-a", BpmnProcessId: "invoice", ProcessVersion: 1},
		{Key: "pd-b", BpmnProcessId: "invoice", ProcessVersion: 2, ProcessVersionTag: "stable"},
	}
	result.Discovery.CandidateProcessDefinitionCount = 2
	result.DeletePlan = ops.AllProcessDefinitionsPurgeDeletePlan{
		Status:                         ops.WorkflowStepStatusPlanned,
		CandidateProcessDefinitionKeys: typex.Keys{"pd-a", "pd-b"},
		Items: []resource.DeleteProcessDefinitionPlanItem{
			{
				Key:                        "pd-a",
				ActiveProcessInstanceCount: 3,
				ActiveProcessInstanceKeys:  []string{"pi-a", "pi-b", "pi-c"},
				CancellationPlan: process.DryRunPIKeyExpansion{
					Collected: typex.Keys{"pi-a", "pi-b", "pi-c"},
					RequiresCancelBeforeDelete: []process.ProcessInstance{
						{Key: "pi-a"},
						{Key: "pi-b"},
						{Key: "pi-c"},
					},
				},
			},
			{Key: "pd-b"},
		},
		DuplicateCandidateProcessDefinitionKeys: typex.Keys{"pd-a"},
		AffectedProcessInstanceCount:            3,
		ActiveProcessInstanceCount:              3,
		RequiresConfirmation:                    true,
		RequiresForce:                           true,
	}
	return result
}

// sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult returns a discovery-only purge result for command rendering tests.
func sampleAllProcessDefinitionsPurgeDryRunDiscoveryResult() ops.AllProcessDefinitionsPurgeResult {
	result := ops.AllProcessDefinitionsPurgeResult{
		Request: ops.AllProcessDefinitionsPurgeRequest{
			CommandName: "ops purge all-process-definitions",
			DryRun:      true,
			BatchSize:   25,
			Limit:       10,
			Selection: ops.ProcessDefinitionSelection{
				BpmnProcessId:     "invoice",
				ProcessVersion:    3,
				ProcessVersionTag: "stable",
				LatestOnly:        true,
			},
		},
		Discovery: ops.ProcessDefinitionDiscoveryResult{
			Status:                         ops.WorkflowStepStatusPlanned,
			Filters:                        ops.ProcessDefinitionSelection{BpmnProcessId: "invoice", ProcessVersion: 3, ProcessVersionTag: "stable", LatestOnly: true},
			CandidateProcessDefinitionKeys: typex.Keys{"2251799813685255"},
			CandidateProcessDefinitions: []process.ProcessDefinition{{
				Key:               "2251799813685255",
				BpmnProcessId:     "invoice",
				ProcessVersion:    3,
				ProcessVersionTag: "stable",
			}},
			DuplicateCandidateProcessDefinitionKeys: typex.Keys{"2251799813685255"},
			CandidateProcessDefinitionCount:         1,
			LatestOnly:                              true,
			Notices: []ops.AllProcessDefinitionsPurgeNotice{
				{Code: "latest_only_scope", Severity: "info", Message: "candidate discovery was narrowed to latest matching process definitions"},
				{Code: "duplicate_candidate_process_definitions", Severity: "info", Message: "duplicate candidate process-definition keys detected"},
			},
		},
		DeletePlan: ops.AllProcessDefinitionsPurgeDeletePlan{Status: ops.WorkflowStepStatusSkipped},
		Deletion:   ops.AllProcessDefinitionsPurgeDeletionResult{Status: ops.WorkflowStepStatusSkipped},
		Outcome:    ops.AllProcessDefinitionsPurgeOutcomePlanned,
	}
	result.Discovery.DiscoveryScopeStatus = ops.DiscoveryScopeStatus{
		Complete:         true,
		BatchSize:        25,
		Pages:            2,
		CandidatesSeen:   2,
		CandidatesFrozen: 1,
	}
	result.Report.Discovery = result.Discovery
	return result
}

// marshalOpsPurgeAllProcessDefinitionsArgsForEnv preserves argument boundaries for subprocess helpers.
func marshalOpsPurgeAllProcessDefinitionsArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

// resetOpsPurgeAllProcessDefinitionsFlagState restores all-process-definitions purge globals between command tests.
func resetOpsPurgeAllProcessDefinitionsFlagState() {
	flagOpsPurgeAllPDKey = ""
	flagOpsPurgeAllPDBpmnProcessID = ""
	flagOpsPurgeAllPDProcessVersion = 0
	flagOpsPurgeAllPDProcessVersionTag = ""
	flagOpsPurgeAllPDLatest = false
	flagOpsPurgeAllPDBatchSize = consts.MaxPISearchSize
	flagOpsPurgeAllPDLimit = 0
	flagOpsPurgeAllPDReportFile = ""
	flagOpsPurgeAllPDReportFormat = ""
	flagDryRun = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagNoWait = false
	flagForce = false
	flagCmdAutoConfirm = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagVerbose = false
}

const (
	opsAllProcessDefinitionsPurgePDKeyA     = "2251799813685255"
	opsAllProcessDefinitionsPurgePDKeyB     = "2251799813685256"
	opsAllProcessDefinitionsPurgeRootA      = "2251799813686201"
	opsAllProcessDefinitionsPurgeRootShared = "2251799813686202"
	opsAllProcessDefinitionsPurgeRootB      = "2251799813686203"
)

// opsPurgeAllProcessDefinitionsNestedState coordinates a fake force-cleanup
// backend with command activity assertions.
type opsPurgeAllProcessDefinitionsNestedState struct {
	cancelStarted     chan struct{}
	cancelRelease     chan struct{}
	drainPolled       chan struct{}
	drainRelease      chan struct{}
	releaseCancel     sync.Once
	releaseDrain      sync.Once
	cancelOnce        sync.Once
	drainOnce         sync.Once
	cancelled         testx.SafeSlice[string]
	deletedPI         testx.SafeSlice[string]
	deletedPD         testx.SafeSlice[string]
	cancelCount       atomic.Int64
	failCancelKey     string
	failHistoryKey    string
	failDefinitionKey string
}

// newOpsPurgeAllProcessDefinitionsNestedState returns synchronization gates
// used to inspect first-root cancellation and drain activity before release.
func newOpsPurgeAllProcessDefinitionsNestedState() *opsPurgeAllProcessDefinitionsNestedState {
	return &opsPurgeAllProcessDefinitionsNestedState{
		cancelStarted: make(chan struct{}),
		cancelRelease: make(chan struct{}),
		drainPolled:   make(chan struct{}),
		drainRelease:  make(chan struct{}),
	}
}

// newOpsPurgeAllProcessDefinitionsActivityCommandContext installs real remote
// services while leaving command activity routed to the test sink.
func newOpsPurgeAllProcessDefinitionsActivityCommandContext(t *testing.T, baseURL string, sink *activitysink.Sink, stderr io.Writer) context.Context {
	t.Helper()

	cfg := config.New()
	cfg.App.CamundaVersion = toolx.V89
	cfg.Auth.Mode = config.ModeNone
	cfg.APIs.Camunda.BaseURL = baseURL
	require.NoError(t, cfg.Normalize())
	require.NoError(t, cfg.Validate())
	ctx := cfg.ToContextWithLogWriter(context.Background(), stderr)
	ctx = logging.ToActivityContext(ctx, sink)
	log, err := logging.FromContext(ctx)
	require.NoError(t, err)
	ctx, err = installRemoteCommandServices(ctx, cfg, log)
	require.NoError(t, err)
	return logging.ToActivityContext(ctx, sink)
}

// requireOpsPurgeAllProcessDefinitionsSignal waits for a synchronized fake
// backend checkpoint so assertions do not race command execution.
func requireOpsPurgeAllProcessDefinitionsSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()

	require.Eventually(t, func() bool {
		select {
		case <-ch:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)
}

// requireOpsPurgeAllProcessDefinitionsActivityContains asserts the command
// workflow activity eventually contains a specific stage snapshot.
func requireOpsPurgeAllProcessDefinitionsActivityContains(t *testing.T, sink *activitysink.Sink, want string) {
	t.Helper()

	require.Eventually(t, func() bool {
		for _, update := range sink.PriorityUpdates() {
			if update.Message == want && update.Importance == logging.ActivityImportanceWorkflow {
				return true
			}
		}
		return false
	}, 5*time.Second, 10*time.Millisecond, "activity updates: %v", sink.PriorityUpdates())
}

func newOpsPurgeAllProcessDefinitionsServer(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string], activeCount int64) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			payload := string(body)
			requests.Append(r.Method + " " + r.URL.Path + " " + payload)
			_, _ = w.Write([]byte(opsAllProcessDefinitionsPurgeSearchResponse(t, payload)))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-definitions/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-definitions/")
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(opsAllProcessDefinitionsPurgeDefinitionJSON(key, "invoice", 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			payload := string(body)
			requests.Append(r.Method + " " + r.URL.Path + " " + payload)
			total := int64(0)
			if strings.Contains(payload, "ACTIVE") {
				total = activeCount
			}
			_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[],"page":{"totalItems":%d,"hasMoreTotalItems":false}}`, total)))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/resources/") && strings.HasSuffix(r.URL.Path, "/deletion"):
			if deleted != nil {
				deleted.Append(r.URL.Path)
			}
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/resources/"), "/deletion")
			_, _ = w.Write([]byte(`{"resourceKey":"` + key + `","batchOperation":{"batchOperationKey":"batch-` + key + `","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsPurgeAllProcessDefinitionsEmptyServer returns an empty APD discovery
// response and fails the test if mutation or cleanup requests are attempted.
func newOpsPurgeAllProcessDefinitionsEmptyServer(t *testing.T, requests *testx.SafeSlice[string], deleted *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/deletion"):
			if deleted != nil {
				deleted.Append(r.URL.Path)
			}
			t.Fatalf("unexpected mutation request: %s %s", r.Method, r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// newOpsPurgeAllProcessDefinitionsNestedServer handles the real force-cleanup
// command path with shared roots and blocking cleanup checkpoints.
func newOpsPurgeAllProcessDefinitionsNestedServer(t *testing.T, state *opsPurgeAllProcessDefinitionsNestedState, requests *testx.SafeSlice[string]) *httptest.Server {
	t.Helper()

	return newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			payload := string(body)
			requests.Append(r.Method + " " + r.URL.Path + " " + payload)
			_, _ = w.Write([]byte(`{"items":[` +
				opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 2) + `,` +
				opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyB, "payment", 1) +
				`],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-definitions/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-definitions/")
			requests.Append(r.Method + " " + r.URL.Path)
			if opsPurgeAllProcessDefinitionsPathSeen(state.deletedPD.Snapshot(), "/v2/resources/"+key+"/deletion") {
				http.NotFound(w, r)
				return
			}
			id := "invoice"
			if key == opsAllProcessDefinitionsPurgePDKeyB {
				id = "payment"
			}
			_, _ = w.Write([]byte(opsAllProcessDefinitionsPurgeDefinitionJSON(key, id, 1)))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			payload := string(body)
			requests.Append(r.Method + " " + r.URL.Path + " " + payload)
			_, _ = w.Write([]byte(opsAllProcessDefinitionsPurgeNestedProcessInstanceSearchResponse(state, payload)))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			requests.Append(r.Method + " " + r.URL.Path)
			if opsPurgeAllProcessDefinitionsPathSeen(state.deletedPI.Snapshot(), "/v2/process-instances/"+key+"/deletion") {
				http.NotFound(w, r)
				return
			}
			instanceState := "ACTIVE"
			if opsPurgeAllProcessDefinitionsPathSeen(state.cancelled.Snapshot(), "/v2/process-instances/"+key+"/cancellation") {
				instanceState = "CANCELED"
			}
			_, _ = w.Write([]byte(opsAllProcessDefinitionsPurgeProcessInstanceJSON(key, key, opsAllProcessDefinitionsPurgePDKeyA, instanceState)))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/process-instances/") && strings.HasSuffix(r.URL.Path, "/cancellation"):
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/process-instances/"), "/cancellation")
			state.cancelOnce.Do(func() { close(state.cancelStarted) })
			state.cancelled.Append(r.URL.Path)
			state.cancelCount.Add(1)
			requests.Append(r.Method + " " + r.URL.Path)
			if key == state.failCancelKey {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"cancel rejected"}`))
				return
			}
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/process-instances/") && strings.HasSuffix(r.URL.Path, "/deletion"):
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/process-instances/"), "/deletion")
			state.deletedPI.Append(r.URL.Path)
			requests.Append(r.Method + " " + r.URL.Path)
			if key == state.failHistoryKey {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"history delete rejected"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/v2/resources/") && strings.HasSuffix(r.URL.Path, "/deletion"):
			state.deletedPD.Append(r.URL.Path)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(r.Method + " " + r.URL.Path + " " + string(body))
			key := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v2/resources/"), "/deletion")
			if key == state.failDefinitionKey {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"definition delete rejected"}`))
				return
			}
			_, _ = w.Write([]byte(`{"resourceKey":"` + key + `","batchOperation":{"batchOperationKey":"batch-` + key + `","batchOperationType":"DELETE_PROCESS_DEFINITION"}}`))
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/batch-operations/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/batch-operations/")
			requests.Append(r.Method + " " + r.URL.Path)
			_, _ = w.Write([]byte(`{"batchOperationKey":"` + key + `","batchOperationType":"DELETE_PROCESS_DEFINITION","state":"COMPLETED","operationsTotalCount":1,"operationsCompletedCount":1,"operationsFailedCount":0}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

// opsAllProcessDefinitionsPurgeNestedProcessInstanceSearchResponse returns
// active planning rows, descendant rows, or drain/stat counts for the test API.
func opsAllProcessDefinitionsPurgeNestedProcessInstanceSearchResponse(state *opsPurgeAllProcessDefinitionsNestedState, payload string) string {
	if strings.Contains(payload, `"parentProcessInstanceKey"`) {
		return `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
	}
	if strings.Contains(payload, `"processDefinitionKey"`) && strings.Contains(payload, "ACTIVE") && strings.Contains(payload, `"processDefinitionName"`) {
		if strings.Contains(payload, opsAllProcessDefinitionsPurgePDKeyA) {
			return `{"items":[` +
				opsAllProcessDefinitionsPurgeProcessInstanceJSON(opsAllProcessDefinitionsPurgeRootA, opsAllProcessDefinitionsPurgeRootA, opsAllProcessDefinitionsPurgePDKeyA, "ACTIVE") + `,` +
				opsAllProcessDefinitionsPurgeProcessInstanceJSON(opsAllProcessDefinitionsPurgeRootShared, opsAllProcessDefinitionsPurgeRootShared, opsAllProcessDefinitionsPurgePDKeyA, "ACTIVE") +
				`],"page":{"totalItems":2,"hasMoreTotalItems":false}}`
		}
		return `{"items":[` +
			opsAllProcessDefinitionsPurgeProcessInstanceJSON(opsAllProcessDefinitionsPurgeRootShared, opsAllProcessDefinitionsPurgeRootShared, opsAllProcessDefinitionsPurgePDKeyB, "ACTIVE") + `,` +
			opsAllProcessDefinitionsPurgeProcessInstanceJSON(opsAllProcessDefinitionsPurgeRootB, opsAllProcessDefinitionsPurgeRootB, opsAllProcessDefinitionsPurgePDKeyB, "ACTIVE") +
			`],"page":{"totalItems":2,"hasMoreTotalItems":false}}`
	}
	if strings.Contains(payload, `"processDefinitionKey"`) && strings.Contains(payload, "ACTIVE") {
		if state.cancelCount.Load() >= 3 {
			state.drainOnce.Do(func() { close(state.drainPolled) })
			<-state.drainRelease
			return `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
		}
		return `{"items":[],"page":{"totalItems":2,"hasMoreTotalItems":false}}`
	}
	return `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
}

// opsAllProcessDefinitionsPurgeProcessInstanceJSON renders the generated v8.9
// process-instance result shape used by real service converters.
func opsAllProcessDefinitionsPurgeProcessInstanceJSON(key string, root string, pdKey string, state string) string {
	return fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"invoice","processDefinitionKey":"%s","processDefinitionVersion":1,"state":"%s","tenantId":"tenant","rootProcessInstanceKey":"%s"}`, key, pdKey, state, root)
}

// opsAllProcessDefinitionsPurgeSearchResponse returns one or two APD discovery pages based on the request page size.
func opsAllProcessDefinitionsPurgeSearchResponse(t *testing.T, payload string) string {
	t.Helper()

	limit, from, after := opsAllProcessDefinitionsPurgeSearchPage(t, payload)
	if limit == 1 {
		if after == "" && from == 0 {
			return `{"items":[` +
				opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 2) +
				`],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"pd-page-2"}}`
		}
		return `{"items":[` +
			opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyB, "payment", 1) +
			`],"page":{"totalItems":2,"hasMoreTotalItems":false}}`
	}
	return `{"items":[` +
		opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 2) + `,` +
		opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyA, "invoice", 2) + `,` +
		opsAllProcessDefinitionsPurgeDefinitionJSON(opsAllProcessDefinitionsPurgePDKeyB, "payment", 1) +
		`],"page":{"totalItems":3,"hasMoreTotalItems":false}}`
}

// opsAllProcessDefinitionsPurgeSearchPage extracts the generated v8.9 page request fields used by command tests.
func opsAllProcessDefinitionsPurgeSearchPage(t *testing.T, payload string) (limit int32, from int32, after string) {
	t.Helper()

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(payload), &body))
	page, _ := body["page"].(map[string]any)
	if raw, ok := page["limit"].(float64); ok {
		limit = int32(raw)
	}
	if raw, ok := page["from"].(float64); ok {
		from = int32(raw)
	}
	if raw, ok := page["after"].(string); ok {
		after = raw
	}
	return limit, from, after
}

func opsAllProcessDefinitionsPurgeDefinitionJSON(key string, id string, version int32) string {
	return fmt.Sprintf(`{"processDefinitionKey":"%s","processDefinitionId":"%s","name":"%s","version":%d,"tenantId":"tenant","versionTag":"stable"}`, key, id, id, version)
}

func countOpsPurgeAllProcessDefinitionsRequests(items []string, prefix string) int {
	count := 0
	for _, item := range items {
		if strings.HasPrefix(item, prefix) {
			count++
		}
	}
	return count
}

// opsPurgeAllProcessDefinitionsPathSeen reports whether a fake backend
// mutation path has already been captured.
func opsPurgeAllProcessDefinitionsPathSeen(items []string, path string) bool {
	for _, item := range items {
		if item == path {
			return true
		}
	}
	return false
}

// requireOpsPurgeAllProcessDefinitionsOutcomeCount asserts diagnostic
// completion lines are neither missing nor duplicated.
func requireOpsPurgeAllProcessDefinitionsOutcomeCount(t *testing.T, output string, want string, count int) {
	t.Helper()

	require.Equal(t, count, strings.Count(output, want), output)
}

// assertOpsPurgeAllProcessDefinitionsNoStageProgress keeps machine, dry-run,
// declined, and empty-selection contracts free of nested stage progress text.
func assertOpsPurgeAllProcessDefinitionsNoStageProgress(t *testing.T, output string) {
	t.Helper()

	for _, disallowed := range []string{
		"cancelling process-instance root trees",
		"waiting for active process instances to drain",
		"deleting process-instance histories",
		"deleting process definitions",
		"affected scope:",
		"stage progress:",
	} {
		require.NotContains(t, output, disallowed)
	}
}
