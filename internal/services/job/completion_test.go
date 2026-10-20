// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/job"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestBulkCompletionPreparationAdapters proves all factories wire read-only preparation,
// enforce discovery tenant, retain explicit authorization, and reject unsupported empty batches.
func TestBulkCompletionPreparationAdapters(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V87, toolx.V88, toolx.V89, toolx.V810} {
		t.Run(string(version), func(t *testing.T) {
			t.Parallel()
			for _, mode := range []string{"search", "explicit", "empty", "incomplete", "listener_payload"} {
				t.Run(mode, func(t *testing.T) {
					var reads, writes atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodPost || r.URL.Path != "/jobs/search" {
							writes.Add(1)
							http.Error(w, "unexpected mutation", 500)
							return
						}
						reads.Add(1)
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Error(err)
							http.Error(w, "bad JSON", 400)
							return
						}
						filter, ok := body["filter"].(map[string]any)
						if !ok {
							t.Error("missing filter")
							http.Error(w, "missing filter", 400)
							return
						}
						w.Header().Set("Content-Type", "application/json")
						if mode == "explicit" {
							if _, exists := filter["tenantId"]; exists {
								t.Error("explicit keys must not inherit discovery tenant")
							}
							key, ok := filter["jobKey"].(string)
							if !ok {
								t.Error("missing key equality")
								return
							}
							fmt.Fprintf(w, `{"items":[{"jobKey":%q,"tenantId":"other","type":"work","state":"CREATED","kind":"BPMN_ELEMENT"}],"page":{"totalItems":1}}`, key)
							return
						}
						tenant, ok := filter["tenantId"].(string)
						if !ok || tenant != "tenant" {
							t.Errorf("missing named tenant: %v", filter)
						}
						if _, exists := body["sort"]; !exists {
							t.Error("missing stable key sort")
						}
						if mode == "empty" {
							fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
							return
						}
						if mode == "incomplete" {
							fmt.Fprint(w, `{"items":[],"page":{"totalItems":1}}`)
							return
						}
						// A wrong-tenant row must be consumed as raw progress but never selected.
						fmt.Fprint(w, `{"items":[{"jobKey":"1","tenantId":"other","type":"work","state":"CREATED","kind":"TASK_LISTENER"},{"jobKey":"2","tenantId":"tenant","type":"work","state":"CREATED","kind":"TASK_LISTENER"}],"page":{"totalItems":2}}`)
					}))
					defer server.Close()
					cfg := testConfig()
					cfg.APIs.Camunda.BaseURL = server.URL
					cfg.App.CamundaVersion = version
					cfg.App.Tenant = "tenant"
					svc, err := job.New(cfg, server.Client(), nil)
					require.NoError(t, err)
					request := d.CompletionRequest{Selection: d.CompletionSelection{Mode: d.CompletionSelectionSearch, InputForm: d.CompletionInputFilters, Filter: d.CompletionFilter{Type: "work"}, BatchSize: 2}}
					if mode == "explicit" {
						request.Selection = d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"9007199254740993", "2", "2"}}
					}
					if mode == "empty" && version == toolx.V87 {
						request.Selection = d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputStdin}
					}
					if mode == "listener_payload" {
						request.Variables = map[string]any{"x": true}
					}
					batch, err := svc.PrepareJobCompletionBatch(context.Background(), request, services.WithIgnoreTenant(), services.WithForce(), services.WithNoStateCheck(), services.WithDryRun())
					require.Zero(t, writes.Load())
					if version == toolx.V87 {
						require.ErrorIs(t, err, d.ErrUnsupported)
						require.Zero(t, reads.Load())
						require.Equal(t, d.CompletionPreparationIncomplete, batch.Status)
						return
					}
					if mode == "incomplete" || mode == "listener_payload" {
						require.Error(t, err)
						require.Equal(t, d.CompletionPreparationIncomplete, batch.Status)
						return
					}
					require.NoError(t, err)
					require.Equal(t, d.CompletionPreparationComplete, batch.Status)
					switch mode {
					case "explicit":
						require.Equal(t, int32(2), reads.Load())
						require.Len(t, batch.Candidates, 2)
						require.Equal(t, []string{"other"}, batch.TenantIDs)
						require.Equal(t, "9007199254740993", batch.Candidates[0].Key)
					case "empty":
						require.Empty(t, batch.Candidates)
						require.NotNil(t, batch.Candidates)
					default:
						require.Len(t, batch.Candidates, 1)
						require.Equal(t, "2", batch.Candidates[0].Key)
						require.Equal(t, int64(2), batch.ObservedCount)
					}
				})
			}
		})
	}
}
