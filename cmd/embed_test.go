// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestEmbedListHelp_DocumentsReadOnlyDiscoveryExamples(t *testing.T) {
	output := executeRootForTest(t, "embed", "list", "--help")

	require.Contains(t, output, "List bundled BPMN fixture files")
	require.Contains(t, output, "Shows files for the configured Camunda version")
	require.Contains(t, output, "./c8volt embed list --details")
	require.Contains(t, output, "./c8volt --json embed list")
}

func TestEmbedListCommand_FiltersFilesForConfiguredCamundaVersion(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	output := executeRootForTest(t, "--config", cfgPath, "embed", "list")

	require.Contains(t, output, "C88_SimpleUserTask.bpmn")
	require.Contains(t, output, "C88_MultipleSubProcessesParent.bpmn")
	require.NotContains(t, output, "C87_")
	require.NotContains(t, output, "C89_")
	require.NotContains(t, output, "processdefinitions/")
}

func TestEmbedListCommand_DetailsFiltersFilesForConfiguredCamundaVersion(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.9")

	output := executeRootForTest(t, "--config", cfgPath, "embed", "list", "--details")

	require.Contains(t, output, "processdefinitions/C89_SimpleUserTask.bpmn")
	require.Contains(t, output, "processdefinitions/C89_MultipleSubProcessesParent.bpmn")
	require.NotContains(t, output, "processdefinitions/C87_")
	require.NotContains(t, output, "processdefinitions/C88_")
}

// TestEmbedListCommand_V810UsesNativeProductionFixtures verifies the shared embed selector exposes only the native C810 family.
func TestEmbedListCommand_V810UsesNativeProductionFixtures(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.10")

	output := executeRootForTest(t, "--config", cfgPath, "embed", "list", "--details")

	require.Equal(t, strings.Join([]string{
		"processdefinitions/C810_DoubleUserTask.bpmn",
		"processdefinitions/C810_MultipleSubProcessesParent.bpmn",
		"processdefinitions/C810_NoOpCompletion.bpmn",
		"processdefinitions/C810_SimpleParent.bpmn",
		"processdefinitions/C810_SimpleParentWithIncidentSubprocess.bpmn",
		"processdefinitions/C810_SimpleServiceTask.bpmn",
		"processdefinitions/C810_SimpleUserTask.bpmn",
		"processdefinitions/C810_SimpleUserTaskWithIncident.bpmn",
	}, "\n"), strings.TrimSpace(output))
	require.NotContains(t, output, "processdefinitions/C87_")
	require.NotContains(t, output, "processdefinitions/C88_")
	require.NotContains(t, output, "processdefinitions/C89_")
}

func TestEmbedExportHelp_DocumentsSelectionWorkflow(t *testing.T) {
	output := executeRootForTest(t, "embed", "export", "--help")

	require.Contains(t, output, "Export bundled BPMN fixtures to local files")
	require.Contains(t, output, "Use --all for the configured Camunda version")
	require.Contains(t, output, "./c8volt embed export --all --out ./fixtures")
	require.Contains(t, output, "quote patterns in the shell like zsh")
}

// Verifies embed deploy keeps --run as a creation shortcut without adding state expectation flags.
func TestEmbedDeployHelp_DocumentsRunWithoutExpectationFlags(t *testing.T) {
	output := executeRootForTest(t, "embed", "deploy", "--help")

	require.Contains(t, output, "Add --run to start one process instance")
	require.Contains(t, output, "does not accept --all-tenants because it creates resources in one concrete tenant")
	require.Contains(t, output, "--run")
	require.NotContains(t, output, "--expected-status")
}

func TestEmbedExportCommand_AllFiltersFilesForConfiguredCamundaVersion(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")
	outDir := t.TempDir()

	output := executeRootForTest(t, "--config", cfgPath, "embed", "export", "--all", "--out", outDir)

	require.Contains(t, output, "exported")
	require.FileExists(t, filepath.Join(outDir, "processdefinitions", "C88_SimpleUserTask.bpmn"))
	require.FileExists(t, filepath.Join(outDir, "processdefinitions", "C88_MultipleSubProcessesParent.bpmn"))
	require.NoFileExists(t, filepath.Join(outDir, "processdefinitions", "C87_SimpleUserTask.bpmn"))
	require.NoFileExists(t, filepath.Join(outDir, "processdefinitions", "C89_SimpleUserTask.bpmn"))
}

