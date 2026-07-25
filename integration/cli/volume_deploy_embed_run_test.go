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

func TestVolumeDeployEmbedRunFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "deploy-embed-run",
		Marker:       suite.marker,
		DatasetCount: datasetCount,
		Profiles:     profiles,
	}
	var failures []string
	for _, profile := range profiles {
		dataset, records, err := runVolumeDeployEmbedRunScenarios(t, profile, datasetCount)
		report.Datasets = append(report.Datasets, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "deploy-embed-run", report.Datasets)
	writeVolumeProgressReport(t, "deploy-embed-run", report.Records)
	writeVolumePipelineReport(t, "deploy-embed-run", nil)
	writeVolumeOpsReportEvidence(t, "deploy-embed-run", nil)
	writeCommandProposals(t, nil)
	writeEmbeddedBPMNProposals(t, nil)
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume deploy/embed/run scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeDeployEmbedRunScenarios(t *testing.T, profile integrationProfile, datasetCount int) (volumeDataset, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	dataset := volumeDataset{
		Marker:         suite.marker,
		Profile:        profile.Name,
		CamundaVersion: profile.ExpectedVersion,
		RequestedCount: datasetCount,
	}

	files, record, err := discoverEmbeddedFixtureFiles(t, profile)
	record.CommandPath = "embed list"
	record.CoveredFlags = []string{"details"}
	record.OutputMode = "json"
	records = append(records, record)
	if err != nil {
		return dataset, records, err
	}

	selection, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "SimpleUserTask.bpmn")
	if err != nil {
		return dataset, records, err
	}
	selection.BpmnProcessID = embeddedFixtureBpmnProcessID(t, selection.Path)
	dataset.PositiveFixturePath = selection.Path
	dataset.PositiveBpmnProcessID = selection.BpmnProcessID

	exportedPath, record, err := exportVolumeEmbeddedFixture(t, profile, selection)
	records = append(records, record)
	if err != nil {
		failures = append(failures, err.Error())
	}

	embedDeployments, record, err := deployEmbeddedFixture(t, profile, selection)
	record.CoveredFlags = []string{"file"}
	record.OutputMode = "json"
	records = append(records, record)
	if err != nil {
		failures = append(failures, err.Error())
	} else {
		dataset.PositiveProcessDefinitionKeys = append(dataset.PositiveProcessDefinitionKeys, processDefinitionKeys(embedDeployments)...)
	}

	if exportedPath != "" {
		deployments, deployRecords, err := runVolumeDeployProcessDefinitionScenarios(t, profile, exportedPath)
		records = append(records, deployRecords...)
		if err != nil {
			failures = append(failures, err.Error())
		}
		dataset.PositiveProcessDefinitionKeys = append(dataset.PositiveProcessDefinitionKeys, processDefinitionKeys(deployments)...)
	}

	runDeployments := embedDeployments
	if len(processDefinitionKeys(runDeployments)) == 0 {
		if len(dataset.PositiveProcessDefinitionKeys) == 0 {
			failures = append(failures, fmt.Sprintf("profile %q has no deployed process definition key for volume run scenarios", profile.Name))
		}
	} else {
		runDataset, runRecords, err := runVolumeRunProcessInstanceScenarios(t, profile, selection, runDeployments, datasetCount)
		records = append(records, runRecords...)
		if err != nil {
			failures = append(failures, err.Error())
		}
		dataset.PositiveProcessInstanceKeys = append(dataset.PositiveProcessInstanceKeys, runDataset.PositiveProcessInstanceKeys...)
		dataset.RetainedResources = append(dataset.RetainedResources, runDataset.PositiveProcessInstanceKeys...)
		dataset.CleanupRecords = append(dataset.CleanupRecords, retainedCleanupRecord(profile, runDataset.PositiveProcessInstanceKeys))
	}

	dataset.PositiveSelectors = []string{"--bpmn-process-id " + dataset.PositiveBpmnProcessID}
	if len(failures) > 0 {
		return dataset, records, errors.New(strings.Join(failures, "\n"))
	}
	return dataset, records, nil
}

