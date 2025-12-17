package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

var (
	flagDeployPDFiles        []string
	flagDeployPDWithRun      bool
	flagDeployPDRunCount     int
	flagDeployPDRunVars      string
	flagDeployPDRunVarsFile  string
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
			if err := runAfterDeploy(cmd, cli, log, cfg, pdds); err != nil {
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

func runAfterDeploy(cmd *cobra.Command, cli interface {
	CreateProcessInstances(ctx context.Context, datas []process.ProcessInstanceData, opts ...foptions.FacadeOption) ([]process.ProcessInstance, error)
	CreateNProcessInstances(ctx context.Context, data process.ProcessInstanceData, count int, workers int, opts ...foptions.FacadeOption) ([]process.ProcessInstance, error)
}, log *slog.Logger, cfg *config.Config, pdds []resource.ProcessDefinitionDeployment) error {
	// Validate run count
	if flagDeployPDRunCount < 1 {
		return fmt.Errorf("--run-count must be a positive integer")
	}

	// Load variables from file or flag
	var vars map[string]interface{}
	if flagDeployPDRunVarsFile != "" && flagDeployPDRunVars != "" {
		return fmt.Errorf("--run-vars and --run-vars-file are mutually exclusive")
	}
	if flagDeployPDRunVarsFile != "" {
		data, err := os.ReadFile(flagDeployPDRunVarsFile)
		if err != nil {
			return fmt.Errorf("reading variables file: %w", err)
		}
		if err := json.Unmarshal(data, &vars); err != nil {
			return fmt.Errorf("parsing variables from file: %w", err)
		}
	} else if flagDeployPDRunVars != "" {
		if err := json.Unmarshal([]byte(flagDeployPDRunVars), &vars); err != nil {
			return fmt.Errorf("parsing --run-vars JSON: %w", err)
		}
	}

	// Ask for confirmation
	prompt := fmt.Sprintf("Run %d process instance(s) for %d deployed process definition(s)?", flagDeployPDRunCount, len(pdds))
	if err := confirmCmdOrAbort(flagCmdAutoConfirm, prompt); err != nil {
		return err
	}

	// Run instances for each deployed definition
	fopts := collectOptions()
	for _, pdd := range pdds {
		data := process.ProcessInstanceData{
			ProcessDefinitionSpecificId: pdd.DefinitionId,
			Variables:                   vars,
			TenantId:                    cfg.App.Tenant,
		}

		var instances []process.ProcessInstance
		var err error
		if flagDeployPDRunCount == 1 {
			instances, err = cli.CreateProcessInstances(cmd.Context(), []process.ProcessInstanceData{data}, fopts...)
		} else {
			instances, err = cli.CreateNProcessInstances(cmd.Context(), data, flagDeployPDRunCount, 0, fopts...)
		}

		if err != nil {
			return fmt.Errorf("running process instance(s) for %s (key %s): %w", pdd.DefinitionKey, pdd.DefinitionId, err)
		}

		log.Debug(fmt.Sprintf("%d process instance(s) started for process definition %s (key %s)", len(instances), pdd.DefinitionKey, pdd.DefinitionId))
	}

	return nil
}