func TestEmbedExportCommand_FileSelectionCanStillExportOtherVersions(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")
	outDir := t.TempDir()

	output, err := testx.RunCmdSubprocess(t, "TestEmbedExportCommand_FileSelectionCanStillExportOtherVersionsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG":     cfgPath,
		"C8VOLT_TEST_EXPORT_OUT": outDir,
	})
	require.NoError(t, err, string(output))

	require.Contains(t, string(output), "exported 1 embedded resource")
	require.FileExists(t, filepath.Join(outDir, "processdefinitions", "C89_SimpleUserTask.bpmn"))
	require.NoFileExists(t, filepath.Join(outDir, "processdefinitions", "C88_SimpleUserTask.bpmn"))
}

func TestEmbedDeployCommand_RegressionPreservesSelectedFixtureDeployOnly(t *testing.T) {
	resetEmbedCommandStateForTest()
	var sawDeploy bool

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			reader, err := r.MultipartReader()
			require.NoError(t, err)
			for {
				part, err := reader.NextPart()
				require.NoError(t, err)
				if part.FormName() != "resources" {
					continue
				}
				_, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
				require.NoError(t, err)
				require.Equal(t, "processdefinitions/C89_MultipleSubProcessesParent.bpmn", params["filename"])
				break
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deploymentKey":"deployment-188","tenantId":"<default>","deployments":[{"processDefinition":{"processDefinitionId":"C89_MultipleSubProcessesParent","processDefinitionKey":"188001","processDefinitionVersion":1,"resourceName":"processdefinitions/C89_MultipleSubProcessesParent.bpmn","tenantId":"<default>"}}]}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-definitions/188001":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processDefinitionId":"C89_MultipleSubProcessesParent","processDefinitionKey":"188001","processDefinitionVersion":1,"resourceName":"processdefinitions/C89_MultipleSubProcessesParent.bpmn","tenantId":"<default>"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output, err := testx.RunCmdSubprocess(t, "TestEmbedDeployCommand_RegressionPreservesSelectedFixtureDeployOnlyHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err, string(output))
	require.True(t, sawDeploy)
}

// TestEmbedDeployCommand_AllTenantsRejectsBeforeEmbeddedOrRequestWork proves
// embedded deployment cannot inspect fixture selections or submit deployments
// when all-tenants is active.
func TestEmbedDeployCommand_AllTenantsRejectsBeforeEmbeddedOrRequestWork(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "unknown fixture",
			args: []string{"embed", "deploy", "--all-tenants", "--file", "processdefinitions/DOES_NOT_EXIST.bpmn"},
		},
		{
			name: "all with optional run",
			args: []string{"--all-tenants", "embed", "deploy", "--all", "--run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Append(r.Method + " " + r.URL.Path)
				t.Fatalf("embed deploy must reject --all-tenants before request work: %s %s", r.Method, r.URL.Path)
			}))
			t.Cleanup(srv.Close)
			args := append([]string{"--config", writeTestConfigForVersion(t, srv.URL, "8.9")}, tt.args...)

			output, err := testx.RunCmdSubprocess(t, "TestEmbedDeployCommand_AllTenantsRejectsBeforeEmbeddedOrRequestWorkHelper", map[string]string{
				"C8VOLT_TEST_ROOT_ARGS": marshalRootArgsForEnv(t, args),
			})
			assertAllTenantsConcreteDestinationSubprocessFailure(t, output, err, "embed deploy")
			require.Empty(t, requests.Snapshot())
			require.NotContains(t, string(output), "embedded file")
			require.NotContains(t, string(output), "deploying embedded resource")
			require.NotContains(t, string(output), "creation target:")
		})
	}
}

