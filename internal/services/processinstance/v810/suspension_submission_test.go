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
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

const suspensionTestKey = "2251799813685249"

// suspensionSubmissionClient makes every unexpected read or hierarchy call fail
// while allowing private submission tests to inject generated response metadata.
type suspensionSubmissionClient struct {
	GenProcessInstanceClientCamunda
	submit func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error)
	t      *testing.T
}

// SuspendProcessInstanceWithResponse verifies the supplied key and empty body on every attempt.
func (c suspensionSubmissionClient) SuspendProcessInstanceWithResponse(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.SuspendProcessInstanceJSONRequestBody, editors ...camundav810.RequestEditorFn) (*camundav810.SuspendProcessInstanceResponse, error) {
	require.Equal(c.t, suspensionTestKey, string(key))
	require.Equal(c.t, camundav810.SuspendProcessInstanceJSONRequestBody{}, body)
	require.Empty(c.t, editors)
	return c.submit(ctx)
}

// suspensionResponse supplies complete metadata and immediate retries without real network I/O.
func suspensionResponse(code int, body string) *camundav810.SuspendProcessInstanceResponse {
	request, _ := http.NewRequest(http.MethodPost, "https://fixture.invalid/v2/process-instances/"+suspensionTestKey+"/suspension", nil)
	return &camundav810.SuspendProcessInstanceResponse{HTTPResponse: &http.Response{
		StatusCode: code, Status: http.StatusText(code), Request: request,
		Header: http.Header{"Retry-After": []string{"0"}},
	}, Body: []byte(body)}
}

// TestProcessInstanceSuspensionSubmissionProtocol requires authoritative acceptance
// and preserves typed rejection/proxy errors without reading or reconciling state.
func TestProcessInstanceSuspensionSubmissionProtocol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		code       int
		body       string
		cause      error
		submission d.ProcessInstanceSuspensionSubmission
		attempts   int
	}{
		{"accepted", 204, "", nil, d.ProcessInstanceSuspensionSubmissionAccepted, 1},
		{"already suspended", 409, `{"detail":"already suspended"}`, d.ErrConflict, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"completed", 409, `{"detail":"completed"}`, d.ErrConflict, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"terminated", 409, `{"detail":"terminated"}`, d.ErrConflict, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"terminating", 409, `{"detail":"terminating"}`, d.ErrConflict, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"missing", 404, "missing target", d.ErrNotFound, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"unauthorized", 401, "unauthorized target", d.ErrUnauthorized, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"forbidden", 403, "forbidden target", d.ErrForbidden, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"invalid", 400, "bad input", d.ErrBadRequest, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"precondition", 412, "precondition", d.ErrPrecondition, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"media", 415, "media", d.ErrUnsupported, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"validation", 422, "validation", d.ErrValidation, d.ProcessInstanceSuspensionSubmissionRejected, 1},
		{"throttle exhausted", 429, "limited", d.ErrRateLimited, d.ProcessInstanceSuspensionSubmissionRejected, 8},
		{"resource exhausted", 503, `{"title":"RESOURCE_EXHAUSTED"}`, d.ErrUnavailable, d.ProcessInstanceSuspensionSubmissionRejected, 8},
		{"resource detail", 503, `{"detail":"RESOURCE_EXHAUSTED"}`, d.ErrUnavailable, d.ProcessInstanceSuspensionSubmissionRejected, 8},
		{"proxy marker", 503, "<html>RESOURCE_EXHAUSTED</html>", d.ErrUnavailable, d.ProcessInstanceSuspensionSubmissionUnknown, 8},
		{"server", 500, "server failure", d.ErrInternal, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
		{"proxy", 502, "proxy failure", d.ErrUpstream, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
		{"unavailable", 503, "unavailable", d.ErrUnavailable, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
		{"gateway timeout", 504, "timeout", d.ErrGatewayTimeout, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
		{"redirect", 307, "proxy redirect", d.ErrUpstream, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
		{"unexpected success", 200, "", d.ErrMalformedResponse, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
		{"unexpected accepted", 202, "", d.ErrMalformedResponse, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
		{"204 with body", 204, "unexpected body", d.ErrMalformedResponse, d.ProcessInstanceSuspensionSubmissionUnknown, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := &Service{cc: suspensionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error) {
				calls++
				return suspensionResponse(tc.code, tc.body), nil
			}}}
			got, err := svc.submitProcessInstanceSuspension(context.Background(), suspensionTestKey)
			if tc.cause == nil {
				require.NoError(t, err)
				require.Equal(t, d.ProcessInstanceSuspensionStatusAccepted, got.Status)
				require.Empty(t, got.Error)
			} else {
				require.ErrorIs(t, err, tc.cause)
				require.Equal(t, err.Error(), got.Error)
				status := d.ProcessInstanceSuspensionStatusFailed
				if tc.code == 504 {
					status = d.ProcessInstanceSuspensionStatusTimedOut
				}
				require.Equal(t, status, got.Status)
				if tc.code >= 400 {
					require.Contains(t, got.Error, tc.body)
				}
			}
			require.Equal(t, tc.attempts, calls)
			require.Equal(t, suspensionTestKey, got.Key)
			require.Equal(t, tc.code, got.StatusCode)
			require.Equal(t, tc.submission, got.Submission)
			require.Equal(t, d.ProcessInstanceSuspensionConfirmationNotRequested, got.Confirmation)
			require.False(t, got.HasObservation)
			require.Empty(t, got.LastObservedState)
			require.Empty(t, got.SuspendedDate)
		})
	}
}

// TestProcessInstanceSuspensionSubmissionMalformed never accepts incomplete metadata
// or a partial response accompanied by a call error, including apparent 204s.
func TestProcessInstanceSuspensionSubmissionMalformed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		response *camundav810.SuspendProcessInstanceResponse
		callErr  error
		code     int
	}{
		{"nil", nil, nil, 0},
		{"no metadata", &camundav810.SuspendProcessInstanceResponse{}, nil, 0},
		{"no request", &camundav810.SuspendProcessInstanceResponse{HTTPResponse: &http.Response{StatusCode: 204}}, nil, 0},
		{"no URL", &camundav810.SuspendProcessInstanceResponse{HTTPResponse: &http.Response{StatusCode: 204, Request: &http.Request{}}}, nil, 0},
		{"invalid status", suspensionResponse(600, ""), nil, 0},
		{"zero status", suspensionResponse(0, ""), nil, 0},
		{"partial acceptance", suspensionResponse(204, ""), io.ErrUnexpectedEOF, 204},
		{"partial rejection", suspensionResponse(409, ""), io.ErrUnexpectedEOF, 409},
		{"transport", nil, io.ErrUnexpectedEOF, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := &Service{cc: suspensionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error) {
				calls++
				return tc.response, tc.callErr
			}}}
			got, err := svc.submitProcessInstanceSuspension(context.Background(), suspensionTestKey)
			cause := tc.callErr
			if cause == nil {
				cause = d.ErrMalformedResponse
			}
			require.ErrorIs(t, err, cause)
			require.Equal(t, 1, calls)
			require.Equal(t, tc.code, got.StatusCode)
			require.Equal(t, d.ProcessInstanceSuspensionSubmissionUnknown, got.Submission)
			require.Equal(t, d.ProcessInstanceSuspensionStatusFailed, got.Status)
			require.Equal(t, d.ProcessInstanceSuspensionConfirmationNotRequested, got.Confirmation)
			require.False(t, got.HasObservation)
		})
	}
}

