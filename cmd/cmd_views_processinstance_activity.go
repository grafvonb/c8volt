// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

type processInstanceActivityItem struct {
	Item      process.ProcessInstance                 `json:"item"`
	Variables []process.ProcessInstanceVariable       `json:"variables,omitempty"`
	Incidents []process.ProcessInstanceIncidentDetail `json:"incidents,omitempty"`
}

type processInstanceActivityInstances struct {
	Total int32                         `json:"total"`
	Items []processInstanceActivityItem `json:"items"`
}

type processInstanceActivityInstancesJSONWithMeta struct {
	Total int32                         `json:"total,omitempty"`
	Items []processInstanceActivityItem `json:"items,omitempty"`
	Meta  processInstanceAgeMeta        `json:"meta"`
}

func processInstanceActivityInstancesView(cmd *cobra.Command, resp processInstanceActivityInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, processInstanceActivityInstancesWithAgeMeta(resp))
	}
	needsIndirectIncidentWarning := renderProcessInstanceActivityRows(cmd, resp.Items)
	if needsIndirectIncidentWarning {
		renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
	}
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
}

func processInstanceActivityInstancesWithAgeMeta(resp processInstanceActivityInstances) processInstanceActivityInstancesJSONWithMeta {
	meta := processInstanceAgeMeta{WithAge: true, AgeDaysBy: map[string]int{}}
	for _, it := range resp.Items {
		if age, ok := processInstanceAgeDays(it.Item.StartDate); ok {
			meta.AgeDaysBy[it.Item.Key] = age
		}
	}
	if len(meta.AgeDaysBy) == 0 {
		meta.AgeDaysBy = nil
	}
	return processInstanceActivityInstancesJSONWithMeta{
		Total: resp.Total,
		Items: resp.Items,
		Meta:  meta,
	}
}

func renderProcessInstanceActivityRows(cmd *cobra.Command, items []processInstanceActivityItem) bool {
	rows := make([]flatRow, 0, len(items))
	for _, it := range items {
		rows = append(rows, flatRowPI(it.Item))
	}
	lines := formatFlatRows(rows)
	needsIndirectIncidentWarning := false
	for i, it := range items {
		renderOutputLine(cmd, "%s", lines[i])
		detailLines, needsWarning := formatProcessInstanceActivityLines("", it.Variables, it.Incidents, it.Item.Incident, 0)
		for _, line := range detailLines {
			renderOutputLine(cmd, "%s", line)
		}
		needsIndirectIncidentWarning = needsIndirectIncidentWarning || needsWarning
	}
	return needsIndirectIncidentWarning
}

func formatProcessInstanceActivityLines(prefix string, variables []process.ProcessInstanceVariable, incidents []process.ProcessInstanceIncidentDetail, hasIncidentMarker bool, followingChildren int) ([]string, bool) {
	hasVars := len(variables) > 0
	hasIncidents := len(incidents) > 0 || hasIncidentMarker
	sectionCount := 0
	if hasVars {
		sectionCount++
	}
	if hasIncidents {
		sectionCount++
	}
	if sectionCount == 0 {
		return nil, false
	}

	totalBranches := sectionCount + followingChildren
	sectionIndex := 0
	lines := make([]string, 0, sectionCount+len(variables)+len(incidents)+1)
	if hasVars {
		branch := incidentTreeBranch(sectionIndex, totalBranches)
		childPrefix := treeChildPrefix(prefix, sectionIndex, totalBranches)
		lines = append(lines, prefix+branch+"vars:")
		for i, variable := range variables {
			lines = append(lines, childPrefix+incidentTreeBranch(i, len(variables))+processInstanceVariableHumanLine(variable))
		}
		sectionIndex++
	}

	needsIndirectIncidentWarning := false
	if hasIncidents {
		branch := incidentTreeBranch(sectionIndex, totalBranches)
		childPrefix := treeChildPrefix(prefix, sectionIndex, totalBranches)
		lines = append(lines, prefix+branch+"incidents:")
		if len(incidents) > 0 {
			for i, incident := range incidents {
				lines = append(lines, childPrefix+incidentTreeBranch(i, len(incidents))+incidentHumanLine(incident))
			}
		} else {
			lines = append(lines, childPrefix+"└─ "+indirectProcessTreeIncidentNote)
			needsIndirectIncidentWarning = true
		}
	}
	return lines, needsIndirectIncidentWarning
}

