// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Verifies get command-local backoff flags participate in the shared configuration precedence chain.
func TestGetCommand_CommandLocalBackoffTimeoutFlagOverridesEnvProfileAndConfig(t *testing.T) {
	t.Setenv("C8VOLT_APP_BACKOFF_TIMEOUT", "21s")

	cfg := resolveCommandConfigForTest(t, getCmd, writeBackoffPrecedenceConfig(t), func(cmd *cobra.Command) {
		require.NoError(t, cmd.PersistentFlags().Set("backoff-timeout", "45s"))
	})

	require.Equal(t, 45*time.Second, cfg.App.Backoff.Timeout)
}

// Verifies `get --help` lists supported get subcommands.
func TestGetHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "--help")

	require.Contains(t, output, "Inspect cluster, process, tenant, and resource state")
	require.Contains(t, output, "cluster")
	require.NotContains(t, output, "cluster-topology")
	require.Contains(t, output, "tenant")
	require.Contains(t, output, "resource")
	require.Contains(t, output, "Check cluster health")
	require.Contains(t, output, "./c8volt get cluster topology")
	require.Contains(t, output, "./c8volt get pi --state active")
	require.Contains(t, output, "./c8volt get tenant")
	require.NotContains(t, output, "canonical non-interactive contract")
}

// Verifies process-instance help remains discoverable from the get command family.
func TestGetCommand_PreservesExistingProcessInstanceHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "process-instance", "--help")

	require.Contains(t, output, "Get process instances by key or by search criteria.")
	require.Contains(t, output, "./c8volt get pi --state active")
	require.Contains(t, output, "--key")
	require.Contains(t, output, "--state")
	require.Contains(t, output, "--json")
}

// Verifies root help advertises the finalized v8.9 runtime support contract.
func TestRootHelp_V89SupportMessaging(t *testing.T) {
	output := executeRootForTest(t, "--help")

	require.Contains(t, output, "Camunda 8.7, 8.8, and 8.9")
	require.Contains(t, output, "Camunda 8.7, 8.8, and 8.9")
	require.Contains(t, output, "capabilities")
	require.NotContains(t, output, "version 8.9 is recognized by config normalization")
	require.NotContains(t, output, "does not yet have a process-instance service implementation")
}

func TestCapabilitiesCommand_ReportsRepresentativeFamilyMetadata(t *testing.T) {
	output := executeRootForTest(t, "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(output), &doc))

	var getPI CommandCapability
	var runPI CommandCapability
	for _, command := range doc.Commands {
		if command.Path == "get" {
			for _, child := range command.Children {
				if child.Path == "get process-instance" {
					getPI = child
				}
			}
		}
		if command.Path == "run" {
			for _, child := range command.Children {
				if child.Path == "run process-instance" {
					runPI = child
				}
			}
		}
	}

	require.Equal(t, ContractSupportFull, getPI.ContractSupport)
	require.Equal(t, ContractSupportFull, getPI.ContractSupport)
	require.Equal(t, ContractSupportFull, runPI.ContractSupport)
	require.Equal(t, AutomationSupportFull, runPI.AutomationSupport)
	require.Contains(t, getPI.Flags, FlagContract{
		Name:        "total",
		Shorthand:   "",
		Type:        "bool",
		Required:    false,
		Repeated:    false,
		Description: "return only the numeric total of matching process instances; capped backend totals are counted by paging",
	})
	require.Contains(t, runPI.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.Contains(t, getPI.OutputModes, OutputModeContract{
		Name:             "json",
		Supported:        true,
		MachinePreferred: true,
	})
	require.NotContains(t, getPI.OutputModes, OutputModeContract{
		Name:      "total",
		Supported: true,
	})
}

func TestCapabilitiesCommand_AutomationJSONKeepsStdoutMachineReadable(t *testing.T) {
	stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--automation", "capabilities", "--json")

	var doc CapabilityDocument
	require.NoError(t, json.Unmarshal([]byte(stdout), &doc))
	require.Equal(t, "capabilities", doc.Command)
	require.NotContains(t, stdout, "Machine-readable CLI capabilities")
	require.Empty(t, stderr)
}

// Verifies `get resource --help` documents required id-based lookup usage.
func TestGetResourceHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "resource", "--help")

	require.Contains(t, output, "Get a single resource by id")
	require.Contains(t, output, "Requires --id")
	require.Contains(t, output, "c8volt get resource")
	require.Contains(t, output, "--id")
	require.Contains(t, output, "resource id to fetch")
	require.Contains(t, output, "--keys-only")
}

