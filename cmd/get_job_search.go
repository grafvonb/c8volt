// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

func searchJobsWithPaging(cmd *cobra.Command, cli job.API, request job.SearchRequest) (job.SearchResult, bool, error) {
	pageReq := newJobSearchPageRequest(0, request.BatchSize, request.Limit, 0)
	collected := job.SearchResult{Items: []job.Job{}, Limit: request.Limit}
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

		continuation := jobSearchContinuationState(page, processedTotal, request.Limit, autoContinue)
		printSearchPageProgress(cmd, searchPageProgressSummary{
			PageSize:          page.Request.Size,
			CurrentPageCount:  len(items),
			CumulativeCount:   processedTotal,
			MoreMatches:       describeJobOverflowState(page.OverflowState),
			ContinuationState: continuation,
		})

		if continuation == processInstanceContinuationLimitReached || continuation == processInstanceContinuationCompleted {
			return printFoundAndReturn()
		}
		if continuation == processInstanceContinuationAutoContinue {
			pageReq = nextJobSearchPageRequest(pageReq, page, request.Limit, processedTotal)
			continue
		}
		if len(items) == 0 {
			pageReq = nextJobSearchPageRequest(pageReq, page, request.Limit, processedTotal)
			continue
		}
		prompt := fmt.Sprintf("Fetched %d job(s) on this page (%d loaded). More matching jobs remain. Continue?", len(items), processedTotal)
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			if isCmdAborted(err) {
				return printFoundAndReturn()
			}
			return job.SearchResult{}, false, err
		}
		pageReq = nextJobSearchPageRequest(pageReq, page, request.Limit, processedTotal)
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
		pageReq = nextJobSearchPageRequest(pageReq, page, 0, int(total))
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

// jobSearchContinuationState translates job overflow metadata into the next CLI paging action.
func jobSearchContinuationState(page job.Page, cumulative int, limit int32, autoContinue bool) processInstanceContinuationState {
	if isJobLimitReached(cumulative, limit) {
		return processInstanceContinuationLimitReached
	}
	if page.OverflowState == job.OverflowStateHasMore {
		if autoContinue {
			return processInstanceContinuationAutoContinue
		}
		return processInstanceContinuationPrompt
	}
	return processInstanceContinuationCompleted
}

// describeJobOverflowState maps job overflow metadata to verbose progress wording.
func describeJobOverflowState(state job.OverflowState) string {
	if state == job.OverflowStateHasMore {
		return "yes"
	}
	return "no"
}

func nextJobSearchPageRequest(current job.PageRequest, page job.Page, limit int32, loaded int) job.PageRequest {
	nextFrom := current.From + int32(len(page.Items))
	if len(page.Items) == 0 {
		nextFrom = current.From + current.Size
	}
	return newJobSearchPageRequest(nextFrom, page.Request.Size, limit, loaded)
}

func newJobSearchPageRequest(from int32, batchSize int32, limit int32, loaded int) job.PageRequest {
	size := batchSize
	if size <= 0 {
		size = consts.MaxPISearchSize
	}
	if limit > 0 {
		remaining := limit - int32(loaded)
		if remaining < size {
			size = remaining
		}
	}
	return job.PageRequest{From: from, Size: size}
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
