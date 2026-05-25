// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"fmt"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

func newJobKeyEqFilterPtr(v string) (*camundav89.JobKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.JobKeyFilterProperty
	if err := f.FromJobKeyFilterProperty0(camundav89.JobKey(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

// newJobSearchFilter builds the generated v8.9 job search filter from the
// version-neutral query while keeping zero retries as an explicit filter.
func newJobSearchFilter(query d.JobSearchQuery) (*camundav89.JobFilter, error) {
	state, err := newJobStateEqFilterPtr(query.State)
	if err != nil {
		return nil, err
	}
	processInstanceKey, err := newProcessInstanceKeyEqFilterPtr(query.ProcessInstanceKey)
	if err != nil {
		return nil, err
	}
	elementInstanceKey, err := newElementInstanceKeyEqFilterPtr(query.ElementInstanceKey)
	if err != nil {
		return nil, err
	}
	retries, err := newJobRetriesEqFilterPtr(query.Retries)
	if err != nil {
		return nil, err
	}
	kind, err := newJobKindEqFilterPtr(query.Kind)
	if err != nil {
		return nil, err
	}
	listenerEventType, err := newJobListenerEventTypeEqFilterPtr(query.ListenerEventType)
	if err != nil {
		return nil, err
	}
	jobType, err := newStringEqFilterPtr(query.Type)
	if err != nil {
		return nil, err
	}
	elementID, err := newStringEqFilterPtr(query.ElementId)
	if err != nil {
		return nil, err
	}
	worker, err := newStringEqFilterPtr(query.Worker)
	if err != nil {
		return nil, err
	}
	return &camundav89.JobFilter{
		State:              state,
		Type:               jobType,
		ProcessInstanceKey: processInstanceKey,
		ElementInstanceKey: elementInstanceKey,
		ElementId:          elementID,
		Worker:             worker,
		Retries:            retries,
		Kind:               kind,
		ListenerEventType:  listenerEventType,
	}, nil
}

// newStringEqFilterPtr wraps a non-empty string as a generated equality filter.
func newStringEqFilterPtr(v string) (*camundav89.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.StringFilterProperty
	if err := f.FromStringFilterProperty0(v); err != nil {
		return nil, err
	}
	return &f, nil
}

// newProcessInstanceKeyEqFilterPtr wraps a process instance key as a generated equality filter.
func newProcessInstanceKeyEqFilterPtr(v string) (*camundav89.ProcessInstanceKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.ProcessInstanceKeyFilterProperty
	if err := f.FromProcessInstanceKeyFilterProperty0(camundav89.ProcessInstanceKey(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

// newJobStateEqFilterPtr wraps a validated job state as a generated equality filter.
func newJobStateEqFilterPtr(v string) (*camundav89.JobStateFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.JobStateFilterProperty
	if err := f.FromJobStateFilterProperty0(camundav89.JobStateEnum(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

// newElementInstanceKeyEqFilterPtr wraps an element instance key as a generated equality filter.
func newElementInstanceKeyEqFilterPtr(v string) (*camundav89.ElementInstanceKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.ElementInstanceKeyFilterProperty
	if err := f.FromElementInstanceKeyFilterProperty0(camundav89.ElementInstanceKey(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

// newJobRetriesEqFilterPtr preserves an explicit zero retry filter.
func newJobRetriesEqFilterPtr(v *int32) (*camundav89.IntegerFilterProperty, error) {
	if v == nil {
		return nil, nil
	}
	var f camundav89.IntegerFilterProperty
	if err := f.FromIntegerFilterProperty0(*v); err != nil {
		return nil, err
	}
	return &f, nil
}

// newJobKindEqFilterPtr wraps a validated job kind as a generated equality filter.
func newJobKindEqFilterPtr(v string) (*camundav89.JobKindFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.JobKindFilterProperty
	if err := f.FromJobKindFilterProperty0(camundav89.JobKindEnum(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

// newJobListenerEventTypeEqFilterPtr wraps a validated listener event type as a generated equality filter.
func newJobListenerEventTypeEqFilterPtr(v string) (*camundav89.JobListenerEventTypeFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.JobListenerEventTypeFilterProperty
	if err := f.FromJobListenerEventTypeFilterProperty0(camundav89.JobListenerEventTypeEnum(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

func newSearchQueryPageRequest(from int32, limit int32) camundav89.SearchQueryPageRequest {
	var page camundav89.SearchQueryPageRequest
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return page
}

func fromJobSearchResult(r camundav89.JobSearchResult) d.Job {
	return d.Job{
		Key:                string(r.JobKey),
		State:              string(r.State),
		Retries:            r.Retries,
		Deadline:           r.Deadline,
		Type:               r.Type,
		Worker:             r.Worker,
		Kind:               string(r.Kind),
		ListenerEventType:  string(r.ListenerEventType),
		ProcessInstanceKey: string(r.ProcessInstanceKey),
		ElementInstanceKey: string(r.ElementInstanceKey),
		ElementId:          elementIDValue(r.ElementId),
		ErrorCode:          stringPtrValue(r.ErrorCode),
		ErrorMessage:       stringPtrValue(r.ErrorMessage),
		TenantId:           string(r.TenantId),
	}
}

// fromJobSearchResults maps generated rows without leaking generated types past the service layer.
func fromJobSearchResults(items []camundav89.JobSearchResult) []d.Job {
	out := make([]d.Job, 0, len(items))
	for _, item := range items {
		out = append(out, fromJobSearchResult(item))
	}
	return out
}

func requireSingleJob(items []camundav89.JobSearchResult, key string) (d.Job, error) {
	switch len(items) {
	case 0:
		return d.Job{}, fmt.Errorf("%w: job %s was not found or is not visible to the configured tenant", d.ErrNotFound, key)
	case 1:
		return fromJobSearchResult(items[0]), nil
	default:
		return d.Job{}, fmt.Errorf("%w: get job for key %s returned %d matches", d.ErrMalformedResponse, key, len(items))
	}
}

func stringPtrValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// elementIDValue unwraps the optional generated element ID alias.
func elementIDValue(value *camundav89.ElementId) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
