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
	return listOrJSON(cmd, resp, resp, pickMode(), oneLinePDDeploy, func(it resource.ProcessDefinitionDeployment) string { return it.DefinitionKey })
}

func oneLinePDDeploy(it resource.ProcessDefinitionDeployment) string {
	return fmt.Sprintf(
		"%-16s %s %s v%d v%s (deployId: %s)",
		it.DefinitionKey, it.TenantId, it.DefinitionId, it.DefinitionVersion, it.ResourceName, it.Key,
	)
}
