// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
)

// AnalyseSlowProcessInstances coordinates read-only slow process analysis below the command layer.
func (s *Service) AnalyseSlowProcessInstances(ctx context.Context, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (d.SlowProcessAnalysisResult, error) {
	capturedAt := request.CapturedNow
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
		request.CapturedNow = capturedAt
	}
	result := d.SlowProcessAnalysisResult{
		Request:    request,
		CapturedAt: capturedAt,
		Items:      []d.SlowProcessAnalysisProcessInstance{},
		Empty:      true,
	}
	if s == nil || s.piAPI == nil {
		return result, fmt.Errorf("%w: process-instance service is required for slow process analysis", d.ErrPrecondition)
	}
	if s.version == toolx.V87 {
		return result, fmt.Errorf("%w: slow process analysis requires Camunda 8.8 or newer", d.ErrUnsupported)
	}

	var instances []d.ProcessInstance
	switch request.SelectionMode {
	case d.SlowProcessAnalysisSelectionModeExplicitKeys:
		keys := request.InputKeys.Unique()
		if len(keys) == 0 {
			return result, fmt.Errorf("%w: at least one process-instance key is required", d.ErrValidation)
		}
		var err error
		instances, err = slowProcessAnalysisLookupExplicitKeys(ctx, s.piAPI, keys, opts...)
		if err != nil {
			return result, err
		}
		request.InputKeys = keys
	case d.SlowProcessAnalysisSelectionModeProcessDefinitionSearch:
		discovery, err := slowProcessAnalysisDiscoverProcessDefinitionInstances(ctx, s.piAPI, request, opts...)
		if err != nil {
			return result, err
		}
		instances = discovery.items
		result.DiscoveredScopeStatus = discovery.scope
	default:
		return result, fmt.Errorf("%w: select process instances with explicit keys or one process-definition selector", d.ErrValidation)
	}

	items := make([]d.SlowProcessAnalysisProcessInstance, 0, len(instances))
	for _, pi := range instances {
		items = append(items, slowProcessAnalysisProcessInstanceFromDomain(pi, capturedAt))
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if left.DurationAvailable != right.DurationAvailable {
			return left.DurationAvailable
		}
		if left.DurationAvailable && left.DurationMillis != right.DurationMillis {
			return left.DurationMillis > right.DurationMillis
		}
		return left.Key < right.Key
	})

	result.Items = items
	result.Count = len(items)
	result.Empty = len(items) == 0
	result.Request = request
	return result, nil
}

// slowProcessAnalysisLookupExplicitKeys resolves already-deduplicated keys through tenant-safe lookup.
func slowProcessAnalysisLookupExplicitKeys(ctx context.Context, api pisvc.API, keys []string, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	instances := make([]d.ProcessInstance, 0, len(keys))
	for _, key := range keys {
		pi, err := pisvc.LookupProcessInstance(ctx, api, key, opts...)
		if err != nil {
			return nil, fmt.Errorf("lookup process instance %s: %w", key, err)
		}
		instances = append(instances, pi)
	}
	return instances, nil
}

type slowProcessAnalysisSearchDiscovery struct {
	items []d.ProcessInstance
	scope d.DiscoveryScopeStatus
}

