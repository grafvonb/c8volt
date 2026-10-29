// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	camunda "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	adapter "github.com/grafvonb/c8volt/internal/services/usertask/v810"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// assignmentClient permits only the explicitly selected native action; all other methods panic.
type assignmentClient struct {
	adapter.GenUserTaskClientCamunda
	action string
	submit func(context.Context, camunda.UserTaskKey, *camunda.AssignUserTaskJSONRequestBody) (*http.Response, []byte, error)
}

// AssignUserTaskWithResponse exposes exact generated arguments while forbidding implicit assignment in removal.
func (c assignmentClient) AssignUserTaskWithResponse(ctx context.Context, key camunda.UserTaskKey, body camunda.AssignUserTaskJSONRequestBody, _ ...camunda.RequestEditorFn) (*camunda.AssignUserTaskResponse, error) {
	if c.action != "assign" {
		panic("unexpected assignment")
	}
	response, raw, err := c.submit(ctx, key, &body)
	if response == nil {
		return nil, err
	}
	return &camunda.AssignUserTaskResponse{HTTPResponse: response, Body: raw}, err
}

// UnassignUserTaskWithResponse forbids implicit removal in assignment and passes raw evidence through.
func (c assignmentClient) UnassignUserTaskWithResponse(ctx context.Context, key camunda.UserTaskKey, _ ...camunda.RequestEditorFn) (*camunda.UnassignUserTaskResponse, error) {
	if c.action != "unassign" {
		panic("unexpected unassignment")
	}
	response, raw, err := c.submit(ctx, key, nil)
	if response == nil {
		return nil, err
	}
	return &camunda.UnassignUserTaskResponse{HTTPResponse: response, Body: raw}, err
}

// assignmentService isolates generated calls and makes any fallback transport fatal.
func assignmentService(t *testing.T, log *slog.Logger, client assignmentClient) *adapter.Service {
	t.Helper()
	if log == nil {
		log = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	cfg := testConfig()
	cfg.App.Tenant = "discovery-tenant-must-not-filter-explicit-keys"
	svc, err := adapter.New(cfg, &http.Client{Transport: completionForbiddenTransport{}}, log, adapter.WithClientCamunda(client))
	require.NoError(t, err)
	return svc
}

// runAssignment invokes only the selected operation, using one opaque identity for assignment.
func runAssignment(ctx context.Context, svc *adapter.Service, action string, opts ...services.CallOption) (d.UserTaskAssignmentResult, error) {
	if action == "assign" {
		return svc.AssignUserTask(ctx, "2251799813685249", "  opaque|用户\nidentity  ", opts...)
	}
	return svc.UnassignUserTask(ctx, "2251799813685249", opts...)
}

// TestAssignUserTaskWirePayload proves protected serialization, bodyless removal and backend-authorized explicit keys.
func TestAssignUserTaskWirePayload(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"assign", "unassign"} {
		for _, override := range []bool{false, true} {
			if action == "unassign" && override {
				continue
			}
			for _, code := range []int{204, 409} {
				t.Run(fmt.Sprintf("%s/override=%t/status=%d", action, override, code), func(t *testing.T) {
					var calls atomic.Int32
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						method, suffix := http.MethodPost, "assignment"
						if action == "unassign" {
							method, suffix = http.MethodDelete, "assignee"
						}
						require.Equal(t, method, r.Method)
						require.Equal(t, "/v2/user-tasks/2251799813685249/"+suffix, r.URL.Path)
						require.Empty(t, r.URL.RawQuery)
						raw, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						if action == "assign" {
							var body map[string]any
							require.NoError(t, json.Unmarshal(raw, &body))
							require.Equal(t, map[string]any{"assignee": "  opaque|用户\nidentity  ", "allowOverride": override, "action": nil}, body)
						} else {
							require.Empty(t, raw)
						}
						w.WriteHeader(code)
					}))
					defer server.Close()
					cfg := testConfig()
					cfg.APIs.Camunda.BaseURL = server.URL + "/v2"
					cfg.App.Tenant = "other-tenant"
					svc, err := adapter.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
					require.NoError(t, err)
					var opts []services.CallOption
					if override {
						opts = append(opts, services.WithUserTaskAssignmentOverride())
					}
					got, err := runAssignment(context.Background(), svc, action, opts...)
					if code == 204 {
						require.NoError(t, err)
						require.Equal(t, d.UserTaskAssignmentStatusAccepted, got.Status)
						require.Equal(t, d.UserTaskAssignmentSubmissionAccepted, got.Submission)
					} else {
						require.ErrorIs(t, err, d.ErrConflict)
						require.Equal(t, d.UserTaskAssignmentStatusFailed, got.Status)
						require.Equal(t, d.UserTaskAssignmentSubmissionRejected, got.Submission)
					}
					require.Equal(t, "2251799813685249", got.Key)
					require.Equal(t, code, got.StatusCode)
					require.EqualValues(t, 1, calls.Load())
				})
			}
		}
	}
}

