// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
)

func retentionPolicyDiscoveryProgress(request d.RetentionPolicyRequest) func(pisvc.RetentionDiscoveryProgress) {
	if request.Progress == nil {
		return nil
	}
	return func(event pisvc.RetentionDiscoveryProgress) {
		if event.PageNumber == 1 {
			preflight := newProcessInstancePurgePreflight(
				request.CommandName,
				"retention-policy",
				"retention-policy will discover eligible process instances, validate delete impact, and delete confirmed roots",
				event.Page,
				normalizeRetentionDiscoveryBatchSizeForProgress(request.BatchSize),
				!request.DryRun && !request.AutoConfirm && !request.Automation,
			)
			request.Progress(d.OpsProgressEvent{Kind: d.OpsProgressEventKindPreflight, Preflight: &preflight})
		}
		pageCount, pageKind := processInstancePurgePageCount(event.Page.ReportedTotal, event.Page.Request.Size)
		progress := d.OpsPageProgress{
			Phase:            "discovering retention process instances",
			CurrentPage:      event.PageNumber,
			PageCount:        pageCount,
			PageCountKind:    pageKind,
			PageSize:         event.Page.Request.Size,
			CurrentPageCount: event.CurrentPageCount,
			Seen:             event.CandidatesSeen,
			Selected:         event.CandidatesFrozen,
			OverflowState:    d.OpsOverflowState(event.Page.OverflowState),
			LimitReached:     event.LimitReached,
		}
		request.Progress(d.OpsProgressEvent{Kind: d.OpsProgressEventKindPage, Page: &progress})
	}
}

func orphanPurgeDiscoveryProgress(request d.OrphanPurgeRequest) func(pisvc.OrphanDiscoveryProgress) {
	if request.Progress == nil {
		return nil
	}
	return func(event pisvc.OrphanDiscoveryProgress) {
		if event.Page == 1 && event.Phase == "checking" {
			preflight := newOrphanPurgePreflight(request, event)
			request.Progress(d.OpsProgressEvent{Kind: d.OpsProgressEventKindPreflight, Preflight: &preflight})
		}
		if event.Phase != "checked" {
			return
		}
		pageCount, pageKind := orphanPurgePageCount(event)
		progress := d.OpsPageProgress{
			Phase:            "discovering orphan process-instance candidates",
			CurrentPage:      event.Page,
			PageCount:        pageCount,
			PageCountKind:    pageKind,
			PageSize:         normalizeOrphanDiscoveryBatchSizeForProgress(request.BatchSize),
			CurrentPageCount: event.CurrentPageCandidates,
			Seen:             event.CandidatesChecked,
			Selected:         event.OrphansFound,
			OverflowState:    d.OpsOverflowState(event.OverflowState),
			LimitReached:     event.Limit > 0 && event.OrphansFound >= int(event.Limit),
		}
		request.Progress(d.OpsProgressEvent{Kind: d.OpsProgressEventKindPage, Page: &progress})
		total := event.CandidatesChecked
		reportOpsFrozenScopeProgress(request.Progress, "checking orphan process-instance parents", event.CandidatesChecked, total)
	}
}

func newOrphanPurgePreflight(request d.OrphanPurgeRequest, event pisvc.OrphanDiscoveryProgress) d.OpsPreflightScope {
	total, totalKind := orphanPurgeTotal(event)
	pageCount, pageKind := orphanPurgePageCount(event)
	return d.OpsPreflightScope{
		Phase:           "preflight",
		Command:         request.CommandName,
		CoreResource:    "process_instance",
		SelectorSummary: "orphan-process-instances purge",
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        normalizeOrphanDiscoveryBatchSizeForProgress(request.BatchSize),
		PageCount:       pageCount,
		PageCountKind:   pageKind,
		ConsequenceSummary: d.OpsConsequenceSummary{
			WorkSummary: "orphan-process-instances purge will discover child candidates, check parent existence, validate delete impact, and delete confirmed roots",
			RiskSummary: "potentially destructive purge",
		},
		RequiresConfirmation: !request.DryRun && !request.AutoConfirm && !request.Automation,
	}
}

func orphanPurgeTotal(event pisvc.OrphanDiscoveryProgress) (*int64, d.OpsTotalCertainty) {
	if event.Page == 1 && event.OverflowState == d.ProcessInstanceOverflowStateNoMore {
		total := int64(event.CurrentPageCandidates)
		return &total, d.OpsTotalCertaintyExact
	}
	return nil, d.OpsTotalCertaintyUnknown
}

func orphanPurgePageCount(event pisvc.OrphanDiscoveryProgress) (*int64, d.OpsPageCountKind) {
	if event.Page == 1 && event.OverflowState == d.ProcessInstanceOverflowStateNoMore {
		pages := int64(1)
		return &pages, d.OpsPageCountKindExact
	}
	return nil, d.OpsPageCountKindUnknown
}

