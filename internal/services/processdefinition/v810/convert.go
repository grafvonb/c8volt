// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
)

// fromProcessDefinitionResult preserves supplied lifecycle values without inference or normalization.
func fromProcessDefinitionResult(r camundav810.ProcessDefinitionResult) d.ProcessDefinition {
	return d.ProcessDefinition{
		State:             string(r.State),
		BpmnProcessId:     r.ProcessDefinitionId,
		Key:               r.ProcessDefinitionKey,
		Name:              valueOrEmpty(r.Name),
		TenantId:          r.TenantId,
		ProcessVersion:    r.Version,
		ProcessVersionTag: valueOrEmpty(r.VersionTag),
	}
}

func fromProcessElementStatisticsResult(r camundav810.ProcessElementStatisticsResult) d.ProcessDefinitionStatistics {
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
