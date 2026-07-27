// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/pool"
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
		result.PreflightScope = discovery.preflight
	default:
		return result, fmt.Errorf("%w: select process instances with explicit keys or one process-definition selector", d.ErrValidation)
	}

	frozenProgress := d.OpsFrozenScopeProgress{
		Phase:        slowProcessAnalysisEnrichmentProgressPhase(request),
		CoreResource: "process instance(s)",
		Total:        len(instances),
	}
	result.FrozenScopeProgress = &frozenProgress
	slowProcessAnalysisEmitProgress(request, slowProcessAnalysisFrozenScopeEvent(frozenProgress))

	enriched := d.ElementEnrichedProcessInstances{Items: make([]d.ElementEnrichedProcessInstance, 0, len(instances))}
	if len(instances) > 0 {
		if s.elementAPI == nil {
			return result, fmt.Errorf("%w: runtime element service is required for slow process analysis", d.ErrPrecondition)
		}
		if request.WithListeners {
			if s.jobAPI == nil {
				return result, fmt.Errorf("%w: runtime job service is required for slow process analysis listener enrichment", d.ErrPrecondition)
			}
		}
		var err error
		enriched, err = slowProcessAnalysisEnrichProcessInstances(ctx, s, request, instances, frozenProgress, opts...)
		if err != nil {
			return result, err
		}
		frozenProgress.Done = len(instances)
	}
	if len(instances) == 0 {
		frozenProgress.Done = len(instances)
	}
	result.FrozenScopeProgress = &frozenProgress
	if len(instances) == 0 {
		slowProcessAnalysisEmitProgress(request, slowProcessAnalysisFrozenScopeEvent(frozenProgress))
	}

	items := make([]d.SlowProcessAnalysisProcessInstance, 0, len(enriched.Items))
	for _, enrichedItem := range enriched.Items {
		item := slowProcessAnalysisProcessInstanceFromDomain(enrichedItem.Item, capturedAt)
		item.Timeline = slowProcessAnalysisCompleteTimeline(enrichedItem.Elements, capturedAt, item.DurationMillis, item.DurationAvailable)
		items = append(items, item)
	}
	items = slowProcessAnalysisApplyRootDurationFilter(items, request.RootDurationLonger)
	slowProcessAnalysisApplyComparisons(items)
	items = slowProcessAnalysisApplyDetailFilters(items, request.DetailFilters)
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

// slowProcessAnalysisEmitProgress keeps service progress callbacks optional and nil-safe.
func slowProcessAnalysisEmitProgress(request d.SlowProcessAnalysisRequest, event d.OpsProgressEvent) {
	if request.Progress == nil {
		return
	}
	request.Progress(event)
}

// slowProcessAnalysisFrozenScopeEvent builds a snapshot event so callbacks cannot observe later local mutations.
func slowProcessAnalysisFrozenScopeEvent(progress d.OpsFrozenScopeProgress) d.OpsProgressEvent {
	return d.OpsProgressEvent{
		Kind:        d.OpsProgressEventKindFrozenScope,
		FrozenScope: &progress,
	}
}

// slowProcessAnalysisEnrichmentProgressPhase picks the operator-facing frozen-scope phase for the requested enrichment path.
func slowProcessAnalysisEnrichmentProgressPhase(request d.SlowProcessAnalysisRequest) string {
	if request.WithListeners {
		return "loading listener jobs"
	}
	return "loading runtime elements"
}

