// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/spf13/cobra"
)

// listIncidentsView renders incident collections through the supported incident output modes.
func listIncidentsView(cmd *cobra.Command, resp incident.Incidents, messageLimit int, omitMessage bool) error {
	if flagGetIncidentPIKeysOnly {
		return renderIncidentProcessInstanceKeys(cmd, resp.Items)
	}
	mode := pickMode()
	switch mode {
	case RenderModeJSON:
		return renderJSONPayload(cmd, mode, resp)
	case RenderModeKeysOnly:
		for _, it := range resp.Items {
			renderOutputLine(cmd, "%s", it.IncidentKey)
		}
	default:
		for _, line := range formatIncidentListRowsWithTimezone(resp.Items, messageLimit, omitMessage, commandShowTimezoneOffset(cmd)) {
			renderOutputLine(cmd, "%s", line)
		}
		renderOutputLine(cmd, "found: %d", len(resp.Items))
	}
	return nil
}

// renderIncidentProcessInstanceKeys renders incident process-instance keys in result order.
func renderIncidentProcessInstanceKeys(cmd *cobra.Command, items []incident.ProcessInstanceIncidentDetail) error {
	for _, it := range items {
		if it.ProcessInstanceKey == "" {
			continue
		}
		renderOutputLine(cmd, "%s", it.ProcessInstanceKey)
	}
	return nil
}
