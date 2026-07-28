// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

func searchJobsWithPaging(cmd *cobra.Command, cli job.API, request job.SearchRequest) (job.SearchResult, bool, error) {
	incremental := shouldRenderJobSearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinueJobSearchPages(cmd)
	processedTotal := 0
	pageNumber := 0
	printFoundAndReturn := func() (job.SearchResult, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine {
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return job.SearchResult{}, true, nil
		}
		return job.SearchResult{}, false, nil
	}

	result, err := cli.SearchJobsPages(cmd.Context(), request, func(step job.SearchPageStep) (job.SearchPageAction, error) {
		page := step.Page
		items := page.Items
		pageNumber++
		processedTotal = int(step.CumulativeCount)
		total, totalKind := jobOpsReportedTotal(page)
		printBasicSearchOpsProgress(cmd, basicSearchProgressMetadata{
			Command:            "get job",
			CoreResource:       "job",
			ResourceLabel:      "jobs",
			SelectorSummary:    "job search",
			ConsequenceSummary: "job search will discover and render matching jobs",
			ReportedTotal:      total,
			TotalKind:          totalKind,
			OverflowState:      jobOpsOverflowState(page.OverflowState),
			PageSize:           page.Request.Size,
			CurrentPage:        pageNumber,
			CurrentPageCount:   len(items),
			CumulativeCount:    processedTotal,
			LimitReached:       step.LimitReached,
		}, pageNumber == 1)
		if incremental {
			if err := renderJobSearchPage(cmd, items); err != nil {
				return job.SearchPageActionStop, err
			}
		}

		continuation := jobSearchContinuationState(page, step.LimitReached, autoContinue)

		if continuation == processInstanceContinuationLimitReached || continuation == processInstanceContinuationCompleted {
			return job.SearchPageActionStop, nil
		}
		if continuation == processInstanceContinuationAutoContinue {
			return job.SearchPageActionContinue, nil
		}
		if len(items) == 0 {
			return job.SearchPageActionContinue, nil
		}
		prompt := fmt.Sprintf("Fetched %d job(s) on this page (%d loaded). More matching jobs remain. Continue?", len(items), processedTotal)
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			if isCmdAborted(err) {
				return job.SearchPageActionStop, nil
			}
			return job.SearchPageActionStop, err
		}
		return job.SearchPageActionContinue, nil
	}, collectOptions()...)
	if err != nil {
		return job.SearchResult{}, false, err
	}
	if incremental {
		return printFoundAndReturn()
	}
	return job.SearchResult{Items: result.Items, Limit: result.Limit}, false, nil
}

func searchJobsTotal(cmd *cobra.Command, cli job.API, request job.SearchRequest) (int64, error) {
	return cli.SearchJobsTotal(cmd.Context(), request, collectOptions()...)
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

// jobSearchContinuationState translates job overflow metadata into the next CLI paging action.
func jobSearchContinuationState(page job.Page, limitReached bool, autoContinue bool) processInstanceContinuationState {
	if limitReached {
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

func jobOpsReportedTotal(page job.Page) (*int64, ops.TotalCertainty) {
	if page.ReportedTotal == nil {
		return nil, ops.TotalCertaintyUnknown
	}
	total := page.ReportedTotal.Count
	switch page.ReportedTotal.Kind {
	case job.ReportedTotalKindExact:
		return &total, ops.TotalCertaintyExact
	case job.ReportedTotalKindLowerBound:
		return &total, ops.TotalCertaintyLowerBound
	default:
		return &total, ops.TotalCertaintyUnknown
	}
}

func jobOpsOverflowState(state job.OverflowState) ops.OverflowState {
	switch state {
	case job.OverflowStateHasMore:
		return ops.OverflowStateHasMore
	case job.OverflowStateNoMore:
		return ops.OverflowStateNoMore
	default:
		return ops.OverflowStateUnknown
	}
}
