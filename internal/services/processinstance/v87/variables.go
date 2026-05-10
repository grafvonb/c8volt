// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// SearchProcessInstanceVariables delegates process-instance variable lookup to the variable service.
func (s *Service) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return s.variableAPI.SearchProcessInstanceVariables(ctx, key, opts...)
}

// UpdateProcessInstanceVariables delegates process-instance variable mutation to the variable service.
func (s *Service) UpdateProcessInstanceVariables(ctx context.Context, key string, variables map[string]any, opts ...services.CallOption) (d.ProcessInstanceVariableUpdateResponse, error) {
	return s.variableAPI.UpdateProcessInstanceVariables(ctx, key, variables, opts...)
}
