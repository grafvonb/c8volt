// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestConfigHelp_ExplainsEffectiveConfigurationWorkflow(t *testing.T) {
	output := executeRootForTest(t, "config", "--help")

	require.Contains(t, output, "Inspect and validate c8volt configuration")
	require.Contains(t, output, "view effective settings")
	require.Contains(t, output, "`config show`")
	require.Contains(t, output, "`config validate`")
	require.Contains(t, output, "`config template`")
	require.Contains(t, output, "`config test-connection`")
	require.Contains(t, output, "show")
	require.Contains(t, output, "validate")
	require.Contains(t, output, "template")
	require.Contains(t, output, "test-connection")
	require.Contains(t, output, "./c8volt config show")
	require.Contains(t, output, "./c8volt --config ./config.yaml config validate")
	require.Contains(t, output, "./c8volt config template")
	require.Contains(t, output, "./c8volt --config ./config.yaml config test-connection")
	require.Contains(t, output, "./c8volt config show --template")
}

func TestConfigShowHelp_ExplainsEffectiveConfigExamples(t *testing.T) {
	output := executeRootForTest(t, "config", "show", "--help")

	require.Contains(t, output, "Show effective configuration with sensitive values sanitized")
	require.Contains(t, output, "flag > env > profile > base config > default")
	require.Contains(t, output, "compatibility shortcuts")
	require.Contains(t, output, "./c8volt --config ./config.yaml config show --validate")
	require.Contains(t, output, "--validate")
	require.Contains(t, output, "compatibility shortcut: validate the effective configuration")
	require.Contains(t, output, "--template")
	require.Contains(t, output, "compatibility shortcut: print a blank configuration template")
}

func TestConfigValidateHelp_ExplainsDirectValidation(t *testing.T) {
	output := executeRootForTest(t, "config", "validate", "--help")

	require.Contains(t, output, "Validate effective configuration")
	require.Contains(t, output, "same validation behavior as `config show --validate`")
	require.Contains(t, output, "./c8volt --config ./config.yaml config validate")
	require.NotContains(t, output, "--template")
}

func TestConfigTemplateHelp_ExplainsDirectTemplateRendering(t *testing.T) {
	output := executeRootForTest(t, "config", "template", "--help")

	require.Contains(t, output, "Print a blank configuration template")
	require.Contains(t, output, "same blank configuration template as `config show --template`")
	require.Contains(t, output, "./c8volt config template")
	require.NotContains(t, output, "--validate")
}

func TestConfigTestConnectionHelp_ExplainsConnectionDiagnostic(t *testing.T) {
	output := executeRootForTest(t, "config", "test-connection", "--help")

	require.Contains(t, output, "Test configured Camunda connection")
	require.Contains(t, output, "validates local configuration before retrieving cluster topology")
	require.Contains(t, output, "./c8volt --config ./config.yaml config test-connection")
	require.NotContains(t, output, "--template")
}

func TestConfigShowCommand_PrintsSanitizedEffectiveConfigAndWarnings(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  tenant: tenant-a
auth:
  mode: oauth2
  oauth2:
    token_url: https://auth.example.test/oauth/token
    client_id: client-id
    client_secret: super-secret
    scopes:
      camunda_api: camunda.scope
apis:
  camunda_api:
    base_url: https://camunda.example.test/v1
    require_scope: true
`)

	output := executeRootForTest(t, "--config", cfgPath, "config", "show")

	require.Contains(t, output, "tenant: tenant-a")
	require.Contains(t, output, "client_id: client-id")
	require.Contains(t, output, "client_secret: '*****'")
	require.NotContains(t, output, "super-secret")
	require.Contains(t, output, "warning: apis.camunda_api.base_url")
	require.Contains(t, output, `corrected "https://camunda.example.test/v1" to "https://camunda.example.test/v2"`)
}

func TestConfigShowCommand_ValidatePreservesValidOutcome(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: https://camunda.example.test
`)

	output, err := testx.RunCmdSubprocess(t, "TestConfigShowCommand_ValidatePreservesValidOutcomeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err)
	require.Contains(t, string(output), "tenant: tenant-a")
	require.Contains(t, string(output), "INFO configuration is valid")
}

