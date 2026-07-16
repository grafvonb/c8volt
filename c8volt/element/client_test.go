// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package element

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/stretchr/testify/require"
)

type fakeElementService struct {
	get    func(context.Context, string, ...services.CallOption) (d.Element, error)
	search func(context.Context, d.ElementSearchQuery, ...services.CallOption) (d.ElementSearchResult, error)
	page   func(context.Context, d.ElementSearchQuery, d.ElementPageRequest, ...services.CallOption) (d.ElementSearchPage, error)
}

func (f fakeElementService) GetElement(ctx context.Context, key string, opts ...services.CallOption) (d.Element, error) {
	return f.get(ctx, key, opts...)
}

func (f fakeElementService) SearchElements(ctx context.Context, request d.ElementSearchQuery, opts ...services.CallOption) (d.ElementSearchResult, error) {
	if f.search == nil {
		return d.ElementSearchResult{}, errors.New("unexpected search")
	}
	return f.search(ctx, request, opts...)
}

func (f fakeElementService) SearchElementsPage(ctx context.Context, request d.ElementSearchQuery, page d.ElementPageRequest, opts ...services.CallOption) (d.ElementSearchPage, error) {
	if f.page == nil {
		return d.ElementSearchPage{}, errors.New("unexpected search page")
	}
	return f.page(ctx, request, page, opts...)
}

func TestClient_GetElement_Found(t *testing.T) {
	api := New(fakeElementService{
		get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
			require.Equal(t, "2251799813689002", key)
			return d.Element{
				ElementInstanceKey:     key,
				ElementId:              "ship-order",
				ElementName:            "Ship order",
				Type:                   "SERVICE_TASK",
				State:                  "ACTIVE",
				StartDate:              "2026-07-15T10:12:01Z",
				EndDate:                "2026-07-15T10:13:02Z",
				ProcessInstanceKey:     "2251799813688001",
				RootProcessInstanceKey: "2251799813688001",
				ProcessDefinitionId:    "order-process",
				ProcessDefinitionKey:   "2251799813687001",
				TenantId:               "tenant-a",
				HasIncident:            true,
				IncidentKey:            "2251799813687777",
			}, nil
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.GetElement(context.Background(), "2251799813689002")

	require.NoError(t, err)
	require.Equal(t, Element{
		ElementInstanceKey:     "2251799813689002",
		ElementId:              "ship-order",
		ElementName:            "Ship order",
		Type:                   "SERVICE_TASK",
		State:                  "ACTIVE",
		StartDate:              "2026-07-15T10:12:01Z",
		EndDate:                "2026-07-15T10:13:02Z",
		ProcessInstanceKey:     "2251799813688001",
		RootProcessInstanceKey: "2251799813688001",
		ProcessDefinitionId:    "order-process",
		ProcessDefinitionKey:   "2251799813687001",
		TenantId:               "tenant-a",
		HasIncident:            true,
		IncidentKey:            "2251799813687777",
	}, result)
}

func TestClient_GetElement_NotFound(t *testing.T) {
	api := New(fakeElementService{
		get: func(_ context.Context, key string, _ ...services.CallOption) (d.Element, error) {
			require.Equal(t, "2251799813689999", key)
			return d.Element{}, d.ErrNotFound
		},
	}, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result, err := api.GetElement(context.Background(), "2251799813689999")

	require.ErrorIs(t, err, ferrors.ErrNotFound)
	require.Empty(t, result)
}
