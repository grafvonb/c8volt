// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	jobsvc "github.com/grafvonb/c8volt/internal/services/job"
	opsvc "github.com/grafvonb/c8volt/internal/services/ops"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDOperationsRepairFacade consumes real v810 plans through old adapters and preserves copied report evidence.
func TestBusinessIDOperationsRepairFacade(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			var calls, mutations atomic.Int32
			s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-instances/search":
					if empty {
						fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
					} else {
						fmt.Fprint(w, `{"items":[{"processInstanceKey":"101","tenantId":"tenant-a","state":"ACTIVE","hasIncident":true}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
					}
				case "/v2/process-instances/101/incidents/search":
					fmt.Fprint(w, `{"items":[{"incidentKey":"501","processInstanceKey":"101","tenantId":"tenant-a","state":"ACTIVE","jobKey":"601"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
				case "/v2/incidents/501/resolution":
					mutations.Add(1)
					w.WriteHeader(400)
					fmt.Fprint(w, `{"detail":"reviewed mutation rejected"}`)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer s.Close()
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			newClient := func(version toolx.CamundaVersion) API {
				cfg := config.New()
				cfg.App.CamundaVersion = version
				cfg.App.Tenant = "tenant-a"
				cfg.APIs.Camunda.BaseURL = s.URL + "/v2"
				pi, err := pisvc.New(cfg, s.Client(), log)
				require.NoError(t, err)
				inc, err := incsvc.New(cfg, s.Client(), log)
				require.NoError(t, err)
				job, err := jobsvc.New(cfg, s.Client(), log)
				require.NoError(t, err)
				return New(opsvc.NewWithRepairDependencies(nil, pi, inc, nil, nil, job, version, log), log)
			}
			cli := newClient(toolx.V810)
			request := RepairRequest{DiscoveryMode: RepairDiscoveryModeSearch, ProcessInstanceSelection: process.ProcessInstanceFilter{BusinessId: new(" ref,*Ω "), HasParent: new(false), VariableFilters: process.ProcessInstanceVariableFilterSet{Clauses: []process.ProcessInstanceVariableFilterClause{{Name: "exists", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: new(true)}}}}, RequestedRetries: new(int32(0)), DryRun: true}
			plan, err := cli.RepairProcessInstances(context.Background(), request)
			require.NoError(t, err)
			require.NotNil(t, plan.FrozenPlan)
			*request.ProcessInstanceSelection.BusinessId = "changed"
			*request.ProcessInstanceSelection.HasParent = true
			*request.ProcessInstanceSelection.VariableFilters.Clauses[0].Exists = false
			require.Equal(t, " ref,*Ω ", *plan.FrozenPlan.Selection.Request.SearchRequest.Filter.BusinessId)
			require.False(t, *plan.FrozenPlan.Selection.Request.SearchRequest.Filter.HasParent)
			require.True(t, *plan.FrozenPlan.Selection.Request.SearchRequest.Filter.VariableFilters.Clauses[0].Exists)
			raw, err := json.Marshal(plan.Report)
			require.NoError(t, err)
			require.NotContains(t, string(raw), "FrozenPlan")
			var report RepairAuditReport
			require.NoError(t, json.Unmarshal(raw, &report))
			roundtrip, err := json.Marshal(report)
			require.NoError(t, err)
			require.JSONEq(t, string(raw), string(roundtrip))
			require.Equal(t, plan.Report.FrozenSet.Selection, report.FrozenSet.Selection)
			request = plan.Request
			request.DryRun = false
			request.FrozenPlan = plan.FrozenPlan
			before := calls.Load()
			for _, v := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
				_, err := newClient(v).RepairProcessInstances(context.Background(), request)
				require.ErrorIs(t, err, ferrors.ErrUnsupported)
				// Removing the live criterion must not let a prepared BID plan evade capability checks.
				erased := request
				erased.ProcessInstanceSelection.BusinessId = nil
				_, err = newClient(v).RepairProcessInstances(context.Background(), erased)
				require.ErrorIs(t, err, ferrors.ErrUnsupported)
			}
			require.Equal(t, before, calls.Load())
			require.Zero(t, mutations.Load())
			result, err := cli.RepairProcessInstances(context.Background(), request, foptions.WithNoWait())
			if empty {
				require.NoError(t, err)
				require.Equal(t, before, calls.Load())
			} else {
				require.Error(t, err)
				require.Equal(t, before+1, calls.Load())
				require.Equal(t, int32(1), mutations.Load())
				require.Len(t, result.Plan, 1)
				require.Nil(t, result.FrozenSet.Selection.Items[0].BusinessId)
			}
			require.Equal(t, " ref,*Ω ", *result.Report.Request.ProcessInstanceSelection.BusinessId)
		})
	}
}

// TestBusinessIDOperationsRepairPlanCopies isolates nested actions and observations across both facade conversion directions.
func TestBusinessIDOperationsRepairPlanCopies(t *testing.T) {
	input := RepairFrozenPlan{Variables: map[string]any{"nested": map[string]any{"list": []any{"approved"}}}, RequestedRetries: new(int32(1)), Selection: process.ProcessInstanceSelection{Keys: []string{"101"}, Items: []process.ProcessInstance{{Key: "101", BusinessId: new("observed"), Variables: map[string]any{"nested": []any{"observed"}}}}}}
	domain := toDomainRepairFrozenPlan(input)
	input.Variables["nested"].(map[string]any)["list"].([]any)[0] = "changed"
	*input.RequestedRetries = 9
	*input.Selection.Items[0].BusinessId = "changed"
	output := fromDomainRepairFrozenPlan(domain)
	require.Equal(t, "approved", output.Variables["nested"].(map[string]any)["list"].([]any)[0])
	require.Equal(t, int32(1), *output.RequestedRetries)
	require.Equal(t, "observed", *output.Selection.Items[0].BusinessId)
	output.Variables["nested"].(map[string]any)["list"].([]any)[0] = "output mutation"
	output.Selection.Items[0].Variables["nested"].([]any)[0] = "output mutation"
	require.Equal(t, "approved", domain.Variables["nested"].(map[string]any)["list"].([]any)[0])
	require.Equal(t, "observed", domain.Selection.Items[0].Variables["nested"].([]any)[0])
}
