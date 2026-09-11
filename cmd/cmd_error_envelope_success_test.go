// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

const commandErrorEnvelopeSuccessHelper = "TestCommandErrorEnvelopeSuccessHelper"

// TestCommandErrorEnvelopeSuccess verifies the error-dispatch correction leaves
// representative successful command results and existing tenant metadata intact.
func TestCommandErrorEnvelopeSuccess(t *testing.T) {
	t.Run("flag and stdin keys retain order and caller deduplication", func(t *testing.T) {
		var requests testx.SafeSlice[string]
		server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			key := strings.TrimPrefix(r.URL.Path, "/v2/process-instances/")
			require.Contains(t, []string{"2251799813711967", "2251799813711968", "2251799813711969"}, key)
			requests.Append(key)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":1,"processInstanceKey":"` + key + `","startDate":"2026-09-11T12:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`))
		}))
		t.Cleanup(server.Close)

		stdout, stderr, err := testx.RunCmdSubprocessInDirWithSeparateOutputs(t, commandErrorEnvelopeSuccessHelper, "", map[string]string{
			"C8VOLT_TEST_CONFIG": testx.WriteTestConfig(t, server.URL),
		}, "2251799813711968\n2251799813711969\n")
		require.NoError(t, err, "stdout=%q stderr=%q", stdout, stderr)
		require.Empty(t, stderr)

		envelope := requireSingleJSONObjectDocument(t, stdout)
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		require.Equal(t, "get process-instance", envelope["command"])
		items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 3)
		require.Equal(t, "2251799813711967", requireJSONObject(t, items[0])["key"])
		require.Equal(t, "2251799813711968", requireJSONObject(t, items[1])["key"])
		require.Equal(t, "2251799813711969", requireJSONObject(t, items[2])["key"])
		require.ElementsMatch(t, []string{"2251799813711967", "2251799813711968", "2251799813711969"}, requests.Snapshot())
		require.NotContains(t, envelope, "tenantContext")
	})

	t.Run("cluster human and JSON results remain successful", func(t *testing.T) {
		server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			switch r.URL.Path {
			case "/v2/topology":
				require.Equal(t, http.MethodGet, r.Method)
				_, _ = w.Write([]byte(unsortedClusterTopologyFixtureJSON()))
			case "/v2/license":
				require.Equal(t, http.MethodGet, r.Method)
				_, _ = w.Write([]byte(optionalClusterLicenseFixtureJSON()))
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		}))
		t.Cleanup(server.Close)
		cfgPath := writeTestConfigForVersion(t, server.URL, "8.8")

		human, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "get", "cluster", "topology")
		require.Empty(t, stderr)
		require.Contains(t, human, "Cluster: GatewayVersion=8.8.2")

		stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "--json", "get", "cluster", "license")
		require.Empty(t, stderr)
		envelope := requireSingleJSONObjectDocument(t, stdout)
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		require.Equal(t, "get cluster license", envelope["command"])
		require.Equal(t, "Enterprise", requireJSONObject(t, envelope["payload"])["LicenseType"])
		require.NotContains(t, envelope, "tenantContext")
	})

	t.Run("process-definition list key and XML outputs remain distinct", func(t *testing.T) {
		var requests testx.SafeSlice[string]
		server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requests.Append(r.Method + " " + r.URL.Path)
			switch {
			case r.Method == http.MethodPost && r.URL.Path == "/v2/process-definitions/search":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"items":[{"processDefinitionKey":"7001","processDefinitionId":"invoice","name":"Invoice","version":3,"tenantId":"tenant-a","versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/7001":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"processDefinitionKey":"7001","processDefinitionId":"invoice","name":"Invoice","version":3,"tenantId":"tenant-a","versionTag":"stable"}`))
			case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/7001/xml":
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(`<definitions id="invoice"/>`))
			default:
				t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		}))
		t.Cleanup(server.Close)
		cfgPath := writeTestConfigForVersion(t, server.URL, "8.8")

		stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "--json", "get", "process-definition")
		require.Empty(t, stderr)
		envelope := requireSingleJSONObjectDocument(t, stdout)
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		require.Equal(t, "get process-definition", envelope["command"])
		requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)
		require.NotContains(t, envelope, "tenantContext")

		human, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "7001")
		require.Empty(t, stderr)
		require.Contains(t, human, "7001 tenant-a invoice v3/stable")

		xml, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "get", "process-definition", "--key", "7001", "--xml")
		require.Empty(t, stderr)
		require.Equal(t, `<definitions id="invoice"/>`, xml)
		require.Equal(t, []string{
			"POST /v2/process-definitions/search",
			"GET /v2/process-definitions/7001",
			"GET /v2/process-definitions/7001/xml",
		}, requests.Snapshot())
	})

	t.Run("embedded human and JSON listings remain filtered", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.9")

		human, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "embed", "list")
		require.Empty(t, stderr)
		require.Contains(t, human, "C89_SimpleUserTask.bpmn")
		require.NotContains(t, human, "C88_")

		stdout, stderr := executeRootWithSeparateOutputsForTest(t, "--config", cfgPath, "--json", "embed", "list")
		require.Empty(t, stderr)
		envelope := requireSingleJSONObjectDocument(t, stdout)
		require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
		require.Equal(t, "embed list", envelope["command"])
		require.NotEmpty(t, requireJSONItems(t, envelope["payload"], 8))
		require.NotContains(t, envelope, "tenantContext")
	})

	t.Run("attached tenant context remains beside the payload", func(t *testing.T) {
		cmd := &cobra.Command{Use: "success-fixture"}
		setContractSupport(cmd, ContractSupportFull)
		attachTenantContext(cmd, newExplicitKeysTenantContext("tenant-a"))
		output := &bytes.Buffer{}
		cmd.SetOut(output)

		require.NoError(t, renderSucceededResult(cmd, map[string]string{"value": "ok"}))
		envelope := requireSingleJSONObjectDocument(t, output.String())
		tenantContext := requireJSONObject(t, envelope["tenantContext"])
		require.Equal(t, "explicit_keys", tenantContext["mode"])
		require.Equal(t, "not_applied", tenantContext["filter"])
		require.Equal(t, "tenant-a", tenantContext["configuredTenantId"])
		require.NotContains(t, requireJSONObject(t, envelope["payload"]), "tenantContext")
	})
}

// TestCommandErrorEnvelopeSuccessHelper executes a real mixed flag/stdin key
// lookup so success coverage observes caller-owned stable deduplication.
func TestCommandErrorEnvelopeSuccessHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	require.Equal(t, commandErrorEnvelopeSuccessHelper, os.Getenv(testx.CmdSubprocessNameEnv))

	os.Args = []string{
		"c8volt",
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
		"--key", "2251799813711968",
		"-",
	}
	Execute()
	os.Exit(0)
}
