package cmd

import (
	"fmt"
	"os"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/spf13/cobra"
)

var (
	flagDeployPDFiles       []string
	flagDeployPDWithRun     bool
	flagDeployPDRunCount    int
	flagDeployPDRunVars     string
	flagDeployPDRunVarsFile string
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
		if flagDeployPDWithRun {
			if err := runProcessInstancesAfterDeploy(cmd, cli, log, cfg, pdds, flagDeployPDRunCount, flagDeployPDRunVars, flagDeployPDRunVarsFile, flagCmdAutoConfirm); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
			}
		}
	},
}

func init() {
	deployCmd.AddCommand(deployProcessDefinitionCmd)
	deployProcessDefinitionCmd.Flags().StringSliceVarP(&flagDeployPDFiles, "file", "f", nil, "paths to BPMN/YAML file(s) or '-' for stdin")
	_ = deployProcessDefinitionCmd.MarkFlagRequired("file")

	deployProcessDefinitionCmd.Flags().BoolVar(&flagDeployPDWithRun, "run", false, "run process instance(s) after deploying process definition(s)")
	deployProcessDefinitionCmd.Flags().IntVar(&flagDeployPDRunCount, "run-count", 1, "number of process instances to start (requires --run)")
	deployProcessDefinitionCmd.Flags().StringVar(&flagDeployPDRunVars, "run-vars", "", "JSON-encoded variables for process instance(s) (requires --run)")
	deployProcessDefinitionCmd.Flags().StringVar(&flagDeployPDRunVarsFile, "run-vars-file", "", "path to JSON file with variables for process instance(s) (requires --run)")
}
