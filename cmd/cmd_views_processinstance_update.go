// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

func renderUpdateProcessInstanceVariableResults(cmd *cobra.Command, results process.ProcessInstanceVariableUpdateResults) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderCommandResult(cmd, results)
	}
	for _, item := range results.Items {
		switch item.Status {
		case process.ProcessInstanceVariableUpdateStatusConfirmed:
			renderOutputLine(cmd, "updated process-instance %s: confirmed", item.Key)
		case process.ProcessInstanceVariableUpdateStatusSubmitted:
			renderOutputLine(cmd, "updated process-instance %s: submitted", item.Key)
		case process.ProcessInstanceVariableUpdateStatusConfirmationFailed:
			renderOutputLine(cmd, "updated process-instance %s: confirmation failed: %s", item.Key, item.Error)
		case process.ProcessInstanceVariableUpdateStatusMutationFailed:
			renderOutputLine(cmd, "updated process-instance %s: mutation failed: %s", item.Key, item.Error)
		default:
			renderOutputLine(cmd, "updated process-instance %s: %s", item.Key, item.Status)
		}
	}
	total, ok, failed := results.Totals()
	renderOutputLine(cmd, "updated: %d (confirmed/submitted: %d, failed: %d)", total, ok, failed)
	if failed > 0 {
		return fmt.Errorf("one or more process-instance variable updates failed")
	}
	return nil
}
