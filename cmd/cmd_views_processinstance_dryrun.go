package cmd

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

type processInstanceDryRunMissingAncestor struct {
	Key      string `json:"key"`
	StartKey string `json:"startKey"`
}

// processInstanceDryRunSelectedFinalState retains the selected resource observation.
type processInstanceDryRunSelectedFinalState struct {
	BusinessId *string       `json:"businessId,omitempty"`
	Key        string        `json:"key"`
	State      process.State `json:"state"`
}

// processInstanceDryRunRequiresCancelBeforeDelete retains the selected resource observation.
type processInstanceDryRunRequiresCancelBeforeDelete struct {
	BusinessId *string       `json:"businessId,omitempty"`
	Key        string        `json:"key"`
	State      process.State `json:"state"`
}

// processInstanceDryRunPreview retains full selected and expanded scope, including optional BID provenance.
type processInstanceDryRunPreview struct {
	Selection                       *process.ProcessInstanceSelection                 `json:"selection,omitempty"`
	Operation                       string                                            `json:"operation"`
	RequestedKeys                   []string                                          `json:"requestedKeys"`
	ResolvedRoots                   []string                                          `json:"resolvedRoots"`
	AffectedFamilyKeys              []string                                          `json:"affectedFamilyKeys"`
	RequestedCount                  int                                               `json:"requestedCount"`
	ResolvedRootCount               int                                               `json:"resolvedRootCount"`
	AffectedCount                   int                                               `json:"affectedCount"`
	FinalStateCount                 int                                               `json:"selectedFinalStateCount"`
	SelectedFinalState              []processInstanceDryRunSelectedFinalState         `json:"selectedFinalState"`
	RequiresCancelBeforeDeleteCount int                                               `json:"requiresCancelBeforeDeleteCount"`
	RequiresCancelBeforeDelete      []processInstanceDryRunRequiresCancelBeforeDelete `json:"requiresCancelBeforeDelete"`
	TraversalOutcome                process.TraversalOutcome                          `json:"traversalOutcome"`
	ScopeComplete                   bool                                              `json:"scopeComplete"`
	Warning                         string                                            `json:"warning"`
	MissingAncestors                []processInstanceDryRunMissingAncestor            `json:"missingAncestors"`
	MutationSubmitted               bool                                              `json:"mutationSubmitted"`
}

type processInstanceDryRunSummary struct {
	Operation                       string                                            `json:"operation"`
	RequestedCount                  int                                               `json:"requestedCount"`
	ResolvedRootCount               int                                               `json:"resolvedRootCount"`
	AffectedCount                   int                                               `json:"affectedCount"`
	FinalStateCount                 int                                               `json:"selectedFinalStateCount"`
	SelectedFinalState              []processInstanceDryRunSelectedFinalState         `json:"selectedFinalState"`
	RequiresCancelBeforeDeleteCount int                                               `json:"requiresCancelBeforeDeleteCount"`
	RequiresCancelBeforeDelete      []processInstanceDryRunRequiresCancelBeforeDelete `json:"requiresCancelBeforeDelete"`
	TraversalOutcome                process.TraversalOutcome                          `json:"traversalOutcome"`
	ScopeComplete                   bool                                              `json:"scopeComplete"`
	Warning                         string                                            `json:"warning"`
	MissingAncestors                []processInstanceDryRunMissingAncestor            `json:"missingAncestors"`
	Previews                        []processInstanceDryRunPreview                    `json:"previews"`
	MutationSubmitted               bool                                              `json:"mutationSubmitted"`
}

