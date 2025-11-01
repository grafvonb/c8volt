package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/spf13/cobra"
)

type RenderMode int

const (
	ModeJSON RenderMode = iota
	ModeOneLine
	ModeKeysOnly
)

func (m RenderMode) String() string {
	switch m {
	case ModeJSON:
		return "json"
	case ModeOneLine:
		return "one-line"
	case ModeKeysOnly:
		return "keys-only"
	default:
		return fmt.Sprintf("unknown(%d)", m)
	}
}

func itemView[Item any](cmd *cobra.Command, item Item, mode RenderMode, oneLine func(Item) string, keyOf func(Item) string) error {
	switch mode {
	case ModeJSON:
		cmd.Println(toolx.ToJSONString(item))
	case ModeKeysOnly:
		cmd.Println(keyOf(item))
	default:
		cmd.Println(strings.TrimSpace(oneLine(item)))
	}
	return nil
}

func listOrJSON[Resp any, Item any](
	cmd *cobra.Command,
	resp Resp,
	items []Item,
	mode RenderMode,
	oneLine func(Item) string,
	keyOf func(Item) string,
) error {
	if len(items) == 0 {
		cmd.Println("found: 0")
		if mode == ModeJSON {
			cmd.Println(toolx.ToJSONString(resp))
		}
		return nil
	}
	cmd.Println("found:", len(items))
	switch mode {
	case ModeJSON:
		cmd.Println(toolx.ToJSONString(resp))
	case ModeKeysOnly:
		for _, it := range items {
			cmd.Println(keyOf(it))
		}
	default: // ModeOneLine
		for _, it := range items {
			cmd.Println(strings.TrimSpace(oneLine(it)))
		}
	}
	return nil
}

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
		"%-16s %s %s v%d%s %s s:%s %s%s i:%t",
		it.Key, it.TenantId, it.BpmnProcessId, it.ProcessVersion, vTag,
		it.State, it.StartDate, eTag, pTag, it.Incident,
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
	if it.VersionTag != "" {
		vTag = "/" + it.VersionTag
	}
	return fmt.Sprintf("%-16s %s %s v%d%s",
		it.Key, it.TenantId, it.BpmnProcessId, it.Version, vTag,
	)
}

func pickMode() RenderMode {
	switch {
	case flagViewAsJson:
		return ModeJSON
	case flagViewKeysOnly:
		return ModeKeysOnly
	default:
		return ModeOneLine
	}
}
