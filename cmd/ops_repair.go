// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/incident"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

var opsRepairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Discover repair and remediation workflows",
	Long: `Discover repair and remediation workflows.

The repair command group lists target-specific remediation workflows for
incidents and process-instance selected incidents. Use a concrete target command
to provide keys, filters, dry-run controls, variable updates, job repair
options, and audit reports. This grouping command does not define target keys or
run remediation behavior by itself.`,
	Example: `  ./c8volt ops repair --help
  ./c8volt capabilities --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
	SuggestFor: []string{"fix", "remediate", "remediation"},
}

func init() {
	opsCmd.AddCommand(opsRepairCmd)

	setCommandMutation(opsRepairCmd, CommandMutationStateChanging)
}

// opsRepairNeedsPreflight reports whether a state-changing repair must run a dry-run plan before mutation.
func opsRepairNeedsPreflight(cmd *cobra.Command) bool {
	return !flagDryRun && !shouldImplicitlyConfirm(cmd)
}

// opsRepairConfirmedRequestFromPlan pins the confirmed mutation request to the target set shown during preflight.
func opsRepairConfirmedRequestFromPlan(request ops.RepairRequest, planned ops.RepairResult) ops.RepairRequest {
	switch request.Target {
	case ops.RepairTargetIncident:
		request.DiscoveryMode = ops.RepairDiscoveryModeKeyed
		request.InputKeys = append(typex.Keys{}, planned.FrozenSet.IncidentKeys...)
		request.IncidentSelection = incident.Filter{}
	case ops.RepairTargetProcessInstance:
		request.DiscoveryMode = ops.RepairDiscoveryModeKeyed
		request.InputKeys = append(typex.Keys{}, planned.FrozenSet.ProcessInstanceKeys...)
		request.ProcessInstanceSelection = process.ProcessInstanceFilter{}
		request.DirectIncidentsOnly = false
	}
	return request
}

// opsRepairPlanHasRepairTargets reports whether the preflight found incident mutations to submit.
func opsRepairPlanHasRepairTargets(planned ops.RepairResult) bool {
	return len(planned.FrozenSet.IncidentKeys) > 0
}

// opsRepairResultWithoutMutation keeps a no-target preflight result renderable as the user's original command.
func opsRepairResultWithoutMutation(request ops.RepairRequest, planned ops.RepairResult) ops.RepairResult {
	planned.Request = request
	planned.Report.Request = request
	planned.Report.DryRun = request.DryRun
	return planned
}

// opsRepairConfirmationPrompt summarizes the preflight target set before mutation.
func opsRepairConfirmationPrompt(planned ops.RepairResult) string {
	jobSteps := len(planned.FrozenSet.JobKeys)
	variableScopes := len(planned.FrozenSet.VariableScopes)
	switch planned.Request.Target {
	case ops.RepairTargetProcessInstance:
		return fmt.Sprintf(
			"Process-instance repair matched %d repairable process instance(s), %d active incident(s), and skipped %d process instance(s) without active incidents; repair will update %d variable scope(s), apply job repair where applicable to %d related job(s), and resolve %d incident(s). Do you want to proceed?",
			len(planned.FrozenSet.ProcessInstanceKeys),
			len(planned.FrozenSet.IncidentKeys),
			len(planned.FrozenSet.SkippedProcessInstanceKeys),
			variableScopes,
			jobSteps,
			len(planned.FrozenSet.IncidentKeys),
		)
	default:
		return fmt.Sprintf(
			"Incident repair matched %d incident(s) across %d process instance(s); repair will update %d variable scope(s), apply job repair where applicable to %d related job(s), and resolve %d incident(s). Do you want to proceed?",
			len(planned.FrozenSet.IncidentKeys),
			len(planned.FrozenSet.ProcessInstanceKeys),
			variableScopes,
			jobSteps,
			len(planned.FrozenSet.IncidentKeys),
		)
	}
}
