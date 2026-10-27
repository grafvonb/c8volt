// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// processInstanceSuspensionTreePayload keeps discovery observations separate from
// submission evidence without changing the default exact-key payload.
type processInstanceSuspensionTreePayload struct {
	Items []process.SuspensionResult            `json:"items"`
	Plan  process.ProcessInstanceSuspensionPlan `json:"plan"`
}

// suspensionTreeCounts projects classifications already resolved by the service.
func suspensionTreeCounts(plan process.ProcessInstanceSuspensionPlan) (suspended, excluded int) {
	for _, item := range plan.Instances {
		switch item.Classification {
		case process.ProcessInstanceSuspensionPlanAlreadySuspended:
			suspended++
		case process.ProcessInstanceSuspensionPlanExcluded:
			excluded++
		}
	}
	return
}

// previewProcessInstanceSuspensionTree creates only eligible preview entries,
// never representing observed states as accepted or confirmed mutations.
func previewProcessInstanceSuspensionTree(plan process.ProcessInstanceSuspensionPlan) process.SuspensionResults {
	items := make([]process.SuspensionResult, len(plan.TargetKeys))
	for i, key := range plan.TargetKeys {
		items[i] = process.SuspensionResult{Key: key, Status: process.SuspensionStatusPreview, Submission: process.SuspensionSubmissionNotSubmitted, Confirmation: process.SuspensionConfirmationNotRequested}
	}
	return process.SuspensionResults{Items: items}
}

// renderProcessInstanceSuspensionTreeApproval shows the frozen decisive counts
// only when human informational output is selected.
func renderProcessInstanceSuspensionTreeApproval(cmd *cobra.Command, plan process.ProcessInstanceSuspensionPlan) {
	if flagQuiet || pickMode() == RenderModeJSON || pickMode() == RenderModeKeysOnly {
		return
	}
	suspended, excluded := suspensionTreeCounts(plan)
	renderHumanLine(cmd, "plan: suspend %d process instance(s); %d selected; %d in scope; %d already suspended; %d excluded", len(plan.TargetKeys), len(plan.RequestedKeys), len(plan.Instances), suspended, excluded)
}

// renderProcessInstanceSuspensionTree preserves one envelope and the original
// failure precedence, including accepted-but-unconfirmed partial results.
func renderProcessInstanceSuspensionTree(cmd *cobra.Command, plan process.ProcessInstanceSuspensionPlan, results process.SuspensionResults, preview bool, operationErr error) error {
	if results.Items == nil {
		results.Items = []process.SuspensionResult{}
	}
	accepted, confirmed, failed, skipped := suspensionCounts(results)
	switch pickMode() {
	case RenderModeJSON:
		payload := processInstanceSuspensionTreePayload{Items: results.Items, Plan: plan}
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = payload
			return renderResultEnvelope(cmd, envelope)
		}
		if !preview && flagNoWait && accepted > 0 {
			return renderAcceptedResult(cmd, payload)
		}
		return renderSucceededResult(cmd, payload)
	case RenderModeKeysOnly:
		return renderProcessInstanceSuspension(cmd, results, preview, operationErr)
	default:
		if flagQuiet {
			return nil
		}
		suspended, excluded := suspensionTreeCounts(plan)
		if preview {
			renderHumanLine(cmd, "dry run: suspend process-instance tree")
		}
		renderHumanLine(cmd, "selected process instances: %d; process instances in scope: %d; active targets: %d; already suspended: %d; excluded: %d", len(plan.RequestedKeys), len(plan.Instances), len(plan.TargetKeys), suspended, excluded)
		switch {
		case preview:
			renderHumanLine(cmd, "outcome: preview; no changes applied")
		case len(plan.TargetKeys) == 0:
			renderHumanLine(cmd, "suspension: no active process instances to suspend; no changes applied")
		default:
			renderHumanLine(cmd, "%s", formatProcessInstanceSuspensionSummary(len(results.Items), accepted, confirmed, failed, skipped))
		}
		if flagVerbose {
			for _, item := range plan.Instances {
				renderHumanLine(cmd, "%s", formatProcessInstanceSuspensionPlanItem(item))
			}
			if !preview {
				for _, item := range results.Items {
					renderHumanLine(cmd, "%s", formatProcessInstanceSuspensionOutcome(item))
				}
			}
		}
	}
	return nil
}

// formatProcessInstanceSuspensionPlanItem allowlists planning facts for verbose
// diagnostics, so neither unknown states nor classifications carry backend text.
func formatProcessInstanceSuspensionPlanItem(item process.ProcessInstanceSuspensionPlanItem) string {
	state, classification := "UNKNOWN", "unknown"
	switch item.ObservedState {
	case process.StateActive, process.StateSuspended, process.StateCompleted, process.StateCanceled, process.StateTerminated, process.StateAbsent:
		state = string(item.ObservedState)
	}
	switch item.Classification {
	case process.ProcessInstanceSuspensionPlanEligible, process.ProcessInstanceSuspensionPlanAlreadySuspended, process.ProcessInstanceSuspensionPlanExcluded:
		classification = string(item.Classification)
	}
	return fmt.Sprintf("%s: observed state: %s; classification: %s", item.Key, state, classification)
}
