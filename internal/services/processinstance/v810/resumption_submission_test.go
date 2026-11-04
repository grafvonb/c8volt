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

// resumptionTestKey is an isolated exact-key target used by the adapter fixtures.
const resumptionTestKey = "2251799813685249"

// resumptionSubmissionClient makes every unexpected read or hierarchy call fail
// while allowing private submission tests to inject generated response metadata.
type resumptionSubmissionClient struct {
	GenProcessInstanceClientCamunda
	submit func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error)
	t      *testing.T
}

// ResumeProcessInstanceWithResponse verifies the supplied key and empty body on every attempt.
func (c resumptionSubmissionClient) ResumeProcessInstanceWithResponse(ctx context.Context, key camundav810.ProcessInstanceKey, body camundav810.ResumeProcessInstanceJSONRequestBody, editors ...camundav810.RequestEditorFn) (*camundav810.ResumeProcessInstanceResponse, error) {
	require.Equal(c.t, resumptionTestKey, string(key))
	require.Equal(c.t, camundav810.ResumeProcessInstanceJSONRequestBody{}, body)
	require.Empty(c.t, editors)
	return c.submit(ctx)
}

// resumptionResponse supplies complete metadata and immediate retries without real network I/O.
func resumptionResponse(code int, body string) *camundav810.ResumeProcessInstanceResponse {
	request, _ := http.NewRequest(http.MethodPost, "https://fixture.invalid/v2/process-instances/"+resumptionTestKey+"/resumption", nil)
	return &camundav810.ResumeProcessInstanceResponse{HTTPResponse: &http.Response{
		StatusCode: code, Status: http.StatusText(code), Request: request,
		Header: http.Header{"Retry-After": []string{"0"}},
	}, Body: []byte(body)}
}

// TestProcessInstanceResumptionSubmissionProtocol requires authoritative acceptance
// and preserves typed rejection/proxy errors without reading or reconciling state.
func TestProcessInstanceResumptionSubmissionProtocol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		code       int
		body       string
		cause      error
		submission d.ProcessInstanceResumptionSubmission
		attempts   int
	}{
		{"accepted", 204, "", nil, d.ProcessInstanceResumptionSubmissionAccepted, 1},
		{"already active", 409, `{"detail":"already active"}`, d.ErrConflict, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"completed", 409, `{"detail":"completed"}`, d.ErrConflict, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"terminated", 409, `{"detail":"terminated"}`, d.ErrConflict, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"terminating", 409, `{"detail":"terminating"}`, d.ErrConflict, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"missing", 404, "missing target", d.ErrNotFound, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"unauthorized", 401, "unauthorized target", d.ErrUnauthorized, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"forbidden", 403, "forbidden target", d.ErrForbidden, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"invalid", 400, "bad input", d.ErrBadRequest, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"precondition", 412, "precondition", d.ErrPrecondition, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"media", 415, "media", d.ErrUnsupported, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"validation", 422, "validation", d.ErrValidation, d.ProcessInstanceResumptionSubmissionRejected, 1},
		{"throttle exhausted", 429, "limited", d.ErrRateLimited, d.ProcessInstanceResumptionSubmissionRejected, 8},
		{"resource exhausted", 503, `{"title":"RESOURCE_EXHAUSTED"}`, d.ErrUnavailable, d.ProcessInstanceResumptionSubmissionRejected, 8},
		{"resource detail", 503, `{"detail":"RESOURCE_EXHAUSTED"}`, d.ErrUnavailable, d.ProcessInstanceResumptionSubmissionRejected, 8},
		{"proxy marker", 503, "<html>RESOURCE_EXHAUSTED</html>", d.ErrUnavailable, d.ProcessInstanceResumptionSubmissionUnknown, 8},
		{"server", 500, "server failure", d.ErrInternal, d.ProcessInstanceResumptionSubmissionUnknown, 1},
		{"proxy", 502, "proxy failure", d.ErrUpstream, d.ProcessInstanceResumptionSubmissionUnknown, 1},
		{"unavailable", 503, "unavailable", d.ErrUnavailable, d.ProcessInstanceResumptionSubmissionUnknown, 1},
		{"gateway timeout", 504, "timeout", d.ErrGatewayTimeout, d.ProcessInstanceResumptionSubmissionUnknown, 1},
		{"redirect", 307, "proxy redirect", d.ErrUpstream, d.ProcessInstanceResumptionSubmissionUnknown, 1},
		{"unexpected success", 200, "", d.ErrMalformedResponse, d.ProcessInstanceResumptionSubmissionUnknown, 1},
		{"unexpected accepted", 202, "", d.ErrMalformedResponse, d.ProcessInstanceResumptionSubmissionUnknown, 1},
		{"204 with body", 204, "unexpected body", d.ErrMalformedResponse, d.ProcessInstanceResumptionSubmissionUnknown, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
				calls++
				return resumptionResponse(tc.code, tc.body), nil
			}}}
			got, err := svc.submitProcessInstanceResumption(context.Background(), resumptionTestKey)
			if tc.cause == nil {
				require.NoError(t, err)
				require.Equal(t, d.ProcessInstanceResumptionStatusAccepted, got.Status)
				require.Empty(t, got.Error)
			} else {
				require.ErrorIs(t, err, tc.cause)
				require.Equal(t, err.Error(), got.Error)
				status := d.ProcessInstanceResumptionStatusFailed
				if tc.code == 504 {
					status = d.ProcessInstanceResumptionStatusTimedOut
				}
				require.Equal(t, status, got.Status)
				if tc.code >= 400 {
					require.Contains(t, got.Error, tc.body)
				}
			}
			require.Equal(t, tc.attempts, calls)
			require.Equal(t, resumptionTestKey, got.Key)
			require.Equal(t, tc.code, got.StatusCode)
			require.Equal(t, tc.submission, got.Submission)
			require.Equal(t, d.ProcessInstanceResumptionConfirmationNotRequested, got.Confirmation)
			require.False(t, got.HasObservation)
			require.Empty(t, got.LastObservedState)
			require.Empty(t, got.SuspendedDate)
		})
	}
}