// suspensionNetworkError supplies deterministic temporary network and timeout failures.
type suspensionNetworkError struct{ timeout bool }

// Error exposes machine-only evidence that must never be used as a functional log message.
func (e suspensionNetworkError) Error() string {
	return "POST https://fixture.invalid/transport secret-body"
}

// Timeout models a transport deadline independently of temporary failure.
func (e suspensionNetworkError) Timeout() bool { return e.timeout }

// Temporary activates the existing finite mutation retry policy.
func (e suspensionNetworkError) Temporary() bool { return true }

// TestProcessInstanceSuspensionSubmissionRetries retains ambiguity until actual
// acceptance and uses virtual time to exercise the unchanged retry budget quickly.
func TestProcessInstanceSuspensionSubmissionRetries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		first   int
		callErr error
		final   int
		want    d.ProcessInstanceSuspensionSubmission
	}{
		{"429 accepted", 429, nil, 204, d.ProcessInstanceSuspensionSubmissionAccepted},
		{"503 accepted", 503, nil, 204, d.ProcessInstanceSuspensionSubmissionAccepted},
		{"ambiguous rejected", 0, suspensionNetworkError{}, 409, d.ProcessInstanceSuspensionSubmissionUnknown},
		{"ambiguous accepted", 0, suspensionNetworkError{}, 204, d.ProcessInstanceSuspensionSubmissionAccepted},
		{"partial accepted then rejected", 204, suspensionNetworkError{}, 409, d.ProcessInstanceSuspensionSubmissionUnknown},
		{"rejected then rejected", 429, nil, 409, d.ProcessInstanceSuspensionSubmissionRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				svc := &Service{cc: suspensionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error) {
					calls++
					if calls == 1 {
						if tc.first == 0 {
							return nil, tc.callErr
						}
						return suspensionResponse(tc.first, `{"title":"RESOURCE_EXHAUSTED"}`), tc.callErr
					}
					return suspensionResponse(tc.final, ""), nil
				}}}
				got, err := svc.submitProcessInstanceSuspension(context.Background(), suspensionTestKey)
				if tc.final == 204 {
					require.NoError(t, err)
					require.Empty(t, got.Error)
				} else {
					require.ErrorIs(t, err, d.ErrConflict)
				}
				require.Equal(t, 2, calls)
				require.Equal(t, tc.final, got.StatusCode)
				require.Equal(t, tc.want, got.Submission)
				require.Equal(t, d.ProcessInstanceSuspensionConfirmationNotRequested, got.Confirmation)
			})
		})
	}
	for _, timeout := range []bool{false, true} {
		t.Run(fmt.Sprintf("exhausted timeout=%t", timeout), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				cause := suspensionNetworkError{timeout: timeout}
				svc := &Service{cc: suspensionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error) { calls++; return nil, cause }}}
				got, err := svc.submitProcessInstanceSuspension(context.Background(), suspensionTestKey)
				require.ErrorIs(t, err, cause)
				require.Equal(t, 8, calls)
				require.Equal(t, d.ProcessInstanceSuspensionSubmissionUnknown, got.Submission)
				require.Zero(t, got.StatusCode)
				if timeout {
					require.Equal(t, d.ProcessInstanceSuspensionStatusTimedOut, got.Status)
					require.ErrorIs(t, err, d.ErrGatewayTimeout)
				} else {
					require.Equal(t, d.ProcessInstanceSuspensionStatusFailed, got.Status)
				}
			})
		})
	}
}

