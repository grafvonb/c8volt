// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/grafvonb/c8volt/toolx/pool"
	"github.com/grafvonb/c8volt/typex"
)

type PIWaiter interface {
	GetProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error)
	GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, d.ProcessInstance, error)
}

func WaitForProcessInstancesState(ctx context.Context, s PIWaiter, cfg *config.Config, log *slog.Logger, keys typex.Keys, desired d.States, wantedWorkers int, opts ...services.CallOption) (d.StateResponses, error) {
	cCfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for %d process instance(s) to reach desired state(s)", lk))
	defer stopActivity()

	rs, err := pool.ExecuteSlice[string, d.StateResponse](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (d.StateResponse, error) {
		sr, _, werr := WaitForProcessInstanceState(ctx, s, cfg, log, key, desired, opts...)
		return sr, werr
	})
	r := d.StateResponses{
		Items: rs,
	}
	return r, err
}

func WaitForProcessInstancesExpectation(ctx context.Context, s PIWaiter, cfg *config.Config, log *slog.Logger, keys typex.Keys, request d.ProcessInstanceExpectationRequest, wantedWorkers int, opts ...services.CallOption) (d.ProcessInstanceExpectationResponses, error) {
	cCfg := services.ApplyCallOptions(opts)
	ukeys := keys.Unique()
	lk := len(ukeys)
	nw := toolx.DetermineNoOfWorkers(lk, wantedWorkers, cCfg.NoWorkerLimit)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for %d process instance(s) to satisfy expectation(s)", lk))
	defer stopActivity()

	rs, err := pool.ExecuteSlice[string, d.ProcessInstanceExpectationResponse](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (d.ProcessInstanceExpectationResponse, error) {
		resp, _, werr := WaitForProcessInstanceExpectation(ctx, s, cfg, log, key, request, opts...)
		return resp, werr
	})
	return d.ProcessInstanceExpectationResponses{Items: rs}, err
}

func WaitForProcessInstanceExpectation(ctx context.Context, s PIWaiter, cfg *config.Config, log *slog.Logger, key string, request d.ProcessInstanceExpectationRequest, opts ...services.CallOption) (d.ProcessInstanceExpectationResponse, d.ProcessInstance, error) {
	if request.Incident == nil {
		sr, pi, err := WaitForProcessInstanceState(ctx, s, cfg, log, key, request.States, opts...)
		return d.ProcessInstanceExpectationResponse{
			Key:    key,
			Ok:     sr.Ok,
			State:  sr.State,
			Status: sr.Status,
		}, pi, err
	}
	status := fmt.Sprintf("process instance %s incident expectation waits are not implemented yet", key)
	return d.ProcessInstanceExpectationResponse{Key: key, Ok: false, Status: status}, d.ProcessInstance{}, fmt.Errorf("%w: %s", d.ErrUnsupported, status)
}

