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
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

type deleteDryRunPreviewFixture struct {
	RequestedKeys      typex.Keys
	ResolvedRoots      typex.Keys
	AffectedFamilyKeys typex.Keys
	TraversalOutcome   process.TraversalOutcome
	Warning            string
	MissingAncestors   []process.MissingAncestor
}

func newDeleteDryRunPreviewFixture() deleteDryRunPreviewFixture {
	return deleteDryRunPreviewFixture{
		RequestedKeys:      typex.Keys{"2251799813711967"},
		ResolvedRoots:      typex.Keys{"2251799813711900"},
		AffectedFamilyKeys: typex.Keys{"2251799813711900", "2251799813711967"},
		TraversalOutcome:   process.TraversalOutcomePartial,
		Warning:            "one or more parent process instances were not found",
		MissingAncestors:   []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
	}
}

func requireDeleteDryRunPreviewPayload(t *testing.T, payload map[string]any, want deleteDryRunPreviewFixture) {
	t.Helper()

	require.Equal(t, "delete", payload["operation"])
	requireDryRunPreviewStringSlice(t, payload, "requestedKeys", want.RequestedKeys)
	requireDryRunPreviewStringSlice(t, payload, "resolvedRoots", want.ResolvedRoots)
	requireDryRunPreviewStringSlice(t, payload, "affectedFamilyKeys", want.AffectedFamilyKeys)
	require.Equal(t, float64(len(want.RequestedKeys)), payload["requestedCount"])
	require.Equal(t, float64(len(want.ResolvedRoots)), payload["resolvedRootCount"])
	require.Equal(t, float64(len(want.AffectedFamilyKeys)), payload["affectedCount"])
	require.Equal(t, string(want.TraversalOutcome), payload["traversalOutcome"])
	require.Equal(t, want.TraversalOutcome == process.TraversalOutcomeComplete, payload["scopeComplete"])
	require.Equal(t, want.Warning, payload["warning"])
	requireDryRunPreviewMissingAncestors(t, payload, want.MissingAncestors)
	require.Equal(t, false, payload["mutationSubmitted"])
}

func TestDeleteProcessInstanceDryRunPreviewPayloadMapping(t *testing.T) {
	want := newDeleteDryRunPreviewFixture()
	preview := newProcessInstanceDryRunPreview("delete", want.RequestedKeys, process.DryRunPIKeyExpansion{
		Roots:            want.ResolvedRoots,
		Collected:        want.AffectedFamilyKeys,
		MissingAncestors: want.MissingAncestors,
		Warning:          want.Warning,
		Outcome:          want.TraversalOutcome,
	})

	var payload map[string]any
	b, err := json.Marshal(preview)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(b, &payload))

	requireDeleteDryRunPreviewPayload(t, payload, want)
}

func TestDeleteProcessInstanceDryRun_KeyedChildEscalatesToRootWithoutMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry run")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"child-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-1"},
				Collected: typex.Keys{"root-1", "child-1"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-1"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-1"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
	require.Equal(t, typex.Keys{"root-1", "child-1"}, typex.Keys(got.DryRunPreview.AffectedFamilyKeys))
	require.Contains(t, buf.String(), "dry run: delete process-instance")
	require.Contains(t, buf.String(), "requested keys: child-1")
	require.Contains(t, buf.String(), "resolved root keys: root-1")
	require.Contains(t, buf.String(), "affected family keys: root-1, child-1")
	require.Contains(t, buf.String(), "no mutation submitted: delete was not submitted")
}

func TestDeleteProcessInstanceDryRun_KeyedRootReportsFullFamilyWithoutMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry run")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"root-1"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-1"},
				Collected: typex.Keys{"root-1", "child-1", "child-2"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"root-1"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 3, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, typex.Keys{"root-1"}, typex.Keys(got.DryRunPreview.ResolvedRoots))
	require.Equal(t, typex.Keys{"root-1", "child-1", "child-2"}, typex.Keys(got.DryRunPreview.AffectedFamilyKeys))
	require.Contains(t, buf.String(), "affected process instances: 3")
	require.Contains(t, buf.String(), "scope: complete")
	require.Contains(t, buf.String(), "affected family keys: root-1, child-1, child-2")
	require.Contains(t, buf.String(), "no mutation submitted: delete was not submitted")
}