func TestConfigValidateCommand_PreservesValidOutcome(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: https://camunda.example.test
`)

	output, err := testx.RunCmdSubprocess(t, "TestConfigDiagnosticValidateHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":            cfgPath,
		"C8VOLT_TEST_CONFIG_DIAGNOSTIC": "validate",
	})
	require.NoError(t, err)
	require.Contains(t, string(output), "INFO configuration is valid")
	require.NotContains(t, string(output), "tenant: tenant-a")
}

func TestConfigShowCommand_ValidatePreservesInvalidOutcome(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `auth:
  mode: none
`)

	output, err := testx.RunCmdSubprocess(t, "TestConfigShowCommand_ValidatePreservesInvalidOutcomeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "configuration is invalid")
	require.Contains(t, string(output), "apis.camunda_api.base_url: base_url is required")
}

func TestConfigValidateCommand_PreservesInvalidOutcome(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `auth:
  mode: none
`)

	output, err := testx.RunCmdSubprocess(t, "TestConfigDiagnosticValidateHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":            cfgPath,
		"C8VOLT_TEST_CONFIG_DIAGNOSTIC": "validate",
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "configuration is invalid")
	require.Contains(t, string(output), "apis.camunda_api.base_url: base_url is required")
}

func TestConfigValidateCommand_MatchesShowValidateOutcomes(t *testing.T) {
	validCfgPath := writeRawTestConfig(t, `app:
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: https://camunda.example.test
`)

	showValidOutput, showValidErr := testx.RunCmdSubprocess(t, "TestConfigDiagnosticValidateHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":            validCfgPath,
		"C8VOLT_TEST_CONFIG_DIAGNOSTIC": "show-validate",
	})
	validateValidOutput, validateValidErr := testx.RunCmdSubprocess(t, "TestConfigDiagnosticValidateHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":            validCfgPath,
		"C8VOLT_TEST_CONFIG_DIAGNOSTIC": "validate",
	})
	require.NoError(t, showValidErr)
	require.NoError(t, validateValidErr)
	require.Contains(t, string(showValidOutput), "INFO configuration is valid")
	require.Contains(t, string(validateValidOutput), "INFO configuration is valid")

	invalidCfgPath := writeRawTestConfig(t, `auth:
  mode: none
`)

	showInvalidOutput, showInvalidErr := testx.RunCmdSubprocess(t, "TestConfigDiagnosticValidateHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":            invalidCfgPath,
		"C8VOLT_TEST_CONFIG_DIAGNOSTIC": "show-validate",
	})
	validateInvalidOutput, validateInvalidErr := testx.RunCmdSubprocess(t, "TestConfigDiagnosticValidateHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":            invalidCfgPath,
		"C8VOLT_TEST_CONFIG_DIAGNOSTIC": "validate",
	})
	require.Error(t, showInvalidErr)
	require.Error(t, validateInvalidErr)

	showExitErr, ok := showInvalidErr.(*exec.ExitError)
	require.True(t, ok)
	validateExitErr, ok := validateInvalidErr.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, showExitErr.ExitCode(), validateExitErr.ExitCode())
	for _, output := range [][]byte{showInvalidOutput, validateInvalidOutput} {
		require.Contains(t, string(output), "local precondition failed")
		require.Contains(t, string(output), "configuration is invalid")
		require.Contains(t, string(output), "apis.camunda_api.base_url: base_url is required")
	}
}

func TestConfigShowCommand_TemplatePreservesBlankTemplateOutput(t *testing.T) {
	_, expected, err := renderBlankConfigTemplateYAML()
	require.NoError(t, err)

	output := executeRootForTest(t, "config", "show", "--template")

	require.Equal(t, expected+"\n", output)
	require.Contains(t, output, "mode: oauth2|cookie|none")
	require.Contains(t, output, "format: plain-time|plain|text|json")
	require.NotContains(t, output, "'*****'")
}

func TestConfigTemplateCommand_MatchesShowTemplateOutput(t *testing.T) {
	showOutput := executeRootForTest(t, "config", "show", "--template")
	templateOutput := executeRootForTest(t, "config", "template")

	require.Equal(t, showOutput, templateOutput)
	require.Contains(t, templateOutput, "mode: oauth2|cookie|none")
	require.Contains(t, templateOutput, "format: plain-time|plain|text|json")
	require.NotContains(t, templateOutput, "'*****'")
}

func TestConfigTestConnectionCommand_InvalidConfigStopsBeforeRemoteTopology(t *testing.T) {
	var topologyRequests atomic.Int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		topologyRequests.Add(1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `auth:
  mode: oauth2
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output, err := testx.RunCmdSubprocess(t, "TestConfigTestConnectionCommand_FailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":                       cfgPath,
		"C8VOLT_TEST_CONFIG_CONNECTION_DIAGNOSTIC": "test-connection",
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, int32(0), topologyRequests.Load())
	require.Contains(t, string(output), "configuration is invalid")
	require.Contains(t, string(output), "auth.oauth2.token_url")
}

