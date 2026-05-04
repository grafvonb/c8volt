// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// Protects bootstrap bypass from matching only by leaf command name; nested commands like
// `get cluster version` still need the full service bootstrap.
func TestBypassRootBootstrap_TreatsCompletionCommandsAsSharedUtilitySeam(t *testing.T) {
	require.True(t, bypassRootBootstrap(&cobra.Command{Use: "__complete"}))
	require.True(t, bypassRootBootstrap(&cobra.Command{Use: "__completeNoDesc"}))
	require.True(t, bypassRootBootstrap(&cobra.Command{Use: "completion"}))
	require.True(t, bypassRootBootstrap(versionCmd))
	require.True(t, bypassRootBootstrap(configShowCmd))
	require.True(t, bypassRootBootstrap(configTemplateCmd))
	require.True(t, bypassRootBootstrap(configValidateCmd))
	require.True(t, bypassRootBootstrap(configTestConnectionCmd))
	require.False(t, bypassRootBootstrap(getCmd))
	require.False(t, bypassRootBootstrap(getClusterVersionCmd))
}

// Verifies completion requests work without config bootstrap and stay focused on candidates.
func TestCompletionCommandsBypassBootstrapWithoutConfig(t *testing.T) {
	output := executeCompletionForTest(t, "walk", "process-instance", "")

	require.Contains(t, output, "--key")
	require.NotContains(t, output, "configuration is invalid")
	require.NotContains(t, output, "Usage:")
}

// Verifies top-level completion includes user-facing command descriptions.
func TestRootCompletion_TopLevelSuggestionsStayReadable(t *testing.T) {
	output := executeCompletionForTest(t, "")

	require.Contains(t, output, "get\tInspect cluster, process, tenant, and resource state\n")
	require.Contains(t, output, "embed\tUse bundled BPMN fixtures\n")
	require.Contains(t, output, "walk\tInspect process-instance relationships\n")
	require.Contains(t, output, "run\tStart process instances\n")
	require.Contains(t, output, "deploy\tDeploy BPMN resources to Camunda\n")
	require.Contains(t, output, "completion\tGenerate the autocompletion script for the specified shell\n")
	requireCompletionOutputStaysUserFacing(t, output)
}

// Verifies partial top-level completion keeps readable user-facing suggestions.
func TestRootCompletion_PartialTopLevelSuggestionsStayReadable(t *testing.T) {
	output := executeCompletionForTest(t, "g")

	require.Contains(t, output, "get\tInspect cluster, process, tenant, and resource state\n")
	requireCompletionOutputStaysUserFacing(t, output)
}

// Verifies nested completion surfaces user-facing get subcommands.
func TestNestedCompletion_SubcommandsStayUserFacing(t *testing.T) {
	output := executeCompletionForTest(t, "get", "")

	require.Contains(t, output, "process-definition\tList or fetch deployed process definitions\n")
	require.Contains(t, output, "process-instance\tList or fetch process instances\n")
	require.Contains(t, output, "tenant\tList tenants\n")
	requireCompletionOutputStaysUserFacing(t, output)
}

// Verifies value completion output stays clean when descriptions are intentionally absent.
func TestCompletionSuggestionsWithoutDescriptionsStayClean(t *testing.T) {
	output := executeCompletionForTest(t, "walk", "process-instance", "")

	require.Contains(t, output, "--key")
	require.NotContains(t, output, "--mode")
	requireCompletionOutputStaysUserFacing(t, output)
}

func requireCompletionOutputStaysUserFacing(t *testing.T, output string) {
	t.Helper()

	require.NotContains(t, output, "__complete")
	require.NotContains(t, output, "Usage:")
	require.NotContains(t, output, "Get resources such as process definitions or process instances.")
	require.NotContains(t, output, "It is a root command and requires a subcommand to specify the resource type to get.")
}

// Verifies `completion zsh` emits description-bearing completion requests.
func TestCompletionCommand_ZshUsesDescriptionBearingCompletionRequests(t *testing.T) {
	cfgPath := writeTestConfig(t, "http://127.0.0.1:1")
	output, err := testx.RunCmdSubprocess(t, "TestCompletionCommand_ZshUsesDescriptionBearingCompletionRequestsHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})
	require.NoError(t, err)

	require.Contains(t, string(output), "__complete")
	require.NotContains(t, string(output), "__completeNoDesc")
}

// Helper-process entrypoint for zsh completion command request-shape validation.
func TestCompletionCommand_ZshUsesDescriptionBearingCompletionRequestsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	root := Root()
	root.SetArgs([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG"), "completion", "zsh"})
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	_ = root.Execute()
}
