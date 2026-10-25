// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestProcessInstanceSuspensionWorkflow exercises real generated serialization,
// post-acceptance observation and status preservation without any network access.
func TestProcessInstanceSuspensionWorkflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		submit           int
		reads            []string
		readStatus       []int
		opts             []services.CallOption
		wantStatus       d.ProcessInstanceSuspensionStatus
		wantSubmission   d.ProcessInstanceSuspensionSubmission
		wantConfirmation d.ProcessInstanceSuspensionConfirmation
		wantState        d.State
		wantDate         string
		wantErr          error
	}{
		{"confirmed", 204, []string{"ACTIVE", "SUSPENDED"}, nil, nil, d.ProcessInstanceSuspensionStatusConfirmed, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationConfirmed, d.StateSuspended, "2026-10-25T12:00:00Z", nil},
		{"no wait", 204, nil, nil, []services.CallOption{services.WithNoWait()}, d.ProcessInstanceSuspensionStatusAccepted, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationNotRequested, "", "", nil},
		{"preview", 0, nil, nil, []services.CallOption{services.WithDryRun()}, d.ProcessInstanceSuspensionStatusPreview, d.ProcessInstanceSuspensionSubmissionNotSubmitted, d.ProcessInstanceSuspensionConfirmationNotRequested, "", "", nil},
		{"preview no wait", 0, nil, nil, []services.CallOption{services.WithDryRun(), services.WithNoWait()}, d.ProcessInstanceSuspensionStatusPreview, d.ProcessInstanceSuspensionSubmissionNotSubmitted, d.ProcessInstanceSuspensionConfirmationNotRequested, "", "", nil},
		{"rejected", 409, nil, nil, nil, d.ProcessInstanceSuspensionStatusFailed, d.ProcessInstanceSuspensionSubmissionRejected, d.ProcessInstanceSuspensionConfirmationNotRequested, "", "", d.ErrConflict},
		{"unknown", 200, nil, nil, nil, d.ProcessInstanceSuspensionStatusFailed, d.ProcessInstanceSuspensionSubmissionUnknown, d.ProcessInstanceSuspensionConfirmationNotRequested, "", "", d.ErrMalformedResponse},
		{"completed", 204, []string{"COMPLETED"}, nil, nil, d.ProcessInstanceSuspensionStatusFailed, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationUnconfirmed, d.StateCompleted, "2026-10-25T12:00:00Z", d.ErrConflict},
		{"terminated", 204, []string{"TERMINATED"}, nil, nil, d.ProcessInstanceSuspensionStatusFailed, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationUnconfirmed, d.StateTerminated, "2026-10-25T12:00:00Z", d.ErrConflict},
		{"denied", 204, []string{""}, []int{403}, nil, d.ProcessInstanceSuspensionStatusFailed, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationUnconfirmed, "", "", d.ErrForbidden},
		{"preserved", 204, []string{"ACTIVE", ""}, []int{200, 403}, nil, d.ProcessInstanceSuspensionStatusFailed, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationUnconfirmed, d.StateActive, "2026-10-25T12:00:00Z", d.ErrForbidden},
		{"absent timeout", 204, []string{"ACTIVE", "", ""}, []int{200, 404, 404}, nil, d.ProcessInstanceSuspensionStatusTimedOut, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationUnconfirmed, d.StateAbsent, "", d.ErrGatewayTimeout},
		{"unknown timeout", 204, []string{"UNKNOWN", "UNKNOWN", "UNKNOWN"}, nil, nil, d.ProcessInstanceSuspensionStatusTimedOut, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationUnconfirmed, d.StateUnknown, "2026-10-25T12:00:00Z", d.ErrGatewayTimeout},
		{"wrong key", 204, []string{"WRONG_KEY"}, nil, nil, d.ProcessInstanceSuspensionStatusFailed, d.ProcessInstanceSuspensionSubmissionAccepted, d.ProcessInstanceSuspensionConfirmationUnconfirmed, "", "", d.ErrMalformedResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				submissions, reads := 0, 0
				var logs bytes.Buffer
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(&http.Client{Transport: suspensionRoundTripper(func(req *http.Request) (*http.Response, error) {
					code := tt.submit
					body := ""
					switch req.Method {
					case http.MethodPost:
						submissions++
						require.Equal(t, "/v2/process-instances/"+suspensionTestKey+"/suspension", req.URL.Path)
						bytes, err := io.ReadAll(req.Body)
						require.NoError(t, err)
						require.JSONEq(t, `{}`, string(bytes))
					case http.MethodGet:
						require.Equal(t, 1, submissions)
						require.Equal(t, "/v2/process-instances/"+suspensionTestKey, req.URL.Path)
						require.Less(t, reads, len(tt.reads))
						state := tt.reads[reads]
						code = 200
						if len(tt.readStatus) > 0 {
							code = tt.readStatus[reads]
						}
						key := suspensionTestKey
						if state == "WRONG_KEY" {
							key = "different"
							state = "SUSPENDED"
						}
						body = fmt.Sprintf(`{"processInstanceKey":%q,"state":%q,"suspendedDate":"2026-10-25T12:00:00Z"}`, key, state)
						reads++
					default:
						t.Errorf("unexpected method %s", req.Method)
					}
					if code >= 400 {
						body = `{"detail":"POST https://fixture.invalid SECRET_BODY not found"}`
					}
					return &http.Response{StatusCode: code, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
				})}))
				require.NoError(t, err)
				cfg := &config.Config{App: config.App{Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: 10 * time.Second, MaxRetries: 3}}}
				service := &Service{cc: client, cfg: cfg, log: slog.New(slog.NewTextHandler(&logs, nil))}
				result, err := service.SuspendProcessInstance(context.Background(), " "+suspensionTestKey+" ", append(tt.opts, services.WithVerbose())...)
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.wantStatus, result.Status)
				require.Equal(t, tt.wantSubmission, result.Submission)
				require.Equal(t, tt.wantConfirmation, result.Confirmation)
				require.Equal(t, suspensionTestKey, result.Key)
				require.Equal(t, tt.wantState, result.LastObservedState)
				require.Equal(t, tt.wantDate, result.SuspendedDate)
				require.Equal(t, tt.wantState != "", result.HasObservation)
				require.Equal(t, tt.submit, result.StatusCode)
				require.Equal(t, len(tt.reads), reads)
				expectedSubmissions := 1
				if tt.submit == 0 {
					expectedSubmissions = 0
				}
				require.Equal(t, expectedSubmissions, submissions)
				require.Empty(t, logs.String(), "normal and verbose service execution must not emit raw or nested diagnostics")
			})
		})
	}
}