func TestConfigTestConnectionCommand_SuccessLogsAndPrintsTopology(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(unsortedClusterTopologyFixtureJSON()))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "config", "test-connection")

	require.Equal(t, "Cluster: GatewayVersion=8.8.2 Brokers=3 Partitions=3 ReplicationFactor=3 LastCompletedChangeId=change-42\n"+
		"├─ Broker 1: broker-a.internal:26501 version=8.8.1\n"+
		"│  ├─ Partition 1: role=leader health=healthy\n"+
		"│  └─ Partition 3: role=follower health=unhealthy\n"+
		"├─ Broker 2: broker-b.internal:26502 version=8.8.2\n"+
		"│  └─ Partition 2: role=leader health=healthy\n"+
		"└─ Broker 3: broker-c.internal:26503 version=8.8.0\n", stdout)
	require.Contains(t, stderr, "INFO config loaded: "+cfgPath)
	require.Contains(t, stderr, "INFO no active profile provided in configuration, using default settings")
	require.Contains(t, stderr, "INFO connection to configured Camunda cluster succeeded base_url="+srv.URL+"/v2")
	require.NotContains(t, stderr, "WARN")
}

func TestConfigTestConnectionCommand_JSONOutput(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(unsortedClusterTopologyFixtureJSON()))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `active_profile: dev
app:
  camunda_version: "8.8"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    apis:
      camunda_api:
        base_url: `+srv.URL+`
`)

	stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "config", "test-connection", "--json")

	var result configTestConnectionView
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.True(t, result.OK)
	require.Equal(t, cfgPath, result.ConfigFile)
	require.Equal(t, "dev", result.Profile)
	require.Equal(t, srv.URL+"/v2", result.BaseURL)
	require.Equal(t, "8.8.2", result.Cluster.GatewayVersion)
	require.Equal(t, int32(3), result.Cluster.Brokers)
	require.Equal(t, int32(3), result.Cluster.Partitions)
	require.Equal(t, int32(3), result.Cluster.ReplicationFactor)
	require.Equal(t, "change-42", result.Cluster.LastCompletedChangeID)
	require.Empty(t, result.Warnings)
	require.Len(t, result.Cluster.BrokerDetails, 3)
	require.Equal(t, int32(1), result.Cluster.BrokerDetails[0].ID)
	require.Equal(t, "broker-a.internal:26501", result.Cluster.BrokerDetails[0].Address)
	require.Equal(t, int32(1), result.Cluster.BrokerDetails[0].Partitions[0].ID)
	require.NotContains(t, stdout, "Cluster:")
	require.NotContains(t, stdout, "├─")
	require.NotContains(t, stdout, "INFO")
	require.Contains(t, stdout, `"ok": true`)
	require.Contains(t, stdout, `"base_url": "`+srv.URL+`/v2"`)
	require.Contains(t, stderr, "INFO connection to configured Camunda cluster succeeded base_url="+srv.URL+"/v2")
}

func TestConfigTestConnectionCommand_JSONIncludesVersionMismatchWarning(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.9.0", 1, 1)))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "config", "test-connection", "--json")

	var result configTestConnectionView
	require.NoError(t, json.Unmarshal([]byte(stdout), &result))
	require.Len(t, result.Warnings, 1)
	require.Contains(t, result.Warnings[0], "configured Camunda version 8.8 differs from gateway version 8.9.0")
	require.Contains(t, result.Warnings[0], "this can cause unexpected errors")
	require.Contains(t, stderr, "WARN "+result.Warnings[0])
	require.NotContains(t, stdout, "WARN")
	require.NotContains(t, stdout, "Cluster:")
}

func TestConfigTestConnectionCommand_RemoteFailureUsesStandardErrorPath(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/topology", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfig(t, srv.URL)

	output, err := testx.RunCmdSubprocess(t, "TestConfigTestConnectionCommand_FailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":                       cfgPath,
		"C8VOLT_TEST_CONFIG_CONNECTION_DIAGNOSTIC": "test-connection",
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "config test-connection")
	require.NotContains(t, string(output), "configuration is invalid")
}

