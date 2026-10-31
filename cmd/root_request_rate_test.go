// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// requestRateServiceStub verifies singleton ownership without starting a reporter.
type requestRateServiceStub struct {
	close func()
	pause func() func()
}

// Close records consumption of the invocation's cleanup ownership.
func (s *requestRateServiceStub) Close() {
	if s.close != nil {
		s.close()
	}
}

// PauseRequestRateReporting records manual input boundaries when a test needs them.
func (s *requestRateServiceStub) PauseRequestRateReporting() func() {
	if s.pause != nil {
		return s.pause()
	}
	return func() {}
}

// requestRateLogCapture synchronizes asynchronous logger output and test-controlled response release.
type requestRateLogCapture struct {
	mu     sync.Mutex
	output strings.Builder
	report chan struct{}
}

// Write signals after recording a report so the fake can finish without racing a timer.
func (w *requestRateLogCapture) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, err := w.output.Write(p)
	if strings.Contains(string(p), "HTTP request rate:") && w.report != nil {
		select {
		case w.report <- struct{}{}:
		default:
		}
	}
	return n, err
}

// String returns a stable snapshot while the invocation may still be running.
func (w *requestRateLogCapture) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.output.String()
}

// appendRequestRateConfig adds settings to a test-owned config that has no HTTP block.
func appendRequestRateConfig(t *testing.T, path, interval string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	require.NoError(t, err)
	_, err = fmt.Fprintf(file, "http:\n  request_rate_report_interval: %q\n", interval)
	require.NoError(t, err)
	require.NoError(t, file.Close())
}

// requireRequestRateSummaries verifies complete safe messages, logger framing and measured precision.
func requireRequestRateSummaries(t *testing.T, output, format string) int {
	t.Helper()
	pattern := regexp.MustCompile(`^HTTP request rate: ([0-9]+) requests in ([^ ]+) \(([0-9]+\.[0-9]{2}) requests/s\)$`)
	count := 0
	for line := range strings.SplitSeq(output, "\n") {
		if !strings.Contains(line, "HTTP request rate:") {
			continue
		}
		count++
		var message string
		if format == "json" {
			var record map[string]any
			require.NoError(t, json.Unmarshal([]byte(line), &record))
			require.Equal(t, "DEBUG", record["level"])
			message = record["msg"].(string)
		} else {
			require.Contains(t, line, "DEBUG HTTP request rate:")
			_, message, _ = strings.Cut(line, "DEBUG ")
		}
		match := pattern.FindStringSubmatch(message)
		require.Len(t, match, 4, "unexpected summary: %s", message)
		starts, err := strconv.ParseUint(match[1], 10, 64)
		require.NoError(t, err)
		elapsed, err := time.ParseDuration(match[2])
		require.NoError(t, err)
		require.Positive(t, elapsed)
		measured, err := strconv.ParseFloat(match[3], 64)
		require.NoError(t, err)
		require.InDelta(t, float64(starts)/elapsed.Seconds(), measured, 0.0051)
	}
	require.Positive(t, count, "expected a rate report: %s", output)
	return count
}

