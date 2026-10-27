// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// ValidateProcessInstanceResumption rejects unsupported mutation without I/O.
func (s *Service) ValidateProcessInstanceResumption() error {
	return fmt.Errorf("%w: process-instance resumption requires Camunda 8.10", d.ErrUnsupported)
}

// ResumeProcessInstance rejects even preview/no-wait calls before any request.
func (s *Service) ResumeProcessInstance(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceResumptionResult, error) {
	err := s.ValidateProcessInstanceResumption()
	return d.ProcessInstanceResumptionResult{Key: key, Status: d.ProcessInstanceResumptionStatusNotAttempted, Submission: d.ProcessInstanceResumptionSubmissionNotSubmitted, Confirmation: d.ProcessInstanceResumptionConfirmationNotRequested, Error: err.Error()}, err
}
