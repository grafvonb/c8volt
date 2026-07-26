// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVolumeOpsExecuteFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "ops-execute",
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

		records, err := runVolumeOpsExecuteScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "ops-execute", report.Datasets)
	writeVolumeProgressReport(t, "ops-execute", report.Records)
	writeVolumePipelineReport(t, "ops-execute", report.Records)
	writeVolumeOpsReportEvidence(t, "ops-execute", report.Records)
	writeCommandProposals(t, appendOpsExecuteCommandGapProposals(nil))
	writeEmbeddedBPMNProposals(t, appendOpsExecuteEmbeddedBPMNGapProposals(nil))
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume ops execute scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeOpsExecuteScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	pdKey := firstString(dataset.PositiveProcessDefinitionKeys)
	if pdKey == "" {
		return records, fmt.Errorf("ops execute volume dataset for profile %q has no process definition key", profile.Name)
	}

	smokeDryRunReport := volumeOpsExecuteReportPath(t, "volume-ops-execute-smoke-dry-run", profile, "json")
	smokeDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-execute-smoke-dry-run", "ops", "execute", "smoke-test", "--dry-run", "--count", "2", "--workers", "1", "--fail-fast", "--report-file", smokeDryRunReport, "--report-format", "json")
	smokeDryRunRecord := volumeOpsExecuteRecord(profile, dataset, smokeDryRunResult, "ops execute smoke-test", "volume-ops-execute-smoke-dry-run", "one-line", []string{"dry-run", "count", "workers", "fail-fast", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsExecuteSmokeDryRun(smokeDryRunResult, smokeDryRunReport); err != nil {
		smokeDryRunRecord.Outcome = volumeOutcomeFail
		smokeDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-execute-smoke-dry-run: %v", err))
	}
	records = append(records, smokeDryRunRecord)

	smokePlanReport := volumeOpsExecuteReportPath(t, "volume-ops-execute-smoke-plan-no-cleanup", profile, "json")
	smokePlanResult := runC8VoltForProfile(t, profile.Name, "volume-ops-execute-smoke-plan-no-cleanup", "ops", "execute", "smoke-test", "--dry-run", "--count", "1", "--no-cleanup", "--no-worker-limit", "--report-file", smokePlanReport, "--report-format", "json")
	smokePlanRecord := volumeOpsExecuteRecord(profile, dataset, smokePlanResult, "ops execute smoke-test", "volume-ops-execute-smoke-plan-no-cleanup", "one-line", []string{"dry-run", "count", "no-cleanup", "no-worker-limit", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsExecuteSmokeDryRun(smokePlanResult, smokePlanReport); err != nil {
		smokePlanRecord.Outcome = volumeOutcomeFail
		smokePlanRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-execute-smoke-plan-no-cleanup: %v", err))
	}
	records = append(records, smokePlanRecord)

	smokeConfirmedReport := volumeOpsExecuteReportPath(t, "volume-ops-execute-smoke-confirmed-nowait", profile, "md")
	smokeConfirmedResult := runC8VoltForProfile(t, profile.Name, "volume-ops-execute-smoke-confirmed-nowait", "--automation", "ops", "execute", "smoke-test", "--count", "1", "--workers", "1", "--no-wait", "--report-file", smokeConfirmedReport)
	smokeConfirmedRecord := volumeOpsExecuteRecord(profile, dataset, smokeConfirmedResult, "ops execute smoke-test", "volume-ops-execute-smoke-confirmed-nowait", "one-line", []string{"automation", "count", "workers", "no-wait", "report-file"}, false, true)
	if err := validateVolumeOpsExecuteSmokeConfirmed(smokeConfirmedResult, smokeConfirmedReport); err != nil {
		smokeConfirmedRecord.Outcome = volumeOutcomeFail
		smokeConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-execute-smoke-confirmed-nowait: %v", err))
	}
	records = append(records, smokeConfirmedRecord)

	retentionDryRunReport := volumeOpsExecuteReportPath(t, "volume-ops-execute-retention-dry-run", profile, "json")
	retentionDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-execute-retention-dry-run", "ops", "execute", "retention-policy", "--pd-key", pdKey, "--retention-days", "9999", "--state", "completed", "--roots-only", "--no-incidents-only", "--batch-size", "1", "--limit", "1", "--workers", "1", "--dry-run", "--report-file", retentionDryRunReport, "--report-format", "json")
	retentionDryRunRecord := volumeOpsExecuteRecord(profile, dataset, retentionDryRunResult, "ops execute retention-policy", "volume-ops-execute-retention-dry-run", "one-line", []string{"pd-key", "retention-days", "state", "roots-only", "no-incidents-only", "batch-size", "limit", "workers", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsExecuteRetentionDryRun(retentionDryRunResult, retentionDryRunReport); err != nil {
		retentionDryRunRecord.Outcome = volumeOutcomeFail
		retentionDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-execute-retention-dry-run: %v", err))
	}
	records = append(records, retentionDryRunRecord)

	retentionConfirmedReport := volumeOpsExecuteReportPath(t, "volume-ops-execute-retention-confirmed-nowait", profile, "json")
	retentionConfirmedResult := runC8VoltForProfile(t, profile.Name, "volume-ops-execute-retention-confirmed-nowait", "--automation", "--json", "ops", "execute", "retention-policy", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--retention-days", "9999", "--children-only", "--no-state-check", "--force", "--fail-fast", "--no-worker-limit", "--no-wait", "--report-file", retentionConfirmedReport, "--report-format", "json")
	retentionConfirmedRecord := volumeOpsExecuteRecord(profile, dataset, retentionConfirmedResult, "ops execute retention-policy", "volume-ops-execute-retention-confirmed-nowait", "json", []string{"automation", "json", "bpmn-process-id", "retention-days", "children-only", "no-state-check", "force", "fail-fast", "no-worker-limit", "no-wait", "report-file", "report-format"}, false, true)
	if err := validateVolumeOpsExecuteRetentionConfirmed(retentionConfirmedResult, retentionConfirmedReport); err != nil {
		retentionConfirmedRecord.Outcome = volumeOutcomeFail
		retentionConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-execute-retention-confirmed-nowait: %v", err))
	}
	records = append(records, retentionConfirmedRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeOpsExecuteRecord(profile integrationProfile, dataset volumeDataset, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, preview bool, confirmed bool) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "mutated", "retained"}
	record.ResourceKeys = append([]string(nil), dataset.allProcessInstanceKeys()...)
	return record
}

func volumeOpsExecuteReportPath(t *testing.T, scenarioName string, profile integrationProfile, ext string) string {
	t.Helper()
	name := sanitizeEvidenceName(scenarioName + "-" + profile.Name)
	path := filepath.Join(suite.workDir, "data", name+"."+ext)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create ops execute report dir: %v", err)
	}
	return path
}

func validateVolumeOpsExecuteSmokeDryRun(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops execute smoke-test dry-run volume"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: execute smoke test", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readVolumeOpsExecuteSmokeJSONReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun {
		return fmt.Errorf("smoke-test JSON report dryRun=false, want true")
	}
	if report.Outcome != "planned" {
		return fmt.Errorf("smoke-test JSON report outcome = %q, want planned", report.Outcome)
	}
	if report.Plan.Status == "" || report.Fixture.File == "" {
		return fmt.Errorf("smoke-test JSON report missing plan status or fixture: %+v", report)
	}
	return nil
}

func validateVolumeOpsExecuteSmokeConfirmed(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops execute smoke-test confirmed volume"); err != nil {
		return err
	}
	humanOutput := volumeHumanOutput(result)
	if err := requireHumanContains(humanOutput, "execute smoke test", "created process instances:", "cleanup:", "outcome:", "report: written"); err != nil {
		return err
	}
	if err := requireNoWaitOrSubmittedText(humanOutput); err != nil {
		return err
	}
	content, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read smoke-test markdown report: %w", err)
	}
	text := string(content)
	for _, token := range []string{"# Execute Smoke Test Audit Report", "No Wait: true", "Outcome:", "## Cleanup"} {
		if !strings.Contains(text, token) {
			return fmt.Errorf("smoke-test markdown report missing %q: %q", token, compactLogSnippet(text, 300))
		}
	}
	return nil
}

func validateVolumeOpsExecuteRetentionDryRun(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops execute retention-policy dry-run volume"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: execute retention policy", "candidate retention process instances:", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readVolumeOpsExecuteRetentionJSONReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun {
		return fmt.Errorf("retention JSON report dryRun=false, want true")
	}
	if report.RetentionDays != 9999 || report.Outcome != "planned" {
		return fmt.Errorf("retention JSON report retentionDays/outcome = %d/%q, want 9999/planned", report.RetentionDays, report.Outcome)
	}
	if report.Deletion.Submitted {
		return fmt.Errorf("retention dry-run report submitted deletion")
	}
	return nil
}

func validateVolumeOpsExecuteRetentionConfirmed(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops execute retention-policy confirmed volume"); err != nil {
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
		Outcome  string `json:"outcome"`
		Deletion struct {
			NoWait    bool `json:"noWait"`
			Submitted bool `json:"submitted"`
		} `json:"deletion"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode retention-policy payload: %w", err)
	}
	if payload.Outcome != "planned" && payload.Outcome != "deleted" {
		return fmt.Errorf("retention-policy payload outcome = %q, want planned or deleted", payload.Outcome)
	}
	report, err := readVolumeOpsExecuteRetentionJSONReport(reportPath)
	if err != nil {
		return err
	}
	if !report.NoWait || !report.Force || !report.NoStateCheck {
		return fmt.Errorf("retention JSON report missing critical flags: noWait=%t force=%t noStateCheck=%t", report.NoWait, report.Force, report.NoStateCheck)
	}
	if report.Outcome != payload.Outcome {
		return fmt.Errorf("retention stdout/report outcome mismatch: %q/%q", payload.Outcome, report.Outcome)
	}
	if report.Deletion.Submitted != payload.Deletion.Submitted {
		return fmt.Errorf("retention stdout/report submitted mismatch: %t/%t", payload.Deletion.Submitted, report.Deletion.Submitted)
	}
	return nil
}

func readVolumeOpsExecuteSmokeJSONReport(path string) (volumeOpsExecuteSmokeReport, error) {
	var report volumeOpsExecuteSmokeReport
	if err := readVolumeOpsExecuteJSONReport(path, &report); err != nil {
		return report, err
	}
	return report, nil
}

func readVolumeOpsExecuteRetentionJSONReport(path string) (volumeOpsExecuteRetentionReport, error) {
	var report volumeOpsExecuteRetentionReport
	if err := readVolumeOpsExecuteJSONReport(path, &report); err != nil {
		return report, err
	}
	return report, nil
}

func readVolumeOpsExecuteJSONReport(path string, value any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read ops execute JSON report: %w", err)
	}
	if err := json.Unmarshal(content, value); err != nil {
		return fmt.Errorf("decode ops execute JSON report %s: %w; content: %q", path, err, compactLogSnippet(string(content), 300))
	}
	return nil
}

type volumeOpsExecuteSmokeReport struct {
	DryRun  bool   `json:"dryRun"`
	Outcome string `json:"outcome"`
	Plan    struct {
		Status string `json:"status"`
	} `json:"plan"`
	Fixture struct {
		File string `json:"file"`
	} `json:"fixture"`
}

type volumeOpsExecuteRetentionReport struct {
	DryRun        bool   `json:"dryRun"`
	RetentionDays int    `json:"retentionDays"`
	NoWait        bool   `json:"noWait"`
	NoStateCheck  bool   `json:"noStateCheck"`
	Force         bool   `json:"force"`
	Outcome       string `json:"outcome"`
	Deletion      struct {
		Submitted bool `json:"submitted"`
	} `json:"deletion"`
}

func requireHumanContains(output string, values ...string) error {
	for _, value := range values {
		if !strings.Contains(output, value) {
			return fmt.Errorf("human output missing %q: %q", value, compactLogSnippet(output, 300))
		}
	}
	return nil
}

func volumeHumanOutput(result commandResult) string {
	return result.Stdout + "\n" + result.Stderr
}