// TestRequestRateCommandReporting exercises effective logger enablement, both destinations,
// complete summary framing and cleanup through real returned ExecuteC invocations.
func TestRequestRateCommandReporting(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, tc := range []struct {
		name, format, interval string
		flags                  []string
		configured, enabled    bool
	}{
		{name: "flag inherited plain", format: "plain", interval: "2ms", flags: []string{"--debug"}, enabled: true},
		{name: "flag configured json", format: "json", interval: "2ms", flags: []string{"--debug"}, configured: true, enabled: true},
		{name: "configured level", format: "plain", interval: "2ms", configured: true, enabled: true},
		{name: "verbose", format: "plain", interval: "2ms", flags: []string{"--verbose"}},
		{name: "quiet debug", format: "json", interval: "2ms", flags: []string{"--debug", "--quiet"}},
		{name: "quiet verbose debug", format: "plain", interval: "2ms", flags: []string{"--debug", "--quiet", "--verbose"}},
		{name: "zero interval", format: "plain", interval: "0s", flags: []string{"--debug"}},
		{name: "short command", format: "plain", interval: "1h", flags: []string{"--debug"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			capture := &requestRateLogCapture{report: make(chan struct{}, 10)}
			var calls atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if tc.enabled {
					select {
					case <-capture.report:
					case <-time.After(2 * time.Second):
						t.Error("no rate summary during request")
					}
				} else {
					time.Sleep(10 * time.Millisecond)
				}
				if serveVersionCheckTopology(w, r, "8.9") {
					return
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, apiDiagnosticsProcessDefinitionFixture)
			}))
			defer server.Close()
			path := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")
			appendRequestRateConfig(t, path, tc.interval)
			if tc.name == "configured level" {
				file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
				require.NoError(t, err)
				_, err = io.WriteString(file, "log:\n  level: debug\n")
				require.NoError(t, err)
				require.NoError(t, file.Close())
			}
			root := Root()
			resetCommandTreeFlags(root)
			resetGetProcessDefinitionCommandGlobals()
			resetRequestRateCommandContexts(root)
			defer func() {
				getProcessDefinitionCmd.SetErr(nil)
				resetCommandTreeFlags(root)
				resetRequestRateCommandContexts(root)
			}()
			var stdout bytes.Buffer
			unused := &requestRateLogCapture{}
			root.SetOut(&stdout)
			root.SetErr(capture)
			getProcessDefinitionCmd.SetErr(nil)
			if tc.configured {
				root.SetErr(unused)
				getProcessDefinitionCmd.SetErr(capture)
			}
			args := append([]string{"--config", path, "--log-format", tc.format, "--json"}, tc.flags...)
			args = append(args, "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey)
			root.SetArgs(args)
			_, err := root.ExecuteC()
			require.NoError(t, err)
			require.Nil(t, currentHTTPService, "finalizer must consume ownership")
			require.Equal(t, int32(2), calls.Load())
			require.Empty(t, unused.String())
			requireSingleJSONValue(t, stdout.String())
			require.NotContains(t, stdout.String(), "HTTP request rate:")
			output := capture.String()
			if tc.enabled {
				requireRequestRateSummaries(t, output, tc.format)
				require.Equal(t, 2, strings.Count(output, "1 requests in"), "probe and operation must each be reported exactly once")
			} else {
				require.NotContains(t, output, "HTTP request rate:")
			}
			if tc.name == "zero interval" {
				require.Contains(t, output, "api #1", "disabling summaries must retain existing DEBUG diagnostics")
			}
			time.Sleep(8 * time.Millisecond)
			require.Equal(t, output, capture.String(), "reporter survived ExecuteC")
		})
	}
}

// TestRequestRateInvocationCleanup covers returned execution/bootstrap failures,
// stale ownership, repeated invocations and a command shorter than its interval.
func TestRequestRateInvocationCleanup(t *testing.T) {
	isolateRequestRateEnvironment(t)
	root := Root()
	sentinel := errors.New("test returned error")
	command := &cobra.Command{Use: "request-rate-lifecycle-test", RunE: func(*cobra.Command, []string) error { return nil }}
	root.AddCommand(command)
	defer root.RemoveCommand(command)
	var requests atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(http.StatusUnauthorized) }))
	defer server.Close()
	for _, mode := range []string{"success", "error", "bootstrap", "cancelled"} {
		t.Run(mode, func(t *testing.T) {
			resetCommandTreeFlags(root)
			resetRequestRateCommandContexts(root)
			defer func() { resetCommandTreeFlags(root); resetRequestRateCommandContexts(root) }()
			auth := "none"
			if mode == "bootstrap" {
				auth = "cookie"
			}
			path := writeAPIDiagnosticsCommandConfig(t, server.URL, auth)
			appendRequestRateConfig(t, path, "1h")
			var staleClosed int
			ownInvocationHTTPService(&requestRateServiceStub{close: func() { staleClosed++ }})
			command.RunE = func(cmd *cobra.Command, _ []string) error {
				require.NotNil(t, currentHTTPService)
				if mode == "error" {
					return sentinel
				}
				return nil
			}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "cancelled" {
				cancel()
			}
			root.SetContext(ctx)
			capture := &requestRateLogCapture{}
			root.SetErr(capture)
			root.SetOut(io.Discard)
			root.SetArgs([]string{"--config", path, "--debug", command.Name()})
			started := time.Now()
			_, err := root.ExecuteC()
			require.Less(t, time.Since(started), time.Second)
			if mode == "error" {
				require.ErrorIs(t, err, sentinel)
			} else if mode == "bootstrap" {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, 1, staleClosed)
			require.Nil(t, currentHTTPService)
			closeInvocationHTTPService()
			require.Equal(t, 1, staleClosed)
			require.NotContains(t, capture.String(), "HTTP request rate:")
		})
	}
	require.Equal(t, int32(1), requests.Load(), "only cookie initialization should reach the fake")
}
