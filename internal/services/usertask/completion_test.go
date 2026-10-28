// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// completionAPI exposes native lookup and submission; other operations remain forbidden.
type completionAPI struct {
	API
	lookup func(context.Context, string) (d.UserTask, error)
	submit func(context.Context, string, map[string]any, ...services.CallOption) (d.UserTaskCompletionResult, error)
}

// GetNativeUserTask supplies one assigned, active task unless a case overrides the lookup.
func (a *completionAPI) GetNativeUserTask(ctx context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
	if a.lookup != nil {
		return a.lookup(ctx, key)
	}
	return d.UserTask{Key: key, State: "CREATED", Assignee: "alice"}, nil
}

// CompleteUserTask delegates to the isolated, test-owned submission gate.
func (a *completionAPI) CompleteUserTask(ctx context.Context, key string, vars map[string]any, opts ...services.CallOption) (d.UserTaskCompletionResult, error) {
	return a.submit(ctx, key, vars, opts...)
}

// completionKeys supplies valid, ordered keys without relying on any backend.
func completionKeys(n int) typex.Keys {
	keys := make(typex.Keys, n)
	for i := range keys {
		keys[i] = fmt.Sprintf("%016d", i+1)
	}
	return keys
}

// acceptedCompletion models the native acceptance-only contract.
func acceptedCompletion(key string) d.UserTaskCompletionResult {
	return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusAccepted, Submission: d.UserTaskCompletionSubmissionAccepted, StatusCode: 204}
}

// TestCompleteUserTasksValidation ensures the entire request is valid before any submission, including previews.
func TestCompleteUserTasksValidation(t *testing.T) {
	t.Parallel()
	cycle := map[string]any{}
	cycle["self"] = cycle
	for _, tc := range []struct {
		name    string
		keys    typex.Keys
		vars    map[string]any
		workers int
		opts    []services.CallOption
	}{
		{name: "empty"},
		{name: "late invalid", keys: append(completionKeys(2), "bad")},
		{name: "empty token", keys: append(completionKeys(1), "")},
		{name: "whitespace", keys: typex.Keys{" 0000000000000001"}},
		{name: "unicode digits", keys: typex.Keys{"０００００００００００００００１"}},
		{name: "long", keys: typex.Keys{"10000000000000000"}},
		{name: "negative workers", keys: completionKeys(1), workers: -1},
		{name: "no wait", keys: completionKeys(1), opts: []services.CallOption{services.WithNoWait()}},
		{name: "channel", keys: completionKeys(2), vars: map[string]any{"bad": make(chan int)}},
		{name: "cycle", keys: completionKeys(2), vars: cycle},
		{name: "nonfinite", keys: completionKeys(2), vars: map[string]any{"bad": math.Inf(1)}},
	} {
		for _, dry := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dry=%t", tc.name, dry), func(t *testing.T) {
				opts := append([]services.CallOption(nil), tc.opts...)
				if dry {
					opts = append(opts, services.WithDryRun())
				}
				api := &completionAPI{lookup: func(context.Context, string) (d.UserTask, error) {
					t.Error("invalid input must not be read")
					return d.UserTask{}, nil
				}}
				out, err := CompleteUserTasks(context.Background(), api, tc.keys, tc.vars, tc.workers, opts...)
				require.ErrorIs(t, err, d.ErrValidation)
				require.NotNil(t, out.Items)
				require.Empty(t, out.Items)
			})
		}
	}
	out, err := CompleteUserTasks(context.Background(), nil, completionKeys(1), nil, 1)
	require.ErrorIs(t, err, d.ErrPrecondition)
	require.NotNil(t, out.Items)
	require.Empty(t, out.Items)
}

