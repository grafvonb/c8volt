// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

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
