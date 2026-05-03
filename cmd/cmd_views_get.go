// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/c8volt/tenant"
	"github.com/spf13/cobra"
)

//nolint:unused
func processInstanceView(cmd *cobra.Command, item process.ProcessInstance) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, processInstanceWithAgeMeta(item))
	}
	return itemView(cmd, item, pickMode(), oneLinePI, func(it process.ProcessInstance) string { return it.Key })
}

func processInstanceTotalView(cmd *cobra.Command, total int64) error {
	_, err := fmt.Fprintln(cmd.OutOrStdout(), total)
	return err
}

func listProcessInstancesView(cmd *cobra.Command, resp process.ProcessInstances) error {
	if pickMode() == RenderModeJSON {
		return renderJSONPayload(cmd, RenderModeJSON, processInstancesWithAgeMeta(resp))
	}
	return listOrJSONFlat(cmd, resp, resp.Items, pickMode(), flatRowPI, func(it process.ProcessInstance) string { return it.Key })
}

// renderProcessInstanceFlatRows shares aligned human output between collected lists and incremental search pages.
func renderProcessInstanceFlatRows(cmd *cobra.Command, items []process.ProcessInstance) error {
	for _, line := range formatProcessInstanceFlatRows(items) {
		renderOutputLine(cmd, "%s", line)
	}
	return nil
}

// formatProcessInstanceFlatRows keeps process-instance page rendering list-aware without changing machine modes.
func formatProcessInstanceFlatRows(items []process.ProcessInstance) []string {
	rows := make([]flatRow, 0, len(items))
	for _, it := range items {
		rows = append(rows, flatRowPI(it))
	}
	return formatFlatRows(rows)
}

func oneLinePI(it process.ProcessInstance) string {
	return compactFlatRow(flatRowPI(it))
}

// flatRowPI defines the process-instance scan order, keeping BPMN IDs before operational state columns.
func flatRowPI(it process.ProcessInstance) flatRow {
	pTag := " p:<root>"
	if it.ParentKey != "" {
		pTag = " p:" + it.ParentKey
	}
	eTag := ""
	if it.EndDate != "" {
		eTag = " e:" + processInstanceTimestampMillis(it.EndDate)
	}
	vTag := ""
	if it.ProcessVersionTag != "" {
		vTag = "/" + it.ProcessVersionTag
	}
	ageTag := ""
	if age, ok := processInstanceAgeDays(it.StartDate); ok {
		if age == 0 {
			ageTag = " (today)"
		} else {
			ageTag = fmt.Sprintf(" (%d days ago)", age)
		}
	}
	incidentTag := ""
	if it.Incident {
		incidentTag = " inc!"
	}
	return flatRow{
		it.Key,
		it.TenantId,
		it.BpmnProcessId,
		fmt.Sprintf("v%d%s", it.ProcessVersion, vTag),
		string(it.State),
		"s:" + processInstanceTimestampMillis(it.StartDate),
		strings.TrimSpace(eTag),
		strings.TrimSpace(pTag),
		strings.TrimSpace(incidentTag),
		strings.TrimSpace(ageTag),
	}
}

func processInstanceTimestampMillis(value string) string {
	if value == "" {
		return ""
	}
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return value
	}
	return t.Format("2006-01-02T15:04:05.000Z07:00")
}

type processInstanceAgeMeta struct {
	WithAge   bool           `json:"withAge"`
	AgeDays   int            `json:"ageDays,omitempty"`
	AgeDaysBy map[string]int `json:"ageDaysByKey,omitempty"`
}

type processInstanceJSONWithMeta struct {
	Item process.ProcessInstance `json:"item"`
	Meta processInstanceAgeMeta  `json:"meta"`
}

type processInstancesJSONWithMeta struct {
	Total int32                     `json:"total,omitempty"`
	Items []process.ProcessInstance `json:"items,omitempty"`
	Meta  processInstanceAgeMeta    `json:"meta"`
}

func processInstanceWithAgeMeta(item process.ProcessInstance) processInstanceJSONWithMeta {
	meta := processInstanceAgeMeta{WithAge: true}
	if age, ok := processInstanceAgeDays(item.StartDate); ok {
		meta.AgeDays = age
	}
	return processInstanceJSONWithMeta{Item: item, Meta: meta}
}

func processInstancesWithAgeMeta(resp process.ProcessInstances) processInstancesJSONWithMeta {
	meta := processInstanceAgeMeta{WithAge: true, AgeDaysBy: map[string]int{}}
	for _, it := range resp.Items {
		if age, ok := processInstanceAgeDays(it.StartDate); ok {
			meta.AgeDaysBy[it.Key] = age
		}
	}
	if len(meta.AgeDaysBy) == 0 {
		meta.AgeDaysBy = nil
	}
	return processInstancesJSONWithMeta{
		Total: resp.Total,
		Items: resp.Items,
		Meta:  meta,
	}
}

func processInstanceAgeDays(startDate string) (int, bool) {
	if startDate == "" {
		return 0, false
	}
	start, err := time.Parse(time.RFC3339Nano, startDate)
	if err != nil {
		return 0, false
	}
	now := relativeDayNow().UTC()
	startDay := time.Date(start.UTC().Year(), start.UTC().Month(), start.UTC().Day(), 0, 0, 0, 0, time.UTC)
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if nowDay.Before(startDay) {
		return 0, false
	}
	days := int(nowDay.Sub(startDay).Hours() / 24)
	return days, true
}

func processDefinitionView(cmd *cobra.Command, item process.ProcessDefinition) error {
	return itemView(cmd, item, pickMode(), oneLinePD, func(it process.ProcessDefinition) string { return it.Key })
}

func listProcessDefinitionsView(cmd *cobra.Command, resp process.ProcessDefinitions) error {
	return listOrJSONFlat(cmd, resp, resp.Items, pickMode(), flatRowPD, func(it process.ProcessDefinition) string { return it.Key })
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

// oneLineTenant formats tenant rows for compact human output.
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

func zeroAsMinus(v int64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", v)
}
