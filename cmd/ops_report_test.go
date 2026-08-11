// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/ops"
	"github.com/stretchr/testify/require"
)

// TestFormatOpsPreflightScopeRendersConsequencesAndConfirmationContext verifies broad selector summaries stay compact and certainty-aware.
func TestFormatOpsPreflightScopeRendersConsequencesAndConfirmationContext(t *testing.T) {
	total := int64(10000)
	pages := int64(10)

	got := formatOpsPreflightScope(ops.PreflightScope{
		Command:         "ops analyse slow-process-instances",
		SelectorSummary: "OrderProcess",
		CoreResource:    "process_instance",
		Total:           &total,
		TotalKind:       ops.TotalCertaintyLowerBound,
		PageSize:        1000,
		PageCount:       &pages,
		PageCountKind:   ops.PageCountKindEstimated,
		ConsequenceSummary: ops.ConsequenceSummary{
			WorkSummary: "discover all matches and load runtime element timelines",
			RiskSummary: "read-only, expensive",
		},
		RequiresConfirmation: true,
	})

	require.Equal(t, []string{
		"slow analysis scope: OrderProcess matched at least 10000 process instances; page size: 1000; discovery pages: at least 10",
		"slow analysis is expensive: discover all matches and load runtime element timelines",
	}, got)
}

// TestFormatOpsPreflightScopeLabelsExactLowerBoundAndUnknownTotals verifies count wording covers all certainty cases.
func TestFormatOpsPreflightScopeLabelsExactLowerBoundAndUnknownTotals(t *testing.T) {
	tests := []struct {
		name      string
		total     *int64
		kind      ops.TotalCertainty
		pageCount *int64
		pageKind  ops.PageCountKind
		want      string
	}{
		{name: "exact", total: ptrInt64(2000), kind: ops.TotalCertaintyExact, pageCount: ptrInt64(2), pageKind: ops.PageCountKindExact, want: "slow analysis scope: OrderProcess matched 2000 process instances; page size: 1000; discovery pages: 2"},
		{name: "zero exact", total: ptrInt64(0), kind: ops.TotalCertaintyExact, pageKind: ops.PageCountKindUnknown, want: "slow analysis scope: OrderProcess matched no process instances; page size: 1000"},
		{name: "lower bound", total: ptrInt64(2000), kind: ops.TotalCertaintyLowerBound, pageCount: ptrInt64(2), pageKind: ops.PageCountKindEstimated, want: "slow analysis scope: OrderProcess matched at least 2000 process instances; page size: 1000; discovery pages: at least 2"},
		{name: "unknown", kind: ops.TotalCertaintyUnknown, pageKind: ops.PageCountKindUnknown, want: "slow analysis scope: OrderProcess matched an unknown number of process instances; page size: 1000"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatOpsPreflightScope(ops.PreflightScope{
				Command:         "ops analyse slow-process-instances",
				SelectorSummary: "OrderProcess",
				CoreResource:    "process_instance",
				Total:           tc.total,
				TotalKind:       tc.kind,
				PageSize:        1000,
				PageCount:       tc.pageCount,
				PageCountKind:   tc.pageKind,
			})

			require.Equal(t, tc.want, got[0])
		})
	}
}

// TestFormatOpsPreflightScopeRendersEmptyScopeWithoutBlankLines verifies zero-match output stays calm and grammatical.
func TestFormatOpsPreflightScopeRendersEmptyScopeWithoutBlankLines(t *testing.T) {
	total := int64(0)

	got := formatOpsPreflightScope(ops.PreflightScope{
		Command:         "ops analyse slow-process-instances",
		SelectorSummary: "EmptyProcess",
		CoreResource:    "process_instance",
		Total:           &total,
		TotalKind:       ops.TotalCertaintyExact,
		PageSize:        1000,
		PageCountKind:   ops.PageCountKindUnknown,
		ConsequenceSummary: ops.ConsequenceSummary{
			WorkSummary: "none; no runtime element timelines will be loaded",
		},
	})

	require.Equal(t, []string{
		"slow analysis scope: EmptyProcess matched no process instances; page size: 1000",
		"slow analysis: none; no runtime element timelines will be loaded",
	}, got)
}

