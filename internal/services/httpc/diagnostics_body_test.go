// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsBodyEarlyClosePreservesIncompleteEvidence verifies that closing is delegated without draining or claiming success.
func TestAPIDiagnosticsBodyEarlyClosePreservesIncompleteEvidence(t *testing.T) {
	var output bytes.Buffer
	delegate := &scriptedDiagnosticBody{steps: []diagnosticBodyStep{{value: "unread"}}}
	response := diagnosticBodyResponse(t, &output, delegate)

	require.NoError(t, response.Body.Close())
	require.Equal(t, int32(0), delegate.reads.Load(), "diagnostics must not drain an early-closed body")
	require.Equal(t, int32(1), delegate.closes.Load())
	require.Contains(t, output.String(), "response-bytes=0 response-complete=false")
	require.NotContains(t, output.String(), "error=")
	require.Equal(t, 1, strings.Count(output.String(), "api #"))
}

// TestAPIDiagnosticsBodyPreservesReadAndCloseFailures verifies partial byte counts and bounded body failure classifications.
func TestAPIDiagnosticsBodyPreservesReadAndCloseFailures(t *testing.T) {
	readErr := errors.New("private read payload")
	closeErr := errors.New("private close payload")
	tests := []struct {
		name       string
		delegate   *scriptedDiagnosticBody
		read       bool
		wantBytes  string
		wantReason string
		wantErr    error
	}{
		{
			name:       "bytes with read error",
			delegate:   &scriptedDiagnosticBody{steps: []diagnosticBodyStep{{value: "part", err: readErr}}},
			read:       true,
			wantBytes:  "response-bytes=4 response-complete=false",
			wantReason: "reason=read-failed",
			wantErr:    readErr,
		},
		{
			name:       "close error",
			delegate:   &scriptedDiagnosticBody{closeErr: closeErr},
			wantBytes:  "response-bytes=0 response-complete=false",
			wantReason: "reason=close-failed",
			wantErr:    closeErr,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			response := diagnosticBodyResponse(t, &output, test.delegate)
			if test.read {
				buffer := make([]byte, 8)
				n, err := response.Body.Read(buffer)
				require.Equal(t, 4, n)
				require.Equal(t, "part", string(buffer[:n]))
				require.ErrorIs(t, err, test.wantErr)
			} else {
				require.ErrorIs(t, response.Body.Close(), test.wantErr)
			}

			line := output.String()
			require.Contains(t, line, "status=200 error=BODY_ERROR")
			require.Contains(t, line, "phase=body "+test.wantReason)
			require.Contains(t, line, test.wantBytes)
			require.NotContains(t, line, "private ")
			require.Equal(t, 1, strings.Count(line, "api #"))
		})
	}
}

// TestAPIDiagnosticsBodyEOFThenRepeatedCloseFreezesCompletion verifies later closes preserve delegate behavior without changing or duplicating the record.
func TestAPIDiagnosticsBodyEOFThenRepeatedCloseFreezesCompletion(t *testing.T) {
	var output bytes.Buffer
	secondCloseErr := errors.New("late private close payload")
	delegate := &scriptedDiagnosticBody{
		steps:       []diagnosticBodyStep{{value: "done", err: io.EOF}},
		closeErrors: []error{nil, secondCloseErr},
	}
	response := diagnosticBodyResponse(t, &output, delegate)

	buffer := make([]byte, 8)
	n, err := response.Body.Read(buffer)
	require.Equal(t, 4, n)
	require.ErrorIs(t, err, io.EOF)
	require.NoError(t, response.Body.Close())
	require.ErrorIs(t, response.Body.Close(), secondCloseErr)
	require.Equal(t, int32(2), delegate.closes.Load(), "every caller close must reach the delegate")
	require.Contains(t, output.String(), "response-bytes=4 response-complete=true")
	require.NotContains(t, output.String(), "error=")
	require.NotContains(t, output.String(), "late private")
	require.Equal(t, 1, strings.Count(output.String(), "api #"))
}

// TestAPIDiagnosticsBodyCountsDecompressedBytes verifies observed response size follows the body presented to the caller, not wire length.
func TestAPIDiagnosticsBodyCountsDecompressedBytes(t *testing.T) {
	const payload = "decoded response payload decoded response payload"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Content-Encoding", "gzip")
		compressed := gzip.NewWriter(writer)
		_, _ = io.WriteString(compressed, payload)
		_ = compressed.Close()
	}))
	defer server.Close()

	var output bytes.Buffer
	client := &http.Client{Transport: &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output)}}
	response, err := client.Get(server.URL)
	require.NoError(t, err)
	content, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.NoError(t, response.Body.Close())
	require.Equal(t, payload, string(content))
	require.Contains(t, output.String(), "response-bytes="+strconv.Itoa(len(payload))+" response-complete=true")
}

