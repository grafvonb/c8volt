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
	"github.com/grafvonb/c8volt/toolx"
)

// BusinessIDAssignmentObservation holds independent later facts without changing submission evidence.
type BusinessIDAssignmentObservation struct {
	HasObservation bool
	BusinessID     *string
}

// WaitForProcessInstanceBusinessIDAssignment polls an exact key only after known assignment acceptance.
func WaitForProcessInstanceBusinessIDAssignment(ctx context.Context, reader ResumptionReader, backoff config.BackoffConfig, key, value string, opts ...services.CallOption) (BusinessIDAssignmentObservation, error) {
	var observation BusinessIDAssignmentObservation
	if err := backoff.Validate(); err != nil {
		return observation, fmt.Errorf("%w: assignment backoff: %w", d.ErrValidation, err)
	}
	ctx, cancel := context.WithTimeout(httpc.WithoutReadRetryLogs(ctx), backoff.Timeout)
	defer cancel()
	nested := append([]services.CallOption(nil), opts...)
	nested = append(nested, services.WithSuppressNestedProcessInstanceLookupLogs(), services.WithSuppressProcessInstanceDetailLogs())
	delay := backoff.InitialDelay
	for attempt := 1; ; attempt++ {
		if err := ctx.Err(); err != nil {
			return observation, err
		}
		pi, err := reader.GetProcessInstance(ctx, key, nested...)
		if ctx.Err() != nil {
			return observation, errors.Join(ctx.Err(), err)
		}
		if err != nil && !errors.Is(err, d.ErrNotFound) {
			return observation, err
		}
		if err == nil {
			if pi.Key != key {
				return observation, fmt.Errorf("%w: assignment observation identity mismatch", d.ErrMalformedResponse)
			}
			observation = BusinessIDAssignmentObservation{HasObservation: true, BusinessID: toolx.CopyPtr(pi.BusinessId)}
			if pi.BusinessId != nil && *pi.BusinessId == value {
				return observation, nil
			}
		}
		if backoff.MaxRetries > 0 && attempt >= backoff.MaxRetries {
			return observation, fmt.Errorf("%w: assignment confirmation attempt limit reached", d.ErrGatewayTimeout)
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
