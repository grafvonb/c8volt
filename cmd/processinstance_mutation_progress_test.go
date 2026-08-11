// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// pendingProcessInstanceMutationProgressT064 marks the historical progress
// contract gate while the concrete tests define the preserved behavior.
func pendingProcessInstanceMutationProgressT064(t *testing.T) {
	t.Helper()
}

// TestCancelProcessInstanceSearchProgressContractPendingT064 defines the shared
// destructive progress contract for search-selected cancel.
func TestCancelProcessInstanceSearchProgressContractPendingT064(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true
	flagGetPISize = 1

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.True(t, autoConfirm)
		require.Contains(t, prompt, "cancel")
		return nil
	}

	cli := stubProcessAPI{
		planProcessInstanceMutationPages: func(_ context.Context, request process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, opts ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
			require.Equal(t, int32(1), request.SearchRequest.Page.Size)
			require.NotNil(t, options.ApplyFacadeOptions(opts).Progress)
			page := process.ProcessInstancePage{
				Items:         []process.ProcessInstance{{Key: "401", State: process.StateActive}},
				Request:       process.ProcessInstancePageRequest{From: 0, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateHasMore,
				ReportedTotal: &process.ProcessInstanceReportedTotal{Count: 2, Kind: process.ProcessInstanceReportedTotalKindLowerBound},
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-401"},
				Collected: typex.Keys{"root-401", "401"},
				Outcome:   process.TraversalOutcomeComplete,
			}
			action, err := visitor(process.ProcessInstanceMutationPlanStep{
				Page:             page,
				RequestedKeys:    []string{"401"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 2,
			})
			require.NoError(t, err)
			require.Equal(t, process.ProcessInstanceSearchPageActionContinue, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: page, RequestedKeys: []string{"401"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 2}},
				Pages:            1,
				RequestedCount:   1,
				CumulativeImpact: 2,
			}, nil
		},
		cancelProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.CancelReports, error) {
			require.Equal(t, typex.Keys{"root-401"}, keys)
			cfg := options.ApplyFacadeOptions(opts)
			require.Equal(t, 2, cfg.AffectedProcessInstanceCount)
			require.NotNil(t, cfg.Progress)
			require.True(t, cfg.SuppressWorkflowDetailLogs)
			require.True(t, cfg.SuppressProcessInstanceDetailLogs)
			cfg.Progress(options.ProgressEvent{
				Kind: options.ProgressEventKindFrozenScope,
				FrozenScope: &options.FrozenScopeProgress{
					Phase:        "cancelling process instances",
					CoreResource: "process instance(s)",
					Done:         1,
					Total:        1,
				},
			})
			return process.CancelReports{Items: []process.CancelReport{{Key: "root-401", Ok: true}}}, nil
		},
	}

	got, err := cancelProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{State: process.StateActive})

	require.NoError(t, err)
	require.Len(t, got.Reports, 1)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "process-instance cancel scope: cancel process-instance matched at least 2 process instances; page size: 1; discovery pages: at least 2")
	require.Contains(t, stderr.String(), "planning process-instance cancel scope 1/1 process instance(s)")
	require.Contains(t, stderr.String(), "cancelling process instances 1/1 process instance(s)")
	require.Contains(t, stderr.String(), "cancellation: canceled 1/1 process-instance tree(s); affected process instances: 2")
	require.NotContains(t, stderr.String(), "/v2/")
	require.NotContains(t, stderr.String(), "cursor")
}

// TestCancelProcessInstanceSearchQuietAndAutomationSuppressProgress verifies machine modes keep mutation progress transient-only.
func TestCancelProcessInstanceSearchQuietAndAutomationSuppressProgress(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)

	for _, mode := range []struct {
		name  string
		setup func()
	}{
		{name: "json", setup: func() { flagViewAsJson = true }},
		{name: "quiet", setup: func() { flagQuiet = true }},
		{name: "automation", setup: func() { flagCmdAutomation = true }},
	} {
		t.Run(mode.name, func(t *testing.T) {
			stdout, stderr := exerciseProcessInstanceMutationProgressOutput(t, "cancel", mode.setup)
			require.NotContains(t, stdout, "scope:")
			require.NotContains(t, stdout, "planning process-instance cancel scope")
			require.NotContains(t, stdout, "cancelling process instances")
			require.NotContains(t, stderr, "scope:")
			require.NotContains(t, stderr, "planning process-instance cancel scope")
			require.NotContains(t, stderr, "cancelling process instances")
		})
	}
}

