// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const apiDiagnosticsProcessDefinitionKey = "2251799813685255"

// TestAPIDiagnosticsCommandReadPreservesResultsAndRequests verifies inherited verbose placement enables one diagnostic without changing command results or traffic.
func TestAPIDiagnosticsCommandReadPreservesResultsAndRequests(t *testing.T) {
	var requests atomic.Int32
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		require.Equal(t, http.MethodGet, request.Method)
		require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		time.Sleep(5 * time.Millisecond)
		_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
	}))
	t.Cleanup(server.Close)
	configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")

	baselineStdout, baselineStderr := executeAPIDiagnosticsRoot(t,
		"--config", configPath, "--log-format", "plain",
		"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey,
	)
	require.NotContains(t, baselineStderr, "api #")

	for _, args := range [][]string{
		{"--verbose", "--config", configPath, "--log-format", "plain", "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey},
		{"--config", configPath, "--log-format", "plain", "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey, "--verbose"},
	} {
		stdout, stderr := executeAPIDiagnosticsRoot(t, args...)
		require.Equal(t, baselineStdout, stdout)
		require.Equal(t, 1, strings.Count(stderr, "api #"))
		require.Contains(t, stderr, "api #1 GET /v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey+": status=200")
		require.Contains(t, stderr, " headers=")
		require.Contains(t, stderr, " body=")
	}

	require.Equal(t, int32(3), requests.Load(), "diagnostics must not add requests")
}

// TestAPIDiagnosticsCommandAuthenticationBootstrap verifies token and cookie initialization traffic shares command diagnostics while auth-none adds no bootstrap exchange.
func TestAPIDiagnosticsCommandAuthenticationBootstrap(t *testing.T) {
	for _, test := range []struct {
		name            string
		authMode        string
		wantRequests    int32
		wantFirstTarget string
	}{
		{name: "none", authMode: "none", wantRequests: 1, wantFirstTarget: "GET /v2/process-definitions/"},
		{name: "oauth2", authMode: "oauth2", wantRequests: 2, wantFirstTarget: "POST /oauth/token"},
		{name: "cookie", authMode: "cookie", wantRequests: 2, wantFirstTarget: "POST /api/login"},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				requests.Add(1)
				switch request.URL.Path {
				case "/oauth/token":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"access_token":"command-token","expires_in":120,"token_type":"Bearer"}`))
				case "/api/login":
					http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "command-cookie", Path: "/"})
					_, _ = w.Write([]byte(`{}`))
				case "/v2/process-definitions/" + apiDiagnosticsProcessDefinitionKey:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
				default:
					http.NotFound(w, request)
				}
			}))
			t.Cleanup(server.Close)
			configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, test.authMode)

			_, stderr := executeAPIDiagnosticsRoot(t,
				"--config", configPath, "--log-format", "plain", "--verbose",
				"get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey,
			)

			require.Equal(t, test.wantRequests, requests.Load())
			require.Equal(t, int(test.wantRequests), strings.Count(stderr, "api #"))
			require.Contains(t, stderr, "api #1 "+test.wantFirstTarget)
			require.NotContains(t, stderr, "command-client-secret")
			require.NotContains(t, stderr, "command-token")
			require.NotContains(t, stderr, "command-cookie")
		})
	}
}

// TestAPIDiagnosticsCommandFiltering verifies verbose-off, debug-only, quiet and restrictive INFO settings install no admitted command diagnostic output.
func TestAPIDiagnosticsCommandFiltering(t *testing.T) {
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, request.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(apiDiagnosticsProcessDefinitionFixture))
	}))
	t.Cleanup(server.Close)
	configPath := writeAPIDiagnosticsCommandConfig(t, server.URL, "none")

	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "verbose off", args: nil},
		{name: "debug only", args: []string{"--debug"}},
		{name: "quiet verbose", args: []string{"--quiet", "--verbose"}},
		{name: "info filtered", args: []string{"--verbose", "--log-level", "warn"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"--config", configPath, "--log-format", "plain"}
			args = append(args, test.args...)
			args = append(args, "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey)
			_, stderr := executeAPIDiagnosticsRoot(t, args...)
			require.NotContains(t, stderr, "api #")
		})
	}
}

// TestAPIDiagnosticsCommandHelpPerformsNoExchange verifies help accepts verbose without bootstrapping remote services.
func TestAPIDiagnosticsCommandHelpPerformsNoExchange(t *testing.T) {
	stdout, stderr := executeAPIDiagnosticsRoot(t, "get", "process-definition", "--verbose", "--help")

	require.Contains(t, stdout, "--verbose")
	require.NotContains(t, stderr, "api #")
}

// executeAPIDiagnosticsRoot resets shared Cobra state and captures result and diagnostic streams separately.
func executeAPIDiagnosticsRoot(t *testing.T, args ...string) (string, string) {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	resetGetProcessDefinitionCommandGlobals()
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		resetGetProcessDefinitionCommandGlobals()
	})
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	require.NoError(t, err)
	return stdout.String(), stderr.String()
}

// withoutAPIDiagnosticLines lets existing progress assertions distinguish the new transport records from command-owned lifecycle text.
func withoutAPIDiagnosticLines(output string) string {
	var retained strings.Builder
	for line := range strings.SplitSeq(output, "\n") {
		if strings.Contains(line, " api #") || strings.HasPrefix(line, "api #") {
			continue
		}
		if line != "" {
			retained.WriteString(line)
			retained.WriteByte('\n')
		}
	}
	return retained.String()
}

// writeAPIDiagnosticsCommandConfig creates auth variants that all use one deterministic local API fixture.
func writeAPIDiagnosticsCommandConfig(t *testing.T, baseURL, authMode string) string {
	t.Helper()

	auth := "  mode: none\n"
	switch authMode {
	case "oauth2":
		auth = fmt.Sprintf("  mode: oauth2\n  oauth2:\n    token_url: %q\n    client_id: command-client\n    client_secret: command-client-secret\n", baseURL+"/oauth")
	case "cookie":
		auth = fmt.Sprintf("  mode: cookie\n  cookie:\n    base_url: %q\n    username: command-user\n    password: command-password\n", baseURL)
	}
	content := fmt.Sprintf("app:\n  camunda_version: %q\nauth:\n%sapis:\n  camunda_api:\n    base_url: %q\n", "8.8", auth, baseURL)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
	return configPath
}

const apiDiagnosticsProcessDefinitionFixture = `{"processDefinitionKey":"2251799813685255","processDefinitionId":"invoice","name":"invoice","version":3,"tenantId":"tenant","versionTag":"stable"}`
