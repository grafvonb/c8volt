// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"net/textproto"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

func newAPIDiagnosticCollector(t *testing.T, output *bytes.Buffer) *diagnosticCollector {
	t.Helper()
	collector := newDiagnosticCollector(config.New(), logging.New(logging.LoggerConfig{Writer: output, Level: "info", Format: "plain"}), true)
	require.NotNil(t, collector)
	return collector
}

// TestAPIDiagnosticsTransportSeparatesFinalHeadersAndBody verifies terminal timing and exact delegated reads.
func TestAPIDiagnosticsTransportSeparatesFinalHeadersAndBody(t *testing.T) {
	var output bytes.Buffer
	var reads, closes atomic.Int32
	body := &countingTestBody{reader: strings.NewReader("payload"), reads: &reads, closes: &closes}
	transport := &DiagnosticsTransport{
		collector: newAPIDiagnosticCollector(t, &output),
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			time.Sleep(15 * time.Millisecond)
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body, Request: req}, nil
		}),
	}
	req, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/v2/topology", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	time.Sleep(10 * time.Millisecond)
	content, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "payload", string(content))
	require.NoError(t, resp.Body.Close())
	require.Equal(t, int32(2), reads.Load(), "diagnostics must not add body reads")
	require.Equal(t, int32(1), closes.Load(), "diagnostics must not add body closes")

	line := output.String()
	require.Contains(t, line, "api #1 GET /v2/topology: status=200")
	require.Contains(t, line, " headers=")
	require.Contains(t, line, " body=")
	require.Contains(t, line, "response-bytes=7 response-complete=true")
	require.Equal(t, 1, strings.Count(line, "api #"))
	total := diagnosticDurationField(t, line, "total")
	headers := diagnosticDurationField(t, line, "headers")
	responseBody := diagnosticDurationField(t, line, "body")
	require.Equal(t, total, headers+responseBody, "total must be derived from the same header/body boundaries before formatting")
}

// TestAPIDiagnosticsTransportPreservesTraceAndInformationalResponses verifies existing callbacks and final-header semantics.
func TestAPIDiagnosticsTransportPreservesTraceAndInformationalResponses(t *testing.T) {
	var output bytes.Buffer
	var informational atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok")
	}))
	defer server.Close()

	cfg := config.New()
	cfg.HTTP.Timeout = "1s"
	service, err := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), WithDiagnostics(true))
	require.NoError(t, err)
	// Replace the gated-away collector with an admitted one while retaining the real base transport.
	logTransport := unwrapLogTransport(service.Client().Transport)
	require.NotNil(t, logTransport)
	logTransport.base = &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output), base: http.DefaultTransport}

	trace := &httptrace.ClientTrace{Got1xxResponse: func(code int, _ textproto.MIMEHeader) error {
		if code == http.StatusEarlyHints {
			informational.Add(1)
		}
		return nil
	}}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(context.Background(), trace), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	resp, err := service.Client().Do(req)
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, int32(1), informational.Load())
	require.Contains(t, output.String(), "status=200")
	require.NotContains(t, output.String(), "status=103")
}

// TestAPIDiagnosticsTransportHandlesBodylessEmptyAndReadWithEOF covers completion evidence without inferred lengths.
func TestAPIDiagnosticsTransportHandlesBodylessEmptyAndReadWithEOF(t *testing.T) {
	tests := []struct {
		name   string
		method string
		status int
		body   io.ReadCloser
		read   bool
		want   string
	}{
		{name: "head", method: http.MethodHead, status: http.StatusOK, body: http.NoBody, want: "body=0s"},
		{name: "no content", method: http.MethodGet, status: http.StatusNoContent, body: http.NoBody, want: "body=0s"},
		{name: "empty observed", method: http.MethodGet, status: http.StatusOK, body: io.NopCloser(strings.NewReader("")), read: true, want: "response-bytes=0 response-complete=true"},
		{name: "bytes with eof", method: http.MethodGet, status: http.StatusOK, body: &singleReadBody{value: []byte("abc")}, read: true, want: "response-bytes=3 response-complete=true"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var output bytes.Buffer
			transport := &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output), base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Header: make(http.Header), Body: test.body, Request: req}, nil
			})}
			req, err := http.NewRequest(test.method, "https://camunda.example.test/resource", nil)
			require.NoError(t, err)
			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			if test.read {
				_, err = io.ReadAll(resp.Body)
				require.NoError(t, err)
			}
			require.Contains(t, output.String(), test.want)
			require.Contains(t, output.String(), "response-bytes=")
			require.Contains(t, output.String(), "response-complete=true")
		})
	}
}

// TestAPIDiagnosticsTransportFailureAndDisabledBehavior verifies no fabricated response evidence or disabled wrapping.
func TestAPIDiagnosticsTransportFailureAndDisabledBehavior(t *testing.T) {
	var output bytes.Buffer
	wantErr := errors.New("private transport detail")
	var calls atomic.Int32
	base := roundTripFunc(func(*http.Request) (*http.Response, error) { calls.Add(1); return nil, wantErr })
	transport := &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output), base: base}
	req, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/fail", nil)
	require.NoError(t, err)
	resp, err := transport.RoundTrip(req)
	require.ErrorIs(t, err, wantErr)
	require.Nil(t, resp)
	require.Equal(t, int32(1), calls.Load())
	require.Contains(t, output.String(), "error=TRANSPORT_ERROR total=")
	require.NotContains(t, output.String(), "status=")
	require.NotContains(t, output.String(), "headers=")
	require.NotContains(t, output.String(), "private transport detail")

	output.Reset()
	disabled := &DiagnosticsTransport{base: base}
	_, _ = disabled.RoundTrip(req)
	require.Empty(t, output.String())
	require.Equal(t, int32(2), calls.Load())
}

