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

func TestRealStateBPMNErrorFamily(t *testing.T) {
	profiles := selectedRealStateC89Profiles(t)

	report := realStateFamilyReport{
		Family:   "bpmn-error",
		Marker:   suite.marker,
		Profiles: profiles,
	}
	var failures []string
	for _, profile := range profiles {
		dataset, records, err := seedRealStateJobDataset(t, profile, 1)
		if dataset.Fixture.FixtureKind != "" {
			dataset.Fixture.RequiredState = "BPMN error-capable job with catchable error path"
			dataset.Fixture.CurrentEvidenceLevel = realStateOutcomeDryRunCovered
			dataset.Fixture.TargetRealStateProof = "update job --throw-bpmn-error drives observable BPMN error process state"
			dataset.Fixture.ObservedState = "dry-run plan uses a real suite-owned job; confirmed mutation prerequisite is missing"
			dataset.Fixture.SkipReason = "missing embedded BPMN error fixture and c8volt activated/catchable job setup"
		}
		report.Fixtures = append(report.Fixtures, dataset.Fixture)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}

		records, err = runRealStateBPMNErrorScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeRealStateDataReport(t, "bpmn-error", report.Fixtures)
	writeRealStateProgressReport(t, "bpmn-error", report.Records)
	writeRealStateOpsReportEvidence(t, "bpmn-error", nil)
	writeRealStateFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("real-state BPMN error scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runRealStateBPMNErrorScenarios(t *testing.T, profile integrationProfile, dataset realStateJobDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	jobs := firstNRealStateJobs(dataset.Jobs, 1)
	if len(jobs) < 1 {
		return records, fmt.Errorf("real-state BPMN error dataset for profile %q has no job", profile.Name)
	}

	dryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-bpmn-error-update-dry-run", "--automation", "--json", "update", "job", "--key", jobs[0].Key, "--throw-bpmn-error", "ErrorTimerCode", "--message", "c8volt real-state BPMN error dry run", "--vars", `{"c8voltBPMNError":true}`, "--dry-run")
	dryRunRecord := realStateBPMNErrorRecord(profile, dryRunResult, "update job", "real-state-bpmn-error-update-dry-run", "json", []string{"key", "throw-bpmn-error", "message", "vars", "dry-run"}, []string{jobs[0].Key, jobs[0].ProcessInstanceKey}, true, false)
	if err := validateRealStateBPMNErrorDryRun(dryRunResult); err != nil {
		dryRunRecord.Outcome = realStateOutcomeFailed
		dryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-bpmn-error-update-dry-run: %v", err))
	}
	records = append(records, dryRunRecord)

	afterResult := runC8VoltForProfile(t, profile.Name, "real-state-bpmn-error-get-job-after-dry-run", "--automation", "--json", "get", "job", "--key", jobs[0].Key)
	afterRecord := realStateBPMNErrorRecord(profile, afterResult, "get job", "real-state-bpmn-error-get-job-after-dry-run", "json", []string{"key"}, []string{jobs[0].Key, jobs[0].ProcessInstanceKey}, false, false)
	if err := validateRealStateGetJobJSONByKey(afterResult, jobs[0]); err != nil {
		afterRecord.Outcome = realStateOutcomeFailed
		afterRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-bpmn-error-get-job-after-dry-run: %v", err))
	}
	records = append(records, afterRecord)
	records = append(records, realStateBPMNErrorConfirmedSkippedRecord(profile, jobs[0]))

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func validateRealStateBPMNErrorDryRun(result commandResult) error {
	if err := requireVolumeCommandSuccess(result, "update job BPMN error dry-run real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "update job BPMN error dry-run real-state"); err != nil {
		return err
	}
	var payload struct {
		Mode              string `json:"mode"`
		ErrorCode         string `json:"errorCode"`
		DryRun            bool   `json:"dryRun"`
		MutationSubmitted bool   `json:"mutationSubmitted"`
		MaterialChange    bool   `json:"materialChange"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode BPMN error dry-run payload: %w", err)
	}
	if payload.Mode != "bpmn_error" {
		return fmt.Errorf("BPMN error dry-run mode=%q, want bpmn_error", payload.Mode)
	}
	if payload.ErrorCode != "ErrorTimerCode" {
		return fmt.Errorf("BPMN error dry-run errorCode=%q, want ErrorTimerCode", payload.ErrorCode)
	}
	if !payload.DryRun {
		return fmt.Errorf("BPMN error dry-run dryRun=false")
	}
	if payload.MutationSubmitted {
		return fmt.Errorf("BPMN error dry-run mutationSubmitted=true")
	}
	if !payload.MaterialChange {
		return fmt.Errorf("BPMN error dry-run materialChange=false")
	}
	return nil
}

func realStateBPMNErrorRecord(profile integrationProfile, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, keys []string, preview bool, confirmed bool) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, realStateOutcomeLiveCovered)
	if preview && !confirmed {
		record.Outcome = realStateOutcomeDryRunCovered
	}
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.ResourceKeys = append([]string(nil), keys...)
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "retained"}
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	return record
}

func realStateBPMNErrorConfirmedSkippedRecord(profile integrationProfile, job realStateJob) evidenceRecord {
	return evidenceRecord{
		CommandPath:       "update job",
		ScenarioName:      "real-state-bpmn-error-confirmed-skipped",
		Profile:           profile.Name,
		CamundaVersion:    profile.ExpectedVersion,
		ResourceKeys:      []string{job.Key, job.ProcessInstanceKey},
		CoveredFlags:      []string{"key", "throw-bpmn-error", "message", "vars", "auto-confirm"},
		OutputMode:        "skipped",
		DataOwnership:     []string{volumeDataSeeded, volumeDataPreexisting, "retained"},
		RequiredState:     "BPMN error-capable job with catchable ErrorTimerCode path",
		ObservedState:     "current embedded C89 fixture provides a real job for dry-run planning but no catchable BPMN error path",
		SkipReason:        "missing embedded C89 catchable BPMN error fixture and c8volt setup path for confirmed mutation",
		Outcome:           realStateOutcomeSkippedPrereq,
		FailureClass:      "",
		Preview:           false,
		ConfirmedMutation: false,
	}
}
