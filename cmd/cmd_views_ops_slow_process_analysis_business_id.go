// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/spf13/cobra"
)

// renderBusinessIDSlowAnalysis preserves partial analysis without presenting failed discovery as an empty success.
func renderBusinessIDSlowAnalysis(cmd *cobra.Command, result ops.SlowProcessAnalysisResult, operationErr error) error {
	if pickMode() == RenderModeJSON {
		if operationErr != nil {
			envelope := resultEnvelopeForError(cmd, operationErr)
			envelope.Payload = result
			return renderResultEnvelope(cmd, envelope)
		}
		return renderSucceededResult(cmd, result)
	}
	if result.Count == 0 && (operationErr != nil || flagQuiet) {
		return nil
	}
	return renderOpsSlowProcessAnalysisResult(cmd, result)
}
