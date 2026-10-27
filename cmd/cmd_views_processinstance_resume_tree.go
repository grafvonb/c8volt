// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// processInstanceResumptionTreePayload keeps discovery observations separate from
// submission evidence without changing the default exact-key payload.
type processInstanceResumptionTreePayload struct {
	Items []process.ResumptionResult            `json:"items"`
	Plan  process.ProcessInstanceResumptionPlan `json:"plan"`
}

// previewProcessInstanceResumptionTree creates only eligible preview entries,
// never representing observed states as accepted or confirmed mutations.
func previewProcessInstanceResumptionTree(plan process.ProcessInstanceResumptionPlan) process.ResumptionResults {
	items := make([]process.ResumptionResult, len(plan.TargetKeys))
	for i, key := range plan.TargetKeys {
		items[i] = process.ResumptionResult{Key: key, Status: process.ResumptionStatusPreview, Submission: process.ResumptionSubmissionNotSubmitted, Confirmation: process.ResumptionConfirmationNotRequested}
	}
	return process.ResumptionResults{Items: items}
}

// renderProcessInstanceResumptionTreeApproval shows the frozen decisive counts
// only when human informational output is selected.
func renderProcessInstanceResumptionTreeApproval(cmd *cobra.Command, plan process.ProcessInstanceResumptionPlan) {
	if flagQuiet || pickMode() == RenderModeJSON || pickMode() == RenderModeKeysOnly {
		return
	}
	excluded := len(plan.Instances) - len(plan.TargetKeys)
	renderHumanLine(cmd, "plan: resume %d process instance(s); %d selected; %d in scope; %d excluded", len(plan.TargetKeys), len(plan.RequestedKeys), len(plan.Instances), excluded)
}

// renderProcessInstanceResumptionTree preserves one envelope and the original
// failure precedence, including accepted-but-unconfirmed partial results.
func renderProcessInstanceResumptionTree(cmd *cobra.Command, plan process.ProcessInstanceResumptionPlan, results process.ResumptionResults, preview bool, operationErr error) error {
	if results.Items == nil {
		results.Items = []process.ResumptionResult{}
	}
	if plan.RequestedKeys == nil {
		plan.RequestedKeys = []string{}
	}
	if plan.Instances == nil {
		plan.Instances = []process.ProcessInstanceResumptionPlanItem{}
	}
	if plan.TargetKeys == nil {
		plan.TargetKeys = []string{}
	}
	accepted, confirmed, failed, skipped := resumptionCounts(results)
	switch pickMode() {
	case RenderModeJSON:
		payload := processInstanceResumptionTreePayload{Items: results.Items, Plan: plan}
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
		return renderProcessInstanceResumption(cmd, results, preview, operationErr)
	default:
		if flagQuiet {
			return nil
		}
		excluded := len(plan.Instances) - len(plan.TargetKeys)
		if preview {
			renderHumanLine(cmd, "dry run: resume process-instance tree")
		}
		renderHumanLine(cmd, "selected process instances: %d; process instances in scope: %d; suspended targets: %d; excluded: %d", len(plan.RequestedKeys), len(plan.Instances), len(plan.TargetKeys), excluded)
		switch {
		case preview:
			renderHumanLine(cmd, "outcome: preview; no changes applied")
		case len(plan.TargetKeys) == 0:
			renderHumanLine(cmd, "resumption: no suspended process instances to resume; no changes applied")
		default:
			renderHumanLine(cmd, "%s", formatProcessInstanceResumptionSummary(len(results.Items), accepted, confirmed, failed, skipped))
		}
		if flagVerbose {
			for _, item := range plan.Instances {
				renderHumanLine(cmd, "%s", formatProcessInstanceResumptionPlanItem(item))
			}
			if !preview {
				for _, item := range results.Items {
					renderHumanLine(cmd, "%s", formatProcessInstanceResumptionOutcome(item))
				}
			}
		}
	}
	return nil
}

// formatProcessInstanceResumptionPlanItem allowlists planning facts for verbose
// diagnostics, so neither unknown states nor classifications carry backend text.
func formatProcessInstanceResumptionPlanItem(item process.ProcessInstanceResumptionPlanItem) string {
	state, classification := "UNKNOWN", "unknown"
	switch item.ObservedState {
	case process.StateActive, process.StateSuspended, process.StateCompleted, process.StateCanceled, process.StateTerminated, process.StateAbsent:
		state = string(item.ObservedState)
	}
	switch item.Classification {
	case process.ProcessInstanceResumptionPlanEligible, process.ProcessInstanceResumptionPlanExcluded:
		classification = string(item.Classification)
	}
	return fmt.Sprintf("%s: observed state: %s; classification: %s", item.Key, state, classification)
}
