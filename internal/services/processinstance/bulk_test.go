// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

type stubProcessInstanceCreator struct {
	create func(context.Context, d.ProcessInstanceData, ...services.CallOption) (d.ProcessInstanceCreation, error)
}

// CreateProcessInstance delegates creation to the configured test callback.
func (s stubProcessInstanceCreator) CreateProcessInstance(ctx context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
	if s.create == nil {
		return d.ProcessInstanceCreation{}, errors.New("unexpected process-instance creation")
	}
	return s.create(ctx, data, opts...)
}

// TestCreateProcessInstancesPreservesOrderAndOptions verifies the service-owned create-many workflow remains sequential and option-aware.
func TestCreateProcessInstancesPreservesOrderAndOptions(t *testing.T) {
	seen := []string{}
	got, err := CreateProcessInstances(context.Background(), stubProcessInstanceCreator{
		create: func(_ context.Context, data d.ProcessInstanceData, opts ...services.CallOption) (d.ProcessInstanceCreation, error) {
			require.True(t, services.ApplyCallOptions(opts).IgnoreTenant)
			seen = append(seen, data.BpmnProcessId)
			return d.ProcessInstanceCreation{Key: "created-" + data.BpmnProcessId, BpmnProcessId: data.BpmnProcessId}, nil
		},
	}, []d.ProcessInstanceData{
		{BpmnProcessId: "alpha"},
		{BpmnProcessId: "beta"},
	}, services.WithIgnoreTenant())

	require.NoError(t, err)
	require.Equal(t, []string{"alpha", "beta"}, seen)
	require.Equal(t, []d.ProcessInstanceCreation{
		{Key: "created-alpha", BpmnProcessId: "alpha"},
		{Key: "created-beta", BpmnProcessId: "beta"},
	}, got)
}

// TestCreateProcessInstancesStopsOnFirstError verifies the previous fail-on-first-error behavior stays intact.
func TestCreateProcessInstancesStopsOnFirstError(t *testing.T) {
	seen := []string{}
	wantErr := errors.New("create failed")
	got, err := CreateProcessInstances(context.Background(), stubProcessInstanceCreator{
		create: func(_ context.Context, data d.ProcessInstanceData, _ ...services.CallOption) (d.ProcessInstanceCreation, error) {
			seen = append(seen, data.BpmnProcessId)
			if data.BpmnProcessId == "beta" {
				return d.ProcessInstanceCreation{}, wantErr
			}
			return d.ProcessInstanceCreation{Key: "created-" + data.BpmnProcessId}, nil
		},
	}, []d.ProcessInstanceData{
		{BpmnProcessId: "alpha"},
		{BpmnProcessId: "beta"},
		{BpmnProcessId: "gamma"},
	})

	require.ErrorIs(t, err, wantErr)
	require.Nil(t, got)
	require.Equal(t, []string{"alpha", "beta"}, seen)
}

type stubBulkProcessInstanceAPI struct {
	API
	cancel func(context.Context, string, ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error)
	delete func(context.Context, string, ...services.CallOption) (d.DeleteResponse, error)
}

type lockedLogBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

// Write appends log bytes while allowing concurrent progress logging and assertions.
func (b *lockedLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// String returns the buffered log output while writes may still be in flight.
func (b *lockedLogBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// CancelProcessInstance delegates cancellation to the configured test callback.
func (s stubBulkProcessInstanceAPI) CancelProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.CancelResponse, []d.ProcessInstance, error) {
	if s.cancel == nil {
		return d.CancelResponse{}, nil, errors.New("unexpected process-instance cancellation")
	}
	return s.cancel(ctx, key, opts...)
}

// DeleteProcessInstance delegates deletion to the configured test callback.
func (s stubBulkProcessInstanceAPI) DeleteProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.DeleteResponse, error) {
	if s.delete == nil {
		return d.DeleteResponse{}, errors.New("unexpected process-instance deletion")
	}
	return s.delete(ctx, key, opts...)
}

// TestDeleteProcessInstancesLogsProgressWhileRootDeleteRuns verifies long root-tree deletes produce durable progress lines before the final summary.
func TestDeleteProcessInstancesLogsProgressWhileRootDeleteRuns(t *testing.T) {
	oldInterval := processInstanceBulkProgressInterval
	processInstanceBulkProgressInterval = 10 * time.Millisecond
	t.Cleanup(func() { processInstanceBulkProgressInterval = oldInterval })

	var logBuf lockedLogBuffer
	log := slog.New(logging.NewPlainHandler(&logBuf, slog.LevelInfo))
	started := make(chan struct{})
	release := make(chan struct{})
	var startedOnce sync.Once
	api := stubBulkProcessInstanceAPI{
		delete: func(ctx context.Context, key string, _ ...services.CallOption) (d.DeleteResponse, error) {
			require.Equal(t, "root-1", key)
			startedOnce.Do(func() { close(started) })
			select {
			case <-ctx.Done():
				return d.DeleteResponse{}, ctx.Err()
			case <-release:
				return d.DeleteResponse{Ok: true, StatusCode: 204, Status: "204 No Content"}, nil
			}
		},
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := DeleteProcessInstances(context.Background(), api, log, typex.Keys{"root-1"}, 1, 4)
		errCh <- err
	}()

	<-started
	require.Eventually(t, func() bool {
		return strings.Contains(logBuf.String(), "pi delete progress; roots 0/1 done, affected 4")
	}, time.Second, 10*time.Millisecond)
	close(release)

	require.NoError(t, <-errCh)
	require.Contains(t, logBuf.String(), "pi delete done; roots 1, affected 4, ok 1, failed 0")
}
