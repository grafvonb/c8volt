// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runOpsSlowProcessAnalysisBusinessID retains partial evidence and one configured diagnostic on failure.
func runOpsSlowProcessAnalysisBusinessID(cmd *cobra.Command, cli ops.API, log *slog.Logger, cfg *config.Config, request ops.SlowProcessAnalysisRequest) {
	progress := configureOpsSlowProcessAnalysisPreflight(cmd, &request)
	result, operationErr := cli.AnalyseSlowProcessInstances(cmd.Context(), request, collectOptions()...)
	progress.Close()
	if err := renderBusinessIDSlowAnalysis(cmd, result, operationErr); err != nil {
		log.Error("analyse slow process instances: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			selected := 0
			if result.Selection != nil {
				selected = len(result.Selection.Keys)
			}
			analysed := result.Count
			if result.FrozenScopeProgress != nil {
				analysed = result.FrozenScopeProgress.Done
			}
			message := fmt.Sprintf("analyse slow process instances: selected %d, analysed %d; %s", selected, analysed, boundedCommandFailureReason(operationErr))
			if errors.Is(operationErr, ferrors.ErrUnsupported) {
				message += "; business-ID search requires Camunda 8.10"
			}
			log.Error(message)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}