func exportVolumeEmbeddedFixture(t *testing.T, profile integrationProfile, selection embeddedFixtureSelection) (string, evidenceRecord, error) {
	t.Helper()
	outDir := filepath.Join(suite.workDir, "data", "volume-deploy-embed-run", sanitizeEvidenceName(profile.Name), "fixtures")
	scenario := "volume-" + profile.Name + "-embed-export-file"
	result := runC8VoltForProfile(t, profile.Name, scenario, "embed", "export", "--file", selection.Path, "--out", outDir, "--force")
	record := commandEvidence("embed export", scenario, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = []string{"file", "out", "force"}
	record.OutputMode = "one-line"
	record.DataOwnership = []string{volumeDataSeeded}
	exportedPath := filepath.Join(outDir, filepath.FromSlash(selection.Path))
	record.ResourceKeys = []string{exportedPath}
	if err := requireVolumeCommandSuccess(result, "embed export volume"); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return "", record, err
	}
	if _, err := os.Stat(exportedPath); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return "", record, fmt.Errorf("exported fixture %s not observable: %w", exportedPath, err)
	}
	if err := requireFinalOutcomeText(result.Stderr); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return "", record, err
	}
	return exportedPath, record, nil
}

func runVolumeDeployProcessDefinitionScenarios(t *testing.T, profile integrationProfile, exportedPath string) ([]seededDeployment, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	var confirmedDeployments []seededDeployment

	confirmedResult := runC8VoltForProfile(t, profile.Name, "volume-deploy-pd-json-confirmed", "--automation", "--json", "deploy", "pd", "--file", exportedPath)
	confirmedRecord := commandEvidence("deploy process-definition", "volume-deploy-pd-json-confirmed", confirmedResult, volumeOutcomePass)
	confirmedRecord.Profile = profile.Name
	confirmedRecord.CamundaVersion = profile.ExpectedVersion
	confirmedRecord.CoveredFlags = []string{"file"}
	confirmedRecord.OutputMode = "json"
	confirmedRecord.DataOwnership = []string{volumeDataSeeded}
	if err := validateVolumeDeploymentJSON(confirmedResult, "succeeded", &confirmedDeployments); err != nil {
		confirmedRecord.Outcome = volumeOutcomeFail
		confirmedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-deploy-pd-json-confirmed: %v", err))
	}
	confirmedRecord.ResourceKeys = append(confirmedRecord.ResourceKeys, processDefinitionKeys(confirmedDeployments)...)
	records = append(records, confirmedRecord)

	noWaitResult := runC8VoltForProfile(t, profile.Name, "volume-deploy-pd-json-no-wait", "--automation", "--json", "deploy", "pd", "--file", exportedPath, "--no-wait")
	noWaitRecord := commandEvidence("deploy process-definition", "volume-deploy-pd-json-no-wait", noWaitResult, volumeOutcomePass)
	noWaitRecord.Profile = profile.Name
	noWaitRecord.CamundaVersion = profile.ExpectedVersion
	noWaitRecord.CoveredFlags = []string{"file", "no-wait"}
	noWaitRecord.OutputMode = "json"
	noWaitRecord.DataOwnership = []string{volumeDataSeeded}
	var noWaitDeployments []seededDeployment
	if err := validateVolumeDeploymentJSON(noWaitResult, "accepted", &noWaitDeployments); err != nil {
		noWaitRecord.Outcome = volumeOutcomeFail
		noWaitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-deploy-pd-json-no-wait: %v", err))
	}
	noWaitRecord.ResourceKeys = append(noWaitRecord.ResourceKeys, processDefinitionKeys(noWaitDeployments)...)
	records = append(records, noWaitRecord)

	if len(failures) > 0 {
		return confirmedDeployments, records, errors.New(strings.Join(failures, "\n"))
	}
	return confirmedDeployments, records, nil
}

