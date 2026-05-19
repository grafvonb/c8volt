// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

type fakeIncidentWaiter struct {
	incidents []d.ProcessInstanceIncidentDetail
}

func (f *fakeIncidentWaiter) GetIncident(context.Context, string, ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	if len(f.incidents) == 0 {
		return d.ProcessInstanceIncidentDetail{}, nil
	}
	incident := f.incidents[0]
	f.incidents = f.incidents[1:]
	return incident, nil
}

func (f *fakeIncidentWaiter) SearchProcessInstanceIncidents(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	return nil, nil
}

func TestWaitForIncidentResolvedUpdatesActivity(t *testing.T) {
	sink := &activitysink.Sink{}
	waiter := &fakeIncidentWaiter{incidents: []d.ProcessInstanceIncidentDetail{
		{IncidentKey: "incident-1", State: "ACTIVE"},
		{IncidentKey: "incident-1", State: "RESOLVED"},
	}}

	got, err := WaitForIncidentResolved(logging.ToActivityContext(context.Background(), sink), waiter, incidentWaiterTestConfig(), incidentWaiterTestLogger(), "incident-1")

	require.NoError(t, err)
	require.True(t, got.Ok)
	require.Equal(t, []string{"incident incident-1 waiting; state ACTIVE, attempt 1"}, sink.Updates())
}

func incidentWaiterTestConfig() *config.Config {
	cfg := config.New()
	cfg.App.Backoff = config.BackoffConfig{
		InitialDelay: time.Nanosecond,
		MaxRetries:   2,
		Timeout:      time.Second,
	}
	return cfg
}

func incidentWaiterTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
