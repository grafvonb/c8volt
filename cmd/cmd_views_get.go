package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
)

//nolint:unused
func processInstanceView(cmd *cobra.Command, item process.ProcessInstance) error {
	if pickMode() == RenderModeJSON && flagGetPIWithAge {
		return renderJSONPayload(cmd, RenderModeJSON, processInstanceWithAgeMeta(item))
	}
	return itemView(cmd, item, pickMode(), oneLinePI, func(it process.ProcessInstance) string { return it.Key })
}

func listProcessInstancesView(cmd *cobra.Command, resp process.ProcessInstances) error {
	if pickMode() == RenderModeJSON && flagGetPIWithAge {
		return renderJSONPayload(cmd, RenderModeJSON, processInstancesWithAgeMeta(resp))
	}
	return listOrJSON(cmd, resp, resp.Items, pickMode(), oneLinePI, func(it process.ProcessInstance) string { return it.Key })
}

func oneLinePI(it process.ProcessInstance) string {
	pTag := " p:<root>"
	if it.ParentKey != "" {
		pTag = " p:" + it.ParentKey
	}
	eTag := ""
	if it.EndDate != "" {
		eTag = " e:" + it.EndDate
	}
	vTag := ""
	if it.ProcessVersionTag != "" {
		vTag = "/" + it.ProcessVersionTag
	}
	ageTag := ""
	if flagGetPIWithAge {
		if age, ok := processInstanceAgeDays(it.StartDate); ok {
			if age == 0 {
				ageTag = " (today)"
			} else {
				ageTag = fmt.Sprintf(" (%d days ago)", age)
			}
		}
	}
	return fmt.Sprintf(
		"%-16s %s %s v%d%s %-10s s:%-20s%s%s i:%t%s",
		it.Key, it.TenantId, it.BpmnProcessId, it.ProcessVersion, vTag, it.State, it.StartDate, eTag, pTag, it.Incident, ageTag,
	)
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
	return listOrJSON(cmd, resp, resp.Items, pickMode(), oneLinePD, func(it process.ProcessDefinition) string { return it.Key })
}

func oneLinePD(it process.ProcessDefinition) string {
	vTag := ""
	if it.ProcessVersionTag != "" {
		vTag = "/" + it.ProcessVersionTag
	}
	core := fmt.Sprintf("%-16s %s %s v%d%s",
		it.Key, it.TenantId, it.BpmnProcessId, it.ProcessVersion, vTag,
	)
	if it.Statistics != nil {
		stats := it.Statistics
		incidentTag := ""
		if stats.IncidentCountSupported {
			incidentTag = fmt.Sprintf(" in:%d", stats.Incidents)
		}
		return fmt.Sprintf("%s [ac:%s cp:%s cx:%s%s]",
			core,
			zeroAsMinus(stats.Active),
			zeroAsMinus(stats.Completed),
			zeroAsMinus(stats.Canceled),
			incidentTag,
		)
	}
	return core
}

func resourceView(cmd *cobra.Command, item resource.Resource) error {
	return resourceItemView(cmd, item, pickMode())
}

func resourceItemView(cmd *cobra.Command, item resource.Resource, mode RenderMode) error {
	return itemView(cmd, item, mode, oneLineResource, func(it resource.Resource) string { return it.ID })
}

func oneLineResource(it resource.Resource) string {
	vTag := ""
	if it.VersionTag != "" {
		vTag = "/" + it.VersionTag
	}
	return fmt.Sprintf("%-24s k:%-16s %s %s v%d%s",
		it.ID, it.Key, it.TenantId, it.Name, it.Version, vTag,
	)
}

func zeroAsMinus(v int64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", v)
}