// TestAPIDiagnosticsBodyInterruptedUploadPreservesPartialRequestEvidence verifies request close is not treated as upload completion.
func TestAPIDiagnosticsBodyInterruptedUploadPreservesPartialRequestEvidence(t *testing.T) {
	var output bytes.Buffer
	uploadErr := errors.New("private upload payload")
	requestBody := &scriptedDiagnosticBody{steps: []diagnosticBodyStep{{value: "part", err: uploadErr}}}
	transport := &DiagnosticsTransport{
		collector: newAPIDiagnosticCollector(t, &output),
		base: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			buffer := make([]byte, 8)
			n, err := request.Body.Read(buffer)
			require.Equal(t, 4, n)
			require.Equal(t, "part", string(buffer[:n]))
			require.ErrorIs(t, err, uploadErr)
			require.NoError(t, request.Body.Close())
			return nil, uploadErr
		}),
	}
	request, err := http.NewRequest(http.MethodPost, "https://camunda.example.test/upload", requestBody)
	require.NoError(t, err)

	response, err := transport.RoundTrip(request)
	require.Nil(t, response)
	require.ErrorIs(t, err, uploadErr)
	require.Equal(t, int32(1), requestBody.reads.Load())
	require.Equal(t, int32(1), requestBody.closes.Load())
	require.Contains(t, output.String(), "error=TRANSPORT_ERROR")
	require.Contains(t, output.String(), "phase=request-write")
	require.Contains(t, output.String(), "request-bytes=4 request-complete=false")
	require.NotContains(t, output.String(), "private upload")
}

// TestAPIDiagnosticsFailurePhaseRequiresUnambiguousEvidence verifies typed and trace-derived phases without last-callback guessing.
func TestAPIDiagnosticsFailurePhaseRequiresUnambiguousEvidence(t *testing.T) {
	tests := []struct {
		name      string
		failure   error
		trace     func(*httptrace.ClientTrace)
		wantPhase string
	}{
		{
			name:      "typed DNS error",
			failure:   &net.DNSError{Err: "private lookup detail", Name: "private.example"},
			wantPhase: "phase=dns",
		},
		{
			name:    "successful write awaiting headers",
			failure: context.DeadlineExceeded,
			trace: func(trace *httptrace.ClientTrace) {
				trace.WroteRequest(httptrace.WroteRequestInfo{})
			},
			wantPhase: "phase=response-headers",
		},
		{
			name:    "overlapping setup phases are ambiguous",
			failure: context.DeadlineExceeded,
			trace: func(trace *httptrace.ClientTrace) {
				trace.DNSStart(httptrace.DNSStartInfo{})
				trace.TLSHandshakeStart()
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			transport := &DiagnosticsTransport{
				collector: newAPIDiagnosticCollector(t, &output),
				base: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					if test.trace != nil {
						test.trace(httptrace.ContextClientTrace(request.Context()))
					}
					return nil, test.failure
				}),
			}
			request, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/failure", nil)
			require.NoError(t, err)

			response, err := transport.RoundTrip(request)
			require.Nil(t, response)
			require.ErrorIs(t, err, test.failure)
			if test.wantPhase == "" {
				require.NotContains(t, output.String(), "phase=")
			} else {
				require.Contains(t, output.String(), test.wantPhase)
			}
			require.NotContains(t, output.String(), "private")
		})
	}
}

// TestAPIDiagnosticsElapsedRequestDeadlineClassifiesTimeout verifies timeout
// classification does not depend on a race with context cancellation state.
func TestAPIDiagnosticsElapsedRequestDeadlineClassifiesTimeout(t *testing.T) {
	var output bytes.Buffer
	base := time.Now()
	times := []time.Time{base, base.Add(2 * time.Second)}
	transport := &DiagnosticsTransport{
		collector: newAPIDiagnosticCollector(t, &output),
		base:      roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("private transport error") }),
		now: func() time.Time {
			value := times[0]
			times = times[1:]
			return value
		},
	}
	ctx, cancel := context.WithDeadline(context.Background(), base.Add(time.Second))
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://camunda.example.test/timeout", nil)
	require.NoError(t, err)

	response, err := transport.RoundTrip(request)
	require.Nil(t, response)
	require.Error(t, err)
	require.Contains(t, output.String(), "error=TIMEOUT")
	require.NotContains(t, output.String(), "private")
}

