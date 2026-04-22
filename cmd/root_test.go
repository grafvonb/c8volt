package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
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

func TestMissingConfigHint_PrefersLocalExampleConfigWhenPresent(t *testing.T) {
	prevWD, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config.example.yaml"), []byte("apis: {}\n"), 0o600))
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	got := missingConfigHint()
	require.Contains(t, got, `found "config.example.yaml" in the current directory`)
	require.Contains(t, got, "config show --validate")
}

func TestMissingConfigHint_FallsBackToTemplateAdviceWhenNoLocalExampleExists(t *testing.T) {
	prevWD, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() {
		_ = os.Chdir(prevWD)
	})

	got := missingConfigHint()
	require.Contains(t, got, "config show --template")
	require.NotContains(t, got, "config.example.yaml")
}

func TestIndicatorEnabled_DefaultsToHumanInteractiveMode(t *testing.T) {
	prevNoIndicator := flagNoIndicator
	prevQuiet := flagQuiet
	prevAutomation := flagCmdAutomation
	t.Cleanup(func() {
		flagNoIndicator = prevNoIndicator
		flagQuiet = prevQuiet
		flagCmdAutomation = prevAutomation
	})

	flagNoIndicator = false
	flagQuiet = false
	flagCmdAutomation = false

	require.True(t, indicatorEnabled(nil, nil))
}

func TestIndicatorEnabled_DisabledByQuietAutomationAndNoIndicator(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	require.NoError(t, root.PersistentFlags().Set("quiet", "true"))
	require.False(t, indicatorEnabled(root, nil))

	resetCommandTreeFlags(root)
	require.NoError(t, root.PersistentFlags().Set("no-indicator", "true"))
	require.False(t, indicatorEnabled(root, nil))

	resetCommandTreeFlags(root)
	cfg := config.New()
	cfg.App.Automation = true
	root.SetContext(cfg.ToContext(context.Background()))
	require.False(t, indicatorEnabled(root, cfg))
}

func TestIndicatorEnabled_DisabledForJSONLogFormat(t *testing.T) {
	cfg := config.New()
	cfg.Log.Format = "json"

	require.False(t, indicatorEnabled(nil, cfg))
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
