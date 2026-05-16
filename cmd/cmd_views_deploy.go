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
	items := resp
	if mode == RenderModeOneLine {
		items = append([]resource.ProcessDefinitionDeployment(nil), resp...)
		slices.SortFunc(items, compareProcessDefinitionDeployments)
	}
	return listOrJSONFlat(cmd, resp, items, mode, flatRowPDDeploy, func(it resource.ProcessDefinitionDeployment) string { return it.DefinitionKey })
}

func oneLinePDDeploy(it resource.ProcessDefinitionDeployment) string {
	return compactFlatRow(flatRowPDDeploy(it))
}

// flatRowPDDeploy mirrors get pd rows for deployment summaries.
func flatRowPDDeploy(it resource.ProcessDefinitionDeployment) flatRow {
	vTag := ""
	if it.VersionTag != "" {
		vTag = "/" + it.VersionTag
	}
	return flatRow{
		it.DefinitionKey,
		it.TenantId,
		it.DefinitionId,
		fmt.Sprintf("v%d%s", it.DefinitionVersion, vTag),
	}
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
