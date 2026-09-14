// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"fmt"
	"math"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// SearchUserTasks collects all selected pages while preserving an initialized
// empty result for completed searches with no matches.
func SearchUserTasks(ctx context.Context, api API, query d.UserTaskSearchQuery, opts ...services.CallOption) ([]d.UserTask, error) {
	result, err := SearchUserTasksPages(ctx, api, query, nil, opts...)
	if err != nil {
		return nil, err
	}
	return result.Items, nil
}

// SearchUserTasksPages owns cursor and offset advancement, sparse-page
// continuation, caller-limit trimming, and successful completion semantics.
func SearchUserTasksPages(ctx context.Context, api API, query d.UserTaskSearchQuery, visitor d.UserTaskSearchPageVisitor, opts ...services.CallOption) (d.UserTaskSearchPagesResult, error) {
	_ = services.ApplyCallOptions(opts)
	normalizeUserTaskSearchQuery(&query)
	result := d.UserTaskSearchPagesResult{
		Items: make([]d.UserTask, 0, initialUserTaskSearchCapacity(query.BatchSize, query.Limit)),
	}

	stopped, err := walkUserTaskSearchPages(ctx, api, query, func(page d.UserTaskSearchPage, _ int64, pages int32) (bool, error) {
		page.Items = trimUserTaskSearchItems(page.Items, query.Limit, result.Total)
		result.Items = append(result.Items, page.Items...)
		result.Total += int64(len(page.Items))
		result.Pages = pages
		limitReached := query.Limit > 0 && result.Total >= int64(query.Limit)

		if visitor != nil {
			action, err := visitor(d.UserTaskSearchPageStep{
				Page:            page,
				CumulativeCount: result.Total,
				LimitReached:    limitReached,
			})
			if err != nil {
				return false, err
			}
			if err := action.Validate(); err != nil {
				return false, malformedUserTaskSearchMetadata("visitor action", err)
			}
			if action == d.UserTaskSearchPageActionStop {
				result.Completion = d.UserTaskSearchCompletionVisitorStopped
				return true, nil
			}
		}
		if limitReached {
			result.Completion = d.UserTaskSearchCompletionLimitReached
			return true, nil
		}
		return false, nil
	}, opts...)
	if err != nil {
		return d.UserTaskSearchPagesResult{}, err
	}
	if !stopped {
		result.Completion = d.UserTaskSearchCompletionExhausted
	}
	return result, nil
}

// SearchUserTasksTotal returns trustworthy exact metadata immediately or
// counts raw matching rows with int64 state and no accumulated task slice.
func SearchUserTasksTotal(ctx context.Context, api API, query d.UserTaskSearchQuery, opts ...services.CallOption) (int64, error) {
	_ = services.ApplyCallOptions(opts)
	query.Limit = 0
	normalizeUserTaskSearchQuery(&query)
	var total int64

	_, err := walkUserTaskSearchPages(ctx, api, query, func(page d.UserTaskSearchPage, rawTotal int64, _ int32) (bool, error) {
		if page.ReportedTotal != nil && page.ReportedTotal.Kind == d.UserTaskReportedTotalKindExact {
			total = page.ReportedTotal.Count
			return true, nil
		}
		total = rawTotal
		return false, nil
	}, opts...)
	if err != nil {
		return 0, err
	}
	return total, nil
}

// userTaskSearchPageObserver receives validated pages and may stop traversal
// without taking ownership of backend page advancement.
type userTaskSearchPageObserver func(d.UserTaskSearchPage, int64, int32) (bool, error)

// walkUserTaskSearchPages centralizes page validation, progress accounting,
// terminal probes, and cursor-cycle protection for collection and count modes.
func walkUserTaskSearchPages(ctx context.Context, api API, query d.UserTaskSearchQuery, observer userTaskSearchPageObserver, opts ...services.CallOption) (bool, error) {
	request := d.UserTaskPageRequest{Size: query.BatchSize}
	seenCursors := make(map[string]struct{})
	var rawTotal int64
	var offsetProgress int64
	var pages int32

	for {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		page, err := api.SearchUserTasksPage(ctx, query, request, opts...)
		if err != nil {
			return false, err
		}
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err := validateUserTaskSearchPage(page, request, rawTotal); err != nil {
			return false, err
		}
		if rawTotal > math.MaxInt64-int64(page.RawItemCount) {
			return false, malformedUserTaskSearchMetadata("raw count", fmt.Errorf("int64 overflow after %d matches", rawTotal))
		}
		rawTotal += int64(page.RawItemCount)
		if pages == math.MaxInt32 {
			return false, malformedUserTaskSearchMetadata("page count", fmt.Errorf("int32 overflow"))
		}
		pages++

		stopped, err := observer(page, rawTotal, pages)
		if err != nil {
			return false, err
		}
		if stopped {
			return true, nil
		}
		exhausted, err := userTaskSearchExhausted(page, rawTotal)
		if err != nil {
			return false, err
		}
		if exhausted {
			return false, nil
		}

		advance := int64(page.RawItemCount)
		if advance == 0 {
			advance = int64(request.Size)
		}
		if offsetProgress > math.MaxInt64-advance {
			return false, malformedUserTaskSearchMetadata("offset", fmt.Errorf("int64 overflow"))
		}
		offsetProgress += advance

		if page.EndCursor != "" {
			if page.EndCursor == request.After {
				return false, malformedUserTaskSearchMetadata("cursor", fmt.Errorf("cursor %q did not advance", page.EndCursor))
			}
			if _, exists := seenCursors[page.EndCursor]; exists {
				return false, malformedUserTaskSearchMetadata("cursor", fmt.Errorf("cursor %q forms a cycle", page.EndCursor))
			}
			seenCursors[page.EndCursor] = struct{}{}
			request = d.UserTaskPageRequest{Size: query.BatchSize, After: page.EndCursor}
			continue
		}
		if offsetProgress > math.MaxInt32 {
			return false, malformedUserTaskSearchMetadata("offset", fmt.Errorf("position %d exceeds int32", offsetProgress))
		}
		request = d.UserTaskPageRequest{From: int32(offsetProgress), Size: query.BatchSize}
	}
}

