// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// businessIDResolutionResult adds optional reviewed work without changing empty resolution payloads.
type businessIDResolutionResult struct {
	incident.ProcessInstanceResolutionResults
	Plan process.SelectedProcessInstanceResolutionPlan `json:"plan"`
}

// businessIDResolutionIncidentCount counts advertised active work without backend reads.
func businessIDResolutionIncidentCount(plan process.SelectedProcessInstanceResolutionPlan) int {
	n := 0
	for _, item := range plan.Incidents.Incidents {
		switch strings.ToUpper(strings.TrimSpace(item.State)) {
		case "", "ACTIVE", "PENDING":
			n++
		}
	}
	return n
}

// renderBusinessIDResolution retains machine evidence and emits only bounded facts in human diagnostics.
func renderBusinessIDResolution(cmd *cobra.Command, plan process.SelectedProcessInstanceResolutionPlan, result incident.ProcessInstanceResolutionResults, operationErr error) error {
	payload := any(businessIDResolutionResult{ProcessInstanceResolutionResults: result, Plan: plan})
	if operationErr == nil && len(plan.Scope.Selection.Keys) == 0 {
		payload = result
	}
	if pickMode() == RenderModeJSON {
		if operationErr != nil {
			e := resultEnvelopeForError(cmd, operationErr)
			e.Payload = payload
			return renderResultEnvelope(cmd, e)
		}
		if result.DryRun || !result.MutationSubmitted {
			return renderSucceededResult(cmd, payload)
		}
		return renderCommandResult(cmd, payload)
	}
	if pickMode() == RenderModeKeysOnly {
		for _, item := range result.Items {
			if item.Status == incident.ProcessInstanceResolutionStatusPlanned || item.Status == incident.ProcessInstanceResolutionStatusConfirmed || item.Status == incident.ProcessInstanceResolutionStatusSubmitted {
				renderOutputLine(cmd, "%s", item.ProcessInstanceKey)
			}
		}
		return nil
	}
	if operationErr != nil && result.Total == 0 {
		return nil
	}
	if len(plan.Scope.Selection.Keys) > 0 {
		renderBusinessIDSelectionScope(cmd, plan.Scope.Selection)
		renderHumanLine(cmd, "selected process instances: %d; roots: %d; affected process instances: %d; planned active incidents: %d", len(plan.Scope.Selection.Keys), len(plan.Scope.Expansion.Roots), len(plan.Scope.Expansion.Collected), businessIDResolutionIncidentCount(plan))
	}
	if result.DryRun {
		renderHumanLine(cmd, "dry run: resolve process-instances: %d target(s) checked, %d ready or already skipped, %d failed; no changes applied", result.Total, result.Total-result.Failed, result.Failed)
	} else {
		renderHumanLine(cmd, "resolved process-instances: %d (confirmed/submitted/skipped: %d, failed: %d)", result.Total, result.Total-result.Failed, result.Failed)
	}
	if flagVerbose {
		printProcessInstanceDryRunKeys(cmd, "selected process-instance keys", plan.Scope.Selection.Keys)
		printProcessInstanceDryRunKeys(cmd, "in-scope process-instance keys", plan.Scope.Expansion.Collected)
		for _, item := range plan.Incidents.Incidents {
			renderHumanLine(cmd, "reviewed incident %q: process instance %q; state %q", item.IncidentKey, item.ProcessInstanceKey, item.State)
		}
		for _, item := range result.Items {
			renderHumanLine(cmd, "resolution process instance %q: %s; submitted=%t; resolved=%d; failed=%d", item.ProcessInstanceKey, item.Status, item.MutationSubmitted, len(item.ResolvedIncidentKeys), len(item.FailedIncidentKeys))
		}
	}
	return nil
}
