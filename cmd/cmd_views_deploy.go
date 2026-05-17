// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"slices"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
)

//nolint:unused
func processDefinitionDeploymentView(cmd *cobra.Command, item resource.ProcessDefinitionDeployment) error {
	return itemView(cmd, item, pickMode(), oneLinePDDeploy, func(it resource.ProcessDefinitionDeployment) string { return it.DefinitionKey })
}

func listProcessDefinitionDeploymentsView(cmd *cobra.Command, resp []resource.ProcessDefinitionDeployment) error {
	mode := pickMode()
	switch mode {
	case RenderModeJSON:
		return renderJSONPayload(cmd, mode, resp)
	case RenderModeKeysOnly:
		for _, it := range resp {
			renderOutputLine(cmd, "%s", it.DefinitionKey)
		}
	default:
		return renderProcessDefinitionDeploymentLogs(cmd, resp)
	}
	return nil
}

func oneLinePDDeploy(it resource.ProcessDefinitionDeployment) string {
	return compactFlatRow(flatRowPDDeploy(it))
}

// flatRowPDDeploy keeps deployment progress on the same pd-first grammar as other mutation logs.
func flatRowPDDeploy(it resource.ProcessDefinitionDeployment) flatRow {
	state := "deployed"
	if flagNoWait {
		state = "submitted"
	}
	return flatRow{
		"pd",
		it.DefinitionKey,
		it.DefinitionId,
		fmt.Sprintf("v%d", it.DefinitionVersion),
		fmt.Sprintf("%s;", it.TenantId),
		state,
	}
}

func renderProcessDefinitionDeploymentLogs(cmd *cobra.Command, resp []resource.ProcessDefinitionDeployment) error {
	items := append([]resource.ProcessDefinitionDeployment(nil), resp...)
	slices.SortFunc(items, compareProcessDefinitionDeployments)

	lines := formatFlatRows(mapProcessDefinitionDeploymentRows(items))
	for i, line := range lines {
		renderHumanLine(cmd, "%s", line)
		if flagVerbose && items[i].ResourceName != "" {
			renderHumanLine(cmd, "  resource: %s", items[i].ResourceName)
		}
	}

	state := "deployed"
	if flagNoWait {
		state = "submitted"
	}
	renderHumanLine(cmd, "pd deploy done; %s %d%s", state, len(items), processDefinitionDeploymentSummarySuffix(items))
	return nil
}

func mapProcessDefinitionDeploymentRows(items []resource.ProcessDefinitionDeployment) []flatRow {
	rows := make([]flatRow, 0, len(items))
	for _, it := range items {
		rows = append(rows, flatRowPDDeploy(it))
	}
	return rows
}

func processDefinitionDeploymentSummarySuffix(items []resource.ProcessDefinitionDeployment) string {
	tenant, tenantOK := commonDeploymentTenant(items)
	deployment, deploymentOK := commonDeploymentKey(items)
	var suffix string
	if tenantOK && tenant != "" {
		suffix += fmt.Sprintf(", tenant %s", tenant)
	}
	if deploymentOK && deployment != "" && deployment != "<unknown>" {
		suffix += fmt.Sprintf(", deployment %s", deployment)
	}
	return suffix
}

func commonDeploymentTenant(items []resource.ProcessDefinitionDeployment) (string, bool) {
	if len(items) == 0 {
		return "", false
	}
	tenant := items[0].TenantId
	for _, it := range items[1:] {
		if it.TenantId != tenant {
			return "", false
		}
	}
	return tenant, true
}

func commonDeploymentKey(items []resource.ProcessDefinitionDeployment) (string, bool) {
	if len(items) == 0 {
		return "", false
	}
	key := items[0].Key
	for _, it := range items[1:] {
		if it.Key != key {
			return "", false
		}
	}
	return key, true
}

func compareProcessDefinitionDeployments(a, b resource.ProcessDefinitionDeployment) int {
	if a.DefinitionId != b.DefinitionId {
		return cmpString(a.DefinitionId, b.DefinitionId)
	}
	if a.DefinitionVersion != b.DefinitionVersion {
		if a.DefinitionVersion < b.DefinitionVersion {
			return -1
		}
		return 1
	}
	if a.TenantId != b.TenantId {
		return cmpString(a.TenantId, b.TenantId)
	}
	return cmpString(a.DefinitionKey, b.DefinitionKey)
}

func cmpString(a, b string) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}
