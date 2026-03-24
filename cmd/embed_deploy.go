package cmd

import (
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/spf13/cobra"
)

var (
	flagEmbedDeployFileNames []string
	flagEmbedDeployAll       bool
	flagEmbedDeployWithRun   bool
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
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no deployable embedded files found for Camunda version %q", cfg.App.CamundaVersion.String())))
			}
		case len(flagEmbedDeployFileNames) == 0:
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, missingDependentFlagsf("either --all or at least one --file is required"))
		default:
			for _, f := range flagEmbedDeployFileNames {
				if !slices.Contains(all, f) {
					ferrors.HandleAndExit(log, cfg.App.NoErrCodes, invalidFlagValuef("embedded file %q not found, run command 'embed list' to see all available embedded files, no deployment done", f))
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
		opts := collectOptions()
		pdds, err := cli.DeployProcessDefinition(cmd.Context(), units, opts...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deploying embedded resource(s): %w", err))
		}
		err = listProcessDefinitionDeploymentsView(cmd, pdds)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("rendering process definition deployment view: %w", err))
		}
		log.Debug(fmt.Sprintf("%d embedded resource(s) to tenant %q deployed successfully", len(pdds), cfg.App.ViewTenant()))

		if flagEmbedDeployWithRun {
			log.Debug(fmt.Sprintf("running process instance(s) for deployed process definition(s) to tenant %q", cfg.App.ViewTenant()))
			datas := make([]process.ProcessInstanceData, 0, len(pdds))
			for _, pdd := range pdds {
				datas = append(datas, process.ProcessInstanceData{
					ProcessDefinitionSpecificId: pdd.DefinitionKey,
					Variables:                   nil,
					TenantId:                    cfg.App.Tenant,
				})
			}
			_, err = cli.CreateProcessInstances(cmd.Context(), datas, opts...)
			if err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("%w: running process instance(s)", err))
			}
		}
	},
}

func init() {
	embedCmd.AddCommand(embedDeployCmd)
	fs := embedDeployCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip waiting for the deployment to be fully processed")
	fs.StringSliceVarP(&flagEmbedDeployFileNames, "file", "f", nil, "embedded file(s) to deploy (repeatable)")
	fs.BoolVar(&flagEmbedDeployAll, "all", false, "deploy all embedded files for the configured Camunda version")
	embedDeployCmd.MarkFlagsMutuallyExclusive("file", "all")

	fs.BoolVar(&flagEmbedDeployWithRun, "run", false, "run single process instance without vars after deploying process definition(s)")
}
