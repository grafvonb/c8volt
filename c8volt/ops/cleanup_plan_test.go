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
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	opsvc "github.com/grafvonb/c8volt/internal/services/ops"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDOperationsCleanupFacade round-trips provenance and rejects actual v810 plans on older receiving adapters before I/O.
func TestBusinessIDOperationsCleanupFacade(t *testing.T) {
	for _, kind := range []string{"retention", "orphan"} {
		for _, empty := range []bool{false, true} {
			t.Run(fmt.Sprint(kind, empty), func(t *testing.T) {
				var calls, mutations atomic.Int32
				item := map[string]any{"processInstanceKey": "101", "state": "COMPLETED", "endDate": "2020-01-01T00:00:00Z", "suspendedDate": "2019-12-31T00:00:00Z", "tenantId": "tenant-a"}
				if kind == "orphan" {
					item["parentProcessInstanceKey"] = "901"
				}
				s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/v2/process-instances/search":
						var b struct {
							Filter map[string]any `json:"filter"`
						}
						require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
						items := []any{}
						if _, ok := b.Filter["businessId"]; ok && !empty {
							items = append(items, item)
						}
						require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": map[string]any{"totalItems": len(items), "hasMoreTotalItems": false}}))
					case "/v2/process-instances/101":
						require.NoError(t, json.NewEncoder(w).Encode(item))
					case "/v2/process-instances/901":
						w.WriteHeader(404)
						fmt.Fprint(w, `{}`)
					case "/v2/process-instances/101/deletion":
						mutations.Add(1)
						w.WriteHeader(400)
						fmt.Fprint(w, `{"detail":"owned rejection"}`)
					default:
						t.Errorf("unexpected path %s", r.URL.Path)
						w.WriteHeader(500)
					}
				}))
				defer s.Close()
				log := slog.New(slog.NewTextHandler(io.Discard, nil))
				client := func(version toolx.CamundaVersion) API {
					cfg := config.New()
					cfg.App.CamundaVersion = version
					cfg.App.Tenant = "tenant-a"
					cfg.APIs.Camunda.BaseURL = s.URL + "/v2"
					pi, err := pisvc.New(cfg, s.Client(), log)
					require.NoError(t, err)
					return New(opsvc.NewWithRepairDependencies(nil, pi, nil, nil, nil, nil, version, log), log)
				}
				filter := process.ProcessInstanceFilter{BusinessId: new(" ref,Ω* "), HasIncident: new(false)}
				var originalReport any
				var roundtripReport any
				var execute func(API) error
				var scope CleanupFrozenScope
				if kind == "retention" {
					req := RetentionPolicyRequest{Selection: filter, DryRun: true, DerivedEndDateBoundary: "2026-01-01", RetentionDays: 30}
					plan, err := client(toolx.V810).ExecuteRetentionPolicy(context.Background(), req)
					require.NoError(t, err)
					require.NotNil(t, plan.FrozenPlan)
					scope = fromDomainCleanupFrozenScope(toDomainCleanupFrozenScope(plan.FrozenPlan.Scope))
					originalReport = plan.Report
					roundtripReport = &RetentionAuditReport{}
					req = plan.Request
					req.DryRun = false
					req.FrozenPlan = plan.FrozenPlan
					execute = func(c API) error { _, err := c.ExecuteRetentionPolicy(context.Background(), req); return err }
					converted := toDomainRetentionPolicyRequest(req)
					*req.FrozenPlan.Scope.Selection.Request.SearchRequest.Filter.BusinessId = "changed-plan"
					require.Equal(t, " ref,Ω* ", *converted.FrozenPlan.Scope.Selection.Request.SearchRequest.Filter.BusinessId)
					req.FrozenPlan = toolx.MapPtr(converted.FrozenPlan, fromDomainRetentionFrozenPlan)
				} else {
					req := OrphanPurgeRequest{Selection: filter, DryRun: true}
					plan, err := client(toolx.V810).PurgeOrphanProcessInstances(context.Background(), req)
					require.NoError(t, err)
					require.NotNil(t, plan.FrozenPlan)
					scope = fromDomainCleanupFrozenScope(toDomainCleanupFrozenScope(plan.FrozenPlan.Scope))
					originalReport = plan.Report
					roundtripReport = &OrphanPurgeReport{}
					req = plan.Request
					req.DryRun = false
					req.FrozenPlan = plan.FrozenPlan
					execute = func(c API) error { _, err := c.PurgeOrphanProcessInstances(context.Background(), req); return err }
					converted := toDomainOrphanPurgeRequest(req)
					*req.FrozenPlan.Scope.Selection.Request.SearchRequest.Filter.BusinessId = "changed-plan"
					require.Equal(t, " ref,Ω* ", *converted.FrozenPlan.Scope.Selection.Request.SearchRequest.Filter.BusinessId)
					req.FrozenPlan = toolx.MapPtr(converted.FrozenPlan, fromDomainOrphanPurgeFrozenPlan)
				}
				*filter.BusinessId = "caller-change"
				*filter.HasIncident = true
				require.Equal(t, " ref,Ω* ", *scope.Selection.Request.SearchRequest.Filter.BusinessId)
				if !empty {
					require.Equal(t, "2019-12-31T00:00:00Z", scope.Selection.Items[0].SuspendedDate)
				}
				raw, err := json.Marshal(originalReport)
				require.NoError(t, err)
				require.NotContains(t, string(raw), "FrozenPlan")
				require.NoError(t, json.Unmarshal(raw, roundtripReport))
				again, err := json.Marshal(roundtripReport)
				require.NoError(t, err)
				require.JSONEq(t, string(raw), string(again))
				before := calls.Load()
				for _, v := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
					require.ErrorIs(t, execute(client(v)), ferrors.ErrUnsupported)
				}
				require.Equal(t, before, calls.Load())
				require.Zero(t, mutations.Load())
				err = execute(client(toolx.V810))
				if empty {
					require.NoError(t, err)
					require.Equal(t, before, calls.Load())
				} else {
					require.Error(t, err)
					require.Equal(t, int32(1), mutations.Load())
					require.LessOrEqual(t, calls.Load(), before+2)
				}
			})
		}
	}
}