func TestEmbedDeployCommand_AllRunFallsBackToBPMNIDForV87(t *testing.T) {
	resetEmbedCommandStateForTest()
	var sawDeploy bool
	var sawRun bool

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tenantId":"<default>"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances":
			sawRun = true
			defer r.Body.Close()

			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			require.NotEmpty(t, body["processDefinitionId"])
			require.Equal(t, "<default>", body["tenantId"])

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"processDefinitionId":"order-process","processDefinitionVersion":1,"tenantId":"<default>","variables":{}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestEmbedDeployCommand_AllRunFallsBackToBPMNIDForV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err, string(output))
	require.True(t, sawDeploy)
	require.True(t, sawRun)
}

func TestEmbedDeployCommand_CreationContextPrecedesDeploymentRequest(t *testing.T) {
	resetEmbedCommandStateForTest()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	var sawDeploy bool

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			require.Contains(t, stderr.String(), "creation target: tenant-a")
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.Equal(t, "tenant-a", r.FormValue("tenantId"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deploymentKey":"deployment-188","tenantId":"tenant-a","deployments":[{"processDefinition":{"processDefinitionId":"C89_MultipleSubProcessesParent","processDefinitionKey":"188001","processDefinitionVersion":1,"resourceName":"processdefinitions/C89_MultipleSubProcessesParent.bpmn","tenantId":"tenant-a"}}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeRawTestConfig(t, `app:
  camunda_version: "8.9"
  tenant: tenant-a
auth:
  mode: none
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

	root := Root()
	resetCommandTreeFlags(root)
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs([]string{
		"--config", cfgPath,
		"embed", "deploy",
		"--file", "processdefinitions/C89_MultipleSubProcessesParent.bpmn",
		"--no-wait",
	})

	_, err := root.ExecuteC()
	require.NoError(t, err)
	require.True(t, sawDeploy)
	require.Empty(t, stdout.String())
	require.Contains(t, stderr.String(), "creation target: tenant-a")
	require.Less(t, strings.Index(stderr.String(), "creation target: tenant-a"), strings.Index(stderr.String(), "pd deploy done"))
}

func TestEmbedDeployCommand_RegressionPreservesSelectedFixtureDeployOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"embed", "deploy",
		"--file", "processdefinitions/C89_MultipleSubProcessesParent.bpmn",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestEmbedExportCommand_FileSelectionCanStillExportOtherVersionsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"embed", "export",
		"--file", "processdefinitions/C89_SimpleUserTask.bpmn",
		"--out", os.Getenv("C8VOLT_TEST_EXPORT_OUT"),
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

func TestEmbedDeployCommand_AllRunFallsBackToBPMNIDForV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"embed", "deploy",
		"--all",
		"--run",
	})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}

// Helper-process entrypoint for all-tenants embedded deployment rejection.
func TestEmbedDeployCommand_AllTenantsRejectsBeforeEmbeddedOrRequestWorkHelper(t *testing.T) {
	executeRootHelperFromArgsEnv(t, "C8VOLT_TEST_ROOT_ARGS")
}

// Verifies embed export requires an explicit selection via --all or at least one --file.
func TestEmbedExportCommand_RequiresSelection(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")

	output, err := testx.RunCmdSubprocess(t, "TestEmbedExportCommand_RequiresSelectionHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
	require.Contains(t, string(output), "invalid input")
	require.Contains(t, string(output), "either --all or at least one --file is required")
}

func resetEmbedCommandStateForTest() {
	flagEmbedListDetails = false
	flagEmbedDeployFileNames = nil
	flagEmbedDeployAll = false
	flagEmbedDeployWithRun = false
	flagEmbedExportFileNames = nil
	flagEmbedExportOut = "."
	flagEmbedExportAll = false
	flagForce = false
	flagNoWait = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagQuiet = false
	resetDeployCommandContextForTest(Root())
}

// Helper-process entrypoint for embed export selection validation.
func TestEmbedExportCommand_RequiresSelectionHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "embed", "export"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