func runVolumeRunProcessInstanceScenarios(t *testing.T, profile integrationProfile, selection embeddedFixtureSelection, deployments []seededDeployment, datasetCount int) (volumeDataset, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	dataset := volumeDataset{
		Marker:                      suite.marker,
		Profile:                     profile.Name,
		CamundaVersion:              profile.ExpectedVersion,
		RequestedCount:              datasetCount,
		PositiveFixturePath:         selection.Path,
		PositiveBpmnProcessID:       selection.BpmnProcessID,
		PositiveSelectors:           []string{"--bpmn-process-id " + selection.BpmnProcessID},
		RetainedResources:           nil,
		PositiveProcessInstanceKeys: nil,
	}

	humanCount := boundedVolumeRunCount(datasetCount, 4)
	humanArgs := []string{"run", "pi"}
	humanArgs = append(humanArgs, runSelectorArgs(selection, deployments)...)
	humanArgs = append(humanArgs, "--count", fmt.Sprint(humanCount), "--workers", "2", "--vars", runMarkerVars(suite.marker))
	humanResult := runC8VoltForProfile(t, profile.Name, "volume-run-pi-human-count-workers", humanArgs...)
	humanRecord := commandEvidence("run process-instance", "volume-run-pi-human-count-workers", humanResult, volumeOutcomePass)
	humanRecord.Profile = profile.Name
	humanRecord.CamundaVersion = profile.ExpectedVersion
	humanRecord.CoveredFlags = []string{"count", "workers", "vars", "pd-key"}
	humanRecord.OutputMode = "one-line"
	humanRecord.DataOwnership = []string{volumeDataSeeded, "retained"}
	if err := validateVolumeRunHumanResult(humanResult, humanCount); err != nil {
		humanRecord.Outcome = volumeOutcomeFail
		humanRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-run-pi-human-count-workers: %v", err))
	}
	humanKeys := extractNumericTokensFromOutput(humanResult.Stdout)
	humanRecord.ResourceKeys = humanKeys
	dataset.PositiveProcessInstanceKeys = append(dataset.PositiveProcessInstanceKeys, humanKeys...)
	records = append(records, humanRecord)

	keysOnlyCount := boundedVolumeRunCount(datasetCount, 2)
	keysResult := runC8VoltForProfile(t, profile.Name, "volume-run-pi-keys-only-no-worker-limit", "--keys-only", "run", "pi", "--pd-key", firstString(processDefinitionKeys(deployments)), "--count", fmt.Sprint(keysOnlyCount), "--no-worker-limit", "--vars", runMarkerVars(suite.marker))
	keysRecord := commandEvidence("run process-instance", "volume-run-pi-keys-only-no-worker-limit", keysResult, volumeOutcomePass)
	keysRecord.Profile = profile.Name
	keysRecord.CamundaVersion = profile.ExpectedVersion
	keysRecord.CoveredFlags = []string{"pd-key", "count", "no-worker-limit", "vars"}
	keysRecord.OutputMode = "keys-only"
	keysRecord.DataOwnership = []string{volumeDataSeeded, "retained"}
	keys, err := validateVolumeRunKeysOnlyResult(keysResult, keysOnlyCount)
	if err != nil {
		keysRecord.Outcome = volumeOutcomeFail
		keysRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-run-pi-keys-only-no-worker-limit: %v", err))
	}
	keysRecord.ResourceKeys = keys
	dataset.PositiveProcessInstanceKeys = append(dataset.PositiveProcessInstanceKeys, keys...)
	records = append(records, keysRecord)

	noWaitResult := runC8VoltForProfile(t, profile.Name, "volume-run-pi-json-no-wait", "--automation", "--json", "run", "pi", "--pd-key", firstString(processDefinitionKeys(deployments)), "--no-wait", "--vars", runMarkerVars(suite.marker))
	noWaitRecord := commandEvidence("run process-instance", "volume-run-pi-json-no-wait", noWaitResult, volumeOutcomePass)
	noWaitRecord.Profile = profile.Name
	noWaitRecord.CamundaVersion = profile.ExpectedVersion
	noWaitRecord.CoveredFlags = []string{"pd-key", "no-wait", "vars"}
	noWaitRecord.OutputMode = "json"
	noWaitRecord.DataOwnership = []string{volumeDataSeeded, "retained"}
	noWaitKeys, err := validateVolumeRunJSONResult(noWaitResult, 1, "accepted")
	if err != nil {
		noWaitRecord.Outcome = volumeOutcomeFail
		noWaitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-run-pi-json-no-wait: %v", err))
	}
	noWaitRecord.ResourceKeys = noWaitKeys
	dataset.PositiveProcessInstanceKeys = append(dataset.PositiveProcessInstanceKeys, noWaitKeys...)
	records = append(records, noWaitRecord)

	jsonCount := boundedVolumeRunCount(datasetCount, 3)
	jsonResult := runC8VoltForProfile(t, profile.Name, "volume-run-pi-json-count-fail-fast", "--automation", "--json", "run", "pi", "--bpmn-process-id", selection.BpmnProcessID, "--count", fmt.Sprint(jsonCount), "--fail-fast", "--vars", runMarkerVars(suite.marker))
	jsonRecord := commandEvidence("run process-instance", "volume-run-pi-json-count-fail-fast", jsonResult, volumeOutcomePass)
	jsonRecord.Profile = profile.Name
	jsonRecord.CamundaVersion = profile.ExpectedVersion
	jsonRecord.CoveredFlags = []string{"bpmn-process-id", "count", "fail-fast", "vars"}
	jsonRecord.OutputMode = "json"
	jsonRecord.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "retained"}
	jsonKeys, err := validateVolumeRunJSONResult(jsonResult, jsonCount, "succeeded")
	if err != nil {
		jsonRecord.Outcome = volumeOutcomeFail
		jsonRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-run-pi-json-count-fail-fast: %v", err))
	}
	jsonRecord.ResourceKeys = jsonKeys
	dataset.PositiveProcessInstanceKeys = append(dataset.PositiveProcessInstanceKeys, jsonKeys...)
	records = append(records, jsonRecord)

	if len(failures) > 0 {
		return dataset, records, errors.New(strings.Join(failures, "\n"))
	}
	return dataset, records, nil
}

