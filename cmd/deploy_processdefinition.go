package cmd

import (
	"fmt"
	"os"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagDeployPDFiles    []string
	flagDeployPDRunCount int
	flagDeployPDVars     string
	flagDeployPDVarsFile string
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
		pdds, err := cli.DeployProcessDefinition(cmd.Context(), cfg.App.Tenant, res, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deploying process definition(s): %w", err))
		}
		err = listProcessDefinitionDeploymentsView(cmd, pdds)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("rendering process definition deployment view: %w", err))
		}
		log.Debug(fmt.Sprintf("%d process definition(s) to tenant %q deployed successfully", len(pdds), cfg.App.ViewTenant()))

		// Run process instances if --run flag is set
		if flagDeployPDRunCount > 0 {
			if err := runProcessInstancesAfterDeploy(cmd, cli, log, cfg, pdds, flagDeployPDRunCount, flagDeployPDVars, flagDeployPDVarsFile, flagCmdAutoConfirm); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
			}
		}
	},
}

func init() {
	deployCmd.AddCommand(deployProcessDefinitionCmd)
	deployProcessDefinitionCmd.Flags().StringSliceVarP(&flagDeployPDFiles, "file", "f", nil, "paths to BPMN/YAML file(s) or '-' for stdin")
	_ = deployProcessDefinitionCmd.MarkFlagRequired("file")

	deployProcessDefinitionCmd.Flags().IntVarP(&flagDeployPDRunCount, "run", "n", 0, "run N process instance(s) after deploying process definition(s)")
	deployProcessDefinitionCmd.Flags().StringVar(&flagDeployPDVars, "vars", "", "JSON-encoded variables for process instance(s) (requires --run)")
	deployProcessDefinitionCmd.Flags().StringVar(&flagDeployPDVarsFile, "vars-file", "", "path to JSON file with variables for process instance(s) (requires --run)")
}
