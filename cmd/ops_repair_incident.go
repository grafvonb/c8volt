// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

const opsRepairIncidentCommandName = "ops repair incident"

var (
	flagOpsRepairIncidentKeys          []string
	flagOpsRepairIncidentRetries       int32
	flagOpsRepairIncidentJobTimeoutRaw string
)

var opsRepairIncidentCmd = &cobra.Command{
	Use:   "incident",
	Short: "Repair incidents by key",
	Long: "Repair incidents by key.\n\n" +
		"The command accepts repeated --key values or newline-separated keys from stdin with '-'. It freezes the requested incident set before mutation, applies job retry and timeout updates only when an incident has a related job, resolves each incident, and confirms clearance unless --no-wait is set. Incidents without related jobs report job steps as not_applicable and still proceed to incident resolution.",
	Example: `  ./c8volt ops repair incident --key <incident-key>
  ./c8volt ops repair inc --key <incident-key> --key <another-incident-key>
  printf '%s\n' "$INCIDENT_KEY_A" "$INCIDENT_KEY_B" | ./c8volt ops repair incident -
  ./c8volt ops repair incident --key <incident-key> --retries 0
  ./c8volt ops repair incident --key <incident-key> --job-timeout 5m
  ./c8volt ops repair incident --key <incident-key> --dry-run
  ./c8volt --json ops repair incident --key <incident-key> --automation`,
	Aliases: []string{"inc"},
	Args: func(cmd *cobra.Command, args []string) error {
		if err := validateOptionalDashArg(args); err != nil {
			return silenceUsageForError(cmd, err)
		}
		return silenceUsageForError(cmd, validateOpsRepairIncidentFlagValues(cmd))
	},
	Run: func(cmd *cobra.Command, args []string) {
		timeout, err := parseOpsRepairIncidentJobTimeout(cmd)
		if err != nil {
			failBeforeCli(cmd, err)
		}
		cli, log, cfg, err := NewCli(cmd)
		if err != nil {
			handleNewCliError(cmd, log, cfg, fmt.Errorf("initializing client: %w", err))
		}
		if err := requireAutomationSupport(cmd); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		stdinKeys, err := readKeysIfDash(args)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		}
		keys := mergeAndValidateKeys(flagOpsRepairIncidentKeys, stdinKeys, log, cfg).Unique()
		if len(keys) == 0 {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, localPreconditionError(fmt.Errorf("no incident keys provided or found to repair")))
		}
		if ok, firstBadKey, _ := validateKeys(keys); !ok {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, invalidFlagValuef("incident key %q is not a valid key", firstBadKey))
		}
		retries := flagOpsRepairIncidentRetries
		mode := ops.RepairDiscoveryModeKeyed
		if len(stdinKeys) > 0 && len(flagOpsRepairIncidentKeys) == 0 {
			mode = ops.RepairDiscoveryModeStdin
		}
		result, err := cli.RepairIncidents(cmd.Context(), ops.RepairRequest{
			CommandName:         opsRepairIncidentCommandName,
			Target:              ops.RepairTargetIncident,
			DiscoveryMode:       mode,
			InputKeys:           append(typex.Keys{}, keys...),
			Workers:             flagWorkers,
			FailFast:            flagFailFast,
			NoWorkerLimit:       flagNoWorkerLimit,
			DryRun:              flagDryRun,
			AutoConfirm:         flagCmdAutoConfirm,
			Automation:          automationModeEnabled(cmd),
			NoWait:              flagNoWait,
			OutputMode:          pickMode().String(),
			RequestedRetries:    &retries,
			RequestedJobTimeout: timeout,
			StartedAt:           time.Now().UTC(),
		}, collectOptions()...)
		renderErr := renderOpsRepairIncidentResult(cmd, result)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("ops repair incident: %w", err))
		}
		if renderErr != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, fmt.Errorf("render ops repair incident: %w", renderErr))
		}
	},
}

func init() {
	opsRepairCmd.AddCommand(opsRepairIncidentCmd)
	useInvalidInputFlagErrors(opsRepairIncidentCmd)

	fs := opsRepairIncidentCmd.Flags()
	fs.StringSliceVarP(&flagOpsRepairIncidentKeys, "key", "k", nil, "incident key(s) to repair; repeat or combine with stdin '-'")
	fs.Int32Var(&flagOpsRepairIncidentRetries, "retries", 1, "retry count to set on related jobs; 0 skips retry restoration")
	fs.StringVar(&flagOpsRepairIncidentJobTimeoutRaw, "job-timeout", "", "timeout duration to submit for related jobs, for example 60s, 5m, or 1h")
	fs.BoolVar(&flagDryRun, "dry-run", false, "freeze repair targets and preview repair steps without submitting mutations")
	fs.BoolVar(&flagNoWait, "no-wait", false, "return after repair mutations are accepted without incident or retry confirmation")
	fs.IntVarP(&flagWorkers, "workers", "w", 0, "maximum concurrent workers when repairing multiple incidents (default: min(count, 2*GOMAXPROCS, 32))")
	fs.BoolVar(&flagNoWorkerLimit, "no-worker-limit", false, "use all queued jobs as workers when --workers is unset")
	fs.BoolVar(&flagFailFast, "fail-fast", false, "stop scheduling incident repairs after the first error")

	setCommandMutation(opsRepairIncidentCmd, CommandMutationStateChanging)
	setContractSupport(opsRepairIncidentCmd, ContractSupportFull)
	setAutomationSupport(opsRepairIncidentCmd, AutomationSupportFull, "supports shared machine output, stdin key pipelines, dry-run previews, and unattended repair")
	setOutputModes(opsRepairIncidentCmd,
		OutputModeContract{Name: RenderModeOneLine.String(), Supported: true},
		OutputModeContract{Name: RenderModeJSON.String(), Supported: true, MachinePreferred: true},
	)
}

func validateOpsRepairIncidentFlagValues(cmd *cobra.Command) error {
	if cmd != nil && cmd.Flags().Changed("workers") && flagWorkers < 1 {
		return invalidFlagValuef("--workers must be positive integer")
	}
	if flagOpsRepairIncidentRetries < 0 {
		return invalidFlagValuef("invalid value for --retries: %d, expected non-negative integer", flagOpsRepairIncidentRetries)
	}
	if pickMode() == RenderModeJSON && flagVerbose {
		return mutuallyExclusiveFlagsf("--json cannot be combined with --verbose for ops repair incident")
	}
	if ok, firstBadKey, _ := validateKeys(flagOpsRepairIncidentKeys); len(flagOpsRepairIncidentKeys) > 0 && !ok {
		return invalidFlagValuef("incident key %q is not a valid key", firstBadKey)
	}
	return nil
}

func parseOpsRepairIncidentJobTimeout(cmd *cobra.Command) (time.Duration, error) {
	if cmd == nil || !cmd.Flags().Changed("job-timeout") {
		return 0, nil
	}
	timeout, err := time.ParseDuration(flagOpsRepairIncidentJobTimeoutRaw)
	if err != nil || timeout <= 0 {
		return 0, invalidFlagValuef("invalid value for --job-timeout: %q, expected positive duration such as 60s, 5m, or 1h", flagOpsRepairIncidentJobTimeoutRaw)
	}
	if timeout.Milliseconds() <= 0 {
		return 0, invalidFlagValuef("invalid value for --job-timeout: %q, duration must be at least 1ms", flagOpsRepairIncidentJobTimeoutRaw)
	}
	return timeout, nil
}
