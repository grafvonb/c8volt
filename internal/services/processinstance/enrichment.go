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

type elementSearcher interface {
	SearchElements(ctx context.Context, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error)
}

type jobSearcher interface {
	SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error)
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

// EnrichProcessInstancesWithElements attaches runtime element instances to selected process-instance results without reordering them.
func EnrichProcessInstancesWithElements(ctx context.Context, api elementSearcher, pis []d.ProcessInstance, opts ...services.CallOption) (d.ElementEnrichedProcessInstances, error) {
	items := make([]d.ElementEnrichedProcessInstance, 0, len(pis))
	for _, pi := range pis {
		result, err := api.SearchElements(ctx, d.ElementSearchQuery{ProcessInstanceKey: pi.Key}, opts...)
		if err != nil {
			return d.ElementEnrichedProcessInstances{}, err
		}
		items = append(items, d.ElementEnrichedProcessInstance{
			Item:     pi,
			Elements: elementsForProcessInstance(pi.Key, result.Items),
		})
	}
	return d.ElementEnrichedProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}, nil
}

// EnrichProcessInstancesWithElementListeners attaches runtime elements and
// requested listener job arrays to selected process-instance results.
func EnrichProcessInstancesWithElementListeners(ctx context.Context, elementAPI elementSearcher, jobAPI jobSearcher, pis []d.ProcessInstance, opts ...services.CallOption) (d.ElementEnrichedProcessInstances, error) {
	items := make([]d.ElementEnrichedProcessInstance, 0, len(pis))
	for _, pi := range pis {
		elementResult, err := elementAPI.SearchElements(ctx, d.ElementSearchQuery{ProcessInstanceKey: pi.Key}, opts...)
		if err != nil {
			return d.ElementEnrichedProcessInstances{}, err
		}
		elements := elementsForProcessInstance(pi.Key, elementResult.Items)
		listeners, err := listenerJobsForProcessInstance(ctx, jobAPI, pi.Key, opts...)
		if err != nil {
			return d.ElementEnrichedProcessInstances{}, err
		}
		items = append(items, d.ElementEnrichedProcessInstance{
			Item:     pi,
			Elements: attachListenersToElements(elements, listeners),
		})
	}
	return d.ElementEnrichedProcessInstances{
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

// elementsForProcessInstance keeps only elements owned by the requested key in stable runtime order.
func elementsForProcessInstance(key string, elements []d.Element) []d.Element {
	out := make([]d.Element, 0, len(elements))
	for _, element := range elements {
		if element.ProcessInstanceKey == key {
			out = append(out, element)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].StartDate == out[j].StartDate {
			return out[i].ElementInstanceKey < out[j].ElementInstanceKey
		}
		return out[i].StartDate < out[j].StartDate
	})
	return out
}

func listenerJobsForProcessInstance(ctx context.Context, api jobSearcher, key string, opts ...services.CallOption) ([]d.RuntimeListenerJob, error) {
	jobs := []d.RuntimeListenerJob{}
	for _, kind := range []string{d.JobKindExecutionListener, d.JobKindTaskListener} {
		result, err := api.SearchJobs(ctx, d.JobSearchQuery{ProcessInstanceKey: key, Kind: kind}, opts...)
		if err != nil {
			return nil, err
		}
		for _, job := range result.Items {
			if job.ProcessInstanceKey != key || !d.IsRuntimeListenerJobKind(job.Kind) || job.ElementInstanceKey == "" {
				continue
			}
			jobs = append(jobs, d.RuntimeListenerJobFromJob(job))
		}
	}
	sort.SliceStable(jobs, func(i, j int) bool {
		return jobs[i].JobKey < jobs[j].JobKey
	})
	return jobs, nil
}

func attachListenersToElements(elements []d.Element, listeners []d.RuntimeListenerJob) []d.Element {
	listenersByElement := make(map[string][]d.RuntimeListenerJob, len(elements))
	for _, listener := range listeners {
		if listener.ElementInstanceKey == "" {
			continue
		}
		listenersByElement[listener.ElementInstanceKey] = append(listenersByElement[listener.ElementInstanceKey], listener)
	}
	out := make([]d.Element, len(elements))
	for i, element := range elements {
		out[i] = element
		listeners := append([]d.RuntimeListenerJob(nil), listenersByElement[element.ElementInstanceKey]...)
		out[i].Listeners = &listeners
	}
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