// exerciseProcessInstanceMutationProgressOutput captures stdout and stderr for
// progress mode gating without running a full destructive command.
func exerciseProcessInstanceMutationProgressOutput(t *testing.T, operation string, setup func()) (string, string) {
	t.Helper()
	resetProcessInstanceCommandGlobals()
	prevQuiet := flagQuiet
	prevAutomation := flagCmdAutomation
	t.Cleanup(func() {
		resetProcessInstanceCommandGlobals()
		flagQuiet = prevQuiet
		flagCmdAutomation = prevAutomation
	})
	if setup != nil {
		setup()
	}

	cmd := &cobra.Command{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	total := int64(1)
	pageCount := int64(1)
	progress := newProcessInstanceMutationProgressReporter(cmd, operation)
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindPreflight,
		Preflight: &options.PreflightScope{
			CoreResource:    "process_instance",
			SelectorSummary: operation + " process-instance",
			Total:           &total,
			TotalKind:       options.TotalCertaintyExact,
			PageSize:        1,
			PageCount:       &pageCount,
			PageCountKind:   options.PageCountKindExact,
			ConsequenceSummary: options.ConsequenceSummary{
				WorkSummary: "plan process-instance " + operation + " scope",
				RiskSummary: "destructive mutation",
			},
			RequiresConfirmation: true,
		},
	})
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "planning process-instance mutation scope",
			CoreResource: "process instance(s)",
			Done:         1,
			Total:        1,
		},
	})
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        processInstanceMutationTestMutationPhase(operation),
			CoreResource: "process instance(s)",
			Done:         1,
			Total:        1,
		},
	})
	return stdout.String(), stderr.String()
}

// processInstanceMutationTestMutationPhase returns the service phase text the
// mutation progress reporter should route for each destructive operation.
func processInstanceMutationTestMutationPhase(operation string) string {
	switch operation {
	case "cancel":
		return "cancelling process instances"
	case "delete":
		return "deleting process instances"
	default:
		return operation + " process instances"
	}
}