// slowProcessAnalysisEnrichProcessInstances emits exact frozen-scope progress after each root enrichment succeeds.
func slowProcessAnalysisEnrichProcessInstances(ctx context.Context, s *Service, request d.SlowProcessAnalysisRequest, instances []d.ProcessInstance, progress d.OpsFrozenScopeProgress, opts ...services.CallOption) (d.ElementEnrichedProcessInstances, error) {
	out := d.ElementEnrichedProcessInstances{Items: make([]d.ElementEnrichedProcessInstance, 0, len(instances))}
	for i, pi := range instances {
		var (
			enriched d.ElementEnrichedProcessInstances
			err      error
		)
		if request.WithListeners {
			enriched, err = pisvc.EnrichProcessInstancesWithElementListeners(ctx, s.elementAPI, s.jobAPI, []d.ProcessInstance{pi}, opts...)
			if err != nil {
				return d.ElementEnrichedProcessInstances{}, fmt.Errorf("lookup runtime elements and listener jobs for slow analysis: %w", err)
			}
		} else {
			enriched, err = pisvc.EnrichProcessInstancesWithElements(ctx, s.elementAPI, []d.ProcessInstance{pi}, opts...)
			if err != nil {
				return d.ElementEnrichedProcessInstances{}, fmt.Errorf("lookup runtime elements for slow analysis: %w", err)
			}
		}
		out.Items = append(out.Items, enriched.Items...)
		progress.Done = i + 1
		slowProcessAnalysisEmitProgress(request, slowProcessAnalysisFrozenScopeEvent(progress))
	}
	out.Total = int32(len(out.Items))
	return out, nil
}

// slowProcessAnalysisApplyRootDurationFilter hides roots whose measured whole-process duration is not above the threshold.
func slowProcessAnalysisApplyRootDurationFilter(items []d.SlowProcessAnalysisProcessInstance, threshold time.Duration) []d.SlowProcessAnalysisProcessInstance {
	if threshold <= 0 {
		return items
	}
	out := make([]d.SlowProcessAnalysisProcessInstance, 0, len(items))
	for _, item := range items {
		if item.DurationAvailable && time.Duration(item.DurationMillis)*time.Millisecond > threshold {
			out = append(out, item)
		}
	}
	return out
}

// slowProcessAnalysisLookupExplicitKeys resolves already-deduplicated keys through tenant-safe lookup.
func slowProcessAnalysisLookupExplicitKeys(ctx context.Context, api pisvc.API, keys []string, opts ...services.CallOption) ([]d.ProcessInstance, error) {
	cfg := services.ApplyCallOptions(opts)
	workers := toolx.DetermineNoOfWorkers(len(keys), 0, cfg.NoWorkerLimit)
	return pool.ExecuteSlice[string, d.ProcessInstance](ctx, keys, workers, cfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstance, error) {
		pi, err := pisvc.LookupProcessInstance(ctx, api, key, opts...)
		if err != nil {
			return d.ProcessInstance{}, fmt.Errorf("lookup process instance %s: %w", key, err)
		}
		return pi, nil
	})
}

type slowProcessAnalysisSearchDiscovery struct {
	items     []d.ProcessInstance
	scope     d.DiscoveryScopeStatus
	preflight *d.OpsPreflightScope
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

	page, err := api.SearchForProcessInstancesPage(ctx, filter, pageReq, opts...)
	if err != nil {
		return slowProcessAnalysisSearchDiscovery{}, fmt.Errorf("discover process instances for slow analysis: %w", err)
	}
	preflight := slowProcessAnalysisPreflightScope(request, page, batchSize)
	discovery.preflight = &preflight
	slowProcessAnalysisEmitProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindPreflight, Preflight: &preflight})
	if request.ConfirmPreflight != nil {
		if err := request.ConfirmPreflight(preflight); err != nil {
			return slowProcessAnalysisSearchDiscovery{}, err
		}
	}

	for {
		var limited bool
		discovery, limited = slowProcessAnalysisCollectDiscoveryPage(discovery, request, page, seen)
		limitReached := limited || (request.Limit > 0 && len(discovery.items) >= int(request.Limit) && page.OverflowState == d.ProcessInstanceOverflowStateHasMore)
		slowProcessAnalysisEmitDiscoveryPageProgress(request, discovery, page, preflight, limitReached)
		if limitReached {
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
		page, err = api.SearchForProcessInstancesPage(ctx, filter, pageReq, opts...)
		if err != nil {
			return slowProcessAnalysisSearchDiscovery{}, fmt.Errorf("discover process instances for slow analysis: %w", err)
		}
	}
}

