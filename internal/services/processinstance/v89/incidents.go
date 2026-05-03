// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v89

import (
	"context"
	"fmt"

	camundav89 "github.com/grafvonb/c8volt/internal/clients/camunda/v89/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// SearchProcessInstanceIncidents uses the scoped process-instance incident endpoint and applies only tenant/page filters locally.
func (s *Service) SearchProcessInstanceIncidents(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceIncidentDetail, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching incidents for process instance with key %s using generated camunda client", key))
	tenantFilter, err := newStringEqFilterPtr(s.cfg.App.Tenant)
	if err != nil {
		return nil, fmt.Errorf("building tenant incident filter: %w", err)
	}
	page := newSearchQueryPageRequest(d.ProcessInstancePageRequest{Size: 1000})
	body := camundav89.SearchProcessInstanceIncidentsJSONRequestBody{
		Page: &page,
	}
	if tenantFilter != nil {
		body.Filter = &camundav89.IncidentFilter{
			TenantId: tenantFilter,
		}
	}
	resp, err := s.cc.SearchProcessInstanceIncidentsWithResponse(ctx, key, body)
	if err != nil {
		return nil, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return nil, err
	}
	return toolx.MapSlice(payload.Items, fromIncidentResult), nil
}
