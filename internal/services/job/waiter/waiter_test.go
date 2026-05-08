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
	"github.com/stretchr/testify/require"
)

type fakeJobLookup struct {
	jobs []d.Job
}

func (f *fakeJobLookup) LookupJob(context.Context, string, ...services.CallOption) (d.Job, error) {
	if len(f.jobs) == 0 {
		return d.Job{}, nil
	}
	job := f.jobs[0]
	f.jobs = f.jobs[1:]
	return job, nil
}

func TestRetryConfirmationSuccess(t *testing.T) {
	lookup := &fakeJobLookup{jobs: []d.Job{
		{Key: "2251799813711967", Retries: 1},
		{Key: "2251799813711967", Retries: 3},
	}}

	job, err := WaitForRetries(context.Background(), lookup, retryWaiterTestConfig(), testLogger(), "2251799813711967", 3)

	require.NoError(t, err)
	require.Equal(t, int32(3), job.Retries)
}

func TestRetryConfirmationExhaustion(t *testing.T) {
	lookup := &fakeJobLookup{jobs: []d.Job{
		{Key: "2251799813711967", Retries: 1},
		{Key: "2251799813711967", Retries: 2},
	}}

	_, err := WaitForRetries(context.Background(), lookup, retryWaiterTestConfig(), testLogger(), "2251799813711967", 3)

	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded max_retries")
}

func retryWaiterTestConfig() *config.Config {
	cfg := config.New()
	cfg.App.Backoff = config.BackoffConfig{
		InitialDelay: time.Nanosecond,
		MaxRetries:   2,
		Timeout:      time.Second,
	}
	return cfg
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