func TestDeleteProcessInstanceDryRun_PartialOrphanParentRendersWarningAndMissingAncestor(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry-run orphan preview")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"child-orphan"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"root-partial"},
				Collected:        typex.Keys{"root-partial", "child-orphan"},
				MissingAncestors: []process.MissingAncestor{{Key: "missing-parent", StartKey: "child-orphan"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"child-orphan"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Empty(t, got.Reports)
	require.NotNil(t, got.DryRunPreview)
	require.Equal(t, process.TraversalOutcomePartial, got.DryRunPreview.TraversalOutcome)
	require.False(t, got.DryRunPreview.ScopeComplete)
	require.Equal(t, []processInstanceDryRunMissingAncestor{{Key: "missing-parent", StartKey: "child-orphan"}}, got.DryRunPreview.MissingAncestors)
	require.Contains(t, buf.String(), "scope: partial")
	require.Contains(t, buf.String(), "warning: one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: missing-parent")
	require.Contains(t, buf.String(), "no mutation submitted: delete was not submitted")
}

func TestDeleteProcessInstanceDryRun_UnresolvedOrphanFailsWithoutMutation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during unresolved delete dry run")
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"unresolved-child"}, keys)
			return process.DryRunPIKeyExpansion{}, fmt.Errorf("%w: no process instances resolved during dependency expansion", services.ErrOrphanedInstance)
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"unresolved-child"}, true)

	require.Error(t, err)
	require.ErrorContains(t, err, "delete validation")
	require.ErrorContains(t, err, "no process instances resolved during dependency expansion")
	require.Equal(t, processInstancePageActionResult{}, got)
	require.Empty(t, buf.String())
}

func TestDeleteProcessInstanceDryRun_SearchPagesAggregateStructuredOutput(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagViewAsJson = true
	flagGetPISize = 2

	cmd := &cobra.Command{Use: "process-instance"}
	setContractSupport(cmd, ContractSupportFull)
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "2"))
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry-run search")
		return nil
	}

	var planned []typex.Keys
	var searchedFrom []int32
	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			searchedFrom = append(searchedFrom, req.From)
			require.EqualValues(t, 2, req.Size)
			switch req.From {
			case 0:
				return process.ProcessInstancePage{
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateHasMore,
					Items: []process.ProcessInstance{
						{Key: "401", State: process.StateCompleted},
						{Key: "402", State: process.StateCompleted},
					},
				}, nil
			case 2:
				return process.ProcessInstancePage{
					Request:       req,
					OverflowState: process.ProcessInstanceOverflowStateNoMore,
					Items: []process.ProcessInstance{
						{Key: "403", State: process.StateCompleted},
					},
				}, nil
			default:
				t.Fatalf("unexpected search page offset %d", req.From)
				return process.ProcessInstancePage{}, nil
			}
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			planned = append(planned, append(typex.Keys(nil), keys...))
			switch strings.Join(keys, ",") {
			case "401,402":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"delete-root-a"},
					Collected: typex.Keys{"delete-root-a", "401", "402"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			case "403":
				return process.DryRunPIKeyExpansion{
					Roots:     typex.Keys{"delete-root-b"},
					Collected: typex.Keys{"delete-root-b"},
					Outcome:   process.TraversalOutcomeComplete,
				}, nil
			default:
				t.Fatalf("unexpected dry-run plan keys %v", keys)
				return process.DryRunPIKeyExpansion{}, nil
			}
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	results, err := processPISearchPagesWithAction(cmd, cli, nil, process.ProcessInstanceFilter{}, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error) {
		keys := make(typex.Keys, 0, len(page.Items))
		for _, pi := range page.Items {
			keys = append(keys, pi.Key)
		}
		return deleteProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, false)
	})
	require.NoError(t, err)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 2)
	require.Equal(t, []int32{0, 2}, searchedFrom)
	require.Equal(t, []typex.Keys{{"401", "402"}, {"403"}}, planned)

	require.NoError(t, renderProcessInstanceDryRunSummary(cmd, newProcessInstanceDryRunSummary("delete", results.DryRunPreviews)))
	var envelope map[string]any
	require.NoError(t, json.Unmarshal(buf.Bytes(), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	payload, ok := envelope["payload"].(map[string]any)
	require.True(t, ok)
	previews := requireDryRunSummaryPayload(t, payload, "delete", 3, 2, 4, 2)

	firstPreview, ok := previews[0].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, firstPreview, "requestedKeys", typex.Keys{"401", "402"})
	requireDryRunPreviewStringSlice(t, firstPreview, "resolvedRoots", typex.Keys{"delete-root-a"})
	requireDryRunPreviewStringSlice(t, firstPreview, "affectedFamilyKeys", typex.Keys{"delete-root-a", "401", "402"})

	secondPreview, ok := previews[1].(map[string]any)
	require.True(t, ok)
	requireDryRunPreviewStringSlice(t, secondPreview, "requestedKeys", typex.Keys{"403"})
	requireDryRunPreviewStringSlice(t, secondPreview, "resolvedRoots", typex.Keys{"delete-root-b"})
	requireDryRunPreviewStringSlice(t, secondPreview, "affectedFamilyKeys", typex.Keys{"delete-root-b"})
}

