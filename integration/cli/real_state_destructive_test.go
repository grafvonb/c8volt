// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type realStateDestructiveDataset struct {
	Fixture   realStateFixture      `json:"fixture"`
	Volume    volumeDataset         `json:"volumeDataset"`
	Incident  volumeIncidentDataset `json:"incidentDataset"`
	Incidents []realStateIncident   `json:"incidents,omitempty"`
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

	incidentDataset, incidentRecords, err := seedVolumeIncidentDataset(t, profile, 8)
	records = append(records, incidentRecords...)
	fixture.ProcessInstanceKeys = append(fixture.ProcessInstanceKeys, incidentDataset.ProcessInstanceKeys...)
	fixture.ProcessDefinitionKeys = append(fixture.ProcessDefinitionKeys, incidentDataset.ProcessDefinitionKeys...)
	if err != nil {
		return realStateDestructiveDataset{Fixture: fixture, Volume: dataset, Incident: incidentDataset}, records, err
	}
	incidents, discoveryRecords, err := discoverRealStateDestructiveIncidents(t, profile, incidentDataset.ProcessInstanceKeys)
	records = append(records, discoveryRecords...)
	incidentDataset.IncidentKeys = realStateIncidentKeys(incidents)
	fixture.IncidentKeys = append([]string(nil), incidentDataset.IncidentKeys...)
	if err != nil {
		return realStateDestructiveDataset{Fixture: fixture, Volume: dataset, Incident: incidentDataset, Incidents: incidents}, records, err
	}
	return realStateDestructiveDataset{Fixture: fixture, Volume: dataset, Incident: incidentDataset, Incidents: incidents}, records, nil
}

