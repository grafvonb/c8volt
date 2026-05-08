// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpdateJobCommand_RejectsJSONVerboseBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagVerbose = true
	flagDryRun = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json cannot be combined with --verbose for update job")
}

func TestUpdateJobCommand_RejectsJSONMutationWithoutAutoConfirmOrAutomationBeforeLookupOrMutation(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "--json update job requires --dry-run, --auto-confirm, or --automation")
}

func TestUpdateJobCommand_AllowsJSONDryRunWithoutAutoConfirm(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)

	flagViewAsJson = true
	flagDryRun = true

	err := validateUpdateJobJSONGuardrails(updateJobCmd)

	require.NoError(t, err)
}

func TestParseUpdateJobRequestRequiresUpdateFlag(t *testing.T) {
	resetUpdateJobFlagState()
	t.Cleanup(resetUpdateJobFlagState)
	resetCommandTreeFlags(Root())

	flagUpdateJobKey = "2251799813711967"

	_, err := parseUpdateJobRequest(updateJobCmd)

	require.Error(t, err)
	require.Contains(t, err.Error(), "update job requires --retries, --timeout, or both")
}

func resetUpdateJobFlagState() {
	flagViewAsJson = false
	flagVerbose = false
	flagDryRun = false
	flagNoWait = false
	flagCmdAutoConfirm = false
	flagCmdAutomation = false
	flagUpdateJobKey = ""
	flagUpdateJobRetries = 0
	flagUpdateJobTimeoutRaw = ""
}