func TestConfigTestConnectionCommand_LogsConfigSource(t *testing.T) {
	t.Run("loaded config file", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/v2/topology", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfig(t, srv.URL)

		_, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "config", "test-connection")

		require.Contains(t, stderr, "INFO config loaded: "+cfgPath)
		require.Contains(t, stderr, "INFO no active profile provided in configuration, using default settings")
	})

	t.Run("active profile", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/v2/topology", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `active_profile: dev
app:
  camunda_version: "8.8"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  prod:
    apis:
      camunda_api:
        base_url: `+srv.URL+`
`)

		_, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "--profile", "prod", "config", "test-connection")

		require.Contains(t, stderr, "INFO config loaded: "+cfgPath)
		require.Contains(t, stderr, "INFO using configuration profile: prod")
		require.Contains(t, stderr, "INFO connection to configured Camunda cluster succeeded base_url="+srv.URL+"/v2")
	})

	t.Run("no config file loaded", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, "/v2/topology", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
		}))
		t.Cleanup(srv.Close)
		t.Setenv("C8VOLT_AUTH_MODE", "none")
		t.Setenv("C8VOLT_APIS_CAMUNDA_API_BASE_URL", srv.URL)
		t.Setenv("C8VOLT_APP_CAMUNDA_VERSION", "8.8")
		t.Setenv("HOME", t.TempDir())
		t.Setenv("XDG_CONFIG_HOME", t.TempDir())

		_, stderr := executeRootWithSeparateOutputsForTest(t, "config", "test-connection")

		require.Contains(t, stderr, "INFO no config file loaded, using defaults and environment variables")
		require.Contains(t, stderr, "INFO no active profile provided in configuration, using default settings")
	})
}

func TestConfigTestConnectionCommand_VersionComparisonWarnings(t *testing.T) {
	testCases := []struct {
		name              string
		configuredVersion string
		gatewayVersion    string
		wantWarning       bool
	}{
		{name: "exact match", configuredVersion: "8.8", gatewayVersion: "8.8", wantWarning: false},
		{name: "patch only difference", configuredVersion: "8.8", gatewayVersion: "8.8.2", wantWarning: false},
		{name: "major minor mismatch", configuredVersion: "8.8", gatewayVersion: "8.9.0", wantWarning: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "/v2/topology", r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, tc.gatewayVersion, 1, 1)))
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, tc.configuredVersion)

			_, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "config", "test-connection")

			if tc.wantWarning {
				require.Contains(t, stderr, "WARN configured Camunda version "+tc.configuredVersion+" differs from gateway version "+tc.gatewayVersion+" by major/minor version; this can cause unexpected errors because Camunda APIs can differ between versions; correct the configured version unless there is a very good reason to keep this mismatch")
				return
			}
			require.NotContains(t, stderr, "differs from gateway version")
			require.NotContains(t, stderr, "WARN")
		})
	}
}

func TestConfigCommand_BootstrapConfigErrorsDoNotPrintUsage(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: "86"
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
`)

	output, err := executeRootExpectErrorForTest(t, "--config", cfgPath, "config", "test-connection")

	require.Error(t, err)
	require.Contains(t, err.Error(), "unknown Camunda version: 86")
	require.Contains(t, err.Error(), "local precondition failed")
	require.NotContains(t, output, "Usage:")
	require.NotContains(t, output, "Examples:")
	require.NotContains(t, output, "Global Flags:")
}

func TestConfigShowCommand_RejectsValidateAndTemplateTogether(t *testing.T) {
	output, err := executeRootExpectErrorForTest(t, "config", "show", "--validate", "--template")

	require.Error(t, err)
	require.Contains(t, err.Error(), `if any flags in the group [validate template] are set none of the others can be; [template validate] were all set`)
	require.Contains(t, output, "Usage:")
}

// Verifies config show surfaces invalid effective configuration through the shared failure model.
func TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfig(t *testing.T) {
	output, err := testx.RunCmdSubprocess(t, "TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfigHelper", nil)
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "local precondition failed")
	require.Contains(t, string(output), "configuration is invalid")
}

// Verifies root persistent flags override env, profile, and base config for baseline settings.
func TestRetrieveAndNormalizeConfig_RootPersistentFlagsOverrideEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_ACTIVE_PROFILE", "dev")
	t.Setenv("C8VOLT_APP_TENANT", "env-tenant")
	t.Setenv("C8VOLT_HTTP_TIMEOUT", "18s")

	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
