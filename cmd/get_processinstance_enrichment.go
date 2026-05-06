// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// enrichProcessInstancesWithIncidentActivity wraps incident enrichment with a
// visible activity indicator only when there is work to do. Empty collections
// still pass through the API path so JSON shapes and downstream behavior remain
// identical.
func enrichProcessInstancesWithIncidentActivity(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.IncidentEnrichedProcessInstances, error) {
	if len(pis.Items) == 0 {
		return cli.EnrichProcessInstancesWithIncidents(cmd.Context(), pis, collectOptions()...)
	}
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("loading incident details for %d process instance(s)", len(pis.Items)))
	defer stopActivity()
	return cli.EnrichProcessInstancesWithIncidents(cmd.Context(), pis, collectOptions()...)
}

// enrichProcessInstancesWithVariableActivity mirrors incident enrichment for
// process-instance-scope variables. The activity boundary keeps large list
// searches understandable without changing the zero-row behavior.
func enrichProcessInstancesWithVariableActivity(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.VariableEnrichedProcessInstances, error) {
	if len(pis.Items) == 0 {
		return cli.EnrichProcessInstancesWithVariables(cmd.Context(), pis, collectOptions()...)
	}
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("loading variable details for %d process instance(s)", len(pis.Items)))
	defer stopActivity()
	return cli.EnrichProcessInstancesWithVariables(cmd.Context(), pis, collectOptions()...)
}
