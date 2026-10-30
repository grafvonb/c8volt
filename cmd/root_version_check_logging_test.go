// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestVersionCheckLoggerBoundary detects direct-print bypasses, logger filtering,
// destination leaks and raw transport disclosure during actual command execution.
func TestVersionCheckLoggerBoundary(t *testing.T) {
	isolateRequestRateEnvironment(t)
	server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/v2/topology" {
			w.WriteHeader(403)
			fmt.Fprint(w, `{"detail":"POST https://private.invalid credential-secret body-secret resource-secret"}`)
			return
		}
		fmt.Fprint(w, apiDiagnosticsProcessDefinitionFixture)
	}))
	t.Cleanup(server.Close)
	cfg := writeTestConfigForVersion(t, server.URL, "8.9")
	for _, format := range []string{"plain-time", "json"} {
		for _, destination := range []string{"inherited", "configured"} {
			for _, tc := range []struct {
				name        string
				flags       []string
				warn, debug bool
			}{
				{name: "normal", warn: true},
				{name: "verbose", flags: []string{"--verbose"}, warn: true},
				{name: "WARN", flags: []string{"--log-level", "warn"}, warn: true},
				{name: "ERROR", flags: []string{"--log-level", "error"}},
				{name: "quiet", flags: []string{"--quiet"}},
				{name: "debug", flags: []string{"--debug"}, warn: true, debug: true},
				{name: "configured debug", flags: []string{"--log-level", "debug"}, warn: true, debug: true},
				{name: "debug verbose", flags: []string{"--debug", "--verbose"}, warn: true, debug: true},
				{name: "quiet debug", flags: []string{"--quiet", "--debug"}},
			} {
				t.Run(format+"/"+destination+"/"+tc.name, func(t *testing.T) {
					root := Root()
					resetCommandTreeFlags(root)
					resetRequestRateCommandContexts(root)
					resetGetProcessDefinitionCommandGlobals()
					defer func() {
						getProcessDefinitionCmd.SetErr(nil)
						resetCommandTreeFlags(root)
						resetRequestRateCommandContexts(root)
					}()
					var stdout, diagnostic, unused bytes.Buffer
					root.SetOut(&stdout)
					root.SetErr(&diagnostic)
					if destination == "configured" {
						root.SetErr(&unused)
						getProcessDefinitionCmd.SetErr(&diagnostic)
					}
					args := []string{"--config", cfg, "--log-format", format, "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey, "--json"}
					root.SetArgs(append(args, tc.flags...))
					_, err := root.ExecuteC()
					require.NoError(t, err)
					require.Empty(t, unused.String())
					requireSingleJSONValue(t, stdout.String())
					require.NotContains(t, stdout.String(), "version check:")
					var warnings, debugs []string
					for _, record := range bulkCompletionDiagnosticRecords(t, diagnostic.String(), format) {
						switch record[0] {
						case "WARN":
							warnings = append(warnings, record[1])
						case "DEBUG":
							debugs = append(debugs, record[1])
						case "INFO":
							require.Contains(t, []string{"retrieving process definition", "process definition retrieved"}, record[1])
						default:
							t.Fatalf("unexpected record: %v", record)
						}
					}
					if tc.warn {
						require.Equal(t, []string{"version check: access denied; using configured 8.9"}, warnings)
					} else {
						require.Empty(t, warnings)
					}
					require.Equal(t, tc.debug, len(debugs) > 0)
					if tc.debug {
						require.Contains(t, strings.Join(debugs, "\n"), "GET /v2/topology: status=403")
					}
					for _, bad := range []string{"private.invalid", "credential-secret", "body-secret", "resource-secret"} {
						require.NotContains(t, stdout.String()+diagnostic.String(), bad)
					}
					if !tc.debug {
						for _, bad := range []string{server.URL, "GET", "POST", "/v2/"} {
							require.NotContains(t, diagnostic.String(), bad)
						}
					}
				})
			}
		}
	}
}
