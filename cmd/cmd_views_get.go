package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

func processInstanceView(cmd *cobra.Command, item process.ProcessInstance) error {
	return itemView(cmd, item, pickMode(), oneLinePI, func(it process.ProcessInstance) string { return it.Key })
}

func listProcessInstancesView(cmd *cobra.Command, resp process.ProcessInstances) error {
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
	return fmt.Sprintf(
		"%-16s %s %s v%d%s %-10s s:%-20s%s%s i:%t",
		it.Key, it.TenantId, it.BpmnProcessId, it.ProcessVersion, vTag, it.State, it.StartDate, eTag, pTag, it.Incident,
	)
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
		return fmt.Sprintf("%s [ac:%s cp:%s cx:%s in:%s]",
			core,
			zeroAsMinus(stats.Active),
			zeroAsMinus(stats.Completed),
			zeroAsMinus(stats.Canceled),
			zeroAsMinus(stats.Incidents),
		)
	}
	return core
}

func zeroAsMinus(v int64) string {
	if v == 0 {
		return "-"
	}
	return fmt.Sprintf("%d", v)
}
