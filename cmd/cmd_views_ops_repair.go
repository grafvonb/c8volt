// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// renderOpsRepairIncidentResult renders explicit incident repair through the shared machine contract or compact human output.
func renderOpsRepairIncidentResult(cmd *cobra.Command, result ops.RepairResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	if result.Request.DryRun {
		renderHumanLine(cmd, "dry run: repair incidents")
	} else {
		renderHumanLine(cmd, "repair incidents")
	}
	renderHumanLine(cmd, "frozen incidents: %d", len(result.FrozenSet.IncidentKeys))
	if len(result.FrozenSet.JobKeys) > 0 || len(result.Plan) > 0 {
		renderHumanLine(cmd, "related jobs: %d applicable, %d not applicable", len(result.FrozenSet.JobKeys), countOpsRepairIncidentJobNotApplicable(result.Plan))
	}
	if flagVerbose {
		renderOpsRepairKeys(cmd, "incident keys", result.FrozenSet.IncidentKeys)
		renderOpsRepairKeys(cmd, "process-instance keys", result.FrozenSet.ProcessInstanceKeys)
		renderOpsRepairKeys(cmd, "job keys", result.FrozenSet.JobKeys)
		for _, item := range result.Plan {
			renderHumanLine(cmd, "incident %s: retry=%s timeout=%s resolution=%s confirmation=%s",
				item.IncidentKey,
				item.RetryUpdateStatus,
				item.TimeoutUpdateStatus,
				item.ResolutionStatus,
				item.ConfirmationStatus,
			)
		}
	}
	if result.Outcome != "" {
		line := fmt.Sprintf("outcome: %s", result.Outcome)
		if result.Request.DryRun || result.Outcome == ops.RepairOutcomePlanned {
			line += "; no changes applied"
		}
		if !flagVerbose && opsRepairIncidentHasHiddenKeys(result) {
			line += "; use --verbose to list keys"
		}
		line += opsWorkflowElapsedSuffix(result.Report.Duration)
		renderHumanLine(cmd, "%s", line)
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

func countOpsRepairIncidentJobNotApplicable(items []ops.RepairPlanItem) int {
	count := 0
	for _, item := range items {
		if item.RetryUpdateStatus == ops.WorkflowStepStatusNotApplicable || item.TimeoutUpdateStatus == ops.WorkflowStepStatusNotApplicable {
			count++
		}
	}
	return count
}

func opsRepairIncidentHasHiddenKeys(result ops.RepairResult) bool {
	return len(result.FrozenSet.IncidentKeys) > 0 ||
		len(result.FrozenSet.ProcessInstanceKeys) > 0 ||
		len(result.FrozenSet.JobKeys) > 0
}

func renderOpsRepairKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		renderHumanLine(cmd, "%s: none", label)
		return
	}
	renderHumanLine(cmd, "%s: %s", label, strings.Join(keys, ", "))
}
