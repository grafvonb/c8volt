// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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

	require.Contains(t, output, "C88_SimpleUserTaskProcess.bpmn")
	require.Contains(t, output, "C88_MultipleSubProcessesParentProcess.bpmn")
	require.NotContains(t, output, "C87_")
	require.NotContains(t, output, "C89_")
	require.NotContains(t, output, "processdefinitions/")
}

func TestEmbedListCommand_DetailsFiltersFilesForConfiguredCamundaVersion(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.9")

	output := executeRootForTest(t, "--config", cfgPath, "embed", "list", "--details")

	require.Contains(t, output, "processdefinitions/C89_SimpleUserTaskProcess.bpmn")
	require.Contains(t, output, "processdefinitions/C89_MultipleSubProcessesParentProcess.bpmn")
	require.NotContains(t, output, "processdefinitions/C87_")
	require.NotContains(t, output, "processdefinitions/C88_")
}

func TestEmbedExportHelp_DocumentsSelectionWorkflow(t *testing.T) {
	output := executeRootForTest(t, "embed", "export", "--help")

	require.Contains(t, output, "Export bundled BPMN fixtures to local files")
	require.Contains(t, output, "Use --all for the configured Camunda version")
	require.Contains(t, output, "./c8volt embed export --all --out ./fixtures")
	require.Contains(t, output, "quote patterns in the shell like zsh")
}

func TestEmbedExportCommand_AllFiltersFilesForConfiguredCamundaVersion(t *testing.T) {
	resetEmbedCommandStateForTest()
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")
	outDir := t.TempDir()

	output := executeRootForTest(t, "--config", cfgPath, "embed", "export", "--all", "--out", outDir)

	require.Contains(t, output, "exported")
	require.FileExists(t, filepath.Join(outDir, "processdefinitions", "C88_SimpleUserTaskProcess.bpmn"))
	require.FileExists(t, filepath.Join(outDir, "processdefinitions", "C88_MultipleSubProcessesParentProcess.bpmn"))
	require.NoFileExists(t, filepath.Join(outDir, "processdefinitions", "C87_SimpleUserTaskProcess.bpmn"))
	require.NoFileExists(t, filepath.Join(outDir, "processdefinitions", "C89_SimpleUserTaskProcess.bpmn"))
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
	require.FileExists(t, filepath.Join(outDir, "processdefinitions", "C89_SimpleUserTaskProcess.bpmn"))
	require.NoFileExists(t, filepath.Join(outDir, "processdefinitions", "C88_SimpleUserTaskProcess.bpmn"))
}

func TestEmbedDeployCommand_RegressionPreservesSelectedFixtureDeployOnly(t *testing.T) {
	resetEmbedCommandStateForTest()
	var sawDeploy bool

	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/deployments":
			sawDeploy = true
			require.NoError(t, r.ParseMultipartForm(1<<20))
			require.Equal(t, "processdefinitions/C89_MultipleSubProcessesParentProcess.bpmn", r.MultipartForm.File["resources"][0].Filename)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"deploymentKey":"deployment-188","tenantId":"<default>","deployments":[{"processDefinition":{"processDefinitionId":"C89_MultipleSubProcessesParentProcess","processDefinitionKey":"188001","processDefinitionVersion":1,"resourceName":"processdefinitions/C89_MultipleSubProcessesParentProcess.bpmn","tenantId":"<default>"}}]}`))
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

func TestEmbedDeployCommand_RegressionPreservesSelectedFixtureDeployOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{
		"--config", os.Getenv("C8VOLT_TEST_CONFIG"),
		"embed", "deploy",
		"--file", "processdefinitions/C89_MultipleSubProcessesParentProcess.bpmn",
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
		"--file", "processdefinitions/C89_SimpleUserTaskProcess.bpmn",
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
