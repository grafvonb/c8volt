// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// assignmentAPI allows only the selected native action; embedded nil methods fail closed on reads or unrelated mutations.
type assignmentAPI struct {
	API
	unassign bool
	submit   func(context.Context, string, string, ...services.CallOption) (d.UserTaskAssignmentResult, error)
}

// AssignUserTask rejects accidental assignment in a removal workflow.
func (a *assignmentAPI) AssignUserTask(ctx context.Context, key, assignee string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	if a.unassign {
		panic("unexpected assignment")
	}
	return a.submit(ctx, key, assignee, opts...)
}

// UnassignUserTask rejects accidental removal in an assignment workflow.
func (a *assignmentAPI) UnassignUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	if !a.unassign {
		panic("unexpected unassignment")
	}
	return a.submit(ctx, key, "", opts...)
}

// acceptedAssignment models documented native acceptance without a task lookup.
func acceptedAssignment(key string) d.UserTaskAssignmentResult {
	return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusAccepted, Submission: d.UserTaskAssignmentSubmissionAccepted, StatusCode: 204}
}

// TestAssignUserTasks exercises the shared collection contract through the assignment entry point.
func TestAssignUserTasks(t *testing.T) {
	t.Parallel()
	testAssignmentCollection(t, false)
}

// TestUnassignUserTasks exercises the same contract through the unassignment entry point.
func TestUnassignUserTasks(t *testing.T) {
	t.Parallel()
	testAssignmentCollection(t, true)
}