// slowProcessAnalysisCollectDiscoveryPage freezes unique candidates from one backend page and updates shared scope counters.
func slowProcessAnalysisCollectDiscoveryPage(discovery slowProcessAnalysisSearchDiscovery, request d.SlowProcessAnalysisRequest, page d.ProcessInstancePage, seen map[string]struct{}) (slowProcessAnalysisSearchDiscovery, bool) {
	discovery.scope.Pages++
	discovery.scope.CandidatesSeen += len(page.Items)
	for _, item := range page.Items {
		if request.Limit > 0 && len(discovery.items) >= int(request.Limit) {
			return discovery, true
		}
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		discovery.items = append(discovery.items, item)
	}
	return discovery, false
}

// slowProcessAnalysisEmitDiscoveryPageProgress publishes one discovery page snapshot after service-owned filtering and limit trimming.
func slowProcessAnalysisEmitDiscoveryPageProgress(request d.SlowProcessAnalysisRequest, discovery slowProcessAnalysisSearchDiscovery, page d.ProcessInstancePage, preflight d.OpsPreflightScope, limitReached bool) {
	progress := d.OpsPageProgress{
		Phase:            "discovering process instances",
		CurrentPage:      discovery.scope.Pages,
		PageCount:        preflight.PageCount,
		PageCountKind:    preflight.PageCountKind,
		PageSize:         discovery.scope.BatchSize,
		CurrentPageCount: len(page.Items),
		Seen:             discovery.scope.CandidatesSeen,
		Selected:         len(discovery.items),
		OverflowState:    slowProcessAnalysisOpsOverflowState(page.OverflowState),
		LimitReached:     limitReached,
	}
	slowProcessAnalysisEmitProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindPage, Page: &progress})
}

// slowProcessAnalysisOpsOverflowState maps process-instance continuation metadata into shared ops progress state.
func slowProcessAnalysisOpsOverflowState(state d.ProcessInstanceOverflowState) d.OpsOverflowState {
	switch state {
	case d.ProcessInstanceOverflowStateNoMore:
		return d.OpsOverflowStateNoMore
	case d.ProcessInstanceOverflowStateHasMore:
		return d.OpsOverflowStateHasMore
	case d.ProcessInstanceOverflowStateIndeterminate:
		return d.OpsOverflowStateIndeterminate
	default:
		return d.OpsOverflowStateUnknown
	}
}

// slowProcessAnalysisPreflightScope builds the cheap operator scope from the first reusable discovery page.
func slowProcessAnalysisPreflightScope(request d.SlowProcessAnalysisRequest, page d.ProcessInstancePage, batchSize int32) d.OpsPreflightScope {
	total, totalKind := slowProcessAnalysisPreflightTotal(page)
	pageCount, pageCountKind := slowProcessAnalysisPreflightPageCount(total, totalKind, batchSize)
	selectorSummary := slowProcessAnalysisSelectorSummary(request.ProcessDefinitionSelector)
	scope := d.OpsPreflightScope{
		Phase:           "preflight",
		Command:         request.CommandName,
		CoreResource:    "process_instance",
		SelectorSummary: selectorSummary,
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        batchSize,
		PageCount:       pageCount,
		PageCountKind:   pageCountKind,
		ConsequenceSummary: d.OpsConsequenceSummary{
			ResourceSummary: slowProcessAnalysisPreflightResourceSummary(total, totalKind),
			WorkSummary:     "slow analysis will discover all matches and load runtime element timelines for each selected process instance",
			RiskSummary:     "read-only expensive analysis",
		},
		RequiresConfirmation: true,
	}
	scope.ConsequenceSummary.ConfirmationText = slowProcessAnalysisPreflightConfirmationText(scope)
	return scope
}

// slowProcessAnalysisPreflightTotal maps backend total metadata without treating lower bounds as exact.
func slowProcessAnalysisPreflightTotal(page d.ProcessInstancePage) (*int64, d.OpsTotalCertainty) {
	if page.ReportedTotal == nil {
		return nil, d.OpsTotalCertaintyUnknown
	}
	count := page.ReportedTotal.Count
	switch page.ReportedTotal.Kind {
	case d.ProcessInstanceReportedTotalKindExact:
		return &count, d.OpsTotalCertaintyExact
	case d.ProcessInstanceReportedTotalKindLowerBound:
		return &count, d.OpsTotalCertaintyLowerBound
	default:
		return nil, d.OpsTotalCertaintyUnknown
	}
}

