// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"fmt"
	"sync"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type assignmentPlanAPI struct {
	*assignmentAPI
	read func(context.Context, string, ...services.CallOption) (d.UserTask, error)
}

func (a *assignmentPlanAPI) GetNativeUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
	return a.read(ctx, key, opts...)
}

func TestPrepareUserTaskAssignments(t *testing.T) {
	for _, mode := range []string{"mixed", "fail-fast", "same", "override", "dry-run", "read failure", "malformed", "cancel", "race"} {
		t.Run(mode, func(t *testing.T) {
			keys := completionKeys(3)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			var mu sync.Mutex
			reads, submitted := []string{}, []string{}
			api := &assignmentPlanAPI{assignmentAPI: &assignmentAPI{submit: func(_ context.Context, key, assignee string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
				mu.Lock()
				defer mu.Unlock()
				submitted = append(submitted, key)
				require.Equal(t, "alice", assignee)
				require.Equal(t, mode == "override", services.ApplyCallOptions(opts).UserTaskAssignmentOverride)
				if mode == "race" {
					return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusFailed, Submission: d.UserTaskAssignmentSubmissionRejected, StatusCode: 409}, d.ErrConflict
				}
				return acceptedAssignment(key), nil
			}}}
			api.read = func(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
				mu.Lock()
				defer mu.Unlock()
				reads = append(reads, key)
				if mode == "read failure" {
					return d.UserTask{}, fmt.Errorf("%w: private transport evidence", d.ErrForbidden)
				}
				if mode == "malformed" {
					return d.UserTask{}, nil
				}
				if mode == "cancel" {
					cancel()
					return d.UserTask{}, context.Canceled
				}
				assignee := ""
				if mode == "same" {
					assignee = "alice"
				} else if key == keys[1] && mode != "race" {
					assignee = "bob"
				}
				return d.UserTask{Key: key, State: "CREATED", Assignee: assignee}, nil
			}
			var opts []services.CallOption
			if mode == "override" {
				opts = append(opts, services.WithUserTaskAssignmentOverride())
			}
			if mode == "dry-run" {
				opts = append(opts, services.WithDryRun())
			}
			if mode == "fail-fast" {
				opts = append(opts, services.WithFailFast())
			}
			plan, err := PrepareUserTaskAssignments(ctx, api, append(append(typex.Keys{}, keys...), keys[0]), "alice", 1, opts...)
			if mode == "read failure" || mode == "malformed" || mode == "cancel" {
				require.Error(t, err)
				_, executeErr := ExecuteUserTaskAssignments(ctx, api, plan, false)
				require.ErrorIs(t, executeErr, d.ErrPrecondition)
				require.Empty(t, submitted)
				return
			}
			require.NoError(t, err)
			require.Empty(t, submitted)
			if mode == "override" || mode == "dry-run" {
				require.Empty(t, reads)
			} else {
				require.Equal(t, keys, typex.Keys(reads))
			}
			if mode == "same" {
				require.Empty(t, plan.ReadyKeys)
				require.Equal(t, 3, plan.UnchangedCount)
			}
			// Caller edits to approval metadata must not change the frozen execution scope.
			plan.ReadyKeys = nil
			result, err := ExecuteUserTaskAssignments(ctx, api, plan, false)
			require.Len(t, result.Items, 3)
			for i, item := range result.Items {
				require.Equal(t, keys[i], item.Key)
			}
			switch mode {
			case "mixed", "fail-fast":
				require.ErrorIs(t, err, d.ErrConflict)
				require.True(t, result.Items[1].ReassignmentDenied)
				require.Equal(t, d.UserTaskAssignmentSubmissionNotSubmitted, result.Items[1].Submission)
				require.Zero(t, result.Items[1].StatusCode)
				if mode == "mixed" {
					require.Equal(t, []string{keys[0], keys[2]}, submitted)
				} else {
					require.Empty(t, submitted)
				}
			case "same":
				require.NoError(t, err)
				require.Empty(t, submitted)
				for _, item := range result.Items {
					require.Equal(t, d.UserTaskAssignmentStatusUnchanged, item.Status)
					require.Equal(t, d.UserTaskAssignmentSubmissionNotSubmitted, item.Submission)
					require.Zero(t, item.StatusCode)
					require.Empty(t, item.Error)
				}
			case "dry-run":
				require.NoError(t, err)
				require.Empty(t, submitted)
			case "race":
				require.ErrorIs(t, err, d.ErrConflict)
				require.Len(t, submitted, 3)
			default:
				require.NoError(t, err)
				require.Len(t, submitted, 3)
			}
			if mode != "override" && mode != "dry-run" {
				require.Len(t, reads, 3)
			}
		})
	}
}

