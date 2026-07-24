// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
)

// searchProcessInstancesTotal implements `get pi --total`. It uses exact
// backend totals when they still describe the command's final output, and falls
// back to page-by-page counting when client-side filters or lower-bound totals
// make the reported metadata insufficient.
func searchProcessInstancesTotal(cmd *cobra.Command, log *slog.Logger, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (int64, error) {
	request := newProcessInstanceSearchRequest(cmd, cfg, filter)
	stopActivity := func() {}
	countingByPaging := false
	defer func() {
		stopActivity()
	}()

	return cli.SearchProcessInstancesTotal(cmd.Context(), request, func(step process.ProcessInstanceSearchTotalStep) error {
		logPITotalPage(cmd, log, step.Page.Request, step.Page, step.TotalBefore)
		if step.ExactTotalUsed {
			return nil
		}
		if !countingByPaging {
			stopActivity = startCommandActivity(cmd, "counting process instances page by page")
			countingByPaging = true
		}
		summary := newPIProgressSummary(step.Page, int(step.TotalAfter), true)
		if cmd != nil {
			logging.UpdateActivity(cmd.Context(), formatPITotalActivityProgress(summary))
		}
		logPISearchProgress(cmd, log, summary)
		return nil
	}, collectOptions()...)
}

// formatPITotalActivityProgress keeps the transient --total activity indicator
// intentionally short. Verbose mode still logs the full pagination diagnostic,
// but the spinner line must fit narrow terminals so carriage-return redraws do
// not wrap into visible output.
func formatPITotalActivityProgress(summary processInstanceProgressSummary) string {
	status := "fetching next page"
	switch summary.ContinuationState {
	case processInstanceContinuationCompleted:
		status = "complete"
	case processInstanceContinuationWarningStop:
		status = "stopped with warning"
	case processInstanceContinuationLimitReached:
		status = "limit reached"
	}
	return fmt.Sprintf("counting process instances: %d counted, %s", summary.CumulativeCount, status)
}

// logPITotalPage records the paging metadata that explains how a total was
// computed. It is intentionally verbose because operators debugging count
// mismatches need to see offset/cursor mode, backend total metadata, and local
// accumulation in one place.
func logPITotalPage(cmd *cobra.Command, log *slog.Logger, req process.ProcessInstancePageRequest, page process.ProcessInstancePage, totalBefore int64) {
	if cmd == nil || log == nil {
		return
	}
	mode := "offset"
	if req.After != "" {
		mode = "cursor"
	}
	reportedTotal := int64(-1)
	reportedKind := "unavailable"
	if page.ReportedTotal != nil {
		reportedTotal = page.ReportedTotal.Count
		reportedKind = string(page.ReportedTotal.Kind)
	}
	log.DebugContext(cmd.Context(), fmt.Sprintf(
		"pi total page; mode %s, from %d, after %q, limit %d, items %d, total before %d, total after %d, overflow %s, reported total %d, reported kind %s, end cursor %q",
		mode,
		req.From,
		req.After,
		req.Size,
		len(page.Items),
		totalBefore,
		totalBefore+int64(len(page.Items)),
		page.OverflowState,
		reportedTotal,
		reportedKind,
		page.EndCursor,
	))
}
