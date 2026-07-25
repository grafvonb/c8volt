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
	"time"
)

func TestVolumeOpsRepairFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "ops-repair",
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

		incidentDataset, incidentRecords, err := seedVolumeIncidentDataset(t, profile, 3)
		report.Records = append(report.Records, incidentRecords...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		incidentKeys, incidentDiscoveryRecords, err := discoverVolumeRepairIncidentKeys(t, profile, incidentDataset)
		report.Records = append(report.Records, incidentDiscoveryRecords...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}
		incidentDataset.IncidentKeys = incidentKeys

		records, err := runVolumeOpsRepairScenarios(t, profile, dataset, incidentDataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "ops-repair", report.Datasets)
	writeVolumeProgressReport(t, "ops-repair", report.Records)
	writeVolumePipelineReport(t, "ops-repair", report.Records)
	writeVolumeOpsReportEvidence(t, "ops-repair", report.Records)
	writeCommandProposals(t, appendOpsRepairCommandGapProposals(nil))
	writeEmbeddedBPMNProposals(t, appendOpsRepairEmbeddedBPMNGapProposals(nil))
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume ops repair scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeOpsRepairScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset, incidentDataset volumeIncidentDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	incidentKeys := firstNStrings(incidentDataset.IncidentKeys, 3)
	incidentPIKeys := firstNStrings(incidentDataset.ProcessInstanceKeys, 3)
	pdKey := firstString(incidentDataset.ProcessDefinitionKeys)
	if len(incidentKeys) < 3 || len(incidentPIKeys) < 3 || pdKey == "" {
		return records, fmt.Errorf("ops repair volume incident dataset for profile %q has incidents=%d processInstances=%d pdKey=%q, want at least 3/3/non-empty", profile.Name, len(incidentKeys), len(incidentPIKeys), pdKey)
	}

	incidentDryRunReport := volumeOpsRepairReportPath(t, "volume-ops-repair-incident-dry-run", profile, "json")
	incidentDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-repair-incident-dry-run", "ops", "repair", "incident", "--bpmn-process-id", incidentDataset.BpmnProcessID, "--state", "active", "--batch-size", "1", "--limit", "1", "--workers", "1", "--retries", "2", "--job-timeout", "1m", "--vars", `{"c8voltRepair":"preview"}`, "--dry-run", "--report-file", incidentDryRunReport, "--report-format", "json")
	incidentDryRunRecord := volumeOpsRepairRecord(profile, incidentDataset, incidentDryRunResult, "ops repair incident", "volume-ops-repair-incident-dry-run", "one-line", []string{"bpmn-process-id", "state", "batch-size", "limit", "workers", "retries", "job-timeout", "vars", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsRepairIncidentDryRun(incidentDryRunResult, incidentDryRunReport); err != nil {
		incidentDryRunRecord.Outcome = volumeOutcomeFail
		incidentDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-repair-incident-dry-run: %v", err))
	}
	records = append(records, incidentDryRunRecord)

	incidentFilterReport := volumeOpsRepairReportPath(t, "volume-ops-repair-incident-filter-no-match", profile, "json")
	incidentFilterResult := runC8VoltForProfile(t, profile.Name, "volume-ops-repair-incident-filter-no-match", "ops", "repair", "incident", "--bpmn-process-id", incidentDataset.BpmnProcessID, "--pd-key", pdKey, "--pi-key", incidentPIKeys[0], "--root-key", incidentPIKeys[0], "--element-id", "c8volt_it_no_match", "--element-instance-key", incidentPIKeys[0], "--error-message", "c8volt-it-no-match", "--creation-time-after", "2000-01-01", "--creation-time-before", "2999-01-01", "--dry-run", "--report-file", incidentFilterReport, "--report-format", "json")
	incidentFilterRecord := volumeOpsRepairRecord(profile, incidentDataset, incidentFilterResult, "ops repair incident", "volume-ops-repair-incident-filter-no-match", "one-line", []string{"bpmn-process-id", "pd-key", "pi-key", "root-key", "element-id", "element-instance-key", "error-message", "creation-time-after", "creation-time-before", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsRepairIncidentDryRun(incidentFilterResult, incidentFilterReport); err != nil {
		incidentFilterRecord.Outcome = volumeOutcomeFail
		incidentFilterRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-repair-incident-filter-no-match: %v", err))
	}
	records = append(records, incidentFilterRecord)

	incidentVarsFile := volumeOpsRepairVarsFile(t, "volume-ops-repair-incident-stdin-nowait", profile)
	incidentConfirmedReport := volumeOpsRepairReportPath(t, "volume-ops-repair-incident-stdin-nowait", profile, "json")
	incidentConfirmedResult := runC8VoltWithInput(t, "volume-ops-repair-incident-stdin-nowait", incidentKeys[1]+"\n", argsForProfile(profile.Name, "--automation", "--json", "ops", "repair", "incident", "-", "--vars-file", incidentVarsFile, "--retries", "1", "--job-timeout", "30s", "--no-wait", "--no-worker-limit", "--report-file", incidentConfirmedReport, "--report-format", "json")...)
	incidentConfirmedRecord := volumeOpsRepairRecord(profile, incidentDataset, incidentConfirmedResult, "ops repair incident", "volume-ops-repair-incident-stdin-nowait", "json", []string{"stdin", "automation", "json", "vars-file", "retries", "job-timeout", "no-wait", "no-worker-limit", "report-file", "report-format"}, false, true)
	incidentConfirmedRecord.StdinPath = writeVolumeStdinKeys(t, "volume-ops-repair-incident-stdin-nowait", []string{incidentKeys[1]})
	if err := validateVolumeOpsRepairConfirmed(incidentConfirmedResult, incidentConfirmedReport, "incident"); err != nil {
		incidentConfirmedRecord.Outcome = volumeOutcomeFail
		incidentConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-repair-incident-stdin-nowait: %v", err))
	}
	records = append(records, incidentConfirmedRecord)

	piDryRunReport := volumeOpsRepairReportPath(t, "volume-ops-repair-pi-dry-run", profile, "json")
	piDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-repair-pi-dry-run", "ops", "repair", "process-instance", "--key", incidentPIKeys[2], "--vars", `{"c8voltRepair":"pi-preview"}`, "--retries", "2", "--job-timeout", "1m", "--workers", "1", "--dry-run", "--report-file", piDryRunReport, "--report-format", "json")
	piDryRunRecord := volumeOpsRepairRecord(profile, incidentDataset, piDryRunResult, "ops repair process-instance", "volume-ops-repair-pi-dry-run", "one-line", []string{"key", "vars", "retries", "job-timeout", "workers", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsRepairPIDryRun(piDryRunResult, piDryRunReport); err != nil {
		piDryRunRecord.Outcome = volumeOutcomeFail
		piDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-repair-pi-dry-run: %v", err))
	}
	records = append(records, piDryRunRecord)

	piSearchReport := volumeOpsRepairReportPath(t, "volume-ops-repair-pi-search-no-match", profile, "json")
	piSearchResult := runC8VoltForProfile(t, profile.Name, "volume-ops-repair-pi-search-no-match", "ops", "repair", "process-instance", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--state", "active", "--roots-only", "--direct-incidents-only", "--incident-state", "active", "--incident-error-message", "c8volt-it-no-match", "--start-date-after", "2000-01-01", "--start-date-before", "2999-01-01", "--batch-size", "1", "--limit", "1", "--no-worker-limit", "--dry-run", "--report-file", piSearchReport, "--report-format", "json")
	piSearchRecord := volumeOpsRepairRecord(profile, incidentDataset, piSearchResult, "ops repair process-instance", "volume-ops-repair-pi-search-no-match", "one-line", []string{"bpmn-process-id", "state", "roots-only", "direct-incidents-only", "incident-state", "incident-error-message", "start-date-after", "start-date-before", "batch-size", "limit", "no-worker-limit", "dry-run", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsRepairPIDryRun(piSearchResult, piSearchReport); err != nil {
		piSearchRecord.Outcome = volumeOutcomeFail
		piSearchRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-repair-pi-search-no-match: %v", err))
	}
	records = append(records, piSearchRecord)

	piVarsFile := volumeOpsRepairVarsFile(t, "volume-ops-repair-pi-nowait", profile)
	piConfirmedReport := volumeOpsRepairReportPath(t, "volume-ops-repair-pi-nowait", profile, "json")
	piConfirmedResult := runC8VoltForProfile(t, profile.Name, "volume-ops-repair-pi-nowait", "--automation", "--json", "ops", "repair", "process-instance", "--key", incidentPIKeys[2], "--vars-file", piVarsFile, "--retries", "1", "--job-timeout", "30s", "--no-wait", "--report-file", piConfirmedReport, "--report-format", "json")
	piConfirmedRecord := volumeOpsRepairRecord(profile, incidentDataset, piConfirmedResult, "ops repair process-instance", "volume-ops-repair-pi-nowait", "json", []string{"automation", "json", "key", "vars-file", "retries", "job-timeout", "no-wait", "report-file", "report-format"}, false, true)
	if err := validateVolumeOpsRepairConfirmed(piConfirmedResult, piConfirmedReport, "process-instance"); err != nil {
		piConfirmedRecord.Outcome = volumeOutcomeFail
		piConfirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-repair-pi-nowait: %v", err))
	}
	records = append(records, piConfirmedRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func discoverVolumeRepairIncidentKeys(t *testing.T, profile integrationProfile, dataset volumeIncidentDataset) ([]string, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var incidentKeys []string
	var failures []string
	for i, piKey := range dataset.ProcessInstanceKeys {
		scenario := fmt.Sprintf("volume-ops-repair-get-incident-%d", i+1)
		var result commandResult
		var found []string
		for attempt := 1; attempt <= 6; attempt++ {
			attemptScenario := scenario
			if attempt > 1 {
				attemptScenario = fmt.Sprintf("%s-attempt-%d", scenario, attempt)
			}
			result = runC8VoltForProfile(t, profile.Name, attemptScenario, "--automation", "--json", "get", "incident", "--pi-key", piKey, "--state", "active", "--limit", "5", "--batch-size", "1")
			if err := requireVolumeCommandSuccess(result, "get incident for ops repair volume"); err != nil {
				break
			}
			found = volumeRepairIncidentKeysFromResult(t, result, piKey)
			if len(found) > 0 {
				break
			}
			time.Sleep(2 * time.Second)
		}
		record := volumeOpsRepairRecord(profile, dataset, result, "get incident", scenario, "json", []string{"pi-key", "state", "limit", "batch-size"}, false, false)
		if err := requireVolumeCommandSuccess(result, "get incident for ops repair volume"); err != nil {
			record.Outcome = volumeOutcomeFail
			record.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("%s: %v", scenario, err))
			records = append(records, record)
			continue
		}
		incidentKeys = append(incidentKeys, found...)
		record.ResourceKeys = append(record.ResourceKeys, incidentKeys...)
		records = append(records, record)
	}
	incidentKeys = uniqueSortedStrings(incidentKeys)
	if len(failures) > 0 {
		return incidentKeys, records, errors.New(strings.Join(failures, "\n"))
	}
	return incidentKeys, records, nil
}

func volumeRepairIncidentKeysFromResult(t *testing.T, result commandResult, piKey string) []string {
	t.Helper()
	var incidents struct {
		Items []struct {
			IncidentKey        string `json:"incidentKey"`
			ProcessInstanceKey string `json:"processInstanceKey"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &incidents); err != nil {
		return nil
	}
	var keys []string
	for _, item := range incidents.Items {
		if item.ProcessInstanceKey == piKey && item.IncidentKey != "" {
			keys = append(keys, item.IncidentKey)
		}
	}
	return keys
}

func volumeOpsRepairRecord(profile integrationProfile, dataset volumeIncidentDataset, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, preview bool, confirmed bool) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "mutated", "retained"}
	record.ResourceKeys = append(append([]string{}, dataset.ProcessInstanceKeys...), dataset.IncidentKeys...)
	return record
}

func volumeOpsRepairReportPath(t *testing.T, scenarioName string, profile integrationProfile, ext string) string {
	t.Helper()
	name := sanitizeEvidenceName(scenarioName + "-" + profile.Name)
	path := filepath.Join(suite.workDir, "data", name+"."+ext)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create ops repair report dir: %v", err)
	}
	return path
}

func volumeOpsRepairVarsFile(t *testing.T, scenarioName string, profile integrationProfile) string {
	t.Helper()
	path := filepath.Join(suite.workDir, "data", sanitizeEvidenceName(scenarioName+"-"+profile.Name)+"-vars.json")
	if err := os.WriteFile(path, []byte(`{"c8voltRepair":"confirmed"}`+"\n"), 0o644); err != nil {
		t.Fatalf("write ops repair vars file: %v", err)
	}
	return path
}

func validateVolumeOpsRepairIncidentDryRun(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops repair incident dry-run volume"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: repair incidents", "candidate incidents:", "repair preview:", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readVolumeOpsRepairReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun || report.Outcome != "planned" {
		return fmt.Errorf("repair incident dry-run report dryRun/outcome = %t/%q, want true/planned", report.DryRun, report.Outcome)
	}
	return nil
}

func validateVolumeOpsRepairPIDryRun(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops repair process-instance dry-run volume"); err != nil {
		return err
	}
	if err := requireHumanContains(volumeHumanOutput(result), "dry run: repair process-instance incidents", "repairable process instances:", "active incidents:", "repair preview:", "outcome: planned", "report: written"); err != nil {
		return err
	}
	report, err := readVolumeOpsRepairReport(reportPath)
	if err != nil {
		return err
	}
	if !report.DryRun || report.Outcome != "planned" {
		return fmt.Errorf("repair pi dry-run report dryRun/outcome = %t/%q, want true/planned", report.DryRun, report.Outcome)
	}
	return nil
}

func validateVolumeOpsRepairConfirmed(result commandResult, reportPath string, label string) error {
	if err := requireVolumeCommandSuccess(result, "ops repair "+label+" confirmed volume"); err != nil {
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
		Outcome string `json:"outcome"`
		Request struct {
			NoWait bool `json:"noWait"`
		} `json:"request"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode repair %s payload: %w", label, err)
	}
	if payload.Outcome != "repaired" && payload.Outcome != "planned" {
		return fmt.Errorf("repair %s payload outcome = %q, want repaired or planned", label, payload.Outcome)
	}
	if !payload.Request.NoWait {
		return fmt.Errorf("repair %s payload noWait=false, want true", label)
	}
	report, err := readVolumeOpsRepairReport(reportPath)
	if err != nil {
		return err
	}
	if report.Outcome != payload.Outcome {
		return fmt.Errorf("repair %s stdout/report outcome mismatch: %q/%q", label, payload.Outcome, report.Outcome)
	}
	if !report.NoWait {
		return fmt.Errorf("repair %s report noWait=false, want true", label)
	}
	return nil
}

func readVolumeOpsRepairReport(path string) (volumeOpsRepairReport, error) {
	var report volumeOpsRepairReport
	content, err := os.ReadFile(path)
	if err != nil {
		return report, fmt.Errorf("read ops repair JSON report: %w", err)
	}
	if err := json.Unmarshal(content, &report); err != nil {
		return report, fmt.Errorf("decode ops repair JSON report %s: %w; content: %q", path, err, compactLogSnippet(string(content), 300))
	}
	return report, nil
}

type volumeOpsRepairReport struct {
	DryRun  bool   `json:"dryRun"`
	NoWait  bool   `json:"noWait"`
	Outcome string `json:"outcome"`
}

// appendOpsRepairCommandGapProposals records remaining setup gaps for repair edge cases.
func appendOpsRepairCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	return registerDirectCamundaSetupFallback(proposals,
		"repair targets with mixed related-job availability, partial failures, and stale incidents",
		"ops repair partial-failure and notice coverage",
		[]string{"ops repair incident", "ops repair process-instance"},
		supportedProposalVersions(),
		"Operators can validate partial repair reporting and notices without direct API incident/job manipulation.",
	)
}

// appendOpsRepairEmbeddedBPMNGapProposals records missing fixtures for deterministic repair edge cases.
func appendOpsRepairEmbeddedBPMNGapProposals(proposals []proposalRecord) []proposalRecord {
	return registerMissingEmbeddedBPMNProposal(proposals,
		"process model with deterministic repairable incidents, jobs, and controlled partial-failure branches",
		"ops repair job and partial-failure coverage",
		[]string{"ops repair incident", "ops repair process-instance"},
		supportedProposalVersions(),
		"Repair volume tests can cover retries, timeouts, variables, and failure reports without external setup.",
	)
}