// TestProcessInstanceResumptionSubmissionMalformed never accepts incomplete metadata
// or a partial response accompanied by a call error, including apparent 204s.
func TestProcessInstanceResumptionSubmissionMalformed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		response *camundav810.ResumeProcessInstanceResponse
		callErr  error
		code     int
	}{
		{"nil", nil, nil, 0},
		{"no metadata", &camundav810.ResumeProcessInstanceResponse{}, nil, 0},
		{"no request", &camundav810.ResumeProcessInstanceResponse{HTTPResponse: &http.Response{StatusCode: 204}}, nil, 0},
		{"no URL", &camundav810.ResumeProcessInstanceResponse{HTTPResponse: &http.Response{StatusCode: 204, Request: &http.Request{}}}, nil, 0},
		{"invalid status", resumptionResponse(600, ""), nil, 0},
		{"zero status", resumptionResponse(0, ""), nil, 0},
		{"partial acceptance", resumptionResponse(204, ""), io.ErrUnexpectedEOF, 204},
		{"partial rejection", resumptionResponse(409, ""), io.ErrUnexpectedEOF, 409},
		{"transport", nil, io.ErrUnexpectedEOF, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
				calls++
				return tc.response, tc.callErr
			}}}
			got, err := svc.submitProcessInstanceResumption(context.Background(), resumptionTestKey)
			cause := tc.callErr
			if cause == nil {
				cause = d.ErrMalformedResponse
			}
			require.ErrorIs(t, err, cause)
			require.Equal(t, 1, calls)
			require.Equal(t, tc.code, got.StatusCode)
			require.Equal(t, d.ProcessInstanceResumptionSubmissionUnknown, got.Submission)
			require.Equal(t, d.ProcessInstanceResumptionStatusFailed, got.Status)
			require.Equal(t, d.ProcessInstanceResumptionConfirmationNotRequested, got.Confirmation)
			require.False(t, got.HasObservation)
		})
	}
}

// resumptionNetworkError supplies deterministic temporary network and timeout failures.
type resumptionNetworkError struct{ timeout bool }

// Error exposes machine-only evidence that must never be used as a functional log message.
func (e resumptionNetworkError) Error() string {
	return "POST https://fixture.invalid/transport secret-body"
}

// Timeout models a transport deadline independently of temporary failure.
func (e resumptionNetworkError) Timeout() bool { return e.timeout }

// Temporary activates the existing finite mutation retry policy.
func (e resumptionNetworkError) Temporary() bool { return true }