// Verifies `get cluster --help` exposes nested cluster resource commands.
func TestGetClusterHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "cluster", "--help")

	require.Contains(t, output, "Inspect cluster-wide topology and license information")
	require.Contains(t, output, "Usage:")
	require.Contains(t, output, "c8volt get cluster")
	require.Contains(t, output, "license")
	require.Contains(t, output, "topology")
	require.Contains(t, output, "brokers, partitions, and gateway details")
	require.Contains(t, output, "./c8volt get cluster license")
}

// Verifies `get cluster license --help` describes license retrieval usage.
func TestGetClusterLicenseHelp(t *testing.T) {
	output := executeRootForTest(t, "get", "cluster", "license", "--help")

	require.Contains(t, output, "Show connected cluster license")
	require.Contains(t, output, "license payload returned by the configured Camunda cluster")
	require.Contains(t, output, "c8volt get cluster license")
	require.Contains(t, output, "./c8volt get cluster license --json")
}

func TestGetProcessDefinitionHelp_DocumentsJSONAndXMLModes(t *testing.T) {
	output := executeRootForTest(t, "get", "process-definition", "--help")

	require.Contains(t, output, "List or fetch deployed process definitions")
	require.Contains(t, output, "Inspect deployed BPMN models")
	require.Contains(t, output, "Use `--xml` only with `--key`")
	require.Contains(t, output, "`--stat` requires Camunda `8.8` or `8.9`")
	require.Contains(t, output, "prints exact-version")
	require.Contains(t, output, "Camunda `8.7` does not support")
	require.Contains(t, output, "./c8volt get pd --key <process-definition-key> --json")
}

