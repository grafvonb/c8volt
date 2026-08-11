// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// renderOpsSlowProcessAnalysisJSONResult preserves the structured automation-visible slow-analysis payload.
func renderOpsSlowProcessAnalysisJSONResult(cmd *cobra.Command, result ops.SlowProcessAnalysisResult) error {
	return renderJSONPayload(cmd, RenderModeJSON, result)
}

// renderOpsSlowProcessAnalysisKeysOnlyResult prints each unique process-instance key once.
func renderOpsSlowProcessAnalysisKeysOnlyResult(cmd *cobra.Command, result ops.SlowProcessAnalysisResult) {
	seen := map[string]struct{}{}
	for _, item := range result.Items {
		if item.Key == "" {
			continue
		}
		if _, ok := seen[item.Key]; ok {
			continue
		}
		seen[item.Key] = struct{}{}
		renderOutputLine(cmd, "%s", item.Key)
	}
}
