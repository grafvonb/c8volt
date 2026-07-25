// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import "testing"

type volumeFamilyReport struct {
	Family       string               `json:"family"`
	Marker       string               `json:"marker"`
	DatasetCount int                  `json:"datasetCount"`
	Profiles     []integrationProfile `json:"profiles,omitempty"`
	Datasets     []volumeDataset      `json:"datasets,omitempty"`
	Records      []evidenceRecord     `json:"records"`
	Summary      volumeSummary        `json:"summary"`
}

type volumeSummary struct {
	ScenarioCount      int                 `json:"scenarioCount"`
	CommandsCovered    []string            `json:"commandsCovered"`
	FlagsCovered       map[string][]string `json:"flagsCovered,omitempty"`
	OutputModesCovered map[string][]string `json:"outputModesCovered,omitempty"`
	DataOwnership      map[string][]string `json:"dataOwnership,omitempty"`
	Failures           []string            `json:"failures,omitempty"`
}

func writeVolumeFamilyReport(t *testing.T, report volumeFamilyReport) string {
	t.Helper()
	report.Summary = summarizeVolumeRecords(report.Records)
	return writeJSON(t, "volume-"+sanitizeEvidenceName(report.Family)+".json", report)
}

func writeVolumeDataReport(t *testing.T, family string, datasets []volumeDataset) string {
	t.Helper()
	return writeJSON(t, "volume-data-"+sanitizeEvidenceName(family)+".json", datasets)
}

func writeVolumeProgressReport(t *testing.T, family string, records []evidenceRecord) string {
	t.Helper()
	return writeJSON(t, "volume-progress-"+sanitizeEvidenceName(family)+".json", evidenceRecordsOrEmpty(records))
}

func writeVolumePipelineReport(t *testing.T, family string, records []evidenceRecord) string {
	t.Helper()
	return writeJSON(t, "volume-pipelines-"+sanitizeEvidenceName(family)+".json", evidenceRecordsOrEmpty(records))
}

func writeVolumeOpsReportEvidence(t *testing.T, family string, records []evidenceRecord) string {
	t.Helper()
	return writeJSON(t, "volume-ops-reports-"+sanitizeEvidenceName(family)+".json", evidenceRecordsOrEmpty(records))
}

func evidenceRecordsOrEmpty(records []evidenceRecord) []evidenceRecord {
	if records == nil {
		return []evidenceRecord{}
	}
	return records
}

func summarizeVolumeRecords(records []evidenceRecord) volumeSummary {
	summary := volumeSummary{
		FlagsCovered:       map[string][]string{},
		OutputModesCovered: map[string][]string{},
		DataOwnership:      map[string][]string{},
	}
	commands := map[string]struct{}{}
	for _, record := range records {
		summary.ScenarioCount++
		if record.CommandPath != "" {
			commands[record.CommandPath] = struct{}{}
		}
		for _, flag := range record.CoveredFlags {
			summary.FlagsCovered[flag] = append(summary.FlagsCovered[flag], record.ScenarioName)
		}
		if record.OutputMode != "" {
			summary.OutputModesCovered[record.OutputMode] = append(summary.OutputModesCovered[record.OutputMode], record.ScenarioName)
		}
		for _, ownership := range record.DataOwnership {
			summary.DataOwnership[ownership] = append(summary.DataOwnership[ownership], record.ScenarioName)
		}
		if record.Outcome != volumeOutcomePass {
			summary.Failures = append(summary.Failures, record.ScenarioName)
		}
	}
	for command := range commands {
		summary.CommandsCovered = append(summary.CommandsCovered, command)
	}
	summary.CommandsCovered = sortedStrings(summary.CommandsCovered)
	return summary
}