// slowProcessAnalysisPreflightPageCount derives exact or estimated page counts only from compatible total metadata.
func slowProcessAnalysisPreflightPageCount(total *int64, totalKind d.OpsTotalCertainty, batchSize int32) (*int64, d.OpsPageCountKind) {
	if total == nil || batchSize <= 0 {
		return nil, d.OpsPageCountKindUnknown
	}
	pages := (*total + int64(batchSize) - 1) / int64(batchSize)
	if pages < 1 {
		pages = 1
	}
	if totalKind == d.OpsTotalCertaintyLowerBound {
		return &pages, d.OpsPageCountKindEstimated
	}
	if totalKind == d.OpsTotalCertaintyExact {
		return &pages, d.OpsPageCountKindExact
	}
	return nil, d.OpsPageCountKindUnknown
}

// slowProcessAnalysisSelectorSummary keeps broad selector wording compact and free of debug filters.
func slowProcessAnalysisSelectorSummary(selector d.SlowProcessAnalysisProcessDefinitionSelector) string {
	if selector.BpmnProcessID != "" {
		return selector.BpmnProcessID
	}
	if selector.ProcessDefinitionKey != "" {
		return "process definition " + selector.ProcessDefinitionKey
	}
	return "process-definition selector"
}

// slowProcessAnalysisPreflightResourceSummary formats the count fact for structured consequence metadata.
func slowProcessAnalysisPreflightResourceSummary(total *int64, kind d.OpsTotalCertainty) string {
	if total == nil || kind == d.OpsTotalCertaintyUnknown {
		return "unknown process instance(s)"
	}
	if kind == d.OpsTotalCertaintyLowerBound {
		return fmt.Sprintf("%d+ process instance(s)", *total)
	}
	if kind == d.OpsTotalCertaintyEstimated {
		return fmt.Sprintf("about %d process instance(s)", *total)
	}
	return fmt.Sprintf("%d process instance(s)", *total)
}

