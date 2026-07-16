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
	"github.com/stretchr/testify/require"
)

type mockElementClient struct {
	getElementInstanceWithResponse     func(context.Context, camundav89.ElementInstanceKey, ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error)
	searchElementInstancesWithResponse func(context.Context, camundav89.SearchElementInstancesJSONRequestBody, ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error)
}

func (m *mockElementClient) GetElementInstanceWithResponse(ctx context.Context, key camundav89.ElementInstanceKey, reqEditors ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error) {
	if m.getElementInstanceWithResponse == nil {
		panic("unexpected GetElementInstanceWithResponse call")
	}
	return m.getElementInstanceWithResponse(ctx, key, reqEditors...)
}

func (m *mockElementClient) SearchElementInstancesWithResponse(ctx context.Context, body camundav89.SearchElementInstancesJSONRequestBody, reqEditors ...camundav89.RequestEditorFn) (*camundav89.SearchElementInstancesResponse, error) {
	if m.searchElementInstancesWithResponse == nil {
		panic("unexpected SearchElementInstancesWithResponse call")
	}
	return m.searchElementInstancesWithResponse(ctx, body, reqEditors...)
}

func TestService_GetElement_MapsPayload(t *testing.T) {
	start := time.Date(2026, 7, 15, 10, 12, 1, 123, time.UTC)
	end := time.Date(2026, 7, 15, 10, 13, 2, 456, time.UTC)
	rootKey := camundav89.ProcessInstanceKey("2251799813688001")
	incidentKey := camundav89.IncidentKey("2251799813687777")
	svc := newElementServiceTest(t, &mockElementClient{
		getElementInstanceWithResponse: func(_ context.Context, key camundav89.ElementInstanceKey, _ ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error) {
			require.Equal(t, camundav89.ElementInstanceKey("2251799813689002"), key)
			return &camundav89.GetElementInstanceResponse{
				HTTPResponse: okHTTPResponse(),
				JSON200: &camundav89.ElementInstanceResult{
					ElementInstanceKey:     key,
					ElementId:              "ship-order",
					ElementName:            "Ship order",
					Type:                   camundav89.ElementInstanceResultType("SERVICE_TASK"),
					State:                  camundav89.ElementInstanceStateEnum("ACTIVE"),
					StartDate:              start,
					EndDate:                &end,
					ProcessInstanceKey:     "2251799813688001",
					RootProcessInstanceKey: &rootKey,
					ProcessDefinitionId:    "order-process",
					ProcessDefinitionKey:   "2251799813687001",
					TenantId:               "tenant-a",
					HasIncident:            true,
					IncidentKey:            &incidentKey,
				},
			}, nil
		},
	})

	element, err := svc.GetElement(context.Background(), "2251799813689002")

	require.NoError(t, err)
	require.Equal(t, d.Element{
		ElementInstanceKey:     "2251799813689002",
		ElementId:              "ship-order",
		ElementName:            "Ship order",
		Type:                   "SERVICE_TASK",
		State:                  "ACTIVE",
		StartDate:              "2026-07-15T10:12:01.000000123Z",
		EndDate:                "2026-07-15T10:13:02.000000456Z",
		ProcessInstanceKey:     "2251799813688001",
		RootProcessInstanceKey: "2251799813688001",
		ProcessDefinitionId:    "order-process",
		ProcessDefinitionKey:   "2251799813687001",
		TenantId:               "tenant-a",
		HasIncident:            true,
		IncidentKey:            "2251799813687777",
	}, element)
}

func TestService_GetElement_NotFound(t *testing.T) {
	svc := newElementServiceTest(t, &mockElementClient{
		getElementInstanceWithResponse: func(_ context.Context, key camundav89.ElementInstanceKey, _ ...camundav89.RequestEditorFn) (*camundav89.GetElementInstanceResponse, error) {
			require.Equal(t, camundav89.ElementInstanceKey("2251799813689999"), key)
			return &camundav89.GetElementInstanceResponse{
				HTTPResponse: responseWithStatus(http.StatusNotFound),
				Body:         []byte(`{"title":"not found"}`),
			}, nil
		},
	})

	element, err := svc.GetElement(context.Background(), "2251799813689999")

	require.ErrorIs(t, err, d.ErrNotFound)
	require.Empty(t, element)
}

func newElementServiceTest(t *testing.T, client GenElementClient) *Service {
	t.Helper()

	svc, err := New(testx.TestConfig(t), &http.Client{}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(client))
	require.NoError(t, err)
	return svc
}

func okHTTPResponse() *http.Response {
	return responseWithStatus(http.StatusOK)
}

func responseWithStatus(status int) *http.Response {
	u, _ := url.Parse("http://camunda.example.test/v2/element-instances/2251799813689002")
	return &http.Response{
		StatusCode: status,
		Status:     http.StatusText(status),
		Request:    &http.Request{Method: http.MethodGet, URL: u},
	}
}
