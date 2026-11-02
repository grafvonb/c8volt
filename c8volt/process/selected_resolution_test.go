// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package process

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
	"github.com/grafvonb/c8volt/config"
	incsvc "github.com/grafvonb/c8volt/internal/services/incident"
	pisvc "github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBusinessIDResolveFacade exercises actual v810 plans, defensive copies, serialization and old-adapter consumption gates.
func TestBusinessIDResolveFacade(t *testing.T) {
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			var calls, mutations atomic.Int32
			s := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-instances/search":
					var b struct {
						Filter map[string]any `json:"filter"`
					}
					require.NoError(t, json.NewDecoder(r.Body).Decode(&b))
					if _, ok := b.Filter["businessId"]; ok && !empty {
						fmt.Fprint(w, `{"items":[{"processInstanceKey":"101","businessId":"ORDER","tenantId":"tenant-a","state":"ACTIVE"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
					} else {
						fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
					}
				case "/v2/process-instances/101":
					fmt.Fprint(w, `{"processInstanceKey":"101","tenantId":"tenant-a","state":"ACTIVE"}`)
				case "/v2/process-instances/101/incidents/search":
					fmt.Fprint(w, `{"items":[{"incidentKey":"501","processInstanceKey":"101","tenantId":"tenant-a","state":"ACTIVE","jobKey":"601","elementInstanceKey":"701"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
				case "/v2/incidents/501/resolution":
					mutations.Add(1)
					w.WriteHeader(400)
					fmt.Fprint(w, `{"detail":"reviewed incident rejected"}`)
				default:
					t.Errorf("unexpected path %s", r.URL.Path)
					w.WriteHeader(500)
				}
			}))
			defer s.Close()
			cfg := config.New()
			cfg.App.CamundaVersion = toolx.V810
			cfg.App.Tenant = "tenant-a"
			cfg.APIs.Camunda.BaseURL = s.URL + "/v2"
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			pi, err := pisvc.New(cfg, s.Client(), log)
			require.NoError(t, err)
			inc, err := incsvc.New(cfg, s.Client(), log)
			require.NoError(t, err)
			cli := New(nil, pi, inc, log)
			request := ProcessInstanceSelectionRequest{Cardinality: ProcessInstanceSelectionCollection, SearchRequest: ProcessInstanceSearchRequest{Filter: ProcessInstanceFilter{BusinessId: toolx.Ptr("ORDER")}}}
			plan, err := cli.PlanSelectedProcessInstanceResolution(context.Background(), request, 1)
			require.NoError(t, err)
			*request.SearchRequest.Filter.BusinessId = "changed"
			require.Equal(t, "ORDER", *plan.Scope.Request.Selection.SearchRequest.Filter.BusinessId)
			raw, err := json.Marshal(plan)
			require.NoError(t, err)
			var copied SelectedProcessInstanceResolutionPlan
			require.NoError(t, json.Unmarshal(raw, &copied))
			require.Equal(t, plan, copied)
			if !empty {
				require.Equal(t, "601", plan.Incidents.Incidents[0].JobKey)
				require.Equal(t, "701", plan.Incidents.Incidents[0].ElementInstanceKey)
				require.Nil(t, plan.Scope.Families[0].Items[0].BusinessId)
			}
			before := calls.Load()
			for _, v := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
				olderCfg := config.New()
				olderCfg.App.CamundaVersion = v
				olderCfg.App.Tenant = "tenant-a"
				olderCfg.APIs.Camunda.BaseURL = s.URL + "/v2"
				older, err := pisvc.New(olderCfg, s.Client(), log)
				require.NoError(t, err)
				oldcli := New(nil, older, inc, log)
				for _, dry := range []bool{false, true} {
					opts := []foptions.FacadeOption{}
					if dry {
						opts = append(opts, foptions.WithDryRun())
					}
					_, err = oldcli.ExecuteSelectedProcessInstanceResolution(context.Background(), copied, opts...)
					require.Error(t, err)
					require.Contains(t, err.Error(), "unsupported")
				}
				_, err = oldcli.PlanSelectedProcessInstanceResolution(context.Background(), plan.Scope.Request.Selection, 1)
				require.Error(t, err)
			}
			require.Equal(t, before, calls.Load())
			require.Zero(t, mutations.Load())
			result, err := cli.ExecuteSelectedProcessInstanceResolution(context.Background(), plan, foptions.WithDryRun())
			require.NoError(t, err)
			require.Equal(t, before, calls.Load())
			if !empty {
				result.Items[0].Incidents[0].JobKey = "changed"
				require.Equal(t, "601", plan.Incidents.Incidents[0].JobKey)
			}
			result, err = cli.ExecuteSelectedProcessInstanceResolution(context.Background(), plan, foptions.WithNoWait())
			if empty {
				require.NoError(t, err)
				require.False(t, result.MutationSubmitted)
			} else {
				require.ErrorIs(t, err, ferrors.ErrInvalidInput)
				require.Equal(t, 1, result.Failed)
				require.Equal(t, "501", result.Items[0].FailedIncidentKeys[0])
				require.Equal(t, int32(1), mutations.Load())
			}
		})
	}
}
