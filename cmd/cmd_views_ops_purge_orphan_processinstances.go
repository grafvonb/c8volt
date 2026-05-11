// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"strings"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

func renderOpsPurgeOrphanProcessInstancesResult(cmd *cobra.Command, result ops.OrphanPurgeResult) error {
	switch pickMode() {
	case RenderModeJSON:
		return renderSucceededResult(cmd, result)
	case RenderModeKeysOnly:
		for _, key := range result.Discovery.Keys {
			renderOutputLine(cmd, "%s", key)
		}
		return nil
	default:
		return renderOpsPurgeOrphanProcessInstancesHuman(cmd, result)
	}
}

func renderOpsPurgeOrphanProcessInstancesHuman(cmd *cobra.Command, result ops.OrphanPurgeResult) error {
	renderHumanLine(cmd, "dry run: purge orphan process-instances")
	if result.Discovery.Count == 0 {
		renderHumanLine(cmd, "discovered orphan process instances: 0")
		renderHumanLine(cmd, "delete plan: skipped")
		renderHumanLine(cmd, "outcome: planned; no changes applied")
		return nil
	}
	renderHumanLine(cmd, "selection filters: %s", result.Discovery.Filters.String())
	renderHumanLine(cmd, "discovered orphan process instances: %d", result.Discovery.Count)
	renderHumanLine(cmd, "discovered keys: %s", strings.Join(result.Discovery.Keys, ", "))
	renderHumanLine(cmd, "delete plan: %s (requested: %d, roots: %d, affected: %d)",
		result.DeletionPlan.Status,
		len(result.DeletionPlan.RequestedKeys),
		len(result.DeletionPlan.RootKeys),
		len(result.DeletionPlan.AffectedKeys),
	)
	if result.DeletionPlan.DryRunPreview.Warning != "" {
		renderHumanWarningLine(cmd, "%s", result.DeletionPlan.DryRunPreview.Warning)
	}
	renderHumanLine(cmd, "deletion: %s; no deletion request submitted", result.Deletion.Status)
	renderHumanLine(cmd, "outcome: %s; no changes applied", result.Outcome)
	return nil
}
