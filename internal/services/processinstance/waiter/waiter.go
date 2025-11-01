package waiter

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

type PIWaiter interface {
	GetProcessInstanceByKey(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstance, error)
	GetProcessInstanceStateByKey(ctx context.Context, key string, opts ...services.CallOption) (d.State, error)
}

// WaitForProcessInstanceState waits until the instance reaches one of the desired states.
// - Respects ctx cancellation/deadline; augments with cfg.Timeout if set
// - Returns nil on success or an error on failure/timeout.
func WaitForProcessInstanceState(ctx context.Context, s PIWaiter, cfg *config.Config, log *slog.Logger, key string, desired d.States, opts ...services.CallOption) (d.State, error) {
	_ = services.ApplyCallOptions(opts)
	backoff := cfg.App.Backoff
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
		if errInDelay := ctx.Err(); errInDelay != nil {
			return "", errInDelay
		}
		attempts++
		got, errInDelay := s.GetProcessInstanceStateByKey(ctx, key)
		if errInDelay == nil {
			if stateIn(got, desired) {
				if attempts == 1 {
					log.Debug(fmt.Sprintf("process instance %s is already in one of the desired state(s) [%s] (current: %s)", key, desired, got))
					return got, nil
				}
				log.Debug(fmt.Sprintf("process instance %s reached one of the desired state(s) [%s] (current: %s) after %d checks", key, desired, got, attempts))
				return got, nil
			}
			log.Info(fmt.Sprintf("process instance %s currently in state %s; waiting...", key, got))
		} else if errInDelay != nil {
			if strings.Contains(errInDelay.Error(), "404") {
				log.Debug(fmt.Sprintf("process instance %s is absent (not found); waiting...", key))
			} else {
				log.Error(fmt.Sprintf("fetching state for %q failed: %v (will retry)", key, errInDelay))
			}
		}
		if backoff.MaxRetries > 0 && attempts >= backoff.MaxRetries {
			return "", fmt.Errorf("exceeded max_retries (%d) waiting for state %q", backoff.MaxRetries, desired)
		}
		select {
		case <-time.After(delay):
			delay = backoff.NextDelay(delay)
		case <-ctx.Done():
			return "", fmt.Errorf("%w: %s", d.ErrGatewayTimeout, ctx.Err().Error())
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
