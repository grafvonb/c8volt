// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"

	processOptions "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// processInstancePageImpact captures per-page impact counts used by cancel/delete paging prompts.
//
// These values are accumulated across pages to present users with a continuation prompt that reflects
// both the visible page size and the real operational impact when dependencies are included.
type processInstancePageImpact struct {
	// Requested is the raw number of keys selected from the current search page.
	Requested int
	// Affected is the expanded number of instances impacted after dependency resolution.
	Affected int
	// Roots is the number of root instances in the expanded impact set.
	Roots int
}

// processInstancePageActionResult is the per-page result produced by mutating
// process-instance commands. It keeps operational impact, reporters, and dry-run
// previews together so the paging loop can aggregate them without knowing the
// command-specific cancel/delete implementation details.
type processInstancePageActionResult struct {
	Impact        processInstancePageImpact
	Reports       []process.Reporter
	DryRunPreview *processInstanceDryRunPreview
}

// processInstancePageActionResults is the accumulated result returned from a
// paged cancel/delete operation after all selected pages are processed.
type processInstancePageActionResults struct {
	Reports        []process.Reporter
	DryRunPreviews []processInstanceDryRunPreview
	TenantEvidence process.TenantEvidence
}

// attachProcessInstanceDiscoveryTenantContext combines search semantics with
// tenant evidence already carried by a frozen process-instance plan.
func attachProcessInstanceDiscoveryTenantContext(cmd *cobra.Command, base tenant.Context, evidence process.TenantEvidence) tenant.Context {
	ctx := withTenantContextEvidence(base, evidence.ResolvedTenantIDs, evidence.UnknownTargetCount)
	attachTenantContext(cmd, ctx)
	return ctx
}

// processInstanceDryRunPlanResult keeps command-owned dry-run planning data
// together before the caller either renders a preview or submits a mutation.
type processInstanceDryRunPlanResult struct {
	Plan    process.DryRunPIKeyExpansion
	Impact  processInstancePageImpact
	Preview processInstanceDryRunPreview
}

// processInstanceMutationProgressState serializes progress rendering shared by
// service worker callbacks and command fallback output.
type processInstanceMutationProgressState struct {
	mu   sync.Mutex
	seen bool
}

// planProcessInstanceDryRunPreview builds the shared dry-run plan, impact
// counts, and render payload for one direct-key process-instance batch.
func planProcessInstanceDryRunPreview(cmd *cobra.Command, cli process.API, operation string, keys types.Keys) (processInstanceDryRunPlanResult, error) {
	return planProcessInstanceDryRunPreviewWithOptions(cmd, cli, operation, keys, collectOptions())
}

// planProcessInstanceDryRunPreviewWithOptions lets direct-key callers preserve
// admin-input semantics while search-derived callers keep tenant scoping.
func planProcessInstanceDryRunPreviewWithOptions(cmd *cobra.Command, cli process.API, operation string, keys types.Keys, opts []processOptions.FacadeOption) (processInstanceDryRunPlanResult, error) {
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("preparing %s dry-run scope for %d process instance(s)", operation, len(keys)))
	defer stopActivity()

	plan, err := cli.DryRunCancelOrDeletePlan(context.Background(), keys, flagWorkers, opts...)
	if err != nil {
		return processInstanceDryRunPlanResult{}, fmt.Errorf("%s validation: %w", operation, err)
	}
	if processOptions.ApplyFacadeOptions(opts).IgnoreTenant {
		attachProcessInstanceExplicitTenantContext(cmd, plan.TenantEvidence)
	}

	return processInstanceDryRunPlanResult{
		Plan:    plan,
		Impact:  processInstancePageImpact{Requested: len(keys), Affected: len(plan.Collected), Roots: len(plan.Roots)},
		Preview: newProcessInstanceDryRunPreview(operation, keys, plan),
	}, nil
}

