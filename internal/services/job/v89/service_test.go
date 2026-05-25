// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

type mockJobClient struct {
	searchJobsWithResponse    func(context.Context, camundav89.SearchJobsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error)
	updateJobWithResponse     func(context.Context, camundav89.JobKey, camundav89.UpdateJobJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error)
	throwJobErrorWithResponse func(context.Context, camundav89.JobKey, camundav89.ThrowJobErrorJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.ThrowJobErrorResponse, error)
	failJobWithResponse       func(context.Context, camundav89.JobKey, camundav89.FailJobJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.FailJobResponse, error)
}

func (m *mockJobClient) SearchJobsWithResponse(ctx context.Context, body camundav89.SearchJobsJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
	return m.searchJobsWithResponse(ctx, body, reqEditors...)
}

func (m *mockJobClient) UpdateJobWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.UpdateJobJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error) {
	if m.updateJobWithResponse == nil {
		panic("unexpected UpdateJobWithResponse call")
	}
	return m.updateJobWithResponse(ctx, jobKey, body, reqEditors...)
}

func (m *mockJobClient) CompleteJobWithResponse(context.Context, camundav89.JobKey, camundav89.CompleteJobJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.CompleteJobResponse, error) {
	panic("unexpected CompleteJobWithResponse call")
}

func (m *mockJobClient) ThrowJobErrorWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.ThrowJobErrorJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.ThrowJobErrorResponse, error) {
	if m.throwJobErrorWithResponse == nil {
		panic("unexpected ThrowJobErrorWithResponse call")
	}
	return m.throwJobErrorWithResponse(ctx, jobKey, body, reqEditors...)
}

func (m *mockJobClient) FailJobWithResponse(ctx context.Context, jobKey camundav89.JobKey, body camundav89.FailJobJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.FailJobResponse, error) {
	if m.failJobWithResponse == nil {
		panic("unexpected FailJobWithResponse call")
	}
	return m.failJobWithResponse(ctx, jobKey, body, reqEditors...)
}

func TestSearchJobsByKey(t *testing.T) {
	deadline := time.Date(2026, 5, 8, 10, 15, 0, 0, time.UTC)
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(_ context.Context, body camundav89.SearchJobsJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			requireJobSearchBody(t, body, "2251799813711967")
			return &camundav89.SearchJobsResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav89.JobSearchQueryResult{
					Items: []camundav89.JobSearchResult{{
						JobKey:             "2251799813711967",
						State:              camundav89.JobStateEnum("FAILED"),
						Retries:            2,
						Deadline:           &deadline,
						ProcessInstanceKey: "2251799813711000",
						ElementInstanceKey: "2251799813711001",
						ErrorCode:          stringPtr("PAYMENT_ERROR"),
						ErrorMessage:       stringPtr("worker failed"),
						TenantId:           "tenant-a",
					}},
				},
			}, nil
		},
	})

	job, err := svc.GetJob(context.Background(), "2251799813711967")

	require.NoError(t, err)
	require.Equal(t, d.Job{
		Key:                "2251799813711967",
		State:              "FAILED",
		Retries:            2,
		Deadline:           &deadline,
		ProcessInstanceKey: "2251799813711000",
		ElementInstanceKey: "2251799813711001",
		ErrorCode:          "PAYMENT_ERROR",
		ErrorMessage:       "worker failed",
		TenantId:           "tenant-a",
	}, job)
}

func TestService_GetJob_NotFound(t *testing.T) {
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(_ context.Context, body camundav89.SearchJobsJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			requireJobSearchBody(t, body, "missing-job")
			return &camundav89.SearchJobsResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      &camundav89.JobSearchQueryResult{},
			}, nil
		},
	})

	job, err := svc.GetJob(context.Background(), "missing-job")

	require.ErrorIs(t, err, d.ErrNotFound)
	require.Empty(t, job)
}