// TestProcessInstanceSuspensionSubmissionCancellation preserves evidence when
// cancellation happens before sending, in flight, or during retry backoff.
func TestProcessInstanceSuspensionSubmissionCancellation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		before    bool
		response  int
		ambiguous bool
		deadline  bool
		want      d.ProcessInstanceSuspensionSubmission
	}{
		{"before send", true, 0, false, false, d.ProcessInstanceSuspensionSubmissionNotSubmitted},
		{"expired", true, 0, false, true, d.ProcessInstanceSuspensionSubmissionNotSubmitted},
		{"in flight", false, 0, true, false, d.ProcessInstanceSuspensionSubmissionUnknown},
		{"backoff rejected", false, 429, false, false, d.ProcessInstanceSuspensionSubmissionRejected},
		{"backoff ambiguous", false, 429, true, false, d.ProcessInstanceSuspensionSubmissionUnknown},
		{"backoff deadline", false, 429, false, true, d.ProcessInstanceSuspensionSubmissionRejected},
		{"accepted despite cancel", false, 204, false, false, d.ProcessInstanceSuspensionSubmissionAccepted},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				if tc.deadline {
					cancel()
					ctx, cancel = context.WithTimeout(context.Background(), time.Second)
				}
				defer cancel()
				if tc.before {
					if tc.deadline {
						<-ctx.Done()
					} else {
						cancel()
					}
				}
				calls := 0
				svc := &Service{cc: suspensionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error) {
					calls++
					if !tc.deadline {
						cancel()
					}
					var response *camundav810.SuspendProcessInstanceResponse
					if tc.response != 0 {
						response = suspensionResponse(tc.response, "")
						response.HTTPResponse.Header.Set("Retry-After", "3600")
					}
					if tc.ambiguous {
						return response, suspensionNetworkError{}
					}
					return response, nil
				}}}
				got, err := svc.submitProcessInstanceSuspension(ctx, suspensionTestKey)
				if tc.response == 204 {
					require.NoError(t, err)
					require.Equal(t, d.ProcessInstanceSuspensionStatusAccepted, got.Status)
				} else if tc.deadline {
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.Equal(t, d.ProcessInstanceSuspensionStatusTimedOut, got.Status)
				} else {
					require.ErrorIs(t, err, context.Canceled)
					require.Equal(t, d.ProcessInstanceSuspensionStatusInterrupted, got.Status)
				}
				if tc.before {
					require.Zero(t, calls)
				} else {
					require.Equal(t, 1, calls)
				}
				require.Equal(t, tc.want, got.Submission)
				require.Equal(t, tc.response, got.StatusCode)
				require.Equal(t, suspensionTestKey, got.Key)
			})
		})
	}
}

