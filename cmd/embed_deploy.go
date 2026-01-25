package cmd

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/spf13/cobra"
)

var (
	flagEmbedDeployFileNames []string
	flagEmbedDeployAll       bool
)

var embedDeployCmd = &cobra.Command{
	Use:     "deploy",
	Short:   "Deploy embedded (virtual) resources",
	Aliases: []string{"dep"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}

		all, err := embedded.List()
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		var toDeploy []string
		switch {
		case flagEmbedDeployAll:
			for _, d := range all {
				if strings.Contains(d, cfg.App.CamundaVersion.FilePrefix()) {
					toDeploy = append(toDeploy, d)
				}
			}
			if len(toDeploy) == 0 {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("no deployable embedded files found for Camunda version %q", cfg.App.CamundaVersion.String()))
			}
		case len(flagEmbedDeployFileNames) == 0:
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("either --all or at least one --file is required"))
		default:
			for _, f := range flagEmbedDeployFileNames {
				if !slices.Contains(all, f) {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("embedded file %q not found, run command 'embed list' to see all available embedded files, no deployment done", f))
				}
			}
			toDeploy = append(toDeploy, flagEmbedDeployFileNames...)
		}

		var units []resource.DeploymentUnitData
		for _, f := range toDeploy {
			data, err := fs.ReadFile(embedded.FS, f)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("read embedded %q: %w", f, err))
			}
			log.Debug(fmt.Sprintf("deploying embedded resource(s) %q to tenant %s", f, cfg.App.ViewTenant()))
			units = append(units, resource.DeploymentUnitData{Name: f, Data: data})
		}

		// TODO (Adam): currently only deployment of process definitions is supported, extend to other resource types as needed
		pdds, err := cli.DeployProcessDefinition(cmd.Context(), units, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deploying embedded resource(s): %w", err))
		}
		err = listProcessDefinitionDeploymentsView(cmd, pdds)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("rendering process definition deployment view: %w", err))
		}
		log.Debug(fmt.Sprintf("%d embedded resource(s) to tenant %q deployed successfully", len(pdds), cfg.App.ViewTenant()))
	},
}

func init() {
	embedCmd.AddCommand(embedDeployCmd)
	fs := embedDeployCmd.Flags()
	fs.StringSliceVarP(&flagEmbedDeployFileNames, "file", "f", nil, "embedded file(s) to deploy (repeatable)")
	fs.BoolVar(&flagEmbedDeployAll, "all", false, "deploy all embedded files for the configured Camunda version")
	embedDeployCmd.MarkFlagsMutuallyExclusive("file", "all")
}
