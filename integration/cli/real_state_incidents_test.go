// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type realStateIncident struct {
	IncidentKey        string `json:"incidentKey,omitempty"`
	ProcessInstanceKey string `json:"processInstanceKey"`
	State              string `json:"state,omitempty"`
	ErrorType          string `json:"errorType,omitempty"`
	ErrorMessage       string `json:"errorMessage,omitempty"`
	ElementId          string `json:"elementId,omitempty"`
	ElementInstanceKey string `json:"elementInstanceKey,omitempty"`
	JobKey             string `json:"jobKey,omitempty"`
}

func TestRealStateIncidentsFamily(t *testing.T) {
	profiles := selectedRealStateC89Profiles(t)

	report := realStateFamilyReport{
		Family:   "incidents",
		Marker:   suite.marker,
		Profiles: profiles,
	}
	var failures []string
	for _, profile := range profiles {
		dataset, records, err := seedRealStateJobDataset(t, profile, 2)
		report.Fixtures = append(report.Fixtures, dataset.Fixture)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}

		records, err = runRealStateIncidentScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeRealStateDataReport(t, "incidents", report.Fixtures)
	writeRealStateProgressReport(t, "incidents", report.Records)
	writeRealStateOpsReportEvidence(t, "incidents", nil)
	writeRealStateFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("real-state incident scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runRealStateIncidentScenarios(t *testing.T, profile integrationProfile, dataset realStateJobDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	jobs := firstNRealStateJobs(dataset.Jobs, 2)
	if len(jobs) < 2 {
		return records, fmt.Errorf("real-state incident dataset for profile %q has %d jobs, want at least 2", profile.Name, len(jobs))
	}

	failResult := runC8VoltForProfile(t, profile.Name, "real-state-incidents-fail-job", "--automation", "--json", "update", "job", "--key", jobs[0].Key, "--fail", "--retries", "0", "--message", "c8volt real-state incident setup", "--no-wait")
	failRecord := realStateJobRecord(profile, failResult, "update job", "real-state-incidents-fail-job", "json", []string{"key", "fail", "retries", "message", "no-wait"}, []string{jobs[0].Key, jobs[0].ProcessInstanceKey}, false, true)
	if err := validateRealStateJobWorkerOutcome(failResult); err != nil {
		failRecord.Outcome = realStateOutcomeFailed
		failRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-incidents-fail-job: %v", err))
	}
	records = append(records, failRecord)

	incidents, result := pollRealStateIncidentsForProcessInstance(t, profile, "real-state-incidents-get-active", jobs[0].ProcessInstanceKey)
	incidentRecord := realStateIncidentRecord(profile, result, "get incident", "real-state-incidents-get-active", "json", []string{"pi-key", "state", "limit", "batch-size"}, []string{jobs[0].ProcessInstanceKey}, false, false)
	incidentRecord.ResourceKeys = append(incidentRecord.ResourceKeys, realStateIncidentKeys(incidents)...)
	if err := validateRealStateIncidentDiscovery(result, incidents, jobs[0]); err != nil {
		incidentRecord.Outcome = realStateOutcomeFailed
		incidentRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-incidents-get-active: %v", err))
	}
	records = append(records, incidentRecord)

	jobEvidenceResult := runC8VoltForProfile(t, profile.Name, "real-state-incidents-related-job", "--automation", "--json", "get", "job", "--key", jobs[0].Key)
	jobEvidenceRecord := realStateJobRecord(profile, jobEvidenceResult, "get job", "real-state-incidents-related-job", "json", []string{"key"}, []string{jobs[0].Key, jobs[0].ProcessInstanceKey}, false, false)
	if err := validateRealStateGetJobJSONByKey(jobEvidenceResult, jobs[0]); err != nil {
		jobEvidenceRecord.Outcome = realStateOutcomeFailed
		jobEvidenceRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-incidents-related-job: %v", err))
	}
	records = append(records, jobEvidenceRecord)

	repairResult := runC8VoltForProfile(t, profile.Name, "real-state-incidents-ops-repair-dry-run", "ops", "repair", "incident", "--key", firstString(realStateIncidentKeys(incidents)), "--retries", "1", "--job-timeout", "30s", "--dry-run")
	repairRecord := realStateIncidentRecord(profile, repairResult, "ops repair incident", "real-state-incidents-ops-repair-dry-run", "one-line", []string{"key", "retries", "job-timeout", "dry-run"}, append(realStateIncidentKeys(incidents), jobs[0].Key), true, false)
	if err := validateRealStateIncidentRepairDryRun(repairResult); err != nil {
		repairRecord.Outcome = realStateOutcomeFailed
		repairRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-incidents-ops-repair-dry-run: %v", err))
	}
	records = append(records, repairRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func pollRealStateIncidentsForProcessInstance(t *testing.T, profile integrationProfile, scenario string, piKey string) ([]realStateIncident, commandResult) {
	t.Helper()
	var result commandResult
	for attempt := 1; attempt <= 10; attempt++ {
		attemptScenario := scenario
		if attempt > 1 {
			attemptScenario = fmt.Sprintf("%s-attempt-%d", scenario, attempt)
		}
		result = runC8VoltForProfile(t, profile.Name, attemptScenario, "--automation", "--json", "get", "incident", "--pi-key", piKey, "--state", "active", "--limit", "5", "--batch-size", "1")
		if result.Err == nil {
			incidents := realStateIncidentsFromResult(t, result, piKey)
			if len(incidents) > 0 {
				return incidents, result
			}
		}
		time.Sleep(2 * time.Second)
	}
	return realStateIncidentsFromResult(t, result, piKey), result
}

func realStateIncidentsFromResult(t *testing.T, result commandResult, piKey string) []realStateIncident {
	t.Helper()
	var payload struct {
		Items []realStateIncident `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return nil
	}
	var incidents []realStateIncident
	for _, item := range payload.Items {
		if item.ProcessInstanceKey == piKey && item.IncidentKey != "" {
			incidents = append(incidents, item)
		}
	}
	return incidents
}

func validateRealStateIncidentDiscovery(result commandResult, incidents []realStateIncident, job realStateJob) error {
	if err := requireVolumeCommandSuccess(result, "get incident real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "get incident real-state"); err != nil {
		return err
	}
	if len(incidents) == 0 {
		return fmt.Errorf("no active incident discovered for process instance %s", job.ProcessInstanceKey)
	}
	for _, incident := range incidents {
		if incident.JobKey == job.Key {
			return nil
		}
	}
	return fmt.Errorf("active incidents for process instance %s do not reference job %s: %q", job.ProcessInstanceKey, job.Key, compactLogSnippet(result.Stdout, 300))
}

func validateRealStateGetJobJSONByKey(result commandResult, want realStateJob) error {
	if err := requireVolumeCommandSuccess(result, "get job by key real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "get job by key real-state"); err != nil {
		return err
	}
	var payload realStateJob
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode get job by key payload: %w", err)
	}
	if payload.Key != want.Key {
		return fmt.Errorf("get job by key returned %q, want %q", payload.Key, want.Key)
	}
	if payload.ProcessInstanceKey != want.ProcessInstanceKey {
		return fmt.Errorf("get job by key processInstanceKey=%q, want %q", payload.ProcessInstanceKey, want.ProcessInstanceKey)
	}
	return nil
}

func validateRealStateIncidentRepairDryRun(result commandResult) error {
	if err := requireVolumeCommandSuccess(result, "ops repair incident dry-run real-state"); err != nil {
		return err
	}
	return requireHumanContains(volumeHumanOutput(result), "dry run: repair incidents", "candidate incidents:", "repair preview:", "outcome: planned")
}

func realStateIncidentRecord(profile integrationProfile, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, keys []string, preview bool, confirmed bool) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, realStateOutcomeLiveCovered)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.ResourceKeys = append([]string(nil), keys...)
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "mutated", "retained"}
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	return record
}

func realStateIncidentKeys(incidents []realStateIncident) []string {
	keys := make([]string, 0, len(incidents))
	for _, incident := range incidents {
		if incident.IncidentKey != "" {
			keys = append(keys, incident.IncidentKey)
		}
	}
	return uniqueSortedStrings(keys)
}
