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

func TestVolumeCancelFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "cancel",
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

		records, err := runVolumeCancelProcessInstanceScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "cancel", report.Datasets)
	writeVolumeProgressReport(t, "cancel", report.Records)
	writeVolumePipelineReport(t, "cancel", nil)
	writeVolumeOpsReportEvidence(t, "cancel", nil)
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume cancel scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeCancelProcessInstanceScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	keys := firstNStrings(dataset.PositiveProcessInstanceKeys, 5)
	if len(keys) < 5 {
		return records, fmt.Errorf("cancel volume dataset for profile %q has %d positive keys, want at least 5", profile.Name, len(keys))
	}

	dryRunResult := runC8VoltForProfile(t, profile.Name, "volume-cancel-pi-json-dry-run", "--automation", "--json", "cancel", "pi", "--key", keys[0], "--key", keys[1], "--dry-run", "--workers", "2")
	dryRunRecord := volumeCancelRecord(profile, dataset, dryRunResult, "volume-cancel-pi-json-dry-run", "json", []string{"key", "dry-run", "workers"})
	if err := validateVolumeCancelDryRun(t, profile, dryRunResult, keys[:2]); err != nil {
		dryRunRecord.Outcome = volumeOutcomeFail
		dryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-cancel-pi-json-dry-run: %v", err))
	}
	records = append(records, dryRunRecord)

	limitResult := runC8VoltForProfile(t, profile.Name, "volume-cancel-pi-search-limit-dry-run", "--automation", "--json", "cancel", "pi", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--state", "active", "--batch-size", "1", "--limit", "1", "--dry-run")
	limitRecord := volumeCancelRecord(profile, dataset, limitResult, "volume-cancel-pi-search-limit-dry-run", "json", []string{"bpmn-process-id", "state", "batch-size", "limit", "dry-run"})
	if err := validateVolumeCancelSearchLimitDryRun(limitResult, 1); err != nil {
		limitRecord.Outcome = volumeOutcomeFail
		limitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-cancel-pi-search-limit-dry-run: %v", err))
	}
	records = append(records, limitRecord)

	confirmedResult := runC8VoltForProfile(t, profile.Name, "volume-cancel-pi-json-confirmed", "--automation", "--json", "cancel", "pi", "--key", keys[2], "--key", keys[3], "--workers", "2")
	confirmedRecord := volumeCancelRecord(profile, dataset, confirmedResult, "volume-cancel-pi-json-confirmed", "json", []string{"key", "workers"})
	confirmedRecord.ConfirmedMutation = true
	if err := validateVolumeCancelConfirmed(t, profile, confirmedResult, keys[2:4]); err != nil {
		confirmedRecord.Outcome = volumeOutcomeFail
		confirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-cancel-pi-json-confirmed: %v", err))
	}
	records = append(records, confirmedRecord)

	stdin := keys[4] + "\n"
	noWaitResult := runC8VoltWithInput(t, "volume-cancel-pi-stdin-no-wait", stdin, argsForProfile(profile.Name, "--automation", "--json", "cancel", "pi", "-", "--no-wait", "--no-state-check")...)
	noWaitRecord := volumeCancelRecord(profile, dataset, noWaitResult, "volume-cancel-pi-stdin-no-wait", "json", []string{"stdin", "no-wait", "no-state-check"})
	noWaitRecord.StdinPath = writeVolumeStdinKeys(t, "volume-cancel-pi-stdin-no-wait", []string{keys[4]})
	noWaitRecord.ConfirmedMutation = true
	if err := validateVolumeCancelAccepted(noWaitResult, []string{keys[4]}); err != nil {
		noWaitRecord.Outcome = volumeOutcomeFail
		noWaitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-cancel-pi-stdin-no-wait: %v", err))
	}
	records = append(records, noWaitRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeCancelRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string) evidenceRecord {
	record := commandEvidence("cancel process-instance", scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, "mutated", "retained"}
	record.ResourceKeys = append([]string(nil), dataset.PositiveProcessInstanceKeys...)
	return record
}

func validateVolumeCancelDryRun(t *testing.T, profile integrationProfile, result commandResult, keys []string) error {
	t.Helper()
	if err := validateVolumeCancelDryRunPayload(result); err != nil {
		return err
	}
	for _, key := range keys {
		if err := requireProcessInstanceState(t, profile, key, "ACTIVE"); err != nil {
			return err
		}
	}
	return nil
}

func validateVolumeCancelSearchLimitDryRun(result commandResult, limit int) error {
	if err := validateVolumeCancelDryRunPayload(result); err != nil {
		return err
	}
	var payload struct {
		RequestedCount    int  `json:"requestedCount"`
		MutationSubmitted bool `json:"mutationSubmitted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode cancel search dry-run payload: %w", err)
	}
	if payload.RequestedCount > limit {
		return fmt.Errorf("cancel search dry-run requested %d keys, want at most limit %d", payload.RequestedCount, limit)
	}
	return nil
}

func validateVolumeCancelDryRunPayload(result commandResult) error {
	if err := requireVolumeCommandSuccess(result, "cancel pi dry-run volume"); err != nil {
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
	if strings.Contains(result.Stdout, `"mutationSubmitted": true`) {
		return fmt.Errorf("cancel dry-run payload reported mutationSubmitted=true")
	}
	return nil
}

func validateVolumeCancelConfirmed(t *testing.T, profile integrationProfile, result commandResult, keys []string) error {
	t.Helper()
	if err := validateVolumeCancelReportEnvelope(result, "succeeded", keys); err != nil {
		return err
	}
	for _, key := range keys {
		if err := requireProcessInstanceStateOneOf(t, profile, key, "CANCELED", "TERMINATED"); err != nil {
			return err
		}
	}
	return nil
}

func validateVolumeCancelAccepted(result commandResult, keys []string) error {
	return validateVolumeCancelReportEnvelope(result, "accepted", keys)
}

func validateVolumeCancelReportEnvelope(result commandResult, wantOutcome string, keys []string) error {
	if err := requireVolumeCommandSuccess(result, "cancel pi volume"); err != nil {
		return err
	}
	if err := requireVolumeJSON(result.Stdout); err != nil {
		return err
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return err
	}
	if err := requireVolumeEnvelopeOutcome(result.Stdout, wantOutcome); err != nil {
		return err
	}
	var payload struct {
		Items []struct {
			Key string `json:"key"`
			Ok  bool   `json:"ok"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode cancel report payload: %w", err)
	}
	got := map[string]bool{}
	for _, item := range payload.Items {
		got[item.Key] = item.Ok
	}
	for _, key := range keys {
		if !got[key] {
			return fmt.Errorf("cancel report missing successful key %s; got=%v", key, got)
		}
	}
	return nil
}

func requireProcessInstanceState(t *testing.T, profile integrationProfile, key string, wantState string) error {
	t.Helper()
	return requireProcessInstanceStateOneOf(t, profile, key, wantState)
}

func requireProcessInstanceStateOneOf(t *testing.T, profile integrationProfile, key string, wantStates ...string) error {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "volume-pi-state-"+sanitizeEvidenceName(strings.Join(wantStates, "-"))+"-"+key, "--automation", "--json", "get", "pi", "--key", key)
	if err := requireVolumeCommandSuccess(result, "get pi state verification"); err != nil {
		return err
	}
	for _, wantState := range wantStates {
		if strings.Contains(result.Stdout, `"state": "`+wantState+`"`) {
			return nil
		}
	}
	return fmt.Errorf("process instance %s state is not one of %v: %q", key, wantStates, compactLogSnippet(result.Stdout, 300))
}
