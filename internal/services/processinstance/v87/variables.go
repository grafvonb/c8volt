// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v87

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	operatev87 "github.com/grafvonb/c8volt/internal/clients/camunda/v87/operate"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/common"
	"github.com/grafvonb/c8volt/toolx"
)

// SearchProcessInstanceVariables uses Operate for v8.7 so variable inspection stays available before the v2 endpoint exists.
func (s *Service) SearchProcessInstanceVariables(ctx context.Context, key string, opts ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	_ = services.ApplyCallOptions(opts)
	s.log.Debug(fmt.Sprintf("searching variables for process instance with key %s using operate client", key))
	keyInt, err := processInstanceKeyInt64(key)
	if err != nil {
		return nil, err
	}
	size := int32(1000)
	order := operatev87.ASC
	sortField := "name"
	body := operatev87.SearchVariablesForProcessInstancesJSONRequestBody{
		Filter: &operatev87.Variable{
			ProcessInstanceKey: &keyInt,
			ScopeKey:           &keyInt,
			TenantId:           toolx.PtrIf(s.cfg.App.Tenant, ""),
		},
		Size: &size,
		Sort: &[]operatev87.Sort{{
			Field: &sortField,
			Order: &order,
		}},
	}
	resp, err := s.co.SearchVariablesForProcessInstancesWithResponse(ctx, body)
	if err != nil {
		return nil, err
	}
	payload, err := common.RequirePayload(resp.HTTPResponse, resp.Body, resp.JSON200)
	if err != nil {
		return nil, err
	}
	if payload.Items == nil {
		return []d.ProcessInstanceVariable{}, nil
	}
	variables := toolx.MapSlice(*payload.Items, fromOperateVariable)
	sort.SliceStable(variables, func(i, j int) bool {
		return variables[i].Name < variables[j].Name
	})
	return variables, nil
}

func fromOperateVariable(v operatev87.Variable) d.ProcessInstanceVariable {
	return d.ProcessInstanceVariable{
		Name:               toolx.Deref(v.Name, ""),
		Value:              toolx.Deref(v.Value, ""),
		VariableKey:        formatOperateVariableKey(v.Key),
		ProcessInstanceKey: formatOperateVariableKey(v.ProcessInstanceKey),
		ScopeKey:           formatOperateVariableKey(v.ScopeKey),
		TenantId:           toolx.Deref(v.TenantId, ""),
		APITruncated:       toolx.Deref(v.Truncated, false),
	}
}

func formatOperateVariableKey(key *int64) string {
	if key == nil {
		return ""
	}
	return strconv.FormatInt(*key, 10)
}