func runRealStateDestructiveScenarios(t *testing.T, profile integrationProfile, dataset realStateDestructiveDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	keys := firstNStrings(dataset.Fixture.ProcessInstanceKeys, 4)
	if len(keys) < 4 {
		return records, fmt.Errorf("real-state destructive dataset for profile %q has %d active keys, want at least 4", profile.Name, len(keys))
	}
	incidents := firstNRealStateIncidents(dataset.Incidents, 8)
	if len(incidents) < 8 {
		return records, fmt.Errorf("real-state destructive incident dataset for profile %q has %d incidents, want at least 8", profile.Name, len(incidents))
	}
	missingPIKey := realStateMissingProcessInstanceKey()
	missingIncidentKey := realStateMissingIncidentKey()

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

	resolveDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-resolve-incident-dry-run", "--automation", "--json", "resolve", "incident", "--key", incidents[0].IncidentKey, "--dry-run", "--workers", "1")
	resolveDryRunRecord := realStateDestructiveRecord(profile, dataset, resolveDryRunResult, "resolve incident", "real-state-destructive-resolve-incident-dry-run", "json", []string{"key", "dry-run", "workers"}, []string{incidents[0].IncidentKey, incidents[0].ProcessInstanceKey}, true, false)
	if err := validateRealStateIncidentResolveDryRun(t, profile, resolveDryRunResult, incidents[0]); err != nil {
		resolveDryRunRecord.Outcome = realStateOutcomeFailed
		resolveDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-resolve-incident-dry-run: %v", err))
	}
	records = append(records, resolveDryRunRecord)

	resolveConfirmedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-resolve-pi-confirmed", "--automation", "--json", "resolve", "pi", "--key", incidents[1].ProcessInstanceKey, "--workers", "1")
	resolveConfirmedRecord := realStateDestructiveRecord(profile, dataset, resolveConfirmedResult, "resolve process-instance", "real-state-destructive-resolve-pi-confirmed", "json", []string{"key", "workers"}, []string{incidents[1].IncidentKey, incidents[1].ProcessInstanceKey}, false, true)
	if err := validateRealStateProcessInstanceResolveConfirmed(t, profile, resolveConfirmedResult, incidents[1]); err != nil {
		resolveConfirmedRecord.Outcome = realStateOutcomeFailed
		resolveConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-resolve-pi-confirmed: %v", err))
	}
	records = append(records, resolveConfirmedRecord)

	repairDryRunReport := volumeOpsRepairReportPath(t, "real-state-destructive-repair-incident-dry-run", profile, "json")
	repairDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-repair-incident-dry-run", "ops", "repair", "incident", "--key", incidents[2].IncidentKey, "--vars", `{"hasIncident":false}`, "--retries", "1", "--job-timeout", "30s", "--workers", "1", "--dry-run", "--report-file", repairDryRunReport, "--report-format", "json")
	repairDryRunRecord := realStateDestructiveRecord(profile, dataset, repairDryRunResult, "ops repair incident", "real-state-destructive-repair-incident-dry-run", "one-line", []string{"key", "vars", "retries", "job-timeout", "workers", "dry-run", "report-file", "report-format"}, []string{incidents[2].IncidentKey, incidents[2].ProcessInstanceKey}, true, false)
	if err := validateRealStateOpsRepairIncidentDryRun(t, profile, repairDryRunResult, repairDryRunReport, incidents[2]); err != nil {
		repairDryRunRecord.Outcome = realStateOutcomeFailed
		repairDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-repair-incident-dry-run: %v", err))
	}
	records = append(records, repairDryRunRecord)

	repairConfirmedReport := volumeOpsRepairReportPath(t, "real-state-destructive-repair-pi-confirmed", profile, "json")
	repairConfirmedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-repair-pi-confirmed", "--automation", "--json", "ops", "repair", "process-instance", "--key", incidents[3].ProcessInstanceKey, "--vars", `{"hasIncident":false}`, "--retries", "1", "--job-timeout", "30s", "--workers", "1", "--report-file", repairConfirmedReport, "--report-format", "json")
	repairConfirmedRecord := realStateDestructiveRecord(profile, dataset, repairConfirmedResult, "ops repair process-instance", "real-state-destructive-repair-pi-confirmed", "json", []string{"automation", "json", "key", "vars", "retries", "job-timeout", "workers", "report-file", "report-format"}, []string{incidents[3].IncidentKey, incidents[3].ProcessInstanceKey}, false, true)
	if err := validateRealStateOpsRepairConfirmed(t, profile, repairConfirmedResult, repairConfirmedReport, incidents[3], "process-instance"); err != nil {
		repairConfirmedRecord.Outcome = realStateOutcomeFailed
		repairConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-repair-pi-confirmed: %v", err))
	}
	records = append(records, repairConfirmedRecord)

	purgeDryRunReport := volumeOpsPurgeReportPath(t, "real-state-destructive-purge-incident-dry-run", profile, "json")
	purgeDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-purge-incident-dry-run", "ops", "purge", "process-instances-with-incidents", "--inc-key", incidents[4].IncidentKey, "--batch-size", "1", "--limit", "1", "--workers", "1", "--dry-run", "--report-file", purgeDryRunReport, "--report-format", "json")
	purgeDryRunRecord := realStateDestructiveRecord(profile, dataset, purgeDryRunResult, "ops purge process-instances-with-incidents", "real-state-destructive-purge-incident-dry-run", "one-line", []string{"inc-key", "batch-size", "limit", "workers", "dry-run", "report-file", "report-format"}, []string{incidents[4].IncidentKey, incidents[4].ProcessInstanceKey}, true, false)
	if err := validateRealStateOpsPurgeIncidentDryRun(t, profile, purgeDryRunResult, purgeDryRunReport, incidents[4]); err != nil {
		purgeDryRunRecord.Outcome = realStateOutcomeFailed
		purgeDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-purge-incident-dry-run: %v", err))
	}
	records = append(records, purgeDryRunRecord)

	purgeConfirmedReport := volumeOpsPurgeReportPath(t, "real-state-destructive-purge-incident-confirmed", profile, "json")
	purgeConfirmedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-purge-incident-confirmed", "--automation", "--json", "ops", "purge", "process-instances-with-incidents", "--inc-key", incidents[5].IncidentKey, "--batch-size", "1", "--limit", "1", "--workers", "1", "--force", "--report-file", purgeConfirmedReport, "--report-format", "json")
	purgeConfirmedRecord := realStateDestructiveRecord(profile, dataset, purgeConfirmedResult, "ops purge process-instances-with-incidents", "real-state-destructive-purge-incident-confirmed", "json", []string{"automation", "json", "inc-key", "batch-size", "limit", "workers", "force", "report-file", "report-format"}, []string{incidents[5].IncidentKey, incidents[5].ProcessInstanceKey}, false, true)
	if err := validateRealStateOpsPurgeIncidentConfirmed(t, profile, purgeConfirmedResult, purgeConfirmedReport, incidents[5]); err != nil {
		purgeConfirmedRecord.Outcome = realStateOutcomeFailed
		purgeConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-purge-incident-confirmed: %v", err))
	}
	records = append(records, purgeConfirmedRecord)

	malformedMixedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-delete-malformed-mixed", "--automation", "--json", "delete", "pi", "--key", keys[0], "--key", "not-a-key", "--force", "--dry-run", "--workers", "1")
	malformedMixedRecord := realStateDestructiveRecord(profile, dataset, malformedMixedResult, "delete process-instance", "real-state-destructive-delete-malformed-mixed", "json", []string{"key", "force", "dry-run", "workers"}, []string{keys[0], "not-a-key"}, true, false)
	if err := validateRealStateMalformedMixedRejected(t, profile, malformedMixedResult, keys[0]); err != nil {
		malformedMixedRecord.Outcome = realStateOutcomeFailed
		malformedMixedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-delete-malformed-mixed: %v", err))
	}
	records = append(records, malformedMixedRecord)

	missingMixedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-cancel-missing-mixed", "--automation", "--json", "cancel", "pi", "--key", keys[2], "--key", missingPIKey, "--dry-run", "--workers", "1")
	missingMixedRecord := realStateDestructiveRecord(profile, dataset, missingMixedResult, "cancel process-instance", "real-state-destructive-cancel-missing-mixed", "json", []string{"key", "dry-run", "workers"}, []string{keys[2], missingPIKey}, true, false)
	if err := validateRealStateMissingMixedRejected(t, profile, missingMixedResult, keys[2], missingPIKey); err != nil {
		missingMixedRecord.Outcome = realStateOutcomeFailed
		missingMixedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-cancel-missing-mixed: %v", err))
	}
	records = append(records, missingMixedRecord)

	alreadyTerminatedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-delete-already-terminated-dry-run", "--automation", "--json", "delete", "pi", "--key", keys[1], "--force", "--dry-run", "--workers", "1")
	alreadyTerminatedRecord := realStateDestructiveRecord(profile, dataset, alreadyTerminatedResult, "delete process-instance", "real-state-destructive-delete-already-terminated-dry-run", "json", []string{"key", "force", "dry-run", "workers"}, []string{keys[1]}, true, false)
	if err := validateRealStateAlreadyTerminatedDryRun(t, profile, alreadyTerminatedResult, keys[1]); err != nil {
		alreadyTerminatedRecord.Outcome = realStateOutcomeFailed
		alreadyTerminatedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-delete-already-terminated-dry-run: %v", err))
	}
	records = append(records, alreadyTerminatedRecord)

	staleDeletedResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-delete-stale-deleted", "--automation", "--json", "delete", "pi", "--key", keys[3], "--force", "--dry-run", "--workers", "1")
	staleDeletedRecord := realStateDestructiveRecord(profile, dataset, staleDeletedResult, "delete process-instance", "real-state-destructive-delete-stale-deleted", "json", []string{"key", "force", "dry-run", "workers"}, []string{keys[3]}, true, false)
	if err := validateRealStateStaleDeletedRejected(staleDeletedResult, keys[3]); err != nil {
		staleDeletedRecord.Outcome = realStateOutcomeFailed
		staleDeletedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-delete-stale-deleted: %v", err))
	}
	records = append(records, staleDeletedRecord)

	partialResolveResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-resolve-incident-partial-no-wait", "--automation", "--json", "resolve", "incident", "--key", incidents[6].IncidentKey, "--key", missingIncidentKey, "--no-wait", "--workers", "1")
	partialResolveRecord := realStateDestructiveRecord(profile, dataset, partialResolveResult, "resolve incident", "real-state-destructive-resolve-incident-partial-no-wait", "json", []string{"key", "no-wait", "workers"}, []string{incidents[6].IncidentKey, missingIncidentKey}, false, true)
	if err := validateRealStateResolveIncidentPartialNoWait(partialResolveResult, incidents[6].IncidentKey, missingIncidentKey); err != nil {
		partialResolveRecord.Outcome = realStateOutcomeFailed
		partialResolveRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-resolve-incident-partial-no-wait: %v", err))
	}
	records = append(records, partialResolveRecord)

	failFastResolveResult := runC8VoltForProfile(t, profile.Name, "real-state-destructive-resolve-incident-fail-fast", "--automation", "--json", "resolve", "incident", "--key", missingIncidentKey, "--key", incidents[7].IncidentKey, "--no-wait", "--workers", "1", "--fail-fast")
	failFastResolveRecord := realStateDestructiveRecord(profile, dataset, failFastResolveResult, "resolve incident", "real-state-destructive-resolve-incident-fail-fast", "json", []string{"key", "no-wait", "workers", "fail-fast"}, []string{missingIncidentKey, incidents[7].IncidentKey}, false, true)
	if err := validateRealStateResolveIncidentFailFast(t, profile, failFastResolveResult, incidents[7], missingIncidentKey); err != nil {
		failFastResolveRecord.Outcome = realStateOutcomeFailed
		failFastResolveRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-destructive-resolve-incident-fail-fast: %v", err))
	}
	records = append(records, failFastResolveRecord)

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

func discoverRealStateDestructiveIncidents(t *testing.T, profile integrationProfile, piKeys []string) ([]realStateIncident, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var incidents []realStateIncident
	var failures []string
	for i, piKey := range piKeys {
		scenario := fmt.Sprintf("real-state-destructive-get-incident-%d", i+1)
		found, result := pollRealStateIncidentsForProcessInstance(t, profile, scenario, piKey)
		record := realStateIncidentRecord(profile, result, "get incident", scenario, "json", []string{"pi-key", "state", "limit", "batch-size"}, []string{piKey}, false, false)
		record.ResourceKeys = append(record.ResourceKeys, realStateIncidentKeys(found)...)
		if err := validateRealStateDestructiveIncidentDiscovery(result, found, piKey); err != nil {
			record.Outcome = realStateOutcomeFailed
			record.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("%s: %v", scenario, err))
		}
		records = append(records, record)
		incidents = append(incidents, found...)
	}
	if len(failures) > 0 {
		return incidents, records, errors.New(strings.Join(failures, "\n"))
	}
	return incidents, records, nil
}

