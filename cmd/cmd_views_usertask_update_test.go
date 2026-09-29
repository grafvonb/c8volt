// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// userTaskUpdateFailingWriter makes destination errors deterministic in view tests.
type userTaskUpdateFailingWriter struct{}

// Write always fails before accepting bytes.
func (userTaskUpdateFailingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}

// TestUserTaskVariableUpdatePreviewJSONContract verifies the public preview
// shape, category null semantics, envelope cardinality, and internal omissions.
func TestUserTaskVariableUpdatePreviewJSONContract(t *testing.T) {
	restore := setUserTaskUpdateViewFlags(t, true, true, true, false, false)
	defer restore()

	cmd, output := newUserTaskUpdateViewCommand()
	plan := userTaskUpdateViewPlan()
	require.NoError(t, renderUpdateUserTaskVariablePreview(cmd, plan))

	decoder := json.NewDecoder(bytes.NewBufferString(output.String()))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	require.Equal(t, "succeeded", envelope["outcome"])
	require.Equal(t, "update user-task", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, "update", payload["operation"])
	require.Equal(t, false, payload["mutationSubmitted"])
	require.NotContains(t, payload, "targets")
	require.NotContains(t, payload, "tenantContext")
	items := requireJSONItems(t, payload["userTasks"], 1)
	item := requireJSONObject(t, items[0])
	require.Nil(t, item["additions"], "nil plan categories remain JSON null")
	require.Equal(t, []any{}, item["unchangedRequested"])
}

// TestUserTaskVariableUpdateHumanPlanContract locks down PI-style tokens,
// inherited scope evidence, compact bulk output, and dry-run wording.
func TestUserTaskVariableUpdateHumanPlanContract(t *testing.T) {
	restore := setUserTaskUpdateViewFlags(t, false, false, false, false, false)
	defer restore()

	t.Run("single task details", func(t *testing.T) {
		cmd, output := newUserTaskUpdateViewCommand()
		require.NoError(t, renderUpdateUserTaskVariablePreview(cmd, userTaskUpdateViewPlan()))
		require.Equal(t,
			"dry run: update user-task variables: 1 user task(s), 1 change(s), 0 addition(s), 0 unchanged, 1 untouched; no changes applied\n"+
				"task-a: ~ approved (inherited scope scope-root): false -> true; = note: \"keep\"\n",
			output.String(),
		)
	})

	t.Run("bulk inherited evidence", func(t *testing.T) {
		cmd, output := newUserTaskUpdateViewCommand()
		plan := userTaskUpdateViewPlan()
		second := plan.UserTasks[0]
		second.UserTaskKey = "task-b"
		plan.UserTasks = append(plan.UserTasks, second)
		plan.RequestedKeys = append(plan.RequestedKeys, "task-b")
		plan.RequestedCount = 2
		plan.UpdateCount = 2
		plan.VariableChangeCount = 2
		require.NoError(t, renderUpdateUserTaskVariablePreview(cmd, plan))
		require.Equal(t,
			"dry run: update user-task variables: 2 user task(s), 2 change(s), 0 addition(s), 0 unchanged, 1 untouched; no changes applied\n"+
				"inherited target scopes: scope-root\n",
			output.String(),
		)
	})
}

