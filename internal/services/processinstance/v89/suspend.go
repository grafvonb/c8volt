// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// ValidateProcessInstanceSuspension rejects unsupported mutation without I/O.
func (s *Service) ValidateProcessInstanceSuspension() error {
	return fmt.Errorf("%w: process-instance suspension requires Camunda 8.10", d.ErrUnsupported)
}

// SuspendProcessInstance rejects even preview/no-wait calls before any request.
func (s *Service) SuspendProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceSuspensionResult, error) {
	err := s.ValidateProcessInstanceSuspension()
	return d.ProcessInstanceSuspensionResult{Key: key, Status: d.ProcessInstanceSuspensionStatusNotAttempted, Submission: d.ProcessInstanceSuspensionSubmissionNotSubmitted, Confirmation: d.ProcessInstanceSuspensionConfirmationNotRequested, Error: err.Error()}, err
}
