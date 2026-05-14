// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

func renderOpsExecuteRetentionPolicyResult(cmd *cobra.Command, result ops.RetentionPolicyResult) error {
	if commandUsesSharedEnvelope(cmd, pickMode()) {
		return renderSucceededResult(cmd, result)
	}
	cmd.Printf("retention policy: %s\n", result.Outcome)
	cmd.Printf("retention days: %d\n", result.Request.RetentionDays)
	if result.Request.DerivedEndDateBoundary != "" {
		cmd.Printf("retention boundary: endDate <= %s\n", result.Request.DerivedEndDateBoundary)
	}
	if filters := result.Discovery.Filters.String(); filters != "" {
		cmd.Printf("selection filters: %s\n", filters)
	}
	if result.Discovery.Status != "" {
		cmd.Printf("retention discovery: %s\n", result.Discovery.Status)
		cmd.Printf("retention seeds: %d\n", result.Discovery.Count)
		if result.Discovery.Count == 0 {
			cmd.Printf("no retention cleanup targets found\n")
		}
	}
	if result.DeletePlan.Status != "" {
		cmd.Printf("delete plan: %s (seeds: %d, roots: %d, affected: %d)\n",
			result.DeletePlan.Status,
			len(result.DeletePlan.SeedKeys),
			len(result.DeletePlan.ResolvedRootKeys),
			len(result.DeletePlan.AffectedKeys),
		)
		if len(result.DeletePlan.DuplicateKeys) > 0 {
			cmd.Printf("duplicate roots: %d\n", len(result.DeletePlan.DuplicateKeys))
		}
		if len(result.DeletePlan.NonFinalAffectedItems) > 0 {
			cmd.Printf("process instances not in final state: %d (use --force to cancel before delete)\n", len(result.DeletePlan.NonFinalAffectedItems))
		}
		if len(result.DeletePlan.MissingAncestors) > 0 {
			cmd.Printf("missing ancestors: %d\n", len(result.DeletePlan.MissingAncestors))
		}
		for _, warning := range result.DeletePlan.TraversalWarnings {
			if warning != "" {
				cmd.Printf("traversal warning: %s\n", warning)
			}
		}
		cmd.Printf("confirmation required: %t\n", result.DeletePlan.RequiresConfirmation)
		if flagVerbose {
			printOpsExecuteRetentionPolicyKeys(cmd, "retention seed keys", result.DeletePlan.SeedKeys)
			printOpsExecuteRetentionPolicyKeys(cmd, "resolved root keys", result.DeletePlan.ResolvedRootKeys)
			printOpsExecuteRetentionPolicyKeys(cmd, "affected process-instance keys", result.DeletePlan.AffectedKeys)
		}
	}
	if result.Deletion.Status != "" {
		if !result.Deletion.Submitted {
			cmd.Printf("deletion: %s; no deletion request submitted\n", result.Deletion.Status)
		} else {
			cmd.Printf("deletion: %s (requests: %d)\n", result.Deletion.Status, len(result.Deletion.Items))
			if result.Deletion.NoWait {
				cmd.Printf("deletion confirmation: skipped (--no-wait)\n")
			} else {
				cmd.Printf("deletion confirmation: %t\n", result.Deletion.Confirmed)
			}
		}
	}
	if result.Outcome != "" {
		if !result.Deletion.Submitted && result.Outcome == ops.RetentionPolicyOutcomePlanned {
			cmd.Printf("outcome: %s; no changes applied\n", result.Outcome)
		} else {
			cmd.Printf("outcome: %s\n", result.Outcome)
		}
	}
	if len(result.Errors) > 0 {
		return fmt.Errorf("%s", result.Errors[0])
	}
	return nil
}

func printOpsExecuteRetentionPolicyKeys(cmd *cobra.Command, label string, keys []string) {
	if len(keys) == 0 {
		cmd.Printf("%s: none\n", label)
		return
	}
	cmd.Printf("%s: %s\n", label, strings.Join(keys, ", "))
}
