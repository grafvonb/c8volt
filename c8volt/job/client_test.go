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
	get     func(context.Context, string, ...services.CallOption) (d.Job, error)
	search  func(context.Context, d.JobSearchQuery, ...services.CallOption) (d.JobSearchResult, error)
	page    func(context.Context, d.JobSearchQuery, d.JobPageRequest, ...services.CallOption) (d.JobSearchPage, error)
	update  func(context.Context, d.JobUpdateRequest, ...services.CallOption) (d.JobUpdateResult, error)
	outcome func(context.Context, d.JobWorkerOutcomeRequest, ...services.CallOption) (d.JobWorkerOutcomeResult, error)
}

func (f fakeJobService) GetJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error) {
	return f.get(ctx, key, opts...)
}

func (f fakeJobService) SearchJobs(ctx context.Context, request d.JobSearchQuery, opts ...services.CallOption) (d.JobSearchResult, error) {
	if f.search == nil {
		return d.JobSearchResult{}, errors.New("unexpected search")
	}
	return f.search(ctx, request, opts...)
}

func (f fakeJobService) SearchJobsPage(ctx context.Context, request d.JobSearchQuery, page d.JobPageRequest, opts ...services.CallOption) (d.JobSearchPage, error) {
	if f.page == nil {
		return d.JobSearchPage{}, errors.New("unexpected search page")
	}
	return f.page(ctx, request, page, opts...)
}

func (f fakeJobService) UpdateJob(ctx context.Context, request d.JobUpdateRequest, opts ...services.CallOption) (d.JobUpdateResult, error) {
	if f.update == nil {
		return d.JobUpdateResult{}, errors.New("unexpected update")
	}
	return f.update(ctx, request, opts...)
}

func (f fakeJobService) SubmitJobWorkerOutcome(ctx context.Context, request d.JobWorkerOutcomeRequest, opts ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
	if f.outcome == nil {
		return d.JobWorkerOutcomeResult{}, errors.New("unexpected worker outcome")
	}
	return f.outcome(ctx, request, opts...)
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
				Type:               "payment-worker",
				Worker:             "worker-a",
				Kind:               "BPMN_ELEMENT",
				ListenerEventType:  "COMPLETING",
				ProcessInstanceKey: "2251799813711000",
				ElementInstanceKey: "2251799813711001",
				ElementId:          "charge-card",
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
	require.Equal(t, "payment-worker", result.Type)
	require.Equal(t, "worker-a", result.Worker)
	require.Equal(t, "BPMN_ELEMENT", result.Kind)
	require.Equal(t, "COMPLETING", result.ListenerEventType)
	require.Equal(t, "2251799813711000", result.ProcessInstanceKey)
	require.Equal(t, "2251799813711001", result.ElementInstanceKey)
	require.Equal(t, "charge-card", result.ElementId)
	require.Equal(t, "PAYMENT_ERROR", result.ErrorCode)
	require.Equal(t, "worker failed", result.ErrorMessage)
	require.Equal(t, "tenant-a", result.TenantId)
}

