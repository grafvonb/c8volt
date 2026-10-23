// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

type DeleteProcessDefinitionPlan struct {
	Items                 []DeleteProcessDefinitionPlanItem
	StateCheckSkipped     bool
	ProcessDefinitionKeys []string
	Warnings              []string
	TenantEvidence        TenantEvidence
}

// DeleteProcessDefinitionPlanItem preserves separate display counts and authoritative cleanup evidence.
type DeleteProcessDefinitionPlanItem struct {
	Key                            string
	BpmnProcessId                  string
	ProcessVersion                 int32
	ProcessVersionTag              string
	TenantId                       string
	ActiveProcessInstanceCount     int64
	ActiveProcessInstanceKeys      []string
	SuspendedProcessInstanceCount  int64
	SuspendedCountSupported        bool
	UnfinishedProcessInstanceCount int64
	UnfinishedProcessInstanceKeys  []string
	CancellationPlan               DryRunPIKeyExpansion
	Warnings                       []string
}

// UnfinishedProcessInstances returns the observed count without understating discovered keys.
func (i DeleteProcessDefinitionPlanItem) UnfinishedProcessInstances() int64 {
	return max(i.UnfinishedProcessInstanceCount, int64(len(i.UnfinishedProcessInstanceKeys)))
}

// ActiveProcessInstances retains the active-only count/key fallback.
func (i DeleteProcessDefinitionPlanItem) ActiveProcessInstances() int64 {
	if i.ActiveProcessInstanceCount > 0 {
		return i.ActiveProcessInstanceCount
	}
	return int64(len(i.ActiveProcessInstanceKeys))
}
