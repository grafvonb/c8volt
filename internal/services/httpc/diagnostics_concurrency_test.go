// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsTraceMatchesOverlappingAttempts verifies phase pairing, start ordering, callback composition and frozen late events.
func TestAPIDiagnosticsTraceMatchesOverlappingAttempts(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	clock := newDiagnosticTestClock(
		time.Unix(0, 0),
		time.Unix(0, int64(10*time.Millisecond)),
		time.Unix(0, int64(20*time.Millisecond)),
		time.Unix(0, int64(25*time.Millisecond)),
		time.Unix(0, int64(30*time.Millisecond)),
		time.Unix(0, int64(35*time.Millisecond)),
		time.Unix(0, int64(40*time.Millisecond)),
		time.Unix(0, int64(50*time.Millisecond)),
		time.Unix(0, int64(60*time.Millisecond)),
		time.Unix(0, int64(70*time.Millisecond)),
		time.Unix(0, int64(80*time.Millisecond)),
	)
	var existingStarts, existingDone, existingGotConn atomic.Int32
	existing := &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) { existingStarts.Add(1) },
		ConnectDone:  func(_, _ string, _ error) { existingDone.Add(1) },
		GotConn:      func(httptrace.GotConnInfo) { existingGotConn.Add(1) },
	}
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		trace := httptrace.ContextClientTrace(req.Context())
		require.NotNil(t, trace)
		trace.ConnectStart("tcp", "first")
		trace.ConnectStart("tcp", "second")
		trace.ConnectStart("udp", "ignored")
		trace.ConnectDone("tcp", "second", nil)
		trace.ConnectDone("udp", "ignored", nil)
		trace.ConnectStart("tcp", "unfinished")
		trace.ConnectDone("tcp", "first", nil)
		trace.GotConn(httptrace.GotConnInfo{Reused: false})
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok")), Request: req}, nil
	})
	transport := &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output), base: base, now: clock.Now}
	req, err := http.NewRequestWithContext(httptrace.WithClientTrace(t.Context(), existing), http.MethodGet, "https://camunda.example.test/v2/topology", nil)
	require.NoError(t, err)
	response, err := transport.RoundTrip(req)
	require.NoError(t, err)
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)

	require.Equal(t, int32(4), existingStarts.Load())
	require.Equal(t, int32(3), existingDone.Load())
	require.Equal(t, int32(1), existingGotConn.Load())
	line := output.String()
	require.Contains(t, line, "conn=new")
	require.Contains(t, line, "tcp=[40ms,10ms]")
	require.Equal(t, 1, strings.Count(line, "api #"))

	trace := httptrace.ContextClientTrace(response.Request.Context())
	trace.ConnectDone("tcp", "unfinished", nil)
	require.Equal(t, line, output.String(), "a late callback must not change or duplicate the frozen record")
}

// TestAPIDiagnosticsTraceOmitsUnperformedReusePhases verifies reuse evidence does not fabricate setup timings.
func TestAPIDiagnosticsTraceOmitsUnperformedReusePhases(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	transport := &DiagnosticsTransport{
		collector: newAPIDiagnosticCollector(t, &output),
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			httptrace.ContextClientTrace(req.Context()).GotConn(httptrace.GotConnInfo{Reused: true})
			return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: http.NoBody, Request: req}, nil
		}),
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://camunda.example.test/v2/topology", nil)
	require.NoError(t, err)
	_, err = transport.RoundTrip(req)
	require.NoError(t, err)

	line := output.String()
	require.Contains(t, line, "conn=reused")
	require.NotContains(t, line, " dns=")
	require.NotContains(t, line, " tcp=")
	require.NotContains(t, line, " tls=")
}

