// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestOpsWorkflowStepStatusesMatchSharedContract protects the status vocabulary promised to future ops reports.
func TestOpsWorkflowStepStatusesMatchSharedContract(t *testing.T) {
	statuses := opsWorkflowStepStatuses()

	require.Equal(t, []OpsWorkflowStepStatus{
		OpsWorkflowStepStatusPlanned,
		OpsWorkflowStepStatusSkipped,
		OpsWorkflowStepStatusSubmitted,
		OpsWorkflowStepStatusConfirmed,
		OpsWorkflowStepStatusConfirmationFailed,
		OpsWorkflowStepStatusBlocked,
		OpsWorkflowStepStatusFailed,
	}, statuses)
	require.Equal(t, []string{
		"planned",
		"skipped",
		"submitted",
		"confirmed",
		"confirmation_failed",
		"blocked",
		"failed",
	}, opsWorkflowStatusStrings(statuses))

	for _, status := range statuses {
		require.True(t, status.IsValid(), "expected %q to be a valid ops workflow status", status)
		require.Equal(t, string(status), status.String())
	}
	require.False(t, OpsWorkflowStepStatus("mutation_failed").IsValid())
}

func TestOpsWorkflowElapsedSuffixUsesApproximateDuration(t *testing.T) {
	require.Empty(t, opsWorkflowElapsedSuffix(""))
	require.Equal(t, "; elapsed: <1s", opsWorkflowElapsedSuffix((250 * time.Millisecond).String()))
	require.Equal(t, "; elapsed: 1m31s", opsWorkflowElapsedSuffix((90*time.Second + 600*time.Millisecond).String()))
	require.Equal(t, "; elapsed: about five minutes", opsWorkflowElapsedSuffix("about five minutes"))
}

// TestOpsWorkflowReportFormatForPath documents explicit and extension-inferred report format behavior.
func TestOpsWorkflowReportFormatForPath(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		requested OpsWorkflowReportFormat
		want      OpsWorkflowReportFormat
		wantErr   string
	}{
		{
			name: "explicit json wins over extension",
			path: "run.md", requested: OpsWorkflowReportFormatJSON,
			want: OpsWorkflowReportFormatJSON,
		},
		{
			name: "json extension",
			path: "run.json",
			want: OpsWorkflowReportFormatJSON,
		},
		{
			name: "markdown extension",
			path: "run.MARKDOWN",
			want: OpsWorkflowReportFormatMarkdown,
		},
		{
			name: "empty path defaults to markdown",
			want: OpsWorkflowReportFormatMarkdown,
		},
		{
			name: "unsupported requested format",
			path: "run.json", requested: OpsWorkflowReportFormat("yaml"),
			wantErr: `unsupported ops workflow report format "yaml"`,
		},
		{
			name: "unknown extension defaults to markdown",
			path: "run.txt",
			want: OpsWorkflowReportFormatMarkdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := opsWorkflowReportFormatForPath(tt.path, tt.requested)

			if tt.wantErr != "" {
				require.EqualError(t, err, tt.wantErr)
				require.Empty(t, got)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.True(t, got.IsValid())
			require.Equal(t, string(got), got.String())
		})
	}
}

func TestValidateOpsWorkflowReportFlags(t *testing.T) {
	require.NoError(t, validateOpsWorkflowReportFlags("", ""))
	require.NoError(t, validateOpsWorkflowReportFlags("run.md", ""))
	require.NoError(t, validateOpsWorkflowReportFlags("run.md", OpsWorkflowReportFormatJSON))

	require.EqualError(t,
		validateOpsWorkflowReportFlags("", OpsWorkflowReportFormatJSON),
		"invalid input: missing dependent flags: --report-format requires --report-file",
	)
	require.EqualError(t,
		validateOpsWorkflowReportFlags("run.json", OpsWorkflowReportFormat("yaml")),
		`invalid input: invalid flag value: unsupported ops workflow report format "yaml"`,
	)
}

func TestOpsExecuteRetentionPolicyReportFlagsReuseSharedContract(t *testing.T) {
	prevFile := flagOpsExecuteRetentionPolicyReportFile
	prevFormat := flagOpsExecuteRetentionPolicyReportFormat
	t.Cleanup(func() {
		flagOpsExecuteRetentionPolicyReportFile = prevFile
		flagOpsExecuteRetentionPolicyReportFormat = prevFormat
	})

	flagOpsExecuteRetentionPolicyReportFile = "retention-report.json"
	flagOpsExecuteRetentionPolicyReportFormat = "json"
	require.NoError(t, validateOpsExecuteRetentionPolicyReportFlags())

	flagOpsExecuteRetentionPolicyReportFile = ""
	flagOpsExecuteRetentionPolicyReportFormat = "json"
	require.EqualError(t,
		validateOpsExecuteRetentionPolicyReportFlags(),
		"invalid input: missing dependent flags: --report-format requires --report-file",
	)

	flagOpsExecuteRetentionPolicyReportFile = "retention-report.yaml"
	flagOpsExecuteRetentionPolicyReportFormat = "yaml"
	require.EqualError(t,
		validateOpsExecuteRetentionPolicyReportFlags(),
		`invalid input: invalid flag value: unsupported ops workflow report format "yaml"`,
	)
}

