// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// Failure messages use typed evidence and never interpolate untrusted error text.
func TestConfigConnectionFailureReasons(t *testing.T) {
	for _, tc := range []struct {
		err     error
		message string
	}{
		{syscall.ECONNREFUSED, "gateway connection refused; check the configured host and port and that the service is running"},
		{&net.DNSError{Name: "private.invalid"}, "cannot resolve gateway host; check the configured hostname and DNS"},
		{x509.UnknownAuthorityError{}, "gateway TLS certificate validation failed; check the certificate trust, hostname and expiry"},
		{context.DeadlineExceeded, "gateway timed out; check service availability, network access and configured timeout"},
		{context.Canceled, "connection test canceled"},
		{syscall.ECONNRESET, "gateway connection interrupted; check the service and proxy"},
		{&json.SyntaxError{}, "invalid gateway response; check the configured URL and proxy routing"},
		{httpc.WithHTTPStatus(errors.New("raw"), 404), "gateway HTTP 404: topology endpoint not found; check the Camunda API base URL and proxy routing"},
		{httpc.WithHTTPStatus(errors.New("raw"), 429), "gateway HTTP 429: rate limit reached; retry later or reduce request rate"},
		{ferrors.Normalize(httpc.AuthenticationFailure(syscall.ECONNREFUSED)), "authentication service connection refused; check the configured host and port and that the service is running"},
		{errors.New("HTTP 401 connection refused"), "unexpected connection failure; use --debug for diagnostic details"},
	} {
		raw := errors.Join(tc.err, errors.New("POST https://private.invalid body-secret resource-key\nforged log"))
		require.Equal(t, "config test-connection: "+tc.message, configConnectionFailureMessage(raw))
	}
}

