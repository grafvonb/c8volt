// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestBoundedConnectionFailureReason(t *testing.T) {
	refused := &url.Error{Op: "Post", URL: "http://operator:credential-secret@localhost:18080/private?token=query-secret#fragment-secret", Err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}}
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"authentication refused", httpc.AuthenticationFailure(refused), "authentication service connection refused; check the configured host and port and that the service is running"},
		{"gateway refused", refused, "gateway connection refused; check the configured host and port and that the service is running"},
		{"dns", &net.DNSError{Name: "private.invalid", Err: "body-secret"}, "cannot resolve gateway host; check the configured hostname and DNS"},
		{"tls", x509.UnknownAuthorityError{}, "gateway TLS certificate validation failed; check the certificate trust, hostname and expiry"},
		{"network timeout", &net.DNSError{Name: "private.invalid", IsTimeout: true}, "gateway timed out; check service availability, network access and configured timeout"},
		{"operation timeout", context.DeadlineExceeded, "timeout"},
		{"credentials", httpc.WithHTTPStatus(errors.New("body-secret"), 401), "gateway HTTP 401: authentication rejected; check the selected profile and credentials"},
		{"permissions", httpc.WithHTTPStatus(errors.New("body-secret"), 403), "gateway HTTP 403: access denied; check credentials, roles and API permissions"},
		{"oauth client", httpc.AuthenticationFailure(httpc.WithHTTPStatus(errors.New("body-secret"), 400)), "authentication service HTTP 400: authentication request rejected; check client credentials and scope configuration"},
		{"oauth endpoint", httpc.AuthenticationFailure(httpc.WithHTTPStatus(errors.New("body-secret"), 404)), "authentication service HTTP 404: authentication endpoint not found; check the authentication URL and proxy routing"},
		{"canceled authentication", httpc.AuthenticationFailure(context.Canceled), "local precondition"},
		{"unknown", errors.New("HTTP 401 Post https://private.invalid: connection refused"), "internal"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := ferrors.FromDomain(errors.Join(tc.err, errors.New("POST https://private.invalid/8675309 body-secret\nforged log")))
			require.Equal(t, tc.want, boundedCommandFailureReason(err))
		})
	}
}

// Exercise the existing command runner, service/facade conversion, logger and
// exit boundary using only test-owned endpoints and explicit fixture config.
func TestCommandConnectionFailureLogging(t *testing.T) {
	for _, cause := range []string{"auth-refused", "gateway-refused", "credentials", "permissions"} {
		for _, format := range []string{"plain-time", "json"} {
			for _, mode := range []string{"normal", "verbose", "quiet", "keys", "json", "quiet-json", "debug-flag", "debug-config", "no-error-codes"} {
				t.Run(cause+"/"+format+"/"+mode, func(t *testing.T) {
					var calls atomic.Int32
					status := http.StatusUnauthorized
					if strings.HasSuffix(cause, "permissions") {
						status = http.StatusForbidden
					}
					server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						require.Equal(t, "/oauth/token", r.URL.Path)
						w.Header().Set("Content-Type", "application/json")
						w.WriteHeader(status)
						fmt.Fprint(w, `{"detail":"POST https://private.invalid/8675309 body-secret","access_token":"token-secret"}`)
					}))
					t.Cleanup(server.Close)
					refused := strings.HasSuffix(cause, "refused")
					if refused {
						// A closed test-owned listener deterministically refuses connections.
						server.Close()
					}
					auth := "oauth2"
					if strings.HasPrefix(cause, "gateway-") {
						auth = "none"
					}
					level := "info"
					if mode == "debug-config" {
						level = "debug"
					}
					cfg := writeRawTestConfig(t, fmt.Sprintf(`app: {camunda_version: "8.10"}
auth:
  mode: %s
  oauth2: {token_url: %q, client_id: operator, client_secret: credential-secret}
apis: {camunda_api: {base_url: %q}}
log: {level: %s, format: %s}
`, auth, server.URL+"/oauth", server.URL, level, format))
					args := []string{"--no-version-check", "--no-indicator", "get", "pd"}
					operation := "search process definitions"
					route := "inherited"
					if format == "json" {
						route = "configured"
					}
					switch mode {
					case "verbose", "quiet", "no-error-codes":
						flag := mode
						if mode == "no-error-codes" {
							flag = "no-err-codes"
						}
						args = append(args, "--"+flag)
					case "keys":
						args = append(args, "--keys-only")
					case "quiet-json":
						args = append(args, "--quiet", "--json")
					case "debug-flag":
						args = append(args, "--debug")
					}
					capture := filepath.Join(t.TempDir(), "diagnostics")
					wantExit := 1
					if mode == "no-error-codes" {
						wantExit = 0
					}
					stdout, stderr := runCommandErrorEnvelopeSubprocess(t, processDefinitionErrorEnvelopeHelper, "", map[string]string{
						"C8VOLT_TEST_CONFIG": cfg, "C8VOLT_TEST_OUTPUT_MODE": mode,
						"C8VOLT_TEST_COMMAND_ARGS": marshalRootArgsForEnv(t, args),
						"C8VOLT_TEST_PD_LOG_DEST":  capture, "C8VOLT_TEST_PD_LOG_ROUTE": route,
					}, "", wantExit)
					require.Empty(t, stderr, "the configured logger must own all diagnostics")
					wantCalls := int32(1)
					if refused {
						wantCalls = 0
					}
					require.Equal(t, wantCalls, calls.Load())
					data, err := os.ReadFile(capture)
					require.NoError(t, err)
					diagnostics := string(data)
					if mode == "json" || mode == "quiet-json" {
						envelope := requireSingleJSONObjectDocument(t, stdout)
						require.Equal(t, "failed", envelope["outcome"])
						class := "local_precondition"
						if refused {
							class = "internal"
						}
						require.Equal(t, class, envelope["class"])
						detail := requireJSONObject(t, envelope["detail"])
						require.Contains(t, detail["message"], operation+": ")
						if !refused {
							require.Contains(t, detail["message"], "body-secret", "machine evidence remains unchanged")
						}
						require.Empty(t, diagnostics)
						return
					}
					require.Empty(t, stdout)
					message := "authentication service HTTP 401: authentication rejected; check the selected profile and credentials"
					if strings.HasSuffix(cause, "permissions") {
						message = "authentication service HTTP 403: access denied; check credentials, roles and API permissions"
					}
					if refused {
						target := "authentication service"
						if auth == "none" {
							target = "gateway"
						}
						message = target + " connection refused; check the configured host and port and that the service is running"
					}
					var errorMessages []string
					for _, record := range bulkCompletionDiagnosticRecords(t, diagnostics, format) {
						if record[0] == "ERROR" {
							errorMessages = append(errorMessages, record[1])
						}
					}
					require.Equal(t, []string{operation + ": " + message}, errorMessages)
					debug := strings.HasPrefix(mode, "debug-")
					require.Equal(t, debug, strings.Contains(diagnostics, "api #"))
					for _, secret := range []string{"credential-secret", "token-secret", "private.invalid", "body-secret", "8675309"} {
						require.NotContains(t, stdout+stderr+diagnostics, secret)
					}
					if !debug {
						for _, raw := range []string{"POST", server.URL, "/oauth", "/v2/"} {
							require.NotContains(t, diagnostics, raw)
						}
					}
				})
			}
		}
	}
}