func treeChildPrefix(prefix string, branchIndex, totalBranches int) string {
	if branchIndex == totalBranches-1 {
		return prefix + "   "
	}
	return prefix + "│  "
}

func writeProcessInstanceActivityLines(out *strings.Builder, prefix string, variables []process.ProcessInstanceVariable, incidents []process.ProcessInstanceIncidentDetail, hasIncidentMarker bool, followingChildren int) bool {
	lines, needsWarning := formatProcessInstanceActivityLines(prefix, variables, incidents, hasIncidentMarker, followingChildren)
	for _, line := range lines {
		out.WriteByte('\n')
		out.WriteString(line)
	}
	return needsWarning
}

func activityFromIncidentEnriched(resp process.IncidentEnrichedProcessInstances) processInstanceActivityInstances {
	items := make([]processInstanceActivityItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, processInstanceActivityItem{
			Item:      it.Item,
			Incidents: it.Incidents,
		})
	}
	return processInstanceActivityInstances{Total: resp.Total, Items: items}
}

func activityFromVariableEnriched(resp process.VariableEnrichedProcessInstances) processInstanceActivityInstances {
	items := make([]processInstanceActivityItem, 0, len(resp.Items))
	for _, it := range resp.Items {
		items = append(items, processInstanceActivityItem{
			Item:      it.Item,
			Variables: it.Variables,
		})
	}
	return processInstanceActivityInstances{Total: resp.Total, Items: items}
}

func mergeIncidentAndVariableActivity(incidents process.IncidentEnrichedProcessInstances, variables process.VariableEnrichedProcessInstances) processInstanceActivityInstances {
	varsByKey := make(map[string][]process.ProcessInstanceVariable, len(variables.Items))
	for _, it := range variables.Items {
		varsByKey[it.Item.Key] = it.Variables
	}

	items := make([]processInstanceActivityItem, 0, len(incidents.Items))
	for _, it := range incidents.Items {
		items = append(items, processInstanceActivityItem{
			Item:      it.Item,
			Variables: varsByKey[it.Item.Key],
			Incidents: it.Incidents,
		})
	}
	return processInstanceActivityInstances{
		Total: int32(len(items)),
		Items: items,
	}
}

func processInstancesFromTraversal(result process.TraversalResult) process.ProcessInstances {
	items := make([]process.ProcessInstance, 0, len(result.Keys))
	for _, key := range result.Keys {
		if item, ok := result.Chain[key]; ok {
			items = append(items, item)
		}
	}
	return process.ProcessInstances{
		Total: int32(len(items)),
		Items: items,
	}
}

func activityItemsFromTraversal(result process.TraversalResult, incidents process.IncidentEnrichedTraversalResult, variables process.VariableEnrichedProcessInstances) []processInstanceActivityItem {
	incidentsByKey := make(map[string][]process.ProcessInstanceIncidentDetail, len(incidents.Items))
	for _, item := range incidents.Items {
		incidentsByKey[item.Item.Key] = item.Incidents
	}
	varsByKey := make(map[string][]process.ProcessInstanceVariable, len(variables.Items))
	for _, item := range variables.Items {
		varsByKey[item.Item.Key] = item.Variables
	}

	items := make([]processInstanceActivityItem, 0, len(result.Keys))
	for _, key := range result.Keys {
		item, ok := result.Chain[key]
		if !ok {
			continue
		}
		items = append(items, processInstanceActivityItem{
			Item:      item,
			Variables: varsByKey[key],
			Incidents: incidentsByKey[key],
		})
	}
	return items
}