// attachProcessInstanceExplicitTenantContext combines direct-key semantics with
// tenant evidence already carried by a frozen process-instance plan.
func attachProcessInstanceExplicitTenantContext(cmd *cobra.Command, evidence process.TenantEvidence) {
	cfg, _ := config.FromContext(commandContextOrBackground(cmd))
	base := newExplicitKeysTenantContext(configuredTenantID(cfg))
	attachTenantContext(cmd, withTenantContextEvidence(base, evidence.ResolvedTenantIDs, evidence.UnknownTargetCount))
}

// commandContextOrBackground gives direct unit tests the same nil-safe context
// fallback as command execution helpers.
func commandContextOrBackground(cmd *cobra.Command) context.Context {
	if cmd == nil || cmd.Context() == nil {
		return context.Background()
	}
	return cmd.Context()
}

// processInstancePageActionResultFromPlan converts a service-owned mutation
// planning step into the command result shape used by cancel/delete pagination.
func processInstancePageActionResultFromPlan(operation string, step process.ProcessInstanceMutationPlanStep) processInstancePageActionResult {
	keys := types.Keys(step.RequestedKeys)
	preview := newProcessInstanceDryRunPreview(operation, keys, step.Plan)
	return processInstancePageActionResult{
		Impact: processInstancePageImpact{
			Requested: len(keys),
			Affected:  len(step.Plan.Collected),
			Roots:     len(step.Plan.Roots),
		},
		DryRunPreview: &preview,
	}
}

// newProcessInstanceMutationProgressReporter creates a serialized command
// progress callback for mutation workflows that may report from workers.
func newProcessInstanceMutationProgressReporter(cmd *cobra.Command, operation string) func(processOptions.ProgressEvent) {
	progress, _ := newProcessInstanceMutationProgressReporterWithState(cmd, operation)
	return progress
}

// newProcessInstanceMutationProgressReporterWithState returns the progress
// callback plus state used by fallback rendering when services emit no progress.
func newProcessInstanceMutationProgressReporterWithState(cmd *cobra.Command, operation string) (func(processOptions.ProgressEvent), *processInstanceMutationProgressState) {
	channel := opsProgressChannelForMode(processInstanceMutationProgressModeForCommand(cmd))
	operation = strings.TrimSpace(operation)
	state := &processInstanceMutationProgressState{}
	return func(event processOptions.ProgressEvent) {
		state.mu.Lock()
		defer state.mu.Unlock()
		if cmd == nil {
			return
		}
		state.seen = true
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
	}, state
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
		logging.UpdateActivityWithImportance(cmd.Context(), lines[0], logging.ActivityImportanceWorkflow)
	}
	if !processInstanceMutationDurableProgressAllowed(channel) {
		return
	}
	printProcessInstanceMutationTenantContext(cmd, channel)
	printOpsPreflightLines(cmd, scope)
}

// printProcessInstanceMutationTenantContext routes attached discovery context
// through the durable progress channel before process-instance mutation scope.
func printProcessInstanceMutationTenantContext(cmd *cobra.Command, channel ops.ProgressChannel) {
	if !processInstanceMutationTenantContextAllowed(channel) {
		return
	}
	ctx, ok := attachedTenantContext(cmd)
	if !ok || !shouldRenderTenantContextHuman(cmd, *ctx) || tenantContextHumanRendered(cmd) {
		return
	}
	markTenantContextHumanRendered(cmd)
	if line := tenantContextPrimaryHumanLine(*ctx); line != "" {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), line)
	}
	switch len(ctx.ResolvedTenantIDs) {
	case 1:
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Resource tenant: %s\n", ctx.ResolvedTenantIDs[0])
	case 0:
	default:
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Resource tenants: %s\n", strings.Join(ctx.ResolvedTenantIDs, ", "))
	}
	for _, warning := range ctx.Warnings {
		if warning.Code == tenant.ContextWarningUnfilteredSelection {
			continue
		}
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), warning.Message)
	}
}

