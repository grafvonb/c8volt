// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestVersionCheckInvocation verifies default enablement, flag placement, fresh
// invocations, safe exact warnings and unchanged results across configured adapters.
func TestVersionCheckInvocation(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, tc := range []struct {
			name, observed, warning string
			status                  int
		}{
			{name: "match", observed: version + ".3"},
			{name: "patch build", observed: version + ".0-alpha4+build.2"},
			{name: "mismatch", observed: "9.10.3", warning: fmt.Sprintf("version mismatch: configured %s, gateway 9.10.3; using %s", version, version)},
			{name: "denied", status: 403, warning: "version check: access denied; using configured " + version},
			{name: "unauthorized", status: 401, warning: "version check: access denied; using configured " + version},
			{name: "unavailable", status: 503, warning: "version check: unavailable; using configured " + version},
			{name: "empty", warning: "version check: invalid gateway version; using configured " + version},
			{name: "unsafe", observed: "8.9\nPOST https://private.invalid body-secret", warning: "version check: invalid gateway version; using configured " + version},
			{name: "overlong", observed: "8.9.0-" + strings.Repeat("a", 129), warning: "version check: invalid gateway version; using configured " + version},
		} {
			t.Run(version+"/"+tc.name, func(t *testing.T) {
				var probes, operations atomic.Int32
				server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "application/json")
					if r.URL.Path == "/v2/topology" {
						probes.Add(1)
						if tc.status != 0 {
							w.WriteHeader(tc.status)
							fmt.Fprint(w, `{"detail":"POST https://private.invalid body-secret"}`)
							return
						}
						require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"gatewayVersion": tc.observed}))
						return
					}
					operations.Add(1)
					fmt.Fprint(w, apiDiagnosticsProcessDefinitionFixture)
				}))
				t.Cleanup(server.Close)
				cfg := writeTestConfigForVersion(t, server.URL, version)
				base := []string{"--config", cfg, "--log-format", "plain-time", "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey, "--keys-only"}
				for i, flags := range [][]string{{}, {"--no-version-check"}, {"--no-version-check=true"}, {"--no-version-check=false"}} {
					before := probes.Load()
					stdout, stderr := executeAPIDiagnosticsRoot(t, append(append([]string{}, base...), flags...)...)
					require.Equal(t, apiDiagnosticsProcessDefinitionKey+"\n", stdout)
					enabled := i == 0 || i == 3
					if enabled {
						require.Equal(t, before+1, probes.Load())
					} else {
						require.Equal(t, before, probes.Load())
					}
					if enabled && tc.warning != "" {
						require.Equal(t, [][2]string{{"WARN", tc.warning}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
					} else {
						require.Empty(t, stderr)
					}
				}
				require.Equal(t, int32(4), operations.Load())
			})
		}
	}
}

