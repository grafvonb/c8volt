// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestVariableEnrichedUserTasksViewPreservesTaskRowsAndEmptyVariables verifies
// nested effective values, backend truncation labels, and omitted empty subtrees.
func TestVariableEnrichedUserTasksViewPreservesTaskRowsAndEmptyVariables(t *testing.T) {
	resetGetUserTaskGlobalModes(t)
	result := task.VariableEnrichedUserTasks{Total: 2, Items: []task.VariableEnrichedUserTask{
		{
			Item:      task.UserTask{Key: "1", TenantId: "tenant-a", ElementId: "approve", State: "CREATED", ProcessInstanceKey: "11", ElementInstanceKey: "12", ProcessDefinitionKey: "13", Assignee: "alice"},
			Variables: []task.UserTaskVariable{{Name: "payload", Value: "{ \"amount\": 120 }", ProcessInstanceKey: "11", ScopeKey: "12", APITruncated: true}},
		},
		{
			Item:      task.UserTask{Key: "2", TenantId: "tenant-a", ElementId: "archive", State: "COMPLETED", ProcessInstanceKey: "21", ElementInstanceKey: "22", ProcessDefinitionKey: "23"},
			Variables: []task.UserTaskVariable{},
		},
	}}
	var stdout bytes.Buffer
	cmd := &cobra.Command{Use: "user-task"}
	cmd.SetOut(&stdout)

	require.NoError(t, variableEnrichedUserTasksView(cmd, result))
	require.Equal(t, ""+
		"1 tenant-a approve CREATED   pi:11 ei:12 pd:13 assignee:alice\n"+
		"└─ vars:\n"+
		"   └─ payload={\"amount\":120} [api-truncated]\n"+
		"2 tenant-a archive COMPLETED pi:21 ei:22 pd:23 assignee:<unassigned>\n"+
		"found: 2\n", stdout.String())
}

// TestVariableEnrichedUserTasksViewJSONKeepsInitializedCollections verifies one
// envelope with explicit items and variables arrays and no process age metadata.
func TestVariableEnrichedUserTasksViewJSONKeepsInitializedCollections(t *testing.T) {
	resetGetUserTaskGlobalModes(t)
	flagViewAsJson = true
	var stdout bytes.Buffer
	cmd := &cobra.Command{Use: "user-task"}
	cmd.SetOut(&stdout)
	setContractSupport(cmd, ContractSupportFull)
	result := task.VariableEnrichedUserTasks{Total: 1, Items: []task.VariableEnrichedUserTask{{
		Item: task.UserTask{Key: "1", State: "CREATED", ProcessInstanceKey: "11"}, Variables: []task.UserTaskVariable{},
	}}}

	require.NoError(t, variableEnrichedUserTasksView(cmd, result))
	decoder := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	payload := envelope["payload"].(map[string]any)
	enriched := payload["items"].([]any)[0].(map[string]any)
	require.Equal(t, []any{}, enriched["variables"])
	require.NotContains(t, payload, "meta")
}

// TestVariableEnrichedUserTasksViewPropagatesWriterErrors prevents incomplete
// enriched task trees from being reported as successful output.
func TestVariableEnrichedUserTasksViewPropagatesWriterErrors(t *testing.T) {
	resetGetUserTaskGlobalModes(t)
	cmd := &cobra.Command{Use: "user-task"}
	cmd.SetOut(failingUserTaskWriter{})
	result := task.VariableEnrichedUserTasks{Total: 1, Items: []task.VariableEnrichedUserTask{{
		Item: task.UserTask{Key: "1", State: "CREATED", ProcessInstanceKey: "11"}, Variables: []task.UserTaskVariable{},
	}}}
	require.ErrorIs(t, variableEnrichedUserTasksView(cmd, result), errUserTaskWriter)
}