// TestCompleteUserTasksOrderedDedupAndPreview preserves first occurrence and never submits a preview.
func TestCompleteUserTasksOrderedDedupAndPreview(t *testing.T) {
	t.Parallel()
	keys := completionKeys(3)
	selection := typex.Keys{keys[2], keys[0], keys[2], keys[1], keys[0]}
	for _, dry := range []bool{false, true} {
		t.Run(fmt.Sprint(dry), func(t *testing.T) {
			var calls, reads []string
			api := &completionAPI{lookup: func(_ context.Context, key string) (d.UserTask, error) {
				reads = append(reads, key)
				return d.UserTask{Key: key, State: "CREATED", Assignee: "alice"}, nil
			}, submit: func(_ context.Context, key string, vars map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
				calls = append(calls, key)
				require.Nil(t, vars)
				return acceptedCompletion(key), nil
			}}
			var opts []services.CallOption
			if dry {
				opts = append(opts, services.WithDryRun())
			}
			out, err := CompleteUserTasks(context.Background(), api, selection, nil, 1, opts...)
			require.NoError(t, err)
			require.Len(t, out.Items, 3)
			for i, key := range []string{keys[2], keys[0], keys[1]} {
				expected := acceptedCompletion(key)
				if dry {
					expected = d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusPreview, Submission: d.UserTaskCompletionSubmissionNotSubmitted}
				}
				require.Equal(t, expected, out.Items[i])
			}
			if dry {
				require.Empty(t, calls)
				require.Empty(t, reads)
			} else {
				require.Equal(t, []string{keys[2], keys[0], keys[1]}, reads)
				require.Equal(t, []string{keys[2], keys[0], keys[1]}, calls)
			}
		})
	}
}

// TestCompleteUserTasksSnapshot freezes keys and nested values before the first native call.
func TestCompleteUserTasksSnapshot(t *testing.T) {
	t.Parallel()
	for _, vars := range []map[string]any{nil, {}, {"nested": map[string]any{"list": []any{json.Number("9007199254740993"), true, nil, "text"}}}} {
		keys := completionKeys(2)
		expected, err := toolx.CopyJSONMap(vars)
		require.NoError(t, err)
		calls := 0
		api := &completionAPI{submit: func(_ context.Context, key string, got map[string]any, opts ...services.CallOption) (d.UserTaskCompletionResult, error) {
			calls++
			require.Equal(t, expected, got)
			cfg := services.ApplyCallOptions(opts)
			require.True(t, cfg.Verbose && cfg.IgnoreTenant && cfg.SuppressWorkflowDetailLogs)
			// Mutating caller-owned data after entry must not change this or later submissions.
			keys[1] = "invalid"
			if vars != nil {
				vars["added"] = true
				if nested, ok := vars["nested"].(map[string]any); ok {
					nested["list"].([]any)[0] = "changed"
				}
			}
			return acceptedCompletion(key), nil
		}}
		out, err := CompleteUserTasks(context.Background(), api, keys, vars, 1, services.WithVerbose(), services.WithIgnoreTenant(), services.WithSuppressWorkflowDetailLogs())
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, completionKeys(2)[1], out.Items[1].Key)
	}
}

// TestCompleteUserTasksWorkers uses a barrier to prove actual worker bounds and automatic/explicit precedence.
func TestCompleteUserTasksWorkers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		workers   int
		unlimited bool
	}{
		{"sequential", 1, false}, {"bounded", 3, false}, {"automatic", 0, false}, {"unlimited", 0, true}, {"explicit wins", 2, true}, {"clamped", 100, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keys := completionKeys(40)
			expected := toolx.DetermineNoOfWorkers(len(keys), tc.workers, tc.unlimited)
			var active, peak, calls atomic.Int32
			ready := make(chan struct{})
			var once sync.Once
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			api := &completionAPI{submit: func(ctx context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
				n := active.Add(1)
				defer active.Add(-1)
				calls.Add(1)
				for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
				}
				if n == int32(expected) {
					once.Do(func() { close(ready) })
				}
				select {
				case <-ready:
					return acceptedCompletion(key), nil
				case <-ctx.Done():
					return d.UserTaskCompletionResult{}, ctx.Err()
				}
			}}
			var opts []services.CallOption
			if tc.unlimited {
				opts = append(opts, services.WithNoWorkerLimit())
			}
			out, err := CompleteUserTasks(ctx, api, keys, nil, tc.workers, opts...)
			require.NoError(t, err)
			require.Equal(t, int32(expected), peak.Load())
			require.Equal(t, int32(len(keys)), calls.Load())
			for i, key := range keys {
				require.Equal(t, acceptedCompletion(key), out.Items[i])
			}
		})
	}
}