func WaitForProcessInstanceState(ctx context.Context, s PIWaiter, cfg *config.Config, log *slog.Logger, key string, desired d.States, opts ...services.CallOption) (d.StateResponse, d.ProcessInstance, error) {
	cCfg := services.ApplyCallOptions(opts)
	stopActivity := logging.StartActivity(ctx, fmt.Sprintf("waiting for process instance %s to reach desired state(s)", key))
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

	attempts := 0
	delay := backoff.InitialDelay
	for {
		if errCtx := ctx.Err(); errCtx != nil {
			elapsed := time.Since(start)
			status := fmt.Sprintf("stopped waiting for process instance %s after %d attempts in %s due to context error", key, attempts, elapsed)
			log.Debug(status)
			return d.StateResponse{Ok: false, State: d.StateUnknown, Status: status}, d.ProcessInstance{}, fmt.Errorf("%w: %s", errCtx, status)
		}
		attempts++
		log.Debug(fmt.Sprintf("attempt #%d to fetch state for process instance %s", attempts, key))
		got, pi, errInDelay := s.GetProcessInstanceStateByKey(ctx, key, opts...)
		if errInDelay == nil {
			if stateIn(got, desired) {
				if attempts == 1 {
					status := fmt.Sprintf("process instance %s is already in one of the desired state(s) [%s] (current: %s)", key, desired, got)
					log.Debug(status)
					return d.StateResponse{Ok: true, State: got, Status: status}, pi, nil
				}
				elapsed := time.Since(start)
				status := fmt.Sprintf("process instance %s reached one of the desired state(s) [%s] (current: %s) after %d checks in %s", key, desired, got, attempts, elapsed)
				log.Debug(status)
				return d.StateResponse{Ok: true, State: got, Status: status}, pi, nil
			}
			logging.InfoIfVerbose(fmt.Sprintf("process instance %s currently in state %s; waiting... (attempt #%d)", key, got, attempts), log, cCfg.Verbose)
		} else if errInDelay != nil {
			if isProcessInstanceAbsentErr(errInDelay) {
				// Only waiter-driven absent/deleted confirmation maps not-found into ABSENT; direct lookups stay strict.
				got = d.StateAbsent
				if stateIn(got, desired) {
					elapsed := time.Since(start)
					status := fmt.Sprintf("process instance %s reached one of the desired state(s) [%s] (current: %s) after %d checks in %s", key, desired, got, attempts, elapsed)
					log.Debug(status)
					return d.StateResponse{Ok: true, State: got, Status: status}, d.ProcessInstance{}, nil
				}
				logging.InfoIfVerbose(fmt.Sprintf("process instance %s is absent (not found); waiting... (attempt #%d)", key, attempts), log, cCfg.Verbose)
			} else {
				elapsed := time.Since(start)
				status := fmt.Sprintf("stopped waiting for process instance %s after %d attempts in %s due to error", key, attempts, elapsed)
				log.Error(status)
				return d.StateResponse{Ok: false, State: got, Status: status}, d.ProcessInstance{}, fmt.Errorf("%w: %s", errInDelay, status)
			}
		}
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			elapsed := time.Since(start)
			status := fmt.Sprintf("exceeded max_retries (%d) waiting for state %q of process instance %s after %d attempts in %s", backoff.MaxRetries, desired, key, attempts, elapsed)
			log.Debug(status)
			return d.StateResponse{Ok: false, State: d.StateUnknown, Status: status}, d.ProcessInstance{}, errors.New(status)
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			elapsed := time.Since(start)
			status := fmt.Sprintf("stopped waiting for process instance %s after %d attempts in %s due to context done", key, attempts, elapsed)
			log.Debug(status)
			return d.StateResponse{Ok: false, State: d.StateUnknown, Status: status}, d.ProcessInstance{}, fmt.Errorf("%w: %s", ctx.Err(), status)
		}
	}
}

func isProcessInstanceAbsentErr(err error) bool {
	if errors.Is(err, d.ErrNotFound) {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "404") ||
		strings.Contains(msg, "not found") ||
		strings.Contains(msg, "does not exist")
}

func stateIn(st d.State, set d.States) bool {
	for _, x := range set {
		if statesEquivalent(st, x) {
			return true
		}
	}
	return false
}

func stateExpectationMatches(state d.State, present bool, desired d.States) bool {
	if !present {
		state = d.StateAbsent
	}
	return len(desired) == 0 || stateIn(state, desired)
}

func incidentExpectationMatches(pi d.ProcessInstance, present bool, desired *bool) bool {
	if desired == nil {
		return true
	}
	return present && pi.Incident == *desired
}

func processInstanceExpectationMatches(pi d.ProcessInstance, present bool, request d.ProcessInstanceExpectationRequest) bool {
	if !stateExpectationMatches(pi.State, present, request.States) {
		return false
	}
	return incidentExpectationMatches(pi, present, request.Incident)
}

func statesEquivalent(left, right d.State) bool {
	if left.EqualsIgnoreCase(right) {
		return true
	}
	return isCanceledLike(left) && isCanceledLike(right)
}

func isCanceledLike(state d.State) bool {
	return state.In(d.StateCanceled, d.StateTerminated)
}
