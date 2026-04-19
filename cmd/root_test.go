package cmd

import (
	"context"
	"testing"

	"github.com/grafvonb/c8volt/config"
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
	)
	assertHelpOutputOmitsAll(t, output,
		"\ncompletion\n",
		"__complete",
		"__completeNoDesc",
	)
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