// TestCompleteUserTasksOutOfOrder forces the last selection to finish first without reordering evidence.
func TestCompleteUserTasksOutOfOrder(t *testing.T) {
	t.Parallel()
	keys := completionKeys(3)
	gates := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{})}
	var entered atomic.Int32
	var completed []string
	var mu sync.Mutex
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	api := &completionAPI{submit: func(ctx context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
		i := 0
		for keys[i] != key {
			i++
		}
		if entered.Add(1) == 3 {
			close(gates[2])
		}
		select {
		case <-gates[i]:
		case <-ctx.Done():
			return d.UserTaskCompletionResult{}, ctx.Err()
		}
		mu.Lock()
		completed = append(completed, key)
		mu.Unlock()
		if i > 0 {
			close(gates[i-1])
		}
		return acceptedCompletion(key), nil
	}}
	out, err := CompleteUserTasks(ctx, api, keys, nil, 3)
	require.NoError(t, err)
	require.Equal(t, []string{keys[2], keys[1], keys[0]}, completed)
	for i, key := range keys {
		require.Equal(t, acceptedCompletion(key), out.Items[i])
	}
}

// TestCompleteUserTasksFailures retains real errors, uncertainty and every queued slot under fail-fast.
func TestCompleteUserTasksFailures(t *testing.T) {
	t.Parallel()
	keys := completionKeys(4)
	for _, fast := range []bool{false, true} {
		t.Run(fmt.Sprint(fast), func(t *testing.T) {
			var calls []string
			api := &completionAPI{submit: func(_ context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
				calls = append(calls, key)
				switch key {
				case keys[1]:
					return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusFailed, Submission: d.UserTaskCompletionSubmissionRejected, StatusCode: 403, Error: "POST https://fake.invalid body secret"}, d.ErrForbidden
				case keys[2]:
					return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusTimedOut, Submission: d.UserTaskCompletionSubmissionUnknown, StatusCode: 504, Error: "timeout"}, d.ErrGatewayTimeout
				default:
					return acceptedCompletion(key), nil
				}
			}}
			var opts []services.CallOption
			if fast {
				opts = append(opts, services.WithFailFast())
			}
			out, err := CompleteUserTasks(context.Background(), api, keys, nil, 1, opts...)
			require.ErrorIs(t, err, d.ErrForbidden)
			require.NotErrorIs(t, err, context.Canceled)
			require.Len(t, out.Items, 4)
			require.Equal(t, acceptedCompletion(keys[0]), out.Items[0])
			require.Equal(t, "POST https://fake.invalid body secret", out.Items[1].Error)
			for i, key := range keys {
				require.Equal(t, key, out.Items[i].Key)
			}
			if fast {
				require.Equal(t, []string(keys[:2]), calls)
				for _, item := range out.Items[2:] {
					require.Equal(t, d.UserTaskCompletionStatusNotAttempted, item.Status)
					require.Equal(t, d.UserTaskCompletionSubmissionNotSubmitted, item.Submission)
				}
			} else {
				require.Equal(t, []string(keys), calls)
				require.ErrorIs(t, err, d.ErrGatewayTimeout)
				require.Equal(t, d.UserTaskCompletionSubmissionUnknown, out.Items[2].Submission)
				require.Equal(t, acceptedCompletion(keys[3]), out.Items[3])
			}
		})
	}
}

// TestCompleteUserTasksFailFastInflight proves scheduling cancellation never becomes the native caller context.
func TestCompleteUserTasksFailFastInflight(t *testing.T) {
	t.Parallel()
	keys := completionKeys(2)
	entered := make(chan struct{})
	rejected := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	api := &completionAPI{submit: func(gotCtx context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
		require.Same(t, ctx, gotCtx)
		if key == keys[0] {
			<-entered
			close(rejected)
			return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusFailed, Submission: d.UserTaskCompletionSubmissionRejected}, d.ErrConflict
		}
		close(entered)
		<-rejected
		return acceptedCompletion(key), nil
	}}
	out, err := CompleteUserTasks(ctx, api, keys, nil, 2, services.WithFailFast())
	require.ErrorIs(t, err, d.ErrConflict)
	require.NotErrorIs(t, err, context.Canceled)
	require.Equal(t, acceptedCompletion(keys[1]), out.Items[1])
}

