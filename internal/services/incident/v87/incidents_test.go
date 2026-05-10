// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	v87 "github.com/grafvonb/c8volt/internal/services/incident/v87"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

func TestUnsupportedIncidentResolutionOperations(t *testing.T) {
	t.Parallel()

	svc := newTestService(t)

	_, err := svc.GetIncident(context.Background(), "2251799813685249")
	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))
	require.Contains(t, err.Error(), "direct incident lookup is not tenant-safe in Camunda 8.7")

	resp, err := svc.ResolveIncident(context.Background(), "2251799813685249")
	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))
	require.False(t, resp.Ok)

	_, err = svc.SearchIncidents(context.Background(), d.IncidentFilter{}, 100)
	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))
	require.Contains(t, err.Error(), "incident search is not tenant-safe in Camunda 8.7")

	_, err = svc.SearchIncidentsPage(context.Background(), d.IncidentFilter{}, d.IncidentPageRequest{Size: 100})
	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))
	require.Contains(t, err.Error(), "incident search is not tenant-safe in Camunda 8.7")

	_, err = svc.WaitForIncidentResolved(context.Background(), "2251799813685249")
	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))

	_, err = svc.WaitForProcessInstanceIncidentsResolved(context.Background(), "2251799813685250", []string{"2251799813685249"})
	require.Error(t, err)
	require.True(t, errors.Is(err, d.ErrUnsupported))
}

func newTestService(t *testing.T) *v87.Service {
	t.Helper()
	cfg := &config.Config{
		App:  config.App{CamundaVersion: toolx.V87},
		APIs: config.APIs{Camunda: config.API{BaseURL: "http://localhost:8080/v2"}},
	}
	svc, err := v87.New(cfg, &http.Client{}, slog.Default())
	require.NoError(t, err)
	return svc
}