// slowProcessAnalysisDiscoverProcessDefinitionInstances freezes a paged process-definition selection before analysis.
func slowProcessAnalysisDiscoverProcessDefinitionInstances(ctx context.Context, api pisvc.API, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (slowProcessAnalysisSearchDiscovery, error) {
	filter, err := slowProcessAnalysisProcessInstanceFilter(request)
	if err != nil {
		return slowProcessAnalysisSearchDiscovery{}, err
	}
	batchSize := slowProcessAnalysisDiscoveryBatchSize(request.BatchSize)
	pageReq := d.ProcessInstancePageRequest{Size: batchSize}
	discovery := slowProcessAnalysisSearchDiscovery{
		items: []d.ProcessInstance{},
		scope: d.DiscoveryScopeStatus{
			Complete:  true,
			BatchSize: batchSize,
			Limit:     request.Limit,
		},
	}
	seen := map[string]struct{}{}

	for {
		page, err := api.SearchForProcessInstancesPage(ctx, filter, pageReq, opts...)
		if err != nil {
			return slowProcessAnalysisSearchDiscovery{}, fmt.Errorf("discover process instances for slow analysis: %w", err)
		}
		discovery.scope.Pages++
		discovery.scope.CandidatesSeen += len(page.Items)
		for _, item := range page.Items {
			if request.Limit > 0 && len(discovery.items) >= int(request.Limit) {
				discovery.scope.Complete = false
				discovery.scope.Limited = true
				discovery.scope.CandidatesFrozen = len(discovery.items)
				return discovery, nil
			}
			if _, ok := seen[item.Key]; ok {
				continue
			}
			seen[item.Key] = struct{}{}
			discovery.items = append(discovery.items, item)
		}
		if request.Limit > 0 && len(discovery.items) >= int(request.Limit) && page.OverflowState == d.ProcessInstanceOverflowStateHasMore {
			discovery.scope.Complete = false
			discovery.scope.Limited = true
			discovery.scope.CandidatesFrozen = len(discovery.items)
			return discovery, nil
		}
		if len(page.Items) == 0 || page.OverflowState != d.ProcessInstanceOverflowStateHasMore {
			discovery.scope.CandidatesFrozen = len(discovery.items)
			return discovery, nil
		}
		pageReq = slowProcessAnalysisNextDiscoveryPage(pageReq, page)
	}
}

// slowProcessAnalysisProcessInstanceFilter maps the normalized analysis selector to process-instance search.
func slowProcessAnalysisProcessInstanceFilter(request d.SlowProcessAnalysisRequest) (d.ProcessInstanceFilter, error) {
	selector := request.ProcessDefinitionSelector
	if (selector.BpmnProcessID == "") == (selector.ProcessDefinitionKey == "") {
		return d.ProcessInstanceFilter{}, fmt.Errorf("%w: process-definition search requires exactly one selector", d.ErrValidation)
	}
	hasIncident := (*bool)(nil)
	if request.ProcessInstanceFilters.NoIncidentsOnly {
		noIncident := false
		hasIncident = &noIncident
	}
	filter := d.ProcessInstanceFilter{
		BpmnProcessId:        selector.BpmnProcessID,
		ProcessDefinitionKey: selector.ProcessDefinitionKey,
		State:                slowProcessAnalysisDiscoveryState(request.ProcessInstanceFilters.State),
		StartDateAfter:       request.ProcessInstanceFilters.StartDateAfter,
		StartDateBefore:      request.ProcessInstanceFilters.StartDateBefore,
		EndDateAfter:         request.ProcessInstanceFilters.EndDateAfter,
		EndDateBefore:        request.ProcessInstanceFilters.EndDateBefore,
		HasIncident:          hasIncident,
	}
	return filter, nil
}

// slowProcessAnalysisDiscoveryState omits the all-state sentinel so search includes every supported state.
func slowProcessAnalysisDiscoveryState(state d.State) d.State {
	if state == d.StateAll {
		return ""
	}
	return state
}

// slowProcessAnalysisDiscoveryBatchSize keeps search page requests inside the server limit.
func slowProcessAnalysisDiscoveryBatchSize(size int32) int32 {
	if size <= 0 || size > consts.MaxPISearchSize {
		return consts.MaxPISearchSize
	}
	return size
}

// slowProcessAnalysisNextDiscoveryPage advances cursor-based pages before falling back to offset paging.
func slowProcessAnalysisNextDiscoveryPage(current d.ProcessInstancePageRequest, page d.ProcessInstancePage) d.ProcessInstancePageRequest {
	next := d.ProcessInstancePageRequest{Size: current.Size}
	if page.EndCursor != "" {
		next.After = page.EndCursor
		return next
	}
	next.From = current.From + int32(len(page.Items))
	return next
}

// slowProcessAnalysisProcessInstanceFromDomain preserves selected root metadata while adding whole-instance duration.
func slowProcessAnalysisProcessInstanceFromDomain(pi d.ProcessInstance, capturedAt time.Time) d.SlowProcessAnalysisProcessInstance {
	duration, millis, available := slowProcessAnalysisProcessDuration(pi, capturedAt)
	return slowProcessAnalysisRootFromProcessInstance(pi, duration, millis, available)
}

// slowProcessAnalysisProcessDuration returns a measured duration only when timestamps prove it.
func slowProcessAnalysisProcessDuration(pi d.ProcessInstance, capturedAt time.Time) (string, int64, bool) {
	start, err := time.Parse(time.RFC3339Nano, pi.StartDate)
	if err != nil || start.IsZero() {
		return "", 0, false
	}
	var end time.Time
	switch {
	case pi.EndDate != "":
		end, err = time.Parse(time.RFC3339Nano, pi.EndDate)
		if err != nil || end.Before(start) {
			return "", 0, false
		}
	case !pi.State.IsTerminal():
		end = capturedAt
		if end.Before(start) {
			return "", 0, false
		}
	default:
		return "", 0, false
	}
	duration := end.Sub(start)
	return duration.String(), duration.Milliseconds(), true
}

// slowProcessAnalysisRootFromProcessInstance copies root process-instance fields into analysis output.
func slowProcessAnalysisRootFromProcessInstance(pi d.ProcessInstance, duration string, millis int64, available bool) d.SlowProcessAnalysisProcessInstance {
	return d.SlowProcessAnalysisProcessInstance{
		Key:                    pi.Key,
		TenantID:               pi.TenantId,
		BpmnProcessID:          pi.BpmnProcessId,
		ProcessDefinitionKey:   pi.ProcessDefinitionKey,
		ProcessVersion:         pi.ProcessVersion,
		State:                  pi.State,
		StartDate:              pi.StartDate,
		EndDate:                pi.EndDate,
		ParentKey:              pi.ParentKey,
		RootProcessInstanceKey: pi.RootProcessInstanceKey,
		Incident:               pi.Incident,
		Duration:               duration,
		DurationMillis:         millis,
		DurationAvailable:      available,
		Timeline:               []d.SlowProcessAnalysisTimelineEntry{},
	}
}
