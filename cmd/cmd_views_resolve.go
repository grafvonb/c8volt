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
		case process.IncidentResolutionStatusConfirmed:
			renderHumanLine(cmd, "resolved incident %s: confirmed", item.IncidentKey)
		case process.IncidentResolutionStatusSubmitted:
			renderHumanLine(cmd, "resolved incident %s: submitted", item.IncidentKey)
		case process.IncidentResolutionStatusSkipped:
			renderHumanLine(cmd, "resolved incident %s: skipped", item.IncidentKey)
		case process.IncidentResolutionStatusMutationFailed:
			renderHumanLine(cmd, "resolved incident %s: mutation failed: %s", item.IncidentKey, item.Error)
		case process.IncidentResolutionStatusConfirmationFailed:
			renderHumanLine(cmd, "resolved incident %s: confirmation failed: %s", item.IncidentKey, item.Error)
		default:
			renderHumanLine(cmd, "resolved incident %s: %s", item.IncidentKey, item.Status)
		}
	}
	total, ok, failed := results.Totals()
	renderHumanLine(cmd, "resolved: %d (confirmed/submitted/skipped: %d, failed: %d)", total, ok, failed)
	if failed > 0 {
		return fmt.Errorf("one or more incident resolutions failed")
	}
	return nil
}
