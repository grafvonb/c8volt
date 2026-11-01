// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/config"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// actionVersionVariables supplies a completed empty root-variable scope without a live service.
type actionVersionVariables struct{}

// SearchProcessInstanceVariables completes variable planning without permitting any mutation API.
func (actionVersionVariables) SearchProcessInstanceVariables(context.Context, string, ...services.CallOption) ([]d.ProcessInstanceVariable, error) {
	return nil, nil
}

// TestBusinessIDActionsPreparedVersionGate rejects actual v810 plans on both older adapters, including empty and dry-run consumption.
func TestBusinessIDActionsPreparedVersionGate(t *testing.T) {
	t.Parallel()
	for _, empty := range []bool{false, true} {
		t.Run(fmt.Sprint(empty), func(t *testing.T) {
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					fmt.Fprint(w, `{"processInstanceKey":"2251799813685101","state":"ACTIVE"}`)
					return
				}
				if empty {
					fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
				} else {
					fmt.Fprint(w, `{"items":[{"processInstanceKey":"2251799813685101","state":"ACTIVE"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
				}
			}))
			defer server.Close()
			cfg := config.New()
			cfg.App.CamundaVersion = toolx.V810
			cfg.APIs.Camunda.BaseURL = server.URL
			log := slog.New(slog.NewTextHandler(io.Discard, nil))
			api, err := processinstance.New(cfg, server.Client(), log)
			require.NoError(t, err)
			request := d.ProcessInstanceSelectionRequest{Cardinality: d.ProcessInstanceSelectionCollection, SearchRequest: d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{BusinessId: toolx.Ptr("ORDER")}}}
			suspend, err := processinstance.PlanSelectedProcessInstanceSuspension(context.Background(), api, nil, request, false, services.WithNoWait())
			require.NoError(t, err)
			resume, err := processinstance.PlanSelectedProcessInstanceResumption(context.Background(), api, nil, request, false, services.WithNoWait())
			require.NoError(t, err)
			updates := processinstance.NewVariableUpdates(api, actionVersionVariables{}, cfg, log)
			update, err := processinstance.PlanSelectedProcessInstanceVariableUpdates(context.Background(), api, nil, updates, request, map[string]any{"x": true})
			require.NoError(t, err)
			before := calls.Load()
			for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
				cfg.App.CamundaVersion = version
				older, err := processinstance.New(cfg, server.Client(), log)
				require.NoError(t, err)
				_, err = processinstance.ExecuteProcessInstanceSuspensionPlan(context.Background(), older, log, suspend, 1, services.WithDryRun())
				require.ErrorIs(t, err, d.ErrUnsupported)
				_, err = processinstance.ExecuteProcessInstanceResumptionPlan(context.Background(), older, log, resume, 1)
				require.ErrorIs(t, err, d.ErrUnsupported)
				olderUpdates := processinstance.NewVariableUpdates(older, nil, cfg, log)
				_, err = olderUpdates.ExecuteProcessInstanceVariableUpdates(context.Background(), update, 1, services.WithDryRun())
				require.ErrorIs(t, err, d.ErrUnsupported)
				_, err = processinstance.PlanSelectedProcessInstanceVariableUpdates(context.Background(), older, nil, olderUpdates, request, nil)
				require.ErrorIs(t, err, d.ErrUnsupported)
			}
			require.Equal(t, before, calls.Load())
		})
	}
}
