// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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

// processInstanceMutationPlanningActivity owns the search planning activity so
// prompts and confirmed mutations can happen after the planning scope stops.
type processInstanceMutationPlanningActivity struct {
	cmd       *cobra.Command
	operation string
	enabled   bool
	stop      func()
}

// processInstanceMutationSemanticProgressNow is overridden by command tests to
// exercise durable milestone pacing without real sleeps.
var processInstanceMutationSemanticProgressNow = time.Now

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

// newProcessInstanceMutationSemanticReporter owns the post-confirmation
// completion scope for one process-instance cancel/delete mutation.
func newProcessInstanceMutationSemanticReporter(cmd *cobra.Command, operation string, impact processInstancePageImpact) *opsSemanticProgressReporter {
	channel := opsProgressChannelForMode(processInstanceMutationProgressModeForCommand(cmd))
	return newOpsSemanticProgressReporter(cmd, opsSemanticProgressConfig{
		Scope:  processInstanceMutationSemanticProgressScope(operation, impact.Roots, processInstanceMutationAffectedCoverageAvailable(impact)),
		Policy: opsSemanticProgressOutputPolicyForChannel(channel),
		Now:    processInstanceMutationSemanticProgressNow,
	})
}

// appendProcessInstanceMutationSemanticProgressOptions installs the shared
// completion reporter after the command has confirmed the frozen mutation scope.
func appendProcessInstanceMutationSemanticProgressOptions(cmd *cobra.Command, operation string, impact processInstancePageImpact, opts []processOptions.FacadeOption, affectedCount int) ([]processOptions.FacadeOption, func()) {
	semanticReporter := newProcessInstanceMutationSemanticReporter(cmd, operation, impact)
	mutationOpts := append(compactProcessInstanceMutationOptions(opts),
		processOptions.WithAffectedProcessInstanceCount(affectedCount),
		processOptions.WithProgress(processInstanceMutationSemanticProgressCallback(semanticReporter)),
	)
	return mutationOpts, semanticReporter.Close
}

// newProcessInstanceMutationPlanningActivity starts the search planning
// activity only for modes that allow transient workflow progress.
func newProcessInstanceMutationPlanningActivity(cmd *cobra.Command, operation string) *processInstanceMutationPlanningActivity {
	channel := opsProgressChannelForMode(processInstanceMutationProgressModeForCommand(cmd))
	progress := &processInstanceMutationPlanningActivity{
		cmd:       cmd,
		operation: strings.TrimSpace(operation),
		enabled:   channel.TransientAllowed,
	}
	progress.Resume()
	return progress
}

// Stop ends the current planning activity if it is active.
func (p *processInstanceMutationPlanningActivity) Stop() {
	if p == nil || p.stop == nil {
		return
	}
	p.stop()
	p.stop = nil
}

// Resume opens a new planning activity after a continuation prompt allows the
// search planning traversal to continue.
func (p *processInstanceMutationPlanningActivity) Resume() {
	if p == nil || !p.enabled || p.stop != nil {
		return
	}
	operation := p.operation
	if operation == "" {
		operation = "mutation"
	}
	p.stop = startCommandActivity(p.cmd, fmt.Sprintf("planning process-instance %s scope", operation))
}

// processInstanceMutationAffectedCoverageAvailable allows affected aggregates
// only when the command can prove each root contributes a trustworthy count.
func processInstanceMutationAffectedCoverageAvailable(impact processInstancePageImpact) bool {
	return impact.Roots > 0 && impact.Affected > 0 && (impact.Roots == 1 || impact.Affected == impact.Roots)
}

