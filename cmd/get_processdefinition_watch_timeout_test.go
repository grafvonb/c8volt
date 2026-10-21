// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestGetProcessDefinitionWatchSessionTimeoutDispatch exercises actual flag and
// configuration resolution, ensuring the backoff budget cannot bound watch.
func TestGetProcessDefinitionWatchSessionTimeoutDispatch(t *testing.T) {
	for _, duration := range []string{"", "0s", "1h"} {
		t.Run("timeout="+duration, func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0}}`))
			}))
			previousSleep := processDefinitionWatchSleep
			t.Cleanup(func() { processDefinitionWatchSleep = previousSleep })
			sleeps := 0
			processDefinitionWatchSleep = func(ctx context.Context, _ time.Duration) error {
				deadline, bounded := ctx.Deadline()
				require.Equal(t, duration == "1h", bounded)
				if bounded {
					require.Greater(t, time.Until(deadline), 59*time.Minute)
				}
				sleeps++
				if sleeps == 2 {
					return context.Canceled
				}
				return nil
			}
			args := []string{"--config", writeTestConfigForVersion(t, server.URL, "8.9"), "get", "pd", "--watch", "--backoff-timeout", "1ns"}
			if duration != "" {
				args = append(args, "--watch-timeout", duration)
			}
			stdout, stderr := executeRootWithSeparateOutputsForTest(t, args...)
			require.Equal(t, 2, sleeps)
			require.Equal(t, 2, strings.Count(stdout, "found: 0"))
			require.NotContains(t, stderr, "watch stopped")
		})
	}
}

// TestGetProcessDefinitionWatchTimeoutValidation verifies public flag parsing,
// nonnegative duration validation and the explicit watch dependency.
func TestGetProcessDefinitionWatchTimeoutValidation(t *testing.T) {
	flag := getProcessDefinitionCmd.Flags().Lookup("watch-timeout")
	require.NotNil(t, flag)
	require.False(t, flag.Hidden)
	require.Equal(t, "0s", flag.DefValue)
	for _, tc := range []struct {
		raw   string
		watch bool
		valid bool
	}{
		{"0s", true, true}, {"2m", true, true}, {"-1s", true, false}, {"soon", true, false}, {"0s", false, false},
	} {
		t.Run(fmt.Sprintf("%s/%v", tc.raw, tc.watch), func(t *testing.T) {
			resetGetProcessDefinitionCommandGlobals()
			t.Cleanup(resetGetProcessDefinitionCommandGlobals)
			cmd := &cobra.Command{}
			cmd.Flags().DurationVar(&flagGetPDWatchTimeout, "watch-timeout", 0, "")
			flagGetPDWatch = tc.watch
			err := cmd.Flags().Set("watch-timeout", tc.raw)
			if err == nil {
				err = validateGetProcessDefinitionFlags(cmd)
			}
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

// TestGetProcessDefinitionWatchRequestTimeoutRecovery uses separate configured
// log and result destinations to catch direct stderr writes and raw errors.
func TestGetProcessDefinitionWatchRequestTimeoutRecovery(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		for _, verbose := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/verbose=%v", format, verbose), func(t *testing.T) {
				resetGetProcessDefinitionCommandGlobals()
				t.Cleanup(resetGetProcessDefinitionCommandGlobals)
				flagVerbose = verbose
				var stdout, stderr, diagnostics bytes.Buffer
				parent := &cobra.Command{Use: "get"}
				cmd := &cobra.Command{Use: "pd"}
				parent.AddCommand(cmd)
				parent.SetOut(&stdout)
				parent.SetErr(&stderr)
				var handler slog.Handler = logging.NewPlainHandler(&diagnostics, slog.LevelInfo)
				if format == "json" {
					handler = slog.NewJSONHandler(&diagnostics, nil)
				}
				cmd.SetContext(logging.ToContext(context.Background(), slog.New(handler)))
				calls := 0
				cli := processDefinitionWatchTestAPI{collect: func(ctx context.Context, _ process.ProcessDefinitionWatchSnapshotRequest, _ ...options.FacadeOption) (process.ProcessDefinitionWatchSnapshot, error) {
					calls++
					item := process.ProcessDefinition{Key: "123", BpmnProcessId: "invoice", ProcessVersion: 1}
					if calls == 2 || calls == 3 {
						return process.ProcessDefinitionWatchSnapshot{Items: []process.ProcessDefinition{{Key: "partial"}}}, fmt.Errorf("POST https://watch-secret.invalid/123 body=secret-body: %w", context.DeadlineExceeded)
					}
					if calls == 4 {
						item.ProcessVersion = 2
					}
					return process.ProcessDefinitionWatchSnapshot{Items: []process.ProcessDefinition{item}, Total: 1}, nil
				}}
				previousSleep := processDefinitionWatchSleep
				t.Cleanup(func() { processDefinitionWatchSleep = previousSleep })
				processDefinitionWatchSleep = func(context.Context, time.Duration) error {
					if calls == 4 {
						return context.Canceled
					}
					return nil
				}
				require.NoError(t, executeGetProcessDefinitionWatch(cmd, cli, process.ProcessDefinitionFilter{}, 0, 2))
				require.Equal(t, 4, calls)
				require.Equal(t, 2, strings.Count(stdout.String(), "found: 1"))
				require.Contains(t, stdout.String(), "v2 *")
				require.NotContains(t, stdout.String(), "partial")
				require.Equal(t, 2, strings.Count(diagnostics.String(), "retrying process-definition watch"))
				if format == "plain" {
					require.Contains(t, diagnostics.String(), "WARN retrying")
				} else {
					require.Contains(t, diagnostics.String(), `"level":"WARN"`)
				}
				require.NotContains(t, stderr.String(), "retrying")
				for _, output := range []string{stdout.String(), stderr.String(), diagnostics.String()} {
					for _, secret := range []string{"POST", "watch-secret.invalid", "secret-body", "watch stopped"} {
						require.NotContains(t, output, secret)
					}
				}
			})
		}
	}
}

// TestGetProcessDefinitionWatchHTTPRequestTimeoutRetries exercises real HTTP
// deadline conversion through the facade without contacting a real backend.
func TestGetProcessDefinitionWatchHTTPRequestTimeoutRetries(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	var requests atomic.Int32
	releaseRequest := make(chan struct{})
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		if requests.Add(1) == 1 {
			select {
			case <-r.Context().Done():
			case <-releaseRequest:
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0}}`))
	}))
	// Release the blocked handler before server cleanup even if cancellation
	// is not observed while the request body remains unread.
	t.Cleanup(func() { close(releaseRequest) })
	previousSleep := processDefinitionWatchSleep
	t.Cleanup(func() { processDefinitionWatchSleep = previousSleep })
	sleeps := 0
	processDefinitionWatchSleep = func(context.Context, time.Duration) error {
		sleeps++
		if sleeps >= 2 {
			return context.Canceled
		}
		return nil
	}
	stdout, stderr := executeRootWithSeparateOutputsForTest(t,
		"--config", writeTestConfigForVersion(t, server.URL, "8.9"), "get", "pd", "--watch", "--timeout", "20ms", "--backoff-max-retries", "2")
	require.EqualValues(t, 2, requests.Load())
	require.Equal(t, 2, sleeps)
	require.Equal(t, 1, strings.Count(stdout, "found: 0"))
	require.Contains(t, stderr, "WARN retrying process-definition watch after refresh 1 failed (1/2 consecutive failures): timeout")
	require.NotContains(t, stderr, "watch stopped")
	require.NotContains(t, stderr, server.URL)
}

