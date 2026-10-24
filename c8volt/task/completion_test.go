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

// facadeCompletionAPI rejects operations other than native lookup and completion.
type facadeCompletionAPI struct {
	utsvc.API
	submit func(context.Context, string, map[string]any, ...services.CallOption) (d.UserTaskCompletionResult, error)
}

// GetNativeUserTask models one successful preflight lookup for facade wiring tests.
func (a *facadeCompletionAPI) GetNativeUserTask(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
	return d.UserTask{Key: key, State: "CREATED", Assignee: "alice"}, nil
}

// CompleteUserTask delegates only the native operation exercised by the facade.
func (a *facadeCompletionAPI) CompleteUserTask(ctx context.Context, key string, vars map[string]any, opts ...services.CallOption) (d.UserTaskCompletionResult, error) {
	return a.submit(ctx, key, vars, opts...)
}

// TestCompleteUserTasksFacadeInputs preserves constructor composition, optional payloads and explicit-key options.
func TestCompleteUserTasksFacadeInputs(t *testing.T) {
	t.Parallel()
	for _, extended := range []bool{false, true} {
		for _, vars := range []map[string]any{nil, {}, {"nested": map[string]any{"list": []any{json.Number("9007199254740993"), true, nil, "text"}}}} {
			keys := typex.Keys{"0000000000000002", "0000000000000001", "0000000000000002"}
			expectedJSON, err := json.Marshal(vars)
			require.NoError(t, err)
			var calls []string
			api := &facadeCompletionAPI{submit: func(_ context.Context, key string, got map[string]any, opts ...services.CallOption) (d.UserTaskCompletionResult, error) {
				calls = append(calls, key)
				gotJSON, err := json.Marshal(got)
				require.NoError(t, err)
				require.JSONEq(t, string(expectedJSON), string(gotJSON))
				cfg := services.ApplyCallOptions(opts)
				require.True(t, cfg.Verbose && cfg.FailFast && cfg.NoWorkerLimit && cfg.IgnoreTenant && cfg.SuppressWorkflowDetailLogs)
				// The public caller retains ownership even after workers have begun.
				keys[1] = "changed"
				if vars != nil {
					vars["added"] = true
					if nested, ok := vars["nested"].(map[string]any); ok {
						nested["list"].([]any)[0] = "changed"
					}
				}
				return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusAccepted, Submission: d.UserTaskCompletionSubmissionAccepted, StatusCode: 204}, nil
			}}
			var client API = New(nil, nil, api, nil)
			if extended {
				client = NewWithVariableUpdates(nil, nil, api, nil, nil)
			}
			out, err := client.CompleteUserTasks(context.Background(), keys, vars, 1, options.WithVerbose(), options.WithFailFast(), options.WithNoWorkerLimit(), options.WithIgnoreTenant(), options.WithSuppressWorkflowDetailLogs())
			require.NoError(t, err)
			require.Equal(t, []string{"0000000000000002", "0000000000000001"}, calls)
			require.Len(t, out.Items, 2)
			require.Equal(t, UserTaskCompletionStatusAccepted, out.Items[1].Status)
			require.Equal(t, UserTaskCompletionSubmissionAccepted, out.Items[1].Submission)
		}
	}
}

// TestCompleteUserTasksFacadeWorkers proves the requested concurrency bound reaches the coordinator unchanged.
func TestCompleteUserTasksFacadeWorkers(t *testing.T) {
	t.Parallel()
	var active, peak atomic.Int32
	var once sync.Once
	ready := make(chan struct{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	api := &facadeCompletionAPI{submit: func(ctx context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
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
			return d.UserTaskCompletionResult{}, ctx.Err()
		}
		return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusAccepted, Submission: d.UserTaskCompletionSubmissionAccepted}, nil
	}}
	keys := make(typex.Keys, 8)
	for i := range keys {
		keys[i] = fmt.Sprintf("%016d", i+1)
	}
	out, err := New(nil, nil, api, nil).CompleteUserTasks(ctx, keys, nil, 2, options.WithNoWorkerLimit())
	require.NoError(t, err)
	require.Equal(t, int32(2), peak.Load())
	require.Len(t, out.Items, 8)
}

