// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"fmt"
	"sort"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/toolx"
)

// AnalyseSlowProcessInstances coordinates read-only slow process analysis below the command layer.
func (s *Service) AnalyseSlowProcessInstances(ctx context.Context, request d.SlowProcessAnalysisRequest, opts ...services.CallOption) (d.SlowProcessAnalysisResult, error) {
	capturedAt := request.CapturedNow
	if capturedAt.IsZero() {
		capturedAt = time.Now().UTC()
		request.CapturedNow = capturedAt
	}
	result := d.SlowProcessAnalysisResult{
		Request:    request,
		CapturedAt: capturedAt,
		Items:      []d.SlowProcessAnalysisProcessInstance{},
		Empty:      true,
	}
	if s == nil || s.piAPI == nil {
		return result, fmt.Errorf("%w: process-instance service is required for slow process analysis", d.ErrPrecondition)
	}
	if s.version == toolx.V87 {
		return result, fmt.Errorf("%w: slow process analysis requires Camunda 8.8 or newer", d.ErrUnsupported)
	}
	if request.SelectionMode != d.SlowProcessAnalysisSelectionModeExplicitKeys {
		return result, fmt.Errorf("%w: slow process analysis search selection is pending", d.ErrUnsupported)
	}

	keys := request.InputKeys.Unique()
	if len(keys) == 0 {
		return result, fmt.Errorf("%w: at least one process-instance key is required", d.ErrValidation)
	}
	instances := make([]d.ProcessInstance, 0, len(keys))
	for _, key := range keys {
		pi, err := pisvc.LookupProcessInstance(ctx, s.piAPI, key, opts...)
		if err != nil {
			return result, fmt.Errorf("lookup process instance %s: %w", key, err)
		}
		instances = append(instances, pi)
	}

	items := make([]d.SlowProcessAnalysisProcessInstance, 0, len(instances))
	for _, pi := range instances {
		items = append(items, slowProcessAnalysisProcessInstanceFromDomain(pi, capturedAt))
	}
	sort.SliceStable(items, func(i, j int) bool {
		left, right := items[i], items[j]
		if left.DurationAvailable != right.DurationAvailable {
			return left.DurationAvailable
		}
		if left.DurationAvailable && left.DurationMillis != right.DurationMillis {
			return left.DurationMillis > right.DurationMillis
		}
		return left.Key < right.Key
	})

	result.Items = items
	result.Count = len(items)
	result.Empty = len(items) == 0
	request.InputKeys = keys
	result.Request = request
	return result, nil
}

// slowProcessAnalysisProcessInstanceFromDomain preserves selected root metadata while adding whole-instance duration.
func slowProcessAnalysisProcessInstanceFromDomain(pi d.ProcessInstance, capturedAt time.Time) d.SlowProcessAnalysisProcessInstance {
	duration, millis, available := slowProcessAnalysisProcessDuration(pi, capturedAt)
	return slowProcessAnalysisRootFromProcessInstance(pi, duration, millis, available)
}

// slowProcessAnalysisProcessDuration returns a measured duration only when timestamps prove it.
func slowProcessAnalysisProcessDuration(pi d.ProcessInstance, capturedAt time.Time) (string, int64, bool) {
	start, err := time.Parse(time.RFC3339Nano, pi.StartDate)
	if err != nil || start.IsZero() {
		return "", 0, false
	}
	var end time.Time
	switch {
	case pi.EndDate != "":
		end, err = time.Parse(time.RFC3339Nano, pi.EndDate)
		if err != nil || end.Before(start) {
			return "", 0, false
		}
	case !pi.State.IsTerminal():
		end = capturedAt
		if end.Before(start) {
			return "", 0, false
		}
	default:
		return "", 0, false
	}
	duration := end.Sub(start)
	return duration.String(), duration.Milliseconds(), true
}

// slowProcessAnalysisRootFromProcessInstance copies root process-instance fields into analysis output.
func slowProcessAnalysisRootFromProcessInstance(pi d.ProcessInstance, duration string, millis int64, available bool) d.SlowProcessAnalysisProcessInstance {
	return d.SlowProcessAnalysisProcessInstance{
		Key:                    pi.Key,
		TenantID:               pi.TenantId,
		BpmnProcessID:          pi.BpmnProcessId,
		ProcessDefinitionKey:   pi.ProcessDefinitionKey,
		ProcessVersion:         pi.ProcessVersion,
		State:                  pi.State,
		StartDate:              pi.StartDate,
		EndDate:                pi.EndDate,
		ParentKey:              pi.ParentKey,
		RootProcessInstanceKey: pi.RootProcessInstanceKey,
		Incident:               pi.Incident,
		Duration:               duration,
		DurationMillis:         millis,
		DurationAvailable:      available,
		Timeline:               []d.SlowProcessAnalysisTimelineEntry{},
	}
}