// TestGetProcessDefinitionWatchExplicitTimeoutStops exercises a positive public
// session duration and verifies a single logger-routed stop after refresh.
func TestGetProcessDefinitionWatchExplicitTimeoutStops(t *testing.T) {
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(resetGetProcessDefinitionCommandGlobals)
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0}}`))
	}))
	previousSleep := processDefinitionWatchSleep
	t.Cleanup(func() { processDefinitionWatchSleep = previousSleep })
	processDefinitionWatchSleep = func(ctx context.Context, _ time.Duration) error { <-ctx.Done(); return ctx.Err() }
	stdout, stderr := executeRootWithSeparateOutputsForTest(t,
		"--config", writeTestConfigForVersion(t, server.URL, "8.9"), "get", "pd", "--watch", "--watch-timeout", "100ms")
	require.Equal(t, 1, strings.Count(stdout, "found: 0"))
	require.NotContains(t, stdout, "watch stopped")
	require.Equal(t, 1, strings.Count(stderr, "INFO watch stopped: timeout reached"))
}

// TestGetProcessDefinitionWatchFailureDiagnostics verifies the command's final
// exit and compact retry/error logs, with HTTP details confined to redacted DEBUG.
func TestGetProcessDefinitionWatchFailureDiagnostics(t *testing.T) {
	for _, format := range []string{"plain", "json"} {
		for _, mode := range []string{"normal", "verbose", "debug-flag", "debug-config"} {
			t.Run(format+"/"+mode, func(t *testing.T) {
				var requests atomic.Int32
				server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusServiceUnavailable)
					_, _ = w.Write([]byte(`{"message":"watch-backend-detail POST https://watch-secret.invalid/8675309","access_token":"watch-token-secret"}`))
				}))
				level := "info"
				if mode == "debug-config" {
					level = "debug"
				}
				cfg := writeRawTestConfig(t, fmt.Sprintf(`app:
  camunda_version: 8.9