// Execute real command failures with isolated HTTP responses. This covers the
// logger boundary, auth stage, retries, JSON policy and successful single reads.
func TestConfigConnectionDiagnosticLogging(t *testing.T) {
	for _, tc := range []struct {
		name, auth, message    string
		status, requests, exit int
	}{
		{"gateway", "none", "gateway HTTP 503: service error; check service health and proxy logs", 503, 4, exitcode.Unavailable},
		{"internal", "none", "gateway HTTP 500: service error; check service health and proxy logs", 500, 4, exitcode.Error},
		{"cookie", "cookie", "authentication service HTTP 403: access denied; check credentials, roles and API permissions", 403, 1, exitcode.Error},
		{"oauth2", "oauth2", "authentication service HTTP 401: authentication rejected; check the selected profile and credentials", 401, 1, exitcode.Error},
		{"none-success", "none", "", 200, 1, 0},
		{"cookie-success", "cookie", "", 200, 2, 0},
		{"oauth2-success", "oauth2", "", 200, 2, 0},
	} {
		for _, format := range []string{"plain-time", "json"} {
			for _, mode := range []struct {
				name, level string
				flags       []string
				debug       bool
			}{
				{"normal", "info", nil, false},
				{"verbose", "info", []string{"--verbose"}, false},
				{"quiet", "info", []string{"--quiet"}, false},
				{"json", "info", []string{"--json"}, false},
				{"quiet-json", "info", []string{"--quiet", "--json"}, false},
				{"debug", "info", []string{"--debug"}, true},
				{"configured-debug", "debug", nil, true},
				{"quiet-debug", "debug", []string{"--quiet", "--debug"}, false},
				{"command-writer", "info", nil, false},
				{"no-error-codes", "info", []string{"--no-err-codes"}, false},
			} {
				if tc.status == 200 && mode.name != "normal" {
					continue
				}
				t.Run(tc.name+"/"+format+"/"+mode.name, func(t *testing.T) {
					var calls atomic.Int32
					server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if tc.status != 200 {
							path := map[string]string{"none": "/v2/topology", "cookie": "/api/login", "oauth2": "/oauth/token"}[tc.auth]
							require.Equal(t, path, r.URL.Path)
							// Both status text and body contain untrusted diagnostic markers.
							conn, rw, err := w.(http.Hijacker).Hijack()
							require.NoError(t, err)
							defer conn.Close()
							body := `{"detail":"POST https://private.invalid body-secret resource-key"}`
							fmt.Fprintf(rw, "HTTP/1.1 %d POST https://private.invalid body-secret\r\nContent-Type: application/json\r\nContent-Length: %d\r\nRetry-After: 0\r\nConnection: close\r\n\r\n%s", tc.status, len(body), body)
							require.NoError(t, rw.Flush())
							return
						}
						w.Header().Set("Content-Type", "application/json")
						http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "cookie-secret", Path: "/"})
						body := map[string]string{"/api/login": `{}`, "/oauth/token": `{"access_token":"token-secret","expires_in":120}`, "/v2/topology": `{"gatewayVersion":"8.10.6"}`}[r.URL.Path]
						require.NotEmpty(t, body, "unexpected request path")
						fmt.Fprint(w, body)
					}))
					t.Cleanup(server.Close)
					cfg := writeRawTestConfig(t, fmt.Sprintf(`app: {camunda_version: "8.10"}
auth:
  mode: %s
  cookie: {base_url: %q, username: operator, password: credential-secret}
  oauth2: {token_url: %q, client_id: operator, client_secret: credential-secret}
apis: {camunda_api: {base_url: %q}}
log: {level: %s, format: %s}
`, tc.auth, server.URL, server.URL+"/oauth", server.URL, mode.level, format))
					flags, debug := mode.flags, mode.debug
					if tc.status == 200 {
						flags, debug = []string{"--json", "--debug"}, true
					}
					args := append([]string{"--config", cfg, "--no-indicator", "config", "test-connection"}, flags...)
					logFile := filepath.Join(t.TempDir(), "diagnostics")
					wantExit := tc.exit
					if mode.name == "no-error-codes" {
						wantExit = 0
					}
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, "TestConfigConnectionDiagnosticsHelper", "", map[string]string{
						"C8VOLT_TEST_COMMAND_ARGS":           marshalRootArgsForEnv(t, args),
						"C8VOLT_TEST_CONNECTION_LOG":         logFile,
						"C8VOLT_TEST_CONNECTION_DESTINATION": mode.name,
					}, "", wantExit)
					require.Empty(t, stderr, "diagnostics must follow the configured logger destination")
					data, err := os.ReadFile(logFile)
					require.NoError(t, err)
					diagnostics := string(data)
					records := bulkCompletionDiagnosticRecords(t, diagnostics, format)
					if tc.status == 200 {
						requireSingleJSONValue(t, stdout)
						var result configTestConnectionView
						require.NoError(t, json.Unmarshal([]byte(stdout), &result))
						require.True(t, result.OK)
						require.Equal(t, "8.10.6", result.Cluster.GatewayVersion)
						require.Nil(t, result.Warnings)
					} else {
						require.Empty(t, stdout, "preserve existing failure policy even with --json")
						var errors []string
						for _, record := range records {
							if record[0] == "ERROR" {
								errors = append(errors, record[1])
							}
							require.NotEqual(t, "WARN", record[0], "no duplicate advisory")
						}
						require.Equal(t, []string{"config test-connection: " + tc.message}, errors)
						if strings.HasPrefix(mode.name, "quiet") {
							require.Len(t, records, 1)
						}
					}
					require.Equal(t, int32(tc.requests), calls.Load())
					require.Equal(t, debug, strings.Contains(diagnostics, "api #"))
					for _, secret := range []string{"private.invalid", "body-secret", "resource-key", "credential-secret", "cookie-secret", "token-secret"} {
						require.NotContains(t, stdout+diagnostics, secret)
					}
					if !debug {
						for _, raw := range []string{server.URL, "GET", "POST", "/v2/"} {
							require.NotContains(t, diagnostics, raw)
						}
					}
				})
			}
		}
	}
}

// Run the real command with configured or inherited diagnostic writers.
func TestConfigConnectionDiagnosticsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" || os.Getenv(testx.CmdSubprocessNameEnv) != t.Name() {
		return
	}
	root := Root()
	root.SetOut(os.Stdout)
	writer, err := os.Create(os.Getenv("C8VOLT_TEST_CONNECTION_LOG"))
	require.NoError(t, err)
	if os.Getenv("C8VOLT_TEST_CONNECTION_DESTINATION") == "command-writer" {
		root.SetErr(os.Stderr)
		configTestConnectionCmd.SetErr(writer)
	} else {
		root.SetErr(writer)
	}
	var args []string
	require.NoError(t, json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_COMMAND_ARGS")), &args))
	root.SetArgs(args)
	_, err = root.ExecuteC()
	require.NoError(t, err)
	os.Exit(0)
}
