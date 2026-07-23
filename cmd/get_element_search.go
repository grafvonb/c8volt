// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/consts"
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
	pageReq := newElementSearchPageRequest(0, request.BatchSize, request.Limit, 0)
	collected := element.SearchResult{Items: []element.Element{}}
	incremental := shouldRenderElementSearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinueElementSearchPages(cmd)
	processedTotal := 0
	capturedNow := time.Now().UTC()
	printFoundAndReturn := func() (element.SearchResult, bool, error) {
		if incremental {
			if pickMode() == RenderModeOneLine {
				renderOutputLine(cmd, "found: %d", processedTotal)
			}
			return element.SearchResult{}, true, nil
		}
		collected.Total = int32(len(collected.Items))
		return collected, false, nil
	}

	for {
		page, err := cli.SearchElementsPage(cmd.Context(), request, pageReq, collectOptions()...)
		if err != nil {
			return element.SearchResult{}, false, err
		}
		items := limitElementItems(page.Items, processedTotal, request.Limit)
		if incremental {
			if err := renderElementSearchPage(cmd, items, capturedNow); err != nil {
				return element.SearchResult{}, false, err
			}
		} else {
			collected.Items = append(collected.Items, items...)
		}
		processedTotal += len(items)

		if isElementLimitReached(processedTotal, request.Limit) || page.OverflowState != element.OverflowStateHasMore {
			return printFoundAndReturn()
		}
		if autoContinue {
			pageReq = nextElementSearchPageRequest(pageReq, page, request.Limit, processedTotal)
			continue
		}
		if len(items) == 0 {
			pageReq = nextElementSearchPageRequest(pageReq, page, request.Limit, processedTotal)
			continue
		}
		prompt := fmt.Sprintf("Fetched %d element(s) on this page (%d loaded). More matching elements remain. Continue?", len(items), processedTotal)
		if err := confirmCmdOrAbortFn(shouldImplicitlyConfirm(cmd), prompt); err != nil {
			if isCmdAborted(err) {
				return printFoundAndReturn()
			}
			return element.SearchResult{}, false, err
		}
		pageReq = nextElementSearchPageRequest(pageReq, page, request.Limit, processedTotal)
	}
}

// searchElementsTotal counts matching elements, trusting exact backend totals
// when available and otherwise walking pages quietly.
func searchElementsTotal(cmd *cobra.Command, cli element.API, request element.SearchRequest) (int64, error) {
	pageReq := element.PageRequest{Size: request.BatchSize}
	total := int64(0)
	for {
		page, err := cli.SearchElementsPage(cmd.Context(), request, pageReq, collectOptions()...)
		if err != nil {
			return 0, err
		}
		if canUseElementExactReportedTotal(page) {
			return page.ReportedTotal.Count, nil
		}
		total += int64(len(page.Items))
		if len(page.Items) == 0 || page.OverflowState != element.OverflowStateHasMore {
			return total, nil
		}
		pageReq = nextElementSearchPageRequest(pageReq, page, 0, int(total))
	}
}

// canUseElementExactReportedTotal reports whether the backend total can be used without paging.
func canUseElementExactReportedTotal(page element.Page) bool {
	return page.ReportedTotal != nil && page.ReportedTotal.Kind == element.ReportedTotalKindExact
}

// shouldRenderElementSearchPageIncrementally keeps human and key-only output streaming by page.
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

// limitElementItems trims the current page to the remaining user-requested limit.
func limitElementItems(items []element.Element, cumulative int, limit int32) []element.Element {
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

// isElementLimitReached reports whether the cross-page limit has been satisfied.
func isElementLimitReached(cumulative int, limit int32) bool {
	return limit > 0 && cumulative >= int(limit)
}

// nextElementSearchPageRequest advances offset pagination using the actual page size.
func nextElementSearchPageRequest(current element.PageRequest, page element.Page, limit int32, loaded int) element.PageRequest {
	nextFrom := current.From + int32(len(page.Items))
	if len(page.Items) == 0 {
		nextFrom = current.From + current.Size
	}
	return newElementSearchPageRequest(nextFrom, page.Request.Size, limit, loaded)
}

// newElementSearchPageRequest computes an effective page size bounded by any remaining limit.
func newElementSearchPageRequest(from int32, batchSize int32, limit int32, loaded int) element.PageRequest {
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
	return element.PageRequest{From: from, Size: size}
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
