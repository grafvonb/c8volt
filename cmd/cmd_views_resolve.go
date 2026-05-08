// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

func renderIncidentResolutionResults(cmd *cobra.Command, results process.IncidentResolutionResults) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, results)
	}
	for _, item := range results.Items {
		switch item.Status {
		case process.IncidentResolutionStatusPlanned:
			renderHumanLine(cmd, "dry run: incident %s would be resolved", item.IncidentKey)
		case process.IncidentResolutionStatusConfirmed:
			renderHumanLine(cmd, "resolved incident %s: confirmed", item.IncidentKey)
		case process.IncidentResolutionStatusSubmitted:
			renderHumanLine(cmd, "resolved incident %s: submitted", item.IncidentKey)
		case process.IncidentResolutionStatusSkipped:
			renderHumanLine(cmd, "resolved incident %s: skipped (%s)", item.IncidentKey, incidentResolutionSkipReason(item))
		case process.IncidentResolutionStatusMutationFailed:
			renderHumanLine(cmd, "resolved incident %s: mutation failed: %s", item.IncidentKey, item.Error)
		case process.IncidentResolutionStatusConfirmationFailed:
			renderHumanLine(cmd, "resolved incident %s: confirmation failed: %s", item.IncidentKey, item.Error)
		default:
			renderHumanLine(cmd, "resolved incident %s: %s", item.IncidentKey, item.Status)
		}
	}
	total, ok, failed := results.Totals()
	if results.DryRun {
		renderHumanLine(cmd, "dry run: resolve incidents: %d target(s), %d planned/skipped, %d failed; no changes applied", total, ok, failed)
		if failed > 0 {
			return fmt.Errorf("one or more incident resolution dry-run lookups failed")
		}
		return nil
	}
	renderHumanLine(cmd, "resolved: %d (confirmed/submitted/skipped: %d, failed: %d)", total, ok, failed)
	if failed > 0 {
		return fmt.Errorf("one or more incident resolutions failed")
	}
	return nil
}

func incidentResolutionSkipReason(item process.IncidentResolutionResult) string {
	if item.IncidentState != "" {
		return item.IncidentState
	}
	if item.ConfirmationStatus != "" {
		return item.ConfirmationStatus
	}
	return "not_active"
}

func renderProcessInstanceResolutionResults(cmd *cobra.Command, results process.ProcessInstanceResolutionResults) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, results)
	}
	for _, item := range results.Items {
		switch item.Status {
		case process.ProcessInstanceResolutionStatusPlanned:
			renderHumanLine(cmd, "dry run: process-instance %s would resolve %d incident(s)", item.ProcessInstanceKey, len(item.AttemptedIncidentKeys))
		case process.ProcessInstanceResolutionStatusConfirmed:
			renderHumanLine(cmd, "resolved process-instance %s: confirmed (%d incident(s))", item.ProcessInstanceKey, len(item.ResolvedIncidentKeys))
		case process.ProcessInstanceResolutionStatusSubmitted:
			renderHumanLine(cmd, "resolved process-instance %s: submitted (%d incident(s))", item.ProcessInstanceKey, len(item.ResolvedIncidentKeys))
		case process.ProcessInstanceResolutionStatusSkipped:
			renderHumanLine(cmd, "resolved process-instance %s: skipped (%s)", item.ProcessInstanceKey, processInstanceResolutionSkipReason(item))
		case process.ProcessInstanceResolutionStatusPartialFailed:
			renderHumanLine(cmd, "resolved process-instance %s: partial failure (resolved: %d, failed: %d): %s", item.ProcessInstanceKey, len(item.ResolvedIncidentKeys), len(item.FailedIncidentKeys), item.Error)
		case process.ProcessInstanceResolutionStatusFailed:
			renderHumanLine(cmd, "resolved process-instance %s: failed: %s", item.ProcessInstanceKey, item.Error)
		default:
			renderHumanLine(cmd, "resolved process-instance %s: %s", item.ProcessInstanceKey, item.Status)
		}
	}
	total, ok, failed := results.Totals()
	if results.DryRun {
		renderHumanLine(cmd, "dry run: resolve process-instances: %d target(s), %d planned/skipped, %d failed; no changes applied", total, ok, failed)
		if failed > 0 {
			return fmt.Errorf("one or more process-instance incident resolution dry-run lookups failed")
		}
		return nil
	}
	renderHumanLine(cmd, "resolved process-instances: %d (confirmed/submitted/skipped: %d, failed: %d)", total, ok, failed)
	if failed > 0 {
		return fmt.Errorf("one or more process-instance incident resolutions failed")
	}
	return nil
}

func processInstanceResolutionSkipReason(item process.ProcessInstanceResolutionResult) string {
	if item.ConfirmationStatus != "" {
		return item.ConfirmationStatus
	}
	return "no_active_incidents"
}