http:
  timeout: 30s
profiles:
  dev:
    app:
      tenant: profile-dev
    apis:
      camunda_api:
        base_url: http://dev.example.test
    http:
      timeout: 9s
  prod:
    app:
      tenant: profile-prod
    apis:
      camunda_api:
        base_url: http://prod.example.test
    http:
      timeout: 7s
`)

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
	require.NoError(t, root.PersistentFlags().Set("profile", "prod"))
	require.NoError(t, root.PersistentFlags().Set("tenant", "flag-tenant"))
	require.NoError(t, root.PersistentFlags().Set("timeout", "45s"))

	v := viper.New()
	bindings, err := initViper(v, root)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)

	require.Equal(t, "prod", cfg.ActiveProfile)
	require.Equal(t, "flag-tenant", cfg.App.Tenant)
	require.Equal(t, "45s", cfg.HTTP.Timeout)
	require.Equal(t, "http://prod.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

// Verifies root bootstrap binds command-local backoff flags into the same effective-config resolver.
func TestRetrieveAndNormalizeConfig_UsesSharedResolverForCommandLocalBackoffFlags(t *testing.T) {
	cfgPath := t.TempDir() + "/config.yaml"
	content := `active_profile: dev
app:
  tenant: base-tenant
  backoff:
    timeout: 12s
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
http:
  timeout: 30s
profiles:
  dev:
    app:
      backoff:
        timeout: 9s
`
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o600))

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
	require.NoError(t, root.PersistentFlags().Set("profile", "dev"))
	require.NoError(t, getCmd.PersistentFlags().Set("backoff-timeout", "45s"))

	v := viper.New()
	bindings, err := initViper(v, getCmd)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)

	require.Equal(t, 45*time.Second, cfg.App.Backoff.Timeout)
}

func TestRetrieveAndNormalizeConfig_EnvironmentTenantWinsWhenRootFlagIsUnset(t *testing.T) {
	t.Setenv("C8VOLT_APP_TENANT", "env-tenant")

	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`)

	cfg := resolveCommandConfigForTest(t, getCmd, cfgPath, nil)

	require.Equal(t, "dev", cfg.ActiveProfile)
	require.Equal(t, "env-tenant", cfg.App.Tenant)
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestRetrieveAndNormalizeConfig_ProfileTenantWinsWhenFlagAndEnvAreUnset(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
profiles:
  dev:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`)

	cfg := resolveCommandConfigForTest(t, walkCmd, cfgPath, nil)

	require.Equal(t, "dev", cfg.ActiveProfile)
	require.Equal(t, "profile-tenant", cfg.App.Tenant)
	require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

func TestRetrieveAndNormalizeConfig_BaseConfigTenantWinsWhenNoHigherPrecedenceSourceExists(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  tenant: base-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://base.example.test
`)

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))

	v := viper.New()
	bindings, err := initViper(v, root)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)

	require.Empty(t, cfg.ActiveProfile)
	require.Equal(t, "base-tenant", cfg.App.Tenant)
	require.Equal(t, "http://base.example.test/v2", cfg.APIs.Camunda.BaseURL)
}

