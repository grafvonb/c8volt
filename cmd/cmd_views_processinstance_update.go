// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	types "github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

type processInstanceVariablePlannedValue struct {
	Name         string `json:"name"`
	Value        any    `json:"value"`
	APITruncated bool   `json:"apiTruncated,omitempty"`
}

type processInstanceVariablePlannedChange struct {
	Name         string `json:"name"`
	Before       any    `json:"before"`
	After        any    `json:"after"`
	APITruncated bool   `json:"apiTruncated,omitempty"`
}

type processInstanceVariableUpdatePlan struct {
	ProcessInstanceKey string                                 `json:"processInstanceKey"`
	Additions          []processInstanceVariablePlannedValue  `json:"additions"`
	Changes            []processInstanceVariablePlannedChange `json:"changes"`
	UnchangedRequested []processInstanceVariablePlannedValue  `json:"unchangedRequested"`
	Untouched          []processInstanceVariablePlannedValue  `json:"untouched"`
}

type processInstanceVariableUpdatePreview struct {
	Operation              string                              `json:"operation"`
	RequestedKeys          []string                            `json:"requestedKeys"`
	RequestedCount         int                                 `json:"requestedCount"`
	UpdateCount            int                                 `json:"updateCount"`
	VariableAddCount       int                                 `json:"variableAddCount"`
	VariableChangeCount    int                                 `json:"variableChangeCount"`
	VariableUnchangedCount int                                 `json:"variableUnchangedCount"`
	VariableUntouchedCount int                                 `json:"variableUntouchedCount"`
	ProcessInstances       []processInstanceVariableUpdatePlan `json:"processInstances,omitempty"`
	MutationSubmitted      bool                                `json:"mutationSubmitted"`
}

// newProcessInstanceVariableUpdatePreview aggregates per-instance variable plans into the command payload contract.
func newProcessInstanceVariableUpdatePreview(requestedKeys types.Keys, plans []processInstanceVariableUpdatePlan) processInstanceVariableUpdatePreview {
	preview := processInstanceVariableUpdatePreview{
		Operation:         "update",
		RequestedKeys:     append([]string(nil), requestedKeys...),
		RequestedCount:    len(requestedKeys),
		ProcessInstances:  append([]processInstanceVariableUpdatePlan(nil), plans...),
		MutationSubmitted: false,
	}
	for _, plan := range plans {
		if plan.HasPlannedChanges() {
			preview.UpdateCount++
		}
		preview.VariableAddCount += len(plan.Additions)
		preview.VariableChangeCount += len(plan.Changes)
		preview.VariableUnchangedCount += len(plan.UnchangedRequested)
		preview.VariableUntouchedCount += len(plan.Untouched)
	}
	return preview
}

// HasPlannedChanges reports whether the plan would submit any variable mutation.
func (p processInstanceVariableUpdatePlan) HasPlannedChanges() bool {
	return len(p.Additions)+len(p.Changes) > 0
}

// HasPlannedChanges reports whether any selected process instance needs a variable mutation.
func (p processInstanceVariableUpdatePreview) HasPlannedChanges() bool {
	return p.VariableAddCount+p.VariableChangeCount > 0
}

// renderUpdateProcessInstanceVariablePreview renders a variable update plan without submitting mutation results.
func renderUpdateProcessInstanceVariablePreview(cmd *cobra.Command, preview processInstanceVariableUpdatePreview) error {
	if pickMode() == RenderModeJSON {
		return renderProcessInstanceDryRunResult(cmd, preview)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, plan := range preview.ProcessInstances {
			renderOutputLine(cmd, "%s", plan.ProcessInstanceKey)
		}
		return nil
	}

	renderProcessInstanceVariableUpdatePlanHuman(cmd, preview, "dry run")
	return nil
}

// renderUpdateProcessInstanceVariablePlan renders the pre-mutation confirmation plan in human form.
func renderUpdateProcessInstanceVariablePlan(cmd *cobra.Command, preview processInstanceVariableUpdatePreview) error {
	renderProcessInstanceVariableUpdatePlanHuman(cmd, preview, "plan")
	return nil
}

