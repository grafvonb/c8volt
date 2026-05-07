// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

// fromIncidentResult maps a Camunda incident result to the shared domain detail model.
func fromIncidentResult(r camundav89.IncidentResult) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{
		IncidentKey:            r.IncidentKey,
		ProcessInstanceKey:     r.ProcessInstanceKey,
		TenantId:               r.TenantId,
		State:                  string(r.State),
		ErrorType:              string(r.ErrorType),
		ErrorMessage:           r.ErrorMessage,
		FlowNodeId:             r.ElementId,
		FlowNodeInstanceKey:    r.ElementInstanceKey,
		JobKey:                 valueOrEmpty(r.JobKey),
		RootProcessInstanceKey: valueOrEmpty(r.RootProcessInstanceKey),
		ProcessDefinitionKey:   r.ProcessDefinitionKey,
		ProcessDefinitionId:    r.ProcessDefinitionId,
	}
}

// newStringEqFilterPtr builds a v8.9 string equality filter when a value is set.
func newStringEqFilterPtr(v string) (*camundav89.StringFilterProperty, error) {
	if v == "" {
		return nil, nil
	}
	var f camundav89.StringFilterProperty
	if err := f.FromStringFilterProperty0(v); err != nil {
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