func TestClient_SearchJobs_MapsFoundationalQueryAndResults(t *testing.T) {
	retries := int32(0)
	api := New(fakeJobService{
		search: func(_ context.Context, request d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
			require.Equal(t, "FAILED", request.State)
			require.Equal(t, "payment-worker", request.Type)
			require.Equal(t, "2251799813711000", request.ProcessInstanceKey)
			require.Equal(t, "2251799813711001", request.ElementInstanceKey)
			require.Equal(t, "charge-card", request.ElementId)
			require.Equal(t, "worker-a", request.Worker)
			require.Equal(t, &retries, request.Retries)
			require.Equal(t, "BPMN_ELEMENT", request.Kind)
			require.Equal(t, "COMPLETING", request.ListenerEventType)
			require.Equal(t, int32(50), request.Limit)
			return d.JobSearchResult{
				Items: []d.Job{{Key: "2251799813711967", State: "FAILED", Type: request.Type}},
				Limit: request.Limit,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SearchJobs(context.Background(), SearchRequest{
		State:              "FAILED",
		Type:               "payment-worker",
		ProcessInstanceKey: "2251799813711000",
		ElementInstanceKey: "2251799813711001",
		ElementId:          "charge-card",
		Worker:             "worker-a",
		Retries:            &retries,
		Kind:               "BPMN_ELEMENT",
		ListenerEventType:  "COMPLETING",
		Limit:              50,
	})

	require.NoError(t, err)
	require.Equal(t, int32(50), result.Limit)
	require.Len(t, result.Items, 1)
	require.Equal(t, "2251799813711967", result.Items[0].Key)
	require.Equal(t, "payment-worker", result.Items[0].Type)
}

// TestClient_SearchJobs_PreservesZeroRetriesAndLimit protects search mapping
// for zero-retry failure discovery and caller-controlled result limits.
func TestClient_SearchJobs_PreservesZeroRetriesAndLimit(t *testing.T) {
	retries := int32(0)
	api := New(fakeJobService{
		search: func(_ context.Context, request d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
			require.NotNil(t, request.Retries)
			require.Equal(t, retries, *request.Retries)
			require.Equal(t, int32(25), request.Limit)
			return d.JobSearchResult{
				Items: []d.Job{{Key: "2251799813711967", Retries: *request.Retries}},
				Limit: request.Limit,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SearchJobs(context.Background(), SearchRequest{
		Retries: &retries,
		Limit:   25,
	})

	require.NoError(t, err)
	require.Equal(t, int32(25), result.Limit)
	require.Len(t, result.Items, 1)
	require.Equal(t, int32(0), result.Items[0].Retries)
}

// TestClient_SearchJobs_ForwardsPageCollectionControls verifies the facade
// preserves service-owned paging controls while mapping collected job rows.
func TestClient_SearchJobs_ForwardsPageCollectionControls(t *testing.T) {
	api := New(fakeJobService{
		search: func(_ context.Context, request d.JobSearchQuery, _ ...services.CallOption) (d.JobSearchResult, error) {
			require.Equal(t, int32(2), request.BatchSize)
			require.Equal(t, int32(3), request.Limit)
			return d.JobSearchResult{
				Items: []d.Job{
					{Key: "2251799813711967", State: "FAILED"},
					{Key: "2251799813711968", State: "FAILED"},
					{Key: "2251799813711969", State: "FAILED"},
				},
				Limit: request.Limit,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SearchJobs(context.Background(), SearchRequest{
		State:     "FAILED",
		BatchSize: 2,
		Limit:     3,
	})

	require.NoError(t, err)
	require.Equal(t, int32(3), result.Limit)
	require.Len(t, result.Items, 3)
	require.Equal(t, "2251799813711967", result.Items[0].Key)
	require.Equal(t, "2251799813711969", result.Items[2].Key)
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

func TestUpdateJobRetriesAndTimeoutFacade_PreservesUpdateRequestAndPlan(t *testing.T) {
	retries := int32(3)
	timeout := 5 * time.Minute
	timeoutMillis := int64(300000)
	api := New(fakeJobService{
		update: func(_ context.Context, request d.JobUpdateRequest, _ ...services.CallOption) (d.JobUpdateResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, &retries, request.Retries)
			require.Equal(t, &timeoutMillis, request.TimeoutMillis)
			require.Equal(t, "5m", request.RequestedTimeout)
			require.Equal(t, timeout, request.RequestedDuration)
			require.True(t, request.ConfirmRetries)
			require.False(t, request.SkipConfirmation)
			return d.JobUpdateResult{
				Key:                request.Key,
				MutationAccepted:   true,
				ConfirmationStatus: "confirmed",
				SubmittedRetries:   request.Retries,
				SubmittedTimeoutMS: request.TimeoutMillis,
				ConfirmedRetries:   request.Retries,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.UpdateJob(context.Background(), UpdateRequest{
		Key:            "2251799813711967",
		Retries:        &retries,
		Timeout:        &timeout,
		TimeoutRaw:     "5m",
		TimeoutMillis:  &timeoutMillis,
		ConfirmRetries: true,
		UpdatePlan: &UpdatePlan{
			Key:            "2251799813711967",
			Mode:           MutationModeUpdate,
			MaterialChange: true,
		},
	})

	require.NoError(t, err)
	require.Equal(t, "confirmed", result.Status)
	require.True(t, result.MutationAccepted)
	require.Equal(t, &retries, result.SubmittedRetries)
	require.Equal(t, &timeoutMillis, result.SubmittedTimeoutMS)
	require.NotNil(t, result.Plan)
	require.True(t, result.Plan.MutationSubmitted)
	require.Equal(t, MutationModeUpdate, result.Plan.Mode)
}

func TestClient_SubmitJobWorkerOutcome_MapsFoundationalRequestAndResult(t *testing.T) {
	retries := int32(0)
	backoffMillis := int64(300000)
	api := New(fakeJobService{
		outcome: func(_ context.Context, request d.JobWorkerOutcomeRequest, _ ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, d.JobWorkerOutcomeTechnicalFailure, request.Mode)
			require.Equal(t, "worker unavailable", request.Message)
			require.Equal(t, map[string]any{"attempt": float64(2)}, request.Variables)
			require.Equal(t, &retries, request.Retries)
			require.Equal(t, &backoffMillis, request.RetryBackoffMillis)
			require.True(t, request.SkipConfirmation)
			return d.JobWorkerOutcomeResult{
				Key:                request.Key,
				Mode:               request.Mode,
				MutationAccepted:   true,
				ConfirmationStatus: "skipped",
				SubmittedRetries:   request.Retries,
				SubmittedBackoffMS: request.RetryBackoffMillis,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{
		Key:                "2251799813711967",
		Mode:               WorkerOutcomeTechnicalFailure,
		Message:            "worker unavailable",
		Variables:          map[string]any{"attempt": float64(2)},
		Retries:            &retries,
		RetryBackoffMillis: &backoffMillis,
		NoWait:             true,
	})

	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
	require.Equal(t, WorkerOutcomeTechnicalFailure, result.Mode)
	require.True(t, result.MutationAccepted)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Equal(t, &retries, result.SubmittedRetries)
	require.Equal(t, &backoffMillis, result.SubmittedBackoffMS)
}

func TestClient_SubmitJobTechnicalFailure_MutationFailureReturnsFailedResult(t *testing.T) {
	retries := int32(0)
	mutationErr := errors.New("camunda rejected failure")
	api := New(fakeJobService{
		outcome: func(_ context.Context, request d.JobWorkerOutcomeRequest, _ ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, d.JobWorkerOutcomeTechnicalFailure, request.Mode)
			require.Equal(t, &retries, request.Retries)
			return d.JobWorkerOutcomeResult{
				Key:           request.Key,
				Mode:          request.Mode,
				MutationError: mutationErr.Error(),
			}, mutationErr
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{
		Key:     "2251799813711967",
		Mode:    WorkerOutcomeTechnicalFailure,
		Retries: &retries,
	})

	require.Error(t, err)
	require.Equal(t, "mutation_failed", result.Status)
	require.Equal(t, WorkerOutcomeTechnicalFailure, result.Mode)
	require.False(t, result.MutationAccepted)
	require.Equal(t, mutationErr.Error(), result.Error)
}

func TestClient_SubmitJobBPMNError_MapsRequestAndResult(t *testing.T) {
	api := New(fakeJobService{
		outcome: func(_ context.Context, request d.JobWorkerOutcomeRequest, _ ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, d.JobWorkerOutcomeBPMNError, request.Mode)
			require.Equal(t, "PAYMENT_DECLINED", request.ErrorCode)
			require.Equal(t, "card declined", request.Message)
			require.Equal(t, map[string]any{"approved": false}, request.Variables)
			require.True(t, request.SkipConfirmation)
			return d.JobWorkerOutcomeResult{
				Key:                request.Key,
				Mode:               request.Mode,
				MutationAccepted:   true,
				ConfirmationStatus: "skipped",
				SubmittedErrorCode: request.ErrorCode,
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{
		Key:       "2251799813711967",
		Mode:      WorkerOutcomeBPMNError,
		ErrorCode: "PAYMENT_DECLINED",
		Message:   "card declined",
		Variables: map[string]any{"approved": false},
		NoWait:    true,
	})

	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
	require.Equal(t, WorkerOutcomeBPMNError, result.Mode)
	require.True(t, result.MutationAccepted)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Equal(t, "PAYMENT_DECLINED", result.SubmittedErrorCode)
}

func TestClient_SubmitJobBPMNError_MutationFailureReturnsFailedResult(t *testing.T) {
	mutationErr := errors.New("camunda rejected BPMN error")
	api := New(fakeJobService{
		outcome: func(_ context.Context, request d.JobWorkerOutcomeRequest, _ ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, d.JobWorkerOutcomeBPMNError, request.Mode)
			require.Equal(t, "PAYMENT_DECLINED", request.ErrorCode)
			return d.JobWorkerOutcomeResult{
				Key:           request.Key,
				Mode:          request.Mode,
				MutationError: mutationErr.Error(),
			}, mutationErr
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{
		Key:       "2251799813711967",
		Mode:      WorkerOutcomeBPMNError,
		ErrorCode: "PAYMENT_DECLINED",
	})

	require.Error(t, err)
	require.Equal(t, "mutation_failed", result.Status)
	require.Equal(t, WorkerOutcomeBPMNError, result.Mode)
	require.False(t, result.MutationAccepted)
	require.Equal(t, mutationErr.Error(), result.Error)
}

func TestClient_SubmitJobCompletion_MapsRequestAndResult(t *testing.T) {
	api := New(fakeJobService{
		outcome: func(_ context.Context, request d.JobWorkerOutcomeRequest, _ ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, d.JobWorkerOutcomeCompletion, request.Mode)
			require.Equal(t, map[string]any{"approved": true}, request.Variables)
			require.True(t, request.SkipConfirmation)
			return d.JobWorkerOutcomeResult{
				Key:                request.Key,
				Mode:               request.Mode,
				MutationAccepted:   true,
				ConfirmationStatus: "skipped",
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{
		Key:       "2251799813711967",
		Mode:      WorkerOutcomeCompletion,
		Variables: map[string]any{"approved": true},
		NoWait:    true,
	})

	require.NoError(t, err)
	require.Equal(t, "submitted", result.Status)
	require.Equal(t, WorkerOutcomeCompletion, result.Mode)
	require.True(t, result.MutationAccepted)
	require.Equal(t, "skipped", result.ConfirmationStatus)
}

func TestClient_SubmitJobCompletion_MutationFailureReturnsFailedResult(t *testing.T) {
	mutationErr := errors.New("camunda rejected completion")
	api := New(fakeJobService{
		outcome: func(_ context.Context, request d.JobWorkerOutcomeRequest, _ ...services.CallOption) (d.JobWorkerOutcomeResult, error) {
			require.Equal(t, "2251799813711967", request.Key)
			require.Equal(t, d.JobWorkerOutcomeCompletion, request.Mode)
			return d.JobWorkerOutcomeResult{
				Key:           request.Key,
				Mode:          request.Mode,
				MutationError: mutationErr.Error(),
			}, mutationErr
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.SubmitJobWorkerOutcome(context.Background(), WorkerOutcomeRequest{
		Key:  "2251799813711967",
		Mode: WorkerOutcomeCompletion,
	})

	require.Error(t, err)
	require.Equal(t, "mutation_failed", result.Status)
	require.Equal(t, WorkerOutcomeCompletion, result.Mode)
	require.False(t, result.MutationAccepted)
	require.Equal(t, mutationErr.Error(), result.Error)
}
