// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type fakeJobService struct {
	get    func(context.Context, string, ...services.CallOption) (d.Job, error)
	update func(context.Context, d.JobUpdateRequest, ...services.CallOption) (d.JobUpdateResult, error)
}

func (f fakeJobService) GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error) {
	return f.get(ctx, key, opts...)
}

func (f fakeJobService) UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error) {
	if f.update == nil {
		return d.JobUpdateResult{}, errors.New("unexpected update")
	}
	return f.update(ctx, request, opts...)
}

func TestClient_GetJob_Found(t *testing.T) {
	deadline := time.Date(2026, 5, 8, 10, 15, 0, 0, time.UTC)
	api := New(fakeJobService{
		get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
			require.Equal(t, "2251799813711967", key)
			return d.Job{
				Key:                key,
				State:              "FAILED",
				Retries:            2,
				Deadline:           &deadline,
				ProcessInstanceKey: "2251799813711000",
				ElementInstanceKey: "2251799813711001",
				ErrorCode:          "PAYMENT_ERROR",
				ErrorMessage:       "worker failed",
				TenantId:           "tenant-a",
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.GetJob(context.Background(), "2251799813711967")

	require.NoError(t, err)
	require.Equal(t, "2251799813711967", result.Key)
	require.Equal(t, "FAILED", result.State)
	require.Equal(t, int32(2), result.Retries)
	require.Equal(t, &deadline, result.Deadline)
	require.Equal(t, "2251799813711000", result.ProcessInstanceKey)
	require.Equal(t, "2251799813711001", result.ElementInstanceKey)
	require.Equal(t, "PAYMENT_ERROR", result.ErrorCode)
	require.Equal(t, "worker failed", result.ErrorMessage)
	require.Equal(t, "tenant-a", result.TenantId)
}

func TestClient_GetJob_NotFound(t *testing.T) {
	api := New(fakeJobService{
		get: func(_ context.Context, key string, _ ...services.CallOption) (d.Job, error) {
			require.Equal(t, "missing-job", key)
			return d.Job{}, d.ErrNotFound
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.GetJob(context.Background(), "missing-job")

	require.ErrorIs(t, err, ferrors.ErrNotFound)
	require.Empty(t, result)
}

func TestUpdateJobRetriesFacade_MutationFailureReturnsFailedResult(t *testing.T) {
	mutationErr := errors.New("camunda rejected update")
	api := New(fakeJobService{
		get: func(context.Context, string, ...services.CallOption) (d.Job, error) {
			t.Fatal("unexpected confirmation lookup after mutation failure")
			return d.Job{}, nil
		},
		update: func(_ context.Context, request d.JobUpdateRequest, _ ...services.CallOption) (d.JobUpdateResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			return d.JobUpdateResult{
				Key:           request.Key,
				MutationError: mutationErr.Error(),
			}, mutationErr
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.UpdateJob(context.Background(), UpdateRequest{Key: "2251799813711967"})

	require.Error(t, err)
	require.Equal(t, "mutation_failed", result.Status)
	require.False(t, result.MutationAccepted)
	require.Equal(t, mutationErr.Error(), result.Error)
}

func TestUpdateJobNoWaitFacade_MutationFailureReturnsFailedResult(t *testing.T) {
	retries := int32(3)
	mutationErr := errors.New("camunda rejected update")
	api := New(fakeJobService{
		get: func(context.Context, string, ...services.CallOption) (d.Job, error) {
			t.Fatal("unexpected confirmation lookup after no-wait mutation failure")
			return d.Job{}, nil
		},
		update: func(_ context.Context, request d.JobUpdateRequest, _ ...services.CallOption) (d.JobUpdateResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, &retries, request.Retries)
			require.True(t, request.SkipConfirmation)
			require.False(t, request.ConfirmRetries)
			return d.JobUpdateResult{
				Key:           request.Key,
				MutationError: mutationErr.Error(),
			}, mutationErr
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.UpdateJob(context.Background(), UpdateRequest{
		Key:     "2251799813711967",
		Retries: &retries,
		NoWait:  true,
	})

	require.Error(t, err)
	require.Equal(t, "mutation_failed", result.Status)
	require.False(t, result.MutationAccepted)
	require.Equal(t, mutationErr.Error(), result.Error)
}

func TestUpdateJobTimeoutOnlyFacade_SkipsDeadlineConfirmation(t *testing.T) {
	timeoutMillis := int64(300000)
	api := New(fakeJobService{
		get: func(context.Context, string, ...services.CallOption) (d.Job, error) {
			t.Fatal("unexpected lookup for timeout-only confirmation")
			return d.Job{}, nil
		},
		update: func(_ context.Context, request d.JobUpdateRequest, _ ...services.CallOption) (d.JobUpdateResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Nil(t, request.Retries)
			require.False(t, request.ConfirmRetries)
			require.NotNil(t, request.TimeoutMillis)
			require.Equal(t, timeoutMillis, *request.TimeoutMillis)
			return d.JobUpdateResult{
				Key:                request.Key,
				MutationAccepted:   true,
				ConfirmationStatus: "skipped",
				SubmittedTimeoutMS: request.TimeoutMillis,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.UpdateJob(context.Background(), UpdateRequest{
		Key:           "2251799813711967",
		TimeoutRaw:    "5m",
		TimeoutMillis: &timeoutMillis,
	})

	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
	require.True(t, result.MutationAccepted)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Nil(t, result.ConfirmedRetries)
	require.Equal(t, &timeoutMillis, result.SubmittedTimeoutMS)
}
