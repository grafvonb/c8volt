package cmd

import (
	"fmt"
	"os"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
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
			ferrors.HandleAndExit(log, err)
		}
		if err := validateFiles(flagDeployPDFiles); err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("validating files: %w", err))
		}
		res, err := loadResources(flagDeployPDFiles, os.Stdin)
		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("collecting resources: %w", err))
		}
		log.Debug(fmt.Sprintf("deploying process definition(s) to tenant %q", cfg.App.ViewTenant()))
		_, err = cli.DeployProcessDefinition(cmd.Context(), cfg.App.Tenant, res, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, fmt.Errorf("deploying process definition: %w", err))
		}
		log.Info(fmt.Sprintf("process definition(s) to tenant %q deployed successfully", cfg.App.ViewTenant()))
	},
}

func init() {
	deployCmd.AddCommand(deployProcessDefinitionCmd)
	deployProcessDefinitionCmd.Flags().StringSliceVarP(&flagDeployPDFiles, "file", "f", nil, "paths to BPMN/YAML file(s) or '-' for stdin")
	_ = deployProcessDefinitionCmd.MarkFlagRequired("file")

	deployProcessDefinitionCmd.Flags().BoolVar(&flagDeployPDWithRun, "with-run", false, "run a process instance after deploying process definition(s)")
}
