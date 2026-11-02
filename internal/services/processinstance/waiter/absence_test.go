// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package waiter

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/stretchr/testify/require"
)

// TestWaitForProcessInstanceLookupErrorsAreNotAbsence preserves failure identity
// when diagnostic URLs, resource keys or bodies contain apparent absence text.
func TestWaitForProcessInstanceLookupErrorsAreNotAbsence(t *testing.T) {
	t.Parallel()
	for name, cause := range map[string]error{
		"bad request port":       fmt.Errorf("%w: 400 GET http://owned.invalid:40401/pi/123", d.ErrBadRequest),
		"forbidden key":          fmt.Errorf("%w: 403 GET http://owned.invalid/pi/404123", d.ErrForbidden),
		"unauthorized body":      fmt.Errorf("%w: upstream configuration not found", d.ErrUnauthorized),
		"upstream body":          fmt.Errorf("%w: dependency does not exist", d.ErrUpstream),
		"unclassified transport": errors.New("GET http://owned.invalid:40401/pi/123: connection refused"),
	} {
		for _, mode := range []string{"absence", "creation", "incident expectation"} {
			t.Run(name+"/"+mode, func(t *testing.T) {
				calls := 0
				lookup := func(context.Context, string) (d.ProcessInstance, error) {
					calls++
					return d.ProcessInstance{}, cause
				}
				reader := stubPIWaiter{getProcessInstance: lookup, getStateByKey: func(ctx context.Context, key string) (d.State, d.ProcessInstance, error) {
					pi, err := lookup(ctx, key)
					return d.StateUnknown, pi, err
				}}
				cfg := testConfig(time.Millisecond, 2, time.Second)
				if mode == "incident expectation" {
					result, _, err := WaitForProcessInstanceExpectation(context.Background(), reader, cfg, testLogger(), "123", d.ProcessInstanceExpectationRequest{Incident: new(false)})
					require.ErrorIs(t, err, cause)
					require.False(t, result.Ok)
				} else {
					desired := d.States{d.StateAbsent}
					if mode == "creation" {
						desired = d.States{d.StateActive}
					}
					result, _, err := WaitForProcessInstanceState(context.Background(), reader, cfg, testLogger(), "123", desired)
					require.ErrorIs(t, err, cause)
					require.False(t, result.Ok)
				}
				require.Equal(t, 1, calls, "lookup errors must not be retried as absence")
			})
		}
	}
}