// Verifies the critical baseline settings resolve identically across the audited command surface.
func TestRetrieveAndNormalizeConfig_CriticalBaselineSettingsStayAlignedAcrossCommands(t *testing.T) {
	t.Setenv("C8VOLT_ACTIVE_PROFILE", "ignored-env-profile")
	t.Setenv("C8VOLT_AUTH_MODE", "oauth2")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_ID", "env-client")
	t.Setenv("C8VOLT_AUTH_OAUTH2_CLIENT_SECRET", "env-secret")
	t.Setenv("C8VOLT_AUTH_OAUTH2_SCOPES_CAMUNDA_API", "env-scope")

	cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  tenant: base-tenant
auth:
  mode: cookie
  oauth2:
    token_url: http://token.example.test
    client_id: base-client
    client_secret: base-secret
    scopes:
      camunda_api: profile-scope
apis:
  camunda_api:
    base_url: http://base.example.test
    require_scope: true
profiles:
  dev:
    app:
      tenant: profile-tenant
    apis:
      camunda_api:
        base_url: http://profile.example.test
`)

	testCases := []struct {
		name string
		cmd  *cobra.Command
	}{
		{name: "config-show", cmd: configShowCmd},
		{name: "get", cmd: getCmd},
		{name: "cancel", cmd: cancelCmd},
		{name: "delete", cmd: deleteCmd},
		{name: "deploy", cmd: deployCmd},
		{name: "expect", cmd: expectCmd},
		{name: "run", cmd: runCmd},
		{name: "walk", cmd: walkCmd},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := resolveCommandConfigForTest(t, tc.cmd, cfgPath, func(cmd *cobra.Command) {
				require.NoError(t, Root().PersistentFlags().Set("tenant", "flag-tenant"))
			})

			require.Equal(t, "dev", cfg.ActiveProfile)
			require.Equal(t, "flag-tenant", cfg.App.Tenant)
			require.Equal(t, "http://profile.example.test/v2", cfg.APIs.Camunda.BaseURL)
			require.Equal(t, config.ModeOAuth2, cfg.Auth.Mode)
			require.Equal(t, "env-client", cfg.Auth.OAuth2.ClientID)
			require.Equal(t, "env-secret", cfg.Auth.OAuth2.ClientSecret)
			require.Equal(t, "env-scope", cfg.Auth.OAuth2.Scope(config.CamundaApiKeyConst))
		})
	}
}

func TestExecute_ProfileFlagMissingProfileUsesInvalidInputFailureModel(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `active_profile: dev
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
profiles:
  dev:
    app:
      tenant: dev-tenant
`)

	output, err := testx.RunCmdSubprocess(t, "TestExecute_ProfileFlagMissingProfileUsesInvalidInputFailureModelHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), `profile not found: "missing"`)
}

// Helper-process entrypoint for invalid effective-config failure-path validation.
func TestConfigShowCommand_UsesSharedFailureModelForInvalidEffectiveConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevValidate := flagShowConfigValidate
	prevTemplate := flagShowConfigTemplate
	t.Cleanup(func() {
		flagShowConfigValidate = prevValidate
		flagShowConfigTemplate = prevTemplate
	})

	cfg := config.New()
	flagShowConfigValidate = true
	flagShowConfigTemplate = false

	configShowCmd.SetContext(cfg.ToContext(context.Background()))
	configShowCmd.SetOut(os.Stdout)
	configShowCmd.SetErr(os.Stderr)
	configShowCmd.Run(configShowCmd, nil)
}

func TestConfigShowCommand_ValidatePreservesValidOutcomeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "show", "--validate"}

	Execute()
}

func TestConfigShowCommand_ValidatePreservesInvalidOutcomeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "show", "--validate"}

	Execute()
}

func TestConfigDiagnosticValidateHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	switch os.Getenv("C8VOLT_TEST_CONFIG_DIAGNOSTIC") {
	case "show-validate":
		os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "show", "--validate"}
	case "validate":
		os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "validate"}
	default:
		t.Fatalf("unsupported config diagnostic helper command: %q", os.Getenv("C8VOLT_TEST_CONFIG_DIAGNOSTIC"))
	}

	Execute()
}

func TestConfigTestConnectionCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	switch os.Getenv("C8VOLT_TEST_CONFIG_CONNECTION_DIAGNOSTIC") {
	case "test-connection":
		os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "config", "test-connection"}
	default:
		t.Fatalf("unsupported config connection diagnostic helper command: %q", os.Getenv("C8VOLT_TEST_CONFIG_CONNECTION_DIAGNOSTIC"))
	}

	Execute()
}

func TestExecute_ProfileFlagMissingProfileUsesInvalidInputFailureModelHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--profile", "missing", "config", "show"}

	Execute()
}

func writeBackoffPrecedenceConfig(t *testing.T) string {
	t.Helper()

	return writeRawTestConfig(t, `active_profile: dev
app:
  backoff:
    timeout: 12s
    max_retries: 2
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
profiles:
  dev:
    app:
      backoff:
        timeout: 9s
        max_retries: 4
`)
}

func resolveCommandConfigForTest(t *testing.T, cmd *cobra.Command, cfgPath string, configure func(*cobra.Command)) *config.Config {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	require.NoError(t, root.PersistentFlags().Set("config", cfgPath))
	require.NoError(t, root.PersistentFlags().Set("profile", "dev"))
	if configure != nil {
		configure(cmd)
	}

	v := viper.New()
	bindings, err := initViper(v, cmd)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)
	return cfg
}

func writeRawTestConfig(t *testing.T, content string) string {
	t.Helper()
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte(strings.TrimLeft(content, "\n")), 0o600))
	return cfgPath
}
