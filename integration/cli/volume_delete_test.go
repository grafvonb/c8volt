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

func TestVolumeDeleteFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "delete",
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

		records, err := runVolumeDeleteProcessInstanceScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "delete", report.Datasets)
	writeVolumeProgressReport(t, "delete", report.Records)
	writeVolumePipelineReport(t, "delete", nil)
	writeVolumeOpsReportEvidence(t, "delete", nil)
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume delete scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeDeleteProcessInstanceScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	keys := firstNStrings(dataset.PositiveProcessInstanceKeys, 5)
	if len(keys) < 5 {
		return records, fmt.Errorf("delete volume dataset for profile %q has %d positive keys, want at least 5", profile.Name, len(keys))
	}

	dryRunResult := runC8VoltForProfile(t, profile.Name, "volume-delete-pi-json-force-dry-run", "--automation", "--json", "delete", "pi", "--key", keys[0], "--force", "--dry-run", "--workers", "2")
	dryRunRecord := volumeDeleteRecord(profile, dataset, dryRunResult, "volume-delete-pi-json-force-dry-run", "json", []string{"key", "force", "dry-run", "workers"})
	dryRunRecord.Preview = true
	if err := validateVolumeDeleteDryRun(t, profile, dryRunResult, []string{keys[0]}); err != nil {
		dryRunRecord.Outcome = volumeOutcomeFail
		dryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-delete-pi-json-force-dry-run: %v", err))
	}
	records = append(records, dryRunRecord)

	noForceResult := runC8VoltForProfile(t, profile.Name, "volume-delete-pi-active-refused-without-force", "--automation", "--json", "delete", "pi", "--key", keys[1])
	noForceRecord := volumeDeleteRecord(profile, dataset, noForceResult, "volume-delete-pi-active-refused-without-force", "json", []string{"key"})
	if err := validateVolumeDeleteActiveRefused(t, profile, noForceResult, keys[1]); err != nil {
		noForceRecord.Outcome = volumeOutcomeFail
		noForceRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-delete-pi-active-refused-without-force: %v", err))
	}
	records = append(records, noForceRecord)

	limitResult := runC8VoltForProfile(t, profile.Name, "volume-delete-pi-search-limit-dry-run", "--automation", "--json", "delete", "pi", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--state", "active", "--batch-size", "1", "--limit", "1", "--force", "--dry-run")
	limitRecord := volumeDeleteRecord(profile, dataset, limitResult, "volume-delete-pi-search-limit-dry-run", "json", []string{"bpmn-process-id", "state", "batch-size", "limit", "force", "dry-run"})
	limitRecord.Preview = true
	if err := validateVolumeDeleteSearchLimitDryRun(limitResult, 1); err != nil {
		limitRecord.Outcome = volumeOutcomeFail
		limitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-delete-pi-search-limit-dry-run: %v", err))
	}
	records = append(records, limitRecord)

	confirmedResult := runC8VoltForProfile(t, profile.Name, "volume-delete-pi-json-force-confirmed", "--automation", "--json", "delete", "pi", "--key", keys[2], "--key", keys[3], "--force", "--workers", "2")
	confirmedRecord := volumeDeleteRecord(profile, dataset, confirmedResult, "volume-delete-pi-json-force-confirmed", "json", []string{"key", "force", "workers"})
	confirmedRecord.ConfirmedMutation = true
	if err := validateVolumeDeleteConfirmed(t, profile, confirmedResult, keys[2:4]); err != nil {
		confirmedRecord.Outcome = volumeOutcomeFail
		confirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-delete-pi-json-force-confirmed: %v", err))
	}
	records = append(records, confirmedRecord)

	stdin := keys[4] + "\n"
	noWaitResult := runC8VoltWithInput(t, "volume-delete-pi-stdin-force-no-wait", stdin, argsForProfile(profile.Name, "--automation", "--json", "delete", "pi", "-", "--force", "--no-wait", "--no-state-check")...)
	noWaitRecord := volumeDeleteRecord(profile, dataset, noWaitResult, "volume-delete-pi-stdin-force-no-wait", "json", []string{"stdin", "force", "no-wait", "no-state-check"})
	noWaitRecord.StdinPath = writeVolumeStdinKeys(t, "volume-delete-pi-stdin-force-no-wait", []string{keys[4]})
	noWaitRecord.ConfirmedMutation = true
	if err := validateVolumeDeleteAccepted(noWaitResult, []string{keys[4]}); err != nil {
		noWaitRecord.Outcome = volumeOutcomeFail
		noWaitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-delete-pi-stdin-force-no-wait: %v", err))
	}
	records = append(records, noWaitRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeDeleteRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string) evidenceRecord {
	record := commandEvidence("delete process-instance", scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, "mutated", "cleanup_failed"}
	record.ResourceKeys = append([]string(nil), dataset.PositiveProcessInstanceKeys...)
	return record
}

func validateVolumeDeleteDryRun(t *testing.T, profile integrationProfile, result commandResult, keys []string) error {
	t.Helper()
	if err := validateVolumeDeleteDryRunPayload(result); err != nil {
		return err
	}
	for _, key := range keys {
		if err := requireProcessInstanceState(t, profile, key, "ACTIVE"); err != nil {
			return err
		}
	}
	return nil
}

func validateVolumeDeleteSearchLimitDryRun(result commandResult, limit int) error {
	if err := validateVolumeDeleteDryRunPayload(result); err != nil {
		return err
	}
	var payload struct {
		RequestedCount    int  `json:"requestedCount"`
		MutationSubmitted bool `json:"mutationSubmitted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode delete search dry-run payload: %w", err)
	}
	if payload.RequestedCount > limit {
		return fmt.Errorf("delete search dry-run requested %d keys, want at most limit %d", payload.RequestedCount, limit)
	}
	return nil
}

func validateVolumeDeleteDryRunPayload(result commandResult) error {
	if err := requireVolumeCommandSuccess(result, "delete pi dry-run volume"); err != nil {
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
		return fmt.Errorf("delete dry-run payload reported mutationSubmitted=true")
	}
	return nil
}

func validateVolumeDeleteActiveRefused(t *testing.T, profile integrationProfile, result commandResult, key string) error {
	t.Helper()
	if result.Err == nil {
		return fmt.Errorf("delete active process instance without --force succeeded unexpectedly")
	}
	combined := strings.ToLower(result.Stdout + "\n" + result.Stderr)
	if !strings.Contains(combined, "use --force") && !strings.Contains(combined, "not in a final state") && !strings.Contains(combined, "refusing to delete") {
		return fmt.Errorf("delete active refusal did not explain force requirement: %q", compactLogSnippet(result.Stdout+"\n"+result.Stderr, 300))
	}
	return requireProcessInstanceState(t, profile, key, "ACTIVE")
}

func validateVolumeDeleteConfirmed(t *testing.T, profile integrationProfile, result commandResult, keys []string) error {
	t.Helper()
	if err := validateVolumeDeleteReportEnvelope(result, "succeeded", keys); err != nil {
		return err
	}
	for _, key := range keys {
		if err := requireProcessInstanceAbsent(t, profile, key); err != nil {
			return err
		}
	}
	return nil
}

func validateVolumeDeleteAccepted(result commandResult, keys []string) error {
	return validateVolumeDeleteReportEnvelope(result, "accepted", keys)
}

func validateVolumeDeleteReportEnvelope(result commandResult, wantOutcome string, keys []string) error {
	if err := requireVolumeCommandSuccess(result, "delete pi volume"); err != nil {
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
		return fmt.Errorf("decode delete report payload: %w", err)
	}
	got := map[string]bool{}
	for _, item := range payload.Items {
		got[item.Key] = item.Ok
	}
	for _, key := range keys {
		if !got[key] {
			return fmt.Errorf("delete report missing successful key %s; got=%v", key, got)
		}
	}
	return nil
}

func requireProcessInstanceAbsent(t *testing.T, profile integrationProfile, key string) error {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "volume-pi-absent-"+key, "--automation", "--json", "get", "pi", "--key", key)
	if result.Err != nil {
		combined := strings.ToLower(result.Stdout + "\n" + result.Stderr)
		if strings.Contains(combined, "not found") || strings.Contains(combined, "404") || strings.Contains(combined, "missing") {
			return nil
		}
		return fmt.Errorf("get pi absent verification failed unexpectedly: %v; %s", result.Err, compactLogSnippet(result.Stdout+"\n"+result.Stderr, 300))
	}
	var payload struct {
		Total int `json:"total"`
		Items []struct {
			Key string `json:"key"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode get pi absent payload: %w", err)
	}
	if payload.Total == 0 && len(payload.Items) == 0 {
		return nil
	}
	return fmt.Errorf("process instance %s still visible after delete: %q", key, compactLogSnippet(result.Stdout, 300))
}