// TestService_SearchJobs_ConstructsFiltersAndConvertsRows verifies v8.9 job
// discovery maps every supported search filter into the generated request body.
func TestService_SearchJobs_ConstructsFiltersAndConvertsRows(t *testing.T) {
	retries := int32(0)
	elementID := camundav89.ElementId("charge-card")
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(_ context.Context, body camundav89.SearchJobsJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			requireJobSearchFilterJSON(t, body, map[string]any{
				"state":              "FAILED",
				"type":               "payment-worker",
				"processInstanceKey": "2251799813711000",
				"elementInstanceKey": "2251799813711001",
				"elementId":          "charge-card",
				"worker":             "worker-a",
				"retries":            float64(0),
				"kind":               "BPMN_ELEMENT",
				"listenerEventType":  "COMPLETING",
			}, 25)
			return &camundav89.SearchJobsResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav89.JobSearchQueryResult{
					Items: []camundav89.JobSearchResult{{
						JobKey:             "2251799813711967",
						State:              camundav89.JobStateEnumFAILED,
						Retries:            retries,
						Type:               "payment-worker",
						Worker:             "worker-a",
						Kind:               camundav89.BPMNELEMENT,
						ListenerEventType:  camundav89.JobListenerEventTypeEnumCOMPLETING,
						ProcessInstanceKey: "2251799813711000",
						ElementInstanceKey: "2251799813711001",
						ElementId:          &elementID,
						TenantId:           "tenant-a",
					}},
				},
			}, nil
		},
	})

	result, err := svc.SearchJobs(context.Background(), d.JobSearchQuery{
		State:              "FAILED",
		Type:               "payment-worker",
		ProcessInstanceKey: "2251799813711000",
		ElementInstanceKey: "2251799813711001",
		ElementId:          "charge-card",
		Worker:             "worker-a",
		Retries:            &retries,
		Kind:               "BPMN_ELEMENT",
		ListenerEventType:  "COMPLETING",
		Limit:              25,
	})

	require.NoError(t, err)
	require.Equal(t, int32(25), result.Limit)
	require.Len(t, result.Items, 1)
	require.Equal(t, "2251799813711967", result.Items[0].Key)
	require.Equal(t, "payment-worker", result.Items[0].Type)
	require.Equal(t, "worker-a", result.Items[0].Worker)
	require.Equal(t, "BPMN_ELEMENT", result.Items[0].Kind)
	require.Equal(t, "COMPLETING", result.Items[0].ListenerEventType)
	require.Equal(t, "charge-card", result.Items[0].ElementId)
}

func TestJobUpdateRetriesRequest(t *testing.T) {
	retries := int32(3)
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(context.Context, camundav89.SearchJobsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			t.Fatal("unexpected retry confirmation lookup")
			return nil, nil
		},
		updateJobWithResponse: func(_ context.Context, jobKey camundav89.JobKey, body camundav89.UpdateJobJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error) {
			require.Equal(t, camundav89.JobKey("2251799813711967"), jobKey)
			require.NotNil(t, body.Changeset.Retries)
			require.Equal(t, retries, *body.Changeset.Retries)
			require.Nil(t, body.Changeset.Timeout)
			return &camundav89.UpdateJobResponse{
				HTTPResponse: okJobUpdateHTTPResponse(),
			}, nil
		},
	})

	result, err := svc.UpdateJob(context.Background(), d.JobUpdateRequest{
		Key:              "2251799813711967",
		Retries:          &retries,
		SkipConfirmation: true,
	})

	require.NoError(t, err)
	require.True(t, result.MutationAccepted)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Equal(t, &retries, result.SubmittedRetries)
}

