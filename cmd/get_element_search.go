// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// searchElementsForCommand selects collected listener search or the existing paging renderer.
func searchElementsForCommand(cmd *cobra.Command, cli element.API, request element.SearchRequest) (element.SearchResult, bool, error) {
	if flagGetElementWithListeners {
		result, err := cli.SearchElementsWithListeners(cmd.Context(), request, collectOptions()...)
		return result, false, err
	}
	return searchElementsWithPaging(cmd, cli, request)
}

// searchElementsWithPaging walks element search pages and either streams rows
// incrementally or returns one bounded collection for JSON rendering.
func searchElementsWithPaging(cmd *cobra.Command, cli element.API, request element.SearchRequest) (element.SearchResult, bool, error) {
	incremental := shouldRenderElementSearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinueElementSearchPages(cmd)
	processedTotal := 0
	pageNumber := 0
	capturedNow := time.Now().UTC()
	printFoundAndReturn := func() (element.SearchResult, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine {
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return element.SearchResult{}, true, nil
		}
		return element.SearchResult{}, false, nil
	}

	result, err := cli.SearchElementsPages(cmd.Context(), request, func(step element.SearchPageStep) (element.SearchPageAction, error) {
		page := step.Page
		pageNumber++
		processedTotal = int(step.CumulativeCount)
		total, totalKind := elementOpsReportedTotal(page)
		printBasicSearchOpsProgress(cmd, basicSearchProgressMetadata{
			Command:            "get element",
			CoreResource:       "element",
			ResourceLabel:      "elements",
			SelectorSummary:    "element search",
			ConsequenceSummary: "element search will discover and render matching elements",
			ReportedTotal:      total,
			TotalKind:          totalKind,
			OverflowState:      elementOpsOverflowState(page.OverflowState),
			PageSize:           page.Request.Size,
			CurrentPage:        pageNumber,
			CurrentPageCount:   len(page.Items),
			CumulativeCount:    processedTotal,
			LimitReached:       step.LimitReached,
		}, pageNumber == 1)
		if incremental {
			if err := renderElementSearchPage(cmd, page.Items, capturedNow); err != nil {
				return element.SearchPageActionStop, err
			}
		}

		continuation := elementSearchContinuationState(page, step.LimitReached, autoContinue)

		if continuation == processInstanceContinuationLimitReached || continuation == processInstanceContinuationCompleted {
			return element.SearchPageActionStop, nil
		}
		if continuation == processInstanceContinuationAutoContinue {
			return element.SearchPageActionContinue, nil
		}
		if len(page.Items) == 0 {
			return element.SearchPageActionContinue, nil
		}
		prompt := fmt.Sprintf("Fetched %d element(s) on this page (%d loaded). More matching elements remain. Continue?", len(page.Items), processedTotal)
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			if isCmdAborted(err) {
				return element.SearchPageActionStop, nil
			}
			return element.SearchPageActionStop, err
		}
		return element.SearchPageActionContinue, nil
	}, collectOptions()...)
	if err != nil {
		return element.SearchResult{}, false, err
	}
	if incremental {
		return printFoundAndReturn()
	}
	return element.SearchResult{Items: result.Items, Total: result.Total}, false, nil
}

// searchElementsTotal counts matching elements, trusting exact backend totals
// when available and otherwise walking pages quietly.
func searchElementsTotal(cmd *cobra.Command, cli element.API, request element.SearchRequest) (int64, error) {
	return cli.SearchElementsTotal(cmd.Context(), request, collectOptions()...)
}

// shouldRenderElementSearchPageIncrementally keeps human and keys-only output streaming by page.
func shouldRenderElementSearchPageIncrementally(cmd *cobra.Command) bool {
	if flagCmdAutoConfirm {
		return false
	}
	mode := pickMode()
	if automationModeEnabled(cmd) {
		return mode == RenderModeOneLine || mode == RenderModeKeysOnly
	}
	return mode == RenderModeOneLine || mode == RenderModeKeysOnly
}

// shouldAutoContinueElementSearchPages reports whether paging may proceed without a prompt.
func shouldAutoContinueElementSearchPages(cmd *cobra.Command) bool {
	return shouldImplicitlyConfirm(cmd) || pickMode() == RenderModeJSON
}

// elementSearchContinuationState translates element overflow metadata into the next CLI paging action.
func elementSearchContinuationState(page element.Page, limitReached bool, autoContinue bool) processInstanceContinuationState {
	if limitReached {
		return processInstanceContinuationLimitReached
	}
	if page.OverflowState == element.OverflowStateHasMore {
		if autoContinue {
			return processInstanceContinuationAutoContinue
		}
		return processInstanceContinuationPrompt
	}
	return processInstanceContinuationCompleted
}

// describeElementOverflowState maps element overflow metadata to verbose progress wording.
func describeElementOverflowState(state element.OverflowState) string {
	if state == element.OverflowStateHasMore {
		return "yes"
	}
	return "no"
}

// renderElementSearchPage renders one page in the current incremental output mode.
func renderElementSearchPage(cmd *cobra.Command, items []element.Element, capturedNow time.Time) error {
	switch pickMode() {
	case RenderModeKeysOnly:
		for _, item := range items {
			renderOutputLine(cmd, "%s", item.ElementInstanceKey)
		}
	default:
		rowOf := flatRowElementWithTimezoneForMode(cmd, capturedNow)
		rows := make([]flatRow, 0, len(items))
		for _, item := range items {
			rows = append(rows, rowOf(item))
		}
		for _, line := range formatFlatRows(rows) {
			renderOutputLine(cmd, "%s", line)
		}
	}
	return nil
}

func elementOpsReportedTotal(page element.Page) (*int64, ops.TotalCertainty) {
	if page.ReportedTotal == nil {
		return nil, ops.TotalCertaintyUnknown
	}
	total := page.ReportedTotal.Count
	switch page.ReportedTotal.Kind {
	case element.ReportedTotalKindExact:
		return &total, ops.TotalCertaintyExact
	case element.ReportedTotalKindLowerBound:
		return &total, ops.TotalCertaintyLowerBound
	default:
		return &total, ops.TotalCertaintyUnknown
	}
}

func elementOpsOverflowState(state element.OverflowState) ops.OverflowState {
	switch state {
	case element.OverflowStateHasMore:
		return ops.OverflowStateHasMore
	case element.OverflowStateNoMore:
		return ops.OverflowStateNoMore
	default:
		return ops.OverflowStateUnknown
	}
}
