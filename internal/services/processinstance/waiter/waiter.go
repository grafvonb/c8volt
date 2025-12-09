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

	rs, err := pool.ExecuteSlice[string, d.StateResponse](ctx, ukeys, nw, cCfg.FailFast, func(ctx context.Context, key string, _ int) (d.StateResponse, error) {
		sr, _, werr := WaitForProcessInstanceState(ctx, s, cfg, log, key, desired, opts...)
		return sr, werr
	})
	r := d.StateResponses{
		Items: rs,
	}
	return r, err
}

func WaitForProcessInstanceState(ctx context.Context, s PIWaiter, cfg *config.Config, log *slog.Logger, key string, desired d.States, opts ...services.CallOption) (d.StateResponse, d.ProcessInstance, error) {
	_ = services.ApplyCallOptions(opts)
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
		got, pi, errInDelay := s.GetProcessInstanceStateByKey(ctx, key)
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
			log.Info(fmt.Sprintf("process instance %s currently in state %s; waiting... (attempt #%d)", key, got, attempts))
		} else if errInDelay != nil {
			if strings.Contains(errInDelay.Error(), "404") {
				got = d.StateAbsent
				if stateIn(got, desired) {
					elapsed := time.Since(start)
					status := fmt.Sprintf("process instance %s reached one of the desired state(s) [%s] (current: %s) after %d checks in %s", key, desired, got, attempts, elapsed)
					log.Debug(status)
					return d.StateResponse{Ok: true, State: got, Status: status}, d.ProcessInstance{}, nil
				}
				log.Info(fmt.Sprintf("process instance %s is absent (not found); waiting... (attempt #%d)", key, attempts))
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

func stateIn(st d.State, set d.States) bool {
	for _, x := range set {
		if st.EqualsIgnoreCase(x) {
			return true
		}
	}
	return false
}
