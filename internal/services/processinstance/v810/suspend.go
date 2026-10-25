// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"errors"
	"fmt"
	"net"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/internal/services/processinstance/waiter"
	"github.com/grafvonb/c8volt/typex"
)

// ValidateProcessInstanceSuspension permits the native Camunda 8.10 operation.
func (s *Service) ValidateProcessInstanceSuspension() error { return nil }

// SuspendProcessInstance previews or submits only the selected key. Default calls
// confirm SUSPENDED after acceptance; NoWait needs no read permission. Backend
// rejection and uncertainty never trigger reconciliation or hierarchy discovery.
func (s *Service) SuspendProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
	result := d.ProcessInstanceSuspensionResult{Key: key, Status: d.ProcessInstanceSuspensionStatusNotAttempted, Submission: d.ProcessInstanceSuspensionSubmissionNotSubmitted, Confirmation: d.ProcessInstanceSuspensionConfirmationNotRequested}
	if err := s.ValidateProcessInstanceSuspension(); err != nil {
		return suspensionFailure(result, err)
	}
	keys, err := common.NormalizeProcessInstanceSuspensionKeys(typex.Keys{key})
	if err != nil {
		return suspensionFailure(result, err)
	}
	key = keys[0]
	result.Key = key
	if err := ctx.Err(); err != nil {
		return suspensionFailure(result, err)
	}
	cfg := services.ApplyCallOptions(opts)
	// Preview is local scope validation. NoWait does not use confirmation timing.
	if !cfg.NoWait {
		if s.cfg == nil {
			return suspensionFailure(result, fmt.Errorf("%w: suspension requires backoff configuration", d.ErrValidation))
		}
		if err := s.cfg.App.Backoff.Validate(); err != nil {
			return suspensionFailure(result, fmt.Errorf("%w: suspension backoff: %w", d.ErrValidation, err))
		}
	}
	if cfg.DryRun {
		result.Status = d.ProcessInstanceSuspensionStatusPreview
		return result, nil
	}
	result, err = s.submitProcessInstanceSuspension(ctx, key)
	if err != nil || cfg.NoWait {
		return result, err
	}
	result.Confirmation = d.ProcessInstanceSuspensionConfirmationUnconfirmed
	observation, err := waiter.WaitForProcessInstanceSuspension(ctx, s, s.cfg.App.Backoff, key, opts...)
	result.HasObservation = observation.HasObservation
	result.LastObservedState = observation.LastObservedState
	result.SuspendedDate = observation.SuspendedDate
	if err != nil {
		return suspensionFailure(result, err)
	}
	result.Status = d.ProcessInstanceSuspensionStatusConfirmed
	result.Confirmation = d.ProcessInstanceSuspensionConfirmationConfirmed
	return result, nil
}

// suspensionFailure changes workflow disposition without losing submission or
// observation facts. Error remains machine evidence, never a human log message.
func suspensionFailure(result d.ProcessInstanceSuspensionResult, err error) (d.ProcessInstanceSuspensionResult, error) {
	result.Status = d.ProcessInstanceSuspensionStatusFailed
	var networkErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		result.Status = d.ProcessInstanceSuspensionStatusInterrupted
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
		result.Status = d.ProcessInstanceSuspensionStatusTimedOut
	case errors.As(err, &networkErr) && networkErr.Timeout():
		result.Status = d.ProcessInstanceSuspensionStatusTimedOut
		err = fmt.Errorf("%w: %w", d.ErrGatewayTimeout, err)
	}
	err = fmt.Errorf("suspend process instance: %w", err)
	result.Error = err.Error()
	return result, err
}
