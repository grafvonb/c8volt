// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"time"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// newElementSearchFilter builds the generated v8.8 element search filter from
// the version-neutral query. Multiple non-empty fields combine as backend AND
// filters.
func newElementSearchFilter(query d.ElementSearchQuery) (*camundav88.ElementInstanceFilter, error) {
	state, err := newElementStateEqFilterPtr(query.State)
	if err != nil {
		return nil, err
	}
	filter := &camundav88.ElementInstanceFilter{
		ElementInstanceKey:   elementInstanceKeyPtr(query.Key),
		ProcessInstanceKey:   elementProcessInstanceKeyPtr(query.ProcessInstanceKey),
		ElementId:            elementIDPtr(query.ElementId),
		State:                state,
		Type:                 elementTypePtr(query.Type),
		ProcessDefinitionKey: elementProcessDefinitionKeyPtr(query.ProcessDefinitionKey),
		ProcessDefinitionId:  elementProcessDefinitionIDPtr(query.BpmnProcessId),
	}
	return filter, nil
}

func newElementStateEqFilterPtr(v string) (*camundav88.ElementInstanceStateFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav88.ElementInstanceStateFilterProperty
	if err := f.FromElementInstanceStateFilterProperty0(camundav88.ElementInstanceStateEnum(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

func elementInstanceKeyPtr(v string) *camundav88.ElementInstanceKey {
	if v == "" {
		return nil
	}
	value := camundav88.ElementInstanceKey(v)
	return &value
}

func elementProcessInstanceKeyPtr(v string) *camundav88.ProcessInstanceKey {
	if v == "" {
		return nil
	}
	value := camundav88.ProcessInstanceKey(v)
	return &value
}

func elementIDPtr(v string) *camundav88.ElementId {
	if v == "" {
		return nil
	}
	value := camundav88.ElementId(v)
	return &value
}

func elementTypePtr(v string) *camundav88.ElementInstanceFilterType {
	if v == "" {
		return nil
	}
	value := camundav88.ElementInstanceFilterType(v)
	return &value
}

func elementProcessDefinitionKeyPtr(v string) *camundav88.ProcessDefinitionKey {
	if v == "" {
		return nil
	}
	value := camundav88.ProcessDefinitionKey(v)
	return &value
}

func elementProcessDefinitionIDPtr(v string) *camundav88.ProcessDefinitionId {
	if v == "" {
		return nil
	}
	value := camundav88.ProcessDefinitionId(v)
	return &value
}

func newSearchQueryPageRequest(from int32, limit int32) camundav88.SearchQueryPageRequest {
	var page camundav88.SearchQueryPageRequest
	_ = page.FromOffsetPagination(camundav88.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return page
}

// fromElementInstanceResult maps a generated runtime element instance into the
// version-neutral element model used by facades and commands.
func fromElementInstanceResult(r camundav88.ElementInstanceResult) d.Element {
	return d.Element{
		ElementInstanceKey:     string(r.ElementInstanceKey),
		ElementId:              string(r.ElementId),
		ElementName:            r.ElementName,
		Type:                   string(r.Type),
		State:                  string(r.State),
		StartDate:              formatElementTime(r.StartDate),
		EndDate:                formatElementTimePtr(r.EndDate),
		ProcessInstanceKey:     string(r.ProcessInstanceKey),
		RootProcessInstanceKey: string(toolx.Deref(r.RootProcessInstanceKey, "")),
		ProcessDefinitionId:    string(r.ProcessDefinitionId),
		ProcessDefinitionKey:   string(r.ProcessDefinitionKey),
		TenantId:               string(r.TenantId),
		HasIncident:            r.HasIncident,
		IncidentKey:            string(toolx.Deref(r.IncidentKey, "")),
	}
}

// fromElementInstanceResults maps generated rows without leaking generated
// types past the service layer.
func fromElementInstanceResults(items []camundav88.ElementInstanceResult) []d.Element {
	out := make([]d.Element, 0, len(items))
	for _, item := range items {
		out = append(out, fromElementInstanceResult(item))
	}
	return out
}

func formatElementTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.Format(time.RFC3339Nano)
}

func formatElementTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatElementTime(*value)
}