// TestCompleteUserTasksFacadeErrors preserves partial machine evidence while converting the original domain class.
func TestCompleteUserTasksFacadeErrors(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		cause      error
		class      error
		status     d.UserTaskCompletionStatus
		submission d.UserTaskCompletionSubmission
		code       int
	}{
		{"forbidden", d.ErrForbidden, ferr.ErrLocalPrecondition, d.UserTaskCompletionStatusFailed, d.UserTaskCompletionSubmissionRejected, 403},
		{"missing", d.ErrNotFound, ferr.ErrNotFound, d.UserTaskCompletionStatusFailed, d.UserTaskCompletionSubmissionRejected, 404},
		{"conflict", d.ErrConflict, ferr.ErrConflict, d.UserTaskCompletionStatusFailed, d.UserTaskCompletionSubmissionRejected, 409},
		{"timeout", d.ErrGatewayTimeout, ferr.ErrTimeout, d.UserTaskCompletionStatusTimedOut, d.UserTaskCompletionSubmissionUnknown, 504},
		{"cancel", context.Canceled, context.Canceled, d.UserTaskCompletionStatusCanceled, d.UserTaskCompletionSubmissionUnknown, 0},
		{"unavailable", d.ErrUnavailable, ferr.ErrUnavailable, d.UserTaskCompletionStatusFailed, d.UserTaskCompletionSubmissionUnknown, 503},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keys := typex.Keys{"0000000000000001", "0000000000000002", "0000000000000003"}
			evidence := "POST https://fake.invalid/completion body=secret"
			api := &facadeCompletionAPI{submit: func(_ context.Context, key string, _ map[string]any, _ ...services.CallOption) (d.UserTaskCompletionResult, error) {
				if key == keys[0] {
					return d.UserTaskCompletionResult{Key: key, Status: d.UserTaskCompletionStatusAccepted, Submission: d.UserTaskCompletionSubmissionAccepted, StatusCode: 204}, nil
				}
				return d.UserTaskCompletionResult{Key: key, Status: tc.status, Submission: tc.submission, StatusCode: tc.code, Error: evidence}, fmt.Errorf("%w: %s", tc.cause, evidence)
			}}
			out, err := New(nil, nil, api, nil).CompleteUserTasks(context.Background(), keys, nil, 1, options.WithFailFast())
			require.ErrorIs(t, err, tc.class)
			require.Equal(t, ferr.ExitCode(ferr.FromDomain(tc.cause)), ferr.ExitCode(err))
			require.Len(t, out.Items, 3)
			require.Equal(t, UserTaskCompletionStatusAccepted, out.Items[0].Status)
			require.Equal(t, UserTaskCompletionResult{Key: keys[1], Status: UserTaskCompletionStatus(tc.status), Submission: UserTaskCompletionSubmission(tc.submission), StatusCode: tc.code, Error: evidence}, out.Items[1])
			require.Equal(t, UserTaskCompletionResult{Key: keys[2], Status: UserTaskCompletionStatusNotAttempted, Submission: UserTaskCompletionSubmissionNotSubmitted}, out.Items[2])
			require.Contains(t, err.Error(), evidence)
		})
	}
}