// suspensionRoundTripper ensures wire tests can never reach a real endpoint.
type suspensionRoundTripper func(*http.Request) (*http.Response, error)

// RoundTrip delegates solely to the test-owned in-memory transport.
func (f suspensionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestProcessInstanceSuspensionSubmissionWire exercises the generated serializer
// and rejects every request except POST to the exact selected instance with {}.
func TestProcessInstanceSuspensionSubmissionWire(t *testing.T) {
	t.Parallel()
	calls := 0
	client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(&http.Client{Transport: suspensionRoundTripper(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/v2/process-instances/"+suspensionTestKey+"/suspension", req.URL.Path)
		require.Empty(t, req.URL.RawQuery)
		require.Equal(t, "application/json", req.Header.Get("Content-Type"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Equal(t, "{}", strings.TrimSpace(string(body)))
		return &http.Response{StatusCode: 204, Request: req, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})}))
	require.NoError(t, err)
	svc := &Service{cc: client}
	got, err := svc.submitProcessInstanceSuspension(context.Background(), suspensionTestKey)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, d.ProcessInstanceSuspensionSubmissionAccepted, got.Submission)
	require.False(t, got.HasObservation)
}

// TestProcessInstanceSuspensionSubmissionPacing proves positively unsent attempts
// preserve all earlier evidence even when the real HTTP client replaces their error.
func TestProcessInstanceSuspensionSubmissionPacing(t *testing.T) {
	for _, prior := range []string{"none", "rejected", "unknown"} {
		for _, substituted := range []bool{false, true} {
			t.Run(fmt.Sprintf("prior=%s substituted=%t", prior, substituted), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					var dispatched atomic.Int32
					shared, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "100ms", RequestRate: 0.01, RequestRateReportInterval: "0s"}}, slog.New(slog.NewTextHandler(io.Discard, nil)), func(s *httpc.Service) {
						s.Client().Transport = suspensionRoundTripper(func(req *http.Request) (*http.Response, error) {
							dispatched.Add(1)
							require.Equal(t, http.MethodPost, req.Method)
							require.Equal(t, "/v2/process-instances/"+suspensionTestKey+"/suspension", req.URL.Path)
							body, err := io.ReadAll(req.Body)
							require.NoError(t, err)
							require.JSONEq(t, "{}", string(body))
							return &http.Response{StatusCode: 429, Request: req, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
						})
					})
					require.NoError(t, err)
					defer shared.Close()
					parent := shared.ToContext(context.Background())
					client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(shared.Client()))
					require.NoError(t, err)
					if prior == "none" {
						// Consume the burst-one allowance outside the measured operation.
						_, err := client.SuspendProcessInstanceWithResponse(parent, suspensionTestKey, camundav810.SuspendProcessInstanceJSONRequestBody{})
						require.NoError(t, err)
					}
					base := shared.Client().Transport
					var substitutions atomic.Int32
					if substituted {
						shared.Client().Transport = suspensionRoundTripper(func(req *http.Request) (*http.Response, error) {
							response, err := base.RoundTrip(req)
							if httpc.IsRequestRateWaitError(err) {
								substitutions.Add(1)
								<-req.Cancel // Force real Client.Timeout substitution after admission failed.
							}
							return response, err
						})
					}
					calls := 0
					svc := &Service{cc: suspensionSubmissionClient{t: t, submit: func(ctx context.Context) (*camundav810.SuspendProcessInstanceResponse, error) {
						calls++
						response, err := client.SuspendProcessInstanceWithResponse(ctx, suspensionTestKey, camundav810.SuspendProcessInstanceJSONRequestBody{})
						if prior == "unknown" && calls == 1 {
							// A call error with a partial HTTP response is ambiguous, even for 429.
							return response, suspensionNetworkError{}
						}
						if substituted && err != nil {
							require.False(t, httpc.IsRequestRateWaitError(err), "HTTP timeout must actually replace the local error")
						}
						return response, err
					}}}
					got, err := svc.submitProcessInstanceSuspension(parent, suspensionTestKey)
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.True(t, httpc.IsRequestRateWaitError(err))
					require.NoError(t, parent.Err())
					require.Equal(t, int32(1), dispatched.Load(), "no physical dispatch after admission failure")
					require.Equal(t, d.ProcessInstanceSuspensionStatusTimedOut, got.Status)
					require.Equal(t, d.ProcessInstanceSuspensionConfirmationNotRequested, got.Confirmation)
					switch prior {
					case "none":
						require.Equal(t, 1, calls)
						require.Zero(t, got.StatusCode)
						require.Equal(t, d.ProcessInstanceSuspensionSubmissionNotSubmitted, got.Submission)
					case "rejected":
						require.Equal(t, 2, calls)
						require.Equal(t, 429, got.StatusCode)
						require.Equal(t, d.ProcessInstanceSuspensionSubmissionRejected, got.Submission)
					case "unknown":
						require.Equal(t, 2, calls)
						require.Equal(t, 429, got.StatusCode)
						require.Equal(t, d.ProcessInstanceSuspensionSubmissionUnknown, got.Submission)
					}
					if substituted {
						require.Equal(t, int32(1), substitutions.Load())
					}
					_, observation := httpc.BeginRequestRateAttempt(parent)
					require.NoError(t, observation.Failure(), "later calls must not inherit attempt-local failure")
				})
			})
		}
	}
}

