// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"math"
	"strings"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// walkProcessInstanceSearchPages owns continuation for collection and counted totals.
// A visitor can intentionally stop indeterminate pages, preserving ordinary query prompt policy.
func walkProcessInstanceSearchPages(ctx context.Context, api API, request d.ProcessInstanceSearchRequest, exactTotalAllowed bool, observe func(d.ProcessInstancePage, int32) (d.ProcessInstanceSearchCompletion, error), opts ...services.CallOption) (d.ProcessInstanceSearchCompletion, error) {
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
			return d.ProcessInstanceSearchCompletionIncomplete, malformedProcessInstanceSearchMetadata("request")
		}
		// Count-only callers can use compatible exact metadata without proving row traversal.
		if exactTotalAllowed && page.ReportedTotal != nil && page.ReportedTotal.Kind == d.ProcessInstanceReportedTotalKindExact {
			if page.ReportedTotal.Count < 0 {
				return d.ProcessInstanceSearchCompletionIncomplete, malformedProcessInstanceSearchMetadata("negative reported total")
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
		if err := validateProcessInstanceSearchPage(page); err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		for _, item := range page.Items {
			rawKeys[item.Key] = struct{}{}
		}
		exhausted, err := processInstanceSearchExhausted(page, int64(request.Page.From)+int64(len(rawKeys)))
		if err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
		if !exhausted {
			if page.EndCursor != "" {
				if _, exists := seen[page.EndCursor]; exists {
					return d.ProcessInstanceSearchCompletionIncomplete, malformedProcessInstanceSearchMetadata("cursor did not advance or forms a cycle")
				}
			} else if len(page.Items) > 0 {
				signature := processInstanceSearchPageSignature(page.Items)
				if _, exists := offsetPages[signature]; exists {
					return d.ProcessInstanceSearchCompletionIncomplete, malformedProcessInstanceSearchMetadata("offset page did not advance")
				}
				offsetPages[signature] = struct{}{}
			}
		}
		if pages == math.MaxInt32 {
			return d.ProcessInstanceSearchCompletionIncomplete, malformedProcessInstanceSearchMetadata("page count overflow")
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
		req, err = advanceProcessInstanceSearchPage(req, page.EndCursor, rawCount, seen)
		if err != nil {
			return d.ProcessInstanceSearchCompletionIncomplete, err
		}
	}
}

// validateProcessInstanceSearchPage rejects unknown and contradictory facts without exposing cursors in errors.
func validateProcessInstanceSearchPage(page d.ProcessInstancePage) error {
	switch page.OverflowState {
	case d.ProcessInstanceOverflowStateNoMore, d.ProcessInstanceOverflowStateHasMore, d.ProcessInstanceOverflowStateIndeterminate:
	default:
		return malformedProcessInstanceSearchMetadata("continuation state")
	}
	if page.ReportedTotal == nil {
		return nil
	}
	total := page.ReportedTotal
	if total.Count < 0 {
		return malformedProcessInstanceSearchMetadata("negative reported total")
	}
	switch total.Kind {
	case d.ProcessInstanceReportedTotalKindExact, d.ProcessInstanceReportedTotalKindLowerBound:
	default:
		return malformedProcessInstanceSearchMetadata("reported total kind")
	}
	position := int64(page.Request.From) + int64(len(page.Items))
	if page.OverflowState == d.ProcessInstanceOverflowStateNoMore && total.Count > position {
		return malformedProcessInstanceSearchMetadata("terminal page before reported total")
	}
	if page.OverflowState == d.ProcessInstanceOverflowStateHasMore && total.Kind == d.ProcessInstanceReportedTotalKindExact && total.Count <= position {
		return malformedProcessInstanceSearchMetadata("continuation beyond exact total")
	}
	return nil
}

// processInstanceSearchExhausted resolves indeterminate metadata or permits an offset/cursor terminal probe.
func processInstanceSearchExhausted(page d.ProcessInstancePage, observed int64) (bool, error) {
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
					return false, malformedProcessInstanceSearchMetadata("unresolved empty page before reported total")
				}
				return false, nil
			}
			if page.ReportedTotal.Kind == d.ProcessInstanceReportedTotalKindExact {
				return true, nil
			}
		}
		return len(page.Items) == 0 && page.EndCursor == "", nil
	default:
		return false, malformedProcessInstanceSearchMetadata("continuation state")
	}
}

// advanceProcessInstanceSearchPage guards cursor cycles and int32 offsets, including sparse raw pages.
func advanceProcessInstanceSearchPage(current d.ProcessInstancePageRequest, cursor string, rawCount int, seen map[string]struct{}) (d.ProcessInstancePageRequest, error) {
	advance := int64(rawCount)
	if advance == 0 {
		advance = int64(current.Size)
	}
	position := int64(current.From) + advance
	if advance <= 0 || position > math.MaxInt32 {
		return current, malformedProcessInstanceSearchMetadata("offset overflow or no progress")
	}
	if cursor != "" {
		if _, exists := seen[cursor]; exists {
			return current, malformedProcessInstanceSearchMetadata("cursor did not advance or forms a cycle")
		}
		seen[cursor] = struct{}{}
	}
	return d.ProcessInstancePageRequest{From: int32(position), Size: current.Size, After: cursor}, nil
}

// distinctProcessInstanceSearchItems keeps the first matching observation in backend order.
func distinctProcessInstanceSearchItems(items []d.ProcessInstance, seen map[string]struct{}) []d.ProcessInstance {
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

// malformedProcessInstanceSearchMetadata preserves the existing failure class with bounded, cursor-free context.
func malformedProcessInstanceSearchMetadata(reason string) error {
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
