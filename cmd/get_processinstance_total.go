// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// searchProcessInstancesTotal implements `get pi --total`. It uses exact
// backend totals when they still describe the command's final output, and falls
// back to page-by-page counting when client-side filters or lower-bound totals
// make the reported metadata insufficient.
func searchProcessInstancesTotal(cmd *cobra.Command, log *slog.Logger, cli process.API, cfg *config.Config, filter process.ProcessInstanceFilter) (int64, error) {
	pageReq := newPISearchPageRequest(cmd, cfg, 0)
	total := int64(0)
	stopActivity := func() {}
	countingByPaging := false
	defer func() {
		stopActivity()
	}()

	for {
		page, err := cli.SearchProcessInstancesPage(cmd.Context(), filter, pageReq, collectOptions()...)
		if err != nil {
			return 0, err
		}
		logPITotalPage(cmd, log, pageReq, page, total)

		if canUsePIExactReportedTotal(page) {
			return page.ReportedTotal.Count, nil
		}
		if !countingByPaging {
			stopActivity = startCommandActivity(cmd, "counting process instances page by page")
			countingByPaging = true
		}

		filtered, err := applyPISearchResultFilters(cmd, cli, process.ProcessInstances{
			Total: int32(len(page.Items)),
			Items: page.Items,
		})
		if err != nil {
			return 0, err
		}

		total += int64(len(filtered.Items))
		logPISearchProgress(cmd, log, newPIProgressSummary(page, int(total), true))

		if len(page.Items) == 0 || page.OverflowState == process.ProcessInstanceOverflowStateNoMore {
			return total, nil
		}
		pageReq = nextPISearchPageRequest(cmd, cfg, pageReq, page)
	}
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
		"process-instance total page: mode=%s, from=%d, after=%q, limit=%d, items=%d, total before=%d, total after=%d, overflow=%s, reported total=%d, reported kind=%s, end cursor=%q",
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
