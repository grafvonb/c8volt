// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package poller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/grafvonb/c8volt/toolx/logging"
)

const (
	DefaultCompletionTimeout = 10 * time.Minute

	defaultInitialDelay   = 2 * time.Second
	defaultMaxDelay       = 30 * time.Second
	defaultBackoffFactor  = 1.6
	defaultJitterMinRatio = 0.8 //  -20%
	defaultJitterMaxRatio = 1.2 // +20%
)

type JobPollStatus struct {
	Success bool
	Message string
}

type JobPollFunc func(ctx context.Context) (JobPollStatus, error)

func WaitForCompletion(ctx context.Context, log *slog.Logger, timeout time.Duration, noProgress bool, poll JobPollFunc) error {
	if poll == nil {
		return errors.New("poll function is required")
	}
	startedAt := time.Now()
	log.Debug(fmt.Sprintf("waiting for completion; timeout %s", timeout.String()))
	var activity logging.ActivitySink
	if !noProgress {
		activity = logging.ActivityFromContext(ctx)
		if activity != nil {
			activity.StartActivity("waiting for completion")
			defer activity.StopActivity()
		}
	}

	deadline := time.Now().Add(timeout)
	delay := defaultInitialDelay
	attempt := 0
	for {
		duration := time.Since(startedAt)
		attempt++
		if !noProgress && activity == nil {
			fmt.Fprint(os.Stderr, ".")
		}
		log.Debug(fmt.Sprintf("waiting for completion; attempt %d, started %s, elapsed %s", attempt, startedAt.String(), duration.String()))

		if err := ctx.Err(); err != nil {
			log.Debug(fmt.Sprintf("waiting for completion stopped; reason context, error %s", err))
			return err
		}
		if time.Now().After(deadline) {
			log.Debug(fmt.Sprintf("waiting for completion stopped; reason timeout, attempts %d", attempt))
			return errors.New("timeout while waiting for completion for backend jobs to finish")
		}

		status, err := poll(ctx)
		if err != nil {
			return err
		}

		if status.Success {
			duration = time.Since(startedAt)
			if !noProgress && activity == nil {
				fmt.Fprintf(os.Stderr, "completed after: %s\n", duration.String())
			}
			log.Debug(fmt.Sprintf("waiting for completion done; attempts %d, elapsed %s", attempt, duration.String()))
			return nil
		}
		if activity != nil {
			msg := fmt.Sprintf("waiting for completion; attempt %d, elapsed %s", attempt, time.Since(startedAt).Round(time.Second))
			if status.Message != "" {
				msg += fmt.Sprintf(", status %s", status.Message)
			}
			logging.UpdateActivity(ctx, msg)
		}
		log.Debug(fmt.Sprintf("waiting for completion; reason %s", status.Message))

		jitterRange := defaultJitterMaxRatio - defaultJitterMinRatio
		jitterFactor := defaultJitterMinRatio + rand.Float64()*jitterRange
		sleep := time.Duration(float64(delay) * jitterFactor)

		select {
		case <-ctx.Done():
			log.Debug("waiting for completion stopped; reason context during sleep")
			return ctx.Err()
		case <-time.After(sleep):
		}

		next := time.Duration(float64(delay) * defaultBackoffFactor)
		if next > defaultMaxDelay {
			next = defaultMaxDelay
		}
		delay = next
	}
}