// TestAPIDiagnosticsTraceRetainsObservedZeroDurations verifies a completed zero sample differs from an absent in-progress phase.
func TestAPIDiagnosticsTraceRetainsObservedZeroDurations(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	clock := newDiagnosticTestClock(
		time.Unix(0, 0),
		time.Unix(0, int64(5*time.Millisecond)),
		time.Unix(0, int64(5*time.Millisecond)),
		time.Unix(0, int64(10*time.Millisecond)),
		time.Unix(0, int64(20*time.Millisecond)),
		time.Unix(0, int64(25*time.Millisecond)),
		time.Unix(0, int64(30*time.Millisecond)),
		time.Unix(0, int64(40*time.Millisecond)),
	)
	transport := &DiagnosticsTransport{
		collector: newAPIDiagnosticCollector(t, &output),
		now:       clock.Now,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			trace := httptrace.ContextClientTrace(req.Context())
			trace.DNSStart(httptrace.DNSStartInfo{})
			trace.DNSDone(httptrace.DNSDoneInfo{})
			trace.TLSHandshakeStart()
			trace.TLSHandshakeDone(tls.ConnectionState{}, nil)
			trace.TLSHandshakeStart()
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("ok")), Request: req}, nil
		}),
	}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://camunda.example.test/v2/topology", nil)
	require.NoError(t, err)
	response, err := transport.RoundTrip(req)
	require.NoError(t, err)
	_, err = io.ReadAll(response.Body)
	require.NoError(t, err)

	line := output.String()
	require.Contains(t, line, "dns=0s")
	require.Contains(t, line, "tls=10ms")
	require.NotContains(t, line, "tls=[", "the unpaired TLS start must not create a second sample")
}

// TestAPIDiagnosticsConcurrentCompletionKeepsRecordsIndivisible verifies sequence and line integrity when completion order differs.
func TestAPIDiagnosticsConcurrentCompletionKeepsRecordsIndivisible(t *testing.T) {
	t.Parallel()

	const exchanges = 64
	var output bytes.Buffer
	transport := &DiagnosticsTransport{
		collector: newAPIDiagnosticCollector(t, &output),
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(req.URL.Path)), Request: req}, nil
		}),
	}
	responses := make([]*http.Response, exchanges)
	for index := range exchanges {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://camunda.example.test/items/"+strconv.Itoa(index), nil)
		require.NoError(t, err)
		responses[index], err = transport.RoundTrip(req)
		require.NoError(t, err)
	}
	_, err := io.ReadAll(responses[exchanges-1].Body)
	require.NoError(t, err)
	_, err = io.ReadAll(responses[0].Body)
	require.NoError(t, err)

	var workers sync.WaitGroup
	readErrors := make([]error, exchanges)
	workers.Add(exchanges - 2)
	for index := exchanges - 2; index >= 1; index-- {
		go func(index int, response *http.Response) {
			defer workers.Done()
			_, readErrors[index] = io.ReadAll(response.Body)
		}(index, responses[index])
	}
	workers.Wait()
	for _, readErr := range readErrors {
		require.NoError(t, readErr)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	require.Len(t, lines, exchanges)
	require.Contains(t, lines[0], "api #64 ")
	require.Contains(t, lines[1], "api #1 ")
	seen := make(map[string]struct{}, exchanges)
	for _, line := range lines {
		require.Equal(t, 1, strings.Count(line, "DEBUG api #"))
		fields := strings.Fields(line)
		require.GreaterOrEqual(t, len(fields), 5)
		for _, field := range fields {
			if strings.HasPrefix(field, "#") {
				seen[field] = struct{}{}
				break
			}
		}
	}
	for sequence := 1; sequence <= exchanges; sequence++ {
		require.Contains(t, seen, "#"+strconv.Itoa(sequence))
	}
}

