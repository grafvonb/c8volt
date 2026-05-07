// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// SearchProcessInstanceIncidents rejects incident lookup because Camunda 8.7 has no tenant-safe endpoint.
func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	_ = ctx
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("rejecting incident lookup for process instance with key %s because Camunda 8.7 has no tenant-safe endpoint", key))
	return nil, fmt.Errorf("%w: process-instance incident lookup is not tenant-safe in Camunda 8.7", d.ErrUnsupported)
}
