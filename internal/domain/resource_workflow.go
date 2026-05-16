// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type DeleteProcessDefinitionPlan struct {
	Items                 []DeleteProcessDefinitionPlanItem
	StateCheckSkipped     bool
	ProcessDefinitionKeys []string
	Warnings              []string
}

type DeleteProcessDefinitionPlanItem struct {
	Key                        string
	BpmnProcessId              string
	ProcessVersion             int32
	ProcessVersionTag          string
	ActiveProcessInstanceCount int64
	ActiveProcessInstanceKeys  []string
	CancellationPlan           DryRunPIKeyExpansion
	Warnings                   []string
}

func (i DeleteProcessDefinitionPlanItem) ActiveProcessInstances() int64 {
	if i.ActiveProcessInstanceCount > 0 {
		return i.ActiveProcessInstanceCount
	}
	return int64(len(i.ActiveProcessInstanceKeys))
}