// TestAPIDiagnosticsConcurrentReadCloseFreezesOnce verifies simultaneous terminal observations retain one immutable line.
func TestAPIDiagnosticsConcurrentReadCloseFreezesOnce(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer
	body := &diagnosticConcurrentBody{}
	transport := &DiagnosticsTransport{collector: newAPIDiagnosticCollector(t, &output), base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body, Request: req}, nil
	})}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://camunda.example.test/v2/topology", nil)
	require.NoError(t, err)
	response, err := transport.RoundTrip(req)
	require.NoError(t, err)

	start := make(chan struct{})
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		<-start
		_, _ = response.Body.Read(make([]byte, 1))
	}()
	go func() {
		defer workers.Done()
		<-start
		_ = response.Body.Close()
	}()
	close(start)
	workers.Wait()

	require.Equal(t, 1, strings.Count(output.String(), "api #"))
	require.Contains(t, output.String(), "response-bytes=0")
}

// TestAPIDiagnosticsCollectorsAndWriterFailuresStayIsolated verifies destinations and HTTP outcomes remain invocation-owned.
func TestAPIDiagnosticsCollectorsAndWriterFailuresStayIsolated(t *testing.T) {
	t.Parallel()

	newCollector := func(profile string, writer io.Writer) *diagnosticCollector {
		cfg := config.New()
		cfg.ActiveProfile = profile
		collector := newDiagnosticCollector(cfg, logging.New(logging.LoggerConfig{Writer: writer, Level: "debug", Format: "plain"}))
		require.NotNil(t, collector)
		return collector
	}
	var firstOutput, secondOutput bytes.Buffer
	first := newCollector("first", &firstOutput)
	second := newCollector("second", &secondOutput)
	first.start(&http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/first"}}).finish(nil)
	second.start(&http.Request{Method: http.MethodGet, URL: &url.URL{Path: "/second"}}).finish(nil)
	require.Contains(t, firstOutput.String(), "api #1 GET /first: total=0s profile=first")
	require.NotContains(t, firstOutput.String(), "second")
	require.Contains(t, secondOutput.String(), "api #1 GET /second: total=0s profile=second")
	require.NotContains(t, secondOutput.String(), "first")

	wantWriteErr := errors.New("diagnostic destination unavailable")
	failing := &diagnosticFailingWriter{err: wantWriteErr}
	transport := &DiagnosticsTransport{collector: newCollector("failure", failing), base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("result")), Request: req}, nil
	})}
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://camunda.example.test/result", nil)
	require.NoError(t, err)
	response, err := transport.RoundTrip(req)
	require.NoError(t, err)
	value, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	require.Equal(t, "result", string(value))
	require.Equal(t, int32(1), failing.writes.Load(), "diagnostic writer failures are neither retried nor surfaced as HTTP failures")
}

// diagnosticTestClock supplies deterministic monotonic-compatible timestamps to trace tests.
type diagnosticTestClock struct {
	mu    sync.Mutex
	times []time.Time
	index int
}

// newDiagnosticTestClock constructs a clock that requires exactly the scripted observations.
func newDiagnosticTestClock(times ...time.Time) *diagnosticTestClock {
	return &diagnosticTestClock{times: times}
}

// Now returns the next scripted timestamp.
func (clock *diagnosticTestClock) Now() time.Time {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	if clock.index >= len(clock.times) {
		panic("diagnostic test clock exhausted")
	}
	value := clock.times[clock.index]
	clock.index++
	return value
}

// diagnosticConcurrentBody permits intentional concurrent lifecycle calls without adding delegate races.
type diagnosticConcurrentBody struct {
	mu sync.Mutex
}

// Read returns EOF under synchronization.
func (body *diagnosticConcurrentBody) Read([]byte) (int, error) {
	body.mu.Lock()
	defer body.mu.Unlock()
	return 0, io.EOF
}

// Close preserves a successful close under synchronization.
func (body *diagnosticConcurrentBody) Close() error {
	body.mu.Lock()
	defer body.mu.Unlock()
	return nil
}

// diagnosticFailingWriter records one attempted log write and returns its configured failure.
type diagnosticFailingWriter struct {
	err    error
	writes atomic.Int32
}

// Write rejects the diagnostic record without retaining it.
func (writer *diagnosticFailingWriter) Write(value []byte) (int, error) {
	writer.writes.Add(1)
	return 0, writer.err
}
