// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/spf13/cobra"
)

//nolint:unused
func processDefinitionDeploymentView(cmd *cobra.Command, item resource.ProcessDefinitionDeployment) error {
	return itemView(cmd, item, pickMode(), oneLinePDDeploy, func(it resource.ProcessDefinitionDeployment) string { return it.DefinitionKey })
}

func listProcessDefinitionDeploymentsView(cmd *cobra.Command, resp []resource.ProcessDefinitionDeployment) error {
	return listOrJSONFlat(cmd, resp, resp, pickMode(), flatRowPDDeploy, func(it resource.ProcessDefinitionDeployment) string { return it.DefinitionKey })
}

func oneLinePDDeploy(it resource.ProcessDefinitionDeployment) string {
	return compactFlatRow(flatRowPDDeploy(it))
}

// flatRowPDDeploy keeps deployment summaries on the same pd-first grammar as other process-definition output.
func flatRowPDDeploy(it resource.ProcessDefinitionDeployment) flatRow {
	return flatRow{
		"pd",
		it.DefinitionKey,
		it.DefinitionId,
		fmt.Sprintf("v%d", it.DefinitionVersion),
		fmt.Sprintf("%s;", it.TenantId),
		"deploy",
		fmt.Sprintf("%s;", it.Key),
		"resource",
		it.ResourceName,
	}
}