// TestCompleteUserTasksFacadeValidationAndPreview keeps invalid requests empty and previews entirely local.
func TestCompleteUserTasksFacadeValidationAndPreview(t *testing.T) {
	t.Parallel()
	key := typex.Keys{"0000000000000001"}
	cycle := map[string]any{}
	cycle["self"] = cycle
	for _, tc := range []struct {
		name    string
		api     utsvc.API
		keys    typex.Keys
		vars    map[string]any
		workers int
		opts    []options.FacadeOption
		class   error
	}{
		{name: "missing dependency", keys: key, class: ferr.ErrLocalPrecondition},
		{name: "empty selection", api: &facadeCompletionAPI{}, class: ferr.ErrInvalidInput},
		{name: "invalid key", api: &facadeCompletionAPI{}, keys: append(key, "bad"), class: ferr.ErrInvalidInput},
		{name: "invalid payload", api: &facadeCompletionAPI{}, keys: key, vars: cycle, class: ferr.ErrInvalidInput},
		{name: "negative workers", api: &facadeCompletionAPI{}, keys: key, workers: -1, class: ferr.ErrInvalidInput},
		{name: "no wait", api: &facadeCompletionAPI{}, keys: key, opts: []options.FacadeOption{options.WithNoWait()}, class: ferr.ErrInvalidInput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := New(nil, nil, tc.api, nil).CompleteUserTasks(context.Background(), tc.keys, tc.vars, tc.workers, tc.opts...)
			require.ErrorIs(t, err, tc.class)
			b, err := json.Marshal(out)
			require.NoError(t, err)
			require.Equal(t, `{"items":[]}`, string(b))
		})
	}
	out, err := New(nil, nil, &facadeCompletionAPI{}, nil).CompleteUserTasks(context.Background(), key, map[string]any{}, 0, options.WithDryRun())
	require.NoError(t, err)
	b, err := json.Marshal(out)
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[{"key":"0000000000000001","status":"preview","submission":"not_submitted"}]}`, string(b))
}

// TestCompletionPublicModels covers every enum, JSON omissions, initialized items and independent output ownership.
func TestCompletionPublicModels(t *testing.T) {
	t.Parallel()
	rows := []d.UserTaskCompletionResult{
		{Status: d.UserTaskCompletionStatusPreview, Submission: d.UserTaskCompletionSubmissionNotSubmitted},
		{Status: d.UserTaskCompletionStatusAccepted, Submission: d.UserTaskCompletionSubmissionAccepted},
		{Status: d.UserTaskCompletionStatusFailed, Submission: d.UserTaskCompletionSubmissionRejected},
		{Status: d.UserTaskCompletionStatusTimedOut, Submission: d.UserTaskCompletionSubmissionUnknown},
		{Status: d.UserTaskCompletionStatusCanceled, Submission: d.UserTaskCompletionSubmissionUnknown},
		{Status: d.UserTaskCompletionStatusNotAttempted, Submission: d.UserTaskCompletionSubmissionNotSubmitted},
	}
	statuses := []UserTaskCompletionStatus{UserTaskCompletionStatusPreview, UserTaskCompletionStatusAccepted, UserTaskCompletionStatusFailed, UserTaskCompletionStatusTimedOut, UserTaskCompletionStatusCanceled, UserTaskCompletionStatusNotAttempted}
	submissions := []UserTaskCompletionSubmission{UserTaskCompletionSubmissionNotSubmitted, UserTaskCompletionSubmissionAccepted, UserTaskCompletionSubmissionRejected, UserTaskCompletionSubmissionUnknown, UserTaskCompletionSubmissionUnknown, UserTaskCompletionSubmissionNotSubmitted}
	for i := range rows {
		rows[i].Key = fmt.Sprintf("%016d", i+1)
	}
	out := fromDomainUserTaskCompletionResults(d.UserTaskCompletionResults{Items: rows})
	for i, item := range out.Items {
		require.Equal(t, statuses[i], item.Status)
		require.Equal(t, submissions[i], item.Submission)
		b, err := json.Marshal(item)
		require.NoError(t, err)
		require.JSONEq(t, fmt.Sprintf(`{"key":%q,"status":%q,"submission":%q}`, rows[i].Key, statuses[i], submissions[i]), string(b))
	}
	rows[0].Key = "changed"
	require.Equal(t, "0000000000000001", out.Items[0].Key)
	item := UserTaskCompletionResult{Key: "key", Status: UserTaskCompletionStatusFailed, Submission: UserTaskCompletionSubmissionUnknown, StatusCode: 500, Error: "evidence"}
	b, err := json.Marshal(item)
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"key","status":"failed","submission":"unknown","statusCode":500,"error":"evidence"}`, string(b))
	b, err = json.Marshal(fromDomainUserTaskCompletionResults(d.UserTaskCompletionResults{}))
	require.NoError(t, err)
	require.Equal(t, `{"items":[]}`, string(b))
}
