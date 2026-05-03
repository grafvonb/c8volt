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

// flatRowPDDeploy aligns deployment summaries without moving the deployed BPMN definition away from the front.
func flatRowPDDeploy(it resource.ProcessDefinitionDeployment) flatRow {
	return flatRow{
		it.DefinitionKey,
		it.TenantId,
		it.DefinitionId,
		fmt.Sprintf("v%d", it.DefinitionVersion),
		it.ResourceName,
		fmt.Sprintf("(deployId: %s)", it.Key),
	}
}
