// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"strconv"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/stretchr/testify/require"
)

// TestRootHelp_PreservesHumanTaxonomyAndDiscoveryCommand protects the root help text as a UX contract.
// The command groups and Cobra's shell completion command must stay discoverable for users, while
// internal completion plumbing remains hidden.
func TestRootHelp_PreservesHumanTaxonomyAndDiscoveryCommand(t *testing.T) {
	output := executeRootForTest(t, "--help")

	assertHelpOutputContainsAll(t, output,
		"get",
		"run",
		"expect",
		"walk",
		"ops",
		"Discover high-level operational workflows",
		"deploy",
		"delete",
		"cancel",
		"completion",
		"config",
		"embed",
		"version",
		"capabilities",
		"Use capabilities for the machine-readable",
		"Camunda 8.7, 8.8, 8.9, and 8.10",
		"Camunda 8.10 baseline: 8.10.0-alpha4 (prerelease)",
		"--automation",
		"Examples:",
		"./c8volt config show --template",
		"./c8volt get cluster topology",
		"./c8volt capabilities --json",
		"./c8volt --config ./config.yaml config show --validate",
	)
	assertHelpOutputOmitsAll(t, output,
		"__complete",
		"__completeNoDesc",
		"cluster-topology",
	)
}

// TestRootHelpAndGeneratedMarkdownShareDiscoveryAnchors keeps CLI help and generated docs aligned on
// automation/discovery guidance, so users do not see different onboarding advice in different surfaces.
func TestRootHelpAndGeneratedMarkdownShareDiscoveryAnchors(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	helpOutput := executeRootForTest(t, "--help")
	markdown := renderMarkdownForCommand(t, root)

	for _, anchor := range []string{
		"Use capabilities for the machine-readable",
		"Camunda 8.7, 8.8, 8.9, and 8.10",
		"Camunda 8.10 baseline: 8.10.0-alpha4 (prerelease)",
	} {
		require.Contains(t, helpOutput, anchor)
		require.Contains(t, markdown, anchor)
	}
}

// TestProcessInstanceHelp_PreservesLocalBeforeGlobalFlagUX guards an intentional Cobra UX detail:
// command-local flags, including locally repeated/derived flags such as --pd-key, must appear before
// inherited global flags like --config and --json.
func TestProcessInstanceHelp_PreservesLocalBeforeGlobalFlagUX(t *testing.T) {
	output := executeRootForTest(t, "get", "process-instance", "--help")

	flags := strings.Index(output, "\nFlags:\n")
	globalFlags := strings.Index(output, "\nGlobal Flags:\n")
	require.NotEqual(t, -1, flags)
	require.NotEqual(t, -1, globalFlags)
	require.Less(t, flags, globalFlags)

	localFlag := strings.Index(output[flags:globalFlags], "--bpmn-process-id")
	derivedLocalFlag := strings.Index(output[flags:globalFlags], "--pd-key")
	globalConfigFlag := strings.Index(output[globalFlags:], "--config")
	globalJSONFlag := strings.Index(output[globalFlags:], "--json")
	require.NotEqual(t, -1, localFlag)
	require.NotEqual(t, -1, derivedLocalFlag)
	require.NotEqual(t, -1, globalConfigFlag)
	require.NotEqual(t, -1, globalJSONFlag)
}

// TestProcessInstanceHelp_ExposesCompactGlobalFlags keeps the visible inherited flag surface focused
// on common command-line controls while advanced tuning remains available through config or hidden
// compatibility flags.
func TestProcessInstanceHelp_ExposesCompactGlobalFlags(t *testing.T) {
	output := executeRootForTest(t, "get", "process-instance", "--help")

	assertHelpOutputContainsAll(t, output,
		"--all-tenants",
		"--auto-confirm",
		"--automation",
		"--config",
		"--debug",
		"--json",
		"--keys-only",
		"--log-level",
		"--no-indicator",
		"--profile",
		"--quiet",
		"--tenant",
		"--timeout",
		"--verbose",
	)
	assertHelpOutputOmitsAll(t, output,
		"--backoff-max-retries",
		"--backoff-timeout",
		"--log-format",
		"--log-with-source",
		"--no-err-codes",
	)
}

// TestAllTenantsRootFlag_DefaultsFalse verifies the inherited override starts inactive
// until explicitly selected on the command line.
func TestAllTenantsRootFlag_DefaultsFalse(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	flag := root.PersistentFlags().Lookup("all-tenants")
	require.NotNil(t, flag)
	require.Equal(t, "bool", flag.Value.Type())
	require.Equal(t, "false", flag.DefValue)
	require.Equal(t, "false", flag.Value.String())
	require.False(t, flag.Changed)
	require.False(t, flagAllTenants)
}