func TestDeleteProcessInstanceDryRun_SearchBatchSizeLimitUsesLimitedPage(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagDryRun = true
	flagGetPISize = 4
	flagGetPILimit = 2

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "4"))
	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(bool, string) error {
		t.Fatal("unexpected confirmation prompt during delete dry-run limited search")
		return nil
	}

	var planned typex.Keys
	var searchRequests []process.ProcessInstancePageRequest
	cli := stubProcessAPI{
		searchProcessInstancesPage: func(_ context.Context, _ process.ProcessInstanceFilter, req process.ProcessInstancePageRequest, _ ...options.FacadeOption) (process.ProcessInstancePage, error) {
			searchRequests = append(searchRequests, req)
			require.EqualValues(t, 4, req.Size)
			return process.ProcessInstancePage{
				Request:       req,
				OverflowState: process.ProcessInstanceOverflowStateHasMore,
				Items: []process.ProcessInstance{
					{Key: "501", State: process.StateCompleted},
					{Key: "502", State: process.StateCompleted},
					{Key: "503", State: process.StateCompleted},
					{Key: "504", State: process.StateCompleted},
				},
			}, nil
		},
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			planned = append(typex.Keys(nil), keys...)
			return process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"delete-root-limit"},
				Collected: typex.Keys{"delete-root-limit", "501", "502"},
				Outcome:   process.TraversalOutcomeComplete,
			}, nil
		},
		deleteProcessInstances: dryRunDeleteMutationGuard(t),
	}

	results, err := processPISearchPagesWithAction(cmd, cli, nil, process.ProcessInstanceFilter{}, func(page process.ProcessInstancePage, firstPage bool) (processInstancePageActionResult, error) {
		keys := make(typex.Keys, 0, len(page.Items))
		for _, pi := range page.Items {
			keys = append(keys, pi.Key)
		}
		return deleteProcessInstancesWithPlanAndRender(cmd, cli, keys, firstPage, false)
	})
	require.NoError(t, err)
	require.Len(t, searchRequests, 1)
	require.EqualValues(t, 0, searchRequests[0].From)
	require.Equal(t, typex.Keys{"501", "502"}, planned)
	require.Empty(t, results.Reports)
	require.Len(t, results.DryRunPreviews, 1)
	require.Equal(t, 2, results.DryRunPreviews[0].RequestedCount)
}

func TestDeleteCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "24s")

	cfg := resolveCommandConfigForTest(t, deleteCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "46s"))
	})

	require.Equal(t, 46*time.Second, cfg.App.Backoff.Timeout)
}