// TestAPIDiagnosticsTransportRedirectsUseSeparateSequences verifies standard redirect behavior without extra requests.
func TestAPIDiagnosticsTransportRedirectsUseSeparateSequences(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests.Add(1)
		if req.URL.Path == "/start" {
			http.Redirect(w, req, "/final", http.StatusFound)
			return
		}
		_, _ = io.WriteString(w, "done")
	}))
	defer server.Close()
	var output bytes.Buffer
	client := &http.Client{Transport: &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output)}}
	resp, err := client.Get(server.URL + "/start")
	require.NoError(t, err)
	_, err = io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.NoError(t, resp.Body.Close())
	require.Equal(t, int32(2), requests.Load())
	require.Equal(t, 2, strings.Count(output.String(), "api #"))
	require.Contains(t, output.String(), "api #1 GET /start: status=302")
	require.Contains(t, output.String(), "api #2 GET /final: status=200")
}

// TestAPIDiagnosticsTransportPreservesRequestReplayAndWriterTo verifies request ownership and optional fast paths.
func TestAPIDiagnosticsTransportPreservesRequestReplayAndWriterTo(t *testing.T) {
	var output bytes.Buffer
	requestBody := &writerToTestBody{value: "request-data"}
	transport := &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output), base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.NotNil(t, req.GetBody)
		replay, err := req.GetBody()
		require.NoError(t, err)
		replayed, err := io.ReadAll(replay)
		require.NoError(t, err)
		require.Equal(t, "request-data", string(replayed))
		writerTo, ok := req.Body.(io.WriterTo)
		require.True(t, ok)
		var sent bytes.Buffer
		n, err := writerTo.WriteTo(&sent)
		require.NoError(t, err)
		require.Equal(t, int64(len("request-data")), n)
		require.Equal(t, "request-data", sent.String())
		require.NoError(t, req.Body.Close())
		return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody, Request: req}, nil
	})}
	req, err := http.NewRequest(http.MethodPost, "https://camunda.example.test/v2/items", strings.NewReader("request-data"))
	require.NoError(t, err)
	req.Body = requestBody
	req.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(strings.NewReader("request-data")), nil }
	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusNoContent, resp.StatusCode)
	require.Equal(t, int32(1), requestBody.closes.Load())
	require.Contains(t, output.String(), "request-bytes=12 request-complete=true")
}

// TestAPIDiagnosticsServiceStackPlacesObservationBelowRetriesAndLogging verifies shared collector discovery through known wrappers.
func TestAPIDiagnosticsServiceStackPlacesObservationBelowRetriesAndLogging(t *testing.T) {
	var output bytes.Buffer
	cfg := config.New()
	cfg.HTTP.Timeout = "1s"
	service, err := New(cfg, logging.New(logging.LoggerConfig{Writer: &output, Level: "info", Format: "plain"}), WithDiagnostics(true))
	require.NoError(t, err)
	retry, ok := service.Client().Transport.(*ReadRetryTransport)
	require.True(t, ok)
	logTransport, ok := retry.base.(*LogTransport)
	require.True(t, ok)
	diagnostics, ok := logTransport.base.(*DiagnosticsTransport)
	require.True(t, ok)
	require.Same(t, diagnostics.collector, diagnosticCollectorFromTransport(service.Client().Transport))
	require.Same(t, http.DefaultTransport, diagnostics.rt())
}

func diagnosticDurationField(t *testing.T, line, name string) time.Duration {
	t.Helper()
	for _, field := range strings.Fields(line) {
		if strings.HasPrefix(field, name+"=") {
			value := strings.TrimPrefix(field, name+"=")
			duration, err := time.ParseDuration(value)
			require.NoError(t, err)
			return duration
		}
	}
	require.FailNow(t, "missing duration field", name)
	return 0
}

type countingTestBody struct {
	reader        *strings.Reader
	reads, closes *atomic.Int32
}

func (body *countingTestBody) Read(buffer []byte) (int, error) {
	body.reads.Add(1)
	return body.reader.Read(buffer)
}
func (body *countingTestBody) Close() error { body.closes.Add(1); return nil }

type singleReadBody struct {
	value []byte
	read  bool
}

func (body *singleReadBody) Read(buffer []byte) (int, error) {
	if body.read {
		return 0, io.EOF
	}
	body.read = true
	return copy(buffer, body.value), io.EOF
}
func (*singleReadBody) Close() error { return nil }

type writerToTestBody struct {
	value  string
	closes atomic.Int32
}

func (body *writerToTestBody) Read(buffer []byte) (int, error) {
	if body.value == "" {
		return 0, io.EOF
	}
	n := copy(buffer, body.value)
	body.value = body.value[n:]
	return n, nil
}

func (body *writerToTestBody) WriteTo(writer io.Writer) (int64, error) {
	n, err := io.WriteString(writer, body.value)
	body.value = body.value[n:]
	return int64(n), err
}

func (body *writerToTestBody) Close() error { body.closes.Add(1); return nil }
