// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"fmt"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
)

// GetIncident rejects direct incident lookup because Camunda 8.7 has no tenant-safe endpoint.
func (s *Service) GetIncident(ctx context.Context, key string, opts ...services.CallOption) (d.ProcessInstanceIncidentDetail, error) {
	_ = ctx
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("rejecting direct incident lookup for key %s because Camunda 8.7 has no tenant-safe endpoint", key))
	return d.ProcessInstanceIncidentDetail{}, fmt.Errorf("%w: direct incident lookup is not tenant-safe in Camunda 8.7", d.ErrUnsupported)
}

// ResolveIncident rejects incident resolution before mutation because Camunda 8.7 has no supported endpoint.
func (s *Service) ResolveIncident(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	_ = ctx
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("rejecting incident resolution for key %s because Camunda 8.7 has no supported endpoint", key))
	return d.IncidentResolutionResponse{Key: key, Ok: false, Status: "unsupported"}, fmt.Errorf("%w: incident resolution is not supported in Camunda 8.7", d.ErrUnsupported)
}

// ResolveProcessInstanceIncidents rejects incident resolution before mutation because Camunda 8.7 has no supported endpoint.
func (s *Service) ResolveProcessInstanceIncidents(ctx context.Context, processInstanceKey string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	_ = ctx
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("rejecting process-instance incident resolution for key %s because Camunda 8.7 has no supported endpoint", processInstanceKey))
	return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: "unsupported"}, fmt.Errorf("%w: process-instance incident resolution is not supported in Camunda 8.7", d.ErrUnsupported)
}

// SearchProcessInstanceIncidents rejects incident lookup because Camunda 8.7 has no tenant-safe endpoint.
func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	_ = ctx
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("rejecting incident lookup for process instance with key %s because Camunda 8.7 has no tenant-safe endpoint", key))
	return nil, fmt.Errorf("%w: process-instance incident lookup is not tenant-safe in Camunda 8.7", d.ErrUnsupported)
}

// WaitForIncidentResolved rejects resolution confirmation because direct incident lookup is unsupported.
func (s *Service) WaitForIncidentResolved(ctx context.Context, key string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	_ = ctx
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("rejecting incident resolution wait for key %s because Camunda 8.7 has no tenant-safe endpoint", key))
	return d.IncidentResolutionResponse{Key: key, Ok: false, Status: "unsupported"}, fmt.Errorf("%w: incident resolution confirmation is not supported in Camunda 8.7", d.ErrUnsupported)
}

// WaitForProcessInstanceIncidentsResolved rejects process-instance incident confirmation because lookup is unsupported.
func (s *Service) WaitForProcessInstanceIncidentsResolved(ctx context.Context, processInstanceKey string, incidentKeys []string, opts ...services.CallOption) (d.IncidentResolutionResponse, error) {
	_ = ctx
	_ = incidentKeys
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("rejecting process-instance incident resolution wait for key %s because Camunda 8.7 has no tenant-safe endpoint", processInstanceKey))
	return d.IncidentResolutionResponse{Key: processInstanceKey, Ok: false, Status: "unsupported"}, fmt.Errorf("%w: process-instance incident resolution confirmation is not supported in Camunda 8.7", d.ErrUnsupported)
}
