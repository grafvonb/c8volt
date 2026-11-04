// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
)

// SuspensionReader restricts confirmation to direct reads of the selected key.
type SuspensionReader interface {
	GetProcessInstance(context.Context, string, ...services.CallOption) (d.ProcessInstance, error)
}

// SuspensionObservation retains the last valid observation independently of submission.
type SuspensionObservation struct {
	HasObservation    bool
	LastObservedState d.State
	SuspendedDate     string
}

// WaitForProcessInstanceSuspension observes without ever submitting a mutation.
func WaitForProcessInstanceSuspension(ctx context.Context, reader SuspensionReader, backoff config.BackoffConfig, key string, opts ...services.CallOption) (SuspensionObservation, error) {
	var observation SuspensionObservation
	if err := backoff.Validate(); err != nil {
		return observation, fmt.Errorf("%w: suspension backoff: %w", d.ErrValidation, err)
	}
	ctx, cancel := context.WithTimeout(httpc.WithoutReadRetryLogs(ctx), backoff.Timeout)
	defer cancel()
	nested := append([]services.CallOption(nil), opts...)
	nested = append(nested, services.WithSuppressNestedProcessInstanceLookupLogs(), services.WithSuppressProcessInstanceDetailLogs())
	delay := backoff.InitialDelay
	for attempts := 1; ; attempts++ {
		if err := ctx.Err(); err != nil {
			return observation, err
		}
		pi, err := reader.GetProcessInstance(ctx, key, nested...)
		// A read that returns after the deadline cannot confirm within this bound.
		if ctx.Err() != nil {
			return observation, errors.Join(ctx.Err(), err)
		}
		switch {
		case errors.Is(err, d.ErrUnauthorized), errors.Is(err, d.ErrForbidden), errors.Is(err, d.ErrPrecondition):
			return observation, err
		case errors.Is(err, d.ErrNotFound):
			observation = SuspensionObservation{HasObservation: true, LastObservedState: d.StateAbsent}
		case err != nil:
			return observation, err
		default:
			if pi.Key != key {
				return observation, fmt.Errorf("%w: suspension observation identity mismatch", d.ErrMalformedResponse)
			}
			state := pi.State
			switch state {
			case d.StateActive, d.StateSuspended, d.StateCompleted, d.StateCanceled, d.StateTerminated, d.StateUnknown:
			default:
				state = d.StateUnknown
			}
			observation = SuspensionObservation{HasObservation: true, LastObservedState: state, SuspendedDate: pi.SuspendedDate}
			switch state {
			case d.StateSuspended:
				return observation, nil
			case d.StateCompleted, d.StateCanceled, d.StateTerminated:
				return observation, fmt.Errorf("%w: process instance reached %s before suspension confirmation", d.ErrConflict, state)
			}
		}
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			return observation, fmt.Errorf("%w: suspension confirmation attempt limit reached", d.ErrGatewayTimeout)
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return observation, ctx.Err()
		case <-timer.C:
		}
		delay = backoff.NextDelay(delay)
	}
}
