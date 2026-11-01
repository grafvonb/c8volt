// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package searchpaging

import (
	"context"
	"fmt"
	"math"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// API is the single-page adapter boundary shared by searches and child traversal.
type API interface {
	SearchForProcessInstancesPage(context.Context, d.ProcessInstanceFilter, d.ProcessInstancePageRequest, ...services.CallOption) (d.ProcessInstancePage, error)
}

// Walk owns continuation for collection and counted totals.
// A visitor can intentionally stop indeterminate pages, preserving ordinary query prompt policy.
func Walk(ctx context.Context, api API, request d.ProcessInstanceSearchRequest, exactTotalAllowed bool, observe func(d.ProcessInstancePage, int32) (d.ProcessInstanceSearchCompletion, error), opts ...services.CallOption) (d.ProcessInstanceSearchCompletion, error) {
	req := request.Page
	if req.From < 0 {
		return d.ProcessInstanceSearchCompletionIncomplete, fmt.Errorf("%w: search offset cannot be negative", d.ErrValidation)
	}
	seen := make(map[string]struct{})
	if req.After != "" {
		seen[req.After] = struct{}{}
	}
	offsetPages := make(map[string]struct{})
	rawKeys := make(map[string]struct{})
	var pages int32
	for {
		if err := ctx.Err(); err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		page, err := api.SearchForProcessInstancesPage(ctx, request.Filter, req, opts...)
		if err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		if err := ctx.Err(); err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		if page.Request != req {
			return d.ProcessInstanceSearchCompletionIncomplete, MalformedMetadata("request")
		}
		// Count-only callers can use compatible exact metadata without proving row traversal.
		if exactTotalAllowed && page.ReportedTotal != nil && page.ReportedTotal.Kind == d.ProcessInstanceReportedTotalKindExact {
			if page.ReportedTotal.Count < 0 {
				return d.ProcessInstanceSearchCompletionIncomplete, MalformedMetadata("negative reported total")
			}
			completion, err := observe(page, pages+1)
			if err != nil {
				return d.ProcessInstanceSearchCompletionIncomplete, err
			}
			if err := ctx.Err(); err != nil {
				return d.ProcessInstanceSearchCompletionIncomplete, err
			}
			return completion, nil
		}
		if err := ValidatePage(page); err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		for _, item := range page.Items {
			rawKeys[item.Key] = struct{}{}
		}
		exhausted, err := Exhausted(page, int64(request.Page.From)+int64(len(rawKeys)))
		if err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		if !exhausted {
			if page.EndCursor != "" {
				if _, exists := seen[page.EndCursor]; exists {
					return d.ProcessInstanceSearchCompletionIncomplete, MalformedMetadata("cursor did not advance or forms a cycle")
				}
			} else if len(page.Items) > 0 {
				signature := processInstanceSearchPageSignature(page.Items)
				if _, exists := offsetPages[signature]; exists {
					return d.ProcessInstanceSearchCompletionIncomplete, MalformedMetadata("offset page did not advance")
				}
				offsetPages[signature] = struct{}{}
			}
		}
		if pages == math.MaxInt32 {
			return d.ProcessInstanceSearchCompletionIncomplete, MalformedMetadata("page count overflow")
		}
		pages++
		// Keep raw advancement separate from deduplicated or locally filtered selections.
		rawCount := len(page.Items)
		completion, err := observe(page, pages)
		if err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		if err := ctx.Err(); err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		if completion != "" {
			if exhausted && completion == d.ProcessInstanceSearchCompletionVisitorStopped {
				return d.ProcessInstanceSearchCompletionExhausted, nil
			}
			return completion, nil
		}
		if exhausted {
			return d.ProcessInstanceSearchCompletionExhausted, nil
		}
		req, err = Advance(req, page.EndCursor, rawCount, seen)
		if err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
	}
}

// ValidatePage rejects unknown and contradictory facts without exposing cursors in errors.
func ValidatePage(page d.ProcessInstancePage) error {
	switch page.OverflowState {
	case d.ProcessInstanceOverflowStateNoMore, d.ProcessInstanceOverflowStateHasMore, d.ProcessInstanceOverflowStateIndeterminate:
	default:
		return MalformedMetadata("continuation state")
	}
	if page.ReportedTotal == nil {
		return nil
	}
	total := page.ReportedTotal
	if total.Count < 0 {
		return MalformedMetadata("negative reported total")
	}
	switch total.Kind {
	case d.ProcessInstanceReportedTotalKindExact, d.ProcessInstanceReportedTotalKindLowerBound:
	default:
		return MalformedMetadata("reported total kind")
	}
	position := int64(page.Request.From) + int64(len(page.Items))
	if page.OverflowState == d.ProcessInstanceOverflowStateNoMore && total.Count > position {
		return MalformedMetadata("terminal page before reported total")
	}
	if page.OverflowState == d.ProcessInstanceOverflowStateHasMore && total.Kind == d.ProcessInstanceReportedTotalKindExact && total.Count <= position {
		return MalformedMetadata("continuation beyond exact total")
	}
	return nil
}

// Exhausted resolves indeterminate metadata or permits an offset/cursor terminal probe.
func Exhausted(page d.ProcessInstancePage, observed int64) (bool, error) {
	switch page.OverflowState {
	case d.ProcessInstanceOverflowStateNoMore:
		return true, nil
	case d.ProcessInstanceOverflowStateHasMore:
		return false, nil
	case d.ProcessInstanceOverflowStateIndeterminate:
		if page.ReportedTotal != nil {
			if observed < page.ReportedTotal.Count {
				// An empty ambiguous page without a cursor cannot justify skipping unseen matches.
				if len(page.Items) == 0 && page.EndCursor == "" {
					return false, MalformedMetadata("unresolved empty page before reported total")
				}
				return false, nil
			}
			if page.ReportedTotal.Kind == d.ProcessInstanceReportedTotalKindExact {
				return true, nil
			}
		}
		return len(page.Items) == 0 && page.EndCursor == "", nil
	default:
		return false, MalformedMetadata("continuation state")
	}
}

// Advance guards cursor cycles and int32 offsets, including sparse raw pages.
func Advance(current d.ProcessInstancePageRequest, cursor string, rawCount int, seen map[string]struct{}) (d.ProcessInstancePageRequest, error) {
	advance := int64(rawCount)
	if advance == 0 {
		advance = int64(current.Size)
	}
	position := int64(current.From) + advance
	if advance <= 0 || position > math.MaxInt32 {
		return current, MalformedMetadata("offset overflow or no progress")
	}
	if cursor != "" {
		if _, exists := seen[cursor]; exists {
			return current, MalformedMetadata("cursor did not advance or forms a cycle")
		}
		seen[cursor] = struct{}{}
	}
	return d.ProcessInstancePageRequest{From: int32(position), Size: current.Size, After: cursor}, nil
}

// DistinctItems keeps the first matching observation in backend order.
func DistinctItems(items []d.ProcessInstance, seen map[string]struct{}) []d.ProcessInstance {
	out := make([]d.ProcessInstance, 0, len(items))
	for _, item := range items {
		if _, exists := seen[item.Key]; exists {
			continue
		}
		seen[item.Key] = struct{}{}
		out = append(out, item)
	}
	return out
}

// MalformedMetadata preserves the existing failure class with bounded, cursor-free context.
func MalformedMetadata(reason string) error {
	return fmt.Errorf("%w: invalid process-instance search metadata: %s", d.ErrMalformedResponse, reason)
}

// processInstanceSearchPageSignature detects repeated offset pages without retaining another copy of resource objects.
func processInstanceSearchPageSignature(items []d.ProcessInstance) string {
	var out strings.Builder
	for _, item := range items {
		fmt.Fprintf(&out, "%d:%s;", len(item.Key), item.Key)
	}
	return out.String()
}
