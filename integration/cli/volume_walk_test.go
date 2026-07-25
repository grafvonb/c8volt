// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestVolumeWalkFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "walk",
		Marker:       suite.marker,
		DatasetCount: datasetCount,
		Profiles:     profiles,
	}
	var failures []string
	for _, profile := range profiles {
		dataset, seedRecords, err := seedVolumeProcessInstanceDataset(t, profile, datasetCount)
		report.Datasets = append(report.Datasets, dataset)
		report.Records = append(report.Records, seedRecords...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}

		records, err := runVolumeWalkProcessInstanceScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "walk", report.Datasets)
	writeVolumeProgressReport(t, "walk", report.Records)
	writeVolumePipelineReport(t, "walk", nil)
	writeVolumeOpsReportEvidence(t, "walk", nil)
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume walk scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeWalkProcessInstanceScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string

	roots := firstNStrings(dataset.PositiveProcessInstanceKeys, 2)
	if len(roots) < 2 {
		return records, fmt.Errorf("walk volume dataset for profile %q has %d positive roots, want at least 2", profile.Name, len(roots))
	}

	for index, key := range roots {
		scenarioName := fmt.Sprintf("volume-walk-pi-flat-root-%d-of-%d", index+1, len(roots))
		result := runC8VoltForProfile(t, profile.Name, scenarioName, "walk", "pi", "--key", key, "--flat")
		record := volumeWalkRecord(profile, dataset, result, scenarioName, "one-line", []string{"key", "flat"})
		record.Behavior = fmt.Sprintf("completed-root-%d-of-%d", index+1, len(roots))
		if err := validateVolumeWalkHumanResult(result, key); err != nil {
			record.Outcome = volumeOutcomeFail
			record.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("%s: %v", scenarioName, err))
		}
		records = append(records, record)
	}

	scenarios := []struct {
		name       string
		args       []string
		outputMode string
		wantKey    string
		flags      []string
		json       bool
	}{
		{
			name:       "volume-walk-pi-parent-human",
			args:       []string{"walk", "pi", "--key", roots[0], "--parent"},
			outputMode: "one-line",
			wantKey:    roots[0],
			flags:      []string{"key", "parent"},
		},
		{
			name:       "volume-walk-pi-children-human",
			args:       []string{"walk", "pi", "--key", roots[0], "--children"},
			outputMode: "one-line",
			wantKey:    roots[0],
			flags:      []string{"key", "children"},
		},
		{
			name:       "volume-walk-pi-json-vars-elements",
			args:       []string{"--json", "walk", "pi", "--key", roots[0], "--flat", "--with-vars", "--var-value-limit", "64", "--with-elements"},
			outputMode: "json",
			wantKey:    roots[0],
			flags:      []string{"key", "flat", "with-vars", "var-value-limit", "with-elements"},
			json:       true,
		},
		{
			name:       "volume-walk-pi-json-incidents",
			args:       []string{"--json", "walk", "pi", "--key", roots[0], "--flat", "--with-incidents", "--incident-state", "active", "--incident-message-limit", "64"},
			outputMode: "json",
			wantKey:    roots[0],
			flags:      []string{"key", "flat", "with-incidents", "incident-state", "incident-message-limit"},
			json:       true,
		},
		{
			name:       "volume-walk-pi-listener-dependency",
			args:       []string{"walk", "pi", "--key", roots[0], "--with-listeners"},
			outputMode: "one-line",
			wantKey:    roots[0],
			flags:      []string{"key", "with-listeners"},
		},
	}

	for _, scenario := range scenarios {
		result := runC8VoltForProfile(t, profile.Name, scenario.name, scenario.args...)
		record := volumeWalkRecord(profile, dataset, result, scenario.name, scenario.outputMode, scenario.flags)
		if err := validateVolumeWalkScenarioResult(result, scenario.wantKey, scenario.json, scenario.name); err != nil {
			record.Outcome = volumeOutcomeFail
			record.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("%s: %v", scenario.name, err))
		}
		records = append(records, record)
	}

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeWalkRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string) evidenceRecord {
	record := commandEvidence("walk process-instance", scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting}
	record.ResourceKeys = append([]string(nil), dataset.allProcessInstanceKeys()...)
	return record
}

func validateVolumeWalkScenarioResult(result commandResult, wantKey string, wantJSON bool, scenarioName string) error {
	if scenarioName == "volume-walk-pi-listener-dependency" {
		if result.Err == nil {
			return fmt.Errorf("--with-listeners without --with-elements succeeded, want dependency validation")
		}
		if !strings.Contains(result.Stderr, "--with-listeners requires --with-elements") {
			return fmt.Errorf("listener dependency error missing expected text; stderr: %s", strings.TrimSpace(result.Stderr))
		}
		return nil
	}
	if wantJSON {
		return validateVolumeWalkJSONResult(result, wantKey)
	}
	return validateVolumeWalkHumanResult(result, wantKey)
}

func validateVolumeWalkHumanResult(result commandResult, wantKey string) error {
	if err := requireVolumeCommandSuccess(result, "walk pi volume"); err != nil {
		return err
	}
	if strings.TrimSpace(result.Stdout) == "" {
		return fmt.Errorf("walk pi human output is empty")
	}
	if !strings.Contains(result.Stdout, wantKey) {
		return fmt.Errorf("walk pi human output does not include selected key %s: %q", wantKey, compactLogSnippet(result.Stdout, 300))
	}
	return nil
}

func validateVolumeWalkJSONResult(result commandResult, wantKey string) error {
	if err := requireVolumeCommandSuccess(result, "walk pi JSON volume"); err != nil {
		return err
	}
	if err := requireVolumeJSON(result.Stdout); err != nil {
		return err
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return err
	}
	var payload struct {
		Keys []string `json:"keys"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode walk pi JSON payload: %w", err)
	}
	if !containsString(payload.Keys, wantKey) {
		return fmt.Errorf("walk pi JSON keys %v do not include selected key %s", payload.Keys, wantKey)
	}
	return nil
}

func firstNStrings(values []string, count int) []string {
	if len(values) < count {
		count = len(values)
	}
	return append([]string(nil), values[:count]...)
}