// newProcessInstanceDryRunPreview maps a dry-run expansion into the command payload contract.
func newProcessInstanceDryRunPreview(operation string, requested types.Keys, plan process.DryRunPIKeyExpansion) processInstanceDryRunPreview {
	outcome := processInstanceDryRunTraversalOutcome(plan)
	requiresCancelBeforeDelete := newProcessInstanceDryRunRequiresCancelBeforeDelete(nil)
	if operation == "delete" {
		requiresCancelBeforeDelete = newProcessInstanceDryRunRequiresCancelBeforeDelete(plan.RequiresCancelBeforeDelete)
	}
	return processInstanceDryRunPreview{
		Operation:                       operation,
		RequestedKeys:                   append([]string(nil), requested...),
		ResolvedRoots:                   append([]string(nil), plan.Roots...),
		AffectedFamilyKeys:              append([]string(nil), plan.Collected...),
		RequestedCount:                  len(requested),
		ResolvedRootCount:               len(plan.Roots),
		AffectedCount:                   len(plan.Collected),
		FinalStateCount:                 len(plan.SelectedFinalState),
		SelectedFinalState:              newProcessInstanceDryRunSelectedFinalState(plan.SelectedFinalState),
		RequiresCancelBeforeDeleteCount: len(requiresCancelBeforeDelete),
		RequiresCancelBeforeDelete:      requiresCancelBeforeDelete,
		TraversalOutcome:                outcome,
		ScopeComplete:                   outcome == process.TraversalOutcomeComplete,
		Warning:                         plan.Warning,
		MissingAncestors:                newProcessInstanceDryRunMissingAncestors(plan.MissingAncestors),
		MutationSubmitted:               false,
	}
}

// newProcessInstanceDryRunSummary aggregates paged dry-run previews into one command payload.
func newProcessInstanceDryRunSummary(operation string, previews []processInstanceDryRunPreview) processInstanceDryRunSummary {
	summary := processInstanceDryRunSummary{
		Operation:         operation,
		TraversalOutcome:  process.TraversalOutcomeComplete,
		ScopeComplete:     true,
		Previews:          append([]processInstanceDryRunPreview(nil), previews...),
		MutationSubmitted: false,
	}
	for i := range summary.Previews {
		summary.Previews[i].SelectedFinalState = toolx.MapSlice(previews[i].SelectedFinalState, func(item processInstanceDryRunSelectedFinalState) processInstanceDryRunSelectedFinalState {
			item.BusinessId = toolx.CopyPtr(item.BusinessId)
			return item
		})
		summary.Previews[i].RequiresCancelBeforeDelete = toolx.MapSlice(previews[i].RequiresCancelBeforeDelete, func(item processInstanceDryRunRequiresCancelBeforeDelete) processInstanceDryRunRequiresCancelBeforeDelete {
			item.BusinessId = toolx.CopyPtr(item.BusinessId)
			return item
		})
	}
	for _, preview := range summary.Previews {
		summary.RequestedCount += preview.RequestedCount
		summary.ResolvedRootCount += preview.ResolvedRootCount
		summary.AffectedCount += preview.AffectedCount
		summary.SelectedFinalState = appendUniqueProcessInstanceDryRunSelectedFinalState(summary.SelectedFinalState, preview.SelectedFinalState...)
		summary.FinalStateCount = len(summary.SelectedFinalState)
		summary.RequiresCancelBeforeDelete = appendUniqueProcessInstanceDryRunRequiresCancelBeforeDelete(summary.RequiresCancelBeforeDelete, preview.RequiresCancelBeforeDelete...)
		summary.RequiresCancelBeforeDeleteCount = len(summary.RequiresCancelBeforeDelete)
		if !preview.ScopeComplete {
			summary.TraversalOutcome = process.TraversalOutcomePartial
			summary.ScopeComplete = false
		}
		if preview.Warning != "" && summary.Warning == "" {
			summary.Warning = preview.Warning
		}
		summary.MissingAncestors = appendUniqueProcessInstanceDryRunMissingAncestors(summary.MissingAncestors, preview.MissingAncestors...)
	}
	return summary
}

// appendUniqueProcessInstanceDryRunMissingAncestors appends missing ancestors while preserving first-seen order.
func appendUniqueProcessInstanceDryRunMissingAncestors(dst []processInstanceDryRunMissingAncestor, src ...processInstanceDryRunMissingAncestor) []processInstanceDryRunMissingAncestor {
	seen := make(map[processInstanceDryRunMissingAncestor]struct{}, len(dst)+len(src))
	for _, item := range dst {
		seen[item] = struct{}{}
	}
	for _, item := range src {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		dst = append(dst, item)
	}
	return dst
}

// appendUniqueProcessInstanceDryRunSelectedFinalState appends final-state entries while preserving first-seen order.
func appendUniqueProcessInstanceDryRunSelectedFinalState(dst []processInstanceDryRunSelectedFinalState, src ...processInstanceDryRunSelectedFinalState) []processInstanceDryRunSelectedFinalState {
	seen := make(map[string]struct{}, len(dst)+len(src))
	for _, item := range dst {
		seen[item.Key] = struct{}{}
	}
	for _, item := range src {
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		item.BusinessId = toolx.CopyPtr(item.BusinessId)
		dst = append(dst, item)
	}
	return dst
}

