// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"time"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

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