// TestProcessInstanceSuspensionLocalValidation proves malformed input, canceled
// contexts and incomplete timing cannot reach a generated client.
func TestProcessInstanceSuspensionLocalValidation(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"", "123", "123456789012345x", "１２３４５６７８９０１２３４５６"} {
		got, err := (&Service{}).SuspendProcessInstance(context.Background(), key, services.WithNoWait())
		require.ErrorIs(t, err, d.ErrValidation)
		require.Equal(t, key, got.Key)
		require.Equal(t, d.ProcessInstanceSuspensionSubmissionNotSubmitted, got.Submission)
	}
	for _, cfg := range []*config.Config{nil, {}} {
		_, err := (&Service{cfg: cfg}).SuspendProcessInstance(context.Background(), suspensionTestKey)
		require.ErrorIs(t, err, d.ErrValidation)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := (&Service{}).SuspendProcessInstance(ctx, suspensionTestKey)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, d.ProcessInstanceSuspensionStatusInterrupted, got.Status)
	// NoWait previews do not need confirmation timing and never claim acceptance.
	got, err = (&Service{}).SuspendProcessInstance(context.Background(), suspensionTestKey, services.WithDryRun(), services.WithNoWait())
	require.NoError(t, err)
	require.Equal(t, d.ProcessInstanceSuspensionStatusPreview, got.Status)
}

