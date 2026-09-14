// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromUserTaskResult(r camundav810.UserTaskResult) d.UserTask {
	return d.UserTask{
		Key:                  r.UserTaskKey,
		State:                string(r.State),
		Name:                 toolx.Deref(r.Name, ""),
		ElementId:            r.ElementId,
		ElementInstanceKey:   r.ElementInstanceKey,
		Assignee:             toolx.Deref(r.Assignee, ""),
		CandidateUsers:       append([]string(nil), r.CandidateUsers...),
		CandidateGroups:      append([]string(nil), r.CandidateGroups...),
		ProcessInstanceKey:   r.ProcessInstanceKey,
		ProcessDefinitionKey: r.ProcessDefinitionKey,
		ProcessDefinitionId:  r.ProcessDefinitionId,
		TenantId:             r.TenantId,
	}
}
