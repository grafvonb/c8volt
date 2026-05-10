// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"github.com/grafvonb/c8volt/c8volt/incident"
	"strings"

	"github.com/spf13/cobra"
)

func renderIncidentResolutionResults(cmd *cobra.Command, results incident.ResolutionResults) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, results)
	}
	for _, item := range results.Items {
		switch item.Status {
		case incident.ResolutionStatusPlanned:
			renderHumanLine(cmd, "dry run: incident %s would be resolved", item.IncidentKey)
		case incident.ResolutionStatusConfirmed:
			renderHumanLine(cmd, "resolved incident %s: confirmed", item.IncidentKey)
		case incident.ResolutionStatusSubmitted:
			renderHumanLine(cmd, "resolved incident %s: submitted", item.IncidentKey)
		case incident.ResolutionStatusSkipped:
			renderHumanLine(cmd, "%s", incidentResolutionSkippedLine(item))
		case incident.ResolutionStatusMutationFailed:
			renderHumanLine(cmd, "resolved incident %s: mutation failed: %s", item.IncidentKey, item.Error)
		case incident.ResolutionStatusConfirmationFailed:
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

func incidentResolutionSkippedLine(item incident.ResolutionResult) string {
	if strings.EqualFold(item.IncidentState, "RESOLVED") {
		if item.Incident != nil && item.Incident.CreationTime != "" {
			return fmt.Sprintf("incident %s already resolved (created %s): skipped", item.IncidentKey, item.Incident.CreationTime)
		}
		return fmt.Sprintf("incident %s already resolved: skipped", item.IncidentKey)
	}
	return fmt.Sprintf("resolved incident %s: skipped (%s)", item.IncidentKey, incidentResolutionSkipReason(item))
}

func incidentResolutionSkipReason(item incident.ResolutionResult) string {
	if item.IncidentState != "" {
		return item.IncidentState
	}
	if item.ConfirmationStatus != "" {
		return item.ConfirmationStatus
	}
	return "not_active"
}

func renderProcessInstanceResolutionResults(cmd *cobra.Command, results incident.ProcessInstanceResolutionResults) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, results)
	}
	for _, item := range results.Items {
		switch item.Status {
		case incident.ProcessInstanceResolutionStatusPlanned:
			renderHumanLine(cmd, "dry run: process-instance %s would resolve %d incident(s)", item.ProcessInstanceKey, len(item.AttemptedIncidentKeys))
		case incident.ProcessInstanceResolutionStatusConfirmed:
			renderHumanLine(cmd, "resolved process-instance %s: confirmed (%d incident(s))", item.ProcessInstanceKey, len(item.ResolvedIncidentKeys))
		case incident.ProcessInstanceResolutionStatusSubmitted:
			renderHumanLine(cmd, "resolved process-instance %s: submitted (%d incident(s))", item.ProcessInstanceKey, len(item.ResolvedIncidentKeys))
		case incident.ProcessInstanceResolutionStatusSkipped:
			renderHumanLine(cmd, "resolved process-instance %s: skipped (%s)", item.ProcessInstanceKey, processInstanceResolutionSkipReason(item))
		case incident.ProcessInstanceResolutionStatusPartialFailed:
			renderHumanLine(cmd, "resolved process-instance %s: partial failure (resolved: %d, failed: %d): %s", item.ProcessInstanceKey, len(item.ResolvedIncidentKeys), len(item.FailedIncidentKeys), item.Error)
		case incident.ProcessInstanceResolutionStatusFailed:
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

func processInstanceResolutionSkipReason(item incident.ProcessInstanceResolutionResult) string {
	if item.ConfirmationStatus != "" {
		return item.ConfirmationStatus
	}
	return "no_active_incidents"
}