// TestUserTaskVariableUpdateKeysContract verifies changed-task selection,
// successful-result selection, precedence, and zero-byte no-work output.
func TestUserTaskVariableUpdateKeysContract(t *testing.T) {
	restore := setUserTaskUpdateViewFlags(t, false, true, false, false, false)
	defer restore()

	t.Run("preview changed keys", func(t *testing.T) {
		cmd, output := newUserTaskUpdateViewCommand()
		plan := userTaskUpdateViewPlan()
		plan.UserTasks = append(plan.UserTasks, task.UserTaskVariablePlan{UserTaskKey: "task-unchanged"})
		require.NoError(t, renderUpdateUserTaskVariablePreview(cmd, plan))
		require.Equal(t, "task-a\n", output.String())
	})

	t.Run("execution successful keys", func(t *testing.T) {
		cmd, output := newUserTaskUpdateViewCommand()
		results := task.UserTaskVariableUpdateResults{Items: []task.UserTaskVariableUpdateResult{
			{Key: "confirmed", Status: task.UserTaskVariableUpdateStatusConfirmed},
			{Key: "submitted", Status: task.UserTaskVariableUpdateStatusSubmitted},
			{Key: "unchanged", Status: task.UserTaskVariableUpdateStatusUnchanged},
			{Key: "failed", Status: task.UserTaskVariableUpdateStatusMutationFailed},
		}}
		require.NoError(t, renderUpdateUserTaskVariableResults(cmd, results))
		require.Equal(t, "confirmed\nsubmitted\n", output.String())
	})

	t.Run("no work", func(t *testing.T) {
		cmd, output := newUserTaskUpdateViewCommand()
		require.NoError(t, renderUpdateUserTaskVariablePlan(cmd, task.UserTaskVariableUpdatePlan{}))
		require.Empty(t, output.String())
	})
}

// TestUserTaskVariableUpdateResultContracts verifies accepted versus succeeded
// envelopes and truthful human state aggregation including verbose scopes.
func TestUserTaskVariableUpdateResultContracts(t *testing.T) {
	results := task.UserTaskVariableUpdateResults{Items: []task.UserTaskVariableUpdateResult{
		{Key: "confirmed", Status: task.UserTaskVariableUpdateStatusConfirmed, Scopes: []task.ScopeVariableUpdateOutcome{{ScopeKey: "scope-a", Status: task.ScopeVariableUpdateStatusSubmitted}}},
		{Key: "unchanged", Status: task.UserTaskVariableUpdateStatusUnchanged},
		{Key: "failed", Status: task.UserTaskVariableUpdateStatusMutationFailed, Error: "denied"},
		{Key: "skipped", Status: task.UserTaskVariableUpdateStatusSkipped},
	}}

	t.Run("human totals and scopes", func(t *testing.T) {
		restore := setUserTaskUpdateViewFlags(t, false, false, false, true, false)
		defer restore()
		cmd, output := newUserTaskUpdateViewCommand()
		require.NoError(t, renderUpdateUserTaskVariableResults(cmd, results))
		require.Equal(t,
			"updated user-task confirmed: confirmed; scopes: scope-a=submitted\n"+
				"updated user-task unchanged: unchanged\n"+
				"updated user-task failed: mutation failed: denied\n"+
				"updated user-task skipped: skipped\n"+
				"updated: 4 (confirmed/submitted: 1, unchanged: 1, failed: 1, skipped: 1)\n",
			output.String(),
		)
	})

	for _, test := range []struct {
		name        string
		noWait      bool
		wantOutcome string
	}{
		{name: "confirmed succeeds", wantOutcome: "succeeded"},
		{name: "no wait accepts", noWait: true, wantOutcome: "accepted"},
	} {
		t.Run(test.name, func(t *testing.T) {
			restore := setUserTaskUpdateViewFlags(t, true, false, false, false, test.noWait)
			defer restore()
			cmd, output := newUserTaskUpdateViewCommand()
			require.NoError(t, renderUpdateUserTaskVariableResults(cmd, results))
			var envelope map[string]any
			require.NoError(t, json.Unmarshal(output.Bytes(), &envelope))
			require.Equal(t, test.wantOutcome, envelope["outcome"])
		})
	}
}

// TestUserTaskVariableUpdateFailureEnvelope verifies partial outcomes and the
// normalized error share one envelope instead of producing a second render.
func TestUserTaskVariableUpdateFailureEnvelope(t *testing.T) {
	restore := setUserTaskUpdateViewFlags(t, true, false, false, false, false)
	defer restore()

	cmd, output := newUserTaskUpdateViewCommand()
	results := task.UserTaskVariableUpdateResults{Items: []task.UserTaskVariableUpdateResult{{Key: "task-a", Status: task.UserTaskVariableUpdateStatusMutationFailed, Error: "denied"}}}
	require.NoError(t, renderUpdateUserTaskVariableFailure(cmd, results, errors.New("scope update failed")))

	decoder := json.NewDecoder(bytes.NewReader(output.Bytes()))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	require.Equal(t, "failed", envelope["outcome"])
	require.Equal(t, "internal error: scope update failed", requireJSONObject(t, envelope["detail"])["message"])
	require.Len(t, requireJSONObject(t, envelope["payload"])["items"], 1)
}

