// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runUpdateUserTaskCompletion freezes local inputs, previews or confirms once,
// and delegates execution to the facade. The service owns all backend mechanics.
func runUpdateUserTaskCompletion(cmd *cobra.Command, args []string, cli task.API, log *slog.Logger, cfg *config.Config) {
	input, err := parseUpdateUserTaskCompletionInput(cmd, args)
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		return
	}
	attachExplicitKeysTenantContext(cmd, cfg)
	if flagDryRun {
		renderAttachedTenantContext(cmd)
		if err := renderUpdateUserTaskCompletion(cmd, newUserTaskCompletionPreview(input), nil); err != nil {
			log.Error("user-task completion: could not write preview")
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		return
	}
	plan, err := cli.PrepareUserTaskCompletions(cmd.Context(), input.Keys, input.Variables, input.WantedWorkers, collectExplicitAdminInputOptions()...)
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		return
	}
	if !shouldImplicitlyConfirm(cmd) && (plan.ReadyCount > 0 || plan.CompletedCount < len(input.Keys)) {
		renderUserTaskCompletionPlan(cmd, input, plan)
	}
	if plan.UnassignedCount > 0 {
		log.Warn(formatUserTaskCompletionAssigneeWarning(len(input.Keys), plan.UnassignedCount))
	}
	if plan.UncheckedCount > 0 {
		log.Warn(formatUserTaskCompletionLookupWarning(plan.UncheckedCount))
	}
	if plan.ReadyCount > 0 && !shouldImplicitlyConfirm(cmd) {
		prompt := userTaskCompletionConfirmationPrompt(input, plan.ReadyCount)
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, prompt); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
	}
	results, operationErr := cli.ExecuteUserTaskCompletions(cmd.Context(), plan)
	report := newUserTaskCompletionReport(input, results)
	if err := renderUpdateUserTaskCompletion(cmd, report, operationErr); err != nil {
		// Rendering may already have written part of an envelope. Never render again,
		// and retain the original operation class when a partial result also failed.
		log.Error("user-task completion: could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatUserTaskCompletionFailure(report, operationErr))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}

// userTaskCompletionConfirmationPrompt identifies a single selected task or the eligible bulk count.
func userTaskCompletionConfirmationPrompt(input userTaskCompletionInput, ready int) string {
	if len(input.Keys) == 1 {
		return fmt.Sprintf("You are about to update user-task %s. Do you want to proceed?", input.Keys[0])
	}
	if ready == 1 {
		return "You are about to complete 1 user task. Do you want to proceed?"
	}
	return fmt.Sprintf("You are about to complete %d user tasks. Do you want to proceed?", ready)
}
