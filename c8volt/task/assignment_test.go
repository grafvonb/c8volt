// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ferr "github.com/grafvonb/c8volt/c8volt/ferrors"
	options "github.com/grafvonb/c8volt/c8volt/foptions"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	utsvc "github.com/grafvonb/c8volt/internal/services/usertask"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// facadeAssignmentAPI permits only the expected native mutation; nil embedded methods reject reads and unrelated operations.
type facadeAssignmentAPI struct {
	utsvc.API
	unassign bool
	submit   func(context.Context, string, string, ...services.CallOption) (d.UserTaskAssignmentResult, error)
}

// AssignUserTask preserves exact assignment inputs and rejects cross-action calls.
func (a *facadeAssignmentAPI) AssignUserTask(ctx context.Context, key, assignee string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	if a.unassign {
		panic("unexpected assignment")
	}
	return a.submit(ctx, key, assignee, opts...)
}

// UnassignUserTask permits removal without an assignee or any preliminary read.
func (a *facadeAssignmentAPI) UnassignUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	if !a.unassign {
		panic("unexpected unassignment")
	}
	return a.submit(ctx, key, "", opts...)
}

// TestAssignUserTasksFacade verifies assignment wiring, error conversion and public result serialization.
func TestAssignUserTasksFacade(t *testing.T) { t.Parallel(); testAssignmentFacade(t, false) }

// TestUnassignUserTasksFacade verifies the same public contract for assignment removal.
func TestUnassignUserTasksFacade(t *testing.T) { t.Parallel(); testAssignmentFacade(t, true) }