func TestUpdateJobTimeoutRequest(t *testing.T) {
	timeoutMillis := int64(300000)
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(context.Context, camundav89.SearchJobsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			t.Fatal("unexpected timeout confirmation lookup")
			return nil, nil
		},
		updateJobWithResponse: func(_ context.Context, jobKey camundav89.JobKey, body camundav89.UpdateJobJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error) {
			require.Equal(t, camundav89.JobKey("2251799813711967"), jobKey)
			require.Nil(t, body.Changeset.Retries)
			require.NotNil(t, body.Changeset.Timeout)
			require.Equal(t, timeoutMillis, *body.Changeset.Timeout)
			return &camundav89.UpdateJobResponse{
				HTTPResponse: okJobUpdateHTTPResponse(),
			}, nil
		},
	})

	result, err := svc.UpdateJob(context.Background(), d.JobUpdateRequest{
		Key:           "2251799813711967",
		TimeoutMillis: &timeoutMillis,
	})

	require.NoError(t, err)
	require.True(t, result.MutationAccepted)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Equal(t, &timeoutMillis, result.SubmittedTimeoutMS)
	require.Nil(t, result.ConfirmedRetries)
}

func TestUpdateJobRetriesAndTimeoutRequest(t *testing.T) {
	retries := int32(3)
	timeoutMillis := int64(300000)
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(context.Context, camundav89.SearchJobsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			t.Fatal("unexpected combined update confirmation lookup")
			return nil, nil
		},
		updateJobWithResponse: func(_ context.Context, jobKey camundav89.JobKey, body camundav89.UpdateJobJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error) {
			require.Equal(t, camundav89.JobKey("2251799813711967"), jobKey)
			require.NotNil(t, body.Changeset.Retries)
			require.Equal(t, retries, *body.Changeset.Retries)
			require.NotNil(t, body.Changeset.Timeout)
			require.Equal(t, timeoutMillis, *body.Changeset.Timeout)
			return &camundav89.UpdateJobResponse{
				HTTPResponse: okJobUpdateHTTPResponse(),
			}, nil
		},
	})

	result, err := svc.UpdateJob(context.Background(), d.JobUpdateRequest{
		Key:              "2251799813711967",
		Retries:          &retries,
		TimeoutMillis:    &timeoutMillis,
		SkipConfirmation: true,
	})

	require.NoError(t, err)
	require.True(t, result.MutationAccepted)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Equal(t, &retries, result.SubmittedRetries)
	require.Equal(t, &timeoutMillis, result.SubmittedTimeoutMS)
}

func TestSubmitJobTechnicalFailureRequest(t *testing.T) {
	retries := int32(0)
	retryBackoffMillis := int64(300000)
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(context.Context, camundav89.SearchJobsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			t.Fatal("unexpected worker outcome confirmation lookup")
			return nil, nil
		},
		failJobWithResponse: func(_ context.Context, jobKey camundav89.JobKey, body camundav89.FailJobJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.FailJobResponse, error) {
			require.Equal(t, camundav89.JobKey("2251799813711967"), jobKey)
			require.Equal(t, &retries, body.Retries)
			require.Equal(t, &retryBackoffMillis, body.RetryBackOff)
			require.NotNil(t, body.ErrorMessage)
			require.Equal(t, "worker unavailable", *body.ErrorMessage)
			require.Nil(t, body.Variables)
			return &camundav89.FailJobResponse{
				HTTPResponse: okJobFailHTTPResponse(),
			}, nil
		},
	})

	result, err := svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{
		Key:                "2251799813711967",
		Mode:               d.JobWorkerOutcomeTechnicalFailure,
		Retries:            &retries,
		RetryBackoffMillis: &retryBackoffMillis,
		Message:            "worker unavailable",
	})

	require.NoError(t, err)
	require.True(t, result.MutationAccepted)
	require.Equal(t, d.JobWorkerOutcomeTechnicalFailure, result.Mode)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Equal(t, &retries, result.SubmittedRetries)
	require.Equal(t, &retryBackoffMillis, result.SubmittedBackoffMS)
}

