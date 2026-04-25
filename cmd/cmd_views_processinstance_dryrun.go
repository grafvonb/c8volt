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

type processInstanceDryRunPreview struct {
	Operation          string                                 `json:"operation"`
	RequestedKeys      []string                               `json:"requestedKeys"`
	ResolvedRoots      []string                               `json:"resolvedRoots"`
	AffectedFamilyKeys []string                               `json:"affectedFamilyKeys"`
	RequestedCount     int                                    `json:"requestedCount"`
	ResolvedRootCount  int                                    `json:"resolvedRootCount"`
	AffectedCount      int                                    `json:"affectedCount"`
	TraversalOutcome   process.TraversalOutcome               `json:"traversalOutcome"`
	ScopeComplete      bool                                   `json:"scopeComplete"`
	Warning            string                                 `json:"warning"`
	MissingAncestors   []processInstanceDryRunMissingAncestor `json:"missingAncestors"`
	MutationSubmitted  bool                                   `json:"mutationSubmitted"`
}

type processInstanceDryRunSummary struct {
	Operation         string                         `json:"operation"`
	RequestedCount    int                            `json:"requestedCount"`
	ResolvedRootCount int                            `json:"resolvedRootCount"`
	AffectedCount     int                            `json:"affectedCount"`
	TraversalOutcome  process.TraversalOutcome       `json:"traversalOutcome"`
	ScopeComplete     bool                           `json:"scopeComplete"`
	Previews          []processInstanceDryRunPreview `json:"previews"`
	MutationSubmitted bool                           `json:"mutationSubmitted"`
}

type processInstanceDryRunPlanResult struct {
	Plan    process.DryRunPIKeyExpansion
	Impact  processInstancePageImpact
	Preview processInstanceDryRunPreview
}

func planProcessInstanceDryRunPreview(cli process.API, operation string, keys types.Keys) (processInstanceDryRunPlanResult, error) {
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

func newProcessInstanceDryRunPreview(operation string, requested types.Keys, plan process.DryRunPIKeyExpansion) processInstanceDryRunPreview {
	outcome := processInstanceDryRunTraversalOutcome(plan)
	return processInstanceDryRunPreview{
		Operation:          operation,
		RequestedKeys:      append([]string(nil), requested...),
		ResolvedRoots:      append([]string(nil), plan.Roots...),
		AffectedFamilyKeys: append([]string(nil), plan.Collected...),
		RequestedCount:     len(requested),
		ResolvedRootCount:  len(plan.Roots),
		AffectedCount:      len(plan.Collected),
		TraversalOutcome:   outcome,
		ScopeComplete:      outcome == process.TraversalOutcomeComplete,
		Warning:            plan.Warning,
		MissingAncestors:   newProcessInstanceDryRunMissingAncestors(plan.MissingAncestors),
		MutationSubmitted:  false,
	}
}

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
		if !preview.ScopeComplete {
			summary.TraversalOutcome = process.TraversalOutcomePartial
			summary.ScopeComplete = false
		}
	}
	return summary
}

func processInstanceDryRunTraversalOutcome(plan process.DryRunPIKeyExpansion) process.TraversalOutcome {
	if plan.Outcome != "" {
		return plan.Outcome
	}
	if plan.Warning != "" || len(plan.MissingAncestors) > 0 {
		return process.TraversalOutcomePartial
	}
	return process.TraversalOutcomeComplete
}

func newProcessInstanceDryRunMissingAncestors(items []process.MissingAncestor) []processInstanceDryRunMissingAncestor {
	out := make([]processInstanceDryRunMissingAncestor, 0, len(items))
	for _, item := range items {
		out = append(out, processInstanceDryRunMissingAncestor{Key: item.Key, StartKey: item.StartKey})
	}
	return out
}

func renderProcessInstanceDryRunPreview(cmd *cobra.Command, preview processInstanceDryRunPreview) error {
	if pickMode() == RenderModeJSON {
		return renderProcessInstanceDryRunResult(cmd, preview)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, key := range preview.AffectedFamilyKeys {
			cmd.Println(key)
		}
		return nil
	}

	cmd.Printf("dry run: %s process-instance\n", preview.Operation)
	cmd.Printf("requested process instances: %d\n", preview.RequestedCount)
	cmd.Printf("resolved root process instances: %d\n", preview.ResolvedRootCount)
	cmd.Printf("affected process instances: %d\n", preview.AffectedCount)
	cmd.Printf("scope: %s\n", preview.TraversalOutcome)
	printProcessInstanceDryRunKeys(cmd, "requested keys", preview.RequestedKeys)
	printProcessInstanceDryRunKeys(cmd, "resolved root keys", preview.ResolvedRoots)
	printProcessInstanceDryRunKeys(cmd, "affected family keys", preview.AffectedFamilyKeys)
	printProcessInstanceDryRunWarning(cmd, preview.Warning, preview.MissingAncestors)
	cmd.Printf("no mutation submitted: %s was not submitted\n", preview.Operation)
	return nil
}

func renderProcessInstanceDryRunSummary(cmd *cobra.Command, summary processInstanceDryRunSummary) error {
	if pickMode() == RenderModeJSON {
		return renderProcessInstanceDryRunResult(cmd, summary)
	}

	cmd.Printf("dry run: %s process-instance\n", summary.Operation)
	cmd.Printf("requested process instances: %d\n", summary.RequestedCount)
	cmd.Printf("resolved root process instances: %d\n", summary.ResolvedRootCount)
	cmd.Printf("affected process instances: %d\n", summary.AffectedCount)
	cmd.Printf("scope: %s\n", summary.TraversalOutcome)
	cmd.Printf("preview pages: %d\n", len(summary.Previews))
	cmd.Printf("no mutation submitted: %s was not submitted\n", summary.Operation)
	return nil
}

func renderProcessInstanceDryRunResult[T any](cmd *cobra.Command, payload T) error {
	if !commandUsesSharedEnvelope(cmd, pickMode()) {
		return nil
	}
	return renderSucceededResult(cmd, payload)
}

func printProcessInstanceDryRunKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		cmd.Printf("%s: none\n", label)
		return
	}
	cmd.Printf("%s: %s\n", label, strings.Join(keys, ", "))
}

func printProcessInstanceDryRunWarning(cmd *cobra.Command, warning string, missingAncestors []processInstanceDryRunMissingAncestor) {
	if warning != "" {
		cmd.Printf("warning: %s\n", warning)
	}
	if len(missingAncestors) == 0 {
		return
	}
	keys := make([]string, 0, len(missingAncestors))
	for _, item := range missingAncestors {
		keys = append(keys, item.Key)
	}
	cmd.Printf("missing ancestor keys: %s\n", strings.Join(keys, ", "))
}
