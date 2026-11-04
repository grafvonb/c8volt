// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestVersionCheckConfigPrecedence verifies effective config and actual requests,
// including explicit false values at each precedence boundary.
func TestVersionCheckConfigPrecedence(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, tc := range []struct {
		name, base, profile, env string
		flags                    []string
		selectProfile, disabled  bool
	}{
		{name: "default"},
		{name: "base true", base: "true", disabled: true},
		{name: "base false", base: "false"},
		{name: "profile true", base: "false", profile: "true", selectProfile: true, disabled: true},
		{name: "profile false", base: "true", profile: "false", selectProfile: true},
		{name: "profile inherits", base: "true", selectProfile: true, disabled: true},
		{name: "inactive profile", base: "false", profile: "true"},
		{name: "env only", env: "true", disabled: true},
		{name: "env true overrides base", base: "false", env: "true", disabled: true},
		{name: "env false overrides base", base: "true", env: "false"},
		{name: "env true overrides profile", profile: "false", selectProfile: true, env: "true", disabled: true},
		{name: "env false overrides profile", profile: "true", selectProfile: true, env: "false"},
		{name: "flag shorthand overrides env", profile: "false", selectProfile: true, env: "false", flags: []string{"--no-version-check"}, disabled: true},
		{name: "flag true overrides env", env: "false", flags: []string{"--no-version-check=true"}, disabled: true},
		{name: "flag false overrides env", base: "true", profile: "true", selectProfile: true, env: "true", flags: []string{"--no-version-check=false"}},
		{name: "flag false overrides base", base: "true", flags: []string{"--no-version-check=false"}},
		{name: "flag false overrides profile", profile: "true", selectProfile: true, flags: []string{"--no-version-check=false"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.env != "" {
				t.Setenv("C8VOLT_APP_NO_VERSION_CHECK", tc.env)
			}
			var probes, operations atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/v2/topology" {
					probes.Add(1)
					fmt.Fprint(w, `{"gatewayVersion":"8.10.6"}`)
					return
				}
				require.Equal(t, "/v2/process-definitions/"+apiDiagnosticsProcessDefinitionKey, r.URL.Path)
				operations.Add(1)
				fmt.Fprint(w, apiDiagnosticsProcessDefinitionFixture)
			}))
			t.Cleanup(server.Close)
			baseApp, profileApp := "", ""
			if tc.base != "" {
				baseApp = "  no_version_check: " + tc.base + "\n"
			}
			if tc.profile != "" {
				profileApp = "      no_version_check: " + tc.profile + "\n"
			}
			path := writeRawTestConfig(t, fmt.Sprintf("app:\n  camunda_version: '8.9'\n%sauth:\n  mode: none\napis:\n  camunda_api:\n    base_url: %q\nprofiles:\n  selected:\n    app:\n      tenant: ''\n%s", baseApp, server.URL, profileApp))
			args := []string{"--config", path, "--debug", "--log-format", "plain-time"}
			if tc.selectProfile {
				args = append(args, "--profile", "selected")
			}
			args = append(args, tc.flags...)
			showArgs := append(append([]string{}, args...), "config", "show")
			shown, _ := executeAPIDiagnosticsRoot(t, showArgs...)
			var effective config.Config
			require.NoError(t, yaml.Unmarshal([]byte(shown), &effective))
			require.Contains(t, shown, fmt.Sprintf("no_version_check: %t", tc.disabled))
			require.Equal(t, tc.disabled, effective.App.NoVersionCheck)
			require.Zero(t, probes.Load(), "config inspection must remain local")
			require.Zero(t, operations.Load())

			stdout, stderr := executeAPIDiagnosticsRoot(t, append(args, "get", "pd", "--key", apiDiagnosticsProcessDefinitionKey, "--json")...)
			requireSingleJSONValue(t, stdout)
			require.NotContains(t, stdout, "version check:")
			require.NotContains(t, stdout, "version mismatch:")
			require.EqualValues(t, 1, operations.Load())
			var warnings, summaries []string
			for _, record := range bulkCompletionDiagnosticRecords(t, stderr, "plain-time") {
				if record[0] == "WARN" {
					warnings = append(warnings, record[1])
				}
				if strings.HasPrefix(record[1], "version check: result=") {
					require.Equal(t, "DEBUG", record[0])
					summaries = append(summaries, record[1])
				}
			}
			if tc.disabled {
				require.Zero(t, probes.Load())
				require.Empty(t, warnings)
				require.Empty(t, summaries)
			} else {
				require.EqualValues(t, 1, probes.Load())
				require.Equal(t, []string{"version mismatch: configured 8.9, gateway 8.10.6; results may be incorrect; match config to gateway unless mismatch is intentional"}, warnings)
				require.Equal(t, []string{"version check: result=mismatched configured=8.9 gateway=8.10.6 using=8.9 comparison=major.minor"}, summaries)
			}
		})
	}
}

// TestVersionCheckConfigInvalidBoolean rejects invalid YAML/environment values
// through configuration resolution without contacting any service.
func TestVersionCheckConfigInvalidBoolean(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, source := range []string{"base", "profile", "environment"} {
		t.Run(source, func(t *testing.T) {
			content := "auth:\n  mode: none\n"
			switch source {
			case "base":
				content += "app:\n  no_version_check: not-a-bool\n"
			case "profile":
				content += "active_profile: selected\nprofiles:\n  selected:\n    app:\n      no_version_check: not-a-bool\n"
			case "environment":
				t.Setenv("C8VOLT_APP_NO_VERSION_CHECK", "not-a-bool")
			}
			root := Root()
			resetCommandTreeFlags(root)
			t.Cleanup(func() { resetCommandTreeFlags(root) })
			require.NoError(t, root.PersistentFlags().Set("config", writeRawTestConfig(t, content)))
			v := viper.New()
			bindings, err := initViper(v, root)
			require.NoError(t, err)
			_, err = retrieveAndNormalizeConfig(v, bindings)
			require.ErrorContains(t, err, "no_version_check")
		})
	}
}

// TestVersionCheckConfigExplicitReads preserves requested topology/version reads
// when configuration disables the automatic advisory.
func TestVersionCheckConfigExplicitReads(t *testing.T) {
	isolateRequestRateEnvironment(t)
	for _, command := range [][]string{{"get", "cluster", "version", "--json"}, {"config", "test-connection", "--json"}} {
		t.Run(strings.Join(command, " "), func(t *testing.T) {
			var requests atomic.Int32
			server := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/topology", r.URL.Path)
				requests.Add(1)
				w.Header().Set("Content-Type", "application/json")
				fmt.Fprint(w, `{"gatewayVersion":"8.9.3"}`)
			}))
			t.Cleanup(server.Close)
			path := writeRawTestConfig(t, fmt.Sprintf("app:\n  camunda_version: '8.9'\n  no_version_check: true\nauth:\n  mode: none\napis:\n  camunda_api:\n    base_url: %q\n", server.URL))
			args := append([]string{"--config", path, "--debug", "--log-format", "plain-time"}, command...)
			stdout, stderr := executeAPIDiagnosticsRoot(t, args...)
			requireSingleJSONValue(t, stdout)
			require.Contains(t, stdout, "8.9.3")
			require.EqualValues(t, 1, requests.Load())
			require.NotContains(t, stderr, "version check: result=")
		})
	}
}