// slowProcessAnalysisPreflightConfirmationText freezes the prompt body used before full discovery begins.
func slowProcessAnalysisPreflightConfirmationText(scope d.OpsPreflightScope) string {
	return fmt.Sprintf("Continue slow analysis for %s?", slowProcessAnalysisPreflightResourceSummary(scope.Total, scope.TotalKind))
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

// slowProcessAnalysisCompleteTimeline calculates complete element and transition timings before applying visibility filters.
func slowProcessAnalysisCompleteTimeline(elements []d.Element, capturedAt time.Time, processMillis int64, processDurationAvailable bool) []d.SlowProcessAnalysisTimelineEntry {
	elementRows := make([]d.SlowProcessAnalysisTimelineEntry, 0, len(elements))
	for _, element := range elements {
		elementRows = append(elementRows, slowProcessAnalysisElementEntry(element, capturedAt, processMillis, processDurationAvailable))
	}

	out := make([]d.SlowProcessAnalysisTimelineEntry, 0, len(elementRows)*2)
	for i, entry := range elementRows {
		out = append(out, entry)
		if i+1 >= len(elementRows) {
			continue
		}
		transition, ok := slowProcessAnalysisTransitionEntry(entry, elementRows[i+1], processMillis, processDurationAvailable)
		if ok {
			out = append(out, transition)
		}
	}
	return out
}

// slowProcessAnalysisApplyComparisons assigns percentile metadata from complete frozen analysis scopes.
func slowProcessAnalysisApplyComparisons(items []d.SlowProcessAnalysisProcessInstance) {
	rootGroups := map[string][]*d.SlowProcessAnalysisProcessInstance{}
	elementGroups := map[string][]*d.SlowProcessAnalysisTimelineEntry{}
	transitionGroups := map[string][]*d.SlowProcessAnalysisTimelineEntry{}

	for i := range items {
		item := &items[i]
		if item.DurationAvailable {
			rootGroups[item.ProcessDefinitionKey] = append(rootGroups[item.ProcessDefinitionKey], item)
		}
		for j := range item.Timeline {
			entry := &item.Timeline[j]
			if !entry.DurationAvailable {
				continue
			}
			switch entry.Kind {
			case d.SlowProcessAnalysisTimelineEntryKindElement:
				key := slowProcessAnalysisElementComparisonKey(item.ProcessDefinitionKey, *entry)
				elementGroups[key] = append(elementGroups[key], entry)
			case d.SlowProcessAnalysisTimelineEntryKindTransition:
				key := slowProcessAnalysisTransitionComparisonKey(item.ProcessDefinitionKey, *entry)
				transitionGroups[key] = append(transitionGroups[key], entry)
			}
		}
	}

	for _, group := range rootGroups {
		slowProcessAnalysisAssignRootComparison(group)
	}
	for _, group := range elementGroups {
		slowProcessAnalysisAssignTimelineComparison(group)
	}
	for _, group := range transitionGroups {
		slowProcessAnalysisAssignTimelineComparison(group)
	}
}

// slowProcessAnalysisApplyDetailFilters keeps only roots that still have matching detail rows when a detail filter is active.
func slowProcessAnalysisApplyDetailFilters(items []d.SlowProcessAnalysisProcessInstance, filters d.SlowProcessAnalysisDetailFilters) []d.SlowProcessAnalysisProcessInstance {
	dropEmptyRoots := slowProcessAnalysisDetailFiltersActive(filters)
	outItems := items[:0]
	for i := range items {
		source := items[i].Timeline
		out := make([]d.SlowProcessAnalysisTimelineEntry, 0, len(source))
		for j, entry := range source {
			switch entry.Kind {
			case d.SlowProcessAnalysisTimelineEntryKindTransition:
				from, to, ok := slowProcessAnalysisAdjacentEndpoints(source, j)
				if ok && slowProcessAnalysisTransitionVisible(entry, from, to, filters) {
					out = append(out, entry)
				}
			default:
				if slowProcessAnalysisElementVisible(entry, filters) {
					out = append(out, entry)
				}
			}
		}
		items[i].Timeline = out
		if !dropEmptyRoots || len(out) > 0 {
			outItems = append(outItems, items[i])
		}
	}
	return outItems
}

// slowProcessAnalysisDetailFiltersActive reports whether detail predicates should narrow result roots.
func slowProcessAnalysisDetailFiltersActive(filters d.SlowProcessAnalysisDetailFilters) bool {
	return filters.ElementID != "" ||
		filters.Type != "" ||
		filters.ElementState != "" ||
		filters.DurationAfter > 0
}

// slowProcessAnalysisAdjacentEndpoints returns the original element rows surrounding a transition.
func slowProcessAnalysisAdjacentEndpoints(entries []d.SlowProcessAnalysisTimelineEntry, transitionIndex int) (d.SlowProcessAnalysisTimelineEntry, d.SlowProcessAnalysisTimelineEntry, bool) {
	if transitionIndex <= 0 || transitionIndex+1 >= len(entries) {
		return d.SlowProcessAnalysisTimelineEntry{}, d.SlowProcessAnalysisTimelineEntry{}, false
	}
	from := entries[transitionIndex-1]
	to := entries[transitionIndex+1]
	if from.Kind != d.SlowProcessAnalysisTimelineEntryKindElement || to.Kind != d.SlowProcessAnalysisTimelineEntryKindElement {
		return d.SlowProcessAnalysisTimelineEntry{}, d.SlowProcessAnalysisTimelineEntry{}, false
	}
	return from, to, true
}

func slowProcessAnalysisElementComparisonKey(processDefinitionKey string, entry d.SlowProcessAnalysisTimelineEntry) string {
	return strings.Join([]string{processDefinitionKey, entry.ElementID, entry.Type}, "\x00")
}

func slowProcessAnalysisTransitionComparisonKey(processDefinitionKey string, entry d.SlowProcessAnalysisTimelineEntry) string {
	return strings.Join([]string{processDefinitionKey, entry.FromElementID, entry.FromElementType, entry.ToElementID, entry.ToElementType}, "\x00")
}

func slowProcessAnalysisAssignRootComparison(group []*d.SlowProcessAnalysisProcessInstance) {
	if len(group) < 3 {
		return
	}
	values := make([]int64, 0, len(group))
	for _, item := range group {
		values = append(values, item.DurationMillis)
	}
	for _, item := range group {
		item.ComparisonSampleCount = len(group)
		item.RelativePercentile = slowProcessAnalysisRelativePercentile(item.DurationMillis, values)
		item.RelativeBar = slowProcessAnalysisRelativeBar(item.RelativePercentile)
	}
}

func slowProcessAnalysisAssignTimelineComparison(group []*d.SlowProcessAnalysisTimelineEntry) {
	if len(group) < 3 {
		return
	}
	values := make([]int64, 0, len(group))
	for _, item := range group {
		values = append(values, item.DurationMillis)
	}
	for _, item := range group {
		item.ComparisonSampleCount = len(group)
		item.RelativePercentile = slowProcessAnalysisRelativePercentile(item.DurationMillis, values)
		item.RelativeBar = slowProcessAnalysisRelativeBar(item.RelativePercentile)
	}
}

// slowProcessAnalysisRelativePercentile ranks a duration by shorter plus half of equal samples.
func slowProcessAnalysisRelativePercentile(value int64, values []int64) int {
	if len(values) == 0 {
		return 0
	}
	shorter := 0
	equal := 0
	for _, candidate := range values {
		switch {
		case candidate < value:
			shorter++
		case candidate == value:
			equal++
		}
	}
	return int(math.Round((float64(shorter) + float64(equal)/2) * 100 / float64(len(values))))
}

// slowProcessAnalysisRelativeBar renders ten ASCII cells from the rounded percentile.
func slowProcessAnalysisRelativeBar(percentile int) string {
	if percentile <= 0 {
		return "[----------]"
	}
	if percentile > 100 {
		percentile = 100
	}
	filled := int(math.Round(float64(percentile) / 10))
	if filled > 10 {
		filled = 10
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", 10-filled) + "]"
}

// slowProcessAnalysisElementEntry copies runtime element identity and adds measured duration fields.
func slowProcessAnalysisElementEntry(element d.Element, capturedAt time.Time, processMillis int64, processDurationAvailable bool) d.SlowProcessAnalysisTimelineEntry {
	duration, millis, available := slowProcessAnalysisElementDuration(element, capturedAt)
	return d.SlowProcessAnalysisTimelineEntry{
		Kind:                 d.SlowProcessAnalysisTimelineEntryKindElement,
		ElementInstanceKey:   element.ElementInstanceKey,
		ElementID:            element.ElementId,
		Type:                 element.Type,
		State:                element.State,
		StartDate:            element.StartDate,
		EndDate:              element.EndDate,
		HasIncident:          element.HasIncident,
		IncidentKey:          element.IncidentKey,
		Duration:             duration,
		DurationMillis:       millis,
		DurationAvailable:    available,
		ProcessDurationShare: slowProcessAnalysisProcessDurationShare(millis, available, processMillis, processDurationAvailable),
		Listeners:            element.Listeners,
	}
}

// slowProcessAnalysisElementDuration returns active element durations from the captured analysis time only for runtime-active elements.
func slowProcessAnalysisElementDuration(element d.Element, capturedAt time.Time) (string, int64, bool) {
	start, err := time.Parse(time.RFC3339Nano, element.StartDate)
	if err != nil || start.IsZero() {
		return "", 0, false
	}
	var end time.Time
	switch {
	case element.EndDate != "":
		end, err = time.Parse(time.RFC3339Nano, element.EndDate)
		if err != nil || end.Before(start) {
			return "", 0, false
		}
	case strings.EqualFold(element.State, "ACTIVE"):
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

// slowProcessAnalysisTransitionEntry measures only adjacent chronological elements with explicit end/start timestamps.
func slowProcessAnalysisTransitionEntry(from d.SlowProcessAnalysisTimelineEntry, to d.SlowProcessAnalysisTimelineEntry, processMillis int64, processDurationAvailable bool) (d.SlowProcessAnalysisTimelineEntry, bool) {
	fromEnd, err := time.Parse(time.RFC3339Nano, from.EndDate)
	if err != nil || fromEnd.IsZero() {
		return d.SlowProcessAnalysisTimelineEntry{}, false
	}
	toStart, err := time.Parse(time.RFC3339Nano, to.StartDate)
	if err != nil || toStart.IsZero() || toStart.Before(fromEnd) {
		return d.SlowProcessAnalysisTimelineEntry{}, false
	}
	duration := toStart.Sub(fromEnd)
	millis := duration.Milliseconds()
	return d.SlowProcessAnalysisTimelineEntry{
		Kind:                   d.SlowProcessAnalysisTimelineEntryKindTransition,
		FromElementInstanceKey: from.ElementInstanceKey,
		FromElementID:          from.ElementID,
		FromElementType:        from.Type,
		FromEndDate:            from.EndDate,
		ToElementInstanceKey:   to.ElementInstanceKey,
		ToElementID:            to.ElementID,
		ToElementType:          to.Type,
		ToStartDate:            to.StartDate,
		Duration:               duration.String(),
		DurationMillis:         millis,
		DurationAvailable:      true,
		ProcessDurationShare:   slowProcessAnalysisProcessDurationShare(millis, true, processMillis, processDurationAvailable),
	}, true
}

// slowProcessAnalysisProcessDurationShare rounds a detail duration as a percentage of the measured root duration.
func slowProcessAnalysisProcessDurationShare(millis int64, available bool, processMillis int64, processDurationAvailable bool) int {
	if !available || !processDurationAvailable || processMillis <= 0 {
		return 0
	}
	return int(math.Round(float64(millis) * 100 / float64(processMillis)))
}

// slowProcessAnalysisElementVisible applies detail filters to element rows after all timings are calculated.
func slowProcessAnalysisElementVisible(entry d.SlowProcessAnalysisTimelineEntry, filters d.SlowProcessAnalysisDetailFilters) bool {
	return slowProcessAnalysisElementMatchesPredicates(entry, filters) && slowProcessAnalysisDurationPassesFilter(entry.DurationAvailable, entry.DurationMillis, filters.DurationAfter)
}

// slowProcessAnalysisTransitionVisible keeps original adjacent transitions when an endpoint matches active element predicates.
func slowProcessAnalysisTransitionVisible(entry d.SlowProcessAnalysisTimelineEntry, from d.SlowProcessAnalysisTimelineEntry, to d.SlowProcessAnalysisTimelineEntry, filters d.SlowProcessAnalysisDetailFilters) bool {
	predicateMatch := slowProcessAnalysisElementMatchesPredicates(from, filters) || slowProcessAnalysisElementMatchesPredicates(to, filters)
	return predicateMatch && slowProcessAnalysisDurationPassesFilter(entry.DurationAvailable, entry.DurationMillis, filters.DurationAfter)
}

// slowProcessAnalysisElementMatchesPredicates checks only element identity predicates, leaving duration filtering separate.
func slowProcessAnalysisElementMatchesPredicates(entry d.SlowProcessAnalysisTimelineEntry, filters d.SlowProcessAnalysisDetailFilters) bool {
	if filters.ElementID != "" && entry.ElementID != filters.ElementID {
		return false
	}
	if filters.Type != "" && !strings.EqualFold(entry.Type, filters.Type) {
		return false
	}
	if filters.ElementState != "" && !strings.EqualFold(entry.State, filters.ElementState) {
		return false
	}
	return true
}

// slowProcessAnalysisDurationPassesFilter applies the detail duration threshold only to measured detail rows.
func slowProcessAnalysisDurationPassesFilter(available bool, millis int64, threshold time.Duration) bool {
	if threshold <= 0 {
		return true
	}
	return available && time.Duration(millis)*time.Millisecond > threshold
}