// TestUserTaskVariableUpdateWriterFailures verifies structured and key output
// failures are returned to command dispatch without attempting another write.
func TestUserTaskVariableUpdateWriterFailures(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		restore := setUserTaskUpdateViewFlags(t, true, false, false, false, false)
		defer restore()
		cmd, _ := newUserTaskUpdateViewCommand()
		cmd.SetOut(userTaskUpdateFailingWriter{})
		require.EqualError(t, renderUpdateUserTaskVariablePreview(cmd, userTaskUpdateViewPlan()), "write failed")
	})

	t.Run("keys", func(t *testing.T) {
		restore := setUserTaskUpdateViewFlags(t, false, true, false, false, false)
		defer restore()
		cmd, _ := newUserTaskUpdateViewCommand()
		cmd.SetOut(userTaskUpdateFailingWriter{})
		require.EqualError(t, renderUpdateUserTaskVariablePreview(cmd, userTaskUpdateViewPlan()), "write failed")
	})
}

// newUserTaskUpdateViewCommand creates a full-contract command whose path and
// output destination match the production renderer without global execution.
func newUserTaskUpdateViewCommand() (*cobra.Command, *bytes.Buffer) {
	root := &cobra.Command{Use: "c8volt"}
	update := &cobra.Command{Use: "update"}
	cmd := &cobra.Command{Use: "user-task"}
	root.AddCommand(update)
	update.AddCommand(cmd)
	setCommandMutation(cmd, CommandMutationStateChanging)
	setContractSupport(cmd, ContractSupportFull)
	output := new(bytes.Buffer)
	cmd.SetOut(output)
	return cmd, output
}

// userTaskUpdateViewPlan supplies stable local and inherited evidence for view tests.
func userTaskUpdateViewPlan() task.UserTaskVariableUpdatePlan {
	return task.UserTaskVariableUpdatePlan{
		RequestedKeys:          []string{"task-a"},
		RequestedCount:         1,
		UpdateCount:            1,
		VariableChangeCount:    1,
		VariableUntouchedCount: 1,
		UserTasks: []task.UserTaskVariablePlan{{
			UserTaskKey: "task-a",
			Changes: []task.UserTaskVariablePlannedChange{{
				Name: "approved", ScopeKey: "scope-root", Inherited: true, Before: false, After: true,
			}},
			UnchangedRequested: []task.UserTaskVariablePlannedValue{},
			Untouched:          []task.UserTaskVariablePlannedValue{{Name: "note", ScopeKey: "scope-root", Inherited: true, Value: "keep"}},
			TargetScopeKeys:    []string{"scope-root"},
		}},
		Targets: []task.ScopeVariableUpdateTarget{{ScopeKey: "scope-root", Variables: map[string]any{"approved": true}}},
	}
}

// setUserTaskUpdateViewFlags isolates package-global output controls used by views.
func setUserTaskUpdateViewFlags(t *testing.T, jsonMode, keysOnly, quiet, verbose, noWait bool) func() {
	t.Helper()
	previousJSON, previousKeys := flagViewAsJson, flagViewKeysOnly
	previousQuiet, previousVerbose, previousNoWait := flagQuiet, flagVerbose, flagNoWait
	flagViewAsJson, flagViewKeysOnly = jsonMode, keysOnly
	flagQuiet, flagVerbose, flagNoWait = quiet, verbose, noWait
	return func() {
		flagViewAsJson, flagViewKeysOnly = previousJSON, previousKeys
		flagQuiet, flagVerbose, flagNoWait = previousQuiet, previousVerbose, previousNoWait
	}
}