// validateUserTaskSearchPage rejects adapter facts that would make traversal
// skip rows, repeat requests, or claim an inconsistent exact completion.
func validateUserTaskSearchPage(page d.UserTaskSearchPage, request d.UserTaskPageRequest, rawBefore int64) error {
	if page.Request != request {
		return malformedUserTaskSearchMetadata("request", fmt.Errorf("got %+v, want %+v", page.Request, request))
	}
	if page.RawItemCount < 0 {
		return malformedUserTaskSearchMetadata("raw count", fmt.Errorf("cannot be negative"))
	}
	if int64(len(page.Items)) > int64(page.RawItemCount) {
		return malformedUserTaskSearchMetadata("raw count", fmt.Errorf("%d selected items exceed %d raw items", len(page.Items), page.RawItemCount))
	}
	if err := page.ContinuationState.Validate(); err != nil {
		return malformedUserTaskSearchMetadata("continuation", err)
	}
	if page.ReportedTotal == nil {
		return nil
	}
	if err := page.ReportedTotal.Validate(); err != nil {
		return malformedUserTaskSearchMetadata("reported total", err)
	}
	if rawBefore > math.MaxInt64-int64(page.RawItemCount) {
		return malformedUserTaskSearchMetadata("raw count", fmt.Errorf("int64 overflow after %d matches", rawBefore))
	}
	observed := rawBefore + int64(page.RawItemCount)
	if page.ReportedTotal.Kind == d.UserTaskReportedTotalKindExact && page.ReportedTotal.Count < observed {
		return malformedUserTaskSearchMetadata("reported total", fmt.Errorf("exact count %d is below %d observed matches", page.ReportedTotal.Count, observed))
	}
	if page.ContinuationState == d.UserTaskContinuationStateNoMore && page.ReportedTotal.Count > observed {
		return malformedUserTaskSearchMetadata("continuation", fmt.Errorf("no-more state leaves %d reported matches unobserved", page.ReportedTotal.Count-observed))
	}
	if page.ContinuationState == d.UserTaskContinuationStateHasMore && page.ReportedTotal.Kind == d.UserTaskReportedTotalKindExact && page.ReportedTotal.Count == observed {
		return malformedUserTaskSearchMetadata("continuation", fmt.Errorf("has-more state exceeds exact count %d", observed))
	}
	return nil
}

// userTaskSearchExhausted interprets normalized continuation together with
// observed totals without treating a capped lower bound as an exact count.
func userTaskSearchExhausted(page d.UserTaskSearchPage, rawTotal int64) (bool, error) {
	switch page.ContinuationState {
	case d.UserTaskContinuationStateNoMore:
		return true, nil
	case d.UserTaskContinuationStateHasMore:
		return false, nil
	case d.UserTaskContinuationStateIndeterminate:
		if page.ReportedTotal != nil {
			switch page.ReportedTotal.Kind {
			case d.UserTaskReportedTotalKindExact:
				return rawTotal == page.ReportedTotal.Count, nil
			case d.UserTaskReportedTotalKindLowerBound:
				if rawTotal < page.ReportedTotal.Count {
					return false, nil
				}
			}
		}
		return page.RawItemCount == 0 && page.EndCursor == "", nil
	default:
		return false, malformedUserTaskSearchMetadata("continuation", fmt.Errorf("unknown state %q", page.ContinuationState))
	}
}

// normalizeUserTaskSearchQuery applies the shared page-size default once so
// adapters and visitor metadata observe the same effective request.
func normalizeUserTaskSearchQuery(query *d.UserTaskSearchQuery) {
	if query.BatchSize <= 0 {
		query.BatchSize = consts.MaxPISearchSize
	}
}

// trimUserTaskSearchItems limits only selected objects; raw page facts remain
// unchanged for subsequent backend progress and exact-count decisions.
func trimUserTaskSearchItems(items []d.UserTask, limit int32, selected int64) []d.UserTask {
	if limit <= 0 {
		return items
	}
	remaining := int64(limit) - selected
	if remaining <= 0 {
		return items[:0]
	}
	if int64(len(items)) > remaining {
		return items[:remaining]
	}
	return items
}

// initialUserTaskSearchCapacity avoids oversized allocation for bounded
// searches while retaining useful capacity for the ordinary first page.
func initialUserTaskSearchCapacity(batchSize int32, limit int32) int {
	if limit > 0 && limit < batchSize {
		return int(limit)
	}
	return int(batchSize)
}

// malformedUserTaskSearchMetadata classifies backend traversal facts through
// the established malformed-response sentinel.
func malformedUserTaskSearchMetadata(field string, err error) error {
	return fmt.Errorf("%w: invalid user-task search %s: %v", d.ErrMalformedResponse, field, err)
}
