// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"sort"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type jobSearcher interface {
	SearchJobs(ctx context.Context, query d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error)
}

// EnrichElementWithListeners attaches requested listener jobs to one runtime element.
func EnrichElementWithListeners(ctx context.Context, elementAPI API, jobAPI jobSearcher, key string, opts ...services.CallOption) (d.Element, error) {
	element, err := elementAPI.GetElement(ctx, key, opts...)
	if err != nil {
		return d.Element{}, err
	}
	if element.ProcessInstanceKey == "" {
		empty := []d.RuntimeListenerJob{}
		element.Listeners = &empty
		return element, nil
	}
	progress := newListenerEnrichmentProgress(1, opts)
	progress.emit(0)
	listeners, err := listenerJobsForProcessInstance(ctx, jobAPI, element.ProcessInstanceKey, opts...)
	if err != nil {
		return d.Element{}, err
	}
	progress.emit(1)
	enriched := attachListenersToElements([]d.Element{element}, listeners)
	return enriched[0], nil
}

// EnrichSearchElementsWithListeners collects runtime elements and attaches requested listener jobs.
func EnrichSearchElementsWithListeners(ctx context.Context, elementAPI API, jobAPI jobSearcher, query d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	result, err := elementAPI.SearchElements(ctx, query, opts...)
	if err != nil {
		return d.ElementSearchResult{}, err
	}
	result.Items, err = enrichElementItemsWithListeners(ctx, jobAPI, result.Items, opts...)
	if err != nil {
		return d.ElementSearchResult{}, err
	}
	return result, nil
}

// enrichElementItemsWithListeners performs one listener lookup per returned process instance key.
func enrichElementItemsWithListeners(ctx context.Context, jobAPI jobSearcher, elements []d.Element, opts ...services.CallOption) ([]d.Element, error) {
	processKeys := processInstanceKeysForElements(elements)
	listeners := make([]d.RuntimeListenerJob, 0)
	progress := newListenerEnrichmentProgress(len(processKeys), opts)
	progress.emit(0)
	for i, key := range processKeys {
		got, err := listenerJobsForProcessInstance(ctx, jobAPI, key, opts...)
		if err != nil {
			return nil, err
		}
		listeners = append(listeners, got...)
		progress.emit(i + 1)
	}
	return attachListenersToElements(elements, listeners), nil
}

type listenerEnrichmentProgress struct {
	total    int
	progress func(d.OpsProgressEvent)
}

func newListenerEnrichmentProgress(total int, opts []services.CallOption) listenerEnrichmentProgress {
	cfg := services.ApplyCallOptions(opts)
	return listenerEnrichmentProgress{
		total:    total,
		progress: cfg.Progress,
	}
}

func (p listenerEnrichmentProgress) emit(done int) {
	if p.progress == nil || p.total == 0 {
		return
	}
	frozen := d.OpsFrozenScopeProgress{
		Phase:        "loading listener jobs",
		CoreResource: "process instance(s)",
		Done:         done,
		Total:        p.total,
	}
	p.progress(d.OpsProgressEvent{
		Kind:        d.OpsProgressEventKindFrozenScope,
		FrozenScope: &frozen,
	})
}

// processInstanceKeysForElements returns stable unique process-instance keys from search results.
func processInstanceKeysForElements(elements []d.Element) []string {
	seen := map[string]bool{}
	keys := make([]string, 0, len(elements))
	for _, element := range elements {
		if element.ProcessInstanceKey == "" || seen[element.ProcessInstanceKey] {
			continue
		}
		seen[element.ProcessInstanceKey] = true
		keys = append(keys, element.ProcessInstanceKey)
	}
	return keys
}

// listenerJobsForProcessInstance loads both supported listener job kinds and filters broad responses.
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

// attachListenersToElements marks every returned element as listener-requested.
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
		matched := listenersByElement[element.ElementInstanceKey]
		elementListeners := make([]d.RuntimeListenerJob, 0, len(matched))
		elementListeners = append(elementListeners, matched...)
		out[i].Listeners = &elementListeners
	}
	return out
}
