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

	apiLatencyDryRunReport := volumeOpsExecuteReportPath(t, "volume-ops-execute-api-latency-dry-run-selected-version", profile, "json")
	apiLatencyDryRunResult := runC8VoltForProfile(t, profile.Name, "volume-ops-execute-api-latency-dry-run-selected-version", "--automation", "--json", "--backoff-timeout", "10s", "--backoff-max-retries", "2", "ops", "execute", "api-latency-test", "--dry-run", "--count", "3", "--workers", "2", "--report-file", apiLatencyDryRunReport, "--report-format", "json")
	apiLatencyDryRunRecord := volumeOpsExecuteAPILatencyRecord(profile, dataset, apiLatencyDryRunResult, "volume-ops-execute-api-latency-dry-run-selected-version", "json", []string{"automation", "json", "backoff-timeout", "backoff-max-retries", "dry-run", "count", "workers", "report-file", "report-format"}, true, false)
	if err := validateVolumeOpsExecuteAPILatencyDryRun(apiLatencyDryRunResult, apiLatencyDryRunReport, profile); err != nil {
		apiLatencyDryRunRecord.Outcome = volumeOutcomeFail
		apiLatencyDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-execute-api-latency-dry-run-selected-version: %v", err))
	}
	records = append(records, apiLatencyDryRunRecord)

	if volumeOpsExecuteAPILatencyCleanupCapable(profile) {
		apiLatencyConfirmedReport := volumeOpsExecuteReportPath(t, "volume-ops-execute-api-latency-confirmed-cleanup", profile, "json")
		apiLatencyConfirmedResult := runC8VoltForProfile(t, profile.Name, "volume-ops-execute-api-latency-confirmed-cleanup", "--automation", "--json", "--backoff-timeout", "30s", "--backoff-max-retries", "2", "ops", "execute", "api-latency-test", "--count", "1", "--workers", "1", "--report-file", apiLatencyConfirmedReport, "--report-format", "json")
		apiLatencyConfirmedRecord := volumeOpsExecuteAPILatencyRecord(profile, dataset, apiLatencyConfirmedResult, "volume-ops-execute-api-latency-confirmed-cleanup", "json", []string{"automation", "json", "backoff-timeout", "backoff-max-retries", "count", "workers", "report-file", "report-format"}, false, true)
		if err := validateVolumeOpsExecuteAPILatencyConfirmed(t, apiLatencyConfirmedResult, apiLatencyConfirmedReport, profile); err != nil {
			apiLatencyConfirmedRecord.Outcome = volumeOutcomeFail
			apiLatencyConfirmedRecord.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("volume-ops-execute-api-latency-confirmed-cleanup: %v", err))
		}
		records = append(records, apiLatencyConfirmedRecord)
	} else {
		records = append(records, volumeOpsExecuteAPILatencyConfirmedSkippedRecord(profile))
	}

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

func volumeOpsExecuteAPILatencyRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string, preview bool, confirmed bool) evidenceRecord {
	record := volumeOpsExecuteRecord(profile, dataset, result, "ops execute api-latency-test", scenarioName, outputMode, flags, preview, confirmed)
	record.VersionBehavior = "active-version-gated"
	return record
}

