// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/typex"
	"github.com/spf13/cobra"
)

// runUpdateElementVariables plans the entire selection before approval and delegates
// the frozen plan exactly once. No renderer or command owns backend traversal or waits.
func runUpdateElementVariables(cmd *cobra.Command, cli element.API, log *slog.Logger, cfg *config.Config, keys typex.Keys, variables map[string]any) {
	stopActivity := startCommandActivity(cmd, fmt.Sprintf("preparing update variable plan for %d element(s)", len(keys)))
	plan, err := cli.PlanElementVariableUpdates(cmd.Context(), keys, variables, collectExplicitAdminInputOptions()...)
	stopActivity()
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "plan element variable update: "+boundedCommandFailureReason(err))
		return
	}
	attachTenantContext(cmd, userTaskUpdateTenantContext(plan.TenantContext))
	if flagDryRun || plan.UpdateCount == 0 {
		if err := renderElementVariablePreview(cmd, plan); err != nil {
			log.Error("element variable update: could not write preview")
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		return
	}
	if !shouldImplicitlyConfirm(cmd) {
		// Confirmation previews never emit machine keys before execution.
		if pickMode() == RenderModeOneLine {
			renderElementVariablePlanHuman(cmd, elementVariablePreviewFromPlan(plan), "plan")
		}
		prompt := fmt.Sprintf("You are about to update %d requested variable value(s) on %d element(s). Do you want to proceed?", plan.VariableAddCount+plan.VariableChangeCount, plan.UpdateCount)
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, prompt); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
	}
	results, operationErr := cli.ExecuteElementVariableUpdates(cmd.Context(), plan, flagWorkers, collectExplicitAdminInputOptions()...)
	if err := renderElementVariableResults(cmd, results, operationErr); err != nil {
		// A partly written envelope must never invoke a second renderer.
		log.Error("element variable update: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatElementVariableFailure(results, operationErr))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}