// appendUniqueProcessInstanceDryRunRequiresCancelBeforeDelete appends non-final
// delete blockers while preserving first-seen order.
func appendUniqueProcessInstanceDryRunRequiresCancelBeforeDelete(dst []processInstanceDryRunRequiresCancelBeforeDelete, src ...processInstanceDryRunRequiresCancelBeforeDelete) []processInstanceDryRunRequiresCancelBeforeDelete {
	seen := make(map[string]struct{}, len(dst)+len(src))
	for _, item := range dst {
		seen[item.Key] = struct{}{}
	}
	for _, item := range src {
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		item.BusinessId = toolx.CopyPtr(item.BusinessId)
		dst = append(dst, item)
	}
	return dst
}

// processInstanceDryRunTraversalOutcome normalizes empty traversal outcomes to the complete state.
func processInstanceDryRunTraversalOutcome(plan process.DryRunPIKeyExpansion) process.TraversalOutcome {
	if plan.Outcome != "" {
		return plan.Outcome
	}
	if plan.Warning != "" || len(plan.MissingAncestors) > 0 {
		return process.TraversalOutcomePartial
	}
	return process.TraversalOutcomeComplete
}

// newProcessInstanceDryRunMissingAncestors converts domain missing-ancestor details to command JSON items.
func newProcessInstanceDryRunMissingAncestors(items []process.MissingAncestor) []processInstanceDryRunMissingAncestor {
	out := make([]processInstanceDryRunMissingAncestor, 0, len(items))
	for _, item := range items {
		out = append(out, processInstanceDryRunMissingAncestor{Key: item.Key, StartKey: item.StartKey})
	}
	return out
}

// newProcessInstanceDryRunSelectedFinalState converts final-state process instances to command JSON items.
func newProcessInstanceDryRunSelectedFinalState(items []process.ProcessInstance) []processInstanceDryRunSelectedFinalState {
	out := make([]processInstanceDryRunSelectedFinalState, 0, len(items))
	for _, item := range items {
		out = append(out, processInstanceDryRunSelectedFinalState{Key: item.Key, State: item.State, BusinessId: toolx.CopyPtr(item.BusinessId)})
	}
	return out
}

// newProcessInstanceDryRunRequiresCancelBeforeDelete converts non-final delete blockers to command JSON items.
func newProcessInstanceDryRunRequiresCancelBeforeDelete(items []process.ProcessInstance) []processInstanceDryRunRequiresCancelBeforeDelete {
	out := make([]processInstanceDryRunRequiresCancelBeforeDelete, 0, len(items))
	for _, item := range items {
		out = append(out, processInstanceDryRunRequiresCancelBeforeDelete{Key: item.Key, State: item.State, BusinessId: toolx.CopyPtr(item.BusinessId)})
	}
	return out
}

// renderProcessInstanceDryRunPreview renders a single keyed or page-level dry-run preview.
func renderProcessInstanceDryRunPreview(cmd *cobra.Command, preview processInstanceDryRunPreview) error {
	if pickMode() == RenderModeJSON {
		return renderProcessInstanceDryRunResult(cmd, preview)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, key := range preview.AffectedFamilyKeys {
			renderOutputLine(cmd, "%s", key)
		}
		return nil
	}

	renderAttachedTenantContext(cmd)
	renderHumanLine(cmd, "dry run: %s process-instance", preview.Operation)
	renderHumanLine(cmd, "selected process instances: %d", preview.RequestedCount)
	if preview.Selection != nil {
		renderBusinessIDSelectionScope(cmd, *preview.Selection)
		renderHumanLine(cmd, "dependency expansion: %d additional process instance(s) due to dependencies", max(0, preview.AffectedCount-preview.RequestedCount))
	}
	renderHumanLine(cmd, "process-instance trees to %s: %d", preview.Operation, preview.ResolvedRootCount)
	renderHumanLine(cmd, "process instances in scope: %d", preview.AffectedCount)
	printProcessInstanceDryRunSelectedFinalState(cmd, preview.Operation, preview.SelectedFinalState)
	printProcessInstanceDryRunRequiresCancelBeforeDelete(cmd, preview.Operation, preview.RequiresCancelBeforeDelete)
	printProcessInstanceDryRunScope(cmd, preview.TraversalOutcome, preview.Warning, preview.MissingAncestors)
	if flagVerbose {
		printProcessInstanceDryRunKeys(cmd, "selected process-instance keys", preview.RequestedKeys)
		printProcessInstanceDryRunKeys(cmd, "root process-instance tree keys", preview.ResolvedRoots)
		printProcessInstanceDryRunKeys(cmd, "in-scope process-instance keys", preview.AffectedFamilyKeys)
	}
	return nil
}

