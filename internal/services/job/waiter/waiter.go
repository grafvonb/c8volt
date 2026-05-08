// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx/logging"
)

type JobLookup interface {
	LookupJob(ctx context.Context, key string, opts ...services.CallOption) (d.Job, error)
}

func WaitForRetries(ctx context.Context, s JobLookup, cfg *config.Config, log *slog.Logger, key string, retries int32, opts ...services.CallOption) (d.Job, error) {
	cCfg := services.ApplyCallOptions(opts)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for job %s retries to be %d", key, retries))
	defer stopActivity()

	backoff := cfg.App.Backoff
	start := time.Now()
	if backoff.Timeout > 0 {
		deadline := time.Now().Add(backoff.Timeout)
		if dl, ok := ctx.Deadline(); !ok || deadline.Before(dl) {
			var cancel context.CancelFunc
			ctx, cancel = context.WithDeadline(ctx, deadline)
			defer cancel()
		}
	}

	delay := backoff.InitialDelay
	if delay <= 0 {
		delay = 500 * time.Millisecond
	}
	attempts := 0
	for {
		if err := ctx.Err(); err != nil {
			return d.Job{}, err
		}
		attempts++
		job, err := s.LookupJob(ctx, key, opts...)
		if err != nil {
			return d.Job{}, err
		}
		if job.Key != "" && job.Retries == retries {
			return job, nil
		}
		logging.InfoIfVerbose(fmt.Sprintf("job %s retries currently %d; waiting for %d... (attempt #%d)", key, job.Retries, retries, attempts), log, cCfg.Verbose)
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			elapsed := time.Since(start)
			return job, fmt.Errorf("exceeded max_retries (%d) waiting for job %s retries to be %d after %d attempts in %s", backoff.MaxRetries, key, retries, attempts, elapsed)
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			return job, ctx.Err()
		}
	}
}