// testAssignmentCollection proves validation, immutable intent, scheduling and evidence for each action.
func testAssignmentCollection(t *testing.T, unassign bool) {
	identity := " \tteam/żółć+opaque\n "
	if unassign {
		identity = ""
	}
	run := func(ctx context.Context, api API, keys typex.Keys, assignee string, workers int, opts ...services.CallOption) (d.UserTaskAssignmentResults, error) {
		if unassign {
			return UnassignUserTasks(ctx, api, keys, workers, opts...)
		}
		return AssignUserTasks(ctx, api, keys, assignee, workers, opts...)
	}
	t.Run("validation", func(t *testing.T) {
		type validationCase struct {
			name     string
			keys     typex.Keys
			workers  int
			opts     []services.CallOption
			identity string
		}
		cases := []validationCase{
			{name: "empty", identity: identity},
			{name: "late invalid", keys: append(completionKeys(2), "bad"), identity: identity},
			{name: "empty key", keys: append(completionKeys(1), ""), identity: identity},
			{name: "whitespace key", keys: typex.Keys{" 000000000000001"}, identity: identity},
			{name: "unicode digits", keys: typex.Keys{"０００００００００００００００１"}, identity: identity},
			{name: "long key", keys: typex.Keys{"10000000000000000"}, identity: identity},
			{name: "negative workers", keys: completionKeys(1), workers: -1, identity: identity},
			{name: "no wait", keys: completionKeys(1), opts: []services.CallOption{services.WithNoWait()}, identity: identity},
		}
		if unassign {
			cases = append(cases, validationCase{name: "override", keys: completionKeys(1), opts: []services.CallOption{services.WithUserTaskAssignmentOverride()}})
		} else {
			for _, blank := range []string{"", " \t\n", "\u2003"} {
				cases = append(cases, validationCase{name: fmt.Sprintf("blank %q", blank), keys: completionKeys(1), identity: blank})
			}
		}
		for _, tc := range cases {
			for _, dry := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/dry=%t", tc.name, dry), func(t *testing.T) {
					opts := append([]services.CallOption{}, tc.opts...)
					if dry {
						opts = append(opts, services.WithDryRun())
					}
					out, err := run(context.Background(), &assignmentAPI{unassign: unassign}, tc.keys, tc.identity, tc.workers, opts...)
					require.ErrorIs(t, err, d.ErrValidation)
					require.NotNil(t, out.Items)
					require.Empty(t, out.Items)
				})
			}
		}
		for _, opts := range [][]services.CallOption{nil, {services.WithDryRun()}} {
			out, err := run(context.Background(), nil, completionKeys(1), identity, 0, opts...)
			require.ErrorIs(t, err, d.ErrPrecondition)
			require.NotNil(t, out.Items)
			require.Empty(t, out.Items)
		}
	})
	t.Run("deduplication and preview", func(t *testing.T) {
		keys := completionKeys(3)
		for _, selection := range []typex.Keys{{keys[0]}, {keys[2], keys[0], keys[2], keys[1], keys[0]}} {
			for _, dry := range []bool{false, true} {
				var calls []string
				api := &assignmentAPI{unassign: unassign, submit: func(_ context.Context, key, gotIdentity string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
					require.Equal(t, identity, gotIdentity)
					require.False(t, services.ApplyCallOptions(opts).UserTaskAssignmentOverride)
					calls = append(calls, key)
					return acceptedAssignment(key), nil
				}}
				var opts []services.CallOption
				if dry {
					opts = append(opts, services.WithDryRun())
				}
				out, err := run(context.Background(), api, selection, identity, 1, opts...)
				require.NoError(t, err)
				require.Len(t, out.Items, len(selection.Unique()))
				for i, key := range selection.Unique() {
					want := acceptedAssignment(key)
					if dry {
						want = d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusPreview, Submission: d.UserTaskAssignmentSubmissionNotSubmitted}
					}
					require.Equal(t, want, out.Items[i])
				}
				if dry {
					require.Empty(t, calls)
				} else {
					require.Equal(t, []string(selection.Unique()), calls)
				}
			}
		}
	})
	t.Run("snapshot", func(t *testing.T) {
		keys := completionKeys(3)
		var evaluations int
		var retained *services.CallCfg
		override := !unassign
		opt := func(cfg *services.CallCfg) {
			evaluations++
			cfg.UserTaskAssignmentOverride = override
			cfg.Verbose = true
			cfg.IgnoreTenant = true
			cfg.SuppressWorkflowDetailLogs = true
			retained = cfg
		}
		api := &assignmentAPI{unassign: unassign, submit: func(_ context.Context, key, gotIdentity string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
			require.Equal(t, identity, gotIdentity)
			cfg := services.ApplyCallOptions(opts)
			require.Equal(t, !unassign, cfg.UserTaskAssignmentOverride)
			require.True(t, cfg.Verbose && cfg.IgnoreTenant && cfg.SuppressWorkflowDetailLogs)
			// Caller keys, closure state, retained evaluation config and per-call config may all change after entry.
			keys[1] = "invalid"
			override = false
			*retained = services.CallCfg{}
			*cfg = services.CallCfg{}
			// Re-evaluation by an adapter must also use the same frozen intent.
			require.Equal(t, !unassign, services.ApplyCallOptions(opts).UserTaskAssignmentOverride)
			return acceptedAssignment(key), nil
		}}
		out, err := run(context.Background(), api, keys, identity, 1, opt)
		require.NoError(t, err)
		require.Equal(t, 1, evaluations)
		for i, key := range completionKeys(3) {
			require.Equal(t, acceptedAssignment(key), out.Items[i])
		}
	})
	t.Run("workers", func(t *testing.T) {
		for _, tc := range []struct {
			name      string
			workers   int
			unlimited bool
		}{
			{"sequential", 1, false}, {"bounded", 3, false}, {"automatic", 0, false}, {"unlimited", 0, true}, {"explicit wins", 2, true}, {"clamped", 100, false},
		} {
			t.Run(tc.name, func(t *testing.T) {
				keys := completionKeys(40)
				want := toolx.DetermineNoOfWorkers(len(keys), tc.workers, tc.unlimited)
				var active, peak, calls atomic.Int32
				ready := make(chan struct{})
				var once sync.Once
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				api := &assignmentAPI{unassign: unassign, submit: func(ctx context.Context, key, _ string, _ ...services.CallOption) (d.UserTaskAssignmentResult, error) {
					n := active.Add(1)
					defer active.Add(-1)
					calls.Add(1)
					for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
					}
					if n == int32(want) {
						once.Do(func() { close(ready) })
					}
					select {
					case <-ready:
						return acceptedAssignment(key), nil
					case <-ctx.Done():
						return d.UserTaskAssignmentResult{}, ctx.Err()
					}
				}}
				var opts []services.CallOption
				if tc.unlimited {
					opts = append(opts, services.WithNoWorkerLimit())
				}
				out, err := run(ctx, api, keys, identity, tc.workers, opts...)
				require.NoError(t, err)
				require.Equal(t, int32(want), peak.Load())
				require.Equal(t, int32(len(keys)), calls.Load())
				for i, key := range keys {
					require.Equal(t, acceptedAssignment(key), out.Items[i])
				}
			})
		}
	})
	t.Run("out of order", func(t *testing.T) {
		keys := completionKeys(3)
		gates := []chan struct{}{make(chan struct{}), make(chan struct{}), make(chan struct{})}
		var entered atomic.Int32
		var completed []string
		var mu sync.Mutex
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		api := &assignmentAPI{unassign: unassign, submit: func(ctx context.Context, key, _ string, _ ...services.CallOption) (d.UserTaskAssignmentResult, error) {
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
				return d.UserTaskAssignmentResult{}, ctx.Err()
			}
			mu.Lock()
			completed = append(completed, key)
			mu.Unlock()
			if i > 0 {
				close(gates[i-1])
			}
			return acceptedAssignment(key), nil
		}}
		out, err := run(ctx, api, keys, identity, 3)
		require.NoError(t, err)
		require.Equal(t, []string{keys[2], keys[1], keys[0]}, completed)
		for i, key := range keys {
			require.Equal(t, acceptedAssignment(key), out.Items[i])
		}
	})
	t.Run("failures", func(t *testing.T) {
		keys := completionKeys(4)
		for _, fast := range []bool{false, true} {
			var calls []string
			api := &assignmentAPI{unassign: unassign, submit: func(_ context.Context, key, _ string, _ ...services.CallOption) (d.UserTaskAssignmentResult, error) {
				calls = append(calls, key)
				switch key {
				case keys[1]:
					return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusFailed, Submission: d.UserTaskAssignmentSubmissionRejected, StatusCode: 403, Error: "POST https://fake.invalid body=secret"}, d.ErrForbidden
				case keys[2]:
					return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusTimedOut, Submission: d.UserTaskAssignmentSubmissionUnknown, StatusCode: 504, Error: "timeout"}, d.ErrGatewayTimeout
				default:
					return acceptedAssignment(key), nil
				}
			}}
			var opts []services.CallOption
			if fast {
				opts = append(opts, services.WithFailFast())
			}
			out, err := run(context.Background(), api, keys, identity, 1, opts...)
			require.ErrorIs(t, err, d.ErrForbidden)
			require.NotErrorIs(t, err, context.Canceled)
			require.Len(t, out.Items, 4)
			require.Equal(t, acceptedAssignment(keys[0]), out.Items[0])
			require.Equal(t, "POST https://fake.invalid body=secret", out.Items[1].Error)
			require.Equal(t, d.UserTaskAssignmentSubmissionRejected, out.Items[1].Submission)
			for i, key := range keys {
				require.Equal(t, key, out.Items[i].Key)
			}
			if fast {
				require.Equal(t, []string(keys[:2]), calls)
				for _, item := range out.Items[2:] {
					require.Equal(t, d.UserTaskAssignmentResult{Key: item.Key, Status: d.UserTaskAssignmentStatusNotAttempted, Submission: d.UserTaskAssignmentSubmissionNotSubmitted}, item)
				}
			} else {
				require.Equal(t, []string(keys), calls)
				require.ErrorIs(t, err, d.ErrGatewayTimeout)
				require.Equal(t, d.UserTaskAssignmentSubmissionUnknown, out.Items[2].Submission)
				require.Equal(t, 504, out.Items[2].StatusCode)
				require.Equal(t, acceptedAssignment(keys[3]), out.Items[3])
			}
		}
	})
	t.Run("fail fast retains in flight and queued evidence", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			keys := completionKeys(6)
			entered, release, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
			ctx := context.Background()
			var calls atomic.Int32
			api := &assignmentAPI{unassign: unassign, submit: func(gotCtx context.Context, key, _ string, _ ...services.CallOption) (d.UserTaskAssignmentResult, error) {
				calls.Add(1)
				if gotCtx != ctx {
					t.Error("submission must use caller context")
				}
				if key == keys[0] {
					<-entered
					return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusFailed, Submission: d.UserTaskAssignmentSubmissionRejected}, d.ErrConflict
				}
				if key != keys[1] {
					t.Error("queued mutation started after failure")
					return acceptedAssignment(key), nil
				}
				close(entered)
				<-release
				if gotCtx.Err() != nil {
					t.Error("fail-fast canceled in-flight submission")
				}
				return acceptedAssignment(key), nil
			}}
			var out d.UserTaskAssignmentResults
			var err error
			go func() { out, err = run(ctx, api, keys, identity, 2, services.WithFailFast()); close(done) }()
			// All runnable work (including observing the rejection) finishes while key 2 remains blocked.
			synctest.Wait()
			require.Equal(t, int32(2), calls.Load())
			close(release)
			<-done
			require.ErrorIs(t, err, d.ErrConflict)
			require.NotErrorIs(t, err, context.Canceled)
			require.Equal(t, acceptedAssignment(keys[1]), out.Items[1])
			for _, item := range out.Items[2:] {
				require.Equal(t, d.UserTaskAssignmentResult{Key: item.Key, Status: d.UserTaskAssignmentStatusNotAttempted, Submission: d.UserTaskAssignmentSubmissionNotSubmitted}, item)
			}
		})
	})
	t.Run("cancellation", func(t *testing.T) {
		keys := completionKeys(4)
		for _, pre := range []bool{false, true} {
			for _, dry := range []bool{false, true} {
				ctx, cancel := context.WithCancel(context.Background())
				if pre {
					cancel()
				}
				calls := 0
				api := &assignmentAPI{unassign: unassign, submit: func(_ context.Context, key, _ string, _ ...services.CallOption) (d.UserTaskAssignmentResult, error) {
					calls++
					if key == keys[0] {
						return acceptedAssignment(key), nil
					}
					cancel()
					return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusCanceled, Submission: d.UserTaskAssignmentSubmissionUnknown, Error: context.Canceled.Error()}, context.Canceled
				}}
				var opts []services.CallOption
				if dry {
					opts = append(opts, services.WithDryRun())
				}
				out, err := run(ctx, api, keys, identity, 1, opts...)
				cancel()
				require.Len(t, out.Items, 4)
				if dry && !pre {
					require.NoError(t, err)
					require.Zero(t, calls)
					continue
				}
				require.ErrorIs(t, err, context.Canceled)
				start := 0
				if !pre {
					require.Equal(t, 2, calls)
					require.Equal(t, acceptedAssignment(keys[0]), out.Items[0])
					require.Equal(t, d.UserTaskAssignmentStatusCanceled, out.Items[1].Status)
					require.Equal(t, d.UserTaskAssignmentSubmissionUnknown, out.Items[1].Submission)
					start = 2
				} else {
					require.Zero(t, calls)
				}
				for _, item := range out.Items[start:] {
					require.Equal(t, d.UserTaskAssignmentResult{Key: item.Key, Status: d.UserTaskAssignmentStatusNotAttempted, Submission: d.UserTaskAssignmentSubmissionNotSubmitted}, item)
				}
			}
		}
	})
	t.Run("caller error after result", func(t *testing.T) {
		for _, reject := range []bool{false, true} {
			ctx, cancel := context.WithCancel(context.Background())
			key := completionKeys(1)[0]
			want := acceptedAssignment(key)
			var nativeErr error
			if reject {
				want = d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusFailed, Submission: d.UserTaskAssignmentSubmissionRejected, StatusCode: 403}
				nativeErr = d.ErrForbidden
			}
			api := &assignmentAPI{unassign: unassign, submit: func(context.Context, string, string, ...services.CallOption) (d.UserTaskAssignmentResult, error) {
				cancel()
				return want, nativeErr
			}}
			out, err := run(ctx, api, typex.Keys{key, key}, identity, 1, services.WithFailFast())
			cancel()
			require.ErrorIs(t, err, context.Canceled)
			if reject {
				require.ErrorIs(t, err, d.ErrForbidden)
			}
			require.Equal(t, []d.UserTaskAssignmentResult{want}, out.Items)
		}
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
		defer cancel()
		out, err := run(ctx, &assignmentAPI{unassign: unassign}, completionKeys(1), identity, 0)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		require.Equal(t, d.UserTaskAssignmentStatusNotAttempted, out.Items[0].Status)
	})
}