// serveVersionCheckTopology is opt-in fixture support for a matching advisory;
// handlers still own and assert every operational request themselves.
func serveVersionCheckTopology(w http.ResponseWriter, r *http.Request, version string) bool {
	if r.URL.Path != "/v2/topology" {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"gatewayVersion":%q}`, version)
	return true
}

// TestVersionCheckConnectionEvidence keeps legacy actual-result warnings separate
// from safe automatic warnings, including changing observations and opt-out.
func TestVersionCheckConnectionEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, probe, actual, want string
		disabled                  bool
		machineWarning            bool
	}{
		{name: "duplicate mismatch", probe: "8.10.3", actual: "8.10.3", want: "version mismatch: configured 8.9, gateway 8.10.3; using 8.9", machineWarning: true},
		{name: "legacy permissive", probe: "8.9.garbage", actual: "8.9.garbage", want: "version check: invalid gateway version; using configured 8.9"},
		{name: "changed observation", probe: "8.9.0", actual: "8.10.3", machineWarning: true},
		{name: "opt out", actual: "8.10.3", disabled: true, machineWarning: true},
		{name: "unsafe fallback", actual: "9.10\nPOST https://private.invalid secret", disabled: true, want: "version check: invalid gateway version; using configured 8.9", machineWarning: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/topology", r.URL.Path)
				value := tc.actual
				if calls.Add(1) == 1 && !tc.disabled {
					value = tc.probe
				}
				w.Header().Set("Content-Type", "application/json")
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"gatewayVersion": value}))
			}))
			t.Cleanup(server.Close)
			cfg := writeTestConfigForVersion(t, server.URL, "8.9")
			args := []string{"--config", cfg, "--log-format", "plain-time", "--log-level", "warn", "config", "test-connection", "--json"}
			if tc.disabled {
				args = append([]string{"--no-version-check"}, args...)
			}
			stdout, stderr := executeAPIDiagnosticsRoot(t, args...)
			requireSingleJSONValue(t, stdout)
			var result configTestConnectionView
			require.NoError(t, json.Unmarshal([]byte(stdout), &result))
			require.Equal(t, tc.machineWarning, len(result.Warnings) > 0)
			require.Equal(t, camundaReleaseLineCompatibilityWarnings("8.9", tc.actual), result.Warnings)
			warning := tc.want
			if warning == "" {
				warning = result.Warnings[0]
			}
			require.Equal(t, [][2]string{{"WARN", warning}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
			if tc.disabled {
				require.Equal(t, int32(1), calls.Load())
			} else {
				require.Equal(t, int32(2), calls.Load())
			}
			require.NotContains(t, stderr, "private.invalid")
		})
	}
}

// TestVersionCheckLocalExecution leaves the lazy checker untouched when no
// operation is needed, including local dry-run and invalid inputs.
func TestVersionCheckLocalExecution(t *testing.T) {
	var calls atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	t.Cleanup(server.Close)
	cfg := writeTestConfigForVersion(t, server.URL, "8.9")
	for _, args := range [][]string{
		{"--help"}, {"capabilities", "--json"}, {"config", "show"}, {"version"},
		{"delete", "process-definition", "--key", "123", "--dry-run", "--no-state-check"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			executeAPIDiagnosticsRoot(t, append([]string{"--config", cfg}, args...)...)
			require.Zero(t, calls.Load())
		})
	}
}

// TestVersionCheckProbeFailureBudget exercises malformed bodies, redirects,
// transport disconnects and a short timeout through the real root command.
func TestVersionCheckProbeFailureBudget(t *testing.T) {
	for _, mode := range []string{"malformed", "null", "redirect", "transport", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			var probes, operations atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/v2/topology" {
					probes.Add(1)
					switch mode {
					case "timeout":
						<-r.Context().Done()
						return
					case "transport":
						conn, _, err := w.(http.Hijacker).Hijack()
						require.NoError(t, err)
						require.NoError(t, conn.Close())
						return
					case "redirect":
						w.Header().Set("Location", "/forbidden-follow")
						w.WriteHeader(302)
						return
					case "null":
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, "null")
						return
					default:
						w.Header().Set("Content-Type", "application/json")
						fmt.Fprint(w, "{broken")
						return
					}
				}
				require.NotEqual(t, "/forbidden-follow", r.URL.Path)
				operations.Add(1)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, apiDiagnosticsProcessDefinitionFixture)
			}))
			t.Cleanup(server.Close)
			cfg := writeTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr := executeAPIDiagnosticsRoot(t, "--config", cfg, "--timeout", "100ms", "--log-format", "plain-time", "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey, "--json")
			requireSingleJSONValue(t, stdout)
			require.Equal(t, int32(1), probes.Load())
			require.Equal(t, int32(1), operations.Load())
			reason := "invalid gateway version"
			switch mode {
			case "timeout":
				reason = "timeout"
			case "redirect", "transport":
				reason = "unavailable"
			}
			require.Equal(t, [][2]string{{"WARN", "version check: " + reason + "; using configured 8.9"}}, bulkCompletionDiagnosticRecords(t, stderr, "plain-time"))
		})
	}
}

// TestVersionCheckSharedCommandContext checks multiple facades and concurrent
// resources share one failed attempt and cannot dispatch before its WARN callback.
func TestVersionCheckSharedCommandContext(t *testing.T) {
	capture := &requestRateLogCapture{}
	var probes, operations atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v2/topology" {
			probes.Add(1)
			w.WriteHeader(403)
			return
		}
		require.Contains(t, capture.String(), "WARN version check: access denied; using configured 8.9")
		operations.Add(1)
		fmt.Fprint(w, apiDiagnosticsProcessDefinitionFixture)
	}))
	t.Cleanup(server.Close)
	cfg := writeTestConfigForVersion(t, server.URL, "8.9")
	root := Root()
	resetCommandTreeFlags(root)
	resetRequestRateCommandContexts(root)
	command := &cobra.Command{Use: "version-check-concurrent-fixture", RunE: func(cmd *cobra.Command, _ []string) error {
		var wg sync.WaitGroup
		failures := make(chan error, 6)
		for range 6 {
			cli, _, _, err := NewCli(cmd)
			if err != nil {
				return err
			}
			wg.Go(func() {
				_, err := cli.GetProcessDefinition(cmd.Context(), apiDiagnosticsProcessDefinitionKey)
				failures <- err
			})
		}
		wg.Wait()
		close(failures)
		for err := range failures {
			if err != nil {
				return err
			}
		}
		return nil
	}}
	root.AddCommand(command)
	defer func() {
		root.RemoveCommand(command)
		resetCommandTreeFlags(root)
		resetRequestRateCommandContexts(root)
	}()
	root.SetOut(io.Discard)
	root.SetErr(capture)
	root.SetArgs([]string{"--config", cfg, "--log-format", "plain-time", command.Name()})
	_, err := root.ExecuteC()
	require.NoError(t, err)
	require.Equal(t, int32(1), probes.Load())
	require.Equal(t, int32(6), operations.Load())
	require.Equal(t, [][2]string{{"WARN", "version check: access denied; using configured 8.9"}}, bulkCompletionDiagnosticRecords(t, capture.String(), "plain-time"))
}

// TestVersionCheckParentCancellation preserves the original caller deadline and
// prevents operational dispatch after a probe consumes that parent budget.
func TestVersionCheckParentCancellation(t *testing.T) {
	var probes, operations atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/topology" {
			probes.Add(1)
			<-r.Context().Done()
			return
		}
		operations.Add(1)
		w.WriteHeader(500)
	}))
	t.Cleanup(server.Close)
	cfg := writeTestConfigForVersion(t, server.URL, "8.9")
	root := Root()
	resetCommandTreeFlags(root)
	resetRequestRateCommandContexts(root)
	command := &cobra.Command{Use: "version-check-cancel-fixture", RunE: func(cmd *cobra.Command, _ []string) error {
		cli, _, _, err := NewCli(cmd)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), 100*time.Millisecond)
		defer cancel()
		_, err = cli.GetProcessDefinition(ctx, apiDiagnosticsProcessDefinitionKey)
		return err
	}}
	root.AddCommand(command)
	defer func() {
		root.RemoveCommand(command)
		resetCommandTreeFlags(root)
		resetRequestRateCommandContexts(root)
	}()
	var output bytes.Buffer
	root.SetOut(io.Discard)
	root.SetErr(&output)
	root.SetArgs([]string{"--config", cfg, "--log-format", "plain-time", command.Name()})
	_, err := root.ExecuteC()
	require.ErrorIs(t, err, ferrors.ErrTimeout)
	require.Equal(t, int32(1), probes.Load())
	require.Zero(t, operations.Load())
	require.Contains(t, output.String(), "WARN version check: timeout; using configured 8.9")
}

// TestVersionCheckLocalTerminalAbort proves eligible confirmation with terminal
// stdin and redirected results can decline before any probe or mutation starts.
func TestVersionCheckLocalTerminalAbort(t *testing.T) {
	var requests atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) }))
	t.Cleanup(server.Close)
	cfg := writeTestConfigForVersion(t, server.URL, "8.10")
	for _, destination := range []string{"configured", "inherited"} {
		t.Run(destination, func(t *testing.T) {
			stdout, stderr, err := runDefinitionDeleteTerminal(t, []string{"--config", cfg, "delete", "process-definition", "--key", "123", "--no-state-check", "--no-wait"}, destination, []testx.CmdTerminalExchange{{Prompt: confirmTerminalPrompt, Response: "n\n"}})
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, confirmTerminalPrompt)
			require.NotContains(t, stderr, "version check")
			require.Zero(t, requests.Load())
		})
	}
}

// TestVersionCheckConnectionFailure preserves the operational exit while both
// advisory and explicit failures remain bounded configured logger records.
func TestVersionCheckConnectionFailure(t *testing.T) {
	var calls atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(403)
		fmt.Fprint(w, "POST https://private.invalid body-secret key-secret")
	}))
	t.Cleanup(server.Close)
	cfg := writeTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := testx.RunCmdSubprocessSeparate(t, "TestConfigTestConnectionCommand_FailureHelper", map[string]string{"C8VOLT_TEST_CONFIG": cfg, "C8VOLT_TEST_CONFIG_CONNECTION_DIAGNOSTIC": "test-connection"})
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 1, exitErr.ExitCode())
	require.Empty(t, stdout)
	require.Equal(t, int32(2), calls.Load())
	var warnings, failures []string
	for _, record := range bulkCompletionDiagnosticRecords(t, stderr, "plain-time") {
		switch record[0] {
		case "WARN":
			warnings = append(warnings, record[1])
		case "ERROR":
			failures = append(failures, record[1])
		case "INFO":
		default:
			t.Fatalf("unexpected record: %v", record)
		}
	}
	require.Equal(t, []string{"version check: access denied; using configured 8.9"}, warnings)
	require.Len(t, failures, 1)
	require.Contains(t, failures[0], "config test-connection:")
	for _, bad := range []string{server.URL, "POST", "GET", "private.invalid", "body-secret", "key-secret"} {
		require.NotContains(t, stdout+stderr, bad)
	}
}

// TestVersionCheckEmptyDiscovery keeps successful empty results identical with
// advisory failure enabled/disabled across result, preview and automation modes.
func TestVersionCheckEmptyDiscovery(t *testing.T) {
	for _, flags := range [][]string{nil, {"--json"}, {"--keys-only"}, {"--quiet"}, {"--quiet", "--json"}, {"--quiet", "--keys-only"}, {"--dry-run", "--json"}, {"--dry-run", "--keys-only"}, {"--auto-confirm"}, {"--automation", "--json"}, {"--no-wait", "--json"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			var probes, discovery atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v2/topology" {
					probes.Add(1)
					w.WriteHeader(403)
					return
				}
				require.Equal(t, "/v2/process-instances/search", r.URL.Path)
				discovery.Add(1)
				fmt.Fprint(w, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
			}))
			t.Cleanup(server.Close)
			cfg := writeTestConfigForVersion(t, server.URL, "8.8")
			args := append([]string{"--config", cfg, "cancel", "process-instance", "--state", "active"}, flags...)
			baseline, baselineLog := executeRootForProcessInstanceWithSeparateOutputs(t, append(append([]string{}, args...), "--no-version-check")...)
			require.Zero(t, probes.Load())
			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
			require.Equal(t, int32(1), probes.Load())
			require.Equal(t, int32(2), discovery.Load())
			require.Equal(t, baseline, stdout)
			records := bulkCompletionDiagnosticRecords(t, stderr, "plain-time")
			var operational [][2]string
			warnings := 0
			for _, record := range records {
				if record[1] == "version check: access denied; using configured 8.8" {
					require.Equal(t, "WARN", record[0])
					warnings++
				} else {
					operational = append(operational, record)
				}
			}
			require.Equal(t, bulkCompletionDiagnosticRecords(t, baselineLog, "plain-time"), append([][2]string{}, operational...))
			if slices.Contains(flags, "--quiet") {
				require.Zero(t, warnings)
			} else {
				require.Equal(t, 1, warnings)
			}
			if slices.Contains(flags, "--json") {
				requireSingleJSONValue(t, stdout)
			}
			if slices.Contains(flags, "--keys-only") {
				require.Empty(t, stdout)
			}
			require.NotContains(t, stderr, "Do you want to proceed?")
		})
	}
}

// TestVersionCheckInvalidInput performs no advisory or operation when local
// argument validation fails; the original invalid-input exit remains authoritative.
func TestVersionCheckInvalidInput(t *testing.T) {
	var requests atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests.Add(1); w.WriteHeader(500) }))
	t.Cleanup(server.Close)
	cfg := writeTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, cfg, "", "get", "ut", "--state", "not-a-state", "--json")
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, 2, exitErr.ExitCode())
	require.Empty(t, stdout)
	require.Contains(t, stderr, "invalid")
	require.NotContains(t, stderr, "version check")
	require.Zero(t, requests.Load())
}
