// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	tasklistv88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromUserTaskResult maps a generated Camunda task into the domain model, normalizing optional
// fields and copying candidate lists.
func fromUserTaskResult(r camundav88.UserTaskResult) d.UserTask {
	return d.UserTask{
		Key:                      r.UserTaskKey,
		State:                    string(toolx.Deref(r.State, "")),
		Name:                     toolx.Deref(r.Name, ""),
		ElementId:                toolx.Deref(r.ElementId, ""),
		ElementInstanceKey:       toolx.Deref(r.ElementInstanceKey, ""),
		Assignee:                 toolx.Deref(r.Assignee, ""),
		CandidateUsers:           append([]string(nil), toolx.Deref(r.CandidateUsers, nil)...),
		CandidateGroups:          append([]string(nil), toolx.Deref(r.CandidateGroups, nil)...),
		ProcessInstanceKey:       toolx.Deref(r.ProcessInstanceKey, ""),
		ProcessDefinitionKey:     toolx.Deref(r.ProcessDefinitionKey, ""),
		ProcessDefinitionId:      toolx.Deref(r.ProcessDefinitionId, ""),
		ProcessDefinitionVersion: toolx.Deref(r.ProcessDefinitionVersion, 0),
		TenantId:                 toolx.Deref(r.TenantId, ""),
	}
}

// fromTaskResponse normalizes a legacy Tasklist task so callers can reuse process-instance lookup.
func fromTaskResponse(r tasklistv88.TaskResponse) d.UserTask {
	return d.UserTask{
		Key:                toolx.Deref(r.Id, ""),
		ProcessInstanceKey: toolx.Deref(r.ProcessInstanceKey, ""),
		TenantId:           toolx.Deref(r.TenantId, ""),
	}
}
