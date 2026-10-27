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

// ValidateProcessInstanceResumption permits the native Camunda 8.10 operation.
func (s *Service) ValidateProcessInstanceResumption() error { return nil }

// ResumeProcessInstance previews or submits only the selected key. Default calls
// confirm ACTIVE or COMPLETED after acceptance; NoWait needs no read permission. Backend
// rejection and uncertainty never trigger reconciliation or hierarchy discovery.
func (s *Service) ResumeProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
	result := d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusNotAttempted, Submission: d.ProcessInstanceResumptionSubmissionNotSubmitted, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested}
	if err := s.ValidateProcessInstanceResumption(); err != nil {
		return resumptionFailure(result, err)
	}
	keys, err := common.NormalizeProcessInstanceKeys(typex.Keys{key})
	if err != nil {
		return resumptionFailure(result, err)
	}
	key = keys[0]
	result.Key = key
	if err := ctx.Err(); err != nil {
		return resumptionFailure(result, err)
	}
	cfg := services.ApplyCallOptions(opts)
	// Preview is local scope validation. NoWait does not use confirmation timing.
	if !cfg.NoWait {
		if s.cfg == nil {
			return resumptionFailure(result, fmt.Errorf("%w: resumption requires backoff configuration", d.ErrValidation))
		}
		if err := s.cfg.App.Backoff.Validate(); err != nil {
			return resumptionFailure(result, fmt.Errorf("%w: resumption backoff: %w", d.ErrValidation, err))
		}
	}
	if cfg.DryRun {
		result.Status = d.ProcessInstanceResumptionStatusPreview
		return result, nil
	}
	result, err = s.submitProcessInstanceResumption(ctx, key)
	if err != nil || cfg.NoWait {
		return result, err
	}
	result.Confirmation = d.ProcessInstanceResumptionConfirmationUnconfirmed
	observation, err := waiter.WaitForProcessInstanceResumption(ctx, s, s.cfg.App.Backoff, key, opts...)
	result.HasObservation = observation.HasObservation
	result.LastObservedState = observation.LastObservedState
	result.SuspendedDate = observation.SuspendedDate
	if err != nil {
		return resumptionFailure(result, err)
	}
	result.Status = d.ProcessInstanceResumptionStatusConfirmed
	result.Confirmation = d.ProcessInstanceResumptionConfirmationConfirmed
	return result, nil
}

// resumptionFailure changes workflow disposition without losing submission or
// observation facts. Error remains machine evidence, never a human log message.
func resumptionFailure(result d.ProcessInstanceResumptionResult, err error) (d.ProcessInstanceResumptionResult, error) {
	result.Status = d.ProcessInstanceResumptionStatusFailed
	var networkErr net.Error
	switch {
	case errors.Is(err, context.Canceled):
		result.Status = d.ProcessInstanceResumptionStatusInterrupted
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, d.ErrGatewayTimeout):
		result.Status = d.ProcessInstanceResumptionStatusTimedOut
	case errors.As(err, &networkErr) && networkErr.Timeout():
		result.Status = d.ProcessInstanceResumptionStatusTimedOut
		err = fmt.Errorf("%w: %w", d.ErrGatewayTimeout, err)
	}
	err = fmt.Errorf("resume process instance: %w", err)
	result.Error = err.Error()
	return result, err
}
