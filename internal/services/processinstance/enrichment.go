// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"sort"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
)

type incidentSearcher interface {
	SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error)
}

type variableSearcher interface {
	SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error)
}

// EnrichProcessInstancesWithIncidents attaches direct incident details to selected process-instance results without reordering them.
func EnrichProcessInstancesWithIncidents(ctx context.Context, api incidentSearcher, pis []d.ProcessInstance, opts ...services.CallOption) (d.IncidentEnrichedProcessInstances, error) {
	items := make([]d.IncidentEnrichedProcessInstance, 0, len(pis))
	for _, pi := range pis {
		incidents, err := api.SearchProcessInstanceIncidents(ctx, pi.Key, opts...)
		if err != nil {
			return d.IncidentEnrichedProcessInstances{}, err
		}
		items = append(items, d.IncidentEnrichedProcessInstance{
			Item:      pi,
			Incidents: incidentsForProcessInstance(pi.Key, incidents),
		})
	}
	return d.IncidentEnrichedProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}, nil
}

// EnrichProcessInstancesWithVariables attaches process-scope variables to selected process-instance results without reordering them.
func EnrichProcessInstancesWithVariables(ctx context.Context, api variableSearcher, pis []d.ProcessInstance, opts ...services.CallOption) (d.VariableEnrichedProcessInstances, error) {
	items := make([]d.VariableEnrichedProcessInstance, 0, len(pis))
	for _, pi := range pis {
		variables, err := api.SearchProcessInstanceVariables(ctx, pi.Key, opts...)
		if err != nil {
			return d.VariableEnrichedProcessInstances{}, err
		}
		items = append(items, d.VariableEnrichedProcessInstance{
			Item:      pi,
			Variables: variablesForProcessInstance(pi.Key, variables),
		})
	}
	return d.VariableEnrichedProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}, nil
}

// EnrichTraversalWithIncidents overlays incident details onto walked items while preserving traversal metadata and warnings.
func EnrichTraversalWithIncidents(ctx context.Context, api incidentSearcher, result pitraversal.Result, opts ...services.CallOption) (d.IncidentEnrichedTraversalResult, error) {
	items := make([]d.IncidentEnrichedTraversalItem, 0, len(result.Keys))
	for _, key := range result.Keys {
		pi, ok := result.Chain[key]
		if !ok {
			continue
		}
		incidents, err := api.SearchProcessInstanceIncidents(ctx, key, opts...)
		if err != nil {
			return d.IncidentEnrichedTraversalResult{}, err
		}
		items = append(items, d.IncidentEnrichedTraversalItem{
			Item:      pi,
			Incidents: incidentsForProcessInstance(key, incidents),
		})
	}
	return d.IncidentEnrichedTraversalResult{
		Mode:             string(result.Mode),
		Outcome:          string(result.Outcome),
		StartKey:         result.StartKey,
		RootKey:          result.RootKey,
		Keys:             append([]string(nil), result.Keys...),
		Edges:            result.Edges,
		Items:            items,
		MissingAncestors: traversalMissingAncestors(result.MissingAncestors),
		Warning:          result.Warning,
	}, nil
}

// incidentsForProcessInstance keeps only details owned by the requested key, guarding against broad backend incident responses.
func incidentsForProcessInstance(key string, incidents []d.ProcessInstanceIncidentDetail) []d.ProcessInstanceIncidentDetail {
	out := make([]d.ProcessInstanceIncidentDetail, 0, len(incidents))
	for _, incident := range incidents {
		if incident.ProcessInstanceKey == key {
			out = append(out, incident)
		}
	}
	return out
}

// variablesForProcessInstance keeps only process-scope variables owned by the requested key in stable name order.
func variablesForProcessInstance(key string, variables []d.ProcessInstanceVariable) []d.ProcessInstanceVariable {
	out := make([]d.ProcessInstanceVariable, 0, len(variables))
	for _, variable := range variables {
		if variable.ProcessInstanceKey == key && variable.ScopeKey == key {
			out = append(out, variable)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})
	return out
}

// traversalMissingAncestors maps traversal package missing-ancestor markers into domain results.
func traversalMissingAncestors(in []pitraversal.MissingAncestor) []d.MissingAncestor {
	if len(in) == 0 {
		return nil
	}
	out := make([]d.MissingAncestor, len(in))
	for i, item := range in {
		out[i] = d.MissingAncestor{
			Key:      item.Key,
			StartKey: item.StartKey,
		}
	}
	return out
}
