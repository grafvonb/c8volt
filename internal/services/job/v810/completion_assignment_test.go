// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestCompletionAssignmentWire covers both public service routes with exact atomic payloads and one-shot failures.
func TestCompletionAssignmentWire(t *testing.T) {
	for _, bulk := range []bool{false, true} {
		for _, scenario := range []string{"accepted", "identical conflict", "child conflict", "uniqueness conflict", "lost", "malformed", "already completed"} {
			t.Run(map[bool]string{false: "scalar", true: "bulk"}[bulk]+"/"+scenario, func(t *testing.T) {
				var posts atomic.Int32
				value := " Order,😀42 "
				server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					require.True(t, r.URL.Path == "/jobs/search" || r.URL.Path == "/jobs/123/completion")
					if r.URL.Path == "/jobs/search" {
						require.Equal(t, http.MethodPost, r.Method)
						var query map[string]any
						require.NoError(t, json.NewDecoder(r.Body).Decode(&query))
						require.Equal(t, "123", query["filter"].(map[string]any)["jobKey"])
						state := "CREATED"
						if scenario == "already completed" || posts.Load() > 0 {
							state = "COMPLETED"
						}
						w.Header().Set("Content-Type", "application/json")
						io.WriteString(w, `{"items":[{"jobKey":"123","state":"`+state+`","kind":"BPMN_ELEMENT","processInstanceKey":"456"}]}`)
						return
					}
					require.Equal(t, http.MethodPost, r.Method)
					require.Equal(t, "/jobs/123/completion", r.URL.Path)
					posts.Add(1)
					var body map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
					require.Equal(t, value, body["businessId"])
					require.Equal(t, map[string]any{"nested": map[string]any{"approved": true}}, body["variables"])
					require.Len(t, body, 2)
					switch scenario {
					case "lost":
						conn, _, err := w.(http.Hijacker).Hijack()
						require.NoError(t, err)
						conn.Close()
					case "malformed":
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(409)
						io.WriteString(w, `{`)
					case "accepted":
						w.WriteHeader(204)
					default:
						w.Header().Set("Content-Type", "application/problem+json")
						w.WriteHeader(409)
						io.WriteString(w, `{"title":"Conflict","detail":"`+scenario+`"}`)
					}
				}))
				t.Cleanup(server.Close)
				cfg := testx.TestConfig(t)
				cfg.APIs.Camunda.BaseURL = server.URL
				svc, err := New(cfg, server.Client(), nil)
				require.NoError(t, err)
				vars := map[string]any{"nested": map[string]any{"approved": true}}
				var assignment *d.CompletionAssignment
				var observation *d.CompletionObservation
				if bulk {
					batch, e := svc.PrepareJobCompletionBatch(context.Background(), d.CompletionRequest{SetBusinessID: &value, Variables: vars, Selection: d.CompletionSelection{Mode: d.CompletionSelectionExplicit, InputForm: d.CompletionInputCollection, Keys: []string{"123", "123"}}})
					require.NoError(t, e)
					require.Len(t, batch.Candidates, 1)
					report, e := svc.ExecuteJobCompletionBatch(context.Background(), batch, 1, services.WithNoWait())
					err = e
					require.Len(t, report.Outcomes, 1)
					assignment = report.Outcomes[0].Assignment
					observation = &report.Outcomes[0].Observation
					require.Nil(t, report.Outcomes[0].Candidate.BusinessId)
				} else {
					result, e := svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{Key: "123", Mode: d.JobWorkerOutcomeCompletion, Variables: vars, SetBusinessID: &value, SkipConfirmation: true})
					err = e
					assignment = result.Assignment
					observation = result.Observation
					require.Equal(t, value, *result.RequestedBusinessID)
				}
				require.NotNil(t, assignment)
				switch scenario {
				case "accepted":
					require.NoError(t, err)
					require.Equal(t, "accepted", assignment.Status)
				case "lost", "malformed":
					require.Error(t, err)
					require.Equal(t, "unknown", assignment.Status)
					require.Equal(t, d.CompletionObservationCompleted, observation.Status)
				case "already completed":
					require.ErrorIs(t, err, d.ErrPrecondition)
					require.Equal(t, "not_submitted", assignment.Status)
				default:
					require.ErrorIs(t, err, d.ErrConflict)
					require.Equal(t, "rejected", assignment.Status)
					require.Equal(t, d.CompletionObservationCompleted, observation.Status)
				}
				if scenario == "already completed" {
					require.Zero(t, posts.Load())
				} else {
					require.Equal(t, int32(1), posts.Load())
				}
			})
		}
	}
}

