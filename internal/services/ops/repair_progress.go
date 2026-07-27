// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"math"

	d "github.com/grafvonb/c8volt/internal/domain"
)

// emitRepairProgress sends one structured repair progress event when a caller installed a callback.
func emitRepairProgress(request d.OpsRepairRequest, event d.OpsProgressEvent) {
	if request.Progress == nil {
		return
	}
	request.Progress(event)
}

// emitRepairFrozenScopeProgress reports exact counters for an immutable repair phase.
func emitRepairFrozenScopeProgress(request d.OpsRepairRequest, phase string, resource string, done int, total int) {
	if request.Progress == nil || total <= 0 {
		return
	}
	if done > total {
		done = total
	}
	progress := d.OpsFrozenScopeProgress{
		Phase:        phase,
		CoreResource: resource,
		Done:         done,
		Total:        total,
	}
	emitRepairProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindFrozenScope, FrozenScope: &progress})
}

// repairPlanningProgressScope selects the frozen repair population shown during dry-run planning.
func repairPlanningProgressScope(request d.OpsRepairRequest, result d.OpsRepairResult) (string, string, int) {
	switch request.Target {
	case d.OpsRepairTargetProcessInstance:
		return "planning process-instance repair scope", "process instance(s)", len(result.FrozenSet.ProcessInstanceKeys)
	default:
		return "planning incident repair scope", "incident(s)", len(result.FrozenSet.IncidentKeys)
	}
}

// emitRepairIncidentPreflight reports the first-page scope for incident search repair.
func emitRepairIncidentPreflight(request d.OpsRepairRequest, page d.IncidentPage) {
	total, totalKind := repairIncidentTotal(page.ReportedTotal)
	pageCount, pageKind := repairIncidentPageCount(page.ReportedTotal, page.Request.Size)
	preflight := d.OpsPreflightScope{
		Phase:           "preflight",
		Command:         request.CommandName,
		CoreResource:    "incident",
		SelectorSummary: "incident repair",
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        page.Request.Size,
		PageCount:       pageCount,
		PageCountKind:   pageKind,
		ConsequenceSummary: d.OpsConsequenceSummary{
			WorkSummary: "incident repair will discover matching incidents, plan repair scope, and repair confirmed incidents",
			RiskSummary: "state-changing repair",
		},
		RequiresConfirmation: !request.DryRun && !request.AutoConfirm && !request.Automation,
	}
	emitRepairProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindPreflight, Preflight: &preflight})
}

// emitRepairIncidentPageProgress reports incident search discovery after each page has been consumed.
func emitRepairIncidentPageProgress(request d.OpsRepairRequest, page d.IncidentPage, status d.DiscoveryScopeStatus, selected int, limited bool) {
	pageCount, pageKind := repairIncidentPageCount(page.ReportedTotal, page.Request.Size)
	progress := d.OpsPageProgress{
		Phase:            "discovering repair incidents",
		CurrentPage:      status.Pages,
		PageCount:        pageCount,
		PageCountKind:    pageKind,
		PageSize:         page.Request.Size,
		CurrentPageCount: len(page.Items),
		Seen:             status.CandidatesSeen,
		Selected:         selected,
		OverflowState:    d.OpsOverflowState(page.OverflowState),
		LimitReached:     limited,
	}
	emitRepairProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindPage, Page: &progress})
}

// emitRepairProcessInstancePreflight reports the first-page scope for process-instance repair search.
func emitRepairProcessInstancePreflight(request d.OpsRepairRequest, page d.ProcessInstancePage) {
	total, totalKind := repairProcessInstanceTotal(page.ReportedTotal)
	pageCount, pageKind := repairProcessInstancePageCount(page.ReportedTotal, page.Request.Size)
	preflight := d.OpsPreflightScope{
		Phase:           "preflight",
		Command:         request.CommandName,
		CoreResource:    "process_instance",
		SelectorSummary: "process-instance repair",
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        page.Request.Size,
		PageCount:       pageCount,
		PageCountKind:   pageKind,
		ConsequenceSummary: d.OpsConsequenceSummary{
			WorkSummary: "process-instance repair will discover incident-bearing process instances, load active incidents, plan repair scope, and repair confirmed incidents",
			RiskSummary: "state-changing repair",
		},
		RequiresConfirmation: !request.DryRun && !request.AutoConfirm && !request.Automation,
	}
	emitRepairProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindPreflight, Preflight: &preflight})
}

