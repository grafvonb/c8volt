// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromIncidentResult maps a Camunda incident result to the shared domain detail model.
func fromIncidentResult(r camundav88.IncidentResult) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{
		IncidentKey:            r.IncidentKey,
		ProcessInstanceKey:     r.ProcessInstanceKey,
		TenantId:               r.TenantId,
		State:                  string(r.State),
		ErrorType:              string(r.ErrorType),
		ErrorMessage:           r.ErrorMessage,
		FlowNodeId:             r.ElementId,
		FlowNodeInstanceKey:    r.ElementInstanceKey,
		JobKey:                 toolx.Deref(r.JobKey, ""),
		RootProcessInstanceKey: toolx.Deref(r.RootProcessInstanceKey, ""),
		ProcessDefinitionKey:   r.ProcessDefinitionKey,
		ProcessDefinitionId:    r.ProcessDefinitionId,
	}
}

// newIncidentStateEqFilterPtr builds a v8.8 incident state equality filter.
func newIncidentStateEqFilterPtr(v camundav88.IncidentStateEnum) (*camundav88.IncidentStateFilterProperty, error) {
	var f camundav88.IncidentStateFilterProperty
	if err := f.FromIncidentStateFilterProperty0(v); err != nil {
		return nil, err
	}
	return new(f), nil
}
