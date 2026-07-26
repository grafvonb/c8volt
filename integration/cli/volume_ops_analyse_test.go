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

func TestVolumeOpsAnalyseFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "ops-analyse",
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

		records, err := runVolumeOpsAnalyseScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "ops-analyse", report.Datasets)
	writeVolumeProgressReport(t, "ops-analyse", report.Records)
	writeVolumePipelineReport(t, "ops-analyse", report.Records)
	writeVolumeOpsReportEvidence(t, "ops-analyse", nil)
	writeCommandProposals(t, appendOpsAnalyseCommandGapProposals(nil))
	writeEmbeddedBPMNProposals(t, appendOpsAnalyseEmbeddedBPMNGapProposals(nil))
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume ops analyse scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeOpsAnalyseScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	keys := firstNStrings(dataset.PositiveProcessInstanceKeys, 4)
	if len(keys) < 4 {
		return records, fmt.Errorf("ops analyse volume dataset for profile %q has %d positive keys, want at least 4", profile.Name, len(keys))
	}
	pdKey := firstString(dataset.PositiveProcessDefinitionKeys)
	if pdKey == "" {
		return records, fmt.Errorf("ops analyse volume dataset for profile %q has no process definition key", profile.Name)
	}

	jsonResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-json-timeline-duration", "--json", "ops", "analyse", "slow-process-instances", "--key", keys[0], "--key", keys[1], "--with-full-timeline", "--dur-longer", "1ms", "--dur-element-longer", "1ms")
	jsonRecord := volumeOpsAnalyseRecord(profile, dataset, jsonResult, "volume-ops-analyse-json-timeline-duration", "json", []string{"key", "with-full-timeline", "dur-longer", "dur-element-longer"})
	if err := validateVolumeOpsAnalyseJSON(jsonResult, 1); err != nil {
		jsonRecord.Outcome = volumeOutcomeFail
		jsonRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-json-timeline-duration: %v", err))
	}
	records = append(records, jsonRecord)

	keysOnlyResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-keys-only-search-limit", "--keys-only", "ops", "analyse", "spi", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--state", "active", "--batch-size", "1", "--limit", "2")
	keysOnlyRecord := volumeOpsAnalyseRecord(profile, dataset, keysOnlyResult, "volume-ops-analyse-keys-only-search-limit", "keys-only", []string{"bpmn-process-id", "state", "batch-size", "limit"})
	if err := validateVolumeOpsAnalyseKeysOnly(keysOnlyResult, 2); err != nil {
		keysOnlyRecord.Outcome = volumeOutcomeFail
		keysOnlyRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-keys-only-search-limit: %v", err))
	}
	records = append(records, keysOnlyRecord)

	stdin := keys[2] + "\n"
	humanResult := runC8VoltWithInput(t, "volume-ops-analyse-human-stdin-element-filter", stdin, argsForProfile(profile.Name, "ops", "analyse", "slow-process-instances", "-", "--with-full-timeline", "--type", "USER_TASK", "--element-state", "active", "--element-id", "SimpleUserTask_UserTask")...)
	humanRecord := volumeOpsAnalyseRecord(profile, dataset, humanResult, "volume-ops-analyse-human-stdin-element-filter", "one-line", []string{"stdin", "with-full-timeline", "type", "element-state", "element-id"})
	humanRecord.StdinPath = writeVolumeStdinKeys(t, "volume-ops-analyse-human-stdin-element-filter", []string{keys[2]})
	if err := validateVolumeOpsAnalyseHuman(humanResult); err != nil {
		humanRecord.Outcome = volumeOutcomeFail
		humanRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-human-stdin-element-filter: %v", err))
	}
	records = append(records, humanRecord)

	pdResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-json-pdkey-no-incidents", "--json", "ops", "analyse", "spi", "--pd-key", pdKey, "--state", "active", "--limit", "1", "--no-incidents-only", "--start-date-after", "2000-01-01", "--start-date-before", "2999-01-01")
	pdRecord := volumeOpsAnalyseRecord(profile, dataset, pdResult, "volume-ops-analyse-json-pdkey-no-incidents", "json", []string{"pd-key", "state", "limit", "no-incidents-only", "start-date-after", "start-date-before"})
	if err := validateVolumeOpsAnalyseJSON(pdResult, 1); err != nil {
		pdRecord.Outcome = volumeOutcomeFail
		pdRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-json-pdkey-no-incidents: %v", err))
	}
	records = append(records, pdRecord)

	listenerResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-json-with-listeners", "--json", "ops", "analyse", "spi", "--key", keys[3], "--with-listeners")
	listenerRecord := volumeOpsAnalyseRecord(profile, dataset, listenerResult, "volume-ops-analyse-json-with-listeners", "json", []string{"key", "with-listeners"})
	if err := validateVolumeOpsAnalyseJSON(listenerResult, 1); err != nil {
		listenerRecord.Outcome = volumeOutcomeFail
		listenerRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-json-with-listeners: %v", err))
	}
	records = append(records, listenerRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeOpsAnalyseRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string) evidenceRecord {
	record := commandEvidence("ops analyse slow-process-instances", scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "retained"}
	record.ResourceKeys = append([]string(nil), dataset.PositiveProcessInstanceKeys...)
	return record
}

func validateVolumeOpsAnalyseJSON(result commandResult, minItems int) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse volume"); err != nil {
		return err
	}
	if err := requireVolumeJSON(result.Stdout); err != nil {
		return err
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return err
	}
	if err := requireVolumeEnvelopeOutcome(result.Stdout, "succeeded"); err != nil {
		return err
	}
	var payload struct {
		Items []struct {
			Key string `json:"key"`
		} `json:"items"`
		Count int  `json:"count"`
		Empty bool `json:"empty"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode ops analyse payload: %w", err)
	}
	if len(payload.Items) < minItems {
		return fmt.Errorf("ops analyse returned %d items, want at least %d; empty=%t count=%d", len(payload.Items), minItems, payload.Empty, payload.Count)
	}
	for _, item := range payload.Items {
		if item.Key == "" {
			return fmt.Errorf("ops analyse item missing key: %q", compactLogSnippet(result.Stdout, 300))
		}
	}
	return nil
}

func validateVolumeOpsAnalyseKeysOnly(result commandResult, limit int) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse keys-only volume"); err != nil {
		return err
	}
	if err := requireVolumeKeysOnly(result.Stdout); err != nil {
		return err
	}
	keys := nonEmptyVolumeLines(result.Stdout)
	if len(keys) == 0 {
		return fmt.Errorf("ops analyse keys-only returned no keys")
	}
	if len(keys) > limit {
		return fmt.Errorf("ops analyse keys-only returned %d keys, want at most %d", len(keys), limit)
	}
	return nil
}

func validateVolumeOpsAnalyseHuman(result commandResult) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse human volume"); err != nil {
		return err
	}
	if !strings.Contains(result.Stdout, "process instances:") {
		return fmt.Errorf("ops analyse human output missing final count: %q", compactLogSnippet(result.Stdout, 300))
	}
	return nil
}

func nonEmptyVolumeLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
