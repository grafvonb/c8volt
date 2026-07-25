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

func TestVolumeUpdateFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "update",
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

		records, err := runVolumeUpdateProcessInstanceScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "update", report.Datasets)
	writeVolumeProgressReport(t, "update", report.Records)
	writeVolumePipelineReport(t, "update", nil)
	writeVolumeOpsReportEvidence(t, "update", nil)
	writeCommandProposals(t, appendUpdateCommandGapProposals(nil))
	writeEmbeddedBPMNProposals(t, appendUpdateEmbeddedBPMNGapProposals(nil))
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume update scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeUpdateProcessInstanceScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	keys := firstNStrings(dataset.PositiveProcessInstanceKeys, 4)
	if len(keys) < 4 {
		return records, fmt.Errorf("update volume dataset for profile %q has %d positive keys, want at least 4", profile.Name, len(keys))
	}

	dryRunVar := volumeVariableName("dryRun", profile.Name)
	dryRunPayload := fmt.Sprintf(`{%q:"planned"}`, dryRunVar)
	dryRunResult := runC8VoltForProfile(t, profile.Name, "volume-update-pi-json-dry-run", "--automation", "--json", "update", "pi", "--key", keys[0], "--vars", dryRunPayload, "--dry-run")
	dryRunRecord := volumeUpdateRecord(profile, dataset, dryRunResult, "volume-update-pi-json-dry-run", "json", []string{"key", "vars", "dry-run"})
	if err := validateVolumeUpdateDryRun(t, profile, dryRunResult, keys[0], dryRunVar); err != nil {
		dryRunRecord.Outcome = volumeOutcomeFail
		dryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-update-pi-json-dry-run: %v", err))
	}
	records = append(records, dryRunRecord)

	confirmedVar := volumeVariableName("confirmed", profile.Name)
	confirmedPayload := fmt.Sprintf(`{%q:"done"}`, confirmedVar)
	confirmedResult := runC8VoltForProfile(t, profile.Name, "volume-update-pi-json-auto-confirm-workers", "--automation", "--json", "update", "pi", "--key", keys[0], "--key", keys[1], "--vars", confirmedPayload, "--workers", "2")
	confirmedRecord := volumeUpdateRecord(profile, dataset, confirmedResult, "volume-update-pi-json-auto-confirm-workers", "json", []string{"key", "vars", "workers"})
	if err := validateVolumeUpdateConfirmed(t, profile, confirmedResult, keys[:2], confirmedVar); err != nil {
		confirmedRecord.Outcome = volumeOutcomeFail
		confirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-update-pi-json-auto-confirm-workers: %v", err))
	}
	records = append(records, confirmedRecord)

	stdinVar := volumeVariableName("stdin", profile.Name)
	varsFile := writeVolumeVarsFile(t, "volume-update-vars-file", map[string]any{
		stdinVar: "done",
	})
	stdinInput := strings.Join([]string{keys[2], keys[2], keys[3]}, "\n") + "\n"
	stdinResult := runC8VoltWithInput(t, "volume-update-pi-stdin-vars-file", stdinInput, argsForProfile(profile.Name, "--automation", "--json", "update", "pi", "-", "--key", keys[2], "--vars-file", varsFile, "--no-worker-limit")...)
	stdinRecord := volumeUpdateRecord(profile, dataset, stdinResult, "volume-update-pi-stdin-vars-file", "json", []string{"key", "vars-file", "no-worker-limit", "stdin"})
	stdinRecord.StdinPath = writeVolumeStdinKeys(t, "volume-update-pi-stdin-vars-file", []string{keys[2], keys[2], keys[3]})
	if err := validateVolumeUpdateConfirmed(t, profile, stdinResult, keys[2:4], stdinVar); err != nil {
		stdinRecord.Outcome = volumeOutcomeFail
		stdinRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-update-pi-stdin-vars-file: %v", err))
	}
	records = append(records, stdinRecord)

	noWaitVar := volumeVariableName("noWait", profile.Name)
	noWaitPayload := fmt.Sprintf(`{%q:"submitted"}`, noWaitVar)
	noWaitResult := runC8VoltForProfile(t, profile.Name, "volume-update-pi-json-no-wait", "--automation", "--json", "update", "pi", "--key", keys[3], "--vars", noWaitPayload, "--no-wait")
	noWaitRecord := volumeUpdateRecord(profile, dataset, noWaitResult, "volume-update-pi-json-no-wait", "json", []string{"key", "vars", "no-wait"})
	if err := validateVolumeUpdateAccepted(noWaitResult, []string{keys[3]}); err != nil {
		noWaitRecord.Outcome = volumeOutcomeFail
		noWaitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-update-pi-json-no-wait: %v", err))
	}
	records = append(records, noWaitRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeUpdateRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string) evidenceRecord {
	record := commandEvidence("update process-instance", scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, "mutated", "retained"}
	record.ResourceKeys = append([]string(nil), dataset.PositiveProcessInstanceKeys...)
	return record
}

