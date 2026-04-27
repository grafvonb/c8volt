// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// TestRootHelp_PreservesHumanTaxonomyAndDiscoveryCommand protects the root help text as a UX contract.
// The command groups and Cobra's shell completion command must stay discoverable for humans, while
// internal completion plumbing remains hidden.
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
		"completion",
		"config",
		"embed",
		"version",
		"capabilities",
		"config validation and a cluster check",
		"follow the leaf command examples",
		"c8volt capabilities --json",
		"Camunda 8.7, 8.8, and 8.9",
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
		"config validation and a cluster check",
		"follow the leaf command examples",
		"c8volt capabilities --json",
		"Camunda 8.7, 8.8, and 8.9",
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

func TestTimeoutFlag_RejectsInvalidDuration(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
	})

	require.Error(t, root.PersistentFlags().Set("timeout", "eventually"))
}

// TestRetrieveAndNormalizeConfig_BindsAutomationFlagAndEnvironment verifies that automation mode can be
// configured through environment variables, not only through CLI flags.
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

// TestAutomationModeEnabled_PrefersResolvedConfigContext ensures runtime decisions read the resolved
// config placed on the command context, even when the raw persistent flag value says otherwise.
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

// TestMissingConfigHint_PrefersLocalExampleConfigWhenPresent keeps the bootstrap error helpful for new
// users by pointing at a nearby config.example.yaml when one exists.
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

// TestMissingConfigHint_FallsBackToTemplateAdviceWhenNoLocalExampleExists covers the no-local-example path,
// where the best recovery hint is to generate a template with config show --template.
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

// TestIndicatorEnabled_DefaultsToHumanInteractiveMode verifies the activity indicator remains enabled for
// normal interactive usage unless a mode explicitly disables transient output.
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

// TestIndicatorEnabled_DisabledByQuietAutomationAndNoIndicator checks every non-interactive or quiet path
// that must suppress transient activity output to avoid corrupting machine-readable streams.
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

// TestIndicatorEnabled_DisabledForJSONLogFormat ensures JSON logs are never mixed with terminal activity
// indicators, because both write to the same user-visible stream.
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
