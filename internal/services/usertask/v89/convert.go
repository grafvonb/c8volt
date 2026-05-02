// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

func fromUserTaskResult(r camundav89.UserTaskResult) d.UserTask {
	return d.UserTask{
		Key:                r.UserTaskKey,
		ProcessInstanceKey: r.ProcessInstanceKey,
		TenantId:           r.TenantId,
	}
}