func volumeOpsExecuteAPILatencyConfirmedSkippedRecord(profile integrationProfile) evidenceRecord {
	return evidenceRecord{
		CommandPath:       "ops execute api-latency-test",
		ScenarioName:      "volume-ops-execute-api-latency-confirmed-cleanup",
		Profile:           profile.Name,
		CamundaVersion:    profile.ExpectedVersion,
		DataOwnership:     []string{volumeDataPreexisting},
		CoveredFlags:      []string{"automation", "json", "count", "workers", "report-file", "report-format"},
		OutputMode:        "json",
		Behavior:          "confirmed-cleanup-skipped",
		VersionBehavior:   "8.9-and-8.10-cleanup-capable",
		Preview:           false,
		ConfirmedMutation: false,
		SkipReason:        "selected profile is not Camunda 8.9 or 8.10 cleanup-capable",
		Outcome:           realStateOutcomeSkippedPrereq,
	}
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

func validateVolumeOpsExecuteAPILatencyDryRun(result commandResult, reportPath string, profile integrationProfile) error {
	if volumeOpsExecuteAPILatencyCleanupCapable(profile) {
		if err := requireVolumeCommandSuccess(result, "ops execute api-latency-test dry-run volume"); err != nil {
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
		var stdoutPayload volumeOpsExecuteAPILatencyPayload
		if err := decodeCommandPayload(result.Stdout, &stdoutPayload); err != nil {
			return fmt.Errorf("decode api-latency dry-run stdout payload: %w", err)
		}
		reportPayload, reportRaw, err := readVolumeOpsExecuteAPILatencyJSONReport(reportPath)
		if err != nil {
			return err
		}
		if err := validateVolumeOpsExecuteAPILatencyPlanPayload(stdoutPayload, nil, profile, 3, 2, "planned", true); err != nil {
			return err
		}
		if err := validateVolumeOpsExecuteAPILatencyPlanPayload(reportPayload, reportRaw, profile, 3, 2, "planned", true); err != nil {
			return fmt.Errorf("report payload mismatch: %w", err)
		}
		if reportPayload.Plan.RunID != stdoutPayload.Plan.RunID || reportPayload.Plan.DerivedRequestLimit != stdoutPayload.Plan.DerivedRequestLimit {
			return fmt.Errorf("api-latency dry-run stdout/report parity mismatch: run %q/%q derived %d/%d", stdoutPayload.Plan.RunID, reportPayload.Plan.RunID, stdoutPayload.Plan.DerivedRequestLimit, reportPayload.Plan.DerivedRequestLimit)
		}
		return nil
	}

	if result.Err == nil {
		return fmt.Errorf("ops execute api-latency-test dry-run unexpectedly succeeded on Camunda %s", profile.ExpectedVersion)
	}
	reportPayload, reportRaw, err := readVolumeOpsExecuteAPILatencyJSONReport(reportPath)
	if err != nil {
		return err
	}
	if err := validateVolumeOpsExecuteAPILatencyPlanPayload(reportPayload, reportRaw, profile, 3, 2, "failed", true); err != nil {
		return err
	}
	if reportPayload.Plan.Cleanup == nil || reportPayload.Plan.Cleanup.BlockReason == "" {
		return fmt.Errorf("api-latency unsupported dry-run report missing cleanup block reason: %+v", reportPayload.Plan.Cleanup)
	}
	if reportPayload.Ownership != nil || len(reportPayload.Cleanup) > 0 {
		return fmt.Errorf("api-latency unsupported dry-run reported mutation evidence: ownership=%+v cleanup=%+v", reportPayload.Ownership, reportPayload.Cleanup)
	}
	return nil
}

func validateVolumeOpsExecuteAPILatencyConfirmed(t *testing.T, result commandResult, reportPath string, profile integrationProfile) error {
	t.Helper()
	if err := requireVolumeCommandSuccess(result, "ops execute api-latency-test confirmed volume"); err != nil {
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

	var stdoutPayload volumeOpsExecuteAPILatencyPayload
	if err := decodeCommandPayload(result.Stdout, &stdoutPayload); err != nil {
		return fmt.Errorf("decode api-latency confirmed stdout payload: %w", err)
	}
	reportPayload, reportRaw, err := readVolumeOpsExecuteAPILatencyJSONReport(reportPath)
	if err != nil {
		return err
	}
	if err := validateVolumeOpsExecuteAPILatencyExecutionPayload(stdoutPayload, nil, profile); err != nil {
		return err
	}
	if err := validateVolumeOpsExecuteAPILatencyExecutionPayload(reportPayload, reportRaw, profile); err != nil {
		return fmt.Errorf("report payload mismatch: %w", err)
	}
	if reportPayload.Outcome != stdoutPayload.Outcome || reportPayload.Ownership.ProcessDefinitionKey != stdoutPayload.Ownership.ProcessDefinitionKey || strings.Join(reportPayload.Ownership.ProcessInstanceKeys, ",") != strings.Join(stdoutPayload.Ownership.ProcessInstanceKeys, ",") {
		return fmt.Errorf("api-latency confirmed stdout/report parity mismatch: outcome %q/%q processDefinitionKey %q/%q processInstanceKeys %v/%v", stdoutPayload.Outcome, reportPayload.Outcome, stdoutPayload.Ownership.ProcessDefinitionKey, reportPayload.Ownership.ProcessDefinitionKey, stdoutPayload.Ownership.ProcessInstanceKeys, reportPayload.Ownership.ProcessInstanceKeys)
	}
	if err := validateVolumeOpsExecuteAPILatencyOwnedResourcesAbsent(t, profile, stdoutPayload); err != nil {
		return err
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

func validateVolumeOpsExecuteAPILatencyPlanPayload(payload volumeOpsExecuteAPILatencyPayload, raw map[string]json.RawMessage, profile integrationProfile, count int, workers int, outcome string, dryRun bool) error {
	if payload.SchemaVersion != "ops.api-latency.v1" {
		return fmt.Errorf("api-latency schemaVersion = %q, want ops.api-latency.v1", payload.SchemaVersion)
	}
	if payload.Context.CommandName != "ops execute api-latency-test" {
		return fmt.Errorf("api-latency commandName = %q, want ops execute api-latency-test", payload.Context.CommandName)
	}
	if payload.Request.Mode != "active" || payload.Request.Count != count || payload.Request.Workers != workers || payload.Request.DryRun != dryRun {
		return fmt.Errorf("api-latency request = mode %q count %d workers %d dryRun %t, want active/%d/%d/%t", payload.Request.Mode, payload.Request.Count, payload.Request.Workers, payload.Request.DryRun, count, workers, dryRun)
	}
	if payload.Plan.Mode != "active" || payload.Plan.PrimarySampleLimit != count || payload.Plan.PrimarySampleAllocation != count {
		return fmt.Errorf("api-latency plan = mode %q limit %d allocation %d, want active/%d/%d", payload.Plan.Mode, payload.Plan.PrimarySampleLimit, payload.Plan.PrimarySampleAllocation, count, count)
	}
	if len(payload.Plan.Stages) == 0 {
		return fmt.Errorf("api-latency plan has no stages")
	}
	if payload.Plan.Stages[len(payload.Plan.Stages)-1].WorkerCount != workers {
		return fmt.Errorf("api-latency final stage worker count = %d, want %d", payload.Plan.Stages[len(payload.Plan.Stages)-1].WorkerCount, workers)
	}
	if payload.Plan.DerivedRequestLimit <= 0 {
		return fmt.Errorf("api-latency derivedRequestLimit = %d, want positive", payload.Plan.DerivedRequestLimit)
	}
	if payload.Plan.VisibilityAttemptLimit <= 0 {
		return fmt.Errorf("api-latency visibilityAttemptLimit = %d, want positive", payload.Plan.VisibilityAttemptLimit)
	}
	if payload.Plan.RunID == "" || len(payload.Plan.RunID) != 32 {
		return fmt.Errorf("api-latency runId = %q, want 128-bit hex", payload.Plan.RunID)
	}
	if payload.Plan.Fixture == nil || !payload.Plan.Fixture.Available {
		return fmt.Errorf("api-latency fixture unavailable: %+v", payload.Plan.Fixture)
	}
	if payload.Plan.Fixture.CamundaVersion != profile.ExpectedVersion {
		return fmt.Errorf("api-latency fixture Camunda version = %q, want %q", payload.Plan.Fixture.CamundaVersion, profile.ExpectedVersion)
	}
	if !strings.Contains(payload.Plan.Fixture.File, apiLatencyFixtureNameForVersion(profile.ExpectedVersion)) {
		return fmt.Errorf("api-latency fixture file = %q, want %s", payload.Plan.Fixture.File, apiLatencyFixtureNameForVersion(profile.ExpectedVersion))
	}
	if payload.Plan.Cleanup == nil {
		return fmt.Errorf("api-latency plan missing cleanup")
	}
	if payload.Outcome != outcome {
		return fmt.Errorf("api-latency outcome = %q, want %q", payload.Outcome, outcome)
	}
	if raw != nil && dryRun {
		if _, ok := raw["ownership"]; ok {
			return fmt.Errorf("api-latency dry-run report contains ownership")
		}
		if _, ok := raw["cleanup"]; ok {
			return fmt.Errorf("api-latency dry-run report contains cleanup records")
		}
	}
	return nil
}

func validateVolumeOpsExecuteAPILatencyExecutionPayload(payload volumeOpsExecuteAPILatencyPayload, raw map[string]json.RawMessage, profile integrationProfile) error {
	if err := validateVolumeOpsExecuteAPILatencyPlanPayload(payload, raw, profile, 1, 1, "completed", false); err != nil {
		return err
	}
	if payload.Ownership == nil {
		return fmt.Errorf("api-latency confirmed payload missing ownership")
	}
	if payload.Ownership.RunID != payload.Plan.RunID {
		return fmt.Errorf("api-latency ownership runId = %q, want plan runId %q", payload.Ownership.RunID, payload.Plan.RunID)
	}
	if !payload.Ownership.DeploymentSubmitted || payload.Ownership.ProcessDefinitionKey == "" || len(payload.Ownership.ProcessInstanceKeys) != 1 {
		return fmt.Errorf("api-latency ownership missing exact submitted resources: %+v", payload.Ownership)
	}
	if !strings.Contains(payload.Ownership.FixtureName, apiLatencyFixtureNameForVersion(profile.ExpectedVersion)) {
		return fmt.Errorf("api-latency ownership fixtureName = %q, want %s", payload.Ownership.FixtureName, apiLatencyFixtureNameForVersion(profile.ExpectedVersion))
	}
	if len(payload.Stages) != 1 {
		return fmt.Errorf("api-latency stage count = %d, want 1", len(payload.Stages))
	}
	stage := payload.Stages[0]
	if stage.Status != "completed" || stage.ActualMaxConcurrency > 1 || stage.PrimaryAttempts != 1 || stage.DerivedAttempts > payload.Plan.DerivedRequestLimit {
		return fmt.Errorf("api-latency confirmed stage summary = %+v, derived limit %d", stage, payload.Plan.DerivedRequestLimit)
	}
	if !stage.hasCategories("process_instance_create", "concurrent_read", "search_visibility") {
		return fmt.Errorf("api-latency confirmed stage missing active categories: %+v", stage.Categories)
	}
	if len(payload.Visibility) != 1 {
		return fmt.Errorf("api-latency visibility count = %d, want 1", len(payload.Visibility))
	}
	visibility := payload.Visibility[0]
	if visibility.ProcessInstanceKey != payload.Ownership.ProcessInstanceKeys[0] || visibility.Attempts < 1 || visibility.Attempts > visibility.AttemptLimit || !visibility.Visible {
		return fmt.Errorf("api-latency visibility evidence invalid: %+v ownership=%+v", visibility, payload.Ownership)
	}
	if len(payload.Cleanup) < 2 {
		return fmt.Errorf("api-latency cleanup count = %d, want at least process instance and process definition", len(payload.Cleanup))
	}
	if !volumeOpsExecuteAPILatencyCleanupDeleted(payload.Cleanup, "process_instance", payload.Ownership.ProcessInstanceKeys[0]) {
		return fmt.Errorf("api-latency cleanup missing deleted process instance %q: %+v", payload.Ownership.ProcessInstanceKeys[0], payload.Cleanup)
	}
	if !volumeOpsExecuteAPILatencyCleanupDeleted(payload.Cleanup, "process_definition", payload.Ownership.ProcessDefinitionKey) {
		return fmt.Errorf("api-latency cleanup missing deleted process definition %q: %+v", payload.Ownership.ProcessDefinitionKey, payload.Cleanup)
	}
	if len(payload.Findings) == 0 {
		return fmt.Errorf("api-latency findings are empty")
	}
	if raw != nil {
		if _, ok := raw["ownership"]; !ok {
			return fmt.Errorf("api-latency raw report missing ownership")
		}
		if _, ok := raw["cleanup"]; !ok {
			return fmt.Errorf("api-latency raw report missing cleanup")
		}
	}
	return nil
}

// validateVolumeOpsExecuteAPILatencyOwnedResourcesAbsent queries every exact key
// returned by the active diagnostic instead of relying on cleanup status fields.
func validateVolumeOpsExecuteAPILatencyOwnedResourcesAbsent(t *testing.T, profile integrationProfile, payload volumeOpsExecuteAPILatencyPayload) error {
	t.Helper()
	if payload.Ownership == nil {
		return fmt.Errorf("api-latency confirmed payload missing ownership for absence verification")
	}
	for _, key := range payload.Ownership.ProcessInstanceKeys {
		if err := requireProcessInstanceAbsent(t, profile, key); err != nil {
			return fmt.Errorf("api-latency owned process instance %s still present after cleanup: %w", key, err)
		}
	}
	if payload.Ownership.ProcessDefinitionKey == "" {
		return fmt.Errorf("api-latency confirmed payload missing process-definition key for absence verification")
	}
	if err := requireProcessDefinitionAbsent(t, profile, payload.Ownership.ProcessDefinitionKey); err != nil {
		return fmt.Errorf("api-latency owned process definition %s still present after cleanup: %w", payload.Ownership.ProcessDefinitionKey, err)
	}
	return nil
}

func volumeOpsExecuteAPILatencyCleanupDeleted(records []volumeOpsAPILatencyCleanupRecord, resourceType string, key string) bool {
	for _, record := range records {
		if record.ResourceType == resourceType && record.Key == key && record.Status == "deleted" {
			return true
		}
	}
	return false
}

func volumeOpsExecuteAPILatencyCleanupCapable(profile integrationProfile) bool {
	switch strings.TrimSpace(profile.ExpectedVersion) {
	case "8.9", "8.10":
		return true
	default:
		return false
	}
}

func apiLatencyFixtureNameForVersion(version string) string {
	switch strings.TrimSpace(version) {
	case "8.7":
		return "C87_SimpleUserTask.bpmn"
	case "8.8":
		return "C88_SimpleUserTask.bpmn"
	case "8.9":
		return "C89_SimpleUserTask.bpmn"
	case "8.10":
		return "C810_SimpleUserTask.bpmn"
	default:
		return "SimpleUserTask.bpmn"
	}
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

func readVolumeOpsExecuteAPILatencyJSONReport(path string) (volumeOpsExecuteAPILatencyPayload, map[string]json.RawMessage, error) {
	var payload volumeOpsExecuteAPILatencyPayload
	var raw map[string]json.RawMessage
	content, err := os.ReadFile(path)
	if err != nil {
		return payload, raw, fmt.Errorf("read api-latency JSON report: %w", err)
	}
	if err := json.Unmarshal(content, &payload); err != nil {
		return payload, raw, fmt.Errorf("decode api-latency JSON report %s: %w; content: %q", path, err, compactLogSnippet(string(content), 300))
	}
	if err := json.Unmarshal(content, &raw); err != nil {
		return payload, raw, fmt.Errorf("decode api-latency raw JSON report %s: %w", path, err)
	}
	return payload, raw, nil
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

type volumeOpsExecuteAPILatencyPayload struct {
	SchemaVersion string `json:"schemaVersion"`
	Context       struct {
		CommandName string `json:"commandName"`
	} `json:"context"`
	Request struct {
		Mode      string `json:"mode"`
		Count     int    `json:"count"`
		Workers   int    `json:"workers"`
		DryRun    bool   `json:"dryRun"`
		NoCleanup bool   `json:"noCleanup"`
	} `json:"request"`
	Plan struct {
		RunID                   string                         `json:"runId"`
		Mode                    string                         `json:"mode"`
		Stages                  []volumeOpsAPILatencyStagePlan `json:"stages"`
		PrimarySampleLimit      int                            `json:"primarySampleLimit"`
		PrimarySampleAllocation int                            `json:"primarySampleAllocation"`
		DerivedRequestLimit     int                            `json:"derivedRequestLimit"`
		VisibilityAttemptLimit  int                            `json:"visibilityAttemptLimit"`
		Fixture                 *struct {
			CamundaVersion string `json:"camundaVersion"`
			File           string `json:"file"`
			Available      bool   `json:"available"`
		} `json:"fixture"`
		Cleanup *struct {
			Requested   bool   `json:"requested"`
			Supported   bool   `json:"supported"`
			BlockReason string `json:"blockReason"`
		} `json:"cleanup"`
	} `json:"plan"`
	Stages    []volumeOpsAPILatencyStageResult `json:"stages"`
	Findings  []struct{ Code string }          `json:"findings"`
	Ownership *struct {
		RunID                string   `json:"runId"`
		FixtureName          string   `json:"fixtureName"`
		DeploymentSubmitted  bool     `json:"deploymentSubmitted"`
		ProcessDefinitionKey string   `json:"processDefinitionKey"`
		ProcessInstanceKeys  []string `json:"processInstanceKeys"`
	} `json:"ownership"`
	Visibility []struct {
		ProcessInstanceKey string `json:"processInstanceKey"`
		Attempts           int    `json:"attempts"`
		AttemptLimit       int    `json:"attemptLimit"`
		Visible            bool   `json:"visible"`
	} `json:"visibility"`
	Cleanup []volumeOpsAPILatencyCleanupRecord `json:"cleanup"`
	Outcome string                             `json:"outcome"`
}

type volumeOpsAPILatencyCleanupRecord struct {
	ResourceType string `json:"resourceType"`
	Key          string `json:"key"`
	Status       string `json:"status"`
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
