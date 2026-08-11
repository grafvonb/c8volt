// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"context"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/stretchr/testify/require"
)

// TestIndicatorEnabled_DefaultsToHumanInteractiveMode verifies the activity indicator remains enabled for
// normal interactive usage unless a mode explicitly disables transient output.
func TestIndicatorEnabled_DefaultsToHumanInteractiveMode(t *testing.T) {
	prevNoIndicator := flagNoIndicator
	prevQuiet := flagQuiet
	prevAutomation := flagCmdAutomation
	prevJSON := flagViewAsJson
	prevKeysOnly := flagViewKeysOnly
	t.Cleanup(func() {
		flagNoIndicator = prevNoIndicator
		flagQuiet = prevQuiet
		flagCmdAutomation = prevAutomation
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
	})

	flagNoIndicator = false
	flagQuiet = false
	flagCmdAutomation = false
	flagViewAsJson = false
	flagViewKeysOnly = false

	require.True(t, indicatorEnabled(nil, nil))
}

// TestIndicatorEnabled_DisabledByMachineQuietAutomationAndNoIndicator checks every non-interactive or quiet path
// that must suppress transient activity output to avoid corrupting machine-readable streams.
func TestIndicatorEnabled_DisabledByMachineQuietAutomationAndNoIndicator(t *testing.T) {
	root := Root()
	resetCommandTreeFlags(root)
	prevJSON := flagViewAsJson
	prevKeysOnly := flagViewKeysOnly
	t.Cleanup(func() {
		resetCommandTreeFlags(root)
		flagViewAsJson = prevJSON
		flagViewKeysOnly = prevKeysOnly
	})

	require.NoError(t, root.PersistentFlags().Set("quiet", "true"))
	require.False(t, indicatorEnabled(root, nil))

	resetCommandTreeFlags(root)
	require.NoError(t, root.PersistentFlags().Set("no-indicator", "true"))
	require.False(t, indicatorEnabled(root, nil))

	resetCommandTreeFlags(root)
	flagViewAsJson = true
	require.False(t, indicatorEnabled(root, nil))

	flagViewAsJson = false
	flagViewKeysOnly = true
	require.False(t, indicatorEnabled(root, nil))

	flagViewKeysOnly = false
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