func TestDeleteHelp_DocumentsDestructiveConfirmationPaths(t *testing.T) {
	output := assertCommandHelpOutput(t, []string{"delete"}, []string{
		"Delete process instances or process definitions",
		"--auto-confirm",
		"which follow-up",
		"./c8volt delete pd --bpmn-process-id C88_SimpleUserTask_Process --latest --auto-confirm",
	}, nil)
	require.Contains(t, output, "process-instance")
	require.Contains(t, output, "process-definition")

	output = assertCommandHelpOutput(t, []string{"delete", "process-instance"}, []string{
		"validates the affected tree",
		"Use --force when active instances should be cancelled",
		"Use --auto-confirm for unattended destructive runs",
		"`expect pi --state absent`",
		"number of process instances to process per page",
		"maximum number of matching process instances to process across all pages",
		"./c8volt delete pi --state completed --batch-size 250 --limit 25",
		"./c8volt delete pi --state completed --batch-size 200 --auto-confirm --no-wait",
	}, []string{"--count"})
	require.Contains(t, output, "--force")
	require.Contains(t, output, "--batch-size int32")
	require.Contains(t, output, "--limit int32")

	output = assertCommandHelpOutput(t, []string{"delete", "process-definition"}, []string{
		"Delete process definition resources from Zeebe",
		"Without --allow-inconsistent",
		"Use --auto-confirm for unattended destructive runs",
		"`get pd`",
		"./c8volt delete pd --bpmn-process-id C88_SimpleUserTask_Process --latest --allow-inconsistent --auto-confirm --no-wait",
	}, nil)
	require.Contains(t, output, "--allow-inconsistent")
}

