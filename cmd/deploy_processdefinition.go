// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagDeployPDFiles   []string
	flagDeployPDWithRun bool
)

var deployProcessDefinitionCmd = &cobra.Command{
	Use:   "process-definition",
	Short: "Deploy BPMN process definition files",
	Long: "Deploy BPMN process definition files and report the deployed definitions.\n\n" +
		"By default c8volt waits for deployment confirmation. Use --run to start one process instance for each deployed definition.\n\n" +
		"Add --no-wait to verify later with `get pd`.",
	Example: `  ./c8volt embed export --file processdefinitions/C88_SimpleUserTaskProcess.bpmn --out ./fixtures
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn --run
  ./c8volt deploy pd --file ./fixtures/processdefinitions/C88_SimpleUserTaskProcess.bpmn --no-wait
  ./c8volt get pd --bpmn-process-id C88_SimpleUserTask_Process --latest --json`,
	Aliases: []string{"pd"},
	Run: func(cmd *cobra.Command, args []string) {
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, err)
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		if err := validateFiles(flagDeployPDFiles); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("validating files with process definition(s): %w", err))
		}
		res, err := loadResources(flagDeployPDFiles, os.Stdin)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("collecting process definition(s): %w", err))
		}
		log.Debug(fmt.Sprintf("deploying process definition(s) to tenant %q", cfg.App.ViewTenant()))
		opts := collectOptions()
		pdds, err := cli.DeployProcessDefinition(cmd.Context(), res, opts...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("deploying process definition(s): %w", err))
		}
		if err := renderCommandResult(cmd, pdds); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render deployment result: %w", err))
		}
		if !commandUsesSharedEnvelope(cmd, pickMode()) {
			err = listProcessDefinitionDeploymentsView(cmd, pdds)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("rendering process definition deployment view: %w", err))
			}
		}
		log.Debug(fmt.Sprintf("%d process definition(s) to tenant %q deployed successfully", len(pdds), cfg.App.ViewTenant()))

		if flagDeployPDWithRun {
			log.Debug(fmt.Sprintf("running process instance(s) for deployed process definition(s) to tenant %q", cfg.App.ViewTenant()))
			datas, err := buildRunProcessInstanceDatasFromDeployments(pdds, res, cfg.App.TargetTenant())
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			}
			_, err = cli.CreateProcessInstances(cmd.Context(), datas, opts...)
			if err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("%w: running process instance(s)", err))
			}
		}
	},
}

func init() {
	deployCmd.AddCommand(deployProcessDefinitionCmd)

	fs := deployProcessDefinitionCmd.Flags()
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after deployment is accepted")
	fs.StringSliceVarP(&flagDeployPDFiles, "file", "f", nil, "paths to BPMN/YAML file(s) or '-' for stdin")
	_ = deployProcessDefinitionCmd.MarkFlagRequired("file")

	fs.BoolVar(&flagDeployPDWithRun, "run", false, "start one process instance per deployed definition")

	setCommandMutation(deployProcessDefinitionCmd, CommandMutationStateChanging)
	setContractSupport(deployProcessDefinitionCmd, ContractSupportFull)
	setAutomationSupport(deployProcessDefinitionCmd, AutomationSupportFull, "supports shared machine output and accepted results with --no-wait")
}
