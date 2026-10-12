// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

// Package search provides version-neutral complete root-variable traversal.
package search

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// PageReader is the narrow adapter contract needed for complete traversal.
type PageReader interface {
	SearchProcessInstanceVariablesPage(context.Context, string, d.ProcessInstanceVariablePageRequest, ...services.CallOption) (d.ProcessInstanceVariablePage, error)
}

// ProcessInstanceVariables reads every authoritative page and returns a stable
// name-ordered collection. An error never returns partial results.
func ProcessInstanceVariables(ctx context.Context, reader PageReader, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	request := d.ProcessInstanceVariablePageRequest{Size: consts.MaxPISearchSize}
	items := make([]d.ProcessInstanceVariable, 0)
	seenCursors := make(map[string]struct{})
	var rawObserved int64
	var retainedLowerBound int64
	var exactTotal *int64
	var totalPresence *bool

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := reader.SearchProcessInstanceVariablesPage(ctx, key, request, opts...)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := validatePage(page, request, rawObserved, retainedLowerBound, exactTotal, totalPresence); err != nil {
			return nil, err
		}
		if totalPresence == nil {
			present := page.ReportedTotal != nil
			totalPresence = &present
		}

		rawObserved += int64(page.RawItemCount)
		items = append(items, page.Items...)
		if page.ReportedTotal != nil {
			switch page.ReportedTotal.Kind {
			case d.ProcessInstanceVariableReportedTotalKindExact:
				if exactTotal == nil {
					value := page.ReportedTotal.Count
					exactTotal = &value
				}
				if rawObserved == page.ReportedTotal.Count {
					return normalize(items), nil
				}
			case d.ProcessInstanceVariableReportedTotalKindLowerBound:
				retainedLowerBound = page.ReportedTotal.Count
			}
		}

		if page.EndCursor != "" {
			if page.EndCursor == request.After {
				return nil, malformed("cursor", fmt.Errorf("cursor %q did not advance", page.EndCursor))
			}
			if _, exists := seenCursors[page.EndCursor]; exists {
				return nil, malformed("cursor", fmt.Errorf("cursor %q repeated", page.EndCursor))
			}
			seenCursors[page.EndCursor] = struct{}{}
			request = d.ProcessInstanceVariablePageRequest{Size: request.Size, After: page.EndCursor}
			continue
		}

		if page.RawItemCount == 0 {
			if page.ReportedTotal == nil || (page.ReportedTotal.Kind == d.ProcessInstanceVariableReportedTotalKindLowerBound && rawObserved >= retainedLowerBound) {
				return normalize(items), nil
			}
			return nil, malformed("continuation", fmt.Errorf("empty page before %d reported variables were observed", page.ReportedTotal.Count))
		}
		next := int64(request.From) + int64(page.RawItemCount)
		if request.After != "" {
			next = rawObserved
		}
		if next > math.MaxInt32 {
			return nil, malformed("offset", fmt.Errorf("position %d exceeds int32", next))
		}
		request = d.ProcessInstanceVariablePageRequest{From: int32(next), Size: request.Size}
	}
}

func validatePage(page d.ProcessInstanceVariablePage, request d.ProcessInstanceVariablePageRequest, rawBefore, retainedLowerBound int64, exactTotal *int64, totalPresence *bool) error {
	if page.Request != request {
		return malformed("request", fmt.Errorf("got %+v, want %+v", page.Request, request))
	}
	if page.RawItemCount < 0 {
		return malformed("raw count", fmt.Errorf("cannot be negative"))
	}
	if int64(len(page.Items)) > int64(page.RawItemCount) {
		return malformed("raw count", fmt.Errorf("%d retained items exceed %d raw items", len(page.Items), page.RawItemCount))
	}
	if rawBefore > math.MaxInt64-int64(page.RawItemCount) {
		return malformed("raw count", fmt.Errorf("int64 overflow after %d variables", rawBefore))
	}
	if page.ReportedTotal == nil {
		if totalPresence != nil && *totalPresence {
			return malformed("reported total", fmt.Errorf("metadata disappeared during traversal"))
		}
		return nil
	}
	if totalPresence != nil && !*totalPresence {
		return malformed("reported total", fmt.Errorf("metadata appeared during traversal"))
	}
	if err := page.ReportedTotal.Validate(); err != nil {
		return malformed("reported total", err)
	}
	observed := rawBefore + int64(page.RawItemCount)
	switch page.ReportedTotal.Kind {
	case d.ProcessInstanceVariableReportedTotalKindExact:
		if page.ReportedTotal.Count < observed {
			return malformed("reported total", fmt.Errorf("exact count %d is below %d observed variables", page.ReportedTotal.Count, observed))
		}
		if page.ReportedTotal.Count < retainedLowerBound {
			return malformed("reported total", fmt.Errorf("exact count %d is below retained lower bound %d", page.ReportedTotal.Count, retainedLowerBound))
		}
		if exactTotal != nil && page.ReportedTotal.Count != *exactTotal {
			return malformed("reported total", fmt.Errorf("exact count changed from %d to %d", *exactTotal, page.ReportedTotal.Count))
		}
	case d.ProcessInstanceVariableReportedTotalKindLowerBound:
		if exactTotal != nil {
			return malformed("reported total", fmt.Errorf("exact total changed to a lower bound"))
		}
		if page.ReportedTotal.Count < retainedLowerBound {
			return malformed("reported total", fmt.Errorf("lower bound decreased from %d to %d", retainedLowerBound, page.ReportedTotal.Count))
		}
	}
	return nil
}

func normalize(items []d.ProcessInstanceVariable) []d.ProcessInstanceVariable {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].VariableKey < items[j].VariableKey
	})
	return items
}

func malformed(field string, err error) error {
	return fmt.Errorf("%w: invalid process-instance variable %s: %v", d.ErrMalformedResponse, field, err)
}