// TestAPIDiagnosticsBodyClassifiesContextFailuresAtObservedBoundary verifies before-header and partial-body context failures retain only available evidence.
func TestAPIDiagnosticsBodyClassifiesContextFailuresAtObservedBoundary(t *testing.T) {
	tests := []struct {
		name        string
		failure     error
		wantFailure string
	}{
		{name: "timeout", failure: context.DeadlineExceeded, wantFailure: "TIMEOUT"},
		{name: "cancellation", failure: context.Canceled, wantFailure: "CANCELED"},
	}
	for _, test := range tests {
		t.Run(test.name+" before headers", func(t *testing.T) {
			var output bytes.Buffer
			transport := &DiagnosticsTransport{
				collector: newAPIDiagnosticCollector(t, &output),
				base:      roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, test.failure }),
			}
			request, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/wait", nil)
			require.NoError(t, err)
			response, err := transport.RoundTrip(request)
			require.Nil(t, response)
			require.ErrorIs(t, err, test.failure)
			require.Contains(t, output.String(), "error="+test.wantFailure)
			require.NotContains(t, output.String(), "status=")
			require.NotContains(t, output.String(), "headers=")
			require.NotContains(t, output.String(), "body=")
			require.NotContains(t, output.String(), "phase=")
		})

		t.Run(test.name+" during body", func(t *testing.T) {
			var output bytes.Buffer
			response := diagnosticBodyResponse(t, &output, &scriptedDiagnosticBody{
				steps: []diagnosticBodyStep{{value: "part", err: test.failure}},
			})
			buffer := make([]byte, 8)
			n, err := response.Body.Read(buffer)
			require.Equal(t, 4, n)
			require.ErrorIs(t, err, test.failure)
			require.Contains(t, output.String(), "status=200 error="+test.wantFailure)
			require.Contains(t, output.String(), "phase=body")
			require.Contains(t, output.String(), "response-bytes=4 response-complete=false")
		})
	}
}

// TestAPIDiagnosticsBodyAbandonedHasNoSyntheticTerminalEvent verifies diagnostics add no read, close, timer or fabricated record.
func TestAPIDiagnosticsBodyAbandonedHasNoSyntheticTerminalEvent(t *testing.T) {
	var output bytes.Buffer
	delegate := &scriptedDiagnosticBody{steps: []diagnosticBodyStep{{value: "unread"}}}
	response := diagnosticBodyResponse(t, &output, delegate)

	require.NotNil(t, response.Body)
	require.Equal(t, int32(0), delegate.reads.Load())
	require.Equal(t, int32(0), delegate.closes.Load())
	require.Empty(t, output.String())
}

type diagnosticBodyStep struct {
	value string
	err   error
}

type scriptedDiagnosticBody struct {
	steps       []diagnosticBodyStep
	closeErr    error
	closeErrors []error
	reads       atomic.Int32
	closes      atomic.Int32
}

// Read returns the next scripted result without introducing diagnostic-side reads.
func (body *scriptedDiagnosticBody) Read(buffer []byte) (int, error) {
	body.reads.Add(1)
	if len(body.steps) == 0 {
		return 0, io.EOF
	}
	step := body.steps[0]
	body.steps = body.steps[1:]
	return copy(buffer, step.value), step.err
}

// Close returns its scripted result so wrapper behavior can be compared exactly.
func (body *scriptedDiagnosticBody) Close() error {
	call := int(body.closes.Add(1)) - 1
	if call < len(body.closeErrors) {
		return body.closeErrors[call]
	}
	return body.closeErr
}

// diagnosticBodyResponse returns a response wrapped by the real diagnostics transport.
func diagnosticBodyResponse(t *testing.T, output *bytes.Buffer, body io.ReadCloser) *http.Response {
	t.Helper()
	transport := &DiagnosticsTransport{
		collector: newAPIDiagnosticCollector(t, output),
		base: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body, Request: request}, nil
		}),
	}
	request, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/body", nil)
	require.NoError(t, err)
	response, err := transport.RoundTrip(request)
	require.NoError(t, err)
	return response
}