// testAssignmentFacade exercises both constructors and all facade-owned mappings through each public entry point.
func testAssignmentFacade(t *testing.T, unassign bool) {
	identity := " \tdepartment/żółć\n "
	if unassign {
		identity = ""
	}
	run := func(ctx context.Context, c API, keys typex.Keys, assignee string, workers int, opts ...options.FacadeOption) (UserTaskAssignmentResults, error) {
		if unassign {
			return c.UnassignUserTasks(ctx, keys, workers, opts...)
		}
		return c.AssignUserTasks(ctx, keys, assignee, workers, opts...)
	}
	t.Run("constructors and frozen options", func(t *testing.T) {
		for _, extended := range []bool{false, true} {
			for _, override := range []bool{false, true} {
				if unassign && override {
					continue
				}
				keys := typex.Keys{"0000000000000002", "0000000000000001", "0000000000000002"}
				var calls []string
				var evaluations int
				var retained *options.FacadeCfg
				opt := func(cfg *options.FacadeCfg) { evaluations++; cfg.UserTaskAssignmentOverride = override; retained = cfg }
				api := &facadeAssignmentAPI{unassign: unassign, submit: func(_ context.Context, key, got string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
					require.Equal(t, identity, got)
					cfg := services.ApplyCallOptions(opts)
					require.Equal(t, override, cfg.UserTaskAssignmentOverride)
					require.True(t, cfg.Verbose && cfg.FailFast && cfg.NoWorkerLimit && cfg.IgnoreTenant && cfg.SuppressWorkflowDetailLogs)
					keys[1] = "changed"
					*retained = options.FacadeCfg{}
					calls = append(calls, key)
					return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusAccepted, Submission: d.UserTaskAssignmentSubmissionAccepted, StatusCode: 204}, nil
				}}
				c := New(nil, nil, api, nil)
				if extended {
					c = NewWithVariableUpdates(nil, nil, api, nil, nil)
				}
				out, err := run(context.Background(), c, keys, identity, 1, opt, options.WithVerbose(), options.WithFailFast(), options.WithNoWorkerLimit(), options.WithIgnoreTenant(), options.WithSuppressWorkflowDetailLogs())
				require.NoError(t, err)
				require.Equal(t, 1, evaluations)
				require.Equal(t, []string{"0000000000000002", "0000000000000001"}, calls)
				require.Len(t, out.Items, 2)
				for i, key := range calls {
					require.Equal(t, UserTaskAssignmentResult{Key: key, Status: UserTaskAssignmentStatusAccepted, Submission: UserTaskAssignmentSubmissionAccepted, StatusCode: 204}, out.Items[i])
				}
			}
		}
	})
	t.Run("worker forwarding", func(t *testing.T) {
		var active, peak atomic.Int32
		var once sync.Once
		ready := make(chan struct{})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		api := &facadeAssignmentAPI{unassign: unassign, submit: func(ctx context.Context, key, _ string, _ ...services.CallOption) (d.UserTaskAssignmentResult, error) {
			n := active.Add(1)
			defer active.Add(-1)
			for old := peak.Load(); n > old && !peak.CompareAndSwap(old, n); old = peak.Load() {
			}
			if n == 2 {
				once.Do(func() { close(ready) })
			}
			select {
			case <-ready:
			case <-ctx.Done():
				return d.UserTaskAssignmentResult{}, ctx.Err()
			}
			return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusAccepted, Submission: d.UserTaskAssignmentSubmissionAccepted, StatusCode: 204}, nil
		}}
		keys := typex.Keys{"0000000000000001", "0000000000000002", "0000000000000003", "0000000000000004"}
		out, err := run(ctx, New(nil, nil, api, nil), keys, identity, 2, options.WithNoWorkerLimit())
		require.NoError(t, err)
		require.Equal(t, int32(2), peak.Load())
		require.Len(t, out.Items, 4)
	})
	t.Run("partial errors", func(t *testing.T) {
		for _, tc := range []struct {
			name         string
			cause, class error
			status       d.UserTaskAssignmentStatus
			submission   d.UserTaskAssignmentSubmission
			code         int
		}{
			{"forbidden", d.ErrForbidden, ferr.ErrLocalPrecondition, d.UserTaskAssignmentStatusFailed, d.UserTaskAssignmentSubmissionRejected, 403},
			{"missing", d.ErrNotFound, ferr.ErrNotFound, d.UserTaskAssignmentStatusFailed, d.UserTaskAssignmentSubmissionRejected, 404},
			{"conflict", d.ErrConflict, ferr.ErrConflict, d.UserTaskAssignmentStatusFailed, d.UserTaskAssignmentSubmissionRejected, 409},
			{"timeout", d.ErrGatewayTimeout, ferr.ErrTimeout, d.UserTaskAssignmentStatusTimedOut, d.UserTaskAssignmentSubmissionUnknown, 504},
			{"cancel", context.Canceled, context.Canceled, d.UserTaskAssignmentStatusCanceled, d.UserTaskAssignmentSubmissionUnknown, 0},
			{"unavailable", d.ErrUnavailable, ferr.ErrUnavailable, d.UserTaskAssignmentStatusFailed, d.UserTaskAssignmentSubmissionUnknown, 503},
		} {
			t.Run(tc.name, func(t *testing.T) {
				keys := typex.Keys{"0000000000000001", "0000000000000002", "0000000000000003"}
				evidence := "POST https://fake.invalid/assignment body=secret"
				api := &facadeAssignmentAPI{unassign: unassign, submit: func(_ context.Context, key, _ string, _ ...services.CallOption) (d.UserTaskAssignmentResult, error) {
					if key == keys[0] {
						return d.UserTaskAssignmentResult{Key: key, Status: d.UserTaskAssignmentStatusAccepted, Submission: d.UserTaskAssignmentSubmissionAccepted, StatusCode: 204}, nil
					}
					return d.UserTaskAssignmentResult{Key: key, Status: tc.status, Submission: tc.submission, StatusCode: tc.code, Error: evidence}, fmt.Errorf("%w: %s", tc.cause, evidence)
				}}
				out, err := run(context.Background(), New(nil, nil, api, nil), keys, identity, 1, options.WithFailFast())
				require.ErrorIs(t, err, tc.class)
				require.Equal(t, ferr.ExitCode(ferr.FromDomain(tc.cause)), ferr.ExitCode(err))
				require.Len(t, out.Items, 3)
				require.Equal(t, UserTaskAssignmentStatusAccepted, out.Items[0].Status)
				require.Equal(t, UserTaskAssignmentResult{Key: keys[1], Status: UserTaskAssignmentStatus(tc.status), Submission: UserTaskAssignmentSubmission(tc.submission), StatusCode: tc.code, Error: evidence}, out.Items[1])
				require.Equal(t, UserTaskAssignmentResult{Key: keys[2], Status: UserTaskAssignmentStatusNotAttempted, Submission: UserTaskAssignmentSubmissionNotSubmitted}, out.Items[2])
				require.Contains(t, err.Error(), evidence)
			})
		}
	})
	t.Run("validation and preview", func(t *testing.T) {
		key := typex.Keys{"0000000000000001"}
		fake := &facadeAssignmentAPI{unassign: unassign}
		type validationCase struct {
			name     string
			api      utsvc.API
			keys     typex.Keys
			workers  int
			opts     []options.FacadeOption
			class    error
			identity string
		}
		cases := []validationCase{
			{name: "missing dependency", keys: key, class: ferr.ErrLocalPrecondition, identity: identity},
			{name: "empty selection", api: fake, class: ferr.ErrInvalidInput, identity: identity},
			{name: "late invalid key", api: fake, keys: append(key, "bad"), class: ferr.ErrInvalidInput, identity: identity},
			{name: "negative workers", api: fake, keys: key, workers: -1, class: ferr.ErrInvalidInput, identity: identity},
			{name: "no wait", api: fake, keys: key, opts: []options.FacadeOption{options.WithNoWait()}, class: ferr.ErrInvalidInput, identity: identity},
		}
		if unassign {
			cases = append(cases, validationCase{name: "override", api: fake, keys: key, opts: []options.FacadeOption{options.WithUserTaskAssignmentOverride()}, class: ferr.ErrInvalidInput})
		} else {
			cases = append(cases, validationCase{name: "blank identity", api: fake, keys: key, identity: " \t", class: ferr.ErrInvalidInput})
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				out, err := run(context.Background(), New(nil, nil, tc.api, nil), tc.keys, tc.identity, tc.workers, tc.opts...)
				require.ErrorIs(t, err, tc.class)
				b, err := json.Marshal(out)
				require.NoError(t, err)
				require.Equal(t, `{"items":[]}`, string(b))
			})
		}
		out, err := run(context.Background(), New(nil, nil, fake, nil), key, identity, 0, options.WithDryRun())
		require.NoError(t, err)
		b, err := json.Marshal(out)
		require.NoError(t, err)
		require.JSONEq(t, `{"items":[{"key":"0000000000000001","status":"preview","submission":"not_submitted"}]}`, string(b))
	})
}