// Verifies search-mode deletion builds the expected date-filtered search request and no-ops cleanly on empty matches.
func TestDeleteProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceSearchScaffoldHelper", cfgPath)

	filter := decodeCapturedPISearchFilter(t, requests)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies reversed date ranges are rejected when the after-bound is later than the before-bound.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateFilter(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateFilterHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid range for --end-date-after and --end-date-before: "2026-02-01" is later than "2026-01-31"`)
}

// Verifies invalid date literals for date flags are rejected with a clear YYYY-MM-DD validation error.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateValue(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsInvalidDateValueHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
}

// Verifies date filters cannot be combined with direct key lookup mode.
func TestDeleteProcessInstanceCommand_RejectsKeyAndDateFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsKeyAndDateFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

// Verifies relative-day filters cannot be combined with direct key lookup mode.
func TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFilters(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper", cfgPath)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
}

func TestDeleteProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "removed count flag is rejected",
			helper: "TestDeleteProcessInstanceCommand_RejectsRemovedCountFlagHelper",
			want:   "unknown flag: --count",
		},
		{
			name:   "non-positive limit is rejected",
			helper: "TestDeleteProcessInstanceCommand_RejectsInvalidLimitHelper",
			want:   "--limit must be positive integer",
		},
		{
			name:   "limit cannot be combined with key",
			helper: "TestDeleteProcessInstanceCommand_RejectsLimitWithKeyHelper",
			want:   "--limit cannot be combined with --key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeDeleteProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, tt.want)
		})
	}
}

// Verifies process-instance date filters are rejected for Camunda 8.7 where the capability is unsupported.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies relative-day process-instance filters are also rejected for Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance date filters require Camunda 8.8")
}

// Verifies date-filtered search selection deletes matched instances and preserves descendant lookup behavior.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatches(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))

			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			if parentKey == "2251799813711967" {
				_, _ = w.Write([]byte(`{"items":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-01-03T18:00:00Z","endDate":"2026-01-12T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-01-03T18:00:00Z","endDate":"2026-01-12T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/2251799813711967":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(requests), 2)
	require.Equal(t, []string{"/v1/process-instances/2251799813711967"}, deleted.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-01-31T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[len(requests)-1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies relative-day search selection derives canonical end-date bounds before deleting matches.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatches(t *testing.T) {
	var requests []string
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			var searchBody map[string]any
			require.NoError(t, json.Unmarshal(body, &searchBody))

			filter, _ := searchBody["filter"].(map[string]any)
			parentKey, _ := filter["parentProcessInstanceKey"].(string)

			w.Header().Set("Content-Type", "application/json")
			if parentKey == "2251799813711967" {
				_, _ = w.Write([]byte(`{"items":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-02-10T18:00:00Z","endDate":"2026-03-15T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","startDate":"2026-02-10T18:00:00Z","endDate":"2026-03-15T08:30:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/2251799813711967":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(requests), 2)
	require.Equal(t, []string{"/v1/process-instances/2251799813711967"}, deleted.Snapshot())
	filter := decodeCapturedPISearchFilter(t, requests[:1])

	require.Equal(t, "COMPLETED", filter["state"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-09T00:00:00Z")
	requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-04-03T23:59:59.999999999Z")
	requireCapturedPISearchDateExists(t, filter, "endDate")

	descendantSearch := decodeCapturedPISearchRequest(t, requests[len(requests)-1])
	descFilter := descendantSearch["filter"].(map[string]any)
	require.Equal(t, "2251799813711967", descFilter["parentProcessInstanceKey"])
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

func TestDeleteProcessInstanceCommand_V89DeletesViaCamundaProcessInstanceAPI(t *testing.T) {
	var requests []string
	var deleted []string

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
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
			case parentKey == "2251799813711967":
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected search body: %s", string(body))
			}
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/2251799813711967":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"2251799813711967","processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/2251799813711967/deletion":
			deleted = append(deleted, r.URL.Path)
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--json",
		"delete", "process-instance",
		"--key", "2251799813711967",
		"--no-wait",
	)

	require.GreaterOrEqual(t, len(requests), 1)
	require.Equal(t, []string{"/v2/process-instances/2251799813711967/deletion"}, deleted)
	require.Contains(t, requests[len(requests)-1], `"parentProcessInstanceKey":"2251799813711967"`)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Equal(t, "delete process-instance", got["command"])
	require.Contains(t, stderr, "INFO")
}

func TestDeleteProcessInstancesWithPlan_PrintsOrphanWarningForKeyedPreflight(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var prompt string
	confirmCmdOrAbortFn = func(_ bool, got string) error {
		prompt = got
		return nil
	}

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"2251799813711967"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"2251799813711900"},
				Collected:        typex.Keys{"2251799813711900", "2251799813711967"},
				MissingAncestors: []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			require.Equal(t, typex.Keys{"2251799813711900"}, keys)
			require.Zero(t, wantedWorkers)
			require.Equal(t, 2, options.ApplyFacadeOptions(opts).AffectedProcessInstanceCount)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "2251799813711900", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstancesWithPlan(cmd, cli, typex.Keys{"2251799813711967"}, true)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.Contains(t, prompt, "requested to delete 1 process instance(s)")
	require.Contains(t, prompt, "a total of 2 instance(s) with 1 root instance(s) will be deleted")
	require.Contains(t, buf.String(), "warning: one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: 1 (use --verbose to list keys)")
	require.NotContains(t, buf.String(), "missing ancestor keys: 2251799813711999")
}

func TestDeleteProcessInstancePage_PrintsOrphanWarningForPagedPreflight(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true

	cmd := &cobra.Command{}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	cli := stubProcessAPI{
		dryRunCancelOrDeletePlan: func(_ context.Context, keys typex.Keys, _ ...options.FacadeOption) (process.DryRunPIKeyExpansion, error) {
			require.Equal(t, typex.Keys{"2251799813711967"}, keys)
			return process.DryRunPIKeyExpansion{
				Roots:            typex.Keys{"2251799813711900"},
				Collected:        typex.Keys{"2251799813711900", "2251799813711967"},
				MissingAncestors: []process.MissingAncestor{{Key: "2251799813711999", StartKey: "2251799813711967"}},
				Warning:          "one or more parent process instances were not found",
				Outcome:          process.TraversalOutcomePartial,
			}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, wantedWorkers int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			require.Equal(t, typex.Keys{"2251799813711900"}, keys)
			require.Zero(t, wantedWorkers)
			require.Equal(t, 2, options.ApplyFacadeOptions(opts).AffectedProcessInstanceCount)
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "2251799813711900", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstancePage(cmd, cli, typex.Keys{"2251799813711967"}, false)

	require.NoError(t, err)
	require.Equal(t, processInstancePageImpact{Requested: 1, Affected: 2, Roots: 1}, got.Impact)
	require.Len(t, got.Reports, 1)
	require.Contains(t, buf.String(), "warning: one or more parent process instances were not found")
	require.Contains(t, buf.String(), "missing ancestor keys: 2251799813711999")
}

// Verifies delete no-ops successfully when a date-filtered search returns no process instances.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatches(t *testing.T) {
	var requests []string

	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies a relative-day-only filter is sufficient to trigger search mode.
func TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficient(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := executeDeleteProcessInstanceSuccessHelper(t, "TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper", cfgPath)

	require.NoError(t, err)
	require.Len(t, requests, 1)
	require.NotContains(t, output, "either at least one --key is required, or sufficient filtering options")
	require.Contains(t, output, "found: 0")
	require.NotContains(t, output, "no process instance keys provided or found to delete")
}

// Verifies paged delete search prompts between pages and continues when confirmations are accepted.
func TestDeleteProcessInstanceCommand_SearchPagingPromptFlow(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]
	searchPage := 0
	var searchMu sync.Mutex

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			searchMu.Lock()
			defer searchMu.Unlock()
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"401","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"402","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"403","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/401" || r.URL.Path == "/v1/process-instances/402" || r.URL.Path == "/v1/process-instances/403"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		prompts = append(prompts, prompt)
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 0, pages[0]["from"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.ElementsMatch(t, []string{
		"/v1/process-instances/401",
		"/v1/process-instances/402",
		"/v1/process-instances/403",
	}, deleted.Snapshot())
	require.Len(t, prompts, 2)
	require.Contains(t, prompts[0], "You are about to delete 2 process instance(s)")
	require.Contains(t, prompts[1], "Processed 2 process instance(s) on this page (2 requested so far, 2 including dependencies). More matching process instances remain. Continue?")
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	require.NotContains(t, output, "next step: auto-continue")
}

// Verifies v8.7 paged delete search fails once keyed tenant-safe preflight reaches the unsupported direct-lookup seam.
func TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotals(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]
	searchPage := 0
	var searchMu sync.Mutex

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if parentKey, ok := filter["parentKey"]; ok && parentKey != nil {
					parent := int64(parentKey.(float64))
					if parent != 901 && parent != 902 {
						w.Header().Set("Content-Type", "application/json")
						_, _ = w.Write([]byte(`{"items":[]}`))
						return
					}
					childKey := "1901"
					if parent == 902 {
						childKey = "1902"
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[{"key":` + childKey + `,"parentKey":` + fmt.Sprintf("%d", parent) + `,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			searchMu.Lock()
			defer searchMu.Unlock()
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"key":901,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":902,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"key":901,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":902,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"},{"key":903,"bpmnProcessId":"demo","processVersion":3,"state":"COMPLETED","startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","tenantId":"tenant"}],"total":3}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/901" || r.URL.Path == "/v1/process-instances/902" || r.URL.Path == "/v1/process-instances/903" || r.URL.Path == "/v1/process-instances/1901" || r.URL.Path == "/v1/process-instances/1902"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")
	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper", cfgPath)

	sizes := decodeCapturedTopLevelPISearchSizes(t, requests.Snapshot())
	require.Equal(t, exitcode.Error, code)
	require.Equal(t, []float64{2}, sizes)
	require.Empty(t, deleted.Snapshot())
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "process-instance direct lookup by key is not tenant-safe in Camunda 8.7")
}

// Verifies paged delete search auto-continues without continuation prompts when --auto-confirm is set.
func TestDeleteProcessInstanceCommand_SearchPagingAutoConfirmFlow(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"501","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"502","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"503","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/501" || r.URL.Path == "/v1/process-instances/502" || r.URL.Path == "/v1/process-instances/503"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--auto-confirm",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/501",
		"/v1/process-instances/502",
		"/v1/process-instances/503",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

func TestDeleteProcessInstanceCommand_SearchPagingLimitFlow(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"501","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"502","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"503","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"504","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/501" || r.URL.Path == "/v1/process-instances/502" || r.URL.Path == "/v1/process-instances/503"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--auto-confirm",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
		"--limit", "3",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/501",
		"/v1/process-instances/502",
		"/v1/process-instances/503",
	}, deleted.Snapshot())
	require.NotContains(t, strings.Join(deleted.Snapshot(), "\n"), "504")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: yes, next step: limit-reached")
}

func TestDeleteProcessInstanceCommand_SearchPagingBatchSizeLimitFlow(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"701","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"702","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"703","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"704","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":6,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/701" || r.URL.Path == "/v1/process-instances/702"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--auto-confirm",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "4",
		"--limit", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.EqualValues(t, 4, pages[0]["limit"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/701",
		"/v1/process-instances/702",
	}, deleted.Snapshot())
	require.NotContains(t, strings.Join(deleted.Snapshot(), "\n"), "703")
	require.Contains(t, output, "page size: 4, current page: 2, total so far: 2, more matches: yes, next step: limit-reached")
}

func TestDeleteProcessInstanceCommand_SearchPagingAutomationFlow(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]
	searchPage := 0

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			switch searchPage {
			case 0:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"601","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"602","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
			case 1:
				_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"603","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected top-level search request %d", searchPage)
			}
			searchPage++
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/601" || r.URL.Path == "/v1/process-instances/602" || r.URL.Path == "/v1/process-instances/603"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	promptCalls := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		promptCalls++
		return nil
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"--automation",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 2)
	require.EqualValues(t, 2, pages[0]["limit"])
	require.EqualValues(t, 2, pages[1]["from"])
	require.Equal(t, 1, promptCalls)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/601",
		"/v1/process-instances/602",
		"/v1/process-instances/603",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
	require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
}

// Verifies paged delete reports a partial-completion summary when continuation is aborted.
func TestDeleteProcessInstanceCommand_SearchPagingPartialCompletionSummary(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"511","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"512","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/511" || r.URL.Path == "/v1/process-instances/512"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	callCount := 0
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		callCount++
		if callCount == 1 {
			return nil
		}
		return ErrCmdAborted
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/511",
		"/v1/process-instances/512",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
	require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
}

// Verifies paged delete emits a warning-stop summary when overflow state is indeterminate.
func TestDeleteProcessInstanceCommand_SearchPagingWarningStopSummary(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))

			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
				if _, ok := filter["parentProcessInstanceKey"]; ok {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"items":[]}`))
					return
				}
			}

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"processInstanceKey":"521","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"},{"processInstanceKey":"522","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{}}`))
		case r.Method == http.MethodDelete && (r.URL.Path == "/v1/process-instances/521" || r.URL.Path == "/v1/process-instances/522"):
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/process-instances/"):
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			if strings.Contains(key, "/") {
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf(`{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`, key)))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error { return nil }
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--verbose",
		"delete", "process-instance",
		"--state", "completed",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Len(t, pages, 1)
	require.ElementsMatch(t, []string{
		"/v1/process-instances/521",
		"/v1/process-instances/522",
	}, deleted.Snapshot())
	require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
	require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
}

// Verifies direct --key deletion bypasses top-level search pagination logic.
func TestDeleteProcessInstanceCommand_DirectKeyBypassesTopLevelSearchPaging(t *testing.T) {
	var requests safeSlice[string]
	var deleted safeSlice[string]

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests.Append(string(body))
			searchBody := decodeCapturedPISearchRequest(t, string(body))
			filter, _ := searchBody["filter"].(map[string]any)
			w.Header().Set("Content-Type", "application/json")
			if filter != nil {
				if key, ok := filter["processInstanceKey"]; ok && key != nil {
					_, _ = w.Write([]byte(fmt.Sprintf(`{"items":[{"processInstanceKey":"%s","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}]}`, key.(string))))
					return
				}
			}
			_, _ = w.Write([]byte(`{"items":[]}`))
		case r.Method == http.MethodDelete && r.URL.Path == "/v1/process-instances/601":
			deleted.Append(r.URL.Path)
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/601":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processInstanceKey":"601","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"startDate":"2026-03-23T18:00:00Z","endDate":"2026-03-24T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error { return nil }
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--automation",
		"--tenant", "tenant",
		"--json",
		"delete", "process-instance",
		"--key", "601",
		"--no-wait",
		"--batch-size", "2",
	)

	pages := decodeCapturedTopLevelPISearchPages(t, requests.Snapshot())
	require.Empty(t, pages)
	require.Equal(t, []string{"/v1/process-instances/601"}, deleted.Snapshot())
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &got))
	require.Equal(t, string(OutcomeAccepted), got["outcome"])
	require.Contains(t, stderr, "INFO")
}

func TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetail(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/601", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"title":"Not Found","status":404,"detail":"resource not found"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeDeleteProcessInstanceFailureHelper(t, "TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetailHelper", cfgPath)

	require.Equal(t, exitcode.NotFound, code)
	require.Contains(t, output, "resource not found")
	require.Contains(t, output, "delete validation")
	require.Contains(t, output, "ancestry")
	require.NotContains(t, output, "validating process instance keys for cancellation")
	require.NotContains(t, output, "ancestry get")
	require.Contains(t, output, "get process instance")
	require.Less(t, strings.Index(output, "delete validation"), strings.Index(output, "ancestry"))
	require.Less(t, strings.Index(output, "ancestry"), strings.Index(output, "get process instance"))
	require.NotContains(t, output, "fetching process instance with key")
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

func TestDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err, string(output))
	require.NotContains(t, string(output), "either --key")
	require.Contains(t, string(output), "preparation for deleting")
}

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
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output, err := testx.RunCmdSubprocess(t, "TestDeleteProcessDefinitionCommand_LatestSearchUsesEffectiveTenantHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	require.Contains(t, string(output), "no process definitions found to delete")

	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok, "expected search request filter object")
	require.Equal(t, "tenant-a", filter["tenantId"])
	require.Equal(t, true, filter["isLatestVersion"])
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

