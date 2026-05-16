// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestOpsPurgeProcessInstancesWithIncidentsHelpDocumentsCommandShape verifies the registered command, alias, and safe examples.
func TestOpsPurgeProcessInstancesWithIncidentsHelpDocumentsCommandShape(t *testing.T) {
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)

	output := executeRootForProcessInstanceTest(t, "ops", "purge", "process-instances-with-incidents", "--help")

	assertHelpOutputContainsAll(t, output,
		"Purge process instances selected by incidents",
		"Aliases:",
		"pi-with-incidents",
		"--key strings",
		"--state string",
		"--error-type string",
		"--error-message string",
		"--bpmn-process-id string",
		"--pd-key string",
		"--pi-key string",
		"--root-key string",
		"--flow-node-id string",
		"--fni-key string",
		"--creation-time-after string",
		"--creation-time-before string",
		"--batch-size int32",
		"--limit int32",
		"--dry-run",
		"--workers int",
		"--no-worker-limit",
		"--fail-fast",
		"--no-wait",
		"--force",
		"--report-file string",
		"--report-format string",
		"./c8volt ops purge process-instances-with-incidents --automation --json --dry-run",
	)
	assertHelpOutputOmitsAll(t, output,
		"incident-pis",
		"./c8volt ops purge process-instances-with-incidents --automation --json\n",
		"--pi-keys-only",
		"--total",
		"--error-message-limit",
		"--with-no-error-message",
	)

	aliasOutput := executeRootForProcessInstanceTest(t, "ops", "purge", "pi-with-incidents", "--help")
	require.Contains(t, aliasOutput, "Purge process instances selected by incidents")
}

// TestOpsPurgeProcessInstancesWithIncidentsRejectsIncidentDisplayOnlyFlags keeps display flags out of the purge surface.
func TestOpsPurgeProcessInstancesWithIncidentsRejectsIncidentDisplayOnlyFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "pi keys only", args: []string{"--pi-keys-only"}, want: "unknown flag: --pi-keys-only"},
		{name: "total", args: []string{"--total"}, want: "unknown flag: --total"},
		{name: "message limit", args: []string{"--error-message-limit", "20"}, want: "unknown flag: --error-message-limit"},
		{name: "omit message", args: []string{"--with-no-error-message"}, want: "unknown flag: --with-no-error-message"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := executeOpsPurgeProcessInstancesWithIncidentsExpectError(t, tt.args...)
			require.Error(t, err)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsUseInvalidInput verifies registered static validation failures.
func TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsUseInvalidInput(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:9", "8.8")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "invalid state",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--state", "open"},
			want: `invalid value for --state: "open"`,
		},
		{
			name: "zero explicit limit",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--limit", "0"},
			want: "--limit must be positive integer",
		},
		{
			name: "batch size too large",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--batch-size", "1001"},
			want: "invalid value for --batch-size: 1001, expected positive integer up to 1000",
		},
		{
			name: "invalid worker count",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--workers", "0"},
			want: "--workers must be positive integer",
		},
		{
			name: "invalid incident key",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--key", "not-a-key"},
			want: `incident key "not-a-key" is not a valid key`,
		},
		{
			name: "invalid report format",
			args: []string{"ops", "purge", "process-instances-with-incidents", "--report-file", "incident-purge.txt", "--report-format", "yaml"},
			want: `unsupported ops workflow report format "yaml"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := testx.RunCmdSubprocess(t, "TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsHelper", map[string]string{
				"C8VOLT_TEST_CONFIG":              cfgPath,
				"C8VOLT_TEST_INCIDENT_PURGE_ARGS": marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t, tt.args),
			})
			require.Error(t, err)

			exitErr, ok := err.(*exec.ExitError)
			require.True(t, ok)
			require.Equal(t, exitcode.InvalidArgs, exitErr.ExitCode())
			require.Contains(t, string(output), "invalid input")
			require.Contains(t, string(output), tt.want)
			require.NotContains(t, string(output), "Usage:")
		})
	}
}

// TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsHelper runs command validation in a subprocess for exit-code assertions.
func TestOpsPurgeProcessInstancesWithIncidentsInvalidFlagsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	var args []string
	if err := json.Unmarshal([]byte(os.Getenv("C8VOLT_TEST_INCIDENT_PURGE_ARGS")), &args); err != nil {
		t.Fatalf("invalid helper args: %v", err)
	}

	root := Root()
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	root.SetArgs(append([]string{"--config", os.Getenv("C8VOLT_TEST_CONFIG")}, args...))
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)
	if err := root.Execute(); err != nil {
		handleBootstrapError(root, err)
	}
}

// executeOpsPurgeProcessInstancesWithIncidentsExpectError runs the command without exiting on Cobra parse errors.
func executeOpsPurgeProcessInstancesWithIncidentsExpectError(t *testing.T, args ...string) (string, error) {
	t.Helper()

	root := Root()
	buf := &bytes.Buffer{}
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	resetOpsPurgeProcessInstancesWithIncidentsFlagState()
	t.Cleanup(resetOpsPurgeProcessInstancesWithIncidentsFlagState)
	root.SetOut(buf)
	root.SetErr(buf)
	fullArgs := append([]string{"ops", "purge", "process-instances-with-incidents"}, args...)
	root.SetArgs(fullArgs)
	_, err := root.ExecuteC()
	if buf.Len() == 0 && err != nil {
		return err.Error(), err
	}
	return buf.String(), err
}

// marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv preserves argument boundaries for subprocess helpers.
func marshalOpsPurgeProcessInstancesWithIncidentsArgsForEnv(t *testing.T, args []string) string {
	t.Helper()

	data, err := json.Marshal(args)
	require.NoError(t, err)
	return string(data)
}

// resetOpsPurgeProcessInstancesWithIncidentsFlagState restores incident-purge globals between command tests.
func resetOpsPurgeProcessInstancesWithIncidentsFlagState() {
	flagOpsPurgeIncidentKeys = nil
	flagOpsPurgeIncidentState = "active"
	flagOpsPurgeIncidentErrorType = ""
	flagOpsPurgeIncidentErrorMessage = ""
	flagOpsPurgeIncidentBpmnProcessID = ""
	flagOpsPurgeIncidentPDKey = ""
	flagOpsPurgeIncidentPIKey = ""
	flagOpsPurgeIncidentRootKey = ""
	flagOpsPurgeIncidentFlowNodeID = ""
	flagOpsPurgeIncidentFNIKey = ""
	flagOpsPurgeIncidentCreationTimeAfter = ""
	flagOpsPurgeIncidentCreationTimeBefore = ""
	flagOpsPurgeIncidentBatchSize = consts.MaxPISearchSize
	flagOpsPurgeIncidentLimit = 0
	flagOpsPurgeIncidentReportFile = ""
	flagOpsPurgeIncidentReportFormat = ""
	flagDryRun = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagNoWait = false
	flagForce = false
	flagCmdAutoConfirm = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagVerbose = false
}