// Verifies get commands consume env-overridden oauth2 scopes when authenticating against the configured API.
func TestGetClusterTopologyCommand_UsesEnvOAuth2ScopeOverride(t *testing.T) {
	t.Setenv("C8VOLT_AUTH_OAUTH2_SCOPES_CAMUNDA_API", "env-scope")

	tokenSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/token", r.URL.Path)
		require.NoError(t, r.ParseForm())
		require.Equal(t, "client_credentials", r.Form.Get("grant_type"))
		require.Equal(t, "env-scope", r.Form.Get("scope"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"env-token","token_type":"Bearer"}`))
	}))
	t.Cleanup(tokenSrv.Close)

	apiSrv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		require.Equal(t, "Bearer env-token", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.8.0", 1, 1)))
	}))
	t.Cleanup(apiSrv.Close)

	cfgPath := writeRawTestConfig(t, "auth:\n  mode: oauth2\n  oauth2:\n    token_url: "+tokenSrv.URL+"\n    client_id: base-client\n    client_secret: base-secret\n    scopes:\n      camunda_api: profile-scope\napis:\n  camunda_api:\n    base_url: "+apiSrv.URL+"\n    require_scope: true\n")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "topology")

	require.Contains(t, output, `"ClusterSize": 1`)
}

// Verifies nested `get cluster topology` succeeds and renders topology fields.
func TestGetClusterTopologyNestedCommand_Success(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/topology", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(singleBrokerClusterTopologyFixtureJSON()))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "topology")

	require.Contains(t, output, `"GatewayVersion": "8.8.0"`)
	require.Contains(t, output, `"ClusterSize": 1`)
}

// Verifies the removed direct cluster-topology command path and aliases no longer resolve.
func TestGetClusterTopologyLegacyCommand_NotFound(t *testing.T) {
	for _, args := range [][]string{
		{"get", "cluster-topology"},
		{"get", "ct"},
		{"get", "cluster-info"},
		{"get", "ci"},
	} {
		output, err := executeRootExpectErrorForTest(t, args...)

		require.Error(t, err)
		require.Contains(t, err.Error(), "unknown command")
		require.Contains(t, output, "Usage:")
	}
}

func TestGetResourceCommand_DefaultOutputRemainsPlainText(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "resourceId": "resource-id-123",
  "resourceKey": "resource-key-123",
  "resourceName": "order-process.bpmn",
  "tenantId": "<default>",
  "version": 7,
  "versionTag": "stable"
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "resource", "--id", "resource-id-123")

	require.Contains(t, output, "resource-id-123")
	require.NotContains(t, output, `"outcome"`)
	require.NotContains(t, output, `"command"`)
}

// Verifies nested `get cluster license` succeeds with required license fields.
func TestGetClusterLicenseNestedCommand_Success(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/license", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(requiredClusterLicenseFixtureJSON()))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "license")

	require.Contains(t, output, `"LicenseType": "SaaS"`)
	require.Contains(t, output, `"ValidLicense": true`)
	require.NotContains(t, output, `"ExpiresAt": null`)
	require.NotContains(t, output, `"IsCommercial": null`)
	require.NotContains(t, output, `"ExpiresAt"`)
	require.NotContains(t, output, `"IsCommercial"`)
}

// Verifies optional license fields are rendered when the API returns them.
func TestGetClusterLicenseNestedCommand_SuccessWithOptionalFields(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/license", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(optionalClusterLicenseFixtureJSON()))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "license")

	require.Contains(t, output, `"ExpiresAt": "2030-01-02T03:04:05Z"`)
	require.Contains(t, output, `"IsCommercial": true`)
	require.Contains(t, output, `"LicenseType": "Enterprise"`)
	require.Contains(t, output, `"ValidLicense": true`)
}

// Verifies cluster license HTTP failures map to unavailable exit behavior.
func TestGetClusterLicenseNestedCommand_Failure(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/license", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	output, err := testx.RunCmdSubprocess(t, "TestGetClusterLicenseNestedCommand_FailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "get cluster license")
	require.NotContains(t, string(output), "error fetching cluster license")
	require.NotContains(t, string(output), "fetch cluster license")
}

// Verifies nested cluster topology HTTP failures map to unavailable exit behavior.
func TestGetClusterTopologyNestedCommand_Failure(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/topology", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	output, err := testx.RunCmdSubprocess(t, "TestGetClusterTopologyNestedCommand_FailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "get cluster topology")
	require.NotContains(t, string(output), "error fetching topology")
	require.NotContains(t, string(output), "fetch cluster topology")
}

func TestGetCommand_V89SupportsClusterProcessDefinitionAndResource(t *testing.T) {
	t.Run("cluster topology", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/topology", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(emptyClusterTopologyFixtureJSON(1, "8.9.0", 1, 1)))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		output := executeRootForTest(t, "--config", cfgPath, "get", "cluster", "topology")

		require.Contains(t, output, "8.9.0")
	})

	t.Run("process definition lookup", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-definitions/2251799813685255", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionKey":"2251799813685255","processDefinitionName":"order-process","processDefinitionVersion":3,"processDefinitionVersionTag":"stable","tenantId":"tenant"}`))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		output := executeRootForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "2251799813685255")

		require.Contains(t, output, "order-process")
		require.Contains(t, output, "2251799813685255")
	})

	t.Run("resource lookup", func(t *testing.T) {
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"resourceId":"resource-id-123","resourceKey":"resource-key-123","resourceName":"order-process.bpmn","version":2,"tenantId":"tenant"}`))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

		output := executeRootForTest(t, "--config", cfgPath, "get", "resource", "--id", "resource-id-123")

		require.Contains(t, output, "resource-id-123")
		require.Contains(t, output, "k:resource-key-123")
	})
}

// Verifies malformed successful license responses are classified as malformed-response failures.
func TestGetClusterLicenseNestedCommand_MalformedResponse(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v2/license", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	cfgPath := writeTestConfig(t, srv.URL)

	output, err := testx.RunCmdSubprocess(t, "TestGetClusterLicenseNestedCommand_MalformedResponseHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "get cluster license")
	require.NotContains(t, string(output), "error fetching cluster license")
	require.NotContains(t, string(output), "fetch cluster license")
	require.Contains(t, string(output), "malformed response")
}

// Verifies `get process-definition --xml` returns raw XML content by key.
func TestGetProcessDefinitionXMLCommand_Success(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255/xml", r.URL.Path)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte("<definitions id=\"order-process\"/>"))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "2251799813685255", "--xml")

	require.Equal(t, "<definitions id=\"order-process\"/>", output)
}

// Verifies XML output preserves formatting and line breaks from the API payload.
func TestGetProcessDefinitionXMLCommand_PreservesFormattedXMLBody(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255/xml", r.URL.Path)
		w.Header().Set("Content-Type", "application/xml")
		_, _ = w.Write([]byte("<definitions id=\"order-process\">\n  <process id=\"order\" />\n</definitions>\n"))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "2251799813685255", "--xml")

	require.Equal(t, "<definitions id=\"order-process\">\n  <process id=\"order\" />\n</definitions>\n", output)
}

// Verifies process-definition key lookup renders model output instead of XML.
func TestGetProcessDefinitionByKeyCommand_Success(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "processDefinitionId": "order-process",
  "processDefinitionKey": "2251799813685255",
  "tenantId": "<default>",
  "version": 7,
  "versionTag": "stable"
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "2251799813685255")

	require.Contains(t, output, "2251799813685255")
	require.Contains(t, output, "<default> order-process v7/stable")
	require.NotContains(t, output, "<definitions")
}

func TestGetProcessDefinitionLatest_UsesEffectiveTenantForSearch(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "items": [
    {
      "processDefinitionId": "order-process",
      "processDefinitionKey": "2251799813685255",
      "tenantId": "tenant-a",
      "version": 7,
      "versionTag": "stable"
    }
  ],
  "page": {
    "totalItems": 1,
    "hasMoreTotalItems": false
  }
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output := executeRootForTest(t,
		"--config", cfgPath,
		"--tenant", "tenant-a",
		"get", "process-definition",
		"--latest",
	)

	body := decodeSingleRequestJSON(t, requests)
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok, "expected search request filter object")
	require.Equal(t, "tenant-a", filter["tenantId"])
	require.Equal(t, true, filter["isLatestVersion"])
	require.Contains(t, output, "tenant-a order-process v7/stable")
	require.NotContains(t, output, "base-tenant")
}

func TestOneLinePD_IncidentCountRenderingByVersionBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		versionLabel      string
		incidents         int64
		supported         bool
		expectedSegment   string
		unexpectedSegment string
	}{
		{
			name:              "v8.8 renders supported non-zero incident count",
			versionLabel:      "8.8",
			incidents:         3,
			supported:         true,
			expectedSegment:   "[ac:4 cp:9 cx:2 inc:3]",
			unexpectedSegment: "inc:-",
		},
		{
			name:              "v8.9 renders supported zero incident count",
			versionLabel:      "8.9",
			incidents:         0,
			supported:         true,
			expectedSegment:   "[ac:4 cp:9 cx:2 inc:-]",
			unexpectedSegment: "inc:0",
		},
		{
			name:              "v8.7 omits unsupported incident count",
			versionLabel:      "8.7",
			incidents:         7,
			supported:         false,
			expectedSegment:   "[ac:4 cp:9 cx:2]",
			unexpectedSegment: " inc:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := oneLinePD(process.ProcessDefinition{
				Key:               "2251799813685255",
				TenantId:          "<default>",
				BpmnProcessId:     "order-process",
				ProcessVersion:    7,
				ProcessVersionTag: "v" + tt.versionLabel,
				Statistics: &process.ProcessDefinitionStatistics{
					Active:                 4,
					Completed:              9,
					Canceled:               2,
					Incidents:              tt.incidents,
					IncidentCountSupported: tt.supported,
				},
			})

			require.Contains(t, got, tt.expectedSegment)
			require.NotContains(t, got, tt.unexpectedSegment)
		})
	}
}

// Verifies `get resource --id` succeeds and renders default table output.
func TestGetResourceCommand_Success(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "resourceId": "resource-id-123",
  "resourceKey": "resource-key-123",
  "resourceName": "order-process.bpmn",
  "tenantId": "<default>",
  "version": 7,
  "versionTag": "stable"
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "get", "resource", "--id", "resource-id-123")

	require.Contains(t, output, "resource-id-123")
	require.Contains(t, output, "k:resource-key-123")
	require.Contains(t, output, "<default>")
	require.Contains(t, output, "order-process.bpmn")
	require.Contains(t, output, "v7/stable")
}

// Verifies resource lookup JSON view uses serialized model field keys.
func TestGetResourceCommand_JSONOutput(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "resourceId": "resource-id-123",
  "resourceKey": "resource-key-123",
  "resourceName": "order-process.bpmn",
  "tenantId": "<default>",
  "version": 7,
  "versionTag": "stable"
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "--json", "get", "resource", "--id", "resource-id-123")

	require.Contains(t, output, `"id": "resource-id-123"`)
	require.Contains(t, output, `"key": "resource-key-123"`)
	require.Contains(t, output, `"name": "order-process.bpmn"`)
	require.Contains(t, output, `"tenantId": "<default>"`)
	require.Contains(t, output, `"version": 7`)
	require.Contains(t, output, `"versionTag": "stable"`)
}

// Verifies keys-only mode emits only the resource id.
func TestGetResourceCommand_KeysOnlyOutput(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "resourceId": "resource-id-123",
  "resourceKey": "resource-key-123",
  "resourceName": "order-process.bpmn",
  "tenantId": "<default>",
  "version": 7,
  "versionTag": "stable"
}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "--keys-only", "get", "resource", "--id", "resource-id-123")

	require.Equal(t, "resource-id-123\n", output)
}

// Verifies resource lookup HTTP failures map to not-found exit behavior.
func TestGetResourceCommand_Failure(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/missing-resource", r.URL.Path)
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetResourceCommand_FailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.NotFound, exitErr.ExitCode())
	require.Contains(t, string(output), "get resource")
	require.Contains(t, string(output), "resource not found")
	require.NotContains(t, string(output), "error fetching resource by id missing-resource")
}

func TestGetResourceCommand_JSONFailureUsesEnvelope(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/missing-resource", r.URL.Path)
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetResourceCommand_JSONFailureUsesEnvelopeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.NotFound, exitErr.ExitCode())

	var got map[string]any
	require.NoError(t, json.Unmarshal(output, &got))
	require.Equal(t, string(OutcomeFailed), got["outcome"])
	require.Equal(t, "get resource", got["command"])
	require.Equal(t, "not_found", got["class"])
}

// Verifies `--no-err-codes` preserves failure output while returning success exit status.
func TestGetResourceCommand_NoErrCodesKeepsFailureOutputButReturnsSuccessExit(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/missing-resource", r.URL.Path)
		http.Error(w, "missing", http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetResourceCommand_NoErrCodesKeepsFailureOutputButReturnsSuccessExitHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err)
	require.Contains(t, string(output), "resource not found")
	require.Contains(t, string(output), "get resource")
	require.NotContains(t, string(output), "error fetching resource by id missing-resource")
}

// Verifies resource command rejects missing `--id` input.
func TestGetResourceCommand_RequiresID(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := executeRootExpectErrorForTest(t, "--config", cfgPath, "get", "resource")

	require.Error(t, err)
	require.Contains(t, err.Error(), "resource lookup requires a non-empty --id")
	require.Contains(t, output, "resource lookup requires a non-empty --id")
}

// Verifies resource command rejects whitespace-only `--id` values.
func TestGetResourceCommand_RejectsWhitespaceID(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := executeRootExpectErrorForTest(t, "--config", cfgPath, "get", "resource", "--id", "   ")

	require.Error(t, err)
	require.Contains(t, err.Error(), "resource lookup requires a non-empty --id")
	require.Contains(t, output, "resource lookup requires a non-empty --id")
}

func TestGetProcessInstanceKeyLookup_WrongTenantLooksLikeNotFound(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)
		requests = append(requests, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"message":"not found"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceKeyLookupWrongTenantHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.NotFound, exitErr.ExitCode())
	require.Len(t, requests, 1)
	require.Contains(t, string(output), "resource not found")
	require.Contains(t, string(output), "get process instance")
	require.NotContains(t, string(output), "error fetching process instance")
	require.NotContains(t, string(output), "missing ancestor keys")
	require.NotContains(t, string(output), "parent process instances were not found")
}

func TestGetProcessInstanceKeyLookup_V87ReportsUnsupported(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.7
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
`)

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceKeyLookupUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "not tenant-safe in Camunda 8.7")
}

