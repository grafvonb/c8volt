// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// assignmentCounts uses only typed workflow facts; raw error evidence is machine-only.
func assignmentCounts(results process.BusinessIDAssignmentResults) (accepted, confirmed, rejected, unknown, skipped int) {
	for _, item := range results.Items {
		switch item.Status {
		case "confirmed":
			accepted++
			confirmed++
		case "accepted":
			accepted++
		case "rejected":
			rejected++
		case "unknown":
			unknown++
		default:
			skipped++
		}
	}
	return
}

// renderBusinessIDAssignment keeps previews, empty scopes and submissions distinct.
// Keys denote planned targets in dry-run and known accepted targets after execution.
func renderBusinessIDAssignment(cmd *cobra.Command, plan process.BusinessIDAssignmentPlan, results process.BusinessIDAssignmentResults, planned bool, operationErr error) error {
	accepted, confirmed, rejected, unknown, skipped := assignmentCounts(results)
	if pickMode() == RenderModeJSON {
		var payload any = results
		if planned {
			payload = plan
		}
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = payload
			return renderResultEnvelope(cmd, envelope)
		}
		if !planned && flagNoWait && accepted > 0 {
			return renderAcceptedResult(cmd, payload)
		}
		return renderSucceededResult(cmd, payload)
	}
	if pickMode() == RenderModeKeysOnly {
		if planned {
			if flagDryRun && operationErr == nil {
				for _, item := range plan.Items {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.Key); err != nil {
						return err
					}
				}
			}
		} else {
			for _, item := range results.Items {
				if item.Status == "accepted" || item.Status == "confirmed" {
					if _, err := fmt.Fprintln(cmd.OutOrStdout(), item.Key); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}
	if planned {
		if operationErr != nil {
			return nil
		}
		label := "plan"
		if flagDryRun {
			label = "dry run"
		}
		if len(plan.Items) == 0 {
			renderHumanLine(cmd, "%s: business-ID assignment: nothing to update", label)
		} else {
			renderHumanLine(cmd, "%s: business-ID assignment: %d process instance(s), requested business ID %q; backend acceptance is not guaranteed", label, len(plan.Items), *plan.RequestedBusinessID)
		}
		if flagVerbose {
			for _, item := range plan.Items {
				if item.PreviouslyObservedBusinessID != nil {
					renderHumanLine(cmd, "%s: state %q; previously observed business ID %q", item.Key, item.State, *item.PreviouslyObservedBusinessID)
				} else {
					renderHumanLine(cmd, "%s: state %q; no prior business ID observation", item.Key, item.State)
				}
			}
		}
		return nil
	}
	renderHumanLine(cmd, "business-ID assignment: %d selected; %d accepted, %d confirmed, %d rejected, %d unknown, %d not submitted", len(results.Items), accepted, confirmed, rejected, unknown, skipped)
	if flagVerbose {
		for _, item := range results.Items {
			renderHumanLine(cmd, "%s: %s; confirmation %s", item.Key, item.Status, item.Confirmation)
		}
	}
	return nil
}

// formatBusinessIDAssignmentFailure groups bounded evidence without transport chains.
func formatBusinessIDAssignmentFailure(plan process.BusinessIDAssignmentPlan, results process.BusinessIDAssignmentResults, planned bool, err error) string {
	if planned {
		return fmt.Sprintf("business-ID assignment: preparation or approval failed; no changes submitted; %s", boundedCommandFailureReason(err))
	}
	accepted, confirmed, rejected, unknown, skipped := assignmentCounts(results)
	return fmt.Sprintf("business-ID assignment: %d selected; %d rejected, %d unknown, %d accepted unconfirmed, %d not submitted; %s", len(plan.Items), rejected, unknown, accepted-confirmed, skipped, boundedCommandFailureReason(err))
}
