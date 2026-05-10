// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
)

// pagingReportedTotal captures total-count metadata in the shape needed for
// continuation prompts without exposing command rendering to facade model
// differences between process-instance and incident searches.
type pagingReportedTotal struct {
	Available  bool
	Count      int64
	LowerBound bool
}

// formatPagingProgress renders a cumulative count with an optional exact or
// lower-bound reported total and the operation label shown in continuation
// prompts.
func formatPagingProgress(cumulative int, total pagingReportedTotal, label string) string {
	if !total.Available {
		return fmt.Sprintf("%d %s", cumulative, label)
	}
	suffix := ""
	if total.LowerBound {
		suffix = "+"
	}
	return fmt.Sprintf("%d/%d%s %s", cumulative, total.Count, suffix, label)
}

// processInstancePagingReportedTotal returns total metadata only when it still
// describes the user-visible process-instance result after command-local
// filtering.
func processInstancePagingReportedTotal(page process.ProcessInstancePage) pagingReportedTotal {
	if !canUsePIReportedTotal() || page.ReportedTotal == nil {
		return pagingReportedTotal{}
	}
	return pagingReportedTotal{
		Available:  true,
		Count:      page.ReportedTotal.Count,
		LowerBound: page.ReportedTotal.Kind == process.ProcessInstanceReportedTotalKindLowerBound,
	}
}

// formatProcessInstancePagingProgress renders process-instance progress for a
// continuation prompt, including exact totals or lower-bound totals when the
// backend supplied metadata that remains valid for the command output.
func formatProcessInstancePagingProgress(page process.ProcessInstancePage, cumulative int, label string) string {
	return formatPagingProgress(cumulative, processInstancePagingReportedTotal(page), label)
}

// incidentPagingReportedTotal returns total metadata for incident search pages.
// Incident services omit this value when local filtering makes the server total
// unsafe to display, so the command can use any reported value directly.
func incidentPagingReportedTotal(page incident.Page) pagingReportedTotal {
	if page.ReportedTotal == nil {
		return pagingReportedTotal{}
	}
	return pagingReportedTotal{
		Available:  true,
		Count:      page.ReportedTotal.Count,
		LowerBound: page.ReportedTotal.Kind == incident.ReportedTotalKindLowerBound,
	}
}

// formatIncidentPagingProgress renders incident progress for a continuation
// prompt, including exact totals or lower-bound totals when present.
func formatIncidentPagingProgress(page incident.Page, cumulative int, label string) string {
	return formatPagingProgress(cumulative, incidentPagingReportedTotal(page), label)
}