func TestGetProcessInstanceOrphanChildrenOnly_V87ReportsUnsupported(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.7
apis:
  camunda_api:
    base_url: http://127.0.0.1:1
`)

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceOrphanChildrenOnlyUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "--orphan-children-only is not supported in Camunda 8.7")
}

// Verifies whitespace-only `--id` failures exit through invalid-args handling.
func TestGetResourceCommand_RejectsWhitespaceID_UsesInvalidArgsExit(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestGetResourceCommand_RejectsWhitespaceID_UsesInvalidArgsExitHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "resource lookup requires a non-empty --id")
}

// Verifies malformed successful resource payloads are treated as malformed-response failures.
func TestGetResourceCommand_MalformedResponse(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/resource-id-123", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"detail":"missing payload"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetResourceCommand_MalformedResponseHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "get resource")
	require.Contains(t, string(output), "malformed response")
	require.NotContains(t, string(output), "error fetching resource by id resource-id-123")
}

// Verifies gateway timeout responses map to timeout exit behavior.
func TestGetResourceCommand_GatewayTimeoutUsesTimeoutExitCode(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/resources/timeout-resource", r.URL.Path)
		http.Error(w, "slow", http.StatusGatewayTimeout)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetResourceCommand_GatewayTimeoutUsesTimeoutExitCodeHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Timeout, exitErr.ExitCode())
	require.Contains(t, string(output), "operation timed out")
	require.Contains(t, string(output), "get resource")
	require.NotContains(t, string(output), "error fetching resource by id timeout-resource")
}

// Verifies XML mode requires `--key` to target a single process definition.
func TestGetProcessDefinitionXMLCommand_RequiresKey(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionXMLCommand_RequiresKeyHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "xml output requires --key")
}

// Verifies XML mode rejects incompatible presentation and selection flags.
func TestGetProcessDefinitionXMLCommand_RejectsIncompatibleFlags(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionXMLCommand_RejectsIncompatibleFlagsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "xml output only supports --key")
	require.Contains(t, string(output), "--json")
	require.Contains(t, string(output), "--latest")
}

// Verifies process-definition XML HTTP failures map to unavailable exit behavior.
func TestGetProcessDefinitionXMLCommand_Failure(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-definitions/2251799813685255/xml", r.URL.Path)
		http.Error(w, "boom", http.StatusServiceUnavailable)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessDefinitionXMLCommand_FailureHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Unavailable, exitErr.ExitCode())
	require.Contains(t, string(output), "get process definition xml")
	require.NotContains(t, string(output), "error fetching process definition xml by key 2251799813685255")
	require.NotContains(t, string(output), "<definitions")
}

// Helper-process entrypoint for nested cluster-topology failure-path coverage.
func TestGetClusterTopologyNestedCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "topology"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for malformed cluster-license response coverage.
func TestGetClusterLicenseNestedCommand_MalformedResponseHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "license"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for cluster-license HTTP failure-path coverage.
func TestGetClusterLicenseNestedCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "cluster", "license"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for XML command missing-key validation.
func TestGetProcessDefinitionXMLCommand_RequiresKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-definition", "--xml"}

	Execute()
}

// Helper-process entrypoint for whitespace resource-id invalid-args validation.
func TestGetResourceCommand_RejectsWhitespaceID_UsesInvalidArgsExitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "resource", "--id", "   "}

	Execute()
}

// Helper-process entrypoint for resource lookup HTTP failure-path coverage.
func TestGetResourceCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "resource", "--id", "missing-resource"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestGetResourceCommand_JSONFailureUsesEnvelopeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "get", "resource", "--id", "missing-resource"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for `--no-err-codes` resource failure behavior coverage.
func TestGetResourceCommand_NoErrCodesKeepsFailureOutputButReturnsSuccessExitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--no-err-codes", "get", "resource", "--id", "missing-resource"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for malformed resource response coverage.
func TestGetResourceCommand_MalformedResponseHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "resource", "--id", "resource-id-123"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for timeout exit-code mapping coverage.
func TestGetResourceCommand_GatewayTimeoutUsesTimeoutExitCodeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "resource", "--id", "timeout-resource"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for XML incompatible-flag validation.
func TestGetProcessDefinitionXMLCommand_RejectsIncompatibleFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--json", "get", "process-definition", "--key", "2251799813685255", "--xml", "--latest"}

	Execute()
}

// Helper-process entrypoint for process-definition XML HTTP failure coverage.
func TestGetProcessDefinitionXMLCommand_FailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	resetCommandTreeFlags(root)
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-definition", "--key", "2251799813685255", "--xml"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for tenant-safe keyed process-instance not-found coverage.
func TestGetProcessInstanceKeyLookupWrongTenantHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant-a", "get", "process-instance", "--key", "123"}

	Execute()
}

// Helper-process entrypoint for unsupported v8.7 keyed process-instance lookup coverage.
func TestGetProcessInstanceKeyLookupUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123"}

	Execute()
}

// Helper-process entrypoint for unsupported v8.7 orphan-child process-instance filtering coverage.
func TestGetProcessInstanceOrphanChildrenOnlyUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--orphan-children-only"}

	Execute()
}

func executeRootForTest(t *testing.T, args ...string) string {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}

func executeRootWithSeparateOutputsForTest(t *testing.T, args ...string) (string, string) {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return stdout.String(), stderr.String()
}

func executeRootExpectErrorForTest(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := Root()
	resetCommandTreeFlags(root)
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)

	_, err := root.ExecuteC()
	return buf.String(), err
}

func executeCompletionForTest(t *testing.T, args ...string) string {
	t.Helper()

	completionArgs := append([]string{"__complete"}, args...)
	return executeRootForTest(t, completionArgs...)
}

func executeCompletionNoDescForTest(t *testing.T, args ...string) string {
	t.Helper()

	completionArgs := append([]string{"__completeNoDesc"}, args...)
	return executeRootForTest(t, completionArgs...)
}

func resetCommandTreeFlags(cmd *cobra.Command) {
	testx.ResetCommandTreeFlags(cmd)
}

func singleBrokerClusterTopologyFixtureJSON() string {
	return `{
  "brokers": [
    {
      "host": "camunda-platform-c88-zeebe-0.camunda-platform-c88-zeebe",
      "nodeId": 0,
      "partitions": [
        {
          "health": "healthy",
          "partitionId": 1,
          "role": "leader"
        }
      ],
      "port": 26501,
      "version": "8.8.0"
    }
  ],
  "clusterSize": 1,
  "gatewayVersion": "8.8.0",
  "partitionsCount": 1,
  "replicationFactor": 1,
  "lastCompletedChangeId": ""
}`
}

func emptyClusterTopologyFixtureJSON(clusterSize int, gatewayVersion string, partitionsCount int, replicationFactor int) string {
	return fmt.Sprintf(`{
  "brokers": [],
  "clusterSize": %d,
  "gatewayVersion": %q,
  "partitionsCount": %d,
  "replicationFactor": %d,
  "lastCompletedChangeId": ""
}`, clusterSize, gatewayVersion, partitionsCount, replicationFactor)
}

func requiredClusterLicenseFixtureJSON() string {
	return `{
  "licenseType": "SaaS",
  "validLicense": true
}`
}

func optionalClusterLicenseFixtureJSON() string {
	return `{
  "expiresAt": "2030-01-02T03:04:05Z",
  "isCommercial": true,
  "licenseType": "Enterprise",
  "validLicense": true
}`
}

func writeTestConfig(t *testing.T, baseURL string) string {
	t.Helper()
	return testx.WriteTestConfig(t, baseURL)
}

func writeTestConfigForVersion(t *testing.T, baseURL string, camundaVersion string) string {
	t.Helper()
	return testx.WriteTestConfigForVersion(t, baseURL, camundaVersion)
}
