// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestEmbedListHelp_DocumentsReadOnlyDiscoveryExamples(t *testing.T) {
	output := executeRootForTest(t, "embed", "list", "--help")

	require.Contains(t, output, "List bundled BPMN fixture files")
	require.Contains(t, output, "Run this before `embed deploy` or `embed export`")
	require.Contains(t, output, "./c8volt embed list --details")
	require.Contains(t, output, "./c8volt --json embed list")
}

func TestEmbedExportHelp_DocumentsSelectionWorkflow(t *testing.T) {
	output := executeRootForTest(t, "embed", "export", "--help")

	require.Contains(t, output, "Export bundled BPMN fixtures to local files")
	require.Contains(t, output, "Choose `--all` for the full set")
	require.Contains(t, output, "./c8volt embed export --all --out ./fixtures")
	require.Contains(t, output, "quote patterns in the shell like zsh")
}

func TestEmbedDeployCommand_AllRunFallsBackToBPMNIDForV87(t *testing.T) {
	var sawDeploy bool
	var sawRun bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
