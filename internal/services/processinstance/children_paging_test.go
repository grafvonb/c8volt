// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package processinstance_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/processinstance"
	"github.com/grafvonb/c8volt/internal/services/processinstance/traversal"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCompleteChildrenTraversal exhausts more than 1000 siblings, a sparse page and a deeper child, or preserves incomplete evidence.
func TestCompleteChildrenTraversal(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"complete", "late error", "cursor cycle", "unresolved total", "cancel"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			rootPages := 0
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.Method == http.MethodGet {
					fmt.Fprint(w, `{"processInstanceKey":"1","state":"ACTIVE"}`)
					return
				}
				var body struct {
					Filter map[string]any `json:"filter"`
					Page   map[string]any `json:"page"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				parent := fmt.Sprint(body.Filter["parentProcessInstanceKey"])
				items := []map[string]any{}
				page := map[string]any{"totalItems": 0}
				switch parent {
				case "1":
					rootPages++
					switch rootPages {
					case 1:
						for i := 2; i <= 1001; i++ {
							items = append(items, map[string]any{"processInstanceKey": fmt.Sprint(i), "parentProcessInstanceKey": "1", "state": "ACTIVE"})
						}
						page = map[string]any{"totalItems": 1001, "hasMoreTotalItems": true, "endCursor": "next"}
					case 2:
						require.Equal(t, "next", body.Page["after"])
						if mode == "late error" {
							w.WriteHeader(400)
							fmt.Fprint(w, `{"detail":"owned later child failure"}`)
							return
						}
						if mode == "cancel" {
							cancel()
							return
						}
						page = map[string]any{"totalItems": 1001, "hasMoreTotalItems": true, "endCursor": "sparse"}
						if mode == "cursor cycle" {
							page["endCursor"] = "next"
						}
						if mode == "unresolved total" {
							page = map[string]any{"totalItems": 1001, "hasMoreTotalItems": true}
						}
					case 3:
						require.Equal(t, "sparse", body.Page["after"])
						items = append(items, map[string]any{"processInstanceKey": "1002", "parentProcessInstanceKey": "1", "state": "ACTIVE"})
						page = map[string]any{"totalItems": 1001}
					default:
						t.Error("unexpected root page")
					}
				case "1002":
					items = append(items, map[string]any{"processInstanceKey": "1003", "parentProcessInstanceKey": "1002", "state": "ACTIVE"})
					page = map[string]any{"totalItems": 1}
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"items": items, "page": page}))
			}))
			defer server.Close()
			cfg := testConfig()
			cfg.App.CamundaVersion = toolx.V810
			cfg.APIs.Camunda.BaseURL = server.URL
			api, err := processinstance.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			result, err := api.FamilyResult(ctx, "1")
			if mode == "complete" {
				require.NoError(t, err)
				require.Equal(t, traversal.OutcomeComplete, result.Outcome)
				require.Len(t, result.Keys, 1003)
				require.Len(t, result.Edges["1"], 1001)
				require.Equal(t, []string{"1003"}, result.Edges["1002"])
			} else {
				require.Error(t, err)
				require.NotEqual(t, traversal.OutcomeComplete, result.Outcome)
				require.NotEmpty(t, result.Chain)
				if mode == "cursor cycle" || mode == "unresolved total" {
					require.ErrorIs(t, err, d.ErrMalformedResponse)
				}
				if mode == "cancel" {
					require.ErrorIs(t, err, context.Canceled)
				}
			}
		})
	}
}

// TestBusinessIDSelectionVersionGate rejects supplied BID before any adapter search for either selection cardinality.
func TestBusinessIDSelectionVersionGate(t *testing.T) {
	t.Parallel()
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request %s", r.URL.Path)
		w.WriteHeader(400)
	}))
	defer server.Close()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89} {
		for _, mode := range []d.ProcessInstanceSelectionCardinality{d.ProcessInstanceSelectionCollection, d.ProcessInstanceSelectionUniqueStart} {
			cfg := testConfig()
			cfg.App.CamundaVersion = version
			cfg.APIs.Camunda.BaseURL = server.URL
			api, err := processinstance.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			bid := "order"
			result, err := processinstance.ResolveProcessInstanceSelection(context.Background(), api, nil, d.ProcessInstanceSelectionRequest{Cardinality: mode, SearchRequest: d.ProcessInstanceSearchRequest{Filter: d.ProcessInstanceFilter{BusinessId: &bid}}})
			require.ErrorIs(t, err, d.ErrUnsupported)
			require.Equal(t, d.ProcessInstanceSearchCompletionIncomplete, result.Completion)
			require.Empty(t, result.Keys)
		}
	}
}