auth:
  mode: none
apis:
  camunda_api:
    base_url: %s
log:
  level: %s
  format: %s
`, server.URL, level, format))
				args := []string{"get", "pd", "--watch", "--watch-interval", "1ms", "--backoff-max-retries", "1"}
				if mode == "verbose" {
					args = append(args, "--verbose")
				}
				if mode == "debug-flag" {
					args = append(args, "--debug")
				}
				stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t,
					"TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookupHelper", "", map[string]string{
						"C8VOLT_TEST_CONFIG": cfg, "C8VOLT_TEST_PD_ARGS": marshalStringSliceForEnv(t, args),
					}, "")
				var exitErr *exec.ExitError
				require.ErrorAs(t, err, &exitErr)
				require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
				require.EqualValues(t, 2, requests.Load())
				require.Empty(t, stdout)
				require.Equal(t, 1, strings.Count(stderr, "retrying process-definition watch"))
				require.Equal(t, 1, strings.Count(stderr, "watch process definitions: retry budget exhausted after 2 consecutive failures (unavailable)"))
				if format == "plain" {
					require.Contains(t, stderr, "WARN retrying")
					require.Contains(t, stderr, "ERROR watch process definitions:")
				} else {
					require.Contains(t, stderr, `"level":"WARN"`)
					require.Contains(t, stderr, `"level":"ERROR"`)
				}
				require.NotContains(t, stderr, "watch-token-secret")
				if strings.HasPrefix(mode, "debug-") {
					require.Contains(t, stderr, "api #")
					require.Contains(t, stderr, "/v2/process-definitions/search")
				} else {
					for _, detail := range []string{"api #", "POST", server.URL, "watch-secret.invalid", "8675309", "watch-backend-detail"} {
						require.NotContains(t, stderr, detail)
					}
				}
			})
		}
	}
}

// TestGetProcessDefinitionWatchRequestTimeoutExhaustion keeps request timeout
// exhaustion a failure with its original exit code, unlike a planned session stop.
func TestGetProcessDefinitionWatchRequestTimeoutExhaustion(t *testing.T) {
	var requests atomic.Int32
	releaseRequests := make(chan struct{})
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		select {
		case <-r.Context().Done():
		case <-releaseRequests:
		}
	}))
	t.Cleanup(func() { close(releaseRequests) })
	args := []string{"get", "pd", "--watch", "--watch-interval", "1ms", "--timeout", "20ms", "--backoff-max-retries", "1"}
	stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t,
		"TestGetProcessDefinitionWatchRejectsMachineModesBeforeLookupHelper", "", map[string]string{
			"C8VOLT_TEST_CONFIG":  writeTestConfigForVersion(t, server.URL, "8.9"),
			"C8VOLT_TEST_PD_ARGS": marshalStringSliceForEnv(t, args),
		}, "")
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, exitcode.Timeout, exitErr.ExitCode())
	require.EqualValues(t, 2, requests.Load())
	require.Empty(t, stdout)
	require.Equal(t, 1, strings.Count(stderr, "WARN retrying"))
	require.Equal(t, 1, strings.Count(stderr, "ERROR watch process definitions: retry budget exhausted after 2 consecutive failures (timeout)"))
	require.NotContains(t, stderr, "watch stopped: timeout reached")
	require.NotContains(t, stderr, server.URL)
}
