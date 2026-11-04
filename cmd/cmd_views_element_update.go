// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/spf13/cobra"
)

// elementVariablePreview omits execution-only maps while preserving the frozen classifications.
type elementVariablePreview struct {
	Operation              string                       `json:"operation"`
	RequestedKeys          []string                     `json:"requestedKeys"`
	RequestedCount         int                          `json:"requestedCount"`
	UpdateCount            int                          `json:"updateCount"`
	VariableAddCount       int                          `json:"variableAddCount"`
	VariableChangeCount    int                          `json:"variableChangeCount"`
	VariableUnchangedCount int                          `json:"variableUnchangedCount"`
	VariableUntouchedCount int                          `json:"variableUntouchedCount"`
	Elements               []elementVariablePreviewItem `json:"elements"`
	MutationSubmitted      bool                         `json:"mutationSubmitted"`
}

// elementVariablePreviewItem identifies the exact local destination, including known tenant evidence.
type elementVariablePreviewItem struct {
	ElementKey         string                                 `json:"elementKey"`
	ProcessInstanceKey string                                 `json:"processInstanceKey"`
	ScopeKey           string                                 `json:"scopeKey"`
	TenantId           string                                 `json:"tenantId,omitempty"`
	Additions          []element.ElementVariablePlannedValue  `json:"additions"`
	Changes            []element.ElementVariablePlannedChange `json:"changes"`
	UnchangedRequested []element.ElementVariablePlannedValue  `json:"unchangedRequested"`
	Untouched          []element.ElementVariablePlannedValue  `json:"untouched"`
}

// elementVariablePreviewFromPlan is a mechanical projection; authoritative counts stay service-owned.
func elementVariablePreviewFromPlan(plan element.ElementVariableUpdatePlan) elementVariablePreview {
	preview := elementVariablePreview{
		Operation:              "update",
		RequestedKeys:          append([]string{}, plan.RequestedKeys...),
		RequestedCount:         plan.RequestedCount,
		UpdateCount:            plan.UpdateCount,
		VariableAddCount:       plan.VariableAddCount,
		VariableChangeCount:    plan.VariableChangeCount,
		VariableUnchangedCount: plan.VariableUnchangedCount,
		VariableUntouchedCount: plan.VariableUntouchedCount,
		Elements:               make([]elementVariablePreviewItem, len(plan.Elements)),
	}
	for i, p := range plan.Elements {
		preview.Elements[i] = elementVariablePreviewItem{
			ElementKey: p.ElementKey, ProcessInstanceKey: p.ProcessInstanceKey,
			ScopeKey: p.ScopeKey, TenantId: p.TenantId,
			Additions:          append([]element.ElementVariablePlannedValue{}, p.Additions...),
			Changes:            append([]element.ElementVariablePlannedChange{}, p.Changes...),
			UnchangedRequested: append([]element.ElementVariablePlannedValue{}, p.UnchangedRequested...),
			Untouched:          append([]element.ElementVariablePlannedValue{}, p.Untouched...),
		}
	}
	return preview
}

// renderElementVariablePreview explicitly succeeds even with no-wait and prints only changed preview keys.
func renderElementVariablePreview(cmd *cobra.Command, plan element.ElementVariableUpdatePlan) error {
	preview := elementVariablePreviewFromPlan(plan)
	switch pickMode() {
	case RenderModeJSON:
		return renderSucceededResult(cmd, preview)
	case RenderModeKeysOnly:
		for _, item := range preview.Elements {
			if len(item.Additions)+len(item.Changes) > 0 {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.ElementKey); err != nil {
					return err
				}
			}
		}
	default:
		label := "plan"
		if flagDryRun {
			label = "dry run"
		}
		renderElementVariablePlanHuman(cmd, preview, label)
	}
	return nil
}

// renderElementVariablePlanHuman follows compact single-target/verbose preview conventions.
func renderElementVariablePlanHuman(cmd *cobra.Command, preview elementVariablePreview, label string) {
	if flagQuiet {
		return
	}
	renderAttachedTenantContext(cmd)
	if preview.UpdateCount == 0 {
		renderHumanLine(cmd, "%s: update element variables: nothing to update (%d requested value(s) already match local variables); no changes applied; no confirmation required", label, preview.VariableUnchangedCount)
		return
	}
	suffix := ""
	if label == "dry run" {
		suffix = "; no changes applied"
	}
	renderHumanLine(cmd, "%s: update element variables: %d element(s), %d change(s), %d addition(s), %d unchanged, %d untouched%s", label, preview.UpdateCount, preview.VariableChangeCount, preview.VariableAddCount, preview.VariableUnchangedCount, preview.VariableUntouchedCount, suffix)
	if flagVerbose || len(preview.Elements) == 1 {
		for _, p := range preview.Elements {
			parts := []string{}
			for _, v := range p.Changes {
				parts = append(parts, fmt.Sprintf("~ %s (element:%s): %s -> %s", v.Name, p.ElementKey, formatProcessInstanceVariablePlanValue(v.Before), formatProcessInstanceVariablePlanValue(v.After)))
			}
			for _, v := range p.Additions {
				parts = append(parts, fmt.Sprintf("+ %s (element:%s): %s", v.Name, p.ElementKey, formatProcessInstanceVariablePlanValue(v.Value)))
			}
			for _, v := range p.UnchangedRequested {
				parts = append(parts, fmt.Sprintf("~ %s: %s (unchanged)", v.Name, formatProcessInstanceVariablePlanValue(v.Value)))
			}
			if len(p.Untouched) > 0 {
				parts = append(parts, fmt.Sprintf("= %d variable(s) left untouched", len(p.Untouched)))
			}
			renderHumanLine(cmd, "%s: %s", p.ElementKey, strings.Join(parts, "; "))
		}
	}
}

