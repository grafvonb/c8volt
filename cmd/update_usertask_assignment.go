// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"

	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
)

// runUpdateUserTaskAssignment freezes local intent and approves it once. Native
// execution, retries and scheduling belong to the service; previews never call it.
func runUpdateUserTaskAssignment(cmd *cobra.Command, args []string, cli task.API, log *slog.Logger, cfg *config.Config) {
	input, err := parseUpdateUserTaskAssignmentInput(cmd, args)
	if err != nil {
		handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
		return
	}
	attachExplicitKeysTenantContext(cmd, cfg)
	label := assignmentOperationLabel(input.Operation)
	if flagDryRun {
		if flagVerbose {
			renderAttachedTenantContext(cmd)
		}
		if err := renderUpdateUserTaskAssignment(cmd, newUserTaskAssignmentPreview(input), nil); err != nil {
			log.Error("user-task " + label + ": could not write preview")
			exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		}
		return
	}
	if !shouldImplicitlyConfirm(cmd) {
		renderUserTaskAssignmentPlan(cmd, input)
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, userTaskAssignmentConfirmationPrompt(input)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
	}
	opts := collectExplicitAdminInputOptions()
	if input.AllowOverride {
		opts = append(opts, foptions.WithUserTaskAssignmentOverride())
	}
	var results task.UserTaskAssignmentResults
	var operationErr error
	if input.Operation == "assign" {
		results, operationErr = cli.AssignUserTasks(cmd.Context(), input.Keys, input.Assignee, input.WantedWorkers, opts...)
	} else {
		results, operationErr = cli.UnassignUserTasks(cmd.Context(), input.Keys, input.WantedWorkers, opts...)
	}
	report := newUserTaskAssignmentReport(input, results)
	if operationErr != nil && renderUserTaskReassignmentDenied(cmd, report) {
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		return
	}
	if err := renderUpdateUserTaskAssignment(cmd, report, operationErr); err != nil {
		// A partial write must never trigger another envelope. Preserve the mutation's
		// original classification when rendering also fails.
		log.Error("user-task " + label + ": could not write result")
		if operationErr != nil {
			exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, err)
		return
	}
	if operationErr != nil {
		if pickMode() != RenderModeJSON {
			log.Error(formatUserTaskAssignmentFailure(report, operationErr))
		}
		exitAfterRenderedResult(cfg.App.NoErrCodes, operationErr)
	}
}

// userTaskAssignmentConfirmationPrompt names the frozen scope and safely quotes
// opaque identity bytes without granting replacement permission through approval.
func userTaskAssignmentConfirmationPrompt(input userTaskAssignmentInput) string {
	scope := fmt.Sprintf("%d user tasks", len(input.Keys))
	if len(input.Keys) == 1 {
		scope = "user-task " + input.Keys[0]
	}
	prompt := "You are about to " + input.Operation + " " + scope
	if input.Operation == "assign" {
		permission := "not allowed"
		if input.AllowOverride {
			permission = "allowed"
		}
		prompt += fmt.Sprintf(" to %q (reassignment: %s)", input.Assignee, permission)
	}
	return prompt + ". Do you want to proceed?"
}