// TestUnassignUserTaskWaitsForNativeResponse proves neither operation returns acceptance before its response arrives.
func TestUnassignUserTaskWaitsForNativeResponse(t *testing.T) {
	t.Parallel()
	for _, action := range []string{"assign", "unassign"} {
		t.Run(action, func(t *testing.T) {
			entered, release := make(chan struct{}), make(chan struct{})
			var calls atomic.Int32
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				suffix, method := "assignment", http.MethodPost
				if action == "unassign" {
					suffix, method = "assignee", http.MethodDelete
				}
				require.Equal(t, method, r.Method)
				require.Equal(t, "/v2/user-tasks/2251799813685249/"+suffix, r.URL.Path)
				close(entered)
				<-release
				w.WriteHeader(204)
			}))
			defer server.Close()
			defer close(release)
			cfg := testConfig()
			cfg.APIs.Camunda.BaseURL = server.URL + "/v2"
			svc, err := adapter.New(cfg, server.Client(), slog.New(slog.NewTextHandler(io.Discard, nil)))
			require.NoError(t, err)
			type outcome struct {
				result d.UserTaskAssignmentResult
				err    error
			}
			done := make(chan outcome, 1)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			go func() { got, err := runAssignment(ctx, svc, action); done <- outcome{got, err} }()
			select {
			case <-entered:
			case <-ctx.Done():
				t.Fatal("request did not arrive")
			}
			select {
			case <-done:
				t.Fatal("returned before native response")
			default:
			}
			release <- struct{}{}
			select {
			case got := <-done:
				require.NoError(t, got.err)
				require.Equal(t, d.UserTaskAssignmentStatusAccepted, got.result.Status)
			case <-ctx.Done():
				t.Fatal("response was not consumed")
			}
			require.EqualValues(t, 1, calls.Load())
		})
	}
}

// TestAssignUserTaskDelegation preserves generated response/body/error mapping for both actions.
func TestAssignUserTaskDelegation(t *testing.T) {
	t.Parallel()
	transport := errors.New("adapter transport failure")
	for _, action := range []string{"assign", "unassign"} {
		for _, tc := range []struct {
			name           string
			code           int
			callErr, cause error
			submission     d.UserTaskAssignmentSubmission
		}{
			{"rejected", 409, nil, d.ErrConflict, "rejected"},
			{"missing response", 0, nil, d.ErrMalformedResponse, "unknown"},
			{"partial 204", 204, transport, transport, "unknown"},
		} {
			t.Run(action+"/"+tc.name, func(t *testing.T) {
				calls := 0
				svc := assignmentService(t, nil, assignmentClient{action: action, submit: func(_ context.Context, key camunda.UserTaskKey, body *camunda.AssignUserTaskJSONRequestBody) (*http.Response, []byte, error) {
					calls++
					require.Equal(t, camunda.UserTaskKey("2251799813685249"), key)
					if action == "assign" {
						require.NotNil(t, body)
						require.NotNil(t, body.Assignee)
						require.NotNil(t, body.AllowOverride)
						require.False(t, *body.AllowOverride)
						require.Nil(t, body.Action)
					} else {
						require.Nil(t, body)
					}
					if tc.code == 0 {
						return nil, nil, tc.callErr
					}
					response := completionResponse(tc.code)
					return response.HTTPResponse, response.Body, tc.callErr
				}})
				got, err := runAssignment(context.Background(), svc, action)
				require.ErrorIs(t, err, tc.cause)
				require.Equal(t, 1, calls)
				require.Equal(t, d.UserTaskAssignmentStatusFailed, got.Status)
				require.Equal(t, tc.submission, got.Submission)
				require.Equal(t, tc.code, got.StatusCode)
				require.Equal(t, err.Error(), got.Error)
				if tc.code == 409 {
					require.Contains(t, got.Error, "DISTINCTIVE_BODY")
				}
			})
		}
	}
}

