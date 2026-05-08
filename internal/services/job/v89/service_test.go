// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
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
	searchJobsWithResponse func(context.Context, camundav89.SearchJobsJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchJobsResponse, error)
	updateJobWithResponse  func(context.Context, camundav89.JobKey, camundav89.UpdateJobJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.UpdateJobResponse, error)
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

func TestSearchJobsByKey(t *testing.T) {
	deadline := time.Date(2026, 5, 8, 10, 15, 0, 0, time.UTC)
	svc := newJobLookupTestService(t, &mockJobClient{
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

func newJobLookupTestService(t *testing.T, client *mockJobClient) *Service {
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

func okHTTPResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Request: &http.Request{
			Method: http.MethodPost,
			URL:    &url.URL{Scheme: "https", Host: "camunda.example", Path: "/v2/jobs/search"},
		},
	}
}

func stringPtr(value string) *string {
	return &value
}
