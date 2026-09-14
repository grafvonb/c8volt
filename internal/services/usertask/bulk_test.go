// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package usertask

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type bulkUserTaskAPI struct {
	getNative func(context.Context, string, ...services.CallOption) (d.UserTask, error)
}

// GetUserTask rejects accidental use of the tenant-scoped legacy resolver path.
func (a *bulkUserTaskAPI) GetUserTask(context.Context, string, ...services.CallOption) (d.UserTask, error) {
	panic("bulk native reads must not call the legacy resolver")
}

// GetNativeUserTask delegates to the behavior configured by each bulk contract case.
func (a *bulkUserTaskAPI) GetNativeUserTask(ctx context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
	return a.getNative(ctx, key, opts...)
}

// SearchUserTasksPage rejects accidental use of search by the native bulk workflow.
func (a *bulkUserTaskAPI) SearchUserTasksPage(context.Context, d.UserTaskSearchQuery, d.UserTaskPageRequest, ...services.CallOption) (d.UserTaskSearchPage, error) {
	panic("bulk native reads must not call user-task search")
}

// TestGetUserTasksDeduplicatesStablyAndPreservesInputOrder verifies concurrent completion cannot reorder the first occurrence of each requested key.
func TestGetUserTasksDeduplicatesStablyAndPreservesInputOrder(t *testing.T) {
	t.Parallel()

	var mu sync.Mutex
	requested := make(typex.Keys, 0, 3)
	delays := map[string]time.Duration{
		"task-c": 30 * time.Millisecond,
		"task-a": 20 * time.Millisecond,
		"task-b": 10 * time.Millisecond,
	}
	api := &bulkUserTaskAPI{getNative: func(ctx context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
		mu.Lock()
		requested = append(requested, key)
		mu.Unlock()
		select {
		case <-ctx.Done():
			return d.UserTask{}, ctx.Err()
		case <-time.After(delays[key]):
			return d.UserTask{Key: key}, nil
		}
	}}

	tasks, err := GetUserTasks(context.Background(), api, typex.Keys{"task-c", "task-a", "task-c", "task-b", "task-a"}, 3)

	require.NoError(t, err)
	require.Equal(t, []d.UserTask{{Key: "task-c"}, {Key: "task-a"}, {Key: "task-b"}}, tasks)
	mu.Lock()
	require.ElementsMatch(t, typex.Keys{"task-c", "task-a", "task-b"}, requested)
	mu.Unlock()
}

// TestGetUserTasksReturnsJoinedFailuresWithoutSubset verifies every non-fail-fast read is attempted while any failure makes the whole lookup unsuccessful.
func TestGetUserTasksReturnsJoinedFailuresWithoutSubset(t *testing.T) {
	t.Parallel()

	errMissing := errors.New("task missing")
	errDenied := errors.New("task denied")
	api := &bulkUserTaskAPI{getNative: func(_ context.Context, key string, _ ...services.CallOption) (d.UserTask, error) {
		switch key {
		case "missing":
			return d.UserTask{}, errMissing
		case "denied":
			return d.UserTask{}, errDenied
		default:
			return d.UserTask{Key: key}, nil
		}
	}}

	tasks, err := GetUserTasks(context.Background(), api, typex.Keys{"found", "missing", "denied"}, 1)

	require.Nil(t, tasks)
	require.ErrorIs(t, err, errMissing)
	require.ErrorIs(t, err, errDenied)
}

// TestGetUserTasksPropagatesPreCanceledContext verifies cancellation prevents requests and cannot be mistaken for a successful zero-valued collection.
func TestGetUserTasksPropagatesPreCanceledContext(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var calls atomic.Int32
	api := &bulkUserTaskAPI{getNative: func(context.Context, string, ...services.CallOption) (d.UserTask, error) {
		calls.Add(1)
		return d.UserTask{}, nil
	}}

	tasks, err := GetUserTasks(ctx, api, typex.Keys{"task-a", "task-b"}, 2)

	require.Nil(t, tasks)
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, calls.Load())
}

// TestGetUserTasksFailFastStopsUnstartedReads verifies the first failure cancels work that has not begun.
func TestGetUserTasksFailFastStopsUnstartedReads(t *testing.T) {
	t.Parallel()

	errRead := errors.New("read failed")
	var mu sync.Mutex
	requested := make(typex.Keys, 0, 1)
	var sawFailFast atomic.Bool
	api := &bulkUserTaskAPI{getNative: func(_ context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
		cfg := services.ApplyCallOptions(opts)
		sawFailFast.Store(cfg.FailFast)
		mu.Lock()
		requested = append(requested, key)
		mu.Unlock()
		if key == "fail" {
			return d.UserTask{}, errRead
		}
		return d.UserTask{Key: key}, nil
	}}

	tasks, err := GetUserTasks(context.Background(), api, typex.Keys{"fail", "not-started-a", "not-started-b"}, 1, services.WithFailFast())

	require.Nil(t, tasks)
	require.ErrorIs(t, err, errRead)
	require.True(t, sawFailFast.Load())
	mu.Lock()
	require.Equal(t, typex.Keys{"fail"}, requested)
	mu.Unlock()
}

// TestGetUserTasksHonorsWorkerLimitAndPropagatesOptions verifies explicit concurrency bounds and unchanged call-option forwarding.
func TestGetUserTasksHonorsWorkerLimitAndPropagatesOptions(t *testing.T) {
	t.Parallel()

	var active atomic.Int32
	var maximum atomic.Int32
	var sawNoWorkerLimit atomic.Bool
	api := &bulkUserTaskAPI{getNative: func(_ context.Context, key string, opts ...services.CallOption) (d.UserTask, error) {
		cfg := services.ApplyCallOptions(opts)
		sawNoWorkerLimit.Store(cfg.NoWorkerLimit)
		current := active.Add(1)
		for observed := maximum.Load(); current > observed && !maximum.CompareAndSwap(observed, current); observed = maximum.Load() {
		}
		time.Sleep(10 * time.Millisecond)
		active.Add(-1)
		return d.UserTask{Key: key}, nil
	}}

	tasks, err := GetUserTasks(context.Background(), api, typex.Keys{"task-a", "task-b", "task-c", "task-d"}, 2, services.WithNoWorkerLimit())

	require.NoError(t, err)
	require.Len(t, tasks, 4)
	require.EqualValues(t, 2, maximum.Load())
	require.True(t, sawNoWorkerLimit.Load())
}

// TestGetUserTasksEmptyInputReturnsInitializedCollection verifies no worker or adapter request is created for an empty selection.
func TestGetUserTasksEmptyInputReturnsInitializedCollection(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	api := &bulkUserTaskAPI{getNative: func(context.Context, string, ...services.CallOption) (d.UserTask, error) {
		calls.Add(1)
		return d.UserTask{}, nil
	}}

	tasks, err := GetUserTasks(context.Background(), api, nil, 0)

	require.NoError(t, err)
	require.NotNil(t, tasks)
	require.Empty(t, tasks)
	require.Zero(t, calls.Load())
}