// TestAssignUserTaskRetrySnapshots ensures options are evaluated once and request pointers cannot alter retry intent.
func TestAssignUserTaskRetrySnapshots(t *testing.T) {
	t.Parallel()
	calls, evaluations := 0, 0
	svc := assignmentService(t, nil, assignmentClient{action: "assign", submit: func(_ context.Context, _ camunda.UserTaskKey, body *camunda.AssignUserTaskJSONRequestBody) (*http.Response, []byte, error) {
		calls++
		require.Equal(t, "  opaque|用户\nidentity  ", *body.Assignee)
		require.True(t, *body.AllowOverride)
		*body.Assignee = "changed"
		*body.AllowOverride = false
		code := 429
		if calls == 2 {
			code = 204
		}
		response := completionResponse(code)
		return response.HTTPResponse, response.Body, nil
	}})
	got, err := runAssignment(context.Background(), svc, "assign", func(c *services.CallCfg) { evaluations++; c.UserTaskAssignmentOverride = true })
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, 1, evaluations)
	require.Equal(t, d.UserTaskAssignmentStatusAccepted, got.Status)
}

// TestAssignUserTaskRetryLogging preserves configured safe progress without logging raw per-key failures.
func TestAssignUserTaskRetryLogging(t *testing.T) {
	format := os.Getenv("C8VOLT_UNIT_ASSIGNMENT_LOG_FORMAT")
	if format == "" {
		for _, format := range []string{"plain", "structured"} {
			t.Run(format, func(t *testing.T) {
				// Retry progress is globally throttled; a fresh child guarantees an observable record for each format.
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAssignUserTaskRetryLogging$", "-test.count=1")
				command.Env = append(os.Environ(), "C8VOLT_UNIT_ASSIGNMENT_LOG_FORMAT="+format)
				output, err := command.CombinedOutput()
				require.NoError(t, err, string(output))
			})
		}
		return
	}
	for _, action := range []string{"assign", "unassign"} {
		for _, mode := range []string{"compact", "verbose", "suppressed"} {
			t.Run(action+"/"+mode, func(t *testing.T) {
				var logs bytes.Buffer
				var handler slog.Handler = slog.NewTextHandler(&logs, nil)
				if format == "structured" {
					handler = slog.NewJSONHandler(&logs, nil)
				}
				calls := 0
				svc := assignmentService(t, slog.New(handler), assignmentClient{action: action, submit: func(context.Context, camunda.UserTaskKey, *camunda.AssignUserTaskJSONRequestBody) (*http.Response, []byte, error) {
					calls++
					response := completionResponse(429)
					return response.HTTPResponse, response.Body, nil
				}})
				var opts []services.CallOption
				if mode != "compact" {
					opts = append(opts, services.WithVerbose())
				}
				if mode == "suppressed" {
					opts = append(opts, services.WithSuppressWorkflowDetailLogs())
				}
				_, err := runAssignment(context.Background(), svc, action, opts...)
				require.ErrorIs(t, err, d.ErrRateLimited)
				require.Equal(t, 8, calls)
				raw := logs.String()
				if mode != "verbose" {
					require.Empty(t, raw)
					return
				}
				require.Equal(t, 1, bytes.Count(logs.Bytes(), []byte("\n")))
				for _, secret := range []string{"POST", "distinctive.invalid", "DISTINCTIVE_BODY", "2251799813685249", "RESOURCE_EXHAUSTED"} {
					require.NotContains(t, raw, secret)
				}
				if format == "structured" {
					var record map[string]any
					decoder := json.NewDecoder(bytes.NewReader(logs.Bytes()))
					require.NoError(t, decoder.Decode(&record))
					require.Equal(t, "INFO", record["level"])
					require.Contains(t, record["msg"], "Camunda throttled "+action+" user task; rate limited; retrying in")
					require.ErrorIs(t, decoder.Decode(&record), io.EOF)
				} else {
					require.Contains(t, raw, `level=INFO msg="Camunda throttled `+action+` user task; rate limited; retrying in`)
				}
			})
		}
	}
}
