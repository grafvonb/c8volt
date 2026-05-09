// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

const (
	indirectProcessTreeIncidentNote    = "process instance is marked as having incidents, but no direct incidents were found; inspect the process tree for child incident details"
	indirectProcessTreeIncidentWarning = "one or more incident markers may refer to incidents in the process-instance tree; inspect with walk pi --key <key> --with-incidents"
)

// incidentEnrichedProcessInstancesView renders direct process-instance incident enrichment.
func incidentEnrichedProcessInstancesView(cmd *cobra.Command, resp process.IncidentEnrichedProcessInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, incidentEnrichedProcessInstancesWithAgeMeta(resp))
	}
	needsIndirectIncidentWarning := renderProcessInstanceActivityRows(cmd, activityFromIncidentEnriched(resp).Items)
	if needsIndirectIncidentWarning {
		renderHumanWarningLine(cmd, indirectProcessTreeIncidentWarning)
	}
	renderOutputLine(cmd, "found: %d", len(resp.Items))
	return nil
}

func renderIncidentEnrichedProcessInstanceRows(cmd *cobra.Command, resp process.IncidentEnrichedProcessInstances) (bool, error) {
	return renderProcessInstanceActivityRows(cmd, activityFromIncidentEnriched(resp).Items), nil
}

func incidentTreeBranch(index, total int) string {
	if index == total-1 {
		return "└─ "
	}
	return "├─ "
}

type incidentEnrichedProcessInstancesJSONWithMeta struct {
	Total int32                                     `json:"total,omitempty"`
	Items []process.IncidentEnrichedProcessInstance `json:"items,omitempty"`
	Meta  processInstanceAgeMeta                    `json:"meta"`
}

// incidentEnrichedProcessInstancesWithAgeMeta keeps incident JSON compatible with default process-instance age metadata.
func incidentEnrichedProcessInstancesWithAgeMeta(resp process.IncidentEnrichedProcessInstances) incidentEnrichedProcessInstancesJSONWithMeta {
	meta := processInstanceAgeMeta{WithAge: true, AgeDaysBy: map[string]int{}}
	for _, it := range resp.Items {
		if age, ok := processInstanceAgeDays(it.Item.StartDate); ok {
			meta.AgeDaysBy[it.Item.Key] = age
		}
	}
	if len(meta.AgeDaysBy) == 0 {
		meta.AgeDaysBy = nil
	}
	return incidentEnrichedProcessInstancesJSONWithMeta{
		Total: resp.Total,
		Items: resp.Items,
		Meta:  meta,
	}
}

// processInstanceHasIndirectIncidentMarker detects tree-propagated incident markers without direct incident details.
func processInstanceHasIndirectIncidentMarker(item process.IncidentEnrichedProcessInstance) bool {
	return item.Item.Incident && len(item.Incidents) == 0
}

// incidentHumanLine formats a human-readable incident detail line with compact attributes.
func incidentHumanLine(incident process.ProcessInstanceIncidentDetail) string {
	return incidentHumanLineWithMessageLimit(incident, flagGetPIIncidentMessageLimit)
}

// incidentHumanLineWithMessageLimit formats shared incident rows for process-instance and plain incident output.
func incidentHumanLineWithMessageLimit(incident process.ProcessInstanceIncidentDetail, messageLimit int) string {
	key := incident.IncidentKey
	if key == "" {
		key = "unknown"
	}
	message := truncateIncidentHumanMessage(incident.ErrorMessage, messageLimit)
	fields := incidentHumanFields(incident, key)
	return fmt.Sprintf("%s message=%s", fields, message)
}

func incidentHumanFields(incident process.ProcessInstanceIncidentDetail, key string) string {
	fields := make([]string, 0, 5)
	fields = append(fields, "key="+key)
	if incident.CreationTime != "" {
		fields = append(fields, "creationTime="+incident.CreationTime)
	}
	if incident.FlowNodeId != "" {
		fields = append(fields, "flowNodeId="+incident.FlowNodeId)
	}
	if incident.FlowNodeInstanceKey != "" {
		fields = append(fields, "flowNodeInstanceKey="+incident.FlowNodeInstanceKey)
	}
	if incident.State != "" {
		fields = append(fields, "state="+incident.State)
	}
	if incident.ErrorType != "" {
		fields = append(fields, "errorType="+incident.ErrorType)
	}
	jobKey := incident.JobKey
	if jobKey == "" {
		jobKey = "n/a"
	}
	fields = append(fields, "jobKey="+jobKey)
	return strings.Join(fields, " ")
}

func incidentAgeTag(creationTime string) string {
	age, ok := processInstanceAgeDays(creationTime)
	if !ok {
		return ""
	}
	if age == 0 {
		return "(today)"
	}
	return fmt.Sprintf("(%d days ago)", age)
}

func incidentListHumanLineWithMessageLimit(incident process.ProcessInstanceIncidentDetail, messageLimit int) string {
	key := incident.IncidentKey
	if key == "" {
		key = "unknown"
	}
	message := truncateIncidentHumanMessage(incident.ErrorMessage, messageLimit)
	fields := incidentListHumanFields(incident, key)
	return fmt.Sprintf("%s message=%s", fields, message)
}

func incidentListHumanFields(incident process.ProcessInstanceIncidentDetail, key string) string {
	fields := make([]string, 0, 13)
	fields = append(fields, "key="+key)
	if incident.TenantId != "" {
		fields = append(fields, "tenant="+incident.TenantId)
	}
	if incident.CreationTime != "" {
		fields = append(fields, "creationTime="+incident.CreationTime)
	}
	if incident.ProcessInstanceKey != "" {
		fields = append(fields, "processInstanceKey="+incident.ProcessInstanceKey)
	}
	if incident.RootProcessInstanceKey != "" {
		fields = append(fields, "rootProcessInstanceKey="+incident.RootProcessInstanceKey)
	}
	if incident.ProcessDefinitionKey != "" {
		fields = append(fields, "processDefinitionKey="+incident.ProcessDefinitionKey)
	}
	if incident.ProcessDefinitionId != "" {
		fields = append(fields, "processDefinitionId="+incident.ProcessDefinitionId)
	}
	if incident.FlowNodeId != "" {
		fields = append(fields, "flowNodeId="+incident.FlowNodeId)
	}
	if incident.FlowNodeInstanceKey != "" {
		fields = append(fields, "flowNodeInstanceKey="+incident.FlowNodeInstanceKey)
	}
	if incident.State != "" {
		fields = append(fields, "state="+incident.State)
	}
	if incident.ErrorType != "" {
		fields = append(fields, "errorType="+incident.ErrorType)
	}
	jobKey := incident.JobKey
	if jobKey == "" {
		jobKey = "n/a"
	}
	fields = append(fields, "jobKey="+jobKey)
	if age := incidentAgeTag(incident.CreationTime); age != "" {
		fields = append(fields, age)
	}
	return strings.Join(fields, " ")
}

// truncateIncidentHumanMessage applies the human-only incident message display limit.
func truncateIncidentHumanMessage(message string, limit int) string {
	return truncateHumanMessage(message, limit)
}

func truncateHumanMessage(message string, limit int) string {
	if limit <= 0 {
		return message
	}
	runes := []rune(message)
	if len(runes) <= limit {
		return message
	}
	return string(runes[:limit]) + "..."
}
