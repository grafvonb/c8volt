// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"testing"
	"time"

	camundav88 "github.com/grafvonb/c8volt/internal/clients/camunda/v88/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

type mockJobClient struct {
	searchJobsWithResponse func(context.Context, camundav88.SearchJobsJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchJobsResponse, error)
	updateJobWithResponse  func(context.Context, camundav88.JobKey, camundav88.UpdateJobJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.UpdateJobResponse, error)
}

func (m *mockJobClient) SearchJobsWithResponse(ctx context.Context, body camundav88.SearchJobsJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.SearchJobsResponse, error) {
	return m.searchJobsWithResponse(ctx, body, reqEditors...)
}

func (m *mockJobClient) UpdateJobWithResponse(ctx context.Context, jobKey camundav88.JobKey, body camundav88.UpdateJobJSONRequestBody, reqEditors ...camundav88.RequestEditorFn) (*camundav88.UpdateJobResponse, error) {
	if m.updateJobWithResponse == nil {
		panic("unexpected UpdateJobWithResponse call")
	}
	return m.updateJobWithResponse(ctx, jobKey, body, reqEditors...)
}

func TestSearchJobsByKey(t *testing.T) {
	deadline := time.Date(2026, 5, 8, 10, 15, 0, 0, time.UTC)
	svc := newJobLookupTestService(t, &mockJobClient{
		searchJobsWithResponse: func(_ context.Context, body camundav88.SearchJobsJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchJobsResponse, error) {
			requireJobSearchBody(t, body, "2251799813711967")
			return &camundav88.SearchJobsResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav88.JobSearchQueryResult{
					Items: []camundav88.JobSearchResult{{
						JobKey:             "2251799813711967",
						State:              camundav88.JobStateEnum("FAILED"),
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

	job, err := svc.LookupJob(context.Background(), "2251799813711967")

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

func TestJobLookupService_NotFound(t *testing.T) {
	svc := newJobLookupTestService(t, &mockJobClient{
		searchJobsWithResponse: func(_ context.Context, body camundav88.SearchJobsJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.SearchJobsResponse, error) {
			requireJobSearchBody(t, body, "missing-job")
			return &camundav88.SearchJobsResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200:      &camundav88.JobSearchQueryResult{},
			}, nil
		},
	})

	job, err := svc.LookupJob(context.Background(), "missing-job")

	require.NoError(t, err)
	require.Empty(t, job)
}

func TestJobUpdateRetriesRequest(t *testing.T) {
	retries := int32(3)
	svc := newJobLookupTestService(t, &mockJobClient{
		searchJobsWithResponse: func(context.Context, camundav88.SearchJobsJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchJobsResponse, error) {
			t.Fatal("unexpected retry confirmation lookup")
			return nil, nil
		},
		updateJobWithResponse: func(_ context.Context, jobKey camundav88.JobKey, body camundav88.UpdateJobJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.UpdateJobResponse, error) {
			require.Equal(t, camundav88.JobKey("2251799813711967"), jobKey)
			require.NotNil(t, body.Changeset.Retries)
			require.Equal(t, retries, *body.Changeset.Retries)
			require.Nil(t, body.Changeset.Timeout)
			return &camundav88.UpdateJobResponse{
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
	svc := newJobLookupTestService(t, &mockJobClient{
		searchJobsWithResponse: func(context.Context, camundav88.SearchJobsJSONRequestBody, ...camundav88.RequestEditorFn) (*camundav88.SearchJobsResponse, error) {
			t.Fatal("unexpected timeout confirmation lookup")
			return nil, nil
		},
		updateJobWithResponse: func(_ context.Context, jobKey camundav88.JobKey, body camundav88.UpdateJobJSONRequestBody, _ ...camundav88.RequestEditorFn) (*camundav88.UpdateJobResponse, error) {
			require.Equal(t, camundav88.JobKey("2251799813711967"), jobKey)
			require.Nil(t, body.Changeset.Retries)
			require.NotNil(t, body.Changeset.Timeout)
			require.Equal(t, timeoutMillis, *body.Changeset.Timeout)
			return &camundav88.UpdateJobResponse{
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

func newJobLookupTestService(t *testing.T, client *mockJobClient) *Service {
	t.Helper()
	cfg := testx.TestConfig(t)
	cfg.App.CamundaVersion = toolx.V88
	svc, err := New(cfg, &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client))
	require.NoError(t, err)
	return svc
}

func requireJobSearchBody(t *testing.T, body camundav88.SearchJobsJSONRequestBody, key string) {
	t.Helper()
	require.NotNil(t, body.Filter)
	require.NotNil(t, body.Filter.JobKey)
	gotKey, err := body.Filter.JobKey.AsJobKeyFilterProperty0()
	require.NoError(t, err)
	require.Equal(t, camundav88.JobKey(key), gotKey)
	require.NotNil(t, body.Page)
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

func stringPtr(value string) *string {
	return &value
}