// renderProcessInstanceDryRunSummary renders the aggregate dry-run result for paged searches.
func renderProcessInstanceDryRunSummary(cmd *cobra.Command, summary processInstanceDryRunSummary) error {
	if pickMode() == RenderModeJSON {
		return renderProcessInstanceDryRunResult(cmd, summary)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, preview := range summary.Previews {
			for _, key := range preview.AffectedFamilyKeys {
				renderOutputLine(cmd, "%s", key)
			}
		}
		return nil
	}

	renderAttachedTenantContext(cmd)
	renderHumanLine(cmd, "dry run: %s process-instance", summary.Operation)
	renderHumanLine(cmd, "selected process instances: %d", summary.RequestedCount)
	renderHumanLine(cmd, "process-instance trees to %s: %d", summary.Operation, summary.ResolvedRootCount)
	renderHumanLine(cmd, "process instances in scope: %d", summary.AffectedCount)
	printProcessInstanceDryRunSelectedFinalState(cmd, summary.Operation, summary.SelectedFinalState)
	printProcessInstanceDryRunRequiresCancelBeforeDelete(cmd, summary.Operation, summary.RequiresCancelBeforeDelete)
	printProcessInstanceDryRunScope(cmd, summary.TraversalOutcome, summary.Warning, summary.MissingAncestors)
	return nil
}

// renderProcessInstanceDryRunResult writes dry-run payloads through the shared result envelope when required.
func renderProcessInstanceDryRunResult[T any](cmd *cobra.Command, payload T) error {
	if !commandUsesSharedEnvelope(cmd, pickMode()) {
		return nil
	}
	return renderSucceededResult(cmd, payload)
}

// printProcessInstanceDryRunKeys writes a labeled verbose dry-run key list.
func printProcessInstanceDryRunKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		renderHumanLine(cmd, "%s: none", label)
		return
	}
	renderHumanLine(cmd, "%s: %s", label, strings.Join(keys, ", "))
}

// printProcessInstanceDryRunSelectedFinalState explains selected instances that are already terminal.
func printProcessInstanceDryRunSelectedFinalState(cmd *cobra.Command, operation string, items []processInstanceDryRunSelectedFinalState) {
	if len(items) == 0 {
		return
	}
	details := []string{fmt.Sprintf("states: %s", formatProcessInstanceDryRunSelectedFinalStateStates(items))}
	if operation == "cancel" {
		details = append(details, "not affected by cancel")
	}
	if flagVerbose {
		details = append(details, formatProcessInstanceDryRunSelectedFinalState(items))
	} else {
		details = append(details, "use --verbose to list keys")
	}
	renderHumanLine(cmd, "selected process instances already in final state: %d (%s)", len(items), strings.Join(details, "; "))
}

// printProcessInstanceDryRunRequiresCancelBeforeDelete explains delete targets that must be canceled first.
func printProcessInstanceDryRunRequiresCancelBeforeDelete(cmd *cobra.Command, operation string, items []processInstanceDryRunRequiresCancelBeforeDelete) {
	if operation != "delete" || len(items) == 0 {
		return
	}

	details := []string{fmt.Sprintf("states: %s", formatProcessInstanceDryRunRequiresCancelBeforeDeleteStates(items))}
	if flagForce {
		details = append(details, "--force would cancel them before delete")
	} else {
		details = append(details, "delete cannot remove them directly")
		details = append(details, "use --force to cancel before delete")
	}
	if flagVerbose {
		details = append(details, formatProcessInstanceDryRunRequiresCancelBeforeDelete(items))
	} else {
		details = append(details, "use --verbose to list keys")
	}
	renderHumanLine(cmd, "process instances not in final state: %d (%s)", len(items), strings.Join(details, "; "))
}