func TestDeleteProcessInstanceCommand_RejectsRemovedCountFlagHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--count", "2"}

	Execute()
}

func TestDeleteProcessInstanceCommand_RejectsInvalidLimitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--limit", "0"}

	Execute()
}

func TestDeleteProcessInstanceCommand_RejectsLimitWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "123", "--limit", "1"}

	Execute()
}

// Helper-process entrypoint for the search scaffold failure test.
func TestDeleteProcessInstanceSearchScaffoldHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm"}

	Execute()
}

func TestDeleteProcessInstanceCommand_SearchPagingPromptFlowV87IncludesDependencyTotalsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "--verbose", "delete", "process-instance", "--state", "completed", "--no-wait", "--batch-size", "2"}

	Execute()
}

// Helper-process entrypoint for invalid date range validation.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--end-date-after", "2026-02-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestDeleteProcessInstanceCommand_RejectsInvalidDateValueHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestDeleteProcessInstanceCommand_RejectsKeyAndDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}

// Helper-process entrypoint for key-and-relative-day-filter exclusivity validation.
func TestDeleteProcessInstanceCommand_RejectsKeyAndRelativeDayFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--key", "2251799813711967", "--end-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for version capability validation on Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsDateFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for relative-day version capability validation on Camunda 8.7.
func TestDeleteProcessInstanceCommand_RejectsRelativeDayFiltersOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--end-date-newer-days", "7", "--auto-confirm"}

	Execute()
}