func TestSubmitJobBPMNErrorRequest(t *testing.T) {
	svc := newJobServiceTest(t, &mockJobClient{
		searchJobsWithResponse: func(context.Context, camundav89.SearchJobsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error) {
			t.Fatal("unexpected worker outcome confirmation lookup")
			return nil, nil
		},
		throwJobErrorWithResponse: func(_ context.Context, jobKey camundav89.JobKey, body camundav89.ThrowJobErrorJSONRequestBody, _ ...camundav89.RequestEditorFn) (*camundav89.ThrowJobErrorResponse, error) {
			require.Equal(t, camundav89.JobKey("2251799813711967"), jobKey)
			require.Equal(t, "PAYMENT_DECLINED", body.ErrorCode)
			require.NotNil(t, body.ErrorMessage)
			require.Equal(t, "card declined", *body.ErrorMessage)
			require.NotNil(t, body.Variables)
			require.Equal(t, map[string]any{"approved": false}, *body.Variables)
			return &camundav89.ThrowJobErrorResponse{
				HTTPResponse: okJobBPMNErrorHTTPResponse(),
			}, nil
		},
	})

	result, err := svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{
		Key:       "2251799813711967",
		Mode:      d.JobWorkerOutcomeBPMNError,
		ErrorCode: "PAYMENT_DECLINED",
		Message:   "card declined",
		Variables: map[string]any{"approved": false},
	})

	require.NoError(t, err)
	require.True(t, result.MutationAccepted)
	require.Equal(t, d.JobWorkerOutcomeBPMNError, result.Mode)
	require.Equal(t, "skipped", result.ConfirmationStatus)
	require.Equal(t, "PAYMENT_DECLINED", result.SubmittedErrorCode)
}

func newJobServiceTest(t *testing.T, client *mockJobClient) *Service {
	t.Helper()
	cfg := testx.TestConfig(t)
	cfg.App.CamundaVersion = toolx.V89
	svc, err := New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client))
	require.NoError(t, err)
	return svc
}

func requireJobSearchBody(t *testing.T, body camundav89.SearchJobsJSONRequestBody, key string) {
	t.Helper()
	require.NotNil(t, body.Filter)
	require.NotNil(t, body.Filter.JobKey)
	gotKey, err := body.Filter.JobKey.AsJobKeyFilterProperty0()
	require.NoError(t, err)
	require.Equal(t, camundav89.JobKey(key), gotKey)
	require.NotNil(t, body.Page)
}

// requireJobSearchFilterJSON asserts generated union filters serialize to the
// simple equality values expected by the Camunda search endpoint.
func requireJobSearchFilterJSON(t *testing.T, body camundav89.SearchJobsJSONRequestBody, want map[string]any, wantLimit int32) {
	t.Helper()
	raw, err := json.Marshal(body)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got))
	filter := got["filter"].(map[string]any)
	for name, value := range want {
		require.Equal(t, value, filter[name], "filter %s", name)
	}
	page := got["page"].(map[string]any)
	require.Equal(t, float64(wantLimit), page["limit"])
}

func okHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Request: &http.Request{
			Method: http.MethodPost,
			URL:    &url.URL{Scheme: "https", Host: "camunda.example", Path: "/v2/jobs/search"},
		},
	}
}

func okJobUpdateHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
		Request: &http.Request{
			Method: http.MethodPatch,
			URL:    &url.URL{Scheme: "https", Host: "camunda.example", Path: "/v2/jobs/2251799813711967"},
		},
	}
}

func okJobFailHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
		Request: &http.Request{
			Method: http.MethodPost,
			URL:    &url.URL{Scheme: "https", Host: "camunda.example", Path: "/v2/jobs/2251799813711967/failure"},
		},
	}
}

func okJobBPMNErrorHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusNoContent,
		Status:     "204 No Content",
		Request: &http.Request{
			Method: http.MethodPost,
			URL:    &url.URL{Scheme: "https", Host: "camunda.example", Path: "/v2/jobs/2251799813711967/error"},
		},
	}
}

func stringPtr(value string) *string {
	return &value
}
