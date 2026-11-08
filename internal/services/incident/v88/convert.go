// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"time"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

// fromIncidentResult maps a Camunda incident result to the shared domain detail model.
func fromIncidentResult(r camundav88.IncidentResult) d.ProcessInstanceIncidentDetail {
	return d.ProcessInstanceIncidentDetail{
		IncidentKey:          toolx.Deref(r.IncidentKey, ""),
		CreationTime:         incidentCreationTime(toolx.Deref(r.CreationTime, time.Time{})),
		ProcessInstanceKey:   toolx.Deref(r.ProcessInstanceKey, ""),
		TenantId:             toolx.Deref(r.TenantId, ""),
		State:                string(toolx.Deref(r.State, "")),
		ErrorType:            string(toolx.Deref(r.ErrorType, "")),
		ErrorMessage:         toolx.Deref(r.ErrorMessage, ""),
		ElementId:            toolx.Deref(r.ElementId, ""),
		ElementInstanceKey:   toolx.Deref(r.ElementInstanceKey, ""),
		JobKey:               toolx.Deref(r.JobKey, ""),
		ProcessDefinitionKey: toolx.Deref(r.ProcessDefinitionKey, ""),
		ProcessDefinitionId:  toolx.Deref(r.ProcessDefinitionId, ""),
	}
}

func incidentCreationTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339Nano)
}