// Helper-process entrypoint for the successful search-select-and-delete flow test.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesDateFiltersAndDeletesMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the successful relative-day search-select-and-delete flow test.
func TestDeleteProcessInstanceCommand_SearchSelectionUsesRelativeDayFiltersAndDeletesMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--end-date-older-days", "7", "--end-date-newer-days", "60", "--auto-confirm", "--no-state-check", "--no-wait"}

	Execute()
}

// Helper-process entrypoint for the no-matches failure test.
func TestDeleteProcessInstanceCommand_FailsWhenDateFilteredSearchFindsNoMatchesHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--state", "completed", "--bpmn-process-id", "order-process", "--start-date-after", "2026-01-01", "--end-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for relative-day-only sufficiency validation.
func TestDeleteProcessInstanceCommand_RelativeDayOnlyFiltersAreSufficientHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-instance", "--end-date-older-days", "72"}

	Execute()
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

func TestHelperDeleteProcessDefinitionCommand_DashStdinSatisfiesTargetSelector(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	_, _ = w.WriteString("2251799813692357\n")
	_ = w.Close()
	os.Stdin = r
	defer func() {
		os.Stdin = oldStdin
		_ = r.Close()
	}()

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "delete", "process-definition", "--auto-confirm", "--no-state-check", "-"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeleteProcessDefinitionCommand_LatestSearchUsesEffectiveTenantHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant-a", "delete", "process-definition", "--bpmn-process-id", "order-process", "--latest", "--allow-inconsistent", "--auto-confirm", "--no-wait"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestDeleteProcessInstanceCommand_DirectKeyFailureKeepsSingleRootDetailHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant", "delete", "process-instance", "--key", "601", "--no-wait"}

	Execute()
}