// TestFormatOpsPreflightScopeNamesBasicInspectionResources verifies the shared preflight formatter has stable resource labels for the basic get rollout.
func TestFormatOpsPreflightScopeNamesBasicInspectionResources(t *testing.T) {
	total := int64(3000)
	pages := int64(3)
	tests := []struct {
		name         string
		coreResource string
		command      string
		selector     string
		want         string
	}{
		{name: "process instances", coreResource: "process_instance", command: "get process-instance", selector: "active instances", want: "process-instance search scope: active instances matched 3000 process instances; page size: 1000; discovery pages: 3"},
		{name: "incidents", coreResource: "incident", command: "get incident", selector: "active incidents", want: "incident search scope: active incidents matched 3000 incidents; page size: 1000; discovery pages: 3"},
		{name: "jobs", coreResource: "job", command: "get job", selector: "failed jobs", want: "job search scope: failed jobs matched 3000 jobs; page size: 1000; discovery pages: 3"},
		{name: "elements", coreResource: "element", command: "get element", selector: "active elements", want: "element search scope: active elements matched 3000 elements; page size: 1000; discovery pages: 3"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := formatOpsPreflightScope(ops.PreflightScope{
				Command:         tc.command,
				SelectorSummary: tc.selector,
				CoreResource:    tc.coreResource,
				Total:           &total,
				TotalKind:       ops.TotalCertaintyExact,
				PageSize:        1000,
				PageCount:       &pages,
				PageCountKind:   ops.PageCountKindExact,
			})

			require.Equal(t, tc.want, got[0])
		})
	}
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

// TestValidateOpsWorkflowReportFlags verifies shared report flag dependency and format errors.
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

// TestResolveOpsRepairReportFormatUsesSharedInference verifies repair targets expose the shared ops report format rules.
func TestResolveOpsRepairReportFormatUsesSharedInference(t *testing.T) {
	got, err := resolveOpsRepairReportFormat("repair.json", "")
	require.NoError(t, err)
	require.Equal(t, "json", got)

	got, err = resolveOpsRepairReportFormat("repair.md", "json")
	require.NoError(t, err)
	require.Equal(t, "json", got)

	got, err = resolveOpsRepairReportFormat("", "")
	require.NoError(t, err)
	require.Empty(t, got)

	_, err = resolveOpsRepairReportFormat("repair.md", "yaml")
	require.EqualError(t, err, `invalid input: invalid flag value: unsupported ops workflow report format "yaml"`)
}

// TestOpsExecuteRetentionPolicyReportFlagsReuseSharedContract verifies retention-policy report flags delegate to shared validation.
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

// TestOpsExecuteSmokeTestReportFlagsReuseSharedContract verifies smoke-test report flags delegate to shared validation.
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

// TestWriteOpsWorkflowReportFilePreservesExistingUntilConfirmed verifies report writes do not clobber existing files before mutation.
func TestWriteOpsWorkflowReportFilePreservesExistingUntilConfirmed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.md")
	require.NoError(t, writeOpsWorkflowReportFile(path, []byte("first"), OpsWorkflowReportPreserveExisting))
	require.Equal(t, "first", readOpsReportTestFile(t, path))

	err := writeOpsWorkflowReportFile(path, []byte("second"), OpsWorkflowReportPreserveExisting)
	require.EqualError(t, err, "report file already exists: "+path)
	require.Equal(t, "first", readOpsReportTestFile(t, path))

	require.NoError(t, writeOpsWorkflowReportFile(path, []byte("second"), OpsWorkflowReportOverwriteExisting))
	require.Equal(t, "second", readOpsReportTestFile(t, path))
}

// TestOpsWorkflowReportWriteModeForConfirmedMutation verifies only confirmed mutations may overwrite audit files.
func TestOpsWorkflowReportWriteModeForConfirmedMutation(t *testing.T) {
	require.Equal(t, OpsWorkflowReportPreserveExisting, opsWorkflowReportWriteModeForConfirmedMutation(false))
	require.Equal(t, OpsWorkflowReportOverwriteExisting, opsWorkflowReportWriteModeForConfirmedMutation(true))
}

// TestValidateOpsWorkflowReportPathForPlanning verifies planning catches existing report files before remote work.
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
	require.Equal(t, "existing", readOpsReportTestFile(t, existing))
}

// readOpsReportTestFile keeps report file assertions close to report helper tests.
func readOpsReportTestFile(t *testing.T, path string) string {
	t.Helper()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return string(data)
}