func newProcessInstancePurgePreflight(command string, selector string, workSummary string, page d.ProcessInstancePage, pageSize int32, requiresConfirmation bool) d.OpsPreflightScope {
	total, totalKind := processInstancePurgeTotal(page.ReportedTotal)
	pageCount, pageKind := processInstancePurgePageCount(page.ReportedTotal, pageSize)
	return d.OpsPreflightScope{
		Phase:           "preflight",
		Command:         command,
		CoreResource:    "process_instance",
		SelectorSummary: selector,
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        pageSize,
		PageCount:       pageCount,
		PageCountKind:   pageKind,
		ConsequenceSummary: d.OpsConsequenceSummary{
			WorkSummary: workSummary,
			RiskSummary: "potentially destructive purge",
		},
		RequiresConfirmation: requiresConfirmation,
	}
}

func processInstancePurgeTotal(total *d.ProcessInstanceReportedTotal) (*int64, d.OpsTotalCertainty) {
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

func processInstancePurgePageCount(total *d.ProcessInstanceReportedTotal, pageSize int32) (*int64, d.OpsPageCountKind) {
	if total == nil || pageSize <= 0 {
		return nil, d.OpsPageCountKindUnknown
	}
	pages := (total.Count + int64(pageSize) - 1) / int64(pageSize)
	if pages < 1 {
		pages = 1
	}
	switch total.Kind {
	case d.ProcessInstanceReportedTotalKindExact:
		return &pages, d.OpsPageCountKindExact
	case d.ProcessInstanceReportedTotalKindLowerBound:
		return &pages, d.OpsPageCountKindEstimated
	default:
		return nil, d.OpsPageCountKindUnknown
	}
}

func processInstancePurgeFrozenTotal(total *d.ProcessInstanceReportedTotal, fallback int) int {
	if total != nil && total.Kind == d.ProcessInstanceReportedTotalKindExact && total.Count >= 0 {
		return int(total.Count)
	}
	return fallback
}

func incidentPurgeTotal(total *d.IncidentReportedTotal) (*int64, d.OpsTotalCertainty) {
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

func newIncidentPurgePreflight(request d.IncidentPurgeRequest, page d.IncidentPage) d.OpsPreflightScope {
	total, totalKind := incidentPurgeTotal(page.ReportedTotal)
	pageCount, pageKind := incidentPurgePageCount(page.ReportedTotal, page.Request.Size)
	return d.OpsPreflightScope{
		Phase:           "preflight",
		Command:         request.CommandName,
		CoreResource:    "incident",
		SelectorSummary: "process-instances-with-incidents purge",
		Total:           total,
		TotalKind:       totalKind,
		PageSize:        page.Request.Size,
		PageCount:       pageCount,
		PageCountKind:   pageKind,
		ConsequenceSummary: d.OpsConsequenceSummary{
			WorkSummary: "process-instances-with-incidents purge will discover matching incidents, validate delete impact, and delete confirmed roots",
			RiskSummary: "potentially destructive purge",
		},
		RequiresConfirmation: !request.DryRun && !request.AutoConfirm && !request.Automation,
	}
}

func incidentPurgePageCount(total *d.IncidentReportedTotal, pageSize int32) (*int64, d.OpsPageCountKind) {
	if total == nil || pageSize <= 0 {
		return nil, d.OpsPageCountKindUnknown
	}
	pages := (total.Count + int64(pageSize) - 1) / int64(pageSize)
	if pages < 1 {
		pages = 1
	}
	switch total.Kind {
	case d.IncidentReportedTotalKindExact:
		return &pages, d.OpsPageCountKindExact
	case d.IncidentReportedTotalKindLowerBound:
		return &pages, d.OpsPageCountKindEstimated
	default:
		return nil, d.OpsPageCountKindUnknown
	}
}

func reportIncidentPurgePageProgress(request d.IncidentPurgeRequest, page d.IncidentPage, discovery d.IncidentDiscoveryResult, currentSelected int, limited bool) {
	if request.Progress == nil {
		return
	}
	pageCount, pageKind := incidentPurgePageCount(page.ReportedTotal, page.Request.Size)
	progress := d.OpsPageProgress{
		Phase:            "discovering incidents",
		CurrentPage:      discovery.Pages,
		PageCount:        pageCount,
		PageCountKind:    pageKind,
		PageSize:         page.Request.Size,
		CurrentPageCount: len(page.Items),
		Seen:             discovery.CandidatesSeen,
		Selected:         len(discovery.CandidateIncidents),
		OverflowState:    d.OpsOverflowState(page.OverflowState),
		LimitReached:     limited,
	}
	if currentSelected == 0 {
		progress.Selected = len(discovery.CandidateIncidents)
	}
	request.Progress(d.OpsProgressEvent{Kind: d.OpsProgressEventKindPage, Page: &progress})
}

func reportOpsFrozenScopeProgress(progress func(d.OpsProgressEvent), phase string, done int, total int) {
	if progress == nil || total <= 0 {
		return
	}
	if done > total {
		done = total
	}
	progress(d.OpsProgressEvent{
		Kind: d.OpsProgressEventKindFrozenScope,
		FrozenScope: &d.OpsFrozenScopeProgress{
			Phase:        phase,
			CoreResource: "process instance(s)",
			Done:         done,
			Total:        total,
		},
	})
}

func normalizeRetentionDiscoveryBatchSizeForProgress(size int32) int32 {
	if size <= 0 {
		return consts.MaxPISearchSize
	}
	return size
}

func normalizeOrphanDiscoveryBatchSizeForProgress(size int32) int32 {
	if size <= 0 {
		return consts.MaxPISearchSize
	}
	return size
}
