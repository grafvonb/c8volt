package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"slices"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/embedded"
	"github.com/spf13/cobra"
)

var (
	flagEmbedDeployFileNames  []string
	flagEmbedDeployAll        bool
	flagEmbedDeployWithRun    bool
	flagEmbedDeployRunCount   int
	flagEmbedDeployRunVars    string
	flagEmbedDeployRunVarsFile string
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
		pdds, err := cli.DeployProcessDefinition(cmd.Context(), cfg.App.Tenant, units, collectOptions()...)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("deploying embedded resource(s): %w", err))
		}
		err = listProcessDefinitionDeploymentsView(cmd, pdds)
		if err != nil {
			ferrors.HandleAndExit(log, cfg.App.NoErrCodes, fmt.Errorf("rendering process definition deployment view: %w", err))
		}
		log.Debug(fmt.Sprintf("%d embedded resource(s) to tenant %q deployed successfully", len(pdds), cfg.App.ViewTenant()))

		// Run process instances if --run flag is set
		if flagEmbedDeployWithRun {
			if err := runAfterEmbedDeploy(cmd, cli, log, cfg, pdds); err != nil {
				ferrors.HandleAndExit(log, cfg.App.NoErrCodes, err)
			}
		}
	},
}

func init() {
	embedCmd.AddCommand(embedDeployCmd)
	embedDeployCmd.Flags().StringSliceVarP(&flagEmbedDeployFileNames, "file", "f", nil, "embedded file(s) to deploy (repeatable)")
	embedDeployCmd.Flags().BoolVar(&flagEmbedDeployAll, "all", false, "deploy all embedded files for the configured Camunda version")

	embedDeployCmd.Flags().BoolVar(&flagEmbedDeployWithRun, "run", false, "run process instance(s) after deploying process definition(s)")
	embedDeployCmd.Flags().IntVar(&flagEmbedDeployRunCount, "run-count", 1, "number of process instances to start (requires --run)")
	embedDeployCmd.Flags().StringVar(&flagEmbedDeployRunVars, "run-vars", "", "JSON-encoded variables for process instance(s) (requires --run)")
	embedDeployCmd.Flags().StringVar(&flagEmbedDeployRunVarsFile, "run-vars-file", "", "path to JSON file with variables for process instance(s) (requires --run)")
}

func runAfterEmbedDeploy(cmd *cobra.Command, cli interface {
	CreateProcessInstances(ctx context.Context, datas []process.ProcessInstanceData, opts ...foptions.FacadeOption) ([]process.ProcessInstance, error)
	CreateNProcessInstances(ctx context.Context, data process.ProcessInstanceData, count int, workers int, opts ...foptions.FacadeOption) ([]process.ProcessInstance, error)
}, log *slog.Logger, cfg *config.Config, pdds []resource.ProcessDefinitionDeployment) error {
	// Validate run count
	if flagEmbedDeployRunCount < 1 {
		return fmt.Errorf("--run-count must be a positive integer")
	}

	// Load variables from file or flag
	var vars map[string]interface{}
	if flagEmbedDeployRunVarsFile != "" && flagEmbedDeployRunVars != "" {
		return fmt.Errorf("--run-vars and --run-vars-file are mutually exclusive")
	}
	if flagEmbedDeployRunVarsFile != "" {
		data, err := os.ReadFile(flagEmbedDeployRunVarsFile)
		if err != nil {
			return fmt.Errorf("reading variables file: %w", err)
		}
		if err := json.Unmarshal(data, &vars); err != nil {
			return fmt.Errorf("parsing variables from file: %w", err)
		}
	} else if flagEmbedDeployRunVars != "" {
		if err := json.Unmarshal([]byte(flagEmbedDeployRunVars), &vars); err != nil {
			return fmt.Errorf("parsing --run-vars JSON: %w", err)
		}
	}

	// Ask for confirmation
	prompt := fmt.Sprintf("Run %d process instance(s) for %d deployed process definition(s)?", flagEmbedDeployRunCount, len(pdds))
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
		if flagEmbedDeployRunCount == 1 {
			instances, err = cli.CreateProcessInstances(cmd.Context(), []process.ProcessInstanceData{data}, fopts...)
		} else {
			instances, err = cli.CreateNProcessInstances(cmd.Context(), data, flagEmbedDeployRunCount, 0, fopts...)
		}

		if err != nil {
			return fmt.Errorf("running process instance(s) for %s (key %s): %w", pdd.DefinitionKey, pdd.DefinitionId, err)
		}

		log.Debug(fmt.Sprintf("%d process instance(s) started for process definition %s (key %s)", len(instances), pdd.DefinitionKey, pdd.DefinitionId))
	}

	return nil
}
