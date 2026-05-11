// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

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

// opsWorkflowStatusStrings keeps test assertions focused on the stable serialized tokens.
func opsWorkflowStatusStrings(statuses []OpsWorkflowStepStatus) []string {
	out := make([]string, 0, len(statuses))
	for _, status := range statuses {
		out = append(out, status.String())
	}
	return out
}