func renderProcessInstanceVariableUpdatePlanHuman(cmd *cobra.Command, preview processInstanceVariableUpdatePreview, label string) {
	status := processInstanceVariableUpdatePlanHumanStatus(preview, label)
	if !preview.HasPlannedChanges() {
		renderHumanLine(cmd, "%s: update process-instance variables: nothing to update (%d requested value(s) already match visible variables); %s", label, preview.VariableUnchangedCount, status)
		return
	}
	summary := fmt.Sprintf("%s: update process-instance variables: %d process instance(s), %d change(s), %d addition(s), %d unchanged, %d untouched",
		label, preview.UpdateCount, preview.VariableChangeCount, preview.VariableAddCount, preview.VariableUnchangedCount, preview.VariableUntouchedCount)
	if status != "" {
		summary += "; " + status
	}
	renderHumanLine(cmd, "%s", summary)
	if flagVerbose || (preview.UpdateCount == 1 && len(preview.ProcessInstances) == 1) {
		for _, plan := range preview.ProcessInstances {
			if !plan.HasPlannedChanges() && !flagVerbose {
				continue
			}
			renderHumanLine(cmd, "%s: %s", plan.ProcessInstanceKey, formatProcessInstanceVariableUpdatePlan(plan))
		}
	}
}

func processInstanceVariableUpdatePlanHumanStatus(preview processInstanceVariableUpdatePreview, label string) string {
	if label == "dry run" {
		return "no changes applied"
	}
	if !preview.HasPlannedChanges() {
		return "no confirmation required"
	}
	return ""
}

// renderUpdateProcessInstanceVariableResults renders update outcomes through the command view contract.
func renderUpdateProcessInstanceVariableResults(cmd *cobra.Command, results process.ProcessInstanceVariableUpdateResults) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, results)
	}
	for _, item := range results.Items {
		switch item.Status {
		case process.ProcessInstanceVariableUpdateStatusConfirmed:
			renderHumanLine(cmd, "updated process-instance %s: confirmed", item.Key)
		case process.ProcessInstanceVariableUpdateStatusSubmitted:
			renderHumanLine(cmd, "updated process-instance %s: submitted", item.Key)
		case process.ProcessInstanceVariableUpdateStatusConfirmationFailed:
			renderHumanLine(cmd, "updated process-instance %s: confirmation failed: %s", item.Key, item.Error)
		case process.ProcessInstanceVariableUpdateStatusMutationFailed:
			renderHumanLine(cmd, "updated process-instance %s: mutation failed: %s", item.Key, item.Error)
		default:
			renderHumanLine(cmd, "updated process-instance %s: %s", item.Key, item.Status)
		}
	}
	total, ok, failed := results.Totals()
	renderHumanLine(cmd, "updated: %d (confirmed/submitted: %d, failed: %d)", total, ok, failed)
	if failed > 0 {
		return fmt.Errorf("one or more process-instance variable updates failed")
	}
	return nil
}

func formatProcessInstanceVariableUpdatePlan(plan processInstanceVariableUpdatePlan) string {
	parts := make([]string, 0, len(plan.Additions)+len(plan.Changes)+len(plan.UnchangedRequested)+1)
	for _, item := range plan.Changes {
		parts = append(parts, fmt.Sprintf("~ %s: %s -> %s", item.Name, formatProcessInstanceVariablePlanValue(item.Before), formatProcessInstanceVariablePlanValue(item.After)))
	}
	for _, item := range plan.Additions {
		parts = append(parts, fmt.Sprintf("+ %s: %s", item.Name, formatProcessInstanceVariablePlanValue(item.Value)))
	}
	for _, item := range plan.UnchangedRequested {
		parts = append(parts, fmt.Sprintf("~ %s: %s (unchanged)", item.Name, formatProcessInstanceVariablePlanValue(item.Value)))
	}
	if len(plan.Untouched) > 0 {
		parts = append(parts, "= "+formatProcessInstanceVariableUntouchedPlan(plan.Untouched))
	}
	if len(parts) == 0 {
		return "no variable changes"
	}
	return strings.Join(parts, "; ")
}

func formatProcessInstanceVariableUntouchedPlan(items []processInstanceVariablePlannedValue) string {
	const compactUntouchedLimit = 5
	if !flagVerbose && len(items) > compactUntouchedLimit {
		return fmt.Sprintf("%d variable(s) left untouched; use --verbose to list", len(items))
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, fmt.Sprintf("%s: %s", item.Name, formatProcessInstanceVariablePlanValue(item.Value)))
	}
	return strings.Join(parts, ", ")
}

func formatProcessInstanceVariablePlanValue(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprint(value)
	}
	return string(data)
}