// TestCompletionAssignmentValidation rejects invalid intent at every entry before backend access.
func TestCompletionAssignmentValidation(t *testing.T) {
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("invalid intent reached backend")
		w.WriteHeader(500)
	}))
	t.Cleanup(server.Close)
	cfg := testx.TestConfig(t)
	cfg.APIs.Camunda.BaseURL = server.URL
	svc, err := New(cfg, server.Client(), nil)
	require.NoError(t, err)
	for _, value := range []string{"", " \t", string([]byte{0xff}), strings.Repeat("a", 257), strings.Repeat("😀", 129)} {
		t.Run(value, func(t *testing.T) {
			_, err := svc.PrepareJobCompletionBatch(context.Background(), d.CompletionRequest{SetBusinessID: &value})
			require.ErrorIs(t, err, d.ErrValidation)
			_, err = svc.ExecuteJobCompletionBatch(context.Background(), d.PreparedCompletionBatch{SetBusinessID: &value}, 1)
			require.ErrorIs(t, err, d.ErrValidation)
			_, err = svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{Key: "123", Mode: d.JobWorkerOutcomeCompletion, SetBusinessID: &value})
			require.ErrorIs(t, err, d.ErrValidation)
			_, err = svc.submitBulkCompletion(context.Background(), d.Job{Key: "123"}, d.CompletionPayload{SetBusinessID: &value})
			require.ErrorIs(t, err, d.ErrValidation)
		})
	}
	_, err = svc.submitBulkCompletion(context.Background(), d.Job{Key: "123", Kind: d.JobKindTaskListener}, d.CompletionPayload{SetBusinessID: toolx.Ptr("order"), Variables: map[string]any{"approved": true}})
	require.ErrorIs(t, err, d.ErrValidation)
	for _, mode := range []d.JobWorkerOutcomeMode{d.JobWorkerOutcomeTechnicalFailure, d.JobWorkerOutcomeBPMNError, "update", "timeout"} {
		_, err := svc.SubmitJobWorkerOutcome(context.Background(), d.JobWorkerOutcomeRequest{Key: "123", Mode: mode, SetBusinessID: toolx.Ptr("order")})
		require.ErrorIs(t, err, d.ErrValidation)
	}
}

// TestCompletionAssignmentValues keeps accepted UTF-16 boundary values byte-exact on the native wire.
func TestCompletionAssignmentValues(t *testing.T) {
	for _, value := range []string{strings.Repeat("a", 256), strings.Repeat("😀", 128), "  Mixed,001  "} {
		t.Run(value, func(t *testing.T) {
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var body map[string]any
				require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
				require.Equal(t, value, body["businessId"])
				w.WriteHeader(204)
			}))
			t.Cleanup(server.Close)
			cfg := testx.TestConfig(t)
			cfg.APIs.Camunda.BaseURL = server.URL
			svc, err := New(cfg, server.Client(), nil)
			require.NoError(t, err)
			evidence, err := svc.submitBulkCompletion(context.Background(), d.Job{Key: "123"}, d.CompletionPayload{SetBusinessID: toolx.Ptr(value)})
			require.NoError(t, err)
			require.Equal(t, d.CompletionSubmissionAccepted, evidence.Status)
		})
	}
}
