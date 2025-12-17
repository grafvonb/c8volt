package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/resource"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// ProcessInstanceCreator defines the interface for creating process instances
type ProcessInstanceCreator interface {
	CreateProcessInstances(ctx context.Context, datas []process.ProcessInstanceData, opts ...foptions.FacadeOption) ([]process.ProcessInstance, error)
	CreateNProcessInstances(ctx context.Context, data process.ProcessInstanceData, count int, workers int, opts ...foptions.FacadeOption) ([]process.ProcessInstance, error)
}

// runProcessInstancesAfterDeploy runs process instances after successful deployment
func runProcessInstancesAfterDeploy(
	cmd *cobra.Command,
	cli ProcessInstanceCreator,
	log *slog.Logger,
	cfg *config.Config,
	pdds []resource.ProcessDefinitionDeployment,
	runCount int,
	varsJSON string,
	varsFile string,
	autoConfirm bool,
) error {
	// Validate run count
	if runCount < 1 {
		return fmt.Errorf("--run-count must be a positive integer")
	}

	// Load variables from file or flag
	var vars map[string]interface{}
	if varsFile != "" && varsJSON != "" {
		return fmt.Errorf("--run-vars and --run-vars-file are mutually exclusive")
	}
	if varsFile != "" {
		data, err := os.ReadFile(varsFile)
		if err != nil {
			return fmt.Errorf("reading variables file: %w", err)
		}
		if err := json.Unmarshal(data, &vars); err != nil {
			return fmt.Errorf("parsing variables from file: %w", err)
		}
	} else if varsJSON != "" {
		if err := json.Unmarshal([]byte(varsJSON), &vars); err != nil {
			return fmt.Errorf("parsing --run-vars JSON: %w", err)
		}
	}

	// Ask for confirmation
	prompt := fmt.Sprintf("Run %d process instance(s) for %d deployed process definition(s)?", runCount, len(pdds))
	if err := confirmCmdOrAbort(autoConfirm, prompt); err != nil {
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
		if runCount == 1 {
			instances, err = cli.CreateProcessInstances(cmd.Context(), []process.ProcessInstanceData{data}, fopts...)
		} else {
			instances, err = cli.CreateNProcessInstances(cmd.Context(), data, runCount, 0, fopts...)
		}

		if err != nil {
			return fmt.Errorf("running process instance(s) for %s (key %s): %w", pdd.DefinitionKey, pdd.DefinitionId, err)
		}

		log.Debug(fmt.Sprintf("%d process instance(s) started for process definition %s (key %s)", len(instances), pdd.DefinitionKey, pdd.DefinitionId))
	}

	return nil
}
