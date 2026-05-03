// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	tasklistv88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/tasklist"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromUserTaskResult(r camundav88.UserTaskResult) d.UserTask {
	return d.UserTask{
		Key:                r.UserTaskKey,
		ProcessInstanceKey: r.ProcessInstanceKey,
		TenantId:           r.TenantId,
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