// TestProcessInstanceSuspensionWorkflowDeadline retains 204 and the prior state
// when the confirmation deadline interrupts an in-flight generated read.
func TestProcessInstanceSuspensionWorkflowDeadline(t *testing.T) {
	t.Parallel()
	for _, cancelCaller := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelCaller), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				reads := 0
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(&http.Client{Transport: suspensionRoundTripper(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodPost {
						return &http.Response{StatusCode: 204, Body: http.NoBody, Request: req}, nil
					}
					reads++
					if reads == 1 {
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"processInstanceKey":"` + suspensionTestKey + `","state":"ACTIVE","suspendedDate":"2026-10-25T11:00:00Z"}`)), Request: req}, nil
					}
					if cancelCaller {
						cancel()
					}
					<-req.Context().Done()
					return nil, req.Context().Err()
				})}))
				require.NoError(t, err)
				cfg := &config.Config{App: config.App{Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: 3 * time.Second}}}
				result, err := (&Service{cc: client, cfg: cfg}).SuspendProcessInstance(ctx, suspensionTestKey)
				wantErr := context.DeadlineExceeded
				wantStatus := d.ProcessInstanceSuspensionStatusTimedOut
				if cancelCaller {
					wantErr = context.Canceled
					wantStatus = d.ProcessInstanceSuspensionStatusInterrupted
				}
				require.ErrorIs(t, err, wantErr)
				require.Equal(t, wantStatus, result.Status)
				require.Equal(t, 204, result.StatusCode)
				require.Equal(t, d.ProcessInstanceSuspensionSubmissionAccepted, result.Submission)
				require.Equal(t, d.ProcessInstanceSuspensionConfirmationUnconfirmed, result.Confirmation)
				require.Equal(t, d.StateActive, result.LastObservedState)
				require.Equal(t, "2026-10-25T11:00:00Z", result.SuspendedDate)
				require.Equal(t, 2, reads)
			})
		})
	}
}

// TestProcessInstanceSuspensionReadRetries bounds transport retries by confirmation
// time and suppresses only functional chatter, preserving redacted DEBUG exchanges.
func TestProcessInstanceSuspensionReadRetries(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"plain", "json"} {
		for _, level := range []string{"info", "debug"} {
			t.Run(format+"/"+level, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					var output bytes.Buffer
					log := logging.New(logging.LoggerConfig{Writer: &output, Level: level, Format: format})
					cfg := &config.Config{HTTP: config.HTTP{Timeout: "30s", RequestRateReportInterval: "0s"}, App: config.App{Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: 2 * time.Second}}}
					reads, submissions := 0, 0
					shared, err := httpc.New(cfg, log, func(s *httpc.Service) {
						s.Client().Transport = suspensionRoundTripper(func(req *http.Request) (*http.Response, error) {
							code := 204
							body := ""
							header := http.Header{}
							if req.Method == http.MethodPost {
								submissions++
							} else {
								reads++
								code = 503
								body = `{"detail":"SECRET_BODY unavailable"}`
								header.Set("Retry-After", "60")
							}
							return &http.Response{StatusCode: code, Header: header, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
						})
					}, httpc.WithDiagnostics())
					require.NoError(t, err)
					defer shared.Close()
					client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(shared.Client()), camundav810.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
						req.Header.Set("Authorization", "Bearer SECRET_TOKEN")
						return nil
					}))
					require.NoError(t, err)
					start := time.Now()
					result, err := (&Service{cc: client, cfg: cfg, log: log}).SuspendProcessInstance(shared.ToContext(context.Background()), suspensionTestKey, services.WithVerbose())
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.Equal(t, 2*time.Second, time.Since(start))
					require.Equal(t, 1, submissions)
					require.Equal(t, 1, reads)
					require.Equal(t, 204, result.StatusCode)
					require.Equal(t, d.ProcessInstanceSuspensionStatusTimedOut, result.Status)
					require.Equal(t, d.ProcessInstanceSuspensionSubmissionAccepted, result.Submission)
					require.False(t, result.HasObservation)
					require.NotContains(t, output.String(), "SECRET_TOKEN")
					require.NotContains(t, output.String(), "retrying")
					require.NotContains(t, output.String(), "INFO")
					require.NotContains(t, output.String(), "ERROR")
					if level == "info" {
						require.Empty(t, output.String())
					} else {
						require.Equal(t, 2, strings.Count(output.String(), "api #"))
						require.Contains(t, output.String(), "GET /v2/process-instances/"+suspensionTestKey)
					}
				})
			})
		}
	}
}

// suspensionNilReadClient verifies a malformed generated-client response cannot panic.
type suspensionNilReadClient struct{ suspensionSubmissionClient }

// GetProcessInstanceWithResponse simulates missing generated response metadata.
func (s suspensionNilReadClient) GetProcessInstanceWithResponse(context.Context, camundav810.ProcessInstanceKey, ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
	return nil, nil
}

// TestProcessInstanceSuspensionNilRead retains acceptance on a malformed read.
func TestProcessInstanceSuspensionNilRead(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{App: config.App{Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: time.Second}}}
	client := suspensionNilReadClient{suspensionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error) {
		return suspensionResponse(204, ""), nil
	}}}
	result, err := (&Service{cc: client, cfg: cfg}).SuspendProcessInstance(context.Background(), suspensionTestKey)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Equal(t, 204, result.StatusCode)
	require.Equal(t, d.ProcessInstanceSuspensionSubmissionAccepted, result.Submission)
	require.False(t, result.HasObservation)
}