func validateVolumeUpdateDryRun(t *testing.T, profile integrationProfile, result commandResult, key string, variableName string) error {
	t.Helper()
	if err := requireVolumeCommandSuccess(result, "update pi dry-run volume"); err != nil {
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
	var preview struct {
		MutationSubmitted bool `json:"mutationSubmitted"`
	}
	if err := decodeCommandPayload(result.Stdout, &preview); err != nil {
		return fmt.Errorf("decode update dry-run payload: %w", err)
	}
	if preview.MutationSubmitted {
		return fmt.Errorf("dry-run payload reported mutationSubmitted=true")
	}
	return requireVariableAbsent(t, profile, key, variableName)
}

func validateVolumeUpdateConfirmed(t *testing.T, profile integrationProfile, result commandResult, keys []string, variableName string) error {
	t.Helper()
	if err := validateVolumeUpdateResultEnvelope(result, "succeeded", keys, "confirmed"); err != nil {
		return err
	}
	for _, key := range keys {
		if err := requireVariablePresent(t, profile, key, variableName); err != nil {
			return err
		}
	}
	return nil
}

func validateVolumeUpdateAccepted(result commandResult, keys []string) error {
	return validateVolumeUpdateResultEnvelope(result, "accepted", keys, "submitted")
}

func validateVolumeUpdateResultEnvelope(result commandResult, wantOutcome string, keys []string, wantStatus string) error {
	if err := requireVolumeCommandSuccess(result, "update pi volume"); err != nil {
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
			Key    string `json:"key"`
			Status string `json:"status"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode update result payload: %w", err)
	}
	got := map[string]string{}
	for _, item := range payload.Items {
		got[item.Key] = item.Status
	}
	for _, key := range keys {
		if got[key] != wantStatus {
			return fmt.Errorf("update result key %s status = %q, want %q; statuses=%v", key, got[key], wantStatus, got)
		}
	}
	return nil
}

func requireVariablePresent(t *testing.T, profile integrationProfile, key string, variableName string) error {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "volume-update-verify-variable-"+key, "--automation", "--json", "get", "pi", "--key", key, "--with-vars")
	if err := requireVolumeCommandSuccess(result, "get pi --with-vars verification"); err != nil {
		return err
	}
	if !strings.Contains(result.Stdout, variableName) {
		return fmt.Errorf("updated variable %q not visible for process instance %s", variableName, key)
	}
	return nil
}

func requireVariableAbsent(t *testing.T, profile integrationProfile, key string, variableName string) error {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "volume-update-verify-variable-absent-"+key, "--automation", "--json", "get", "pi", "--key", key, "--with-vars")
	if err := requireVolumeCommandSuccess(result, "get pi --with-vars dry-run verification"); err != nil {
		return err
	}
	if strings.Contains(result.Stdout, variableName) {
		return fmt.Errorf("dry-run variable %q became visible for process instance %s", variableName, key)
	}
	return nil
}

func writeVolumeVarsFile(t *testing.T, name string, values map[string]any) string {
	t.Helper()
	path := filepath.Join(suite.workDir, "data", sanitizeEvidenceName(name)+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create vars file dir: %v", err)
	}
	data, err := json.Marshal(values)
	if err != nil {
		t.Fatalf("marshal vars file: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write vars file: %v", err)
	}
	return path
}

func volumeVariableName(prefix string, profile string) string {
	return "c8voltIT_" + sanitizeEvidenceName(prefix) + "_" + sanitizeEvidenceName(profile) + "_" + sanitizeEvidenceName(suite.marker)
}