// renderProcessInstanceMutationTenantContextStderr emits search confirmation
// context on stderr so stdout remains reserved for machine-oriented results.
func renderProcessInstanceMutationTenantContextStderr(cmd *cobra.Command, ctx tenant.Context) {
	if flagCmdAutomation || !shouldRenderTenantContextHuman(cmd, ctx) || tenantContextHumanRendered(cmd) {
		return
	}
	markTenantContextHumanRendered(cmd)
	if line := tenantContextPrimaryHumanLine(ctx); line != "" {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), line)
	}
	switch len(ctx.ResolvedTenantIDs) {
	case 1:
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Resource tenant: %s\n", ctx.ResolvedTenantIDs[0])
	case 0:
	default:
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Resource tenants: %s\n", strings.Join(ctx.ResolvedTenantIDs, ", "))
	}
	for _, warning := range ctx.Warnings {
		if warning.Code == tenant.ContextWarningUnfilteredSelection {
			continue
		}
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), warning.Message)
	}
}

// processInstanceMutationTenantContextAllowed keeps tenant preflight lines on
// the same human stderr channel as durable process-instance progress.
func processInstanceMutationTenantContextAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed &&
		(channel.Mode == ops.ProgressModeHuman || channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

func printProcessInstanceMutationProgressLine(cmd *cobra.Command, line string, channel ops.ProgressChannel) {
	if strings.TrimSpace(line) == "" {
		return
	}
	if channel.TransientAllowed {
		logging.UpdateActivityWithImportance(cmd.Context(), line, logging.ActivityImportanceWorkflow)
	}
	if !processInstanceMutationDurableProgressAllowed(channel) {
		return
	}
	printOpsDurableLine(cmd, line, false)
}

func processInstanceMutationDurableProgressAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed && (channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

func renderProcessInstanceMutationResultSummary(cmd *cobra.Command, operation string, reports []process.Reporter, impact processInstancePageImpact) {
	if !processInstanceMutationHumanResultAllowed(cmd) || len(reports) == 0 {
		return
	}
	total, ok, failed := process.TotalsOf(reports)
	if total == 0 {
		return
	}
	label, verb := processInstanceMutationResultWords(operation, flagNoWait)
	line := fmt.Sprintf("%s: %s %d/%d process-instance tree(s)", label, verb, ok, total)
	if failed > 0 {
		line += fmt.Sprintf(", failed %d", failed)
	}
	if impact.Affected > total {
		line += fmt.Sprintf("; affected process instances: %d", impact.Affected)
	}
	printOpsDurableLine(cmd, line, false)
}

func processInstanceMutationHumanResultAllowed(cmd *cobra.Command) bool {
	input := processInstanceMutationProgressModeForCommand(cmd)
	return !input.Quiet && !input.Automation && input.RenderMode == RenderModeOneLine
}

func processInstanceMutationResultWords(operation string, noWait bool) (label string, verb string) {
	switch strings.TrimSpace(operation) {
	case "cancel":
		if noWait {
			return "cancellation", "submitted"
		}
		return "cancellation", "canceled"
	default:
		if noWait {
			return "deletion", "submitted"
		}
		return "deletion", "deleted"
	}
}

func compactProcessInstanceMutationOptions(opts []processOptions.FacadeOption) []processOptions.FacadeOption {
	out := append([]processOptions.FacadeOption{}, opts...)
	return append(out,
		processOptions.WithSuppressWorkflowDetailLogs(),
		processOptions.WithSuppressProcessInstanceDetailLogs(),
	)
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

// printProcessInstanceMutationPlanStepFallbackProgress preserves legacy
// progress lines when an older service path does not emit shared progress.
func printProcessInstanceMutationPlanStepFallbackProgress(cmd *cobra.Command, operation string, step process.ProcessInstanceMutationPlanStep, progressState *processInstanceMutationProgressState) {
	if progressState == nil {
		return
	}
	progressState.mu.Lock()
	defer progressState.mu.Unlock()
	if progressState.seen {
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
