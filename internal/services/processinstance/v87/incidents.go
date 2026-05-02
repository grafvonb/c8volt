// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	_ = ctx
	_ = key
	_ = services.ApplyCallOptions(opts)
	return nil, fmt.Errorf("%w: process-instance incident lookup is not tenant-safe in Camunda 8.7", d.ErrUnsupported)
}