func validateVolumeDeploymentJSON(result commandResult, wantOutcome string, deployments *[]seededDeployment) error {
	if err := requireVolumeCommandSuccess(result, "deploy pd volume"); err != nil {
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
	if err := decodeCommandPayload(result.Stdout, deployments); err != nil {
		return fmt.Errorf("decode deployment payload: %w", err)
	}
	if len(processDefinitionKeys(*deployments)) == 0 {
		return fmt.Errorf("deployment payload does not include process definition keys")
	}
	return nil
}

func validateVolumeRunHumanResult(result commandResult, wantCount int) error {
	if err := requireVolumeCommandSuccess(result, "run pi human volume"); err != nil {
		return err
	}
	if err := requireFinalOutcomeText(result.Stdout); err != nil {
		return err
	}
	if !strings.Contains(result.Stdout, fmt.Sprintf("found: %d", wantCount)) {
		return fmt.Errorf("run pi human output missing found count %d: %q", wantCount, compactLogSnippet(result.Stdout, 300))
	}
	if !strings.Contains(strings.ToLower(result.Stderr), "waiting for pi") {
		return fmt.Errorf("run pi human stderr missing visible wait progress: %q", compactLogSnippet(result.Stderr, 300))
	}
	if got := len(extractNumericTokensFromOutput(result.Stdout)); got < wantCount {
		return fmt.Errorf("run pi human output exposed %d numeric keys, want at least %d", got, wantCount)
	}
	return nil
}

func validateVolumeRunKeysOnlyResult(result commandResult, wantCount int) ([]string, error) {
	if err := requireVolumeCommandSuccess(result, "run pi keys-only volume"); err != nil {
		return nil, err
	}
	if err := requireVolumeKeysOnly(result.Stdout); err != nil {
		return nil, err
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return nil, err
	}
	keys := splitNonEmptyLines(result.Stdout)
	if len(keys) != wantCount {
		return keys, fmt.Errorf("run pi keys-only returned %d keys, want %d", len(keys), wantCount)
	}
	return keys, nil
}

func validateVolumeRunJSONResult(result commandResult, wantCount int, wantOutcome string) ([]string, error) {
	if err := requireVolumeCommandSuccess(result, "run pi JSON volume"); err != nil {
		return nil, err
	}
	if err := requireVolumeJSON(result.Stdout); err != nil {
		return nil, err
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return nil, err
	}
	if err := requireVolumeEnvelopeOutcome(result.Stdout, wantOutcome); err != nil {
		return nil, err
	}
	if wantOutcome == "accepted" {
		if err := requireNoWaitOrSubmittedText(result.Stdout); err != nil {
			return nil, err
		}
	}
	var instances seededProcessInstances
	if err := decodeCommandPayload(result.Stdout, &instances); err != nil {
		return nil, fmt.Errorf("decode run pi JSON payload: %w", err)
	}
	keys := processInstanceKeys(instances)
	if len(keys) != wantCount {
		return keys, fmt.Errorf("run pi JSON returned %d keys, want %d", len(keys), wantCount)
	}
	return keys, nil
}

func requireVolumeEnvelopeOutcome(output string, wantOutcome string) error {
	var envelope struct {
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal([]byte(output), &envelope); err != nil {
		return fmt.Errorf("decode command envelope: %w", err)
	}
	if envelope.Outcome != wantOutcome {
		return fmt.Errorf("command outcome = %q, want %q", envelope.Outcome, wantOutcome)
	}
	return nil
}

func boundedVolumeRunCount(datasetCount int, maxCount int) int {
	if datasetCount < maxCount {
		return datasetCount
	}
	return maxCount
}

func extractNumericTokensFromOutput(output string) []string {
	var keys []string
	seen := map[string]struct{}{}
	for _, field := range strings.FieldsFunc(output, func(r rune) bool {
		return r < '0' || r > '9'
	}) {
		if len(field) < 10 {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		keys = append(keys, field)
	}
	return keys
}