// elementVariableCounts separates acceptance, confirmation and failure; accepted interrupted work is still failed.
func elementVariableCounts(results element.ElementVariableUpdateResults) (accepted, confirmed, unchanged, failed, skipped int) {
	for _, item := range results.Items {
		if item.MutationAccepted {
			accepted++
		}
		switch item.Status {
		case element.ElementVariableUpdateStatusConfirmed:
			confirmed++
		case element.ElementVariableUpdateStatusUnchanged:
			unchanged++
		case element.ElementVariableUpdateStatusSkipped:
			skipped++
		}
		if !item.OK() && item.Status != element.ElementVariableUpdateStatusSkipped {
			failed++
		}
	}
	return
}

// renderElementVariableResults chooses the failure envelope before no-wait and retains all partial evidence.
func renderElementVariableResults(cmd *cobra.Command, results element.ElementVariableUpdateResults, operationErr error) error {
	accepted, confirmed, unchanged, failed, skipped := elementVariableCounts(results)
	switch pickMode() {
	case RenderModeJSON:
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = results
			return renderResultEnvelope(cmd, envelope)
		}
		if flagNoWait && accepted > 0 {
			return renderAcceptedResult(cmd, results)
		}
		return renderSucceededResult(cmd, results)
	case RenderModeKeysOnly:
		for _, item := range results.Items {
			if item.MutationAccepted {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.Key); err != nil {
					return err
				}
			}
		}
	default:
		if flagQuiet {
			return nil
		}
		renderHumanLine(cmd, "update element variables: %d selected; accepted: %d, confirmed: %d, unchanged: %d, failed: %d, skipped: %d", len(results.Items), accepted, confirmed, unchanged, failed, skipped)
		if flagVerbose {
			for _, item := range results.Items {
				renderHumanLine(cmd, "%s", formatElementVariableOutcome(item))
			}
		}
	}
	return nil
}

// formatElementVariableOutcome allows only known lifecycle labels, never arbitrary status/error/message text.
func formatElementVariableOutcome(item element.ElementVariableUpdateResult) string {
	status := "unknown"
	submission := "unknown"
	confirmation := "unknown"
	switch item.Status {
	case element.ElementVariableUpdateStatusUnchanged, element.ElementVariableUpdateStatusSkipped, element.ElementVariableUpdateStatusMutationFailed, element.ElementVariableUpdateStatusSubmitted, element.ElementVariableUpdateStatusConfirmed, element.ElementVariableUpdateStatusConfirmationFailed:
		status = string(item.Status)
	}
	switch item.SubmissionStatus {
	case element.VariableSubmissionNotSent, element.VariableSubmissionRejected, element.VariableSubmissionUnknown, element.VariableSubmissionAccepted:
		submission = string(item.SubmissionStatus)
	}
	switch item.ConfirmationStatus {
	case "not_required", "skipped", "not_started", "confirmed", "failed":
		confirmation = item.ConfirmationStatus
	}
	return fmt.Sprintf("%s: %s; submission: %s; confirmation: %s", item.Key, status, submission, confirmation)
}

// formatElementVariableFailure groups safe reasons in a bounded single ERROR, retaining failures under quiet.
func formatElementVariableFailure(results element.ElementVariableUpdateResults, operationErr error) string {
	counts := map[string]int{}
	reasons := []string{}
	for _, item := range results.Items {
		if item.OK() {
			continue
		}
		reason := "mutation failed"
		switch {
		case item.Status == element.ElementVariableUpdateStatusSkipped:
			reason = "not attempted"
		case item.MutationAccepted && item.ConfirmationStatus == "not_started":
			reason = "confirmation not started"
		case item.MutationAccepted:
			reason = "confirmation failed"
		case item.SubmissionStatus == element.VariableSubmissionUnknown:
			reason = "acceptance unknown"
		case item.SubmissionStatus == element.VariableSubmissionRejected:
			reason = "mutation rejected"
		case item.SubmissionStatus == element.VariableSubmissionNotSent:
			reason = "mutation not sent"
		}
		if counts[reason] == 0 {
			reasons = append(reasons, reason)
		}
		counts[reason]++
	}
	if len(reasons) == 0 {
		return "element variable update: operation failed: 1; " + boundedCommandFailureReason(operationErr)
	}
	parts := []string{}
	other := 0
	for i, reason := range reasons {
		if i < 3 {
			parts = append(parts, fmt.Sprintf("%s: %d", reason, counts[reason]))
		} else {
			other += counts[reason]
		}
	}
	if other > 0 {
		parts = append(parts, fmt.Sprintf("other failures: %d", other))
	}
	return "element variable update: " + strings.Join(parts, "; ")
}
