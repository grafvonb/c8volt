// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
)

// resourceView renders a resource lookup result using the active get-output mode.
func resourceView(cmd *cobra.Command, item resource.Resource) error {
	return resourceItemView(cmd, item, pickMode())
}

// resourceItemView renders one resource through the shared item view contract.
func resourceItemView(cmd *cobra.Command, item resource.Resource, mode RenderMode) error {
	return itemView(cmd, item, mode, oneLineResource, func(it resource.Resource) string { return it.ID })
}

// oneLineResource formats a compact resource row for human lookup output.
func oneLineResource(it resource.Resource) string {
	return compactFlatRow(flatRowResource(it))
}

// flatRowResource keeps resource names in the same human position while aligning IDs and keys in list output.
func flatRowResource(it resource.Resource) flatRow {
	vTag := ""
	if it.VersionTag != "" {
		vTag = "/" + it.VersionTag
	}
	return flatRow{it.ID, "k:" + it.Key, it.TenantId, it.Name, fmt.Sprintf("v%d%s", it.Version, vTag)}
}

// deleteProcessDefinitionDryRunPreview carries observed impact without claiming submission.
type deleteProcessDefinitionDryRunPreview struct {
	Operation         string                                     `json:"operation"`
	Plan              resource.DeleteProcessDefinitionPlan       `json:"plan"`
	Totals            resource.DeleteProcessDefinitionPlanTotals `json:"totals"`
	MutationSubmitted bool                                       `json:"mutationSubmitted"`
}

// renderDeleteProcessDefinitionDryRun keeps previews in the selected output mode.
func renderDeleteProcessDefinitionDryRun(cmd *cobra.Command, plan resource.DeleteProcessDefinitionPlan) error {
	preview := deleteProcessDefinitionDryRunPreview{
		Operation:         "delete",
		Plan:              plan,
		Totals:            plan.Totals(),
		MutationSubmitted: false,
	}
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, preview)
	}
	if pickMode() != RenderModeOneLine {
		return nil
	}
	renderAttachedTenantContext(cmd)
	renderHumanLine(cmd, "dry run: delete process-definition")
	renderDeleteProcessDefinitionImpact(cmd, plan)
	return nil
}

// renderDeleteProcessDefinitionImpact reports direct unfinished workload and expanded family impact.
func renderDeleteProcessDefinitionImpact(cmd *cobra.Command, plan resource.DeleteProcessDefinitionPlan) {
	if pickMode() != RenderModeOneLine {
		return
	}
	renderAttachedTenantContext(cmd)
	totals := plan.Totals()
	if plan.StateCheckSkipped {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); process-instance state check skipped; no changes made yet", totals.ProcessDefinitions)
		renderHumanWarningLine(cmd, "deletion is irreversible: process-definition resources and associated history will be removed.")
		return
	}
	if totals.UnfinishedProcessInstances == 0 {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); no unfinished process instances found; no changes made yet", totals.ProcessDefinitions)
		renderHumanWarningLine(cmd, "deletion is irreversible: process-definition resources and associated history will be removed.")
		return
	}
	if flagForce {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); %s found; no changes made yet", totals.ProcessDefinitions, processDefinitionUnfinishedSummary(totals))
		renderHumanLine(cmd, "--force will cancel %d root process instance(s), then delete %d affected process instance(s), before deleting process definitions", totals.CancellationRoots, totals.CancellationAffected)
	} else {
		renderHumanLine(cmd, "delete impact check: %d process definition(s); %s found; no changes made yet", totals.ProcessDefinitions, processDefinitionUnfinishedSummary(totals))
		return
	}
	for _, warning := range plan.Warnings {
		renderHumanWarningLine(cmd, "%s", warning)
	}
	renderHumanWarningLine(cmd, "deletion is irreversible: process-definition resources and associated history will be removed.")
}

// processDefinitionUnfinishedSummary distinguishes union evidence from independently observed buckets.
func processDefinitionUnfinishedSummary(t resource.DeleteProcessDefinitionPlanTotals) string {
	summary := fmt.Sprintf("%d unfinished process instance(s)", t.UnfinishedProcessInstances)
	if t.SuspendedCountSupported {
		summary += fmt.Sprintf(" (active %d, suspended %d)", t.ActiveProcessInstances, t.SuspendedProcessInstances)
	}
	return summary
}

// formatProcessDefinitionDeleteFailure uses bounded facts, never backend text or per-key chains.
func formatProcessDefinitionDeleteFailure(stage string, selected int, reports resource.DeleteReports, err error) string {
	_, accepted, _ := reports.Totals()
	return fmt.Sprintf("%s: selected %d, accepted %d; outcome unconfirmed; %s", stage, selected, accepted, boundedCommandFailureReason(err))
}

// renderDeleteProcessDefinitionFailure preserves partial machine results and logs one aggregate human error.
// HTTP details remain in the existing redacted DEBUG transport path.
func renderDeleteProcessDefinitionFailure(cmd *cobra.Command, log *slog.Logger, selected int, reports resource.DeleteReports, err error) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		envelope := resultEnvelopeForError(cmd, err)
		envelope.Payload = reports
		return renderResultEnvelope(cmd, envelope)
	}
	log.Error(formatProcessDefinitionDeleteFailure("deleting process definitions", selected, reports, err))
	return nil
}

// renderDeleteProcessDefinitionResult keeps compact completion visible when per-key logs are suppressed.
func renderDeleteProcessDefinitionResult(cmd *cobra.Command, reports resource.DeleteReports) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, reports)
	}
	if pickMode() == RenderModeOneLine && (!flagVerbose || flagNoWait) {
		total, oks, noks := reports.Totals()
		if flagNoWait {
			renderHumanLine(cmd, "pd delete accepted; requested %d, ok %d, failed %d", total, oks, noks)
		} else {
			renderHumanLine(cmd, "pd delete done; requested %d, ok %d, failed %d", total, oks, noks)
		}
	}
	return nil
}
