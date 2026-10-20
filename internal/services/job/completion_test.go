// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package job_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	facade "github.com/grafvonb/c8volt/c8volt/job"
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

// TestBulkCompletionExecutionAdapters exercises the public facade through real version adapters,
// proving strict direct-call dry-run, cross-tenant explicit keys, one-shot HTTP evidence and no hidden mutations.
func TestBulkCompletionExecutionAdapters(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V87, toolx.V88, toolx.V89, toolx.V810} {
		t.Run(string(version), func(t *testing.T) {
			t.Parallel()
			for _, mode := range []string{"accepted", "dry-run", "empty", "conflict completed", "uncertain missing", "listener changed", "completed", "canceled"} {
				t.Run(mode, func(t *testing.T) {
					var executing atomic.Bool
					var reads, writes, unexpected atomic.Int32
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.Header().Set("Content-Type", "application/json")
						if r.Method == http.MethodPost && r.URL.Path == "/jobs/search" {
							reads.Add(1)
							var body struct {
								Filter map[string]any `json:"filter"`
							}
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
								t.Error(err)
								w.WriteHeader(400)
								return
							}
							if _, exists := body.Filter["tenantId"]; exists {
								t.Error("explicit execution inherited discovery tenant")
							}
							key, ok := body.Filter["jobKey"].(string)
							if !ok {
								t.Error("execution rediscovered instead of reading frozen key")
								w.WriteHeader(400)
								return
							}
							if key != "1" {
								t.Errorf("unexpected key %q", key)
							}
							if executing.Load() && mode == "uncertain missing" && writes.Load() > 0 {
								fmt.Fprint(w, `{"items":[],"page":{"totalItems":0}}`)
								return
							}
							state, kind := "CREATED", "BPMN_ELEMENT"
							if executing.Load() {
								if mode == "completed" || (mode == "conflict completed" && writes.Load() > 0) {
									state = "COMPLETED"
								}
								if mode == "canceled" {
									state = "CANCELED"
								}
								if mode == "listener changed" {
									kind = "TASK_LISTENER"
								}
							}
							fmt.Fprintf(w, `{"items":[{"jobKey":%q,"tenantId":"other","type":"work","state":%q,"kind":%q,"retries":0}],"page":{"totalItems":1}}`, key, state, kind)
							return
						}
						if r.Method == http.MethodPost && r.URL.Path == "/jobs/1/completion" {
							writes.Add(1)
							if !executing.Load() {
								t.Error("mutation before execution")
							}
							switch mode {
							case "conflict completed":
								w.WriteHeader(http.StatusConflict)
								fmt.Fprint(w, `{"title":"conflict","status":409}`)
							case "uncertain missing":
								w.WriteHeader(http.StatusBadGateway)
								fmt.Fprint(w, `{"title":"proxy lost response","status":502}`)
							default:
								w.WriteHeader(http.StatusNoContent)
							}
							return
						}
						unexpected.Add(1)
						t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
						w.WriteHeader(500)
					}))
					defer server.Close()
					cfg := testConfig()
					cfg.APIs.Camunda.BaseURL = server.URL
					cfg.App.CamundaVersion = version
					cfg.App.Tenant = "tenant"
					svc, err := job.New(cfg, server.Client(), nil)
					require.NoError(t, err)
					api := facade.New(svc, slog.New(slog.NewTextHandler(io.Discard, nil)))
					request := facade.CompletionRequest{Selection: facade.CompletionSelection{Mode: facade.CompletionSelectionExplicit, InputForm: facade.CompletionInputStdin, Keys: []string{"1"}}}
					if mode == "empty" {
						request.Selection.Keys = nil
					}
					if mode == "listener changed" {
						request.Variables = map[string]any{"work": true}
					}
					batch, prepareErr := api.PrepareJobCompletionBatch(context.Background(), request)
					if version != toolx.V87 {
						require.NoError(t, prepareErr)
					} else {
						require.ErrorIs(t, prepareErr, ferrors.ErrUnsupported)
					}
					before := reads.Load()
					executing.Store(true)
					opts := []foptions.FacadeOption{foptions.WithNoWait(), foptions.WithForce(), foptions.WithNoStateCheck(), foptions.WithIgnoreTenant()}
					if mode == "dry-run" {
						opts = append(opts, foptions.WithDryRun())
					}
					report, err := api.ExecuteJobCompletionBatch(context.Background(), batch, 2, opts...)
					require.Zero(t, unexpected.Load())
					if version == toolx.V87 {
						require.ErrorIs(t, err, ferrors.ErrUnsupported)
						require.Zero(t, reads.Load())
						require.Zero(t, writes.Load())
						return
					}
					require.Equal(t, "tenant", report.DiscoveryTenant)
					switch mode {
					case "dry-run":
						require.NoError(t, err)
						require.Equal(t, facade.CompletionBatchPreview, report.Status)
						require.Equal(t, before, reads.Load())
						require.Zero(t, writes.Load())
						require.Len(t, report.Candidates, 1)
						require.Nil(t, report.Outcomes)
					case "empty":
						require.NoError(t, err)
						require.Zero(t, writes.Load())
						require.Zero(t, reads.Load())
						require.Zero(t, report.Counts.Selected)
						require.Equal(t, facade.CompletionBatchFinished, report.Status)
					case "accepted":
						require.NoError(t, err)
						require.Equal(t, int32(1), writes.Load())
						require.Equal(t, before+1, reads.Load())
						require.Equal(t, 1, report.Counts.Accepted)
						require.Equal(t, facade.CompletionObservationSkipped, report.Outcomes[0].Observation.Status)
					case "conflict completed":
						require.NoError(t, err)
						require.Equal(t, int32(1), writes.Load())
						require.Equal(t, before+2, reads.Load())
						require.Equal(t, 1, report.Counts.Completed)
						require.Equal(t, facade.CompletionSubmissionRejected, report.Outcomes[0].Submission.Status)
						require.Equal(t, 409, *report.Outcomes[0].Submission.HTTPStatus)
					case "uncertain missing":
						require.Error(t, err)
						require.Equal(t, int32(1), writes.Load())
						require.Equal(t, before+2, reads.Load())
						require.Equal(t, 1, report.Counts.Unsatisfied)
						require.Equal(t, facade.CompletionSubmissionUncertain, report.Outcomes[0].Submission.Status)
						require.Equal(t, facade.CompletionObservationMissing, report.Outcomes[0].Observation.Status)
					case "completed":
						require.NoError(t, err)
						require.Zero(t, writes.Load())
						require.Equal(t, 1, report.Counts.Completed)
					default:
						require.Error(t, err)
						require.Zero(t, writes.Load())
						require.Equal(t, 1, report.Counts.Unsatisfied)
					}
				})
			}
		})
	}
}