// TestProcessInstanceSuspensionSubmissionDiagnostics keeps normal and verbose-only
// service logging silent while preserving configured, redacted HTTP DEBUG records.
func TestProcessInstanceSuspensionSubmissionDiagnostics(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"plain", "json"} {
		for _, level := range []string{"info", "debug"} {
			t.Run(format+"/"+level, func(t *testing.T) {
				var output bytes.Buffer
				log := logging.New(logging.LoggerConfig{Writer: &output, Level: level, Format: format})
				calls := 0
				cfg := &config.Config{HTTP: config.HTTP{Timeout: "1s", RequestRateReportInterval: "0s"}}
				shared, err := httpc.New(cfg, log, func(s *httpc.Service) {
					s.Client().Transport = suspensionRoundTripper(func(req *http.Request) (*http.Response, error) {
						calls++
						code, body := 429, `{"detail":"rate limited"}`
						if calls == 2 {
							code, body = 409, `{"detail":"already suspended"}`
						}
						return &http.Response{StatusCode: code, Request: req, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
					})
				}, httpc.WithDiagnostics())
				require.NoError(t, err)
				defer shared.Close()
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(shared.Client()), camundav810.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
					req.Header.Set("Authorization", "Bearer suspension-test-secret")
					return nil
				}))
				require.NoError(t, err)
				svc := &Service{cc: client, log: log}
				got, err := svc.submitProcessInstanceSuspension(shared.ToContext(context.Background()), suspensionTestKey)
				require.ErrorIs(t, err, d.ErrConflict)
				require.Equal(t, 2, calls)
				require.Equal(t, d.ProcessInstanceSuspensionSubmissionRejected, got.Submission)
				require.Contains(t, got.Error, "https://fixture.invalid/v2/process-instances/"+suspensionTestKey+"/suspension")
				require.NotContains(t, output.String(), "suspension-test-secret")
				require.NotContains(t, output.String(), "retrying")
				if level == "info" {
					require.Empty(t, output.String())
					return
				}
				require.Equal(t, 2, strings.Count(output.String(), "api #"))
				require.Contains(t, output.String(), "POST /v2/process-instances/"+suspensionTestKey+"/suspension")
				if format == "json" {
					decoder := json.NewDecoder(&output)
					for range 2 {
						var record map[string]any
						require.NoError(t, decoder.Decode(&record))
						require.Equal(t, "DEBUG", record["level"])
					}
					var extra any
					require.ErrorIs(t, decoder.Decode(&extra), io.EOF)
				}
			})
		}
	}
}

// TestProcessInstanceSuspensionSubmissionLastStatus retains the last usable HTTP
// status across later transport failures or malformed metadata without accepting them.
func TestProcessInstanceSuspensionSubmissionLastStatus(t *testing.T) {
	t.Parallel()
	for _, malformed := range []bool{false, true} {
		t.Run(fmt.Sprint(malformed), func(t *testing.T) {
			calls := 0
			svc := &Service{cc: suspensionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.SuspendProcessInstanceResponse, error) {
				calls++
				if calls == 1 {
					return suspensionResponse(429, ""), nil
				}
				if malformed {
					return &camundav810.SuspendProcessInstanceResponse{HTTPResponse: &http.Response{StatusCode: 204}}, nil
				}
				return nil, io.ErrUnexpectedEOF
			}}}
			got, err := svc.submitProcessInstanceSuspension(context.Background(), suspensionTestKey)
			if malformed {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
			} else {
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			}
			require.Equal(t, 2, calls)
			require.Equal(t, 429, got.StatusCode)
			require.Equal(t, d.ProcessInstanceSuspensionSubmissionUnknown, got.Submission)
			require.Equal(t, d.ProcessInstanceSuspensionStatusFailed, got.Status)
		})
	}
}