func validateRealStateDestructiveIncidentDiscovery(result commandResult, incidents []realStateIncident, piKey string) error {
	if err := requireVolumeCommandSuccess(result, "get incident destructive real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "get incident destructive real-state"); err != nil {
		return err
	}
	if len(incidents) == 0 {
		return fmt.Errorf("no active incident discovered for process instance %s", piKey)
	}
	return nil
}

func validateRealStateIncidentResolveDryRun(t *testing.T, profile integrationProfile, result commandResult, incident realStateIncident) error {
	t.Helper()
	if err := validateVolumeIncidentResolveDryRun(result, []string{incident.IncidentKey}); err != nil {
		return err
	}
	return requireRealStateActiveIncidentEventually(t, profile, incident.ProcessInstanceKey, true, 30*time.Second)
}

func validateRealStateProcessInstanceResolveConfirmed(t *testing.T, profile integrationProfile, result commandResult, incident realStateIncident) error {
	t.Helper()
	if err := validateRealStateProcessInstanceResolveConfirmedPayload(result, incident.ProcessInstanceKey); err != nil {
		return err
	}
	return requireRealStateActiveIncidentEventually(t, profile, incident.ProcessInstanceKey, true, 30*time.Second)
}

func validateRealStateProcessInstanceResolveConfirmedPayload(result commandResult, piKey string) error {
	if err := requireVolumeCommandSuccess(result, "resolve pi confirmed real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "resolve pi confirmed real-state"); err != nil {
		return err
	}
	if err := requireVolumeEnvelopeOutcome(result.Stdout, "succeeded"); err != nil {
		return err
	}
	var payload struct {
		MutationSubmitted bool `json:"mutationSubmitted"`
		Items             []struct {
			ProcessInstanceKey string `json:"processInstanceKey"`
			Status             string `json:"status"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve pi confirmed payload: %w", err)
	}
	if !payload.MutationSubmitted {
		return fmt.Errorf("resolve pi confirmed mutationSubmitted=false")
	}
	for _, item := range payload.Items {
		if item.ProcessInstanceKey == piKey && item.Status == "confirmed" {
			return nil
		}
	}
	return fmt.Errorf("resolve pi confirmed missing confirmed key %s: %q", piKey, compactLogSnippet(result.Stdout, 300))
}

func validateRealStateOpsRepairIncidentDryRun(t *testing.T, profile integrationProfile, result commandResult, reportPath string, incident realStateIncident) error {
	t.Helper()
	if err := validateVolumeOpsRepairIncidentDryRun(result, reportPath); err != nil {
		return err
	}
	return requireRealStateActiveIncidentEventually(t, profile, incident.ProcessInstanceKey, true, 30*time.Second)
}

func validateRealStateOpsRepairConfirmed(t *testing.T, profile integrationProfile, result commandResult, reportPath string, incident realStateIncident, label string) error {
	t.Helper()
	if err := requireVolumeCommandSuccess(result, "ops repair "+label+" confirmed real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "ops repair "+label+" confirmed real-state"); err != nil {
		return err
	}
	if err := requireVolumeEnvelopeOutcome(result.Stdout, "succeeded"); err != nil {
		return err
	}
	var payload struct {
		Outcome string `json:"outcome"`
		Request struct {
			NoWait bool `json:"noWait"`
		} `json:"request"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode repair %s confirmed payload: %w", label, err)
	}
	if payload.Outcome != "repaired" {
		return fmt.Errorf("repair %s payload outcome=%q, want repaired", label, payload.Outcome)
	}
	if payload.Request.NoWait {
		return fmt.Errorf("repair %s payload noWait=true, want false", label)
	}
	report, err := readVolumeOpsRepairReport(reportPath)
	if err != nil {
		return err
	}
	if report.Outcome != payload.Outcome {
		return fmt.Errorf("repair %s stdout/report outcome mismatch: %q/%q", label, payload.Outcome, report.Outcome)
	}
	return requireRealStateActiveIncidentEventually(t, profile, incident.ProcessInstanceKey, false, 60*time.Second)
}

func validateRealStateOpsPurgeIncidentDryRun(t *testing.T, profile integrationProfile, result commandResult, reportPath string, incident realStateIncident) error {
	t.Helper()
	if err := validateVolumeOpsPurgeIncidentDryRun(result, reportPath); err != nil {
		return err
	}
	return requireRealStateActiveIncidentEventually(t, profile, incident.ProcessInstanceKey, true, 30*time.Second)
}

func validateRealStateOpsPurgeIncidentConfirmed(t *testing.T, profile integrationProfile, result commandResult, reportPath string, incident realStateIncident) error {
	t.Helper()
	if err := requireVolumeCommandSuccess(result, "ops purge process-instances-with-incidents confirmed real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "ops purge process-instances-with-incidents confirmed real-state"); err != nil {
		return err
	}
	if err := requireVolumeEnvelopeOutcome(result.Stdout, "succeeded"); err != nil {
		return err
	}
	var payload struct {
		Outcome   string `json:"outcome"`
		Discovery struct {
			IncidentKeys                 []string `json:"incidentKeys"`
			CandidateProcessInstanceKeys []string `json:"candidateProcessInstanceKeys"`
		} `json:"discovery"`
		Deletion struct {
			Submitted bool `json:"submitted"`
			Confirmed bool `json:"confirmed"`
		} `json:"deletion"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode incident purge confirmed payload: %w", err)
	}
	report, err := readRealStateOpsPurgeIncidentReport(reportPath)
	if err != nil {
		return err
	}
	if payload.Outcome != "deleted" || report.Outcome != "deleted" {
		return fmt.Errorf("incident purge outcome stdout/report=%q/%q, want deleted/deleted", payload.Outcome, report.Outcome)
	}
	if !containsString(payload.Discovery.IncidentKeys, incident.IncidentKey) || !containsString(report.Discovery.IncidentKeys, incident.IncidentKey) {
		return fmt.Errorf("incident purge did not freeze requested incident key %s: stdout=%v report=%v", incident.IncidentKey, payload.Discovery.IncidentKeys, report.Discovery.IncidentKeys)
	}
	if !containsString(payload.Discovery.CandidateProcessInstanceKeys, incident.ProcessInstanceKey) || !containsString(report.Discovery.CandidateProcessInstanceKeys, incident.ProcessInstanceKey) {
		return fmt.Errorf("incident purge did not freeze requested process-instance key %s: stdout=%v report=%v", incident.ProcessInstanceKey, payload.Discovery.CandidateProcessInstanceKeys, report.Discovery.CandidateProcessInstanceKeys)
	}
	if !payload.Deletion.Submitted || !report.Deletion.Submitted {
		return fmt.Errorf("incident purge did not submit deletion: stdout=%t report=%t", payload.Deletion.Submitted, report.Deletion.Submitted)
	}
	if !payload.Deletion.Confirmed || !report.Deletion.Confirmed {
		return fmt.Errorf("incident purge did not confirm deletion: stdout=%t report=%t", payload.Deletion.Confirmed, report.Deletion.Confirmed)
	}
	return requireProcessInstanceAbsentEventually(t, profile, incident.ProcessInstanceKey, 60*time.Second)
}

func validateRealStateMalformedMixedRejected(t *testing.T, profile integrationProfile, result commandResult, validKey string) error {
	t.Helper()
	if err := validateRealStateErrorEnvelope(result, "invalid", "invalid_input", "not-a-key"); err != nil {
		return err
	}
	return requireProcessInstanceState(t, profile, validKey, "ACTIVE")
}

func validateRealStateMissingMixedRejected(t *testing.T, profile integrationProfile, result commandResult, validKey string, missingKey string) error {
	t.Helper()
	if err := validateRealStateErrorEnvelope(result, "failed", "not_found", missingKey); err != nil {
		return err
	}
	return requireProcessInstanceState(t, profile, validKey, "ACTIVE")
}

func validateRealStateAlreadyTerminatedDryRun(t *testing.T, profile integrationProfile, result commandResult, key string) error {
	t.Helper()
	if err := validateVolumeDeleteDryRunPayload(result); err != nil {
		return err
	}
	var payload struct {
		MutationSubmitted  bool `json:"mutationSubmitted"`
		SelectedFinalState []struct {
			Key   string `json:"key"`
			State string `json:"state"`
		} `json:"selectedFinalState"`
		SelectedFinalStateCount int `json:"selectedFinalStateCount"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode already-terminated delete dry-run payload: %w", err)
	}
	if payload.MutationSubmitted {
		return fmt.Errorf("already-terminated delete dry-run submitted mutation")
	}
	if payload.SelectedFinalStateCount < 1 {
		return fmt.Errorf("already-terminated delete dry-run selectedFinalStateCount=%d, want >=1", payload.SelectedFinalStateCount)
	}
	for _, item := range payload.SelectedFinalState {
		if item.Key == key && item.State == "TERMINATED" {
			return requireProcessInstanceState(t, profile, key, "TERMINATED")
		}
	}
	return fmt.Errorf("already-terminated delete dry-run missing final-state key %s: %q", key, compactLogSnippet(result.Stdout, 300))
}

func validateRealStateStaleDeletedRejected(result commandResult, key string) error {
	return validateRealStateErrorEnvelope(result, "failed", "not_found", key)
}

func validateRealStateResolveIncidentPartialNoWait(result commandResult, validIncidentKey string, missingIncidentKey string) error {
	if err := validateRealStateResolveIncidentPartialEnvelope(result, validIncidentKey, missingIncidentKey, 1, 1); err != nil {
		return err
	}
	return nil
}

func validateRealStateResolveIncidentFailFast(t *testing.T, profile integrationProfile, result commandResult, unscheduledIncident realStateIncident, missingIncidentKey string) error {
	t.Helper()
	if err := validateRealStateResolveIncidentPartialEnvelope(result, "", missingIncidentKey, 0, 1); err != nil {
		return err
	}
	if strings.Contains(result.Stdout, unscheduledIncident.IncidentKey) {
		return fmt.Errorf("fail-fast output contains unscheduled incident key %s", unscheduledIncident.IncidentKey)
	}
	return requireRealStateActiveIncidentEventually(t, profile, unscheduledIncident.ProcessInstanceKey, true, 30*time.Second)
}

func validateRealStateResolveIncidentPartialEnvelope(result commandResult, validIncidentKey string, missingIncidentKey string, wantSubmitted int, wantFailed int) error {
	if result.Err == nil {
		return fmt.Errorf("resolve incident partial scenario unexpectedly succeeded")
	}
	if err := requireRealStateJSONStdoutClean(result, "resolve incident partial real-state"); err != nil {
		return err
	}
	if err := requireVolumeEnvelopeOutcome(result.Stdout, "accepted"); err != nil {
		return err
	}
	var payload struct {
		Submitted         int  `json:"submitted"`
		Failed            int  `json:"failed"`
		MutationSubmitted bool `json:"mutationSubmitted"`
		Items             []struct {
			IncidentKey       string `json:"incidentKey"`
			Status            string `json:"status"`
			MutationSubmitted bool   `json:"mutationSubmitted"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve incident partial payload: %w", err)
	}
	if payload.Submitted != wantSubmitted || payload.Failed != wantFailed {
		return fmt.Errorf("resolve incident partial submitted/failed=%d/%d, want %d/%d", payload.Submitted, payload.Failed, wantSubmitted, wantFailed)
	}
	if wantSubmitted > 0 && !payload.MutationSubmitted {
		return fmt.Errorf("resolve incident partial mutationSubmitted=false, want true")
	}
	if validIncidentKey != "" && !resolveIncidentItemMatches(payload.Items, validIncidentKey, "submitted", true) {
		return fmt.Errorf("resolve incident partial missing submitted valid incident %s: %q", validIncidentKey, compactLogSnippet(result.Stdout, 300))
	}
	if !resolveIncidentItemMatches(payload.Items, missingIncidentKey, "mutation_failed", false) {
		return fmt.Errorf("resolve incident partial missing failed incident %s: %q", missingIncidentKey, compactLogSnippet(result.Stdout, 300))
	}
	return nil
}

func readRealStateOpsPurgeIncidentReport(path string) (realStateOpsPurgeIncidentReport, error) {
	var report realStateOpsPurgeIncidentReport
	if err := readVolumeOpsPurgeJSONReport(path, &report); err != nil {
		return report, err
	}
	return report, nil
}

type realStateOpsPurgeIncidentReport struct {
	Outcome   string `json:"outcome"`
	Discovery struct {
		IncidentKeys                 []string `json:"incidentKeys"`
		CandidateProcessInstanceKeys []string `json:"candidateProcessInstanceKeys"`
	} `json:"discovery"`
	Deletion struct {
		Submitted bool `json:"submitted"`
		Confirmed bool `json:"confirmed"`
	} `json:"deletion"`
}

func validateRealStateErrorEnvelope(result commandResult, wantOutcome string, wantClass string, wantMessageContains string) error {
	if result.Err == nil {
		return fmt.Errorf("expected command failure")
	}
	if err := requireRealStateJSONStdoutClean(result, "real-state expected error"); err != nil {
		return err
	}
	var envelope struct {
		Outcome string `json:"outcome"`
		Class   string `json:"class"`
		Detail  struct {
			Message string `json:"message"`
			Class   string `json:"class"`
		} `json:"detail"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stdout)), &envelope); err != nil {
		return fmt.Errorf("decode error envelope: %w", err)
	}
	if envelope.Outcome != wantOutcome || envelope.Class != wantClass || envelope.Detail.Class != wantClass {
		return fmt.Errorf("error envelope outcome/class/detail=%q/%q/%q, want %q/%q/%q", envelope.Outcome, envelope.Class, envelope.Detail.Class, wantOutcome, wantClass, wantClass)
	}
	if !strings.Contains(envelope.Detail.Message, wantMessageContains) {
		return fmt.Errorf("error envelope message does not contain %q: %q", wantMessageContains, envelope.Detail.Message)
	}
	return nil
}

