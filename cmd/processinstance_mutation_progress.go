// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

func newProcessInstanceMutationProgressReporter(cmd *cobra.Command, operation string) func(processOptions.ProgressEvent) {
	progress, _ := newProcessInstanceMutationProgressReporterWithState(cmd, operation)
	return progress
}

func newProcessInstanceMutationProgressReporterWithState(cmd *cobra.Command, operation string) (func(processOptions.ProgressEvent), *bool) {
	channel := opsProgressChannelForMode(processInstanceMutationProgressModeForCommand(cmd))
	operation = strings.TrimSpace(operation)
	seen := false
	return func(event processOptions.ProgressEvent) {
		if cmd == nil {
			return
		}
		seen = true
		switch event.Kind {
		case processOptions.ProgressEventKindPreflight:
			if event.Preflight == nil {
				return
			}
			scope := opsPreflightScopeFromProcessOption(*event.Preflight)
			scope.SelectorSummary = operation + " process-instance"
			printProcessInstanceMutationPreflight(cmd, scope, channel)
		case processOptions.ProgressEventKindPage:
			if event.Page == nil {
				return
			}
			progress := opsPageProgressFromProcessOption(*event.Page)
			printProcessInstanceMutationProgressLine(cmd, formatOpsPageProgress(progress, "process instance(s)"), channel)
		case processOptions.ProgressEventKindFrozenScope:
			if event.FrozenScope == nil {
				return
			}
			progress := opsFrozenScopeProgressFromProcessOption(*event.FrozenScope)
			if strings.TrimSpace(progress.Phase) == "planning process-instance mutation scope" && operation != "" {
				progress.Phase = "planning process-instance " + operation + " scope"
			}
			printProcessInstanceMutationProgressLine(cmd, formatProcessInstanceMutationFrozenProgress(progress), channel)
		}
	}, &seen
}

func processInstanceMutationProgressModeForCommand(cmd *cobra.Command) opsProgressModeInput {
	input := opsProgressModeInput{
		RenderMode: pickMode(),
		Verbose:    flagVerbose,
		Quiet:      flagQuiet,
		Debug:      flagDebug,
	}
	if cmd == nil || cmd.Context() == nil {
		return input
	}
	if cfg, err := config.FromContext(cmd.Context()); err == nil && cfg != nil {
		input.Automation = cfg.App.Automation
		return input
	}
	if flag := cmd.Flags().Lookup("automation"); flag != nil && flag.Value.String() == "true" {
		input.Automation = true
	}
	return input
}

func printProcessInstanceMutationPreflight(cmd *cobra.Command, scope ops.PreflightScope, channel ops.ProgressChannel) {
	lines := formatOpsPreflightScope(scope)
	if channel.TransientAllowed && len(lines) > 0 {
		logging.UpdateActivity(cmd.Context(), lines[0])
	}
	if !processInstanceMutationDurableProgressAllowed(channel) {
		return
	}
	for _, line := range lines {
		fmt.Fprintln(cmd.ErrOrStderr(), line)
	}
}

func printProcessInstanceMutationProgressLine(cmd *cobra.Command, line string, channel ops.ProgressChannel) {
	if strings.TrimSpace(line) == "" {
		return
	}
	if channel.TransientAllowed {
		logging.UpdateActivity(cmd.Context(), line)
	}
	if !processInstanceMutationDurableProgressAllowed(channel) {
		return
	}
	fmt.Fprintln(cmd.ErrOrStderr(), line)
}

func processInstanceMutationDurableProgressAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed && (channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

func formatProcessInstanceMutationFrozenProgress(progress ops.FrozenScopeProgress) string {
	resource := strings.TrimSpace(progress.CoreResource)
	if resource == "" {
		resource = "resource(s)"
	}
	phase := strings.TrimSpace(progress.Phase)
	if phase == "" {
		return fmt.Sprintf("%d/%d %s", progress.Done, progress.Total, resource)
	}
	return fmt.Sprintf("%s %d/%d %s", phase, progress.Done, progress.Total, resource)
}

func printProcessInstanceMutationPlanStepFallbackProgress(cmd *cobra.Command, operation string, step process.ProcessInstanceMutationPlanStep, progressSeen *bool) {
	if progressSeen == nil || *progressSeen {
		return
	}
	channel := opsProgressChannelForMode(processInstanceMutationProgressModeForCommand(cmd))
	total, totalKind := processInstanceMutationStepTotal(step)
	pageCount, pageCountKind := pageCountFromBasicSearchTotal(total, totalKind, step.Page.Request.Size)
	printProcessInstanceMutationPreflight(cmd, ops.PreflightScope{
		Phase:           "preflight",
		CoreResource:    "process_instance",
		SelectorSummary: strings.TrimSpace(operation) + " process-instance",
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        step.Page.Request.Size,
		PageCount:       pageCount,
		PageCountKind:   pageCountKind,
		ConsequenceSummary: ops.ConsequenceSummary{
			WorkSummary: "plan process-instance " + strings.TrimSpace(operation) + " scope",
			RiskSummary: "destructive mutation",
		},
		RequiresConfirmation: true,
	}, channel)
	printProcessInstanceMutationProgressLine(cmd, formatProcessInstanceMutationFrozenProgress(ops.FrozenScopeProgress{
		Phase:        "planning process-instance " + strings.TrimSpace(operation) + " scope",
		CoreResource: "process instance(s)",
		Done:         len(step.RequestedKeys),
		Total:        len(step.RequestedKeys),
	}), channel)
}

func processInstanceMutationStepTotal(step process.ProcessInstanceMutationPlanStep) (*int64, ops.TotalCertainty) {
	if step.Page.ReportedTotal == nil {
		return nil, ops.TotalCertaintyUnknown
	}
	total := step.Page.ReportedTotal.Count
	switch step.Page.ReportedTotal.Kind {
	case process.ProcessInstanceReportedTotalKindExact:
		return &total, ops.TotalCertaintyExact
	case process.ProcessInstanceReportedTotalKindLowerBound:
		return &total, ops.TotalCertaintyLowerBound
	default:
		return nil, ops.TotalCertaintyUnknown
	}
}

func opsPreflightScopeFromProcessOption(scope processOptions.PreflightScope) ops.PreflightScope {
	return ops.PreflightScope{
		Phase:           scope.Phase,
		Command:         scope.Command,
		CoreResource:    scope.CoreResource,
		SelectorSummary: scope.SelectorSummary,
		Total:           scope.Total,
		TotalKind:       ops.TotalCertainty(scope.TotalKind),
		PageSize:        scope.PageSize,
		PageCount:       scope.PageCount,
		PageCountKind:   ops.PageCountKind(scope.PageCountKind),
		ConsequenceSummary: ops.ConsequenceSummary{
			ResourceSummary:  scope.ConsequenceSummary.ResourceSummary,
			WorkSummary:      scope.ConsequenceSummary.WorkSummary,
			RiskSummary:      scope.ConsequenceSummary.RiskSummary,
			ConfirmationText: scope.ConsequenceSummary.ConfirmationText,
		},
		RequiresConfirmation: scope.RequiresConfirmation,
		ExpensivePreflight:   scope.ExpensivePreflight,
	}
}

func opsPageProgressFromProcessOption(progress processOptions.PageProgress) ops.PageProgress {
	return ops.PageProgress{
		Phase:            progress.Phase,
		CurrentPage:      progress.CurrentPage,
		PageCount:        progress.PageCount,
		PageCountKind:    ops.PageCountKind(progress.PageCountKind),
		PageSize:         progress.PageSize,
		CurrentPageCount: progress.CurrentPageCount,
		Seen:             progress.Seen,
		Selected:         progress.Selected,
		OverflowState:    ops.OverflowState(progress.OverflowState),
		LimitReached:     progress.LimitReached,
	}
}

func opsFrozenScopeProgressFromProcessOption(progress processOptions.FrozenScopeProgress) ops.FrozenScopeProgress {
	return ops.FrozenScopeProgress{
		Phase:        progress.Phase,
		CoreResource: progress.CoreResource,
		Done:         progress.Done,
		Total:        progress.Total,
		Elapsed:      progress.Elapsed,
		Rate:         progress.Rate,
		ETA:          progress.ETA,
		Errors:       progress.Errors,
	}
}
