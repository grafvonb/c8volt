// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

func searchElementsWithPaging(cmd *cobra.Command, cli element.API, request element.SearchRequest) (element.SearchResult, bool, error) {
	pageReq := newElementSearchPageRequest(0, request.BatchSize, request.Limit, 0)
	collected := element.SearchResult{Items: []element.Element{}}
	incremental := shouldRenderElementSearchPageIncrementally(cmd)
	autoContinue := shouldAutoContinueElementSearchPages(cmd)
	processedTotal := 0
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
			if err := renderElementSearchPage(cmd, items); err != nil {
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

func canUseElementExactReportedTotal(page element.Page) bool {
	return page.ReportedTotal != nil && page.ReportedTotal.Kind == element.ReportedTotalKindExact
}

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

func shouldAutoContinueElementSearchPages(cmd *cobra.Command) bool {
	return shouldImplicitlyConfirm(cmd) || pickMode() == RenderModeJSON
}

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

func isElementLimitReached(cumulative int, limit int32) bool {
	return limit > 0 && cumulative >= int(limit)
}

func nextElementSearchPageRequest(current element.PageRequest, page element.Page, limit int32, loaded int) element.PageRequest {
	nextFrom := current.From + int32(len(page.Items))
	if len(page.Items) == 0 {
		nextFrom = current.From + current.Size
	}
	return newElementSearchPageRequest(nextFrom, page.Request.Size, limit, loaded)
}

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

func renderElementSearchPage(cmd *cobra.Command, items []element.Element) error {
	switch pickMode() {
	case RenderModeKeysOnly:
		for _, item := range items {
			renderOutputLine(cmd, "%s", item.ElementInstanceKey)
		}
	default:
		rows := make([]flatRow, 0, len(items))
		for _, item := range items {
			rows = append(rows, flatRowElement(item))
		}
		for _, line := range formatFlatRows(rows) {
			renderOutputLine(cmd, "%s", line)
		}
	}
	return nil
}
