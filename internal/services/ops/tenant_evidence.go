// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/common"
	pitraversal "github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/typex"
)

// opsTenantEvidenceFromTraversalResults records affected process-instance
// tenant metadata from traversal chains that were already loaded for planning.
func opsTenantEvidenceFromTraversalResults(keys typex.Keys, results ...[]pitraversal.Result) d.TenantEvidence {
	tenantByKey := make(map[string]string)
	for _, group := range results {
		for _, result := range group {
			for key, pi := range result.Chain {
				if key == "" {
					key = pi.Key
				}
				if key == "" {
					continue
				}
				if existing, ok := tenantByKey[key]; ok && existing != "" {
					continue
				}
				tenantByKey[key] = pi.TenantId
			}
		}
	}
	targets := make([]d.TenantEvidenceTarget, 0, len(keys.Unique()))
	for _, key := range keys.Unique() {
		targets = append(targets, d.TenantEvidenceTarget{Key: key, TenantID: tenantByKey[key]})
	}
	return opsTenantEvidenceFromTargets(targets)
}

// opsTenantEvidenceFromProcessDefinitionPlan records aggregate delete-plan
// tenant evidence produced by the process-definition service preview.
func opsTenantEvidenceFromProcessDefinitionPlan(plan d.DeleteProcessDefinitionPlan) d.TenantEvidence {
	if plan.TenantEvidence.TargetCount > 0 || len(plan.TenantEvidence.Targets) > 0 || len(plan.TenantEvidence.ResolvedTenantIDs) > 0 || plan.TenantEvidence.UnknownTargetCount > 0 {
		return opsMergeTenantEvidence(plan.TenantEvidence)
	}
	return opsTenantEvidenceFromProcessDefinitionPlanItems(plan.Items)
}

// opsTenantEvidenceFromProcessDefinitionPlanItems records evidence from
// process-definition items and nested process-instance cancellation plans.
func opsTenantEvidenceFromProcessDefinitionPlanItems(items []d.DeleteProcessDefinitionPlanItem) d.TenantEvidence {
	var inputs []d.TenantEvidence
	targets := make([]d.TenantEvidenceTarget, 0, len(items))
	for _, item := range items {
		if item.Key != "" {
			targets = append(targets, d.TenantEvidenceTarget{Key: item.Key, TenantID: item.TenantId})
		}
		inputs = append(inputs, item.CancellationPlan.TenantEvidence)
	}
	inputs = append(inputs, opsTenantEvidenceFromTargets(targets))
	return opsMergeTenantEvidence(inputs...)
}

// opsTenantEvidenceFromProcessInstances records one target observation per
// process-instance key, treating missing tenant IDs as unknown.
func opsTenantEvidenceFromProcessInstances(items []d.ProcessInstance) d.TenantEvidence {
	targets := make([]d.TenantEvidenceTarget, 0, len(items))
	for _, item := range items {
		targets = append(targets, d.TenantEvidenceTarget{Key: item.Key, TenantID: item.TenantId})
	}
	return opsTenantEvidenceFromTargets(targets)
}

// opsTenantEvidenceFromIncidents records one target observation per incident
// key from already-discovered incident payloads.
func opsTenantEvidenceFromIncidents(items []d.ProcessInstanceIncidentDetail) d.TenantEvidence {
	targets := make([]d.TenantEvidenceTarget, 0, len(items))
	for _, item := range items {
		targets = append(targets, d.TenantEvidenceTarget{Key: item.IncidentKey, TenantID: item.TenantId})
	}
	return opsTenantEvidenceFromTargets(targets)
}

// opsTenantEvidenceFromProcessInstanceKeysAndDetails records PI-targeted repair
// evidence for the frozen key set and lets already-loaded PI or incident data
// fill missing tenant metadata for the same process-instance key.
func opsTenantEvidenceFromProcessInstanceKeysAndDetails(keys typex.Keys, processInstances []d.ProcessInstance, incidents []d.ProcessInstanceIncidentDetail) d.TenantEvidence {
	tenantByKey := make(map[string]string, len(processInstances))
	for _, item := range processInstances {
		if item.Key == "" {
			continue
		}
		tenantByKey[item.Key] = item.TenantId
	}
	for _, incident := range incidents {
		if incident.ProcessInstanceKey == "" || incident.TenantId == "" {
			continue
		}
		if existing := tenantByKey[incident.ProcessInstanceKey]; existing != "" {
			continue
		}
		tenantByKey[incident.ProcessInstanceKey] = incident.TenantId
	}
	targets := make([]d.TenantEvidenceTarget, 0, len(keys.Unique()))
	for _, key := range keys.Unique() {
		targets = append(targets, d.TenantEvidenceTarget{Key: key, TenantID: tenantByKey[key]})
	}
	return opsTenantEvidenceFromTargets(targets)
}

// opsTenantEvidenceFromDeployment records the deployed process definition as
// the smoke-test creation evidence when the deployment response exposes it.
func opsTenantEvidenceFromDeployment(deployment d.SmokeTestDeploymentResult) d.TenantEvidence {
	key := deployment.ProcessDefinitionKey
	if key == "" {
		key = deployment.BpmnProcessID
	}
	return opsTenantEvidenceFromTargets([]d.TenantEvidenceTarget{{Key: key, TenantID: deployment.TenantID}})
}

// opsTenantEvidenceFromCreations records created process-instance tenant
// evidence from create responses already returned by the run step.
func opsTenantEvidenceFromCreations(items []d.ProcessInstanceCreation) d.TenantEvidence {
	targets := make([]d.TenantEvidenceTarget, 0, len(items))
	for _, item := range items {
		targets = append(targets, d.TenantEvidenceTarget{Key: item.Key, TenantID: item.TenantId})
	}
	return opsTenantEvidenceFromTargets(targets)
}

// opsTenantEvidenceFromTargets normalizes target observations into the shared
// domain evidence shape while preserving first-observation target order.
func opsTenantEvidenceFromTargets(targets []d.TenantEvidenceTarget) d.TenantEvidence {
	return opsMergeTenantEvidence(d.TenantEvidence{Targets: targets})
}

// opsMergeTenantEvidence combines evidence snapshots by target key so reports
// do not double-count resources observed in multiple frozen workflow steps.
func opsMergeTenantEvidence(items ...d.TenantEvidence) d.TenantEvidence {
	acc := common.NewTenantEvidenceAccumulator()
	targets := make([]d.TenantEvidenceTarget, 0)
	seen := make(map[string]struct{})
	for _, item := range items {
		if len(item.Targets) == 0 {
			acc.AddAggregate(item.ResolvedTenantIDs, item.TargetCount, item.UnknownTargetCount)
			continue
		}
		for _, target := range item.Targets {
			if target.Key == "" {
				continue
			}
			acc.Add(target.Key, target.TenantID)
			if _, ok := seen[target.Key]; ok {
				continue
			}
			seen[target.Key] = struct{}{}
			targets = append(targets, d.TenantEvidenceTarget{Key: target.Key, TenantID: target.TenantID})
		}
	}
	snapshot := acc.Snapshot()
	return d.TenantEvidence{
		ResolvedTenantIDs:  append([]string(nil), snapshot.ResolvedTenantIDs...),
		UnknownTargetCount: snapshot.UnknownTargetCount,
		TargetCount:        snapshot.TargetCount,
		Targets:            targets,
	}
}