func resolveIncidentItemMatches(items []struct {
	IncidentKey       string `json:"incidentKey"`
	Status            string `json:"status"`
	MutationSubmitted bool   `json:"mutationSubmitted"`
}, key string, status string, mutationSubmitted bool) bool {
	for _, item := range items {
		if item.IncidentKey == key && item.Status == status && item.MutationSubmitted == mutationSubmitted {
			return true
		}
	}
	return false
}

func requireRealStateActiveIncidentEventually(t *testing.T, profile integrationProfile, piKey string, wantIncident bool, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for attempt := 1; ; attempt++ {
		scenario := fmt.Sprintf("real-state-destructive-active-incident-%s-attempt-%d", piKey, attempt)
		result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "get", "incident", "--pi-key", piKey, "--state", "active", "--limit", "5", "--batch-size", "1")
		if result.Err != nil {
			last = fmt.Errorf("get active incidents failed: %w; %s", result.Err, compactLogSnippet(result.Stdout+"\n"+result.Stderr, 300))
		} else if err := requireRealStateJSONStdoutClean(result, "get active incident post-state"); err != nil {
			last = err
		} else {
			count := len(realStateIncidentsFromResult(t, result, piKey))
			if wantIncident && count > 0 {
				return nil
			}
			if !wantIncident && count == 0 {
				return nil
			}
			last = fmt.Errorf("active incident count for process instance %s = %d, want incident=%t", piKey, count, wantIncident)
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(2 * time.Second)
	}
}

func requireProcessInstanceAbsentEventually(t *testing.T, profile integrationProfile, key string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for {
		last = requireProcessInstanceAbsent(t, profile, key)
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(2 * time.Second)
	}
}

func realStateMissingProcessInstanceKey() string {
	return "1000000000000001"
}

func realStateMissingIncidentKey() string {
	return "1000000000000002"
}

func firstNRealStateIncidents(values []realStateIncident, count int) []realStateIncident {
	if len(values) < count {
		count = len(values)
	}
	return append([]realStateIncident(nil), values[:count]...)
}
