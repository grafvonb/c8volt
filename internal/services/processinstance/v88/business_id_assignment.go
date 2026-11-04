// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// ValidateBusinessIDAssignment rejects unsupported native assignment before any I/O.
func (s *Service) ValidateBusinessIDAssignment(value *string, opts ...services.CallOption) error {
	return fmt.Errorf("%w: business ID assignment requires Camunda 8.10", d.ErrUnsupported)
}

// AssignProcessInstanceBusinessID rejects even caller-prepared assignment without touching the backend.
func (s *Service) AssignProcessInstanceBusinessID(ctx context.Context, key string, value *string, opts ...services.CallOption) (d.BusinessIDAssignmentResult, error) {
	return d.BusinessIDAssignmentResult{Key: key, Status: "not_submitted", Confirmation: "skipped"}, s.ValidateBusinessIDAssignment(value, opts...)
}
