// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
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
				configDebug bool
				warn, debug bool
			}{
				{name: "normal", warn: true},
				{name: "verbose", flags: []string{"--verbose"}, warn: true},
				{name: "WARN", flags: []string{"--log-level", "warn"}, warn: true},
				{name: "ERROR", flags: []string{"--log-level", "error"}},
				{name: "quiet", flags: []string{"--quiet"}},
				{name: "debug", flags: []string{"--debug"}, warn: true, debug: true},
				{name: "log-level debug", flags: []string{"--log-level", "debug"}, warn: true, debug: true},
				{name: "configured debug", configDebug: true, warn: true, debug: true},
				{name: "configured debug verbose", configDebug: true, flags: []string{"--verbose"}, warn: true, debug: true},
				{name: "configured debug quiet", configDebug: true, flags: []string{"--quiet"}},
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
					selectedConfig := cfg
					if tc.configDebug {
						content, err := os.ReadFile(cfg)
						require.NoError(t, err)
						selectedConfig = writeRawTestConfig(t, string(content)+"\nlog:\n  level: debug\n")
					}
					args := []string{"--config", selectedConfig, "--log-format", format, "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey, "--json"}
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
						var summaries []string
						for _, message := range debugs {
							if strings.HasPrefix(message, "version check:") {
								summaries = append(summaries, message)
							}
						}
						require.Equal(t, []string{"version check: result=failed configured=8.9 gateway=unknown using=8.9 comparison=major.minor reason=access_denied"}, summaries)
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

// TestVersionCheckDebugSummary exercises topology facts and unchanged warnings
// through commands, including opt-out, output filtering, and incomplete metadata.
func TestVersionCheckDebugSummary(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, tc := range []struct {
		name, topology, summary, warning string
		status                           int
	}{
		{name: "matched rolling versions", topology: `{"gatewayVersion":"8.9.3","clusterSize":3,"partitionsCount":12,"replicationFactor":3,"brokers":[{"version":"8.9.3"},{"version":"8.10.2"},{"version":"8.9.3"}]}`,
			summary: "result=matched configured=8.9 gateway=8.9.3 using=8.9 comparison=major.minor broker_versions=[8.10.2,8.9.3] brokers=3 cluster_size=3 partitions=12 replication_factor=3"},
		{name: "mismatch", topology: `{"gatewayVersion":"8.10.3","clusterSize":1,"partitionsCount":1,"replicationFactor":1,"brokers":[{"version":"8.10.3"}]}`,
			summary: "result=mismatched configured=8.9 gateway=8.10.3 using=8.9 comparison=major.minor broker_versions=[8.10.3] brokers=1 cluster_size=1 partitions=1 replication_factor=1",
			warning: "version mismatch: configured 8.9, gateway 8.10.3; results may be incorrect; match config to gateway unless mismatch is intentional"},
		{name: "missing details", topology: `{"gatewayVersion":"8.9.3"}`,
			summary: "result=matched configured=8.9 gateway=8.9.3 using=8.9 comparison=major.minor"},
		{name: "invalid broker", topology: `{"gatewayVersion":"8.9.3","brokers":[{"version":"8.9\nPOST https://private.invalid body-secret"},{"version":""}],"clusterSize":-1,"partitionsCount":-1,"replicationFactor":-1}`,
			summary: "result=matched configured=8.9 gateway=8.9.3 using=8.9 comparison=major.minor broker_versions=[unknown] brokers=2"},
		{name: "invalid gateway retains details", topology: `{"gatewayVersion":"8.9\nPOST https://private.invalid body-secret","clusterSize":1,"brokers":[{"version":"8.9.3"}]}`,
			summary: "result=failed configured=8.9 gateway=unknown using=8.9 comparison=major.minor reason=invalid_version broker_versions=[8.9.3] brokers=1 cluster_size=1",
			warning: "version check: invalid gateway version; using configured 8.9"},
		{name: "timeout", status: 504, topology: `{}`,
			summary: "result=failed configured=8.9 gateway=unknown using=8.9 comparison=major.minor reason=timeout",
			warning: "version check: timeout; using configured 8.9"},
		{name: "unavailable", status: 503, topology: `{}`,
			summary: "result=failed configured=8.9 gateway=unknown using=8.9 comparison=major.minor reason=unavailable",
			warning: "version check: unavailable; using configured 8.9"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var probes, operations atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v2/topology" {
					probes.Add(1)
					if tc.status != 0 {
						w.WriteHeader(tc.status)
					}
					fmt.Fprint(w, tc.topology)
					return
				}
				operations.Add(1)
				fmt.Fprint(w, apiDiagnosticsProcessDefinitionFixture)
			}))
			t.Cleanup(server.Close)
			cfg := writeTestConfigForVersion(t, server.URL, "8.9")
			for _, format := range []string{"plain-time", "json"} {
				for _, mode := range []struct {
					name                  string
					flags                 []string
					debug, warn, disabled bool
				}{
					{name: "normal", warn: true},
					{name: "verbose", flags: []string{"--verbose"}, warn: true},
					{name: "debug", flags: []string{"--debug"}, debug: true, warn: true},
					{name: "quiet debug", flags: []string{"--quiet", "--debug"}},
					{name: "disabled", flags: []string{"--debug", "--no-version-check"}, disabled: true},
				} {
					t.Run(format+"/"+mode.name, func(t *testing.T) {
						before, operationsBefore := probes.Load(), operations.Load()
						args := []string{"--config", cfg, "--log-format", format, "get", "process-definition", "--key", apiDiagnosticsProcessDefinitionKey, "--keys-only"}
						stdout, stderr := executeAPIDiagnosticsRoot(t, append(args, mode.flags...)...)
						require.Equal(t, apiDiagnosticsProcessDefinitionKey+"\n", stdout)
						require.Equal(t, operationsBefore+1, operations.Load())
						if mode.disabled {
							require.Equal(t, before, probes.Load())
						} else {
							require.Equal(t, before+1, probes.Load())
						}
						var summaries, warnings []string
						for _, record := range bulkCompletionDiagnosticRecords(t, stderr, format) {
							if record[0] == "WARN" {
								warnings = append(warnings, record[1])
							}
							if strings.HasPrefix(record[1], "version check: result=") {
								require.Equal(t, "DEBUG", record[0])
								summaries = append(summaries, record[1])
							}
						}
						if mode.debug {
							require.Equal(t, []string{"version check: " + tc.summary}, summaries)
						} else {
							require.Empty(t, summaries)
						}
						if mode.warn && tc.warning != "" {
							require.Equal(t, []string{tc.warning}, warnings)
						} else {
							require.Empty(t, warnings)
						}
						for _, secret := range []string{"private.invalid", "body-secret"} {
							require.NotContains(t, stdout+stderr, secret)
						}
					})
				}
			}
		})
	}
}
