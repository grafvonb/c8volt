// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

func fromProcessDefinitionResult(r camundav89.ProcessDefinitionResult) d.ProcessDefinition {
	return d.ProcessDefinition{
		BpmnProcessId:     r.ProcessDefinitionId,
		Key:               r.ProcessDefinitionKey,
		Name:              valueOrEmpty(r.Name),
		TenantId:          r.TenantId,
		ProcessVersion:    r.Version,
		ProcessVersionTag: valueOrEmpty(r.VersionTag),
	}
}

func fromProcessElementStatisticsResult(r camundav89.ProcessElementStatisticsResult) d.ProcessDefinitionStatistics {
	return d.ProcessDefinitionStatistics{
		Active:    r.Active,
		Canceled:  r.Canceled,
		Completed: r.Completed,
		Incidents: r.Incidents,
	}
}

func valueOrEmpty[T ~string](v *T) T {
	if v == nil {
		return ""
	}
	return *v
}
