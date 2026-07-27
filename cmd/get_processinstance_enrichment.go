// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/services/incidentfilter"
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
	state, _ := incidentfilter.NormalizeState(flagGetPIIncidentState)
	errorType, _ := incidentfilter.NormalizeErrorType(flagGetPIIncidentErrorType)
	return append(collectOptions(),
		options.WithIncidentState(state),
		options.WithIncidentErrorType(errorType),
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

// enrichProcessInstancesWithElementActivity wraps runtime element enrichment with the shared command activity behavior.
func enrichProcessInstancesWithElementActivity(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (process.ElementEnrichedProcessInstances, error) {
	return enrichProcessInstancesWithElementActivityOptions(cmd, cli, pis, collectOptions())
}

// enrichProcessInstancesWithElementActivityOptions lets explicit-key callers keep admin-input options.
func enrichProcessInstancesWithElementActivityOptions(cmd *cobra.Command, cli process.API, pis process.ProcessInstances, opts []options.FacadeOption) (process.ElementEnrichedProcessInstances, error) {
	if len(pis.Items) == 0 {
		return cli.EnrichProcessInstancesWithElements(cmd.Context(), pis, opts...)
	}
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("loading element details for %d process instance(s)", len(pis.Items)))
	defer stopActivity()
	return cli.EnrichProcessInstancesWithElements(cmd.Context(), pis, appendFrozenScopeProgressOption(cmd, opts)...)
}

// enrichProcessInstancesWithElementListenerActivityOptions routes element
// enrichment through the listener-aware facade path while preserving explicit
// key options and requested-empty behavior.
func enrichProcessInstancesWithElementListenerActivityOptions(cmd *cobra.Command, cli process.API, pis process.ProcessInstances, opts []options.FacadeOption) (process.ElementEnrichedProcessInstances, error) {
	if len(pis.Items) == 0 {
		return cli.EnrichProcessInstancesWithElementListeners(cmd.Context(), pis, opts...)
	}
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("loading listener jobs for %d process instance(s)", len(pis.Items)))
	defer stopActivity()
	return cli.EnrichProcessInstancesWithElementListeners(cmd.Context(), pis, appendFrozenScopeProgressOption(cmd, opts)...)
}

func appendFrozenScopeProgressOption(cmd *cobra.Command, opts []options.FacadeOption) []options.FacadeOption {
	out := append([]options.FacadeOption{}, opts...)
	return append(out, options.WithProgress(func(event options.ProgressEvent) {
		if event.Kind != options.ProgressEventKindFrozenScope || event.FrozenScope == nil {
			return
		}
		progress := ops.FrozenScopeProgress{
			Phase:        event.FrozenScope.Phase,
			CoreResource: event.FrozenScope.CoreResource,
			Done:         event.FrozenScope.Done,
			Total:        event.FrozenScope.Total,
			Elapsed:      event.FrozenScope.Elapsed,
			Rate:         event.FrozenScope.Rate,
			ETA:          event.FrozenScope.ETA,
			Errors:       event.FrozenScope.Errors,
		}
		channel := opsProgressChannelForMode(opsProgressModeForCommand(cmd, pickMode()))
		printOpsSlowProcessAnalysisProgress(cmd, formatOpsFrozenScopeProgress(progress), channel)
	}))
}

// collectRequestedProcessInstanceActivity builds the shared activity view model
// by invoking each requested enrichment facade exactly once.
func collectRequestedProcessInstanceActivity(cmd *cobra.Command, cli process.API, pis process.ProcessInstances) (processInstanceActivityInstances, error) {
	return collectRequestedProcessInstanceActivityOptions(cmd, cli, pis, collectOptions(), collectIncidentEnrichmentOptions())
}

// collectRequestedProcessInstanceActivityOptions lets explicit-key callers
// preserve admin-input options while sharing combined enrichment orchestration.
func collectRequestedProcessInstanceActivityOptions(cmd *cobra.Command, cli process.API, pis process.ProcessInstances, generalOpts []options.FacadeOption, incidentOpts []options.FacadeOption) (processInstanceActivityInstances, error) {
	enrichments := processInstanceActivityEnrichments{}
	if flagGetPIWithIncidents {
		incidentEnriched, err := enrichProcessInstancesWithIncidentActivityOptions(cmd, cli, pis, incidentOpts)
		if err != nil {
			return processInstanceActivityInstances{}, fmt.Errorf("get process instance incidents: %w", err)
		}
		enrichments.Incidents = &incidentEnriched
	}
	if flagGetPIWithVars {
		variableEnriched, err := enrichProcessInstancesWithVariableActivityOptions(cmd, cli, pis, generalOpts)
		if err != nil {
			return processInstanceActivityInstances{}, fmt.Errorf("get process instance variables: %w", err)
		}
		enrichments.Variables = &variableEnriched
	}
	if flagGetPIWithElements {
		var elementEnriched process.ElementEnrichedProcessInstances
		var err error
		if flagGetPIWithListeners {
			elementEnriched, err = enrichProcessInstancesWithElementListenerActivityOptions(cmd, cli, pis, generalOpts)
		} else {
			elementEnriched, err = enrichProcessInstancesWithElementActivityOptions(cmd, cli, pis, generalOpts)
		}
		if err != nil {
			return processInstanceActivityInstances{}, fmt.Errorf("get process instance elements: %w", err)
		}
		enrichments.Elements = &elementEnriched
	}
	return mergeProcessInstanceActivity(pis, enrichments), nil
}