// processInstanceMutationSemanticProgressCallback converts facade completion
// facts into the ops reporter envelope used by command progress rendering.
func processInstanceMutationSemanticProgressCallback(reporter *opsSemanticProgressReporter) func(processOptions.ProgressEvent) {
	return func(event processOptions.ProgressEvent) {
		if reporter == nil || event.Kind != processOptions.ProgressEventKindCompletion || event.Completion == nil {
			return
		}
		reporter.Report(ops.ProgressEvent{
			Kind: ops.ProgressEventKindCompletion,
			Completion: &ops.CompletionProgress{
				Phase:            event.Completion.Phase,
				CoreResource:     event.Completion.CoreResource,
				Total:            event.Completion.Total,
				Identity:         event.Completion.Identity,
				Disposition:      ops.CompletionDisposition(event.Completion.Disposition),
				FailureDetail:    event.Completion.FailureDetail,
				AffectedResource: event.Completion.AffectedResource,
				AffectedCount:    event.Completion.AffectedCount,
			},
		})
	}
}

// processInstanceMutationProgressModeForCommand derives progress routing from
// render mode, root verbosity flags, and command automation context.
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

// printProcessInstanceMutationPreflight updates workflow activity and emits
// durable preflight detail only in modes where diagnostics are allowed.
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
	for _, line := range tenantContextHumanLines(cmd, *ctx) {
		printOpsDurableLine(cmd, line.Text, line.Warn)
	}
}

// renderProcessInstanceMutationTenantContextStderr emits search confirmation
// context on stderr so stdout remains reserved for machine-oriented results.
func renderProcessInstanceMutationTenantContextStderr(cmd *cobra.Command, ctx tenant.Context) {
	if flagCmdAutomation || !shouldRenderTenantContextHuman(cmd, ctx) || tenantContextHumanRendered(cmd) {
		return
	}
	markTenantContextHumanRendered(cmd)
	for _, line := range tenantContextHumanLines(cmd, ctx) {
		printOpsDurableLine(cmd, line.Text, line.Warn)
	}
}

// processInstanceMutationTenantContextAllowed keeps tenant preflight lines on
// the same human stderr channel as durable process-instance progress.
func processInstanceMutationTenantContextAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed &&
		(channel.Mode == ops.ProgressModeHuman || channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

// printProcessInstanceMutationProgressLine keeps discovery/planning progress on
// stderr and transient activity without leaking text into result stdout.
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

// processInstanceMutationDurableProgressAllowed limits detailed mutation
// planning counters to verbose and debug diagnostics.
func processInstanceMutationDurableProgressAllowed(channel ops.ProgressChannel) bool {
	return channel.DurableAllowed && channel.StderrAllowed && (channel.Mode == ops.ProgressModeVerbose || channel.Mode == ops.ProgressModeDebug)
}

// renderProcessInstanceMutationResultSummary emits the compact final mutation
// progress summary only when human one-line output allows it.
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

// processInstanceMutationHumanResultAllowed protects quiet, automation, and
// machine-readable modes from human summary progress.
func processInstanceMutationHumanResultAllowed(cmd *cobra.Command) bool {
	input := processInstanceMutationProgressModeForCommand(cmd)
	return !input.Quiet && !input.Automation && input.RenderMode == RenderModeOneLine
}

// processInstanceMutationResultWords chooses final summary wording that matches
// the mutation operation and no-wait lifecycle.
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

// compactProcessInstanceMutationOptions suppresses duplicate low-level service
// detail once semantic command progress is installed.
func compactProcessInstanceMutationOptions(opts []processOptions.FacadeOption) []processOptions.FacadeOption {
	out := append([]processOptions.FacadeOption{}, opts...)
	return append(out,
		processOptions.WithSuppressWorkflowDetailLogs(),
		processOptions.WithSuppressProcessInstanceDetailLogs(),
	)
}

// formatProcessInstanceMutationFrozenProgress renders planning counters without
// introducing completion lifecycle wording.
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

// processInstanceMutationStepTotal converts service page total metadata into
// the ops progress certainty model used by preflight rendering.
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

// opsPreflightScopeFromProcessOption mechanically maps process facade preflight
// progress into the shared ops progress model.
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

// opsPageProgressFromProcessOption mechanically maps process facade page
// progress into the shared ops progress model.
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

// opsFrozenScopeProgressFromProcessOption mechanically maps process facade
// frozen-scope progress into the shared ops progress model.
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
