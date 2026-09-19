// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"errors"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestUserTasksView_RendersContractModes pins aligned optional cells, display
// identities, labelled details, collection JSON, keys, and quiet precedence.
func TestUserTasksView_RendersContractModes(t *testing.T) {
	result := task.UserTasks{Total: 2, Items: []task.UserTask{
		{Key: "2251799815391233", State: "CREATED", Name: "Approve invoice", ElementId: "approve_invoice", Assignee: "alice", ProcessInstanceKey: "2251799813711967", TenantId: "tenant-a", ProcessDefinitionId: "invoice", ProcessDefinitionKey: "2251799813689000", ElementInstanceKey: "2251799815391200"},
		{Key: "2251799815391234", State: "COMPLETED", ElementId: "archive_invoice", ProcessInstanceKey: "2251799813711968", TenantId: "tenant-b", ProcessDefinitionId: "invoice", ProcessDefinitionKey: "2251799813689000", ElementInstanceKey: "2251799815391200"},
	}}

	for _, test := range []struct {
		name     string
		json     bool
		keys     bool
		quiet    bool
		want     string
		contains []string
	}{
		{name: "human", want: "2251799815391233 tenant-a approve_invoice CREATED   pi:2251799813711967 ei:2251799815391200 pd:2251799813689000 assignee:alice\n2251799815391234 tenant-b archive_invoice COMPLETED pi:2251799813711968 ei:2251799815391200 pd:2251799813689000 assignee:<unassigned>\nfound: 2\n"},
		{name: "keys", keys: true, want: "2251799815391233\n2251799815391234\n"},
		{name: "quiet", quiet: true, want: ""},
		{name: "quiet keys", keys: true, quiet: true, want: "2251799815391233\n2251799815391234\n"},
		{name: "json precedence", json: true, keys: true, quiet: true, contains: []string{`"outcome": "succeeded"`, `"command": "user-task"`, `"total": 2`, `"items"`}},
	} {
		t.Run(test.name, func(t *testing.T) {
			resetGetUserTaskGlobalModes(t)
			flagViewAsJson, flagViewKeysOnly, flagQuiet = test.json, test.keys, test.quiet
			var output bytes.Buffer
			cmd := &cobra.Command{Use: "user-task"}
			cmd.SetOut(&output)
			setContractSupport(cmd, ContractSupportFull)

			require.NoError(t, userTasksView(cmd, result))
			if test.contains == nil {
				require.Equal(t, test.want, output.String())
			}
			for _, fragment := range test.contains {
				require.Contains(t, output.String(), fragment)
			}
		})
	}
}

// TestUserTasksView_PropagatesWriterErrors prevents truncated human and key
// output from being reported as a successful command result.
func TestUserTasksView_PropagatesWriterErrors(t *testing.T) {
	result := task.UserTasks{Total: 1, Items: []task.UserTask{{Key: "2251799815391233", State: "CREATED", ProcessInstanceKey: "2251799813711967"}}}
	for _, keysOnly := range []bool{false, true} {
		resetGetUserTaskGlobalModes(t)
		flagViewKeysOnly = keysOnly
		cmd := &cobra.Command{Use: "user-task"}
		cmd.SetOut(failingUserTaskWriter{})
		require.ErrorIs(t, userTasksView(cmd, result), errUserTaskWriter)
	}
}

// TestUserTaskTotalView_PropagatesWriterErrors keeps numeric count output from
// being reported as successful after a truncated write.
func TestUserTaskTotalView_PropagatesWriterErrors(t *testing.T) {
	cmd := &cobra.Command{Use: "user-task"}
	cmd.SetOut(failingUserTaskWriter{})
	require.ErrorIs(t, userTaskTotalView(cmd, 42), errUserTaskWriter)
}

var errUserTaskWriter = errors.New("writer failed")

type failingUserTaskWriter struct{}

func (failingUserTaskWriter) Write([]byte) (int, error) { return 0, errUserTaskWriter }

// resetGetUserTaskGlobalModes restores shared output flags after direct view tests.
func resetGetUserTaskGlobalModes(t *testing.T) {
	t.Helper()
	previousJSON, previousKeys, previousQuiet, previousWithVars, previousValueLimit, previousPIValueLimit := flagViewAsJson, flagViewKeysOnly, flagQuiet, flagGetUserTaskWithVars, flagGetUserTaskVarValueLimit, flagGetPIVarValueLimit
	t.Cleanup(func() {
		flagViewAsJson, flagViewKeysOnly, flagQuiet, flagGetUserTaskWithVars = previousJSON, previousKeys, previousQuiet, previousWithVars
		flagGetUserTaskVarValueLimit = previousValueLimit
		flagGetPIVarValueLimit = previousPIValueLimit
	})
	flagViewAsJson, flagViewKeysOnly, flagQuiet = false, false, false
	flagGetUserTaskWithVars = false
	flagGetUserTaskVarValueLimit = 0
}

// TestUserTasksView_CompletesIncrementalOutput verifies page rendering and final
// summaries share the collected view without duplicating rows or keys.
func TestUserTasksView_CompletesIncrementalOutput(t *testing.T) {
	for _, keysOnly := range []bool{false, true} {
		resetGetUserTaskGlobalModes(t)
		flagViewKeysOnly = keysOnly
		items := []task.UserTask{
			{Key: "2251799815391233", State: "CREATED"},
			{Key: "2251799815391234", State: "CREATED"},
		}
		var incremental, collected bytes.Buffer
		command := &cobra.Command{Use: "user-task"}
		command.SetOut(&incremental)
		for _, item := range items {
			require.NoError(t, renderUserTaskSearchPage(command, []task.UserTask{item}))
		}
		require.NoError(t, userTasksView(command, task.UserTasks{Total: 2}))
		command.SetOut(&collected)
		require.NoError(t, userTasksView(command, task.UserTasks{Total: 2, Items: items}))
		require.Equal(t, collected.String(), incremental.String())
		command.SetOut(failingUserTaskWriter{})
		if keysOnly {
			require.ErrorIs(t, renderUserTaskSearchPage(command, items), errUserTaskWriter)
		} else {
			require.ErrorIs(t, userTasksView(command, task.UserTasks{Total: 2}), errUserTaskWriter)
		}
	}
}

// TestFlatRowUserTask_UsesGetGrammar verifies the reported task keeps its name,
// technical identities, and related keys in the agreed human-output order.
func TestFlatRowUserTask_UsesGetGrammar(t *testing.T) {
	item := task.UserTask{
		Key: "2251799813900041", TenantId: "tenant-a", ProcessDefinitionId: "C89_SimpleUserTask",
		ElementId: "SimpleUserTask_UserTask", State: "CREATED", Name: "Simple User Task",
		ProcessInstanceKey: "2251799813900036", ElementInstanceKey: "2251799813900040", ProcessDefinitionKey: "2251799813873873",
	}
	require.Equal(t, "2251799813900041 tenant-a SimpleUserTask_UserTask CREATED pi:2251799813900036 ei:2251799813900040 pd:2251799813873873 assignee:<unassigned>", compactFlatRow(flatRowUserTask(item)))
}
