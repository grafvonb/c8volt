// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape verifies the registered command, alias, and safe examples.
func TestOpsPurgeAllProcessDefinitionsHelpDocumentsCommandShape(t *testing.T) {
	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	output := executeRootForTest(t, "ops", "purge", "all-process-definitions", "--help")

	assertHelpOutputContainsAll(t, output,
		"Purge all selected process definitions",
		"Aliases:",
		"all-pds",
		"--key string",
		"--bpmn-process-id string",
		"--pd-version int32",
		"--pd-version-tag string",
		"--latest",
		"--dry-run",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"--no-wait",
		"--force",
		"--report-file string",
		"--report-format string",
		"./c8volt ops purge all-process-definitions --dry-run",
		"./c8volt ops purge all-pds --bpmn-process-id invoice --latest --dry-run",
		"./c8volt ops purge all-process-definitions --automation --json --dry-run",
	)
	assertHelpOutputOmitsAll(t, output,
		"purge-definitions",
		"delete-all",
		"./c8volt ops purge all-process-definitions --automation --json\n",
		"--xml",
		"--stat",
	)

	aliasOutput := executeRootForTest(t, "ops", "purge", "all-pds", "--help")
	require.Contains(t, aliasOutput, "Purge all selected process definitions")
}

// TestOpsPurgeAllProcessDefinitionsRejectsDisplayOnlyPDFlags keeps get-pd display flags out of the purge surface.
func TestOpsPurgeAllProcessDefinitionsRejectsDisplayOnlyPDFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "xml", args: []string{"--xml"}, want: "unknown flag: --xml"},
		{name: "stat", args: []string{"--stat"}, want: "unknown flag: --stat"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeAllProcessDefinitionsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestOpsPurgeAllProcessDefinitionsInvalidFlagsUseInvalidInput verifies local flag validation before remote work.
func TestOpsPurgeAllProcessDefinitionsInvalidFlagsUseInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid key",
			args: []string{"--key", "not-a-key"},
			want: `process definition key "not-a-key" is not a valid key`,
		},
		{
			name: "zero explicit process definition version",
			args: []string{"--pd-version", "0"},
			want: "--pd-version must be positive integer",
		},
		{
			name: "negative process definition version",
			args: []string{"--pd-version", "-1"},
			want: "--pd-version must be positive integer",
		},
		{
			name: "invalid worker count",
			args: []string{"--workers", "0"},
			want: "--workers must be positive integer",
		},
		{
			name: "report format without file",
			args: []string{"--report-format", "json"},
			want: "--report-format requires --report-file",
		},
		{
			name: "unsupported report format",
			args: []string{"--report-file", "purge.txt", "--report-format", "yaml"},
			want: `unsupported ops workflow report format "yaml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeAllProcessDefinitionsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// executeOpsPurgeAllProcessDefinitionsExpectError runs all-process-definitions purge and returns Cobra parse/validation errors.
func executeOpsPurgeAllProcessDefinitionsExpectError(t *testing.T, args ...string) (string, error) {
	t.Helper()

	resetOpsPurgeAllProcessDefinitionsFlagState()
	t.Cleanup(resetOpsPurgeAllProcessDefinitionsFlagState)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(append([]string{"ops", "purge", "all-process-definitions"}, args...))
	resetCommandTreeFlags(root)
	resetOpsPurgeAllProcessDefinitionsFlagState()

	_, err := root.ExecuteC()
	if err != nil {
		return buf.String() + err.Error(), err
	}
	return buf.String(), nil
}

// resetOpsPurgeAllProcessDefinitionsFlagState restores all-process-definitions purge globals between command tests.
func resetOpsPurgeAllProcessDefinitionsFlagState() {
	flagOpsPurgeAllPDKey = ""
	flagOpsPurgeAllPDBpmnProcessID = ""
	flagOpsPurgeAllPDProcessVersion = 0
	flagOpsPurgeAllPDProcessVersionTag = ""
	flagOpsPurgeAllPDLatest = false
	flagOpsPurgeAllPDReportFile = ""
	flagOpsPurgeAllPDReportFormat = ""
	flagDryRun = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagNoWait = false
	flagForce = false
	flagCmdAutoConfirm = false
	flagViewAsJson = false
}
