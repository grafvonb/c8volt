package cmd

import (
	"fmt"
	"os"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/spf13/cobra"
)

var (
	flagDeployPDFiles   []string
	flagDeployPDWithRun bool
)

var deployProcessDefinitionCmd = &cobra.Command{
	Use:     "process-definition",
	Short:   "Deploy a process definition",
	Aliases: []string{"pd"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
		}
		if err := validateFiles(flagDeployPDFiles); err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("validating files with process definition(s): %w", err))
		}
		res, err := loadResources(flagDeployPDFiles, os.Stdin)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("collecting process definition(s): %w", err))
		}
		log.Debug(fmt.Sprintf("deploying process definition(s) to tenant %q", cfg.App.ViewTenant()))
		opts := collectOptions()
		pdds, err := cli.DeployProcessDefinition(cmd.Context(), res, opts...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deploying process definition(s): %w", err))
		}
		err = listProcessDefinitionDeploymentsView(cmd, pdds)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("rendering process definition deployment view: %w", err))
		}
		log.Debug(fmt.Sprintf("%d process definition(s) to tenant %q deployed successfully", len(pdds), cfg.App.ViewTenant()))

		if flagDeployPDWithRun {
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
	deployCmd.AddCommand(deployProcessDefinitionCmd)

	fs := deployProcessDefinitionCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "skip waiting for the deployment to be fully processed")
	fs.StringSliceVarP(&flagDeployPDFiles, "file", "f", nil, "paths to BPMN/YAML file(s) or '-' for stdin")
	_ = deployProcessDefinitionCmd.MarkFlagRequired("file")

	fs.BoolVar(&flagDeployPDWithRun, "run", false, "run single process instance without vars after deploying process definition(s)")
}