func TestOpsExecuteSmokeTestReportFlagsReuseSharedContract(t *testing.T) {
	prevFile := flagOpsExecuteSmokeTestReportFile
	prevFormat := flagOpsExecuteSmokeTestReportFormat
	t.Cleanup(func() {
		flagOpsExecuteSmokeTestReportFile = prevFile
		flagOpsExecuteSmokeTestReportFormat = prevFormat
	})

	flagOpsExecuteSmokeTestReportFile = "smoke-test.json"
	flagOpsExecuteSmokeTestReportFormat = ""
	require.NoError(t, validateOpsExecuteSmokeTestReportFlags())
	format, err := opsWorkflowReportFormatForPath(flagOpsExecuteSmokeTestReportFile, OpsWorkflowReportFormat(flagOpsExecuteSmokeTestReportFormat))
	require.NoError(t, err)
	require.Equal(t, OpsWorkflowReportFormatJSON, format)

	flagOpsExecuteSmokeTestReportFile = "smoke-test.md"
	flagOpsExecuteSmokeTestReportFormat = "json"
	require.NoError(t, validateOpsExecuteSmokeTestReportFlags())

	flagOpsExecuteSmokeTestReportFile = ""
	flagOpsExecuteSmokeTestReportFormat = "json"
	require.EqualError(t,
		validateOpsExecuteSmokeTestReportFlags(),
		"invalid input: missing dependent flags: --report-format requires --report-file",
	)

	flagOpsExecuteSmokeTestReportFile = "smoke-test.yaml"
	flagOpsExecuteSmokeTestReportFormat = "yaml"
	require.EqualError(t,
		validateOpsExecuteSmokeTestReportFlags(),
		`invalid input: invalid flag value: unsupported ops workflow report format "yaml"`,
	)
}

func TestWriteOpsWorkflowReportFilePreservesExistingUntilConfirmed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.md")
	require.NoError(t, writeOpsWorkflowReportFile(path, []byte("first"), OpsWorkflowReportPreserveExisting))
	require.Equal(t, "first", readOpsContractTestFile(t, path))

	err := writeOpsWorkflowReportFile(path, []byte("second"), OpsWorkflowReportPreserveExisting)
	require.EqualError(t, err, "report file already exists: "+path)
	require.Equal(t, "first", readOpsContractTestFile(t, path))

	require.NoError(t, writeOpsWorkflowReportFile(path, []byte("second"), OpsWorkflowReportOverwriteExisting))
	require.Equal(t, "second", readOpsContractTestFile(t, path))
}

func TestOpsWorkflowReportWriteModeForConfirmedMutation(t *testing.T) {
	require.Equal(t, OpsWorkflowReportPreserveExisting, opsWorkflowReportWriteModeForConfirmedMutation(false))
	require.Equal(t, OpsWorkflowReportOverwriteExisting, opsWorkflowReportWriteModeForConfirmedMutation(true))
}

func TestValidateOpsWorkflowReportPathForPlanning(t *testing.T) {
	dir := t.TempDir()
	existing := filepath.Join(dir, "report.md")
	missing := filepath.Join(dir, "new-report.md")
	require.NoError(t, os.WriteFile(existing, []byte("existing"), 0o600))

	require.NoError(t, validateOpsWorkflowReportPathForPlanning("", OpsWorkflowReportPreserveExisting))
	require.NoError(t, validateOpsWorkflowReportPathForPlanning(missing, OpsWorkflowReportPreserveExisting))
	require.NoError(t, validateOpsWorkflowReportPathForPlanning(existing, OpsWorkflowReportOverwriteExisting))

	err := validateOpsWorkflowReportPathForPlanning(existing, OpsWorkflowReportPreserveExisting)
	require.EqualError(t, err, "local precondition failed: report file already exists: "+existing)
	require.Equal(t, "existing", readOpsContractTestFile(t, existing))
}

// opsWorkflowStatusStrings keeps test assertions focused on the stable serialized tokens.
func opsWorkflowStatusStrings(statuses []OpsWorkflowStepStatus) []string {
	out := make([]string, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, status.String())
	}
	return out
}

func readOpsContractTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
