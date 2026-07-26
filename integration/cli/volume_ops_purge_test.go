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

func TestVolumeOpsPurgeFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "ops-purge",
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

		records, err := runVolumeOpsPurgeScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "ops-purge", report.Datasets)
	writeVolumeProgressReport(t, "ops-purge", report.Records)
	writeVolumePipelineReport(t, "ops-purge", report.Records)
	writeVolumeOpsReportEvidence(t, "ops-purge", report.Records)
	writeCommandProposals(t, nil)
	writeEmbeddedBPMNProposals(t, nil)
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume ops purge scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeOpsPurgeScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	pdKey := firstString(dataset.PositiveProcessDefinitionKeys)
	piKey := firstString(dataset.PositiveProcessInstanceKeys)
	if pdKey == "" || piKey == "" {
		return records, fmt.Errorf("ops purge volume dataset for profile %q missing pd/pi key", profile.Name)
	}
	missingKey := "2251799813685249"
	noMatchBPMN := sanitizeEvidenceName(suite.marker + "-ops-purge-no-match")

	allPDDryRunReport := volumeOpsPurgeReportPath(t, "volume-ops-purge-all-pds-dry-run", profile, "json")
	allPDDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-purge-all-pds-dry-run", "ops", "purge", "all-process-definitions", "--key", pdKey, "--batch-size", "1", "--limit", "1", "--workers", "1", "--dry-run", "--report-file", allPDDryRunReport, "--report-format", "json")
	allPDDryRunRecord := volumeOpsPurgeRecord(profile, dataset, allPDDryRunResult, "ops purge all-process-definitions", "volume-ops-purge-all-pds-dry-run", "one-line", []string{"key", "batch-size", "limit", "workers", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsPurgeAllPDDryRun(allPDDryRunResult, allPDDryRunReport); err != nil {
		allPDDryRunRecord.Outcome = volumeOutcomeFail
		allPDDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-purge-all-pds-dry-run: %v", err))
	}
	records = append(records, allPDDryRunRecord)

	allPDConfirmedReport := volumeOpsPurgeReportPath(t, "volume-ops-purge-all-pds-confirmed-no-match", profile, "json")
	allPDConfirmedResult := runC8VoltForProfile(t, profile.Name, "volume-ops-purge-all-pds-confirmed-no-match", "--automation", "--json", "ops", "purge", "all-process-definitions", "--bpmn-process-id", noMatchBPMN, "--pd-version", "9999", "--pd-version-tag", "c8volt-it-no-match", "--latest", "--force", "--fail-fast", "--no-worker-limit", "--no-wait", "--report-file", allPDConfirmedReport, "--report-format", "json")
	allPDConfirmedRecord := volumeOpsPurgeRecord(profile, dataset, allPDConfirmedResult, "ops purge all-process-definitions", "volume-ops-purge-all-pds-confirmed-no-match", "json", []string{"automation", "json", "bpmn-process-id", "pd-version", "pd-version-tag", "latest", "force", "fail-fast", "no-worker-limit", "no-wait", "report-file", "report-format"}, false, true)
	if err := validateVolumeOpsPurgeAllPDConfirmed(allPDConfirmedResult, allPDConfirmedReport); err != nil {
		allPDConfirmedRecord.Outcome = volumeOutcomeFail
		allPDConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-purge-all-pds-confirmed-no-match: %v", err))
	}
	records = append(records, allPDConfirmedRecord)

	orphanDryRunReport := volumeOpsPurgeReportPath(t, "volume-ops-purge-orphan-dry-run", profile, "json")
	orphanDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-purge-orphan-dry-run", "ops", "purge", "orphan-process-instances", "--pd-key", pdKey, "--state", "active", "--start-date-after", "2000-01-01", "--start-date-before", "2999-01-01", "--no-incidents-only", "--batch-size", "1", "--limit", "1", "--workers", "1", "--dry-run", "--report-file", orphanDryRunReport, "--report-format", "json")
	orphanDryRunRecord := volumeOpsPurgeRecord(profile, dataset, orphanDryRunResult, "ops purge orphan-process-instances", "volume-ops-purge-orphan-dry-run", "one-line", []string{"pd-key", "state", "start-date-after", "start-date-before", "no-incidents-only", "batch-size", "limit", "workers", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsPurgeOrphanDryRun(orphanDryRunResult, orphanDryRunReport); err != nil {
		orphanDryRunRecord.Outcome = volumeOutcomeFail
		orphanDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-purge-orphan-dry-run: %v", err))
	}
	records = append(records, orphanDryRunRecord)

	orphanConfirmedReport := volumeOpsPurgeReportPath(t, "volume-ops-purge-orphan-confirmed-no-match", profile, "json")
	orphanConfirmedResult := runC8VoltForProfile(t, profile.Name, "volume-ops-purge-orphan-confirmed-no-match", "--automation", "--json", "ops", "purge", "orphan-process-instances", "--bpmn-process-id", dataset.NegativeBpmnProcessID, "--parent-key", missingKey, "--incidents-only", "--batch-size", "1", "--limit", "1", "--force", "--fail-fast", "--no-worker-limit", "--no-wait", "--report-file", orphanConfirmedReport, "--report-format", "json")
	orphanConfirmedRecord := volumeOpsPurgeRecord(profile, dataset, orphanConfirmedResult, "ops purge orphan-process-instances", "volume-ops-purge-orphan-confirmed-no-match", "json", []string{"automation", "json", "bpmn-process-id", "parent-key", "incidents-only", "batch-size", "limit", "force", "fail-fast", "no-worker-limit", "no-wait", "report-file", "report-format"}, false, true)
	if err := validateVolumeOpsPurgeOrphanConfirmed(orphanConfirmedResult, orphanConfirmedReport); err != nil {
		orphanConfirmedRecord.Outcome = volumeOutcomeFail
		orphanConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-purge-orphan-confirmed-no-match: %v", err))
	}
	records = append(records, orphanConfirmedRecord)

	incidentDryRunReport := volumeOpsPurgeReportPath(t, "volume-ops-purge-incident-dry-run", profile, "json")
	incidentDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-purge-incident-dry-run", "ops", "purge", "process-instances-with-incidents", "--inc-key", missingKey, "--state", "active", "--batch-size", "1", "--limit", "1", "--workers", "1", "--dry-run", "--report-file", incidentDryRunReport, "--report-format", "json")
	incidentDryRunRecord := volumeOpsPurgeRecord(profile, dataset, incidentDryRunResult, "ops purge process-instances-with-incidents", "volume-ops-purge-incident-dry-run", "one-line", []string{"inc-key", "state", "batch-size", "limit", "workers", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsPurgeIncidentDryRun(incidentDryRunResult, incidentDryRunReport); err != nil {
		incidentDryRunRecord.Outcome = volumeOutcomeFail
		incidentDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-purge-incident-dry-run: %v", err))
	}
	records = append(records, incidentDryRunRecord)

	incidentConfirmedReport := volumeOpsPurgeReportPath(t, "volume-ops-purge-incident-confirmed-no-match", profile, "json")
	incidentConfirmedResult := runC8VoltForProfile(t, profile.Name, "volume-ops-purge-incident-confirmed-no-match", "--automation", "--json", "ops", "purge", "process-instances-with-incidents", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--pd-key", pdKey, "--pi-key", piKey, "--root-key", piKey, "--element-id", "c8volt_it_no_match", "--element-instance-key", piKey, "--error-message", "c8volt-it-no-match", "--creation-time-after", "2000-01-01", "--creation-time-before", "2999-01-01", "--force", "--fail-fast", "--no-worker-limit", "--no-wait", "--report-file", incidentConfirmedReport, "--report-format", "json")
	incidentConfirmedRecord := volumeOpsPurgeRecord(profile, dataset, incidentConfirmedResult, "ops purge process-instances-with-incidents", "volume-ops-purge-incident-confirmed-no-match", "json", []string{"automation", "json", "bpmn-process-id", "pd-key", "pi-key", "root-key", "element-id", "element-instance-key", "error-message", "creation-time-after", "creation-time-before", "force", "fail-fast", "no-worker-limit", "no-wait", "report-file", "report-format"}, false, true)
	if err := validateVolumeOpsPurgeIncidentConfirmed(incidentConfirmedResult, incidentConfirmedReport); err != nil {
		incidentConfirmedRecord.Outcome = volumeOutcomeFail
		incidentConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-purge-incident-confirmed-no-match: %v", err))
	}
	records = append(records, incidentConfirmedRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeOpsPurgeRecord(profile integrationProfile, dataset volumeDataset, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, preview bool, confirmed bool) evidenceRecord {
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

func volumeOpsPurgeReportPath(t *testing.T, scenarioName string, profile integrationProfile, ext string) string {
	t.Helper()
	name := sanitizeEvidenceName(scenarioName + "-" + profile.Name)
	path := filepath.Join(suite.workDir, "data", name+"."+ext)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create ops purge report dir: %v", err)
	}
	return path
}

func validateVolumeOpsPurgeAllPDDryRun(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops purge all-process-definitions dry-run volume"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: purge all process definitions", "candidate process definitions:", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readVolumeOpsPurgeAllPDReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun || report.Outcome != "planned" {
		return fmt.Errorf("all-pds dry-run report dryRun/outcome = %t/%q, want true/planned", report.DryRun, report.Outcome)
	}
	if report.Discovery.CandidateProcessDefinitionCount == 0 {
		return fmt.Errorf("all-pds dry-run report found no candidate process definitions")
	}
	return nil
}

func validateVolumeOpsPurgeAllPDConfirmed(result commandResult, reportPath string) error {
	report, err := readVolumeOpsPurgeAllPDReport(reportPath)
	if err != nil {
		return err
	}
	return validateVolumeOpsPurgeJSONNoMatch(result, "all-pds", report.Outcome, report.NoWait, report.Force, report.Deletion.Submitted)
}

func validateVolumeOpsPurgeOrphanDryRun(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops purge orphan-process-instances dry-run volume"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: purge orphan process-instances", "candidate orphan process instances:", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readVolumeOpsPurgeOrphanReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun || report.Outcome != "planned" {
		return fmt.Errorf("orphan dry-run report dryRun/outcome = %t/%q, want true/planned", report.DryRun, report.Outcome)
	}
	return nil
}

func validateVolumeOpsPurgeOrphanConfirmed(result commandResult, reportPath string) error {
	report, err := readVolumeOpsPurgeOrphanReport(reportPath)
	if err != nil {
		return err
	}
	return validateVolumeOpsPurgeJSONNoMatch(result, "orphan", report.Outcome, report.NoWait, true, report.Deletion.Submitted)
}

func validateVolumeOpsPurgeIncidentDryRun(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops purge process-instances-with-incidents dry-run volume"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: purge process-instances with incidents", "candidate incidents:", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readVolumeOpsPurgeIncidentReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun || report.Outcome != "planned" {
		return fmt.Errorf("incident dry-run report dryRun/outcome = %t/%q, want true/planned", report.DryRun, report.Outcome)
	}
	return nil
}

func validateVolumeOpsPurgeIncidentConfirmed(result commandResult, reportPath string) error {
	report, err := readVolumeOpsPurgeIncidentReport(reportPath)
	if err != nil {
		return err
	}
	return validateVolumeOpsPurgeJSONNoMatch(result, "incident", report.Outcome, report.NoWait, report.Force, report.Deletion.Submitted)
}

func validateVolumeOpsPurgeJSONNoMatch(result commandResult, label string, reportOutcome string, reportNoWait bool, reportCriticalFlag bool, reportSubmitted bool) error {
	if err := requireVolumeCommandSuccess(result, "ops purge "+label+" confirmed volume"); err != nil {
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
			Submitted bool `json:"submitted"`
		} `json:"deletion"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode ops purge %s payload: %w", label, err)
	}
	if payload.Outcome != "planned" {
		return fmt.Errorf("ops purge %s payload outcome = %q, want planned", label, payload.Outcome)
	}
	if reportOutcome != payload.Outcome {
		return fmt.Errorf("ops purge %s stdout/report outcome mismatch: %q/%q", label, payload.Outcome, reportOutcome)
	}
	if !reportNoWait || !reportCriticalFlag {
		return fmt.Errorf("ops purge %s report missing no-wait or critical confirmed flag: noWait=%t critical=%t", label, reportNoWait, reportCriticalFlag)
	}
	if reportSubmitted || payload.Deletion.Submitted {
		return fmt.Errorf("ops purge %s no-match submitted deletion: stdout=%t report=%t", label, payload.Deletion.Submitted, reportSubmitted)
	}
	return nil
}

func readVolumeOpsPurgeAllPDReport(path string) (volumeOpsPurgeAllPDReport, error) {
	var report volumeOpsPurgeAllPDReport
	if err := readVolumeOpsPurgeJSONReport(path, &report); err != nil {
		return report, err
	}
	return report, nil
}

func readVolumeOpsPurgeOrphanReport(path string) (volumeOpsPurgeOrphanReport, error) {
	var report volumeOpsPurgeOrphanReport
	if err := readVolumeOpsPurgeJSONReport(path, &report); err != nil {
		return report, err
	}
	return report, nil
}

func readVolumeOpsPurgeIncidentReport(path string) (volumeOpsPurgeIncidentReport, error) {
	var report volumeOpsPurgeIncidentReport
	if err := readVolumeOpsPurgeJSONReport(path, &report); err != nil {
		return report, err
	}
	return report, nil
}

func readVolumeOpsPurgeJSONReport(path string, value any) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read ops purge JSON report: %w", err)
	}
	if err := json.Unmarshal(content, value); err != nil {
		return fmt.Errorf("decode ops purge JSON report %s: %w; content: %q", path, err, compactLogSnippet(string(content), 300))
	}
	return nil
}

type volumeOpsPurgeAllPDReport struct {
	DryRun    bool   `json:"dryRun"`
	NoWait    bool   `json:"noWait"`
	Force     bool   `json:"force"`
	Outcome   string `json:"outcome"`
	Discovery struct {
		CandidateProcessDefinitionCount int `json:"candidateProcessDefinitionCount"`
	} `json:"discovery"`
	Deletion struct {
		Submitted bool                         `json:"submitted"`
		Confirmed bool                         `json:"confirmed"`
		Items     []realStateCascadeDeleteItem `json:"items"`
	} `json:"deletion"`
}

type volumeOpsPurgeOrphanReport struct {
	DryRun          bool   `json:"dryRun"`
	NoWait          bool   `json:"noWait"`
	DeleteRequested bool   `json:"deleteRequested"`
	Outcome         string `json:"outcome"`
	Deletion        struct {
		Submitted bool `json:"submitted"`
	} `json:"deletion"`
}

type volumeOpsPurgeIncidentReport struct {
	DryRun   bool   `json:"dryRun"`
	NoWait   bool   `json:"noWait"`
	Force    bool   `json:"force"`
	Outcome  string `json:"outcome"`
	Deletion struct {
		Submitted bool `json:"submitted"`
	} `json:"deletion"`
}