// TestAllTenantsRootFlag_ParsesRootAndSubcommandPlacement proves Cobra accepts
// the boolean override wherever inherited root flags are accepted.
func TestAllTenantsRootFlag_ParsesRootAndSubcommandPlacement(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{
			name: "before subcommand",
			args: []string{"--all-tenants", "get", "process-instance", "--help"},
			want: true,
		},
		{
			name: "after subcommand",
			args: []string{"get", "process-instance", "--all-tenants", "--help"},
			want: true,
		},
		{
			name: "explicit true",
			args: []string{"get", "process-instance", "--all-tenants=true", "--help"},
			want: true,
		},
		{
			name: "explicit false stays inactive",
			args: []string{"get", "process-instance", "--all-tenants=false", "--help"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := Root()
			resetCommandTreeFlags(root)
			t.Cleanup(func() {
				resetCommandTreeFlags(root)
			})

			buf := &bytes.Buffer{}
			root.SetOut(buf)
			root.SetErr(buf)
			root.SetArgs(tt.args)

			_, err := root.ExecuteC()
			require.NoError(t, err)

			flag := root.PersistentFlags().Lookup("all-tenants")
			require.NotNil(t, flag)
			require.True(t, flag.Changed)
			require.Equal(t, tt.want, flagAllTenants)
			require.Equal(t, strconv.FormatBool(tt.want), flag.Value.String())
		})
	}
}

// TestAllTenantsRootFlag_RejectsExplicitTenantChoiceConflicts proves the two
// command-line tenant selectors are mutually exclusive, even when --tenant is
// explicitly empty.
func TestAllTenantsRootFlag_RejectsExplicitTenantChoiceConflicts(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "named tenant before subcommand",
			args: []string{"--all-tenants", "--tenant", "tenant-a", "config", "show", "--template"},
		},
		{
			name: "empty tenant before subcommand",
			args: []string{"--all-tenants", "--tenant", "", "config", "show", "--template"},
		},
		{
			name: "named tenant after subcommand",
			args: []string{"config", "show", "--template", "--all-tenants", "--tenant", "tenant-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeRootExpectErrorForTest(t, tt.args...)

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid input")
			require.Contains(t, err.Error(), "--tenant cannot be combined with --all-tenants")
			require.NotContains(t, output, "Usage:")
			require.NotContains(t, output, "Examples:")
		})
	}
}

// TestAllTenantsRootFlag_AllowsInactiveOrConfiguredTenantSources keeps existing
// tenant behavior valid when all-tenants is false, absent, or only overriding a
// non-command-line configured source.
func TestAllTenantsRootFlag_AllowsInactiveOrConfiguredTenantSources(t *testing.T) {
	cfgPath := writeRawTestConfig(t, `
app:
  tenant: configured-tenant
auth:
  mode: none
apis:
  camunda_api:
    base_url: http://camunda.example.test
`)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "explicit false allows explicit tenant",
			args: []string{"--all-tenants=false", "--tenant", "tenant-a", "config", "show", "--template"},
		},
		{
			name: "absent all-tenants allows explicit tenant",
			args: []string{"--tenant", "tenant-a", "config", "show", "--template"},
		},
		{
			name: "configured source allows active all-tenants",
			args: []string{"--config", cfgPath, "--all-tenants", "config", "show"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := executeRootForTest(t, tt.args...)

			require.NotContains(t, output, "--tenant cannot be combined with --all-tenants")
		})
	}
}

// TestAllTenantsRootFlag_ResetClearsConflictStateBetweenExecutions verifies
// in-process executions do not retain a previous active all-tenants selection.
func TestAllTenantsRootFlag_ResetClearsConflictStateBetweenExecutions(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs([]string{"--all-tenants", "--tenant", "tenant-a", "config", "show", "--template"})

	_, err := root.ExecuteC()
	require.Error(t, err)
	require.Contains(t, err.Error(), "--tenant cannot be combined with --all-tenants")

	resetCommandTreeFlags(root)
	buf.Reset()
	root.SetArgs([]string{"--tenant", "tenant-a", "config", "show", "--template"})

	_, err = root.ExecuteC()
	require.NoError(t, err)
	require.False(t, flagAllTenants)
}

func TestTimeoutFlag_RejectsInvalidDuration(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	require.Error(t, root.PersistentFlags().Set("timeout", "eventually"))
}

func TestFlagParseErrorsDoNotPrintUsage(t *testing.T) {
	tests := [][]string{
		{"--config"},
		{"get", "tenant", "--key"},
		{"run", "pi", "--vars"},
		{"walk", "pi", "--key"},
		{"delete", "pd", "--key"},
		{"update", "pi", "--vars-file"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			output, err := executeRootExpectErrorForTest(t, args...)

			require.Error(t, err)
			require.Contains(t, err.Error(), "invalid input")
			require.Contains(t, err.Error(), "flag needs an argument")
			require.NotContains(t, output, "Usage:")
			require.NotContains(t, output, "Examples:")
			require.NotContains(t, output, "Global Flags:")
		})
	}
}

func assertCommandHelpOutput(t *testing.T, args []string, contains []string, omits []string) string {
	t.Helper()

	output := executeRootForTest(t, append(args, "--help")...)
	assertHelpOutputContainsAll(t, output, contains...)
	assertHelpOutputOmitsAll(t, output, omits...)
	return output
}

func assertHelpOutputContainsAll(t *testing.T, output string, substrings ...string) {
	t.Helper()

	for _, substring := range substrings {
		require.Contains(t, output, substring)
	}
}

func assertHelpOutputOmitsAll(t *testing.T, output string, substrings ...string) {
	t.Helper()

	for _, substring := range substrings {
		require.NotContains(t, output, substring)
	}
}

func renderMarkdownForCommand(t *testing.T, command *cobra.Command) string {
	t.Helper()

	var buf bytes.Buffer
	require.NoError(t, doc.GenMarkdown(command, &buf))
	return buf.String()
}
