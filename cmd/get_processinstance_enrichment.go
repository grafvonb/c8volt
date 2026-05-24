// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

// enrichProcessInstancesWithIncidentActivity wraps incident enrichment with a
// visible activity indicator only when there is work to do. Empty collections
// still pass through the API path so JSON shapes and downstream behavior remain
// identical.
func enrichProcessInstancesWithIncidentActivity(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.IncidentEnrichedProcessInstances, error) {
	return enrichProcessInstancesWithIncidentActivityOptions(cmd, cli, pis, collectIncidentEnrichmentOptions())
}

// enrichProcessInstancesWithIncidentActivityOptions lets explicit-key callers
// keep admin-input options while preserving the shared activity behavior.
func enrichProcessInstancesWithIncidentActivityOptions(cmd *cobra.Command, cli process.API, pis process.ProcessInstances, opts []options.FacadeOption) (process.IncidentEnrichedProcessInstances, error) {
	if len(pis.Items) == 0 {
		return cli.EnrichProcessInstancesWithIncidents(cmd.Context(), pis, opts...)
	}
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("loading incident details for %d process instance(s)", len(pis.Items)))
	defer stopActivity()
	return cli.EnrichProcessInstancesWithIncidents(cmd.Context(), pis, opts...)
}

func collectIncidentEnrichmentOptions() []options.FacadeOption {
	return append(collectOptions(),
		options.WithIncidentState(flagGetPIIncidentState),
		options.WithIncidentErrorType(flagGetPIIncidentErrorType),
		options.WithIncidentErrorMessage(flagGetPIIncidentErrorMessage),
	)
}

// enrichProcessInstancesWithVariableActivity mirrors incident enrichment for
// process-instance-scope variables. The activity boundary keeps large list
// searches understandable without changing the zero-row behavior.
func enrichProcessInstancesWithVariableActivity(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.VariableEnrichedProcessInstances, error) {
	return enrichProcessInstancesWithVariableActivityOptions(cmd, cli, pis, collectOptions())
}

// enrichProcessInstancesWithVariableActivityOptions mirrors the incident
// variant for direct-key admin input and search-derived list modes.
func enrichProcessInstancesWithVariableActivityOptions(cmd *cobra.Command, cli process.API, pis process.ProcessInstances, opts []options.FacadeOption) (process.VariableEnrichedProcessInstances, error) {
	if len(pis.Items) == 0 {
		return cli.EnrichProcessInstancesWithVariables(cmd.Context(), pis, opts...)
	}
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("loading variable details for %d process instance(s)", len(pis.Items)))
	defer stopActivity()
	return cli.EnrichProcessInstancesWithVariables(cmd.Context(), pis, opts...)
}
