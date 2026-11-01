// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateNProcessInstancesBusinessIDSnapshot freezes caller identity and gives each attempt its own mutable copy.
func TestCreateNProcessInstancesBusinessIDSnapshot(t *testing.T) {
	t.Parallel()
	original := new("  order,AbC  ")
	calls := 0
	api := stubBulkProcessInstanceAPI{create: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
		calls++
		assert.Equal(t, "  order,AbC  ", *data.BusinessId)
		assert.NotSame(t, original, data.BusinessId)
		*original = "caller changed after scheduling"
		*data.BusinessId = "worker changed its input"
		if calls == 2 {
			return d.ProcessInstanceCreation{Key: "failed"}, d.ErrConflict
		}
		return d.ProcessInstanceCreation{Key: fmt.Sprint(calls), BusinessId: new("observed")}, nil
	}}
	got, err := CreateNProcessInstances(context.Background(), api, slog.New(slog.NewTextHandler(io.Discard, nil)), d.ProcessInstanceData{BusinessId: original}, 4, 1)
	require.ErrorIs(t, err, d.ErrConflict)
	require.Equal(t, []d.ProcessInstanceCreation{{Key: "1", BusinessId: new("observed")}, {Key: "3", BusinessId: new("observed")}, {Key: "4", BusinessId: new("observed")}}, got)
	require.Equal(t, 4, calls)
}

// TestCreateNProcessInstancesBusinessIDConcurrent retains late successes amid bounded out-of-order failures without zero slots.
func TestCreateNProcessInstancesBusinessIDConcurrent(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entered := make(chan struct{}, 2)
	release := make(chan struct{})
	var calls, active, peak atomic.Int32
	api := stubBulkProcessInstanceAPI{create: func(ctx context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
		n := calls.Add(1)
		now := active.Add(1)
		defer active.Add(-1)
		for old := peak.Load(); now > old; old = peak.Load() {
			if peak.CompareAndSwap(old, now) {
				break
			}
		}
		assert.Equal(t, new("order"), data.BusinessId)
		if n <= 2 {
			entered <- struct{}{}
		}
		if n == 1 {
			select {
			case <-release:
			case <-ctx.Done():
				return d.ProcessInstanceCreation{}, ctx.Err()
			}
			return d.ProcessInstanceCreation{Key: "late", BusinessId: new("old")}, nil
		}
		if n == 2 {
			return d.ProcessInstanceCreation{Key: "failed"}, d.ErrConflict
		}
		return d.ProcessInstanceCreation{Key: fmt.Sprint(n), BusinessId: new("old")}, nil
	}}
	type outcome struct {
		items []d.ProcessInstanceCreation
		err   error
	}
	done := make(chan outcome, 1)
	go func() {
		items, err := CreateNProcessInstances(ctx, api, slog.New(slog.NewTextHandler(io.Discard, nil)), d.ProcessInstanceData{BusinessId: new("order")}, 6, 2, services.WithNoWorkerLimit())
		done <- outcome{items, err}
	}()
	for range 2 {
		select {
		case <-entered:
		case <-ctx.Done():
			t.Fatal("workers did not start")
		}
	}
	// Keep the first success pending until every other attempt has completed.
	for calls.Load() < 6 {
		select {
		case <-ctx.Done():
			t.Fatal("workers stalled")
		case <-time.After(time.Millisecond):
		}
	}
	close(release)
	result := <-done
	require.ErrorIs(t, result.err, d.ErrConflict)
	require.Len(t, result.items, 5)
	keys := make([]string, 0, len(result.items))
	for _, item := range result.items {
		keys = append(keys, item.Key)
		require.Equal(t, new("old"), item.BusinessId)
	}
	require.ElementsMatch(t, []string{"late", "3", "4", "5", "6"}, keys)
	require.EqualValues(t, 2, peak.Load())
	// After the two initial workers, one worker consumes the remaining slots in order.
	var ordered []string
	for _, key := range keys {
		if key != "late" {
			ordered = append(ordered, key)
		}
	}
	require.Equal(t, []string{"3", "4", "5", "6"}, ordered)
}

// TestCreateNProcessInstancesBusinessIDFailFastAndCancellation excludes all unattempted slots and retains error classes.
func TestCreateNProcessInstancesBusinessIDFailFastAndCancellation(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"fail fast", "cancel in flight", "pre-canceled", "cancel after success", "all failed", "zero count"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "pre-canceled" {
				cancel()
			}
			calls := 0
			api := stubBulkProcessInstanceAPI{create: func(ctx context.Context, _ d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
				calls++
				switch mode {
				case "cancel in flight":
					cancel()
					return d.ProcessInstanceCreation{}, ctx.Err()
				case "cancel after success":
					cancel()
					return d.ProcessInstanceCreation{Key: "1", BusinessId: new("observed")}, nil
				case "all failed":
					return d.ProcessInstanceCreation{Key: "failed"}, d.ErrConflict
				}
				if calls == 1 {
					return d.ProcessInstanceCreation{Key: "1", BusinessId: new("observed")}, nil
				}
				return d.ProcessInstanceCreation{Key: "failed"}, d.ErrConflict
			}}
			opts := []services.CallOption{services.WithSuppressWorkflowDetailLogs()}
			if mode == "fail fast" {
				opts = append(opts, services.WithFailFast())
			}
			n := 5
			if mode == "zero count" {
				n = 0
			}
			got, err := CreateNProcessInstances(ctx, api, slog.New(slog.NewTextHandler(io.Discard, nil)), d.ProcessInstanceData{BusinessId: new("order")}, n, 1, opts...)
			switch mode {
			case "fail fast":
				require.ErrorIs(t, err, d.ErrConflict)
				require.Len(t, got, 1)
				require.Equal(t, 2, calls)
			case "cancel after success":
				require.ErrorIs(t, err, context.Canceled)
				require.Len(t, got, 1)
				require.Equal(t, 1, calls)
			case "cancel in flight":
				require.ErrorIs(t, err, context.Canceled)
				require.Empty(t, got)
				require.Equal(t, 1, calls)
			case "pre-canceled":
				require.ErrorIs(t, err, context.Canceled)
				require.Empty(t, got)
				require.Zero(t, calls)
			case "all failed":
				require.ErrorIs(t, err, d.ErrConflict)
				require.Empty(t, got)
				require.Equal(t, 5, calls)
			case "zero count":
				require.NoError(t, err)
				require.Nil(t, got)
				require.Zero(t, calls)
			}
		})
	}
}
