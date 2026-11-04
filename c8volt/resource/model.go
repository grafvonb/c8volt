// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	pdsvc "github.com/grafvonb/c8volt/internal/services/processdefinition"
	"github.com/grafvonb/c8volt/toolx"

	"github.com/grafvonb/c8volt/c8volt/process"
)

type ProcessDefinitionDeployment struct {
	Key               string `json:"key"`
	DefinitionId      string `json:"processDefinitionId,omitempty"`
	DefinitionKey     string `json:"processDefinitionKey,omitempty"`
	DefinitionVersion int32  `json:"processDefinitionVersion,omitempty"`
	ResourceName      string `json:"resourceName,omitempty"`
	TenantId          string `json:"tenantId,omitempty"`
}

type Resource struct {
	ID         string `json:"id,omitempty"`
	Key        string `json:"key,omitempty"`
	Name       string `json:"name,omitempty"`
	TenantId   string `json:"tenantId,omitempty"`
	Version    int32  `json:"version,omitempty"`
	VersionTag string `json:"versionTag,omitempty"`
}

type DeploymentUnitData struct {
	Name        string // filename for multipart
	ContentType string // e.g. application/xml
	Data        []byte
}

type DeleteReport struct {
	Key               string `json:"key,omitempty"`
	Ok                bool   `json:"ok,omitempty"`
	StatusCode        int    `json:"statusCode,omitempty"`
	Status            string `json:"status,omitempty"`
	DeleteHistory     bool   `json:"deleteHistory,omitempty"`
	BatchOperationKey string `json:"batchOperationKey,omitempty"`
	BatchState        string `json:"batchState,omitempty"`
}

func (r DeleteReport) OK() bool {
	return r.Ok
}

type DeleteReports struct {
	Items []DeleteReport `json:"items,omitempty"`
}

func (c DeleteReports) Totals() (total int, oks int, noks int) {
	return process.TotalsOf(c.Items)
}

type DeleteProcessDefinitionPlan struct {
	Items                 []DeleteProcessDefinitionPlanItem `json:"items,omitempty"`
	StateCheckSkipped     bool                              `json:"stateCheckSkipped,omitempty"`
	ProcessDefinitionKeys []string                          `json:"processDefinitionKeys,omitempty"`
	Warnings              []string                          `json:"warnings,omitempty"`
}

// DeleteProcessDefinitionPlanItem preserves separate display counts and authoritative cleanup evidence.
type DeleteProcessDefinitionPlanItem struct {
	Key                            string                       `json:"key,omitempty"`
	BpmnProcessId                  string                       `json:"bpmnProcessId,omitempty"`
	ProcessVersion                 int32                        `json:"processVersion,omitempty"`
	ProcessVersionTag              string                       `json:"versionTag,omitempty"`
	TenantId                       string                       `json:"tenantId,omitempty"`
	ActiveProcessInstanceCount     int64                        `json:"activeProcessInstanceCount,omitempty"`
	ActiveProcessInstanceKeys      []string                     `json:"activeProcessInstanceKeys,omitempty"`
	SuspendedProcessInstanceCount  int64                        `json:"suspendedProcessInstanceCount,omitempty"`
	SuspendedCountSupported        bool                         `json:"suspendedCountSupported,omitempty"`
	UnfinishedProcessInstanceCount int64                        `json:"unfinishedProcessInstanceCount,omitempty"`
	UnfinishedProcessInstanceKeys  []string                     `json:"unfinishedProcessInstanceKeys,omitempty"`
	CancellationPlan               process.DryRunPIKeyExpansion `json:"cancellationPlan,omitempty"`
	Warnings                       []string                     `json:"warnings,omitempty"`
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

// Totals aggregates observed workload without presenting skipped or unavailable suspension as measured zero.
func (p DeleteProcessDefinitionPlan) Totals() DeleteProcessDefinitionPlanTotals {
	totals := DeleteProcessDefinitionPlanTotals{ProcessDefinitions: len(p.Items), SuspendedCountSupported: !p.StateCheckSkipped && len(p.Items) > 0}
	for _, item := range p.Items {
		totals.ActiveProcessInstances += item.ActiveProcessInstances()
		totals.SuspendedProcessInstances += item.SuspendedProcessInstanceCount
		totals.UnfinishedProcessInstances += item.UnfinishedProcessInstances()
		totals.SuspendedCountSupported = totals.SuspendedCountSupported && item.SuspendedCountSupported
		totals.CancellationRoots += len(item.CancellationPlan.Roots)
		totals.CancellationAffected += len(item.CancellationPlan.Collected)
		totals.Warnings += len(item.Warnings)
	}
	totals.Warnings += len(p.Warnings)
	return totals
}

// TenantEvidence aggregates tenant metadata already present in the process-definition
// impact plan and any nested process-instance cancellation plan.
func (p DeleteProcessDefinitionPlan) TenantEvidence() process.TenantEvidence {
	evidence := pdsvc.ProcessDefinitionPlanTenantEvidence(toolx.MapSlice(p.Items, toDomainDeletePlanEvidenceItem))
	return fromDomainPlanTenantEvidence(evidence)
}

// DeleteProcessDefinitionPlanTotals keeps direct workload distinct from expanded cancellation families.
type DeleteProcessDefinitionPlanTotals struct {
	SuspendedProcessInstances  int64 `json:"suspendedProcessInstances,omitempty"`
	SuspendedCountSupported    bool  `json:"suspendedCountSupported,omitempty"`
	UnfinishedProcessInstances int64 `json:"unfinishedProcessInstances,omitempty"`
	ProcessDefinitions         int   `json:"processDefinitions"`
	ActiveProcessInstances     int64 `json:"activeProcessInstances"`
	CancellationRoots          int   `json:"cancellationRoots"`
	CancellationAffected       int   `json:"cancellationAffected"`
	Warnings                   int   `json:"warnings"`
}