// TestProcessInstanceResumptionSubmissionRetries retains ambiguity until actual
// acceptance and uses virtual time to exercise the unchanged retry budget quickly.
func TestProcessInstanceResumptionSubmissionRetries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		first   int
		callErr error
		final   int
		want    d.ProcessInstanceResumptionSubmission
	}{
		{"429 accepted", 429, nil, 204, d.ProcessInstanceResumptionSubmissionAccepted},
		{"503 accepted", 503, nil, 204, d.ProcessInstanceResumptionSubmissionAccepted},
		{"ambiguous rejected", 0, resumptionNetworkError{}, 409, d.ProcessInstanceResumptionSubmissionUnknown},
		{"ambiguous accepted", 0, resumptionNetworkError{}, 204, d.ProcessInstanceResumptionSubmissionAccepted},
		{"partial accepted then rejected", 204, resumptionNetworkError{}, 409, d.ProcessInstanceResumptionSubmissionUnknown},
		{"rejected then rejected", 429, nil, 409, d.ProcessInstanceResumptionSubmissionRejected},
	} {
		t.Run(tc.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
					calls++
					if calls == 1 {
						if tc.first == 0 {
							return nil, tc.callErr
						}
						return resumptionResponse(tc.first, `{"title":"RESOURCE_EXHAUSTED"}`), tc.callErr
					}
					return resumptionResponse(tc.final, ""), nil
				}}}
				got, err := svc.submitProcessInstanceResumption(context.Background(), resumptionTestKey)
				if tc.final == 204 {
					require.NoError(t, err)
					require.Empty(t, got.Error)
				} else {
					require.ErrorIs(t, err, d.ErrConflict)
				}
				require.Equal(t, 2, calls)
				require.Equal(t, tc.final, got.StatusCode)
				require.Equal(t, tc.want, got.Submission)
				require.Equal(t, d.ProcessInstanceResumptionConfirmationNotRequested, got.Confirmation)
			})
		})
	}
	for _, timeout := range []bool{false, true} {
		t.Run(fmt.Sprintf("exhausted timeout=%t", timeout), func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				calls := 0
				cause := resumptionNetworkError{timeout: timeout}
				svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) { calls++; return nil, cause }}}
				got, err := svc.submitProcessInstanceResumption(context.Background(), resumptionTestKey)
				require.ErrorIs(t, err, cause)
				require.Equal(t, 8, calls)
				require.Equal(t, d.ProcessInstanceResumptionSubmissionUnknown, got.Submission)
				require.Zero(t, got.StatusCode)
				if timeout {
					require.Equal(t, d.ProcessInstanceResumptionStatusTimedOut, got.Status)
					require.ErrorIs(t, err, d.ErrGatewayTimeout)
				} else {
					require.Equal(t, d.ProcessInstanceResumptionStatusFailed, got.Status)
				}
			})
		})
	}
}

// TestProcessInstanceResumptionSubmissionCancellation preserves evidence when
// cancellation happens before sending, in flight, or during retry backoff.
func TestProcessInstanceResumptionSubmissionCancellation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		before    bool
		response  int
		ambiguous bool
		deadline  bool
		want      d.ProcessInstanceResumptionSubmission
	}{
		{"before send", true, 0, false, false, d.ProcessInstanceResumptionSubmissionNotSubmitted},
		{"expired", true, 0, false, true, d.ProcessInstanceResumptionSubmissionNotSubmitted},
		{"in flight", false, 0, true, false, d.ProcessInstanceResumptionSubmissionUnknown},
		{"backoff rejected", false, 429, false, false, d.ProcessInstanceResumptionSubmissionRejected},
		{"backoff ambiguous", false, 429, true, false, d.ProcessInstanceResumptionSubmissionUnknown},
		{"backoff deadline", false, 429, false, true, d.ProcessInstanceResumptionSubmissionRejected},
		{"accepted despite cancel", false, 204, false, false, d.ProcessInstanceResumptionSubmissionAccepted},
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
				svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
					calls++
					if !tc.deadline {
						cancel()
					}
					var response *camundav810.ResumeProcessInstanceResponse
					if tc.response != 0 {
						response = resumptionResponse(tc.response, "")
						response.HTTPResponse.Header.Set("Retry-After", "3600")
					}
					if tc.ambiguous {
						return response, resumptionNetworkError{}
					}
					return response, nil
				}}}
				got, err := svc.submitProcessInstanceResumption(ctx, resumptionTestKey)
				if tc.response == 204 {
					require.NoError(t, err)
					require.Equal(t, d.ProcessInstanceResumptionStatusAccepted, got.Status)
				} else if tc.deadline {
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.Equal(t, d.ProcessInstanceResumptionStatusTimedOut, got.Status)
				} else {
					require.ErrorIs(t, err, context.Canceled)
					require.Equal(t, d.ProcessInstanceResumptionStatusInterrupted, got.Status)
				}
				if tc.before {
					require.Zero(t, calls)
				} else {
					require.Equal(t, 1, calls)
				}
				require.Equal(t, tc.want, got.Submission)
				require.Equal(t, tc.response, got.StatusCode)
				require.Equal(t, resumptionTestKey, got.Key)
			})
		})
	}
}

