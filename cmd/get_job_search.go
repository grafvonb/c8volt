// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/spf13/cobra"
)

func searchJobsWithPaging(cmd *cobra.Command, cli job.API, request job.SearchRequest) (job.SearchResult, bool, error) {
	pageReq := job.PageRequest{Size: request.BatchSize}
	var collected job.SearchResult
	collected.Limit = request.Limit
	incremental := shouldRenderJobSearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinueJobSearchPages(cmd)
	processedTotal := 0
	printFoundAndReturn := func() (job.SearchResult, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine {
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return job.SearchResult{}, true, nil
		}
		return collected, false, nil
	}

	for {
		page, err := cli.SearchJobsPage(cmd.Context(), request, pageReq, collectOptions()...)
		if err != nil {
			return job.SearchResult{}, false, err
		}
		items := limitJobItems(page.Items, processedTotal, request.Limit)
		if incremental {
			if err := renderJobSearchPage(cmd, items); err != nil {
				return job.SearchResult{}, false, err
			}
		} else {
			collected.Items = append(collected.Items, items...)
		}
		processedTotal += len(items)

		if isJobLimitReached(processedTotal, request.Limit) || page.OverflowState != job.OverflowStateHasMore {
			return printFoundAndReturn()
		}
		if autoContinue {
			pageReq = nextJobSearchPageRequest(pageReq, page)
			continue
		}
		if len(items) == 0 {
			pageReq = nextJobSearchPageRequest(pageReq, page)
			continue
		}
		prompt := fmt.Sprintf("Fetched %d job(s) on this page (%d loaded). More matching jobs remain. Continue?", len(items), processedTotal)
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			if isCmdAborted(err) {
				return printFoundAndReturn()
			}
			return job.SearchResult{}, false, err
		}
		pageReq = nextJobSearchPageRequest(pageReq, page)
	}
}

func searchJobsTotal(cmd *cobra.Command, cli job.API, request job.SearchRequest) (int64, error) {
	pageReq := job.PageRequest{Size: request.BatchSize}
	total := int64(0)
	for {
		page, err := cli.SearchJobsPage(cmd.Context(), request, pageReq, collectOptions()...)
		if err != nil {
			return 0, err
		}
		if canUseJobExactReportedTotal(page) {
			return page.ReportedTotal.Count, nil
		}
		total += int64(len(page.Items))
		if len(page.Items) == 0 || page.OverflowState != job.OverflowStateHasMore {
			return total, nil
		}
		pageReq = nextJobSearchPageRequest(pageReq, page)
	}
}

func canUseJobExactReportedTotal(page job.Page) bool {
	return page.ReportedTotal != nil && page.ReportedTotal.Kind == job.ReportedTotalKindExact
}

func shouldRenderJobSearchPageIncrementally(cmd *cobra.Command) bool {
	if flagCmdAutoConfirm {
		return false
	}
	mode := pickMode()
	if automationModeEnabled(cmd) {
		return mode == RenderModeOneLine || mode == RenderModeKeysOnly
	}
	return mode == RenderModeOneLine || mode == RenderModeKeysOnly
}

func shouldAutoContinueJobSearchPages(cmd *cobra.Command) bool {
	return shouldImplicitlyConfirm(cmd) || pickMode() == RenderModeJSON
}

func limitJobItems(items []job.Job, cumulative int, limit int32) []job.Job {
	if limit <= 0 {
		return items
	}
	remaining := int(limit) - cumulative
	if remaining <= 0 {
		return nil
	}
	if len(items) > remaining {
		return items[:remaining]
	}
	return items
}

func isJobLimitReached(cumulative int, limit int32) bool {
	return limit > 0 && cumulative >= int(limit)
}

func nextJobSearchPageRequest(current job.PageRequest, page job.Page) job.PageRequest {
	size := current.Size
	if size <= 0 {
		size = page.Request.Size
	}
	return job.PageRequest{From: current.From + int32(len(page.Items)), Size: size}
}

func renderJobSearchPage(cmd *cobra.Command, items []job.Job) error {
	switch pickMode() {
	case RenderModeKeysOnly:
		for _, item := range items {
			renderOutputLine(cmd, "%s", item.Key)
		}
	default:
		rows := make([]flatRow, 0, len(items))
		for _, item := range items {
			rows = append(rows, flatRowJobWithTimezoneForMode(cmd)(item))
		}
		for _, line := range formatFlatRows(rows) {
			renderOutputLine(cmd, "%s", line)
		}
	}
	return nil
}
