// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
)

// searchUserTasksWithPaging observes service-owned traversal while keeping
// incremental result output and interactive control text on separate streams.
func searchUserTasksWithPaging(cmd *cobra.Command, cli task.API, request task.SearchRequest) (task.UserTasks, bool, error) {
	incremental := shouldRenderUserTaskSearchIncrementally(cmd)
	result, err := cli.SearchUserTasksPages(cmd.Context(), request, func(step task.SearchPageStep) (task.SearchPageAction, error) {
		if incremental {
			if err := renderUserTaskSearchPage(cmd, step.Page.Items); err != nil {
				return task.SearchPageActionStop, err
			}
		}
		if step.LimitReached || step.Page.ContinuationState == task.ContinuationStateNoMore {
			return task.SearchPageActionContinue, nil
		}
		if step.Page.ContinuationState == task.ContinuationStateIndeterminate || len(step.Page.Items) == 0 || shouldAutoContinueUserTaskSearchPages(cmd) {
			return task.SearchPageActionContinue, nil
		}
		prompt := fmt.Sprintf("Fetched %d user task(s) on this page (%d loaded). More matching user tasks remain. Continue?", len(step.Page.Items), step.CumulativeCount)
		if err := confirmCmdOrAbortFn(cmd.ErrOrStderr(), false, prompt); err != nil {
			if isCmdAborted(err) {
				return task.SearchPageActionStop, nil
			}
			return task.SearchPageActionStop, err
		}
		return task.SearchPageActionContinue, nil
	}, collectOptions()...)
	if err != nil {
		return task.UserTasks{}, false, err
	}
	if incremental {
		if err := userTasksView(cmd, task.UserTasks{Total: result.Total}); err != nil {
			return task.UserTasks{}, false, err
		}
		return task.UserTasks{}, true, nil
	}
	return task.UserTasks{Total: result.Total, Items: result.Items}, false, nil
}

// shouldAutoContinueUserTaskSearchPages keeps collected JSON and explicitly
// unattended modes free of paging prompts while preserving terminal paging.
func shouldAutoContinueUserTaskSearchPages(cmd *cobra.Command) bool {
	return pickMode() == RenderModeJSON || shouldImplicitlyConfirm(cmd)
}

// shouldRenderUserTaskSearchIncrementally preserves one-result-at-a-time human
// and key output while collected modes emit one complete payload.
func shouldRenderUserTaskSearchIncrementally(cmd *cobra.Command) bool {
	if flagCmdAutoConfirm || flagQuiet {
		return false
	}
	mode := pickMode()
	return mode == RenderModeOneLine || mode == RenderModeKeysOnly
}
