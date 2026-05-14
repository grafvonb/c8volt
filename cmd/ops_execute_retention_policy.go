// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/consts"
	"github.com/spf13/cobra"
)

const opsExecuteRetentionPolicyCommandName = "ops execute retention-policy"

var flagOpsExecuteRetentionPolicyRetentionDays int

var opsExecuteRetentionPolicyCmd = &cobra.Command{
	Use:   "retention-policy",
	Short: "Execute process-instance retention cleanup",
	Long: "Execute process-instance retention cleanup.\n\n" +
		"The workflow discovers process instances older than the required retention age, freezes that seed set, and skips deletion until later delete planning and execution stages are available. Use --dry-run to inspect discovery without mutation.",
	Example: `  ./c8volt ops execute retention-policy --retention-days 90 --dry-run
  ./c8volt ops execute retention-policy --retention-days 90 --automation --json`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := validateOpsExecuteRetentionPolicyFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		if err := validatePISearchFlags(cmd); err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		boundary := pickPIDateUpperBound("", flagOpsExecuteRetentionPolicyRetentionDays)
		selection := populatePISearchFilterOpts()
		selection.EndDateBefore = boundary
		request := ops.RetentionPolicyRequest{
			CommandName:            opsExecuteRetentionPolicyCommandName,
			RetentionDays:          flagOpsExecuteRetentionPolicyRetentionDays,
			DerivedEndDateBoundary: boundary,
			DryRun:                 flagDryRun,
			AutoConfirm:            flagCmdAutoConfirm,
			Automation:             automationModeEnabled(cmd),
			OutputMode:             pickMode().String(),
			Selection:              selection,
			BatchSize:              resolvePISearchSize(cmd, cfg),
			Limit:                  flagGetPILimit,
			StartedAt:              time.Now().UTC(),
		}
		result, err := cli.ExecuteRetentionPolicy(cmd.Context(), request, collectOptions()...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops execute retention-policy: %w", err))
		}
		if err := renderOpsExecuteRetentionPolicyResult(cmd, result); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops execute retention-policy: %w", err))
		}
	},
}

func init() {
	opsExecuteCmd.AddCommand(opsExecuteRetentionPolicyCmd)
	useInvalidInputFlagErrors(opsExecuteRetentionPolicyCmd)

	fs := opsExecuteRetentionPolicyCmd.Flags()
	fs.IntVar(&flagOpsExecuteRetentionPolicyRetentionDays, "retention-days", 0, "required non-negative age in days for process-instance retention eligibility")
	fs.BoolVar(&flagDryRun, "dry-run", false, "discover and validate retention cleanup without submitting deletion requests")
	fs.StringSliceVarP(&flagGetPIKeys, "key", "k", nil, "unsupported explicit process-instance key selector")
	registerPISharedProcessDefinitionFilterFlags(fs)
	fs.StringVar(&flagGetPIProcessDefinitionKey, "pd-key", "", "process definition key (mutually exclusive with bpmn-process-id, pd-version, and pd-version-tag)")
	fs.Int32VarP(&flagGetPISize, "batch-size", "n", consts.MaxPISearchSize, fmt.Sprintf("number of process instances to inspect per page (max limit %d enforced by server)", consts.MaxPISearchSize))
	fs.Int32VarP(&flagGetPILimit, "limit", "l", 0, "maximum number of matching process instances to inspect across all pages")
	fs.StringVar(&flagGetPIParentKey, "parent-key", "", "parent process instance key to narrow retention discovery")
	fs.StringVarP(&flagGetPIState, "state", "s", "all", "state to filter process instances: all, active, completed, canceled, terminated")
	fs.BoolVar(&flagGetPIRootsOnly, "roots-only", false, "discover only root process instances")
	fs.BoolVar(&flagGetPIChildrenOnly, "children-only", false, "discover only child process instances")
	fs.BoolVar(&flagGetPIIncidentsOnly, "incidents-only", false, "discover only process instances that have incidents")
	fs.BoolVar(&flagGetPINoIncidentsOnly, "no-incidents-only", false, "discover only process instances that have no incidents")

	setCommandMutation(opsExecuteRetentionPolicyCmd, CommandMutationStateChanging)
	setContractSupport(opsExecuteRetentionPolicyCmd, ContractSupportFull)
	setAutomationSupport(opsExecuteRetentionPolicyCmd, AutomationSupportFull, "supports unattended dry-run previews and implicitly confirmed retention cleanup with shared machine output")
	setFlagContractRequired(opsExecuteRetentionPolicyCmd, "retention-days")
}

func validateOpsExecuteRetentionPolicyFlags(cmd *cobra.Command) error {
	if cmd == nil || !cmd.Flags().Changed("retention-days") {
		return invalidFlagValuef("ops execute retention-policy requires --retention-days")
	}
	if flagOpsExecuteRetentionPolicyRetentionDays < 0 {
		return invalidFlagValuef("invalid value for --retention-days: %d, expected non-negative integer", flagOpsExecuteRetentionPolicyRetentionDays)
	}
	if len(flagGetPIKeys) > 0 {
		return invalidFlagValuef("retention policy discovers eligible process instances and does not accept explicit process-instance keys")
	}
	return nil
}
