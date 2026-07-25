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

type realStateDestructiveDataset struct {
	Fixture realStateFixture `json:"fixture"`
	Volume  volumeDataset    `json:"volumeDataset"`
}

func TestRealStateDestructiveFamily(t *testing.T) {
	profiles := selectedRealStateC89Profiles(t)

	report := realStateFamilyReport{
		Family:   "destructive",
		Marker:   suite.marker,
		Profiles: profiles,
	}
	var failures []string
	for _, profile := range profiles {
		dataset, records, err := seedRealStateDestructiveDataset(t, profile, 4)
		report.Fixtures = append(report.Fixtures, dataset.Fixture)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}

		records, err = runRealStateDestructiveScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeRealStateDataReport(t, "destructive", report.Fixtures)
	writeRealStateProgressReport(t, "destructive", report.Records)
	writeRealStateOpsReportEvidence(t, "destructive", nil)
	writeRealStateFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("real-state destructive scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func seedRealStateDestructiveDataset(t *testing.T, profile integrationProfile, count int) (realStateDestructiveDataset, []evidenceRecord, error) {
	t.Helper()
	dataset, records, err := seedVolumeProcessInstanceDataset(t, profile, count)
	fixture := realStateFixture{
		FixtureKind:           "embedded user-task BPMN through c8volt commands",
		BpmnProcessID:         dataset.PositiveBpmnProcessID,
		ProcessDefinitionKeys: append([]string(nil), dataset.PositiveProcessDefinitionKeys...),
		ProcessInstanceKeys:   append([]string(nil), dataset.PositiveProcessInstanceKeys...),
		Marker:                suite.marker,
		Profile:               profile.Name,
		CamundaVersion:        profile.ExpectedVersion,
		RequiredState:         "active process instances for destructive post-state proof",
		CurrentEvidenceLevel:  realStateOutcomePartialLive,
		TargetRealStateProof:  "dry-run retains active candidates; confirmed cancel/delete changes observable state",
		ObservedState:         "active process instances",
	}
	if err != nil {
		return realStateDestructiveDataset{Fixture: fixture, Volume: dataset}, records, err
	}
	return realStateDestructiveDataset{Fixture: fixture, Volume: dataset}, records, nil
}

func runRealStateDestructiveScenarios(t *testing.T, profile integrationProfile, dataset realStateDestructiveDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	keys := firstNStrings(dataset.Fixture.ProcessInstanceKeys, 4)
	if len(keys) < 4 {
		return records, fmt.Errorf("real-state destructive dataset for profile %q has %d active keys, want at least 4", profile.Name, len(keys))
	}

	cancelDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-cancel-dry-run", "--automation", "--json", "cancel", "pi", "--key", keys[0], "--dry-run", "--workers", "1")
	cancelDryRunRecord := realStateDestructiveRecord(profile, dataset, cancelDryRunResult, "cancel process-instance", "real-state-destructive-cancel-dry-run", "json", []string{"key", "dry-run", "workers"}, []string{keys[0]}, true, false)
	if err := validateVolumeCancelDryRun(t, profile, cancelDryRunResult, []string{keys[0]}); err != nil {
		cancelDryRunRecord.Outcome = realStateOutcomeFailed
		cancelDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-cancel-dry-run: %v", err))
	}
	records = append(records, cancelDryRunRecord)

	cancelConfirmedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-cancel-confirmed", "--automation", "--json", "cancel", "pi", "--key", keys[1], "--workers", "1")
	cancelConfirmedRecord := realStateDestructiveRecord(profile, dataset, cancelConfirmedResult, "cancel process-instance", "real-state-destructive-cancel-confirmed", "json", []string{"key", "workers"}, []string{keys[1]}, false, true)
	if err := validateVolumeCancelConfirmed(t, profile, cancelConfirmedResult, []string{keys[1]}); err != nil {
		cancelConfirmedRecord.Outcome = realStateOutcomeFailed
		cancelConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-cancel-confirmed: %v", err))
	}
	records = append(records, cancelConfirmedRecord)

	deleteDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-delete-dry-run", "--automation", "--json", "delete", "pi", "--key", keys[2], "--force", "--dry-run", "--workers", "1")
	deleteDryRunRecord := realStateDestructiveRecord(profile, dataset, deleteDryRunResult, "delete process-instance", "real-state-destructive-delete-dry-run", "json", []string{"key", "force", "dry-run", "workers"}, []string{keys[2]}, true, false)
	if err := validateVolumeDeleteDryRun(t, profile, deleteDryRunResult, []string{keys[2]}); err != nil {
		deleteDryRunRecord.Outcome = realStateOutcomeFailed
		deleteDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-delete-dry-run: %v", err))
	}
	records = append(records, deleteDryRunRecord)

	deleteConfirmedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-delete-confirmed", "--automation", "--json", "delete", "pi", "--key", keys[3], "--force", "--workers", "1")
	deleteConfirmedRecord := realStateDestructiveRecord(profile, dataset, deleteConfirmedResult, "delete process-instance", "real-state-destructive-delete-confirmed", "json", []string{"key", "force", "workers"}, []string{keys[3]}, false, true)
	if err := validateVolumeDeleteConfirmed(t, profile, deleteConfirmedResult, []string{keys[3]}); err != nil {
		deleteConfirmedRecord.Outcome = realStateOutcomeFailed
		deleteConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-delete-confirmed: %v", err))
	}
	records = append(records, deleteConfirmedRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func realStateDestructiveRecord(profile integrationProfile, dataset realStateDestructiveDataset, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, keys []string, preview bool, confirmed bool) evidenceRecord {
	outcome := realStateOutcomeLiveCovered
	if preview && !confirmed {
		outcome = realStateOutcomeDryRunCovered
	}
	record := commandEvidence(commandPath, scenarioName, result, outcome)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.ResourceKeys = append([]string(nil), keys...)
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "mutated", "retained", "cleanup_failed"}
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	record.RequiredState = "active process-instance candidates"
	record.ObservedState = "dry-run retained or confirmed post-state verified"
	return record
}
