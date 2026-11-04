// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// businessIDRepairOutcomeCounts derives bounded outcome facts without formatting raw backend errors.
func businessIDRepairOutcomeCounts(result ops.RepairResult) (submitted, confirmed, failed int) {
	for _, item := range result.Plan {
		if item.ResolutionStatus == ops.WorkflowStepStatusSubmitted {
			submitted++
		}
		if item.ConfirmationStatus == ops.WorkflowStepStatusConfirmed {
			confirmed++
		}
		if len(item.Errors) > 0 {
			failed++
		}
	}
	return
}

// renderBusinessIDRepair preserves the existing audit payload and human wording without double error rendering.
func renderBusinessIDRepair(cmd *cobra.Command, result ops.RepairResult, operationErr error) error {
	if pickMode() == RenderModeJSON {
		if operationErr != nil {
			e := resultEnvelopeForError(cmd, operationErr)
			e.Payload = result
			return renderResultEnvelope(cmd, e)
		}
		submitted, _, _ := businessIDRepairOutcomeCounts(result)
		if result.Request.DryRun || submitted == 0 {
			return renderSucceededResult(cmd, result)
		}
		return renderCommandResult(cmd, result)
	}
	if operationErr != nil && result.FrozenSet.Status == ops.WorkflowStepStatusFailed {
		return nil
	}
	// Human view prints only typed facts; raw errors remain in machine evidence and the original exit.
	result.Errors = nil
	return renderOpsRepairProcessInstanceResult(cmd, result)
}