// TestCompleteUserTasksCancellation preserves accepted facts and distinguishes unstarted from attempted cancellation.
func TestCompleteUserTasksCancellation(t *testing.T) {
	t.Parallel()
	keys := completionKeys(4)
	for _, pre := range []bool{false, true} {
		for _, dry := range []bool{false, true} {
			t.Run(fmt.Sprintf("pre=%t/dry=%t", pre, dry), func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if pre {
					cancel()
				}
				calls := 0
				api := &completionAPI{submit: func(_ context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
					calls++
					if key == keys[0] {
						return acceptedCompletion(key), nil
					}
					cancel()
					return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusCanceled, Submission: d.UserTaskCompletionSubmissionUnknown, Error: context.Canceled.Error()}, context.Canceled
				}}
				var opts []services.CallOption
				if dry {
					opts = append(opts, services.WithDryRun())
				}
				out, err := CompleteUserTasks(ctx, api, keys, nil, 1, opts...)
				require.Len(t, out.Items, 4)
				if dry && !pre {
					require.NoError(t, err)
					require.Zero(t, calls)
					return
				}
				require.ErrorIs(t, err, context.Canceled)
				start := 0
				if !pre {
					require.Equal(t, 2, calls)
					require.Equal(t, acceptedCompletion(keys[0]), out.Items[0])
					require.Equal(t, d.UserTaskCompletionStatusCanceled, out.Items[1].Status)
					require.Equal(t, d.UserTaskCompletionSubmissionUnknown, out.Items[1].Submission)
					start = 2
				} else {
					require.Zero(t, calls)
				}
				for _, item := range out.Items[start:] {
					require.Equal(t, d.UserTaskCompletionStatusNotAttempted, item.Status)
					require.Equal(t, d.UserTaskCompletionSubmissionNotSubmitted, item.Submission)
				}
			})
		}
	}
}

// TestCompleteUserTasksCallerErrorAfterResult keeps authoritative acceptance or rejection alongside caller cancellation.
func TestCompleteUserTasksCallerErrorAfterResult(t *testing.T) {
	t.Parallel()
	for _, reject := range []bool{false, true} {
		t.Run(fmt.Sprint(reject), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			key := completionKeys(1)[0]
			expected := acceptedCompletion(key)
			var nativeErr error
			if reject {
				expected = d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusFailed, Submission: d.UserTaskCompletionSubmissionRejected, StatusCode: 403}
				nativeErr = d.ErrForbidden
			}
			api := &completionAPI{submit: func(context.Context, string, map[string]any, ...services.CallOption) (d.UserTaskCompletionResult, error) {
				cancel()
				return expected, nativeErr
			}}
			out, err := CompleteUserTasks(ctx, api, typex.Keys{key, key}, nil, 1, services.WithFailFast())
			require.ErrorIs(t, err, context.Canceled)
			if reject {
				require.ErrorIs(t, err, d.ErrForbidden)
			}
			require.Equal(t, []d.UserTaskCompletionResult{expected}, out.Items)
		})
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
	defer cancel()
	out, err := CompleteUserTasks(ctx, &completionAPI{}, completionKeys(1), nil, 0)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Equal(t, []d.UserTaskCompletionResult{{Key: completionKeys(1)[0], Status: d.UserTaskCompletionStatusNotAttempted, Submission: d.UserTaskCompletionSubmissionNotSubmitted}}, out.Items)
}

// TestCompleteUserTasksPreflight checks each unique key once and never submits a known completed task.
func TestCompleteUserTasksPreflight(t *testing.T) {
	for _, failFast := range []bool{false, true} {
		t.Run(fmt.Sprint(failFast), func(t *testing.T) {
			keys := completionKeys(3)
			reads := map[string]int{}
			submissions := []string{}
			api := &completionAPI{
				lookup: func(_ context.Context, key string) (d.UserTask, error) {
					reads[key]++
					if key == keys[0] {
						return d.UserTask{Key: key, State: "COMPLETED"}, nil
					}
					if key == keys[2] {
						return d.UserTask{}, fmt.Errorf("GET https://secret.invalid body=secret")
					}
					return d.UserTask{Key: key, State: "CREATED"}, nil
				},
				submit: func(_ context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
					submissions = append(submissions, key)
					return acceptedCompletion(key), nil
				},
			}
			var opts []services.CallOption
			if failFast {
				opts = append(opts, services.WithFailFast())
			}
			out, err := CompleteUserTasks(context.Background(), api, append(keys, keys[0]), map[string]any{"value": 1}, 1, opts...)
			require.ErrorIs(t, err, d.ErrConflict)
			require.NotContains(t, err.Error(), "secret.invalid")
			require.Equal(t, map[string]int{keys[0]: 1, keys[1]: 1, keys[2]: 1}, reads)
			require.Len(t, out.Items, 3)
			require.Equal(t, d.UserTaskCompletionStatusFailed, out.Items[0].Status)
			require.Equal(t, d.UserTaskCompletionSubmissionNotSubmitted, out.Items[0].Submission)
			require.Zero(t, out.Items[0].StatusCode)
			require.Contains(t, out.Items[0].Error, "already completed")
			require.Contains(t, out.Items[0].Error, "variables not applied")
			if failFast {
				require.Empty(t, submissions)
				require.Equal(t, d.UserTaskCompletionStatusNotAttempted, out.Items[1].Status)
			} else {
				require.Equal(t, keys[1:], typex.Keys(submissions))
				require.Equal(t, d.UserTaskCompletionStatusAccepted, out.Items[2].Status)
			}
		})
	}
}

