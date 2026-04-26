// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	camundav87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/camunda"
	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

const (
	unknownProcessInstanceKeyV87   = "<unknown in v87>"
	unknownProcessDefinitionKeyV87 = "<unknown in v87>"
)

func fromProcessInstanceResponse(r operatev87.ProcessInstance) d.ProcessInstance {
	return d.ProcessInstance{
		BpmnProcessId:             toolx.Deref(r.BpmnProcessId, ""),
		EndDate:                   toolx.Deref(r.EndDate, ""),
		Incident:                  toolx.Deref(r.Incident, false),
		Key:                       toolx.Int64PtrToString(r.Key),
		ParentFlowNodeInstanceKey: toolx.Int64PtrToString(r.ParentFlowNodeInstanceKey),
		ParentKey:                 toolx.Int64PtrToString(r.ParentKey),
		ProcessDefinitionKey:      toolx.Int64PtrToString(r.ProcessDefinitionKey),
		ProcessVersion:            toolx.Deref(r.ProcessVersion, int32(0)),
		ProcessVersionTag:         toolx.Deref(r.ProcessVersionTag, ""),
		StartDate:                 toolx.Deref(r.StartDate, ""),
		State:                     d.State(*r.State),
		TenantId:                  toolx.Deref(r.TenantId, ""),
	}
}

func toProcessInstanceCreationInstruction(pi d.ProcessInstanceData) camundav87.ProcessInstanceCreationInstruction {
	return camundav87.PostProcessInstancesJSONRequestBody{
		ProcessDefinitionId:      toolx.PtrIf(pi.BpmnProcessId, ""),
		ProcessDefinitionVersion: toolx.PtrIfNonZero(pi.ProcessDefinitionVersion),
		TenantId:                 toolx.PtrIf(pi.TenantId, ""),
		Variables:                toolx.PtrCopyMap(pi.Variables),
	}
}

func fromPostProcessInstancesResponse(r camundav87.CreateProcessInstanceResult) d.ProcessInstanceCreation {
	return d.ProcessInstanceCreation{
		Key:                      unknownProcessInstanceKeyV87,
		BpmnProcessId:            toolx.Deref(r.ProcessDefinitionId, ""),
		ProcessDefinitionKey:     unknownProcessDefinitionKeyV87,
		ProcessDefinitionVersion: toolx.Deref(r.ProcessDefinitionVersion, int32(0)),
		TenantId:                 toolx.Deref(r.TenantId, ""),
		Variables:                toolx.CopyMap(*r.Variables),
	}
}