// emitRepairProcessInstancePageProgress reports process-instance search discovery after each page has been consumed.
func emitRepairProcessInstancePageProgress(request d.OpsRepairRequest, page d.ProcessInstancePage, status d.DiscoveryScopeStatus, selected int, limited bool) {
	pageCount, pageKind := repairProcessInstancePageCount(page.ReportedTotal, page.Request.Size)
	progress := d.OpsPageProgress{
		Phase:            "discovering repair process instances",
		CurrentPage:      status.Pages,
		PageCount:        pageCount,
		PageCountKind:    pageKind,
		PageSize:         page.Request.Size,
		CurrentPageCount: len(page.Items),
		Seen:             status.CandidatesSeen,
		Selected:         selected,
		OverflowState:    d.OpsOverflowState(page.OverflowState),
		LimitReached:     limited,
	}
	emitRepairProgress(request, d.OpsProgressEvent{Kind: d.OpsProgressEventKindPage, Page: &progress})
}

// repairIncidentTotal maps backend incident total certainty for repair preflight.
func repairIncidentTotal(total *d.IncidentReportedTotal) (*int64, d.OpsTotalCertainty) {
	if total == nil {
		return nil, d.OpsTotalCertaintyUnknown
	}
	count := total.Count
	switch total.Kind {
	case d.IncidentReportedTotalKindExact:
		return &count, d.OpsTotalCertaintyExact
	case d.IncidentReportedTotalKindLowerBound:
		return &count, d.OpsTotalCertaintyLowerBound
	default:
		return nil, d.OpsTotalCertaintyUnknown
	}
}

// repairIncidentPageCount derives incident repair page count from usable reported totals.
func repairIncidentPageCount(total *d.IncidentReportedTotal, pageSize int32) (*int64, d.OpsPageCountKind) {
	if total == nil || pageSize <= 0 {
		return nil, d.OpsPageCountKindUnknown
	}
	pages := int64(math.Ceil(float64(total.Count) / float64(pageSize)))
	switch total.Kind {
	case d.IncidentReportedTotalKindExact:
		return &pages, d.OpsPageCountKindExact
	case d.IncidentReportedTotalKindLowerBound:
		return &pages, d.OpsPageCountKindEstimated
	default:
		return nil, d.OpsPageCountKindUnknown
	}
}

// repairProcessInstanceTotal maps backend process-instance total certainty for repair preflight.
func repairProcessInstanceTotal(total *d.ProcessInstanceReportedTotal) (*int64, d.OpsTotalCertainty) {
	if total == nil {
		return nil, d.OpsTotalCertaintyUnknown
	}
	count := total.Count
	switch total.Kind {
	case d.ProcessInstanceReportedTotalKindExact:
		return &count, d.OpsTotalCertaintyExact
	case d.ProcessInstanceReportedTotalKindLowerBound:
		return &count, d.OpsTotalCertaintyLowerBound
	default:
		return nil, d.OpsTotalCertaintyUnknown
	}
}

// repairProcessInstancePageCount derives process-instance repair page count from usable reported totals.
func repairProcessInstancePageCount(total *d.ProcessInstanceReportedTotal, pageSize int32) (*int64, d.OpsPageCountKind) {
	if total == nil || pageSize <= 0 {
		return nil, d.OpsPageCountKindUnknown
	}
	pages := int64(math.Ceil(float64(total.Count) / float64(pageSize)))
	switch total.Kind {
	case d.ProcessInstanceReportedTotalKindExact:
		return &pages, d.OpsPageCountKindExact
	case d.ProcessInstanceReportedTotalKindLowerBound:
		return &pages, d.OpsPageCountKindEstimated
	default:
		return nil, d.OpsPageCountKindUnknown
	}
}