// formatProcessInstanceDryRunRequiresCancelBeforeDeleteStates formats unique non-final states for delete blockers.
func formatProcessInstanceDryRunRequiresCancelBeforeDeleteStates(items []processInstanceDryRunRequiresCancelBeforeDelete) string {
	seen := make(map[process.State]struct{}, len(items))
	states := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.State]; ok {
			continue
		}
		seen[item.State] = struct{}{}
		states = append(states, item.State.String())
	}
	return strings.Join(states, ", ")
}

// formatProcessInstanceDryRunRequiresCancelBeforeDelete formats delete blockers as key/state pairs.
func formatProcessInstanceDryRunRequiresCancelBeforeDelete(items []processInstanceDryRunRequiresCancelBeforeDelete) string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strings.TrimSpace(fmt.Sprintf("%s=%s %s", item.Key, item.State, processInstanceBusinessIDColumn(item.BusinessId))))
	}
	return strings.Join(out, ", ")
}

// formatProcessInstanceDryRunSelectedFinalStateStates formats unique terminal states for selected instances.
func formatProcessInstanceDryRunSelectedFinalStateStates(items []processInstanceDryRunSelectedFinalState) string {
	seen := make(map[process.State]struct{}, len(items))
	states := make([]string, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item.State]; ok {
			continue
		}
		seen[item.State] = struct{}{}
		states = append(states, item.State.String())
	}
	return strings.Join(states, ", ")
}

// formatProcessInstanceDryRunSelectedFinalState formats terminal selected instances as key/state pairs.
func formatProcessInstanceDryRunSelectedFinalState(items []processInstanceDryRunSelectedFinalState) string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, strings.TrimSpace(fmt.Sprintf("%s=%s %s", item.Key, item.State, processInstanceBusinessIDColumn(item.BusinessId))))
	}
	return strings.Join(out, ", ")
}

// printProcessInstanceDryRunScope writes the traversal completeness line for dry-run output.
func printProcessInstanceDryRunScope(cmd *cobra.Command, outcome process.TraversalOutcome, warning string, missingAncestors []processInstanceDryRunMissingAncestor) {
	renderHumanLine(cmd, "process-instance family scope: %s", formatProcessInstanceDryRunScope(outcome, warning, missingAncestors))
}

// formatProcessInstanceDryRunScope formats traversal completeness and partial-scope details.
func formatProcessInstanceDryRunScope(outcome process.TraversalOutcome, warning string, missingAncestors []processInstanceDryRunMissingAncestor) string {
	if outcome != process.TraversalOutcomePartial || (warning == "" && len(missingAncestors) == 0) {
		if outcome == process.TraversalOutcomeComplete {
			return "complete (all related process instances were found)"
		}
		return string(outcome)
	}

	if warning == "" {
		warning = "one or more parent process instances were not found"
	}

	details := []string{warning}
	if len(missingAncestors) > 0 {
		details = append(details, formatProcessInstanceDryRunMissingAncestors(missingAncestors))
	}
	return fmt.Sprintf("%s (%s)", outcome, strings.Join(details, "; "))
}

// formatProcessInstanceDryRunMissingAncestors summarizes or lists missing ancestor keys based on verbosity.
func formatProcessInstanceDryRunMissingAncestors(missingAncestors []processInstanceDryRunMissingAncestor) string {
	keys := make([]string, 0, len(missingAncestors))
	for _, missingAncestor := range missingAncestors {
		keys = append(keys, missingAncestor.Key)
	}
	if flagVerbose {
		return fmt.Sprintf("missing ancestor keys: %s", strings.Join(keys, ", "))
	}
	return fmt.Sprintf("missing ancestor keys: %d; use --verbose to list keys", len(keys))
}

// businessIDCancellationResult adds reviewed scope to the existing cancellation items payload.
type businessIDCancellationResult struct {
	process.CancelReports
	Scope processInstanceDryRunPreview `json:"scope"`
}

// selectedMutationPreview preserves explicit incomplete evidence instead of the legacy empty-outcome default.
func selectedMutationPreview(plan process.ProcessInstanceSelectedMutationPlan) processInstanceDryRunPreview {
	preview := newProcessInstanceDryRunPreview("cancel", plan.Selection.Keys, plan.Expansion)
	preview.Selection = &plan.Selection
	preview.ScopeComplete = plan.Expansion.Outcome == process.TraversalOutcomeComplete && (plan.Selection.Completion == process.ProcessInstanceSearchCompletionExhausted || plan.Selection.Completion == process.ProcessInstanceSearchCompletionLimitReached)
	return preview
}

