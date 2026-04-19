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

	require.Contains(t, output, "get")
	require.Contains(t, output, "run")
	require.Contains(t, output, "expect")
	require.Contains(t, output, "walk")
	require.Contains(t, output, "deploy")
	require.Contains(t, output, "delete")
	require.Contains(t, output, "cancel")
	require.Contains(t, output, "config")
	require.Contains(t, output, "embed")
	require.Contains(t, output, "version")
	require.Contains(t, output, "capabilities")
	require.Contains(t, output, "For machine discovery, use \"c8volt capabilities --json\"")
	require.Contains(t, output, "Use --automation for the dedicated non-interactive execution contract")
	require.Contains(t, output, "outside the explicit automation flag")
	require.Contains(t, output, "--automation")
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
