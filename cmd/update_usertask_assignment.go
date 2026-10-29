// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
	opts := collectExplicitAdminInputOptions()
	if input.AllowOverride {
		opts = append(opts, foptions.WithUserTaskAssignmentOverride())
	}
	var plan task.UserTaskAssignmentPlan
	protected := input.Operation == "assign" && !input.AllowOverride
	approval := input
	allowReassignment := false
	reassignmentPrompted := false
	if protected {
		plan, err = cli.PrepareUserTaskAssignments(cmd.Context(), input.Keys, input.Assignee, input.WantedWorkers, opts...)
		if err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err, "user-task assignment: could not check current assignees ("+string(ferrors.Classify(err))+")")
			return
		}
		approval.Keys = plan.ReadyKeys
		if plan.BlockedCount > 0 && !shouldImplicitlyConfirm(cmd) && term.IsTerminal(int(os.Stdin.Fd())) {
			for _, replacement := range plan.Replacements {
				log.Warn(fmt.Sprintf("user-task %s is already assigned to %q.", replacement.Key, replacement.Assignee))
			}
			if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, userTaskReassignmentConfirmationPrompt(input, plan)); err != nil {
				handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
				return
			}
			allowReassignment = true
			input.AllowOverride = true
			reassignmentPrompted = true
			approval.Keys = input.Keys
		}
		if !allowReassignment && len(approval.Keys) > 0 && plan.BlockedCount > 0 {
			renderUserTaskAssignmentBlockedPlan(cmd, plan.BlockedCount)
		}
	}
	if !reassignmentPrompted && len(approval.Keys) > 0 && !input.AllowOverride && !shouldImplicitlyConfirm(cmd) {
		renderUserTaskAssignmentPlan(cmd, approval)
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, userTaskAssignmentConfirmationPrompt(approval)); err != nil {
			handleCommandError(cmd, log, cfg.App.NoErrCodes, err)
			return
		}
	}
	var results task.UserTaskAssignmentResults
	var operationErr error
	if protected {
		results, operationErr = cli.ExecuteUserTaskAssignments(cmd.Context(), plan, allowReassignment)
	} else if input.Operation == "assign" {
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
		prompt += fmt.Sprintf(" to %q", input.Assignee)
	}
	return prompt + ". Do you want to proceed?"
}

// userTaskReassignmentConfirmationPrompt replaces the ordinary approval question,
// explicitly authorizing ownership replacement for the frozen selected tasks.
func userTaskReassignmentConfirmationPrompt(input userTaskAssignmentInput, plan task.UserTaskAssignmentPlan) string {
	changedCount := len(input.Keys) - plan.UnchangedCount
	if changedCount == 1 && len(plan.Replacements) == 1 {
		return fmt.Sprintf("Replace assignee %q with %q?", plan.Replacements[0].Assignee, input.Assignee)
	}
	return fmt.Sprintf("Assign %d user tasks to %q, replacing %d existing assignee(s)?", changedCount, input.Assignee, plan.BlockedCount)
}
