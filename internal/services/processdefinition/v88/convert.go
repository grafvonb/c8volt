// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/toolx"
)

func fromProcessDefinitionResult(r camundav88.ProcessDefinitionResult) d.ProcessDefinition {
	return d.ProcessDefinition{
		BpmnProcessId:     toolx.Deref(r.ProcessDefinitionId, ""),
		Key:               toolx.Deref(r.ProcessDefinitionKey, ""),
		Name:              valueOrEmpty(r.Name),
		TenantId:          toolx.Deref(r.TenantId, ""),
		ProcessVersion:    toolx.Deref(r.Version, 0),
		ProcessVersionTag: valueOrEmpty(r.VersionTag),
	}
}

func fromProcessElementStatisticsResult(r camundav88.ProcessElementStatisticsResult) d.ProcessDefinitionStatistics {
	return d.ProcessDefinitionStatistics{
		Active:    toolx.Deref(r.Active, 0),
		Canceled:  toolx.Deref(r.Canceled, 0),
		Completed: toolx.Deref(r.Completed, 0),
		Incidents: toolx.Deref(r.Incidents, 0),
	}
}

func valueOrEmpty[T ~string](v *T) T {
	if v == nil {
		return ""
	}
	return *v
}
