// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"time"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// newElementSearchFilter builds the generated v8.9 element search filter from
// the version-neutral query. Multiple non-empty fields combine as backend AND
// filters.
func newElementSearchFilter(query d.ElementSearchQuery) (*camundav89.ElementInstanceFilter, error) {
	state, err := newElementStateEqFilterPtr(query.State)
	if err != nil {
		return nil, err
	}
	filter := &camundav89.ElementInstanceFilter{
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

func newElementStateEqFilterPtr(v string) (*camundav89.ElementInstanceStateFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.ElementInstanceStateFilterProperty
	if err := f.FromElementInstanceStateFilterProperty0(camundav89.ElementInstanceStateEnum(v)); err != nil {
		return nil, err
	}
	return &f, nil
}

func elementInstanceKeyPtr(v string) *camundav89.ElementInstanceKey {
	if v == "" {
		return nil
	}
	value := camundav89.ElementInstanceKey(v)
	return &value
}

func elementProcessInstanceKeyPtr(v string) *camundav89.ProcessInstanceKey {
	if v == "" {
		return nil
	}
	value := camundav89.ProcessInstanceKey(v)
	return &value
}

func elementIDPtr(v string) *camundav89.ElementId {
	if v == "" {
		return nil
	}
	value := camundav89.ElementId(v)
	return &value
}

func elementTypePtr(v string) *camundav89.ElementInstanceFilterType {
	if v == "" {
		return nil
	}
	value := camundav89.ElementInstanceFilterType(v)
	return &value
}

func elementProcessDefinitionKeyPtr(v string) *camundav89.ProcessDefinitionKey {
	if v == "" {
		return nil
	}
	value := camundav89.ProcessDefinitionKey(v)
	return &value
}

func elementProcessDefinitionIDPtr(v string) *camundav89.ProcessDefinitionId {
	if v == "" {
		return nil
	}
	value := camundav89.ProcessDefinitionId(v)
	return &value
}

func newSearchQueryPageRequest(from int32, limit int32) camundav89.SearchQueryPageRequest {
	var page camundav89.SearchQueryPageRequest
	_ = page.FromOffsetPagination(camundav89.OffsetPagination{
		From:  &from,
		Limit: &limit,
	})
	return page
}

// fromElementInstanceResult maps a generated runtime element instance into the
// version-neutral element model used by facades and commands.
func fromElementInstanceResult(r camundav89.ElementInstanceResult) d.Element {
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
func fromElementInstanceResults(items []camundav89.ElementInstanceResult) []d.Element {
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