// TestPrepareUserTaskCompletionsFreezes checks advisory counts and prevents a second read after approval.
func TestPrepareUserTaskCompletionsFreezes(t *testing.T) {
	keys := completionKeys(3)
	vars := map[string]any{"nested": map[string]any{"n": json.Number("9007199254740993")}}
	reads, calls := 0, 0
	api := &completionAPI{lookup: func(_ context.Context, key string) (d.UserTask, error) {
		reads++
		if key == keys[0] {
			return d.UserTask{Key: key, State: "COMPLETED"}, nil
		}
		if key == keys[1] {
			return d.UserTask{Key: key, State: "CREATED"}, nil
		}
		return d.UserTask{}, fmt.Errorf("unavailable")
	}, submit: func(_ context.Context, key string, got map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
		calls++
		require.Equal(t, json.Number("9007199254740993"), got["nested"].(map[string]any)["n"])
		return acceptedCompletion(key), nil
	}}
	plan, err := PrepareUserTaskCompletions(context.Background(), api, keys, vars, 1)
	require.NoError(t, err)
	require.Equal(t, 2, plan.ReadyCount)
	require.Equal(t, 1, plan.CompletedCount)
	require.Equal(t, 1, plan.UnassignedCount)
	require.Equal(t, 1, plan.UncheckedCount)
	keys[1] = "changed"
	vars["nested"].(map[string]any)["n"] = "changed"
	out, err := ExecuteUserTaskCompletions(context.Background(), api, plan)
	require.ErrorIs(t, err, d.ErrConflict)
	require.Equal(t, 3, reads)
	require.Equal(t, 2, calls)
	require.Equal(t, completionKeys(3)[1], out.Items[1].Key)
}

// TestPrepareUserTaskCompletionsCancellation submits nothing when preparation is canceled.
func TestPrepareUserTaskCompletionsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api := &completionAPI{lookup: func(_ context.Context, key string) (d.UserTask, error) {
		cancel()
		return d.UserTask{}, context.Canceled
	}, submit: func(context.Context, string, map[string]any, ...services.CallOption) (d.UserTaskCompletionResult, error) {
		t.Fatal("must not submit")
		return d.UserTaskCompletionResult{}, nil
	}}
	_, err := CompleteUserTasks(ctx, api, completionKeys(2), nil, 1)
	require.ErrorIs(t, err, context.Canceled)
}

// TestPrepareUserTaskCompletionsUnusableRead never treats missing or contradictory evidence as completed/unassigned.
func TestPrepareUserTaskCompletionsUnusableRead(t *testing.T) {
	for _, observed := range []d.UserTask{{}, {Key: "wrong", State: "COMPLETED"}, {Key: completionKeys(1)[0]}} {
		api := &completionAPI{lookup: func(context.Context, string) (d.UserTask, error) { return observed, nil },
			submit: func(_ context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
				return acceptedCompletion(key), nil
			},
		}
		plan, err := PrepareUserTaskCompletions(context.Background(), api, completionKeys(1), nil, 1)
		require.NoError(t, err)
		require.Equal(t, 1, plan.ReadyCount)
		require.Equal(t, 1, plan.UncheckedCount)
		require.Zero(t, plan.UnassignedCount)
		out, err := ExecuteUserTaskCompletions(context.Background(), api, plan)
		require.NoError(t, err)
		require.Equal(t, d.UserTaskCompletionStatusAccepted, out.Items[0].Status)
	}
	out, err := ExecuteUserTaskCompletions(context.Background(), &completionAPI{}, CompletionPlan{})
	require.ErrorIs(t, err, d.ErrPrecondition)
	require.Empty(t, out.Items)
}

// AssignUserTask fails closed when an unrelated fixture receives an assignment.
func (a *completionAPI) AssignUserTask(context.Context, string, string, ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	panic("unexpected user-task assignment")
}

// UnassignUserTask fails closed when an unrelated fixture receives an unassignment.
func (a *completionAPI) UnassignUserTask(context.Context, string, ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	panic("unexpected user-task unassignment")
}