// resumptionRoundTripper ensures wire tests can never reach a real endpoint.
type resumptionRoundTripper func(*http.Request) (*http.Response, error)

// RoundTrip delegates solely to the test-owned in-memory transport.
func (f resumptionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestProcessInstanceResumptionSubmissionWire exercises the generated serializer
// and rejects every request except POST to the exact selected instance with {}.
func TestProcessInstanceResumptionSubmissionWire(t *testing.T) {
	t.Parallel()
	calls := 0
	client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(&http.Client{Transport: resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
		calls++
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "/v2/process-instances/"+resumptionTestKey+"/resumption", req.URL.Path)
		require.Empty(t, req.URL.RawQuery)
		require.Equal(t, "application/json", req.Header.Get("Content-Type"))
		body, err := io.ReadAll(req.Body)
		require.NoError(t, err)
		require.Equal(t, "{}", strings.TrimSpace(string(body)))
		return &http.Response{StatusCode: 204, Request: req, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(""))}, nil
	})}))
	require.NoError(t, err)
	svc := &Service{cc: client}
	got, err := svc.submitProcessInstanceResumption(context.Background(), resumptionTestKey)
	require.NoError(t, err)
	require.Equal(t, 1, calls)
	require.Equal(t, d.ProcessInstanceResumptionSubmissionAccepted, got.Submission)
	require.False(t, got.HasObservation)
}

