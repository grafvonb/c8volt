// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
)

func listIncidentsView(cmd *cobra.Command, resp incident.Incidents, messageLimit int, omitMessage bool) error {
	if flagGetIncidentPIKeysOnly {
		return renderIncidentProcessInstanceKeys(cmd, resp.Items)
	}
	mode := pickMode()
	switch mode {
	case RenderModeJSON:
		return renderJSONPayload(cmd, mode, resp)
	case RenderModeKeysOnly:
		for _, it := range resp.Items {
			renderOutputLine(cmd, "%s", it.IncidentKey)
		}
	default:
		for _, line := range formatIncidentListRowsWithTimezone(resp.Items, messageLimit, omitMessage, commandShowTimezoneOffset(cmd)) {
			renderOutputLine(cmd, "%s", line)
		}
		renderOutputLine(cmd, "found: %d", len(resp.Items))
	}
	return nil
}

func renderIncidentProcessInstanceKeys(cmd *cobra.Command, items []incident.ProcessInstanceIncidentDetail) error {
	for _, it := range items {
		if it.ProcessInstanceKey == "" {
			continue
		}
		renderOutputLine(cmd, "%s", it.ProcessInstanceKey)
	}
	return nil
}

func processDefinitionView(cmd *cobra.Command, item process.ProcessDefinition) error {
	return itemView(cmd, item, pickMode(), oneLinePD, func(it process.ProcessDefinition) string { return it.Key })
}

func listProcessDefinitionsView(cmd *cobra.Command, resp process.ProcessDefinitions) error {
	return listOrJSONFlat(cmd, resp, resp.Items, pickMode(), flatRowPD, func(it process.ProcessDefinition) string { return it.Key })
}

// processDefinitionWatchView keeps each refresh body identical to normal list output.
func processDefinitionWatchView(cmd *cobra.Command, snapshot process.ProcessDefinitionWatchSnapshot) error {
	return listProcessDefinitionsView(cmd, process.ProcessDefinitions{
		Total: snapshot.Total,
		Items: snapshot.Items,
	})
}

func formatProcessDefinitionFlatRows(items []process.ProcessDefinition) []string {
	rows := make([]flatRow, 0, len(items))
	for _, it := range items {
		rows = append(rows, flatRowPD(it))
	}
	return formatFlatRows(rows)
}

func oneLinePD(it process.ProcessDefinition) string {
	return compactFlatRow(flatRowPD(it))
}

// flatRowPD mirrors the process-definition human order while allowing statistics to remain an optional tail column.
func flatRowPD(it process.ProcessDefinition) flatRow {
	vTag := ""
	if it.ProcessVersionTag != "" {
		vTag = "/" + it.ProcessVersionTag
	}
	row := flatRow{it.Key, it.TenantId, it.BpmnProcessId, fmt.Sprintf("v%d%s", it.ProcessVersion, vTag)}
	if it.Statistics != nil {
		stats := it.Statistics
		incidentTag := ""
		if stats.IncidentCountSupported {
			incidentTag = fmt.Sprintf(" inc:%s", zeroAsMinus(stats.Incidents))
		}
		row = append(row, fmt.Sprintf("[ac:%s cp:%s cx:%s%s]",
			zeroAsMinus(stats.Active),
			zeroAsMinus(stats.Completed),
			zeroAsMinus(stats.Canceled),
			incidentTag,
		))
	}
	return row
}

func resourceView(cmd *cobra.Command, item resource.Resource) error {
	return resourceItemView(cmd, item, pickMode())
}

func resourceItemView(cmd *cobra.Command, item resource.Resource, mode RenderMode) error {
	return itemView(cmd, item, mode, oneLineResource, func(it resource.Resource) string { return it.ID })
}

func oneLineResource(it resource.Resource) string {
	return compactFlatRow(flatRowResource(it))
}

// flatRowResource keeps resource names in the same human position while aligning IDs and keys in list output.
func flatRowResource(it resource.Resource) flatRow {
	vTag := ""
	if it.VersionTag != "" {
		vTag = "/" + it.VersionTag
	}
	return flatRow{it.ID, "k:" + it.Key, it.TenantId, it.Name, fmt.Sprintf("v%d%s", it.Version, vTag)}
}

// listTenantsView renders tenant discovery output through the shared list, keys-only, and JSON modes.
func listTenantsView(cmd *cobra.Command, resp tenant.Tenants) error {
	return listOrJSONFlat(cmd, resp, resp.Items, pickMode(), flatRowTenant, func(it tenant.Tenant) string { return it.TenantId })
}

// tenantView renders a single tenant through the same mode contract as other get commands.
func tenantView(cmd *cobra.Command, item tenant.Tenant) error {
	return itemView(cmd, item, pickMode(), oneLineTenant, func(it tenant.Tenant) string { return it.TenantId })
}

// oneLineTenant formats compact tenant rows.
func oneLineTenant(it tenant.Tenant) string {
	return compactFlatRow(flatRowTenant(it))
}

// flatRowTenant omits the description column when absent so sparse tenant lists stay compact.
func flatRowTenant(it tenant.Tenant) flatRow {
	if it.Description == "" {
		return flatRow{it.TenantId, it.Name}
	}
	return flatRow{it.TenantId, it.Name, it.Description}
}
