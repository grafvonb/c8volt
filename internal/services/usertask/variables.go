// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"fmt"
	"math"
	"sort"

	"github.com/grafvonb/c8volt/consts"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// SearchUserTaskEffectiveVariables retrieves and validates every offset page
// before normalizing the backend-selected effective names.
func SearchUserTaskEffectiveVariables(ctx context.Context, api API, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	_ = services.ApplyCallOptions(opts)
	request := d.UserTaskVariablePageRequest{Size: consts.MaxPISearchSize}
	items := make([]d.ProcessInstanceVariable, 0)
	var rawObserved int64
	var retainedLowerBound int64

	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		page, err := api.SearchUserTaskEffectiveVariablesPage(ctx, key, request, opts...)
		if err != nil {
			return nil, err
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := validateUserTaskVariablePage(page, request, rawObserved, retainedLowerBound); err != nil {
			return nil, err
		}
		if rawObserved > math.MaxInt64-int64(page.RawItemCount) {
			return nil, malformedUserTaskVariableMetadata("raw count", fmt.Errorf("int64 overflow after %d variables", rawObserved))
		}
		rawObserved += int64(page.RawItemCount)
		items = append(items, page.Items...)

		switch page.ReportedTotal.Kind {
		case d.UserTaskReportedTotalKindExact:
			if rawObserved == page.ReportedTotal.Count {
				return normalizeUserTaskEffectiveVariables(items)
			}
		case d.UserTaskReportedTotalKindLowerBound:
			if page.ReportedTotal.Count > retainedLowerBound {
				retainedLowerBound = page.ReportedTotal.Count
			}
			if page.RawItemCount == 0 && rawObserved >= retainedLowerBound && !page.HasContinuationEvidence {
				return normalizeUserTaskEffectiveVariables(items)
			}
		}

		advance := int64(page.RawItemCount)
		if advance == 0 {
			advance = int64(request.Size)
		}
		nextOffset := int64(request.From) + advance
		if nextOffset > math.MaxInt32 {
			return nil, malformedUserTaskVariableMetadata("offset", fmt.Errorf("position %d exceeds int32", nextOffset))
		}
		request = d.UserTaskVariablePageRequest{From: int32(nextOffset), Size: request.Size}
	}
}

// EnrichUserTasksWithVariables attaches complete effective-variable collections
// sequentially without changing selected task order or metadata.
func EnrichUserTasksWithVariables(ctx context.Context, api API, tasks []d.UserTask, opts ...services.CallOption) (d.VariableEnrichedUserTasks, error) {
	_ = services.ApplyCallOptions(opts)
	items := make([]d.VariableEnrichedUserTask, 0, len(tasks))
	for _, task := range tasks {
		if err := ctx.Err(); err != nil {
			return d.VariableEnrichedUserTasks{}, err
		}
		variables, err := SearchUserTaskEffectiveVariables(ctx, api, task.Key, opts...)
		if err != nil {
			return d.VariableEnrichedUserTasks{}, err
		}
		items = append(items, d.VariableEnrichedUserTask{Item: task, Variables: variables})
	}
	return d.VariableEnrichedUserTasks{Total: int64(len(items)), Items: items}, nil
}

// validateUserTaskVariablePage rejects page facts that could skip variables,
// terminate early, or contradict previously retained lower-bound metadata.
func validateUserTaskVariablePage(page d.UserTaskVariablePage, request d.UserTaskVariablePageRequest, rawBefore int64, retainedLowerBound int64) error {
	if page.Request != request {
		return malformedUserTaskVariableMetadata("request", fmt.Errorf("got %+v, want %+v", page.Request, request))
	}
	if page.RawItemCount < 0 {
		return malformedUserTaskVariableMetadata("raw count", fmt.Errorf("cannot be negative"))
	}
	if int64(len(page.Items)) > int64(page.RawItemCount) {
		return malformedUserTaskVariableMetadata("raw count", fmt.Errorf("%d items exceed %d raw items", len(page.Items), page.RawItemCount))
	}
	if err := page.ReportedTotal.Validate(); err != nil {
		return malformedUserTaskVariableMetadata("reported total", err)
	}
	if rawBefore > math.MaxInt64-int64(page.RawItemCount) {
		return malformedUserTaskVariableMetadata("raw count", fmt.Errorf("int64 overflow after %d variables", rawBefore))
	}
	observed := rawBefore + int64(page.RawItemCount)
	if page.ReportedTotal.Kind == d.UserTaskReportedTotalKindExact {
		if page.ReportedTotal.Count < observed {
			return malformedUserTaskVariableMetadata("reported total", fmt.Errorf("exact count %d is below %d observed variables", page.ReportedTotal.Count, observed))
		}
		if page.ReportedTotal.Count < retainedLowerBound {
			return malformedUserTaskVariableMetadata("reported total", fmt.Errorf("exact count %d is below retained lower bound %d", page.ReportedTotal.Count, retainedLowerBound))
		}
	}
	return nil
}

// normalizeUserTaskEffectiveVariables sorts effective names and rejects
// inconsistent duplicate records instead of choosing a winning scope locally.
func normalizeUserTaskEffectiveVariables(items []d.ProcessInstanceVariable) ([]d.ProcessInstanceVariable, error) {
	byName := make(map[string]d.ProcessInstanceVariable, len(items))
	for _, item := range items {
		if existing, ok := byName[item.Name]; ok {
			if existing != item {
				return nil, malformedUserTaskVariableMetadata("effective name", fmt.Errorf("conflicting records for %q", item.Name))
			}
			continue
		}
		byName[item.Name] = item
	}
	normalized := make([]d.ProcessInstanceVariable, 0, len(byName))
	for _, item := range byName {
		normalized = append(normalized, item)
	}
	sort.SliceStable(normalized, func(i, j int) bool { return normalized[i].Name < normalized[j].Name })
	return normalized, nil
}

// malformedUserTaskVariableMetadata classifies traversal inconsistencies
// through the repository's malformed-response sentinel.
func malformedUserTaskVariableMetadata(field string, err error) error {
	return fmt.Errorf("%w: invalid user-task effective-variable %s: %v", d.ErrMalformedResponse, field, err)
}