// TestProcessInstanceResumptionSubmissionPacing proves positively unsent attempts
// preserve all earlier evidence even when the real HTTP client replaces their error.
func TestProcessInstanceResumptionSubmissionPacing(t *testing.T) {
	for _, prior := range []string{"none", "rejected", "unknown"} {
		for _, substituted := range []bool{false, true} {
			t.Run(fmt.Sprintf("prior=%s substituted=%t", prior, substituted), func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					var dispatched atomic.Int32
					shared, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "100ms", RequestRate: 0.01, RequestRateReportInterval: "0s"}}, slog.New(slog.NewTextHandler(io.Discard, nil)), func(s *httpc.Service) {
						s.Client().Transport = resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
							dispatched.Add(1)
							require.Equal(t, http.MethodPost, req.Method)
							require.Equal(t, "/v2/process-instances/"+resumptionTestKey+"/resumption", req.URL.Path)
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
						_, err := client.ResumeProcessInstanceWithResponse(parent, resumptionTestKey, camundav810.ResumeProcessInstanceJSONRequestBody{})
						require.NoError(t, err)
					}
					base := shared.Client().Transport
					var substitutions atomic.Int32
					if substituted {
						shared.Client().Transport = resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
							response, err := base.RoundTrip(req)
							if httpc.IsRequestRateWaitError(err) {
								substitutions.Add(1)
								<-req.Cancel // Force real Client.Timeout substitution after admission failed.
							}
							return response, err
						})
					}
					calls := 0
					svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(ctx context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
						calls++
						response, err := client.ResumeProcessInstanceWithResponse(ctx, resumptionTestKey, camundav810.ResumeProcessInstanceJSONRequestBody{})
						if prior == "unknown" && calls == 1 {
							// A call error with a partial HTTP response is ambiguous, even for 429.
							return response, resumptionNetworkError{}
						}
						if substituted && err != nil {
							require.False(t, httpc.IsRequestRateWaitError(err), "HTTP timeout must actually replace the local error")
						}
						return response, err
					}}}
					got, err := svc.submitProcessInstanceResumption(parent, resumptionTestKey)
					require.ErrorIs(t, err, context.DeadlineExceeded)
					require.True(t, httpc.IsRequestRateWaitError(err))
					require.NoError(t, parent.Err())
					require.Equal(t, int32(1), dispatched.Load(), "no physical dispatch after admission failure")
					require.Equal(t, d.ProcessInstanceResumptionStatusTimedOut, got.Status)
					require.Equal(t, d.ProcessInstanceResumptionConfirmationNotRequested, got.Confirmation)
					switch prior {
					case "none":
						require.Equal(t, 1, calls)
						require.Zero(t, got.StatusCode)
						require.Equal(t, d.ProcessInstanceResumptionSubmissionNotSubmitted, got.Submission)
					case "rejected":
						require.Equal(t, 2, calls)
						require.Equal(t, 429, got.StatusCode)
						require.Equal(t, d.ProcessInstanceResumptionSubmissionRejected, got.Submission)
					case "unknown":
						require.Equal(t, 2, calls)
						require.Equal(t, 429, got.StatusCode)
						require.Equal(t, d.ProcessInstanceResumptionSubmissionUnknown, got.Submission)
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

// TestProcessInstanceResumptionSubmissionDiagnostics keeps normal and verbose-only
// service logging silent while preserving configured, redacted HTTP DEBUG records.
func TestProcessInstanceResumptionSubmissionDiagnostics(t *testing.T) {
	t.Parallel()
	for _, format := range []string{"plain", "json"} {
		for _, level := range []string{"info", "debug"} {
			t.Run(format+"/"+level, func(t *testing.T) {
				var output bytes.Buffer
				log := logging.New(logging.LoggerConfig{Writer: &output, Level: level, Format: format})
				calls := 0
				cfg := &config.Config{HTTP: config.HTTP{Timeout: "1s", RequestRateReportInterval: "0s"}}
				shared, err := httpc.New(cfg, log, func(s *httpc.Service) {
					s.Client().Transport = resumptionRoundTripper(func(req *http.Request) (*http.Response, error) {
						calls++
						code, body := 429, `{"detail":"rate limited"}`
						if calls == 2 {
							code, body = 409, `{"detail":"already active"}`
						}
						return &http.Response{StatusCode: code, Request: req, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(body))}, nil
					})
				}, httpc.WithDiagnostics())
				require.NoError(t, err)
				defer shared.Close()
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(shared.Client()), camundav810.WithRequestEditorFn(func(_ context.Context, req *http.Request) error {
					req.Header.Set("Authorization", "Bearer resumption-test-secret")
					return nil
				}))
				require.NoError(t, err)
				svc := &Service{cc: client, log: log}
				got, err := svc.submitProcessInstanceResumption(shared.ToContext(context.Background()), resumptionTestKey)
				require.ErrorIs(t, err, d.ErrConflict)
				require.Equal(t, 2, calls)
				require.Equal(t, d.ProcessInstanceResumptionSubmissionRejected, got.Submission)
				require.Contains(t, got.Error, "https://fixture.invalid/v2/process-instances/"+resumptionTestKey+"/resumption")
				require.NotContains(t, output.String(), "resumption-test-secret")
				require.NotContains(t, output.String(), "retrying")
				if level == "info" {
					require.Empty(t, output.String())
					return
				}
				require.Equal(t, 2, strings.Count(output.String(), "api #"))
				require.Contains(t, output.String(), "POST /v2/process-instances/"+resumptionTestKey+"/resumption")
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

// TestProcessInstanceResumptionSubmissionLastStatus retains the last usable HTTP
// status across later transport failures or malformed metadata without accepting them.
func TestProcessInstanceResumptionSubmissionLastStatus(t *testing.T) {
	t.Parallel()
	for _, malformed := range []bool{false, true} {
		t.Run(fmt.Sprint(malformed), func(t *testing.T) {
			calls := 0
			svc := &Service{cc: resumptionSubmissionClient{t: t, submit: func(context.Context) (*camundav810.ResumeProcessInstanceResponse, error) {
				calls++
				if calls == 1 {
					return resumptionResponse(429, ""), nil
				}
				if malformed {
					return &camundav810.ResumeProcessInstanceResponse{HTTPResponse: &http.Response{StatusCode: 204}}, nil
				}
				return nil, io.ErrUnexpectedEOF
			}}}
			got, err := svc.submitProcessInstanceResumption(context.Background(), resumptionTestKey)
			if malformed {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
			} else {
				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
			}
			require.Equal(t, 2, calls)
			require.Equal(t, 429, got.StatusCode)
			require.Equal(t, d.ProcessInstanceResumptionSubmissionUnknown, got.Submission)
			require.Equal(t, d.ProcessInstanceResumptionStatusFailed, got.Status)
		})
	}
}
