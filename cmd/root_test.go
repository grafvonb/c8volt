package cmd

import (
	"bytes"
	"context"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestRootHelp_PreservesHumanTaxonomyAndDiscoveryCommand(t *testing.T) {
	output := executeRootForTest(t, "--help")

	assertHelpOutputContainsAll(t, output,
		"get",
		"run",
		"expect",
		"walk",
		"deploy",
		"delete",
		"cancel",
		"config",
		"embed",
		"version",
		"capabilities",
		"c8volt <group> --help",
		"c8volt capabilities --json",
		"flag metadata, output modes, mutation behavior, and automation support",
		"Prefer --json where a command exposes structured output",
		"automation:full",
		"--automation",
		"Examples:",
		"./c8volt get --help",
		"./c8volt capabilities --json",
		"./c8volt --config ./config.yaml config show --validate",
	)
	assertHelpOutputOmitsAll(t, output,
		"\ncompletion\n",
		"__complete",
		"__completeNoDesc",
	)
}

func TestRootHelpAndGeneratedMarkdownShareDiscoveryAnchors(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	helpOutput := executeRootForTest(t, "--help")
	markdown := renderMarkdownForCommand(t, root)

	for _, anchor := range []string{
		"c8volt <group> --help",
		"c8volt capabilities --json",
		"flag metadata, output modes, mutation behavior, and automation support",
		"Prefer --json where a command exposes structured output",
		"automation:full",
	} {
		require.Contains(t, helpOutput, anchor)
		require.Contains(t, markdown, anchor)
	}
}

func TestRetrieveAndNormalizeConfig_BindsAutomationFlagAndEnvironment(t *testing.T) {
	t.Setenv("C8VOLT_APP_AUTOMATION", "true")

	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	v := viper.New()
	bindings, err := initViper(v, root)
	require.NoError(t, err)

	cfg, err := retrieveAndNormalizeConfig(v, bindings)
	require.NoError(t, err)
	require.True(t, cfg.App.Automation)
}

func TestAutomationModeEnabled_PrefersResolvedConfigContext(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})
	require.NoError(t, root.PersistentFlags().Set("automation", "false"))

	cfg := config.New()
	cfg.App.Automation = true
	root.SetContext(cfg.ToContext(context.Background()))

	require.True(t, automationModeEnabled(root))
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
