// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"bytes"
	"context"
	"encoding/json"
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

// TestProcessInstanceResumptionWorkflow exercises real generated serialization,
// post-acceptance observation and status preservation without any network access.
func TestProcessInstanceResumptionWorkflow(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		submit           int
		reads            []string
		readStatus       []int
		opts             []services.CallOption
		wantStatus       d.ProcessInstanceResumptionStatus
		wantSubmission   d.ProcessInstanceResumptionSubmission
		wantConfirmation d.ProcessInstanceResumptionConfirmation
		wantState        d.State
		wantDate         string
		wantErr          error
	}{
		{"confirmed", 204, []string{"SUSPENDED", "ACTIVE"}, nil, nil, d.ProcessInstanceResumptionStatusConfirmed, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationConfirmed, d.StateActive, "2026-10-25T12:00:00Z", nil},
		{"no wait", 204, nil, nil, []services.CallOption{services.WithNoWait()}, d.ProcessInstanceResumptionStatusAccepted, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationNotRequested, "", "", nil},
		{"preview", 0, nil, nil, []services.CallOption{services.WithDryRun()}, d.ProcessInstanceResumptionStatusPreview, d.ProcessInstanceResumptionSubmissionNotSubmitted, d.ProcessInstanceResumptionConfirmationNotRequested, "", "", nil},
		{"preview no wait", 0, nil, nil, []services.CallOption{services.WithDryRun(), services.WithNoWait()}, d.ProcessInstanceResumptionStatusPreview, d.ProcessInstanceResumptionSubmissionNotSubmitted, d.ProcessInstanceResumptionConfirmationNotRequested, "", "", nil},
		{"rejected", 409, nil, nil, nil, d.ProcessInstanceResumptionStatusFailed, d.ProcessInstanceResumptionSubmissionRejected, d.ProcessInstanceResumptionConfirmationNotRequested, "", "", d.ErrConflict},
		{"unknown", 200, nil, nil, nil, d.ProcessInstanceResumptionStatusFailed, d.ProcessInstanceResumptionSubmissionUnknown, d.ProcessInstanceResumptionConfirmationNotRequested, "", "", d.ErrMalformedResponse},
		{"completed", 204, []string{"COMPLETED"}, nil, nil, d.ProcessInstanceResumptionStatusConfirmed, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationConfirmed, d.StateCompleted, "2026-10-25T12:00:00Z", nil},
		{"terminated", 204, []string{"TERMINATED"}, nil, nil, d.ProcessInstanceResumptionStatusFailed, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationUnconfirmed, d.StateTerminated, "2026-10-25T12:00:00Z", d.ErrConflict},
		{"canceled", 204, []string{"CANCELED"}, nil, nil, d.ProcessInstanceResumptionStatusFailed, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationUnconfirmed, d.StateCanceled, "2026-10-25T12:00:00Z", d.ErrConflict},
		{"denied", 204, []string{""}, []int{403}, nil, d.ProcessInstanceResumptionStatusFailed, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationUnconfirmed, "", "", d.ErrForbidden},
		{"preserved", 204, []string{"SUSPENDED", ""}, []int{200, 403}, nil, d.ProcessInstanceResumptionStatusFailed, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationUnconfirmed, d.StateSuspended, "2026-10-25T12:00:00Z", d.ErrForbidden},
		{"absent timeout", 204, []string{"SUSPENDED", "", ""}, []int{200, 404, 404}, nil, d.ProcessInstanceResumptionStatusTimedOut, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationUnconfirmed, d.StateAbsent, "", d.ErrGatewayTimeout},
		{"unknown timeout", 204, []string{"UNKNOWN", "UNKNOWN", "UNKNOWN"}, nil, nil, d.ProcessInstanceResumptionStatusTimedOut, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationUnconfirmed, d.StateUnknown, "2026-10-25T12:00:00Z", d.ErrGatewayTimeout},
		{"wrong key", 204, []string{"WRONG_KEY"}, nil, nil, d.ProcessInstanceResumptionStatusFailed, d.ProcessInstanceResumptionSubmissionAccepted, d.ProcessInstanceResumptionConfirmationUnconfirmed, "", "", d.ErrMalformedResponse},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				submissions, reads := 0, 0
				var logs bytes.Buffer
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(&http.Client{Transport: resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
					code := tt.submit
					body := ""
					switch req.Method {
					case http.MethodPost:
						submissions++
						require.Equal(t, "/v2/process-instances/"+resumptionTestKey+"/resumption", req.URL.Path)
						bytes, err := io.ReadAll(req.Body)
						require.NoError(t, err)
						require.JSONEq(t, `{}`, string(bytes))
					case http.MethodGet:
						require.Equal(t, 1, submissions)
						require.Equal(t, "/v2/process-instances/"+resumptionTestKey, req.URL.Path)
						require.Less(t, reads, len(tt.reads))
						state := tt.reads[reads]
						code = 200
						if len(tt.readStatus) > 0 {
							code = tt.readStatus[reads]
						}
						key := resumptionTestKey
						if state == "WRONG_KEY" {
							key = "different"
							state = "ACTIVE"
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
				result, err := service.ResumeProcessInstance(context.Background(), " "+resumptionTestKey+" ", append(tt.opts, services.WithVerbose())...)
				require.ErrorIs(t, err, tt.wantErr)
				require.Equal(t, tt.wantStatus, result.Status)
				require.Equal(t, tt.wantSubmission, result.Submission)
				require.Equal(t, tt.wantConfirmation, result.Confirmation)
				require.Equal(t, resumptionTestKey, result.Key)
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

// TestProcessInstanceResumptionLocalValidation proves malformed input, canceled
// contexts and incomplete timing cannot reach a generated client.
func TestProcessInstanceResumptionLocalValidation(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"", "123", "123456789012345x", "１２３４５６７８９０１２３４５６"} {
		got, err := (&Service{}).ResumeProcessInstance(context.Background(), key, services.WithNoWait())
		require.ErrorIs(t, err, d.ErrValidation)
		require.Equal(t, key, got.Key)
		require.Equal(t, d.ProcessInstanceResumptionSubmissionNotSubmitted, got.Submission)
	}
	for _, cfg := range []*config.Config{nil, {}} {
		for _, opts := range [][]services.CallOption{nil, {services.WithDryRun()}} {
			got, err := (&Service{cfg: cfg}).ResumeProcessInstance(context.Background(), resumptionTestKey, opts...)
			require.ErrorIs(t, err, d.ErrValidation)
			require.Equal(t, d.ProcessInstanceResumptionSubmissionNotSubmitted, got.Submission)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := (&Service{}).ResumeProcessInstance(ctx, resumptionTestKey)
	require.ErrorIs(t, err, context.Canceled)
	require.Equal(t, d.ProcessInstanceResumptionStatusInterrupted, got.Status)
	// NoWait previews do not need confirmation timing and never claim acceptance.
	got, err = (&Service{}).ResumeProcessInstance(context.Background(), resumptionTestKey, services.WithDryRun(), services.WithNoWait())
	require.NoError(t, err)
	require.Equal(t, d.ProcessInstanceResumptionStatusPreview, got.Status)
}

// TestProcessInstanceResumptionWorkflowDeadline retains 204 and the prior state
// when the confirmation deadline interrupts an in-flight generated read.
func TestProcessInstanceResumptionWorkflowDeadline(t *testing.T) {
	t.Parallel()
	for _, cancelCaller := range []bool{false, true} {
		t.Run(fmt.Sprint(cancelCaller), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				reads := 0
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(&http.Client{Transport: resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodPost {
						return &http.Response{StatusCode: 204, Body: http.NoBody, Request: req}, nil
					}
					reads++
					if reads == 1 {
						return &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"processInstanceKey":"` + resumptionTestKey + `","state":"SUSPENDED","suspendedDate":"2026-10-25T11:00:00Z"}`)), Request: req}, nil
					}
					if cancelCaller {
						cancel()
					}
					<-req.Context().Done()
					return nil, req.Context().Err()
				})}))
				require.NoError(t, err)
				cfg := &config.Config{App: config.App{Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: 3 * time.Second}}}
				result, err := (&Service{cc: client, cfg: cfg}).ResumeProcessInstance(ctx, resumptionTestKey)
				wantErr := context.DeadlineExceeded
				wantStatus := d.ProcessInstanceResumptionStatusTimedOut
				if cancelCaller {
					wantErr = context.Canceled
					wantStatus = d.ProcessInstanceResumptionStatusInterrupted
				}
				require.ErrorIs(t, err, wantErr)
				require.Equal(t, wantStatus, result.Status)
				require.Equal(t, 204, result.StatusCode)
				require.Equal(t, d.ProcessInstanceResumptionSubmissionAccepted, result.Submission)
				require.Equal(t, d.ProcessInstanceResumptionConfirmationUnconfirmed, result.Confirmation)
				require.Equal(t, d.StateSuspended, result.LastObservedState)
				require.Equal(t, "2026-10-25T11:00:00Z", result.SuspendedDate)
				require.Equal(t, 2, reads)
			})
		})
	}
}

// TestProcessInstanceResumptionReadRetries bounds transport retries by confirmation
// time and suppresses only functional chatter, preserving redacted DEBUG exchanges.
func TestProcessInstanceResumptionReadRetries(t *testing.T) {
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
						s.Client().Transport = resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
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
					result, err := (&Service{cc: client, cfg: cfg, log: log}).ResumeProcessInstance(shared.ToContext(context.Background()), resumptionTestKey, services.WithVerbose())
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.Equal(t, 2*time.Second, time.Since(start))
					require.Equal(t, 1, submissions)
					require.Equal(t, 1, reads)
					require.Equal(t, 204, result.StatusCode)
					require.Equal(t, d.ProcessInstanceResumptionStatusTimedOut, result.Status)
					require.Equal(t, d.ProcessInstanceResumptionSubmissionAccepted, result.Submission)
					require.False(t, result.HasObservation)
					require.NotContains(t, output.String(), "SECRET_TOKEN")
					require.NotContains(t, output.String(), "retrying")
					require.NotContains(t, output.String(), "INFO")
					require.NotContains(t, output.String(), "ERROR")
					if level == "info" {
						require.Empty(t, output.String())
					} else {
						require.Equal(t, 2, strings.Count(output.String(), "api #"))
						require.Contains(t, output.String(), "GET /v2/process-instances/"+resumptionTestKey)
					}
				})
			})
		}
	}
}

// resumptionNilReadClient verifies a malformed generated-client response cannot panic.
type resumptionNilReadClient struct{ resumptionSubmissionClient }

// GetProcessInstanceWithResponse simulates missing generated response metadata.
func (s resumptionNilReadClient) GetProcessInstanceWithResponse(context.Context, camundav810.ProcessInstanceKey, ...camundav810.RequestEditorFn) (*camundav810.GetProcessInstanceResponse, error) {
	return nil, nil
}

// TestProcessInstanceResumptionNilRead retains acceptance on a malformed read.
func TestProcessInstanceResumptionNilRead(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{App: config.App{Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: time.Second}}}
	client := resumptionNilReadClient{resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
		return resumptionResponse(204, ""), nil
	}}}
	result, err := (&Service{cc: client, cfg: cfg}).ResumeProcessInstance(context.Background(), resumptionTestKey)
	require.ErrorIs(t, err, d.ErrMalformedResponse)
	require.Equal(t, 204, result.StatusCode)
	require.Equal(t, d.ProcessInstanceResumptionSubmissionAccepted, result.Submission)
	require.False(t, result.HasObservation)
}

// TestProcessInstanceResumptionFreshObservation proves generated reads replace
// stale dates, both in workflow evidence and in a subsequent independent lookup.
func TestProcessInstanceResumptionFreshObservation(t *testing.T) {
	t.Parallel()
	for _, state := range []d.State{d.StateActive, d.StateCompleted} {
		t.Run(string(state), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				submissions, reads := 0, 0
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(&http.Client{Transport: resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
					if req.Method == http.MethodPost {
						submissions++
						require.Equal(t, "/v2/process-instances/"+resumptionTestKey+"/resumption", req.URL.Path)
						return &http.Response{StatusCode: 204, Request: req, Body: http.NoBody}, nil
					}
					require.Equal(t, http.MethodGet, req.Method)
					require.Equal(t, "/v2/process-instances/"+resumptionTestKey, req.URL.Path)
					reads++
					body := fmt.Sprintf(`{"processInstanceKey":%q,"state":%q}`, resumptionTestKey, state)
					if reads == 1 {
						body = fmt.Sprintf(`{"processInstanceKey":%q,"state":"SUSPENDED","suspendedDate":"2026-10-25T12:00:00Z"}`, resumptionTestKey)
					}
					return &http.Response{StatusCode: 200, Request: req, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
				})}))
				require.NoError(t, err)
				cfg := &config.Config{App: config.App{Backoff: config.BackoffConfig{Strategy: config.BackoffFixed, InitialDelay: time.Second, Timeout: 5 * time.Second}}}
				svc := &Service{cc: client, cfg: cfg, log: slog.New(slog.NewTextHandler(io.Discard, nil))}
				got, err := svc.ResumeProcessInstance(context.Background(), resumptionTestKey)
				require.NoError(t, err)
				require.Equal(t, d.ProcessInstanceResumptionStatusConfirmed, got.Status)
				require.Equal(t, d.ProcessInstanceResumptionSubmissionAccepted, got.Submission)
				require.Equal(t, d.ProcessInstanceResumptionConfirmationConfirmed, got.Confirmation)
				require.True(t, got.HasObservation)
				require.Equal(t, state, got.LastObservedState)
				require.Equal(t, 204, got.StatusCode)
				require.Empty(t, got.SuspendedDate)
				data, err := json.Marshal(got)
				require.NoError(t, err)
				require.NotContains(t, string(data), "suspendedDate")
				fresh, err := svc.GetProcessInstance(context.Background(), resumptionTestKey)
				require.NoError(t, err)
				require.Equal(t, state, fresh.State)
				require.Empty(t, fresh.SuspendedDate)
				require.Equal(t, 1, submissions)
				require.Equal(t, 3, reads)
			})
		})
	}
}

// TestProcessInstanceResumptionNoWaitTiming exempts submission-only calls from
// confirmation configuration without accidentally starting a read.
func TestProcessInstanceResumptionNoWaitTiming(t *testing.T) {
	t.Parallel()
	calls := 0
	svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
		calls++
		return resumptionResponse(204, ""), nil
	}}}
	got, err := svc.ResumeProcessInstance(context.Background(), resumptionTestKey, services.WithNoWait())
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, d.ProcessInstanceResumptionStatusAccepted, got.Status)
	require.Equal(t, d.ProcessInstanceResumptionConfirmationNotRequested, got.Confirmation)
	require.False(t, got.HasObservation)
}
