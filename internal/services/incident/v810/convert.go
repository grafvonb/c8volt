// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"time"

	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

// fromIncidentResult maps a Camunda incident result to the shared domain detail model.
func fromIncidentResult(r camundav810.IncidentResult) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{
		IncidentKey:            r.IncidentKey,
		CreationTime:           incidentCreationTime(r.CreationTime),
		ProcessInstanceKey:     r.ProcessInstanceKey,
		TenantId:               r.TenantId,
		State:                  string(r.State),
		ErrorType:              string(r.ErrorType),
		ErrorMessage:           r.ErrorMessage,
		ElementId:              r.ElementId,
		ElementInstanceKey:     r.ElementInstanceKey,
		JobKey:                 valueOrEmpty(r.JobKey),
		RootProcessInstanceKey: valueOrEmpty(r.RootProcessInstanceKey),
		ProcessDefinitionKey:   r.ProcessDefinitionKey,
		ProcessDefinitionId:    r.ProcessDefinitionId,
	}
}

// newStringEqFilterPtr builds a v8.10 string equality filter when a value is set.
func newStringEqFilterPtr(v string) (*camundav810.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav810.StringFilterProperty
	if err := f.FromStringFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}

// newBasicStringInFilterPtr builds a v8.10 basic string filter for one or more keys.
func newBasicStringInFilterPtr(values []string) (*camundav810.BasicStringFilterProperty, error) {
	switch len(values) {
	case 0:
		return nil, nil
	case 1:
		var f camundav810.BasicStringFilterProperty
		if err := f.FromBasicStringFilterProperty0(values[0]); err != nil {
			return nil, err
		}
		return new(f), nil
	default:
		copied := append([]string(nil), values...)
		var f camundav810.BasicStringFilterProperty
		if err := f.FromBasicStringFilter(camundav810.BasicStringFilter{In: &copied}); err != nil {
			return nil, err
		}
		return new(f), nil
	}
}

// newIncidentStateEqFilterPtr builds a v8.10 incident state equality filter.
func newIncidentStateEqFilterPtr(v camundav810.IncidentStateEnum) (*camundav810.IncidentStateFilterProperty, error) {
	var f camundav810.IncidentStateFilterProperty
	if err := f.FromIncidentStateFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}

func newIncidentErrorTypeEqFilterPtr(v camundav810.IncidentErrorTypeEnum) (*camundav810.IncidentErrorTypeFilterProperty, error) {
	var f camundav810.IncidentErrorTypeFilterProperty
	if err := f.FromIncidentErrorTypeFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}

func newProcessInstanceKeyEqFilterPtr(v string) (*camundav810.ProcessInstanceKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav810.ProcessInstanceKeyFilterProperty
	if err := f.FromProcessInstanceKeyFilterProperty0(camundav810.ProcessInstanceKey(v)); err != nil {
		return nil, err
	}
	return new(f), nil
}

func newProcessDefinitionKeyEqFilterPtr(v string) (*camundav810.ProcessDefinitionKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav810.ProcessDefinitionKeyFilterProperty
	if err := f.FromProcessDefinitionKeyFilterProperty0(camundav810.ProcessDefinitionKey(v)); err != nil {
		return nil, err
	}
	return new(f), nil
}

func newElementInstanceKeyEqFilterPtr(v string) (*camundav810.ElementInstanceKeyFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav810.ElementInstanceKeyFilterProperty
	if err := f.FromElementInstanceKeyFilterProperty0(camundav810.ElementInstanceKey(v)); err != nil {
		return nil, err
	}
	return new(f), nil
}

func newDateTimeRangeFilterPtr(after, before *time.Time, exists *bool) (*camundav810.DateTimeFilterProperty, error) {
	if after == nil && before == nil && exists == nil {
		return nil, nil
	}
	var f camundav810.DateTimeFilterProperty
	if err := f.FromAdvancedDateTimeFilter(camundav810.AdvancedDateTimeFilter{
		Gte:    after,
		Lte:    before,
		Exists: exists,
	}); err != nil {
		return nil, err
	}
	return new(f), nil
}

func valueOrEmpty[T ~string](v *T) T {
	if v == nil {
		return ""
	}
	return *v
}

func incidentCreationTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
