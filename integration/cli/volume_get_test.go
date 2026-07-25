// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestVolumeGetFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "get",
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

		records, err := runVolumeGetProcessInstanceScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "get", report.Datasets)
	writeVolumeProgressReport(t, "get", report.Records)
	writeVolumePipelineReport(t, "get", nil)
	writeVolumeOpsReportEvidence(t, "get", nil)
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume get scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeGetProcessInstanceScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string

	scenarios := []struct {
		name        string
		args        []string
		outputMode  string
		wantJSON    bool
		wantKeys    bool
		limit       int
		wantKeysHit []string
		wantAbsent  []string
		flags       []string
	}{
		{
			name:        "volume-get-pi-json-bpmn-limit-batch",
			args:        []string{"--automation", "--json", "get", "pi", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--batch-size", "1", "--limit", "2"},
			outputMode:  "json",
			wantJSON:    true,
			limit:       2,
			wantKeysHit: dataset.PositiveProcessInstanceKeys,
			wantAbsent:  dataset.NegativeProcessInstanceKeys,
			flags:       []string{"bpmn-process-id", "batch-size", "limit"},
		},
		{
			name:        "volume-get-pi-keys-only-bpmn-limit",
			args:        []string{"--keys-only", "get", "pi", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--limit", "2"},
			outputMode:  "keys-only",
			wantKeys:    true,
			limit:       2,
			wantKeysHit: dataset.PositiveProcessInstanceKeys,
			wantAbsent:  dataset.NegativeProcessInstanceKeys,
			flags:       []string{"bpmn-process-id", "limit"},
		},
		{
			name:        "volume-get-pi-json-negative-selector",
			args:        []string{"--automation", "--json", "get", "pi", "--bpmn-process-id", dataset.NegativeBpmnProcessID, "--limit", "1"},
			outputMode:  "json",
			wantJSON:    true,
			limit:       1,
			wantKeysHit: dataset.NegativeProcessInstanceKeys,
			wantAbsent:  dataset.PositiveProcessInstanceKeys,
			flags:       []string{"bpmn-process-id", "limit"},
		},
		{
			name:        "volume-get-pi-json-key-with-vars",
			args:        []string{"--automation", "--json", "get", "pi", "--key", firstString(dataset.PositiveProcessInstanceKeys), "--with-vars"},
			outputMode:  "json",
			wantJSON:    true,
			limit:       1,
			wantKeysHit: dataset.PositiveProcessInstanceKeys[:1],
			flags:       []string{"key", "with-vars"},
		},
	}

	for _, scenario := range scenarios {
		result := runC8VoltForProfile(t, profile.Name, scenario.name, scenario.args...)
		record := commandEvidence("get process-instance", scenario.name, result, volumeOutcomePass)
		record.Profile = profile.Name
		record.CamundaVersion = profile.ExpectedVersion
		record.CoveredFlags = scenario.flags
		record.OutputMode = scenario.outputMode
		record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting}
		record.ResourceKeys = append([]string(nil), dataset.allProcessInstanceKeys()...)
		if err := validateVolumeGetResult(result, scenario.wantJSON, scenario.wantKeys, scenario.limit, scenario.wantKeysHit, scenario.wantAbsent); err != nil {
			record.Outcome = volumeOutcomeFail
			record.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("%s: %v", scenario.name, err))
		}
		records = append(records, record)
	}

	totalResult := runC8VoltForProfile(t, profile.Name, "volume-get-pi-total-bpmn", "get", "pi", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--total")
	totalRecord := commandEvidence("get process-instance", "volume-get-pi-total-bpmn", totalResult, volumeOutcomePass)
	totalRecord.Profile = profile.Name
	totalRecord.CamundaVersion = profile.ExpectedVersion
	totalRecord.CoveredFlags = []string{"bpmn-process-id", "total"}
	totalRecord.OutputMode = "one-line"
	totalRecord.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting}
	totalRecord.ResourceKeys = append([]string(nil), dataset.PositiveProcessInstanceKeys...)
	if err := requireVolumeCommandSuccess(totalResult, "get pi --total"); err != nil {
		totalRecord.Outcome = volumeOutcomeFail
		totalRecord.FailureClass = volumeFailureProduct
		failures = append(failures, err.Error())
	} else {
		total, err := parseVolumeTotalOutput(totalResult.Stdout)
		if err != nil {
			totalRecord.Outcome = volumeOutcomeFail
			totalRecord.FailureClass = volumeFailureProduct
			failures = append(failures, err.Error())
		} else if total < len(dataset.PositiveProcessInstanceKeys) {
			totalRecord.Outcome = volumeOutcomeFail
			totalRecord.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("get pi --total returned %d, want at least seeded count %d", total, len(dataset.PositiveProcessInstanceKeys)))
		}
	}
	records = append(records, totalRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func parseVolumeTotalOutput(output string) (int, error) {
	total, err := strconv.Atoi(strings.TrimSpace(output))
	if err != nil {
		return 0, fmt.Errorf("parse total output %q: %w", strings.TrimSpace(output), err)
	}
	return total, nil
}

func validateVolumeGetResult(result commandResult, wantJSON bool, wantKeys bool, limit int, wantPresent []string, wantAbsent []string) error {
	if err := requireVolumeCommandSuccess(result, "get pi volume"); err != nil {
		return err
	}
	if wantJSON {
		if err := requireVolumeJSON(result.Stdout); err != nil {
			return err
		}
		if err := requireMachineStdoutClean(result.Stdout); err != nil {
			return err
		}
		var payload seededProcessInstances
		if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
			return fmt.Errorf("decode get pi JSON payload: %w", err)
		}
		if limit > 0 && len(payload.Items) > limit {
			return fmt.Errorf("returned %d items, want at most limit %d", len(payload.Items), limit)
		}
		gotKeys := observedProcessInstanceKeys(payload.Items)
		if len(wantPresent) > 0 && len(gotKeys) == 0 {
			foundInWrappedPayload := false
			for _, key := range wantPresent {
				if key != "" && strings.Contains(result.Stdout, key) {
					foundInWrappedPayload = true
					break
				}
			}
			if !foundInWrappedPayload {
				return fmt.Errorf("expected at least one suite-owned key in JSON output")
			}
		}
		for _, absent := range wantAbsent {
			if containsString(gotKeys, absent) {
				return fmt.Errorf("unexpected key %s from opposite selector in JSON output", absent)
			}
		}
		return nil
	}
	if wantKeys {
		if err := requireVolumeKeysOnly(result.Stdout); err != nil {
			return err
		}
		if err := requireMachineStdoutClean(result.Stdout); err != nil {
			return err
		}
		lines := splitNonEmptyLines(result.Stdout)
		if limit > 0 && len(lines) > limit {
			return fmt.Errorf("returned %d key lines, want at most limit %d", len(lines), limit)
		}
		if len(lines) == 0 {
			return fmt.Errorf("expected at least one keys-only line")
		}
		for _, absent := range wantAbsent {
			if containsString(lines, absent) {
				return fmt.Errorf("unexpected key %s from opposite selector in keys-only output", absent)
			}
		}
	}
	return nil
}

func splitNonEmptyLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
