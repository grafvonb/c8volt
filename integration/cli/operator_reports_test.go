// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestOpsRepairFamilyReportPaths proves existing reports survive rejected
// previews and executions, and a fresh execution path works for an empty scope.
func TestOpsRepairFamilyReportPaths(t *testing.T) {
	for _, profile := range selectedBatchProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			path := filepath.Join(suite.workDir, "report-path-"+profile.Name+".json")
			base := []string{"--automation", "--json", "ops", "repair", "incident", "--bpmn-process-id", suite.marker + "-absent", "--report-format", "json"}
			args := append(append([]string{}, base...), "--report-file", path, "--dry-run")
			preview := runC8VoltForProfile(t, profile.Name, "report-path-preview", args...)
			require.NoError(t, preview.Err, preview.Stderr)
			before, err := os.ReadFile(path)
			require.NoError(t, err)
			require.True(t, json.Valid(before))
			for _, dryRun := range []bool{true, false} {
				args := append(append([]string{}, base...), "--report-file", path)
				label := "report-path-execution-rejected"
				if dryRun {
					args = append(args, "--dry-run")
					label = "report-path-preview-rejected"
				}
				result := runC8VoltForProfile(t, profile.Name, label, args...)
				require.NotZero(t, result.ExitCode)
				require.Contains(t, result.Stdout+result.Stderr, "already exists")
				var envelope struct {
					Outcome string `json:"outcome"`
				}
				require.NoError(t, json.Unmarshal([]byte(result.Stdout), &envelope))
				require.Equal(t, "failed", envelope.Outcome)
				after, err := os.ReadFile(path)
				require.NoError(t, err)
				require.Equal(t, before, after)
			}
			fresh := filepath.Join(suite.workDir, "report-path-execution-"+profile.Name+".json")
			result := runC8VoltForProfile(t, profile.Name, "report-path-fresh-execution", append(base, "--report-file", fresh)...)
			require.NoError(t, result.Err, result.Stderr)
			require.NoError(t, requireVolumeEnvelopeOutcome(result.Stdout, "succeeded"))
			report, err := os.ReadFile(fresh)
			require.NoError(t, err)
			require.True(t, json.Valid(report))
			writeJSON(t, "report-paths-"+profile.Name+".json", map[string]any{"existingPreserved": true, "freshExecutionWritten": true, "scope": "authoritative empty unique process ID"})
		})
	}
}

// TestDeployEmbedRunFamilyC810Export exercises both current export examples
// without mixing 8.8/8.9 fixtures into the configured 8.10 output.
func TestDeployEmbedRunFamilyC810Export(t *testing.T) {
	for _, profile := range selectedBatchProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			want, err := filepath.Glob(filepath.Join(suite.repoRoot, "embedded", "processdefinitions", "C810_*.bpmn"))
			require.NoError(t, err)
			require.NotEmpty(t, want)
			var expected []string
			for _, file := range want {
				expected = append(expected, filepath.Base(file))
			}
			for _, selection := range []struct {
				name  string
				flags []string
			}{
				{"all", []string{"--all"}},
				{"glob", []string{"--file", "processdefinitions/C810_*.bpmn"}},
			} {
				out := filepath.Join(suite.workDir, "c810-export-"+profile.Name+"-"+selection.name)
				args := append([]string{"embed", "export", "--out", out}, selection.flags...)
				result := runC8VoltForProfile(t, profile.Name, "c810-export-"+selection.name, args...)
				require.NoError(t, result.Err, result.Stderr)
				var actual []string
				require.NoError(t, filepath.WalkDir(out, func(path string, entry os.DirEntry, err error) error {
					if err == nil && !entry.IsDir() {
						actual = append(actual, entry.Name())
					}
					return err
				}))
				require.ElementsMatch(t, expected, actual)
				writeJSON(t, "c810-export-"+profile.Name+"-"+selection.name+".json", actual)
			}
		})
	}
}
