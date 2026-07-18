// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// renderOpsSlowProcessAnalysisResult dispatches slow-analysis output through the shared render modes.
func renderOpsSlowProcessAnalysisResult(cmd *cobra.Command, result ops.SlowProcessAnalysisResult) error {
	switch pickMode() {
	case RenderModeJSON:
		return renderJSONPayload(cmd, RenderModeJSON, result)
	case RenderModeKeysOnly:
		for _, item := range result.Items {
			renderOutputLine(cmd, "%s", item.Key)
		}
	default:
		renderOutputLine(cmd, "process instances: %d", result.Count)
	}
	return nil
}
