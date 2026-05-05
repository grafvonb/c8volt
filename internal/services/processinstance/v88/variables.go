// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v88

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

func (s *Service) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	_ = ctx
	_ = key
	_ = services.ApplyCallOptions(opts)
	return nil, fmt.Errorf("%w: process-instance variable lookup is not implemented for Camunda 8.8", d.ErrUnsupported)
}