// renderBusinessIDCancellation renders full frozen scope or partial execution once, with no backend reads.
func renderBusinessIDCancellation(cmd *cobra.Command, plan process.ProcessInstanceSelectedMutationPlan, reports process.CancelReports, previewOnly bool, operationErr error) error {
	preview := selectedMutationPreview(plan)
	submitted := len(domain.ProcessInstanceMutationFailures(operationErr)) > 0
	for _, report := range reports.Items {
		if report.Ok && report.StatusCode == http.StatusNoContent {
			submitted = true
		}
	}
	preview.MutationSubmitted = !previewOnly && submitted
	payload := any(businessIDCancellationResult{CancelReports: reports, Scope: preview})
	if previewOnly {
		payload = preview
	}
	if pickMode() == RenderModeJSON {
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = payload
			return renderResultEnvelope(cmd, envelope)
		}
		if previewOnly || !preview.MutationSubmitted {
			return renderSucceededResult(cmd, payload)
		}
		return renderCommandResult(cmd, payload)
	}
	if previewOnly {
		if operationErr != nil {
			return nil
		}
		return renderProcessInstanceDryRunPreview(cmd, preview)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, report := range reports.Items {
			if report.Ok {
				renderOutputLine(cmd, "%s", report.Key)
			}
		}
		return nil
	}
	renderBusinessIDSelectionScope(cmd, plan.Selection)
	renderHumanLine(cmd, "selected process instances: %d; roots: %d; affected process instances: %d", len(plan.Selection.Keys), len(plan.Expansion.Roots), len(plan.Expansion.Collected))
	renderHumanLine(cmd, "dependency expansion: %d additional process instance(s) due to dependencies", max(0, len(plan.Expansion.Collected)-len(plan.Selection.Keys)))
	if operationErr == nil && !preview.MutationSubmitted {
		renderHumanLine(cmd, "cancellation: already terminal; no mutation submitted")
	} else if operationErr == nil {
		renderProcessInstanceMutationResultSummary(cmd, "cancel", toolx.MapSlice(reports.Items, func(r process.CancelReport) process.Reporter { return process.Reporter(r) }), processInstancePageImpact{Requested: len(plan.Selection.Keys), Roots: len(plan.Expansion.Roots), Affected: len(plan.Expansion.Collected)})
	}
	if flagVerbose {
		printProcessInstanceDryRunKeys(cmd, "selected process-instance keys", preview.RequestedKeys)
		printProcessInstanceDryRunKeys(cmd, "root process-instance tree keys", preview.ResolvedRoots)
		printProcessInstanceDryRunKeys(cmd, "in-scope process-instance keys", preview.AffectedFamilyKeys)
		for _, report := range reports.Items {
			renderHumanLine(cmd, "cancellation root %s: successful=%t", report.Key, report.Ok)
		}
	}
	return nil
}

// renderBusinessIDSelectionScope quotes opaque criteria and distinguishes bounded seeds from exhaustive discovery.
func renderBusinessIDSelectionScope(cmd *cobra.Command, selection process.ProcessInstanceSelection) {
	if bid := selection.Request.SearchRequest.Filter.BusinessId; bid != nil {
		renderHumanLine(cmd, "selection business ID: %q", *bid)
	}
	if selection.Completion == process.ProcessInstanceSearchCompletionLimitReached {
		renderHumanLine(cmd, "discovery user-limited: limit %d; pages %d", selection.Request.SearchRequest.Limit, selection.Pages)
	}
}

// formatBusinessIDCancellationFailure groups all failures into one bounded diagnostic without raw backend chains.
func formatBusinessIDCancellationFailure(plan process.ProcessInstanceSelectedMutationPlan, reports process.CancelReports, err error) string {
	_, ok, _ := reports.Totals()
	message := fmt.Sprintf("cancel process instances: selected %d, roots %d, successful %d; %s", len(plan.Selection.Keys), len(plan.Expansion.Roots), ok, boundedCommandFailureReason(err))
	if n := len(domain.ProcessInstanceMutationFailures(err)); n > 0 {
		message += fmt.Sprintf("; cancellation submitted for %d root(s), outcome unconfirmed", n)
	}
	return message
}
