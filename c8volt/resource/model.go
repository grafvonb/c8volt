// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package resource

import (
	"slices"

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

type DeleteProcessDefinitionPlanItem struct {
	Key                        string                       `json:"key,omitempty"`
	BpmnProcessId              string                       `json:"bpmnProcessId,omitempty"`
	ProcessVersion             int32                        `json:"processVersion,omitempty"`
	ProcessVersionTag          string                       `json:"versionTag,omitempty"`
	TenantId                   string                       `json:"tenantId,omitempty"`
	ActiveProcessInstanceCount int64                        `json:"activeProcessInstanceCount,omitempty"`
	ActiveProcessInstanceKeys  []string                     `json:"activeProcessInstanceKeys,omitempty"`
	CancellationPlan           process.DryRunPIKeyExpansion `json:"cancellationPlan,omitempty"`
	Warnings                   []string                     `json:"warnings,omitempty"`
}

func (i DeleteProcessDefinitionPlanItem) ActiveProcessInstances() int64 {
	if i.ActiveProcessInstanceCount > 0 {
		return i.ActiveProcessInstanceCount
	}
	return int64(len(i.ActiveProcessInstanceKeys))
}

func (p DeleteProcessDefinitionPlan) Totals() DeleteProcessDefinitionPlanTotals {
	totals := DeleteProcessDefinitionPlanTotals{ProcessDefinitions: len(p.Items)}
	for _, item := range p.Items {
		totals.ActiveProcessInstances += item.ActiveProcessInstances()
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
	evidence := processDefinitionPlanTenantEvidence{}
	for _, item := range p.Items {
		evidence.Add("pd", item.Key, item.TenantId)
		evidence.MergeProcessInstanceEvidence(item.CancellationPlan.TenantEvidence)
	}
	return evidence.Snapshot()
}

// processDefinitionPlanTenantEvidence accumulates tenant observations across
// process-definition items and nested process-instance cancellation plans.
type processDefinitionPlanTenantEvidence struct {
	tenantSet          map[string]struct{}
	seenTargets        map[string]struct{}
	targets            []process.TenantEvidenceTarget
	targetCount        int
	unknownTargetCount int
}

// Add records one affected process-definition or process-instance target,
// counting missing tenant IDs as unknown evidence.
func (e *processDefinitionPlanTenantEvidence) Add(kind string, key string, tenantID string) {
	if key == "" {
		return
	}
	e.ensureMaps()
	seenKey := kind + ":" + key
	if _, ok := e.seenTargets[seenKey]; ok {
		return
	}
	e.seenTargets[seenKey] = struct{}{}
	e.targetCount++
	if tenantID == "" {
		e.unknownTargetCount++
	} else {
		e.tenantSet[tenantID] = struct{}{}
	}
	e.targets = append(e.targets, process.TenantEvidenceTarget{Key: key, TenantID: tenantID})
}

// MergeProcessInstanceEvidence folds nested cancellation evidence into the
// process-definition impact evidence without double-counting repeated targets.
func (e *processDefinitionPlanTenantEvidence) MergeProcessInstanceEvidence(nested process.TenantEvidence) {
	if len(nested.Targets) > 0 {
		for _, target := range nested.Targets {
			e.Add("pi", target.Key, target.TenantID)
		}
		return
	}
	e.ensureMaps()
	for _, tenantID := range nested.ResolvedTenantIDs {
		if tenantID != "" {
			e.tenantSet[tenantID] = struct{}{}
		}
	}
	e.unknownTargetCount += nested.UnknownTargetCount
	e.targetCount += nested.TargetCount
}

// Snapshot returns deterministic public tenant evidence for command context.
func (e *processDefinitionPlanTenantEvidence) Snapshot() process.TenantEvidence {
	e.ensureMaps()
	resolvedTenantIDs := make([]string, 0, len(e.tenantSet))
	for tenantID := range e.tenantSet {
		resolvedTenantIDs = append(resolvedTenantIDs, tenantID)
	}
	return process.TenantEvidence{
		ResolvedTenantIDs:  sortProcessDefinitionPlanTenantIDs(resolvedTenantIDs),
		UnknownTargetCount: e.unknownTargetCount,
		TargetCount:        e.targetCount,
		Targets:            append([]process.TenantEvidenceTarget(nil), e.targets...),
	}
}

// ensureMaps lazily initializes aggregation maps for zero-value use.
func (e *processDefinitionPlanTenantEvidence) ensureMaps() {
	if e.tenantSet == nil {
		e.tenantSet = make(map[string]struct{})
	}
	if e.seenTargets == nil {
		e.seenTargets = make(map[string]struct{})
	}
}

// sortProcessDefinitionPlanTenantIDs keeps command-rendered resource tenant
// evidence deterministic for previews and structured envelopes.
func sortProcessDefinitionPlanTenantIDs(ids []string) []string {
	slices.Sort(ids)
	return slices.Compact(ids)
}

type DeleteProcessDefinitionPlanTotals struct {
	ProcessDefinitions     int   `json:"processDefinitions"`
	ActiveProcessInstances int64 `json:"activeProcessInstances"`
	CancellationRoots      int   `json:"cancellationRoots"`
	CancellationAffected   int   `json:"cancellationAffected"`
	Warnings               int   `json:"warnings"`
}
