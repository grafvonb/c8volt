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

type realStateRetentionDataset struct {
	Fixture realStateFixture `json:"fixture"`
}

func TestRealStateRetentionFamily(t *testing.T) {
	profiles := selectedRealStateC89Profiles(t)

	report := realStateFamilyReport{
		Family:   "retention",
		Marker:   suite.marker,
		Profiles: profiles,
	}
	var failures []string
	for _, profile := range profiles {
		dataset, records, err := seedRealStateCompletedProcessInstances(t, profile, 2)
		report.Fixtures = append(report.Fixtures, dataset.Fixture)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}

		records, err = runRealStateRetentionScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeRealStateDataReport(t, "retention", report.Fixtures)
	writeRealStateProgressReport(t, "retention", report.Records)
	writeRealStateOpsReportEvidence(t, "retention", report.Records)
	writeRealStateFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("real-state retention scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func seedRealStateCompletedProcessInstances(t *testing.T, profile integrationProfile, count int) (realStateRetentionDataset, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	files, record, err := discoverEmbeddedFixtureFiles(t, profile)
	records = append(records, record)
	fixture := realStateFixture{
		FixtureKind:          "embedded no-op completion BPMN through c8volt commands",
		Marker:               suite.marker,
		Profile:              profile.Name,
		CamundaVersion:       profile.ExpectedVersion,
		RequiredState:        "completed process instances",
		CurrentEvidenceLevel: realStateOutcomeDryRunCovered,
		TargetRealStateProof: "fresh completed suite-owned process instances remain retained after retention dry-run",
	}
	if err != nil {
		return realStateRetentionDataset{Fixture: fixture}, records, err
	}

	selection, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "NoOpCompletion.bpmn")
	if err != nil {
		fixture.CurrentEvidenceLevel = realStateOutcomeSkippedPrereq
		fixture.SkipReason = "missing embedded NoOpCompletion BPMN fixture for Camunda 8.9 real-state retention setup"
		return realStateRetentionDataset{Fixture: fixture}, records, err
	}
	selection.BpmnProcessID = embeddedFixtureBpmnProcessID(t, selection.Path)
	fixture.BpmnProcessID = selection.BpmnProcessID

	deployments, record, err := deployEmbeddedFixture(t, profile, selection)
	record.CoveredFlags = []string{"file"}
	record.OutputMode = "json"
	records = append(records, record)
	fixture.ProcessDefinitionKeys = processDefinitionKeys(deployments)
	if err != nil {
		return realStateRetentionDataset{Fixture: fixture}, records, err
	}

	instances, record, err := runVolumeProcessInstances(t, profile, selection, deployments, count, "real-state-retention-completed")
	records = append(records, record)
	fixture.ProcessInstanceKeys = processInstanceKeys(instances)
	if err != nil {
		return realStateRetentionDataset{Fixture: fixture}, records, err
	}

	for _, key := range fixture.ProcessInstanceKeys {
		if err := requireProcessInstanceStateEventually(t, profile, key, []string{"COMPLETED"}, 30*time.Second); err != nil {
			return realStateRetentionDataset{Fixture: fixture}, records, err
		}
	}
	fixture.ObservedState = "completed-fresh-retained"
	return realStateRetentionDataset{Fixture: fixture}, records, nil
}

func runRealStateRetentionScenarios(t *testing.T, profile integrationProfile, dataset realStateRetentionDataset) ([]evidenceRecord, error) {
	t.Helper()
	pdKey := firstString(dataset.Fixture.ProcessDefinitionKeys)
	if pdKey == "" {
		return nil, fmt.Errorf("real-state retention dataset for profile %q has no process definition key", profile.Name)
	}
	keys := firstNStrings(dataset.Fixture.ProcessInstanceKeys, 1)
	if len(keys) == 0 {
		return nil, fmt.Errorf("real-state retention dataset for profile %q has no completed process instance key", profile.Name)
	}

	reportPath := volumeOpsExecuteReportPath(t, "real-state-retention-dry-run", profile, "json")
	result := runC8VoltForProfile(t, profile.Name, "real-state-retention-dry-run", "ops", "execute", "retention-policy", "--pd-key", pdKey, "--retention-days", "0", "--state", "completed", "--roots-only", "--no-incidents-only", "--batch-size", "1", "--limit", "1", "--workers", "1", "--dry-run", "--report-file", reportPath, "--report-format", "json")
	record := realStateRetentionRecord(profile, dataset, result, "real-state-retention-dry-run", "one-line", []string{"pd-key", "retention-days", "state", "roots-only", "no-incidents-only", "batch-size", "limit", "workers", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateRealStateRetentionDryRun(t, profile, result, reportPath, keys); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, errors.New("real-state-retention-dry-run: " + err.Error())
	}
	return []evidenceRecord{record}, nil
}

func validateRealStateRetentionDryRun(t *testing.T, profile integrationProfile, result commandResult, reportPath string, retainedKeys []string) error {
	t.Helper()
	if err := requireVolumeCommandSuccess(result, "ops execute retention-policy dry-run real-state"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: execute retention policy", "candidate retention process instances:", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readRealStateRetentionJSONReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun {
		return fmt.Errorf("real-state retention JSON report dryRun=false, want true")
	}
	if report.RetentionDays != 0 || report.Outcome != "planned" {
		return fmt.Errorf("real-state retention JSON report retentionDays/outcome = %d/%q, want 0/planned", report.RetentionDays, report.Outcome)
	}
	if report.Deletion.Submitted {
		return fmt.Errorf("real-state retention dry-run report submitted deletion")
	}
	if report.Discovery.Count == 0 || len(report.DeletePlan.SeedKeys) == 0 {
		return fmt.Errorf("real-state retention dry-run report has no candidates: discovery=%d seedKeys=%d", report.Discovery.Count, len(report.DeletePlan.SeedKeys))
	}
	for _, key := range retainedKeys {
		if err := requireProcessInstanceState(t, profile, key, "COMPLETED"); err != nil {
			return err
		}
	}
	return nil
}

func readRealStateRetentionJSONReport(path string) (realStateRetentionJSONReport, error) {
	var report realStateRetentionJSONReport
	if err := readVolumeOpsExecuteJSONReport(path, &report); err != nil {
		return report, err
	}
	return report, nil
}

type realStateRetentionJSONReport struct {
	DryRun        bool   `json:"dryRun"`
	RetentionDays int    `json:"retentionDays"`
	Outcome       string `json:"outcome"`
	Discovery     struct {
		Count int `json:"count"`
	} `json:"discovery"`
	DeletePlan struct {
		SeedKeys []string `json:"seedKeys"`
	} `json:"deletePlan"`
	Deletion struct {
		Submitted bool `json:"submitted"`
	} `json:"deletion"`
}

func realStateRetentionRecord(profile integrationProfile, dataset realStateRetentionDataset, result commandResult, scenarioName string, outputMode string, flags []string, preview bool, confirmed bool) evidenceRecord {
	record := commandEvidence("ops execute retention-policy", scenarioName, result, realStateOutcomeDryRunCovered)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.ResourceKeys = append([]string(nil), dataset.Fixture.ProcessInstanceKeys...)
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "retained"}
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	record.RequiredState = "completed process instances eligible for retention deletion with retention-days=0"
	record.ObservedState = "non-empty dry-run candidates retained after dry-run"
	return record
}

func requireProcessInstanceStateEventually(t *testing.T, profile integrationProfile, key string, states []string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for {
		last = requireProcessInstanceStateOneOf(t, profile, key, states...)
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(2 * time.Second)
	}
}