// TestAssignmentPublicModels covers every enum, JSON omissions, initialized items and independent output ownership.
func TestAssignmentPublicModels(t *testing.T) {
	t.Parallel()
	rows := []d.UserTaskAssignmentResult{
		{Status: d.UserTaskAssignmentStatusPreview, Submission: d.UserTaskAssignmentSubmissionNotSubmitted},
		{Status: d.UserTaskAssignmentStatusAccepted, Submission: d.UserTaskAssignmentSubmissionAccepted},
		{Status: d.UserTaskAssignmentStatusFailed, Submission: d.UserTaskAssignmentSubmissionRejected},
		{Status: d.UserTaskAssignmentStatusTimedOut, Submission: d.UserTaskAssignmentSubmissionUnknown},
		{Status: d.UserTaskAssignmentStatusCanceled, Submission: d.UserTaskAssignmentSubmissionUnknown},
		{Status: d.UserTaskAssignmentStatusNotAttempted, Submission: d.UserTaskAssignmentSubmissionNotSubmitted},
		{Status: d.UserTaskAssignmentStatusUnchanged, Submission: d.UserTaskAssignmentSubmissionNotSubmitted},
	}
	statuses := []UserTaskAssignmentStatus{UserTaskAssignmentStatusPreview, UserTaskAssignmentStatusAccepted, UserTaskAssignmentStatusFailed, UserTaskAssignmentStatusTimedOut, UserTaskAssignmentStatusCanceled, UserTaskAssignmentStatusNotAttempted, UserTaskAssignmentStatusUnchanged}
	submissions := []UserTaskAssignmentSubmission{UserTaskAssignmentSubmissionNotSubmitted, UserTaskAssignmentSubmissionAccepted, UserTaskAssignmentSubmissionRejected, UserTaskAssignmentSubmissionUnknown, UserTaskAssignmentSubmissionUnknown, UserTaskAssignmentSubmissionNotSubmitted, UserTaskAssignmentSubmissionNotSubmitted}
	for i := range rows {
		rows[i].Key = fmt.Sprintf("%016d", i+1)
	}
	rows[2].ReassignmentDenied = true
	out := fromDomainUserTaskAssignmentResults(d.UserTaskAssignmentResults{Items: rows})
	require.True(t, out.Items[2].ReassignmentDenied)
	for i, item := range out.Items {
		require.Equal(t, statuses[i], item.Status)
		require.Equal(t, submissions[i], item.Submission)
		b, err := json.Marshal(item)
		require.NoError(t, err)
		require.JSONEq(t, fmt.Sprintf(`{"key":%q,"status":%q,"submission":%q}`, rows[i].Key, statuses[i], submissions[i]), string(b))
	}
	rows[0].Key = "changed"
	require.Equal(t, "0000000000000001", out.Items[0].Key)
	item := UserTaskAssignmentResult{Key: "key", Status: UserTaskAssignmentStatusFailed, Submission: UserTaskAssignmentSubmissionUnknown, StatusCode: 500, Error: "evidence"}
	b, err := json.Marshal(item)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"key","status":"failed","submission":"unknown","statusCode":500,"error":"evidence"}`, string(b))
	b, err = json.Marshal(fromDomainUserTaskAssignmentResults(d.UserTaskAssignmentResults{}))
	require.NoError(t, err)
	require.Equal(t, `{"items":[]}`, string(b))
}

type facadeAssignmentPlanAPI struct{ *facadeAssignmentAPI }

func (a *facadeAssignmentPlanAPI) GetNativeUserTask(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{Key: key, State: "CREATED", Assignee: "bob"}, nil
}
func TestPrepareUserTaskAssignmentsFacade(t *testing.T) {
	api := &facadeAssignmentPlanAPI{&facadeAssignmentAPI{submit: func(context.Context, string, string, ...services.CallOption) (d.UserTaskAssignmentResult, error) {
		t.Fatal("blocked task must not be assigned")
		return d.UserTaskAssignmentResult{}, nil
	}}}
	c := New(nil, nil, api, nil)
	plan, err := c.PrepareUserTaskAssignments(context.Background(), typex.Keys{"2251799813685249"}, "alice", 1)
	require.NoError(t, err)
	require.Equal(t, 1, plan.BlockedCount)
	require.Empty(t, plan.ReadyKeys)
	plan.ReadyKeys = typex.Keys{"2251799813685250"}
	plan.BlockedCount = 0
	result, err := c.ExecuteUserTaskAssignments(context.Background(), plan, false)
	require.ErrorIs(t, err, ferr.ErrConflict)
	require.Len(t, result.Items, 1)
	require.Equal(t, "2251799813685249", result.Items[0].Key)
	require.True(t, result.Items[0].ReassignmentDenied)
	require.Equal(t, UserTaskAssignmentSubmissionNotSubmitted, result.Items[0].Submission)
	require.Zero(t, result.Items[0].StatusCode)
	_, err = c.ExecuteUserTaskAssignments(context.Background(), UserTaskAssignmentPlan{}, false)
	require.ErrorIs(t, err, ferr.ErrLocalPrecondition)

	plan, err = c.PrepareUserTaskAssignments(context.Background(), typex.Keys{"2251799813685249"}, "bob", 1)
	require.NoError(t, err)
	require.Empty(t, plan.ReadyKeys)
	require.Zero(t, plan.BlockedCount)
	require.Equal(t, 1, plan.UnchangedCount)
	result, err = c.ExecuteUserTaskAssignments(context.Background(), plan, true)
	require.NoError(t, err)
	require.Equal(t, []UserTaskAssignmentResult{{Key: "2251799813685249", Status: UserTaskAssignmentStatusUnchanged, Submission: UserTaskAssignmentSubmissionNotSubmitted}}, result.Items)
}
