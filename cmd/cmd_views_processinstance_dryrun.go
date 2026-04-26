package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

type processInstanceDryRunMissingAncestor struct {
	Key      string `json:"key"`
	StartKey string `json:"startKey"`
}

type processInstanceDryRunSelectedFinalState struct {
	Key   string        `json:"key"`
	State process.State `json:"state"`
}

type processInstanceDryRunRequiresCancelBeforeDelete struct {
	Key   string        `json:"key"`
	State process.State `json:"state"`
}

type processInstanceDryRunPreview struct {
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

type processInstanceDryRunPlanResult struct {
	Plan    process.DryRunPIKeyExpansion
	Impact  processInstancePageImpact
	Preview processInstanceDryRunPreview
}

// planProcessInstanceDryRunPreview builds the shared dry-run plan, impact counts, and render payload for one key batch.
func planProcessInstanceDryRunPreview(cmd *cobra.Command, cli process.API, operation string, keys types.Keys) (processInstanceDryRunPlanResult, error) {
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("preparing %s dry-run scope for %d process instance(s)", operation, len(keys)))
	defer stopActivity()

	plan, err := cli.DryRunCancelOrDeletePlan(context.Background(), keys, collectOptions()...)
	if err != nil {
		return processInstanceDryRunPlanResult{}, fmt.Errorf("%s validation: %w", operation, err)
	}

	return processInstanceDryRunPlanResult{
		Plan:    plan,
		Impact:  processInstancePageImpact{Requested: len(keys), Affected: len(plan.Collected), Roots: len(plan.Roots)},
		Preview: newProcessInstanceDryRunPreview(operation, keys, plan),
	}, nil
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
	for _, preview := range previews {
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
		out = append(out, processInstanceDryRunSelectedFinalState{Key: item.Key, State: item.State})
	}
	return out
}

// newProcessInstanceDryRunRequiresCancelBeforeDelete converts non-final delete blockers to command JSON items.
func newProcessInstanceDryRunRequiresCancelBeforeDelete(items []process.ProcessInstance) []processInstanceDryRunRequiresCancelBeforeDelete {
	out := make([]processInstanceDryRunRequiresCancelBeforeDelete, 0, len(items))
	for _, item := range items {
		out = append(out, processInstanceDryRunRequiresCancelBeforeDelete{Key: item.Key, State: item.State})
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

	renderHumanLine(cmd, "dry run: %s process-instance", preview.Operation)
	renderHumanLine(cmd, "selected process instances: %d", preview.RequestedCount)
	renderHumanLine(cmd, "process-instance trees to %s: %d", preview.Operation, preview.ResolvedRootCount)
	renderHumanLine(cmd, "process instances in scope: %d", preview.AffectedCount)
	printProcessInstanceDryRunSelectedFinalState(cmd, preview.Operation, preview.SelectedFinalState)
	printProcessInstanceDryRunRequiresCancelBeforeDelete(cmd, preview.Operation, preview.RequiresCancelBeforeDelete)
	printProcessInstanceDryRunScope(cmd, preview.TraversalOutcome, preview.Warning, preview.MissingAncestors)
	printProcessInstanceDryRunKeys(cmd, "selected process-instance keys", preview.RequestedKeys)
	printProcessInstanceDryRunKeys(cmd, "root process-instance tree keys", preview.ResolvedRoots)
	printProcessInstanceDryRunKeys(cmd, "in-scope process-instance keys", preview.AffectedFamilyKeys)
	return nil
}

// renderProcessInstanceDryRunSummary renders the aggregate dry-run result for paged searches.
func renderProcessInstanceDryRunSummary(cmd *cobra.Command, summary processInstanceDryRunSummary) error {
	if pickMode() == RenderModeJSON {
		return renderProcessInstanceDryRunResult(cmd, summary)
	}

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

// printProcessInstanceDryRunKeys writes a labeled dry-run key list for human output.
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
		out = append(out, fmt.Sprintf("%s=%s", item.Key, item.State))
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
		out = append(out, fmt.Sprintf("%s=%s", item.Key, item.State))
	}
	return strings.Join(out, ", ")
}

// printProcessInstanceDryRunScope writes the traversal completeness line for human dry-run output.
func printProcessInstanceDryRunScope(cmd *cobra.Command, outcome process.TraversalOutcome, warning string, missingAncestors []processInstanceDryRunMissingAncestor) {
	renderHumanLine(cmd, "scope: %s", formatProcessInstanceDryRunScope(outcome, warning, missingAncestors))
}

// formatProcessInstanceDryRunScope formats traversal completeness and partial-scope details.
func formatProcessInstanceDryRunScope(outcome process.TraversalOutcome, warning string, missingAncestors []processInstanceDryRunMissingAncestor) string {
	if outcome != process.TraversalOutcomePartial || (warning == "" && len(missingAncestors) == 0) {
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