// TestCancelProcessInstanceProgressUsesWorkflowImportance verifies cancel progress stays above nested service waits and requests.
func TestCancelProcessInstanceProgressUsesWorkflowImportance(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	progress := newProcessInstanceMutationProgressReporter(cmd, "cancel")
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "cancelling process instances",
			CoreResource: "process instance(s)",
			Done:         3,
			Total:        10,
		},
	})

	require.Equal(t, []activitysink.Update{{
		Message:    "cancelling process instances 3/10 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

// TestDeleteProcessInstanceProgressUsesWorkflowImportance verifies delete progress stays above nested service waits and requests.
func TestDeleteProcessInstanceProgressUsesWorkflowImportance(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	progress := newProcessInstanceMutationProgressReporter(cmd, "delete")
	progress(options.ProgressEvent{
		Kind: options.ProgressEventKindFrozenScope,
		FrozenScope: &options.FrozenScopeProgress{
			Phase:        "deleting process instances",
			CoreResource: "process instance(s)",
			Done:         4,
			Total:        12,
		},
	})

	require.Equal(t, []activitysink.Update{{
		Message:    "deleting process instances 4/12 process instance(s)",
		Importance: logging.ActivityImportanceWorkflow,
	}}, sink.PriorityUpdates())
}

// TestDeleteProcessInstanceSearchProgressContractPendingT064 defines the shared
// destructive progress contract for search-selected delete.
func TestDeleteProcessInstanceSearchProgressContractPendingT064(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	flagCmdAutoConfirm = true
	flagVerbose = true
	flagGetPISize = 1

	cmd := &cobra.Command{}
	cmd.Flags().Int32("batch-size", 1000, "")
	require.NoError(t, cmd.Flags().Set("batch-size", "1"))
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	prevConfirm := confirmCmdOrAbortFn
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		require.True(t, autoConfirm)
		require.Contains(t, prompt, "delete")
		return nil
	}

	cli := stubProcessAPI{
		planProcessInstanceMutationPages: func(_ context.Context, request process.ProcessInstanceMutationPlanRequest, visitor process.ProcessInstanceMutationPlanVisitor, opts ...options.FacadeOption) (process.ProcessInstanceMutationPlanPagesResult, error) {
			require.Equal(t, int32(1), request.SearchRequest.Page.Size)
			require.NotNil(t, options.ApplyFacadeOptions(opts).Progress)
			page := process.ProcessInstancePage{
				Items:         []process.ProcessInstance{{Key: "401", State: process.StateCompleted}},
				Request:       process.ProcessInstancePageRequest{From: 0, Size: 1},
				OverflowState: process.ProcessInstanceOverflowStateNoMore,
				ReportedTotal: &process.ProcessInstanceReportedTotal{Count: 1, Kind: process.ProcessInstanceReportedTotalKindExact},
			}
			plan := process.DryRunPIKeyExpansion{
				Roots:     typex.Keys{"root-401"},
				Collected: typex.Keys{"root-401", "401"},
				Outcome:   process.TraversalOutcomeComplete,
			}
			action, err := visitor(process.ProcessInstanceMutationPlanStep{
				Page:             page,
				RequestedKeys:    []string{"401"},
				Plan:             plan,
				CumulativeCount:  1,
				CumulativeImpact: 2,
			})
			require.NoError(t, err)
			require.Equal(t, process.ProcessInstanceSearchPageActionStop, action)
			return process.ProcessInstanceMutationPlanPagesResult{
				Plans:            []process.ProcessInstanceMutationPlanStep{{Page: page, RequestedKeys: []string{"401"}, Plan: plan, CumulativeCount: 1, CumulativeImpact: 2}},
				Pages:            1,
				RequestedCount:   1,
				CumulativeImpact: 2,
			}, nil
		},
		deleteProcessInstances: func(_ context.Context, keys typex.Keys, _ int, opts ...options.FacadeOption) (process.DeleteReports, error) {
			require.Equal(t, typex.Keys{"root-401"}, keys)
			cfg := options.ApplyFacadeOptions(opts)
			require.Equal(t, 2, cfg.AffectedProcessInstanceCount)
			require.NotNil(t, cfg.Progress)
			require.True(t, cfg.SuppressWorkflowDetailLogs)
			require.True(t, cfg.SuppressProcessInstanceDetailLogs)
			cfg.Progress(options.ProgressEvent{
				Kind: options.ProgressEventKindFrozenScope,
				FrozenScope: &options.FrozenScopeProgress{
					Phase:        "deleting process instances",
					CoreResource: "process instance(s)",
					Done:         1,
					Total:        1,
				},
			})
			return process.DeleteReports{Items: []process.DeleteReport{{Key: "root-401", Ok: true}}}, nil
		},
	}

	got, err := deleteProcessInstanceSearchPages(cmd, cli, nil, process.ProcessInstanceFilter{State: process.StateCompleted})

	require.NoError(t, err)
	require.Len(t, got.Reports, 1)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "process-instance delete scope: delete process-instance matched 1 process instance; page size: 1; discovery pages: 1")
	require.Contains(t, stderr.String(), "planning process-instance delete scope 1/1 process instance(s)")
	require.Contains(t, stderr.String(), "deleting process instances 1/1 process instance(s)")
	require.Contains(t, stderr.String(), "deletion: deleted 1/1 process-instance tree(s); affected process instances: 2")
	require.NotContains(t, stderr.String(), "/v2/")
	require.NotContains(t, stderr.String(), "cursor")
}

// TestDeleteProcessInstanceSearchQuietAndAutomationSuppressProgress verifies machine modes keep mutation progress transient-only.
func TestDeleteProcessInstanceSearchQuietAndAutomationSuppressProgress(t *testing.T) {
	pendingProcessInstanceMutationProgressT064(t)

	for _, mode := range []struct {
		name  string
		setup func()
	}{
		{name: "json", setup: func() { flagViewAsJson = true }},
		{name: "quiet", setup: func() { flagQuiet = true }},
		{name: "automation", setup: func() { flagCmdAutomation = true }},
	} {
		t.Run(mode.name, func(t *testing.T) {
			stdout, stderr := exerciseProcessInstanceMutationProgressOutput(t, "delete", mode.setup)
			require.NotContains(t, stdout, "scope:")
			require.NotContains(t, stdout, "planning process-instance delete scope")
			require.NotContains(t, stdout, "deleting process instances")
			require.NotContains(t, stderr, "scope:")
			require.NotContains(t, stderr, "planning process-instance delete scope")
			require.NotContains(t, stderr, "deleting process instances")
		})
	}
}