func TestPrepareUserTaskAssignmentsFreezesInputs(t *testing.T) {
	keys := completionKeys(1)
	key := keys[0]
	evaluations := 0
	api := &assignmentPlanAPI{assignmentAPI: &assignmentAPI{submit: func(_ context.Context, got, assignee string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
		require.Equal(t, key, got)
		require.Equal(t, "alice", assignee)
		require.False(t, services.ApplyCallOptions(opts).UserTaskAssignmentOverride)
		return acceptedAssignment(got), nil
	}}, read: func(_ context.Context, got string, _ ...services.CallOption) (d.UserTask, error) {
		require.Equal(t, key, got)
		return d.UserTask{Key: got, State: "CREATED"}, nil
	}}
	plan, err := PrepareUserTaskAssignments(context.Background(), api, keys, "alice", 1, func(*services.CallCfg) { evaluations++; keys[0] = "changed" })
	require.NoError(t, err)
	result, err := ExecuteUserTaskAssignments(context.Background(), api, plan, false)
	require.NoError(t, err)
	require.Equal(t, 1, evaluations)
	require.Equal(t, key, result.Items[0].Key)
}

func TestExecuteUserTaskAssignmentsApprovedReplacement(t *testing.T) {
	for _, failFast := range []bool{false, true} {
		t.Run(fmt.Sprint(failFast), func(t *testing.T) {
			keys := completionKeys(3)
			reads := 0
			submitted := []string{}
			api := &assignmentPlanAPI{assignmentAPI: &assignmentAPI{submit: func(_ context.Context, key, assignee string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
				require.Equal(t, "alice", assignee)
				require.Equal(t, key == keys[0], services.ApplyCallOptions(opts).UserTaskAssignmentOverride, "only observed occupied keys receive replacement permission")
				require.NotEqual(t, keys[2], key, "unchanged tasks must never be submitted")
				submitted = append(submitted, key)
				return acceptedAssignment(key), nil
			}}, read: func(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
				reads++
				assignee := ""
				if key == keys[0] {
					assignee = "bob"
				} else if key == keys[2] {
					assignee = "alice"
				}
				return d.UserTask{Key: key, State: "CREATED", Assignee: assignee}, nil
			}}
			var opts []services.CallOption
			if failFast {
				opts = append(opts, services.WithFailFast())
			}
			plan, err := PrepareUserTaskAssignments(context.Background(), api, keys, "alice", 1, opts...)
			require.NoError(t, err)
			require.Equal(t, []AssignmentReplacement{{Key: keys[0], Assignee: "bob"}}, plan.Replacements)
			plan.Replacements[0].Key = keys[1] // Public presentation cannot change authorized execution keys.
			result, err := ExecuteUserTaskAssignments(context.Background(), api, plan, true)
			require.NoError(t, err)
			require.Equal(t, 3, reads)
			require.Equal(t, []string{keys[0], keys[1]}, submitted)
			require.Equal(t, 1, plan.UnchangedCount)
			require.Equal(t, d.UserTaskAssignmentStatusUnchanged, result.Items[2].Status)
			for _, item := range result.Items[:2] {
				require.Equal(t, d.UserTaskAssignmentStatusAccepted, item.Status)
				require.False(t, item.ReassignmentDenied)
			}
		})
	}
}
