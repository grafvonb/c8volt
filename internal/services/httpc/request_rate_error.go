// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"context"
	"errors"
	"sync/atomic"
)

// RequestRateWaitError identifies only the affected attempt as locally unsent;
// earlier attempts and other resources may already have reached the backend.
type RequestRateWaitError struct{ cause error }

// Error describes local admission failure without transport details.
func (e *RequestRateWaitError) Error() string {
	return "request not sent: request-rate wait: " + e.cause.Error()
}

// Unwrap retains cancellation/deadline classification through service and facade conversion.
func (e *RequestRateWaitError) Unwrap() error { return e.cause }

// IsRequestRateWaitError recognizes an unsent attempt through ordinary wrapping.
func IsRequestRateWaitError(err error) bool {
	var local *RequestRateWaitError
	return errors.As(err, &local)
}

type requestRateObservationKey struct{}

// RequestRateObservation retains local failure evidence even when http.Client.Timeout
// replaces the returned transport error. Each retry callback owns a fresh instance.
type RequestRateObservation struct {
	failure atomic.Pointer[RequestRateWaitError]
}

// BeginRequestRateAttempt adds local failure evidence only for positively paced
// shared clients. Disabled pacing returns the original context without allocation.
func BeginRequestRateAttempt(ctx context.Context) (context.Context, *RequestRateObservation) {
	service, err := FromContext(ctx)
	if err != nil || service.requestRate == nil || service.requestRate.limiter == nil {
		return ctx, nil
	}
	observation := &RequestRateObservation{}
	return context.WithValue(ctx, requestRateObservationKey{}, observation), observation
}

// Failure returns a typed error or nil, including for the disabled-rate fast path.
func (o *RequestRateObservation) Failure() error {
	if o == nil {
		return nil
	}
	if failure := o.failure.Load(); failure != nil {
		return failure
	}
	return nil
}

// recordRequestRateFailure records the first unsent cause for this callback only.
func recordRequestRateFailure(ctx context.Context, cause error) error {
	failure := &RequestRateWaitError{cause: cause}
	if observation, ok := ctx.Value(requestRateObservationKey{}).(*RequestRateObservation); ok {
		observation.failure.CompareAndSwap(nil, failure)
	}
	return failure
}
