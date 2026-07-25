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

type volumeIncidentDataset struct {
	FixturePath           string   `json:"fixturePath,omitempty"`
	BpmnProcessID         string   `json:"bpmnProcessId,omitempty"`
	ProcessDefinitionKeys []string `json:"processDefinitionKeys,omitempty"`
	ProcessInstanceKeys   []string `json:"processInstanceKeys,omitempty"`
	IncidentKeys          []string `json:"incidentKeys,omitempty"`
}

func TestVolumeExpectResolveFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "expect-resolve",
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

		incidentDataset, incidentRecords, err := seedVolumeIncidentDataset(t, profile, 2)
		report.Records = append(report.Records, incidentRecords...)
		if err != nil {
			failures = append(failures, err.Error())
			continue
		}

		records, err := runVolumeExpectResolveScenarios(t, profile, dataset, incidentDataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "expect-resolve", report.Datasets)
	writeVolumeProgressReport(t, "expect-resolve", report.Records)
	writeVolumePipelineReport(t, "expect-resolve", report.Records)
	writeVolumeOpsReportEvidence(t, "expect-resolve", nil)
	writeCommandProposals(t, appendExpectResolveCommandGapProposals(nil))
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume expect/resolve scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

// appendExpectResolveCommandGapProposals records consistency gaps found while
// validating expect/resolve output semantics.
func appendExpectResolveCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	return registerDirectCamundaSetupFallback(proposals,
		"state-only expect process-instance JSON rows with stable key and ok fields",
		"expect process-instance state-only machine-output identity coverage",
		[]string{"expect process-instance"},
		supportedProposalVersions(),
		"Operators can correlate every expect result row with the requested key consistently across state-only and incident expectation modes.",
	)
}

func seedVolumeIncidentDataset(t *testing.T, profile integrationProfile, count int) (volumeIncidentDataset, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	files, record, err := discoverEmbeddedFixtureFiles(t, profile)
	records = append(records, record)
	if err != nil {
		return volumeIncidentDataset{}, records, err
	}

	selection, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "SimpleUserTaskWithIncident.bpmn")
	if err != nil {
		record.Outcome = "blocked"
		record.FailureClass = "missing_fixture_support"
		return volumeIncidentDataset{}, records, err
	}
	selection.BpmnProcessID = embeddedFixtureBpmnProcessID(t, selection.Path)

	deployments, deployRecord, err := deployEmbeddedFixture(t, profile, selection)
	deployRecord.CoveredFlags = []string{"file"}
	deployRecord.OutputMode = "json"
	records = append(records, deployRecord)
	if err != nil {
		return volumeIncidentDataset{}, records, err
	}

	instances, runRecord, err := runVolumeProcessInstances(t, profile, selection, deployments, count, "incident")
	records = append(records, runRecord)
	if err != nil {
		return volumeIncidentDataset{}, records, err
	}

	dataset := volumeIncidentDataset{
		FixturePath:           selection.Path,
		BpmnProcessID:         selection.BpmnProcessID,
		ProcessDefinitionKeys: processDefinitionKeys(deployments),
		ProcessInstanceKeys:   processInstanceKeys(instances),
	}
	return dataset, records, nil
}

func runVolumeExpectResolveScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset, incidentDataset volumeIncidentDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	activeKeys := firstNStrings(dataset.PositiveProcessInstanceKeys, 2)
	incidentPIKeys := firstNStrings(incidentDataset.ProcessInstanceKeys, 2)
	if len(activeKeys) < 2 {
		return records, fmt.Errorf("expect/resolve volume dataset for profile %q has %d active keys, want at least 2", profile.Name, len(activeKeys))
	}
	if len(incidentPIKeys) < 2 {
		return records, fmt.Errorf("expect/resolve volume incident dataset for profile %q has %d incident keys, want at least 2", profile.Name, len(incidentPIKeys))
	}

	stateResult := runC8VoltForProfile(t, profile.Name, "volume-expect-pi-state-json-workers", "--json", "expect", "pi", "--key", activeKeys[0], "--key", activeKeys[1], "--state", "active", "--workers", "2")
	stateRecord := volumeExpectResolveRecord(profile, "expect process-instance", stateResult, "volume-expect-pi-state-json-workers", "json", []string{"key", "state", "workers"}, dataset.PositiveProcessInstanceKeys)
	if err := validateVolumeExpectState(stateResult, activeKeys, "ACTIVE"); err != nil {
		stateRecord.Outcome = volumeOutcomeFail
		stateRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-expect-pi-state-json-workers: %v", err))
	}
	records = append(records, stateRecord)

	combinedResult := runC8VoltForProfile(t, profile.Name, "volume-expect-pi-state-incident-false", "--json", "expect", "pi", "--key", activeKeys[0], "--state", "active", "--incident", "false", "--no-worker-limit")
	combinedRecord := volumeExpectResolveRecord(profile, "expect process-instance", combinedResult, "volume-expect-pi-state-incident-false", "json", []string{"key", "state", "incident", "no-worker-limit"}, []string{activeKeys[0]})
	if err := validateVolumeExpectIncident(combinedResult, []string{activeKeys[0]}, false); err != nil {
		combinedRecord.Outcome = volumeOutcomeFail
		combinedRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-expect-pi-state-incident-false: %v", err))
	}
	records = append(records, combinedRecord)

	stdin := incidentPIKeys[0] + "\n"
	incidentExpectResult := runC8VoltWithInput(t, "volume-expect-pi-stdin-incident-true", stdin, argsForProfile(profile.Name, "--json", "expect", "pi", "--incident", "true", "-")...)
	incidentExpectRecord := volumeExpectResolveRecord(profile, "expect process-instance", incidentExpectResult, "volume-expect-pi-stdin-incident-true", "json", []string{"stdin", "incident"}, []string{incidentPIKeys[0]})
	incidentExpectRecord.StdinPath = writeVolumeStdinKeys(t, "volume-expect-pi-stdin-incident-true", []string{incidentPIKeys[0]})
	if err := validateVolumeExpectIncident(incidentExpectResult, []string{incidentPIKeys[0]}, true); err != nil {
		incidentExpectRecord.Outcome = volumeOutcomeFail
		incidentExpectRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-expect-pi-stdin-incident-true: %v", err))
	}
	records = append(records, incidentExpectRecord)

	incidentKeys, incidentDiscoveryRecord, err := discoverVolumeIncidentKeys(t, profile, incidentDataset, incidentPIKeys)
	records = append(records, incidentDiscoveryRecord)
	if err != nil {
		failures = append(failures, fmt.Sprintf("volume-get-incident-for-resolve: %v", err))
	} else if len(incidentKeys) == 0 {
		failures = append(failures, "volume-get-incident-for-resolve: no active incident keys discovered")
	} else {
		incidentDataset.IncidentKeys = incidentKeys

		dryRunResult := runC8VoltForProfile(t, profile.Name, "volume-resolve-incident-json-dry-run", "--automation", "--json", "resolve", "incident", "--key", incidentKeys[0], "--dry-run", "--workers", "1")
		dryRunRecord := volumeExpectResolveRecord(profile, "resolve incident", dryRunResult, "volume-resolve-incident-json-dry-run", "json", []string{"key", "dry-run", "workers"}, incidentKeys)
		dryRunRecord.Preview = true
		if err := validateVolumeIncidentResolveDryRun(dryRunResult, []string{incidentKeys[0]}); err != nil {
			dryRunRecord.Outcome = volumeOutcomeFail
			dryRunRecord.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("volume-resolve-incident-json-dry-run: %v", err))
		}
		records = append(records, dryRunRecord)

		noWaitResult := runC8VoltForProfile(t, profile.Name, "volume-resolve-incident-json-no-wait", "--automation", "--json", "resolve", "incident", "--key", incidentKeys[0], "--no-wait")
		noWaitRecord := volumeExpectResolveRecord(profile, "resolve incident", noWaitResult, "volume-resolve-incident-json-no-wait", "json", []string{"key", "no-wait"}, []string{incidentKeys[0]})
		noWaitRecord.ConfirmedMutation = true
		if err := validateVolumeIncidentResolveSubmitted(noWaitResult, []string{incidentKeys[0]}); err != nil {
			noWaitRecord.Outcome = volumeOutcomeFail
			noWaitRecord.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("volume-resolve-incident-json-no-wait: %v", err))
		}
		records = append(records, noWaitRecord)
	}

	piDryRunResult := runC8VoltWithInput(t, "volume-resolve-pi-stdin-dry-run", incidentPIKeys[1]+"\n", argsForProfile(profile.Name, "--automation", "--json", "resolve", "pi", "-", "--dry-run", "--no-worker-limit")...)
	piDryRunRecord := volumeExpectResolveRecord(profile, "resolve process-instance", piDryRunResult, "volume-resolve-pi-stdin-dry-run", "json", []string{"stdin", "dry-run", "no-worker-limit"}, []string{incidentPIKeys[1]})
	piDryRunRecord.StdinPath = writeVolumeStdinKeys(t, "volume-resolve-pi-stdin-dry-run", []string{incidentPIKeys[1]})
	piDryRunRecord.Preview = true
	if err := validateVolumeProcessInstanceResolveDryRun(piDryRunResult, []string{incidentPIKeys[1]}); err != nil {
		piDryRunRecord.Outcome = volumeOutcomeFail
		piDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-resolve-pi-stdin-dry-run: %v", err))
	}
	records = append(records, piDryRunRecord)

	piNoWaitResult := runC8VoltForProfile(t, profile.Name, "volume-resolve-pi-json-no-wait", "--automation", "--json", "resolve", "pi", "--key", incidentPIKeys[1], "--no-wait", "--workers", "1")
	piNoWaitRecord := volumeExpectResolveRecord(profile, "resolve process-instance", piNoWaitResult, "volume-resolve-pi-json-no-wait", "json", []string{"key", "no-wait", "workers"}, []string{incidentPIKeys[1]})
	piNoWaitRecord.ConfirmedMutation = true
	if err := validateVolumeProcessInstanceResolveSubmitted(piNoWaitResult, []string{incidentPIKeys[1]}); err != nil {
		piNoWaitRecord.Outcome = volumeOutcomeFail
		piNoWaitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-resolve-pi-json-no-wait: %v", err))
	}
	records = append(records, piNoWaitRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeExpectResolveRecord(profile integrationProfile, commandPath string, result commandResult, scenarioName string, outputMode string, flags []string, keys []string) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, "mutated", "retained"}
	record.ResourceKeys = append([]string(nil), keys...)
	return record
}

func discoverVolumeIncidentKeys(t *testing.T, profile integrationProfile, dataset volumeIncidentDataset, piKeys []string) ([]string, evidenceRecord, error) {
	t.Helper()
	scenario := "volume-get-incident-for-resolve"
	result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "get", "incident", "--bpmn-process-id", dataset.BpmnProcessID, "--state", "active", "--limit", "5", "--batch-size", "1")
	record := volumeExpectResolveRecord(profile, "get incident", result, scenario, "json", []string{"bpmn-process-id", "state", "limit", "batch-size"}, piKeys)
	if err := requireVolumeCommandSuccess(result, "get incident for resolve volume"); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return nil, record, err
	}
	if err := requireVolumeJSON(result.Stdout); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return nil, record, err
	}
	var incidents struct {
		Items []struct {
			IncidentKey        string `json:"incidentKey"`
			ProcessInstanceKey string `json:"processInstanceKey"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &incidents); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return nil, record, fmt.Errorf("decode incident discovery payload: %w", err)
	}
	piSet := map[string]struct{}{}
	for _, key := range piKeys {
		piSet[key] = struct{}{}
	}
	var keys []string
	for _, item := range incidents.Items {
		if _, ok := piSet[item.ProcessInstanceKey]; ok && item.IncidentKey != "" {
			keys = append(keys, item.IncidentKey)
		}
	}
	record.ResourceKeys = append(record.ResourceKeys, keys...)
	return uniqueSortedStrings(keys), record, nil
}

func validateVolumeExpectState(result commandResult, keys []string, wantState string) error {
	if err := validateVolumeExpectEnvelope(result, keys); err != nil {
		return err
	}
	var payload struct {
		Items []struct {
			Key   string `json:"key"`
			Ok    bool   `json:"ok"`
			State string `json:"state"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode expect state payload: %w", err)
	}
	if len(payload.Items) < len(keys) {
		return fmt.Errorf("expect state returned %d items, want at least %d", len(payload.Items), len(keys))
	}
	states := map[string]string{}
	keyed := true
	for i, item := range payload.Items {
		if item.State != wantState {
			return fmt.Errorf("expect state item %d state = %q, want %q", i, item.State, wantState)
		}
		if item.Key == "" {
			keyed = false
			continue
		}
		if item.Ok {
			states[item.Key] = item.State
		}
	}
	if !keyed {
		return nil
	}
	for _, key := range keys {
		if states[key] != wantState {
			return fmt.Errorf("expect state key %s = %q, want %q; states=%v", key, states[key], wantState, states)
		}
	}
	return nil
}

func validateVolumeExpectIncident(result commandResult, keys []string, wantIncident bool) error {
	if err := validateVolumeExpectEnvelope(result, keys); err != nil {
		return err
	}
	want := "false"
	if wantIncident {
		want = "true"
	}
	if !strings.Contains(result.Stdout, `"incident": `+want) {
		return fmt.Errorf("expect incident payload does not show incident=%s: %q", want, compactLogSnippet(result.Stdout, 300))
	}
	return nil
}

func validateVolumeExpectEnvelope(result commandResult, keys []string) error {
	if err := requireVolumeCommandSuccess(result, "expect pi volume"); err != nil {
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
		Items []struct {
			Key string `json:"key"`
			Ok  bool   `json:"ok"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode expect payload: %w", err)
	}
	got := map[string]bool{}
	unkeyed := 0
	for _, item := range payload.Items {
		if item.Key == "" {
			unkeyed++
			continue
		}
		got[item.Key] = item.Ok
	}
	if len(got) == 0 && unkeyed >= len(keys) {
		return nil
	}
	for _, key := range keys {
		if !got[key] {
			return fmt.Errorf("expect result missing successful key %s; got=%v", key, got)
		}
	}
	return nil
}

func validateVolumeIncidentResolveDryRun(result commandResult, incidentKeys []string) error {
	if err := validateVolumeIncidentResolveEnvelope(result, "succeeded", incidentKeys); err != nil {
		return err
	}
	var payload struct {
		DryRun            bool `json:"dryRun"`
		MutationSubmitted bool `json:"mutationSubmitted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve incident dry-run payload: %w", err)
	}
	if !payload.DryRun {
		return fmt.Errorf("resolve incident dry-run payload dryRun=false")
	}
	if payload.MutationSubmitted {
		return fmt.Errorf("resolve incident dry-run payload mutationSubmitted=true")
	}
	return nil
}

func validateVolumeIncidentResolveSubmitted(result commandResult, incidentKeys []string) error {
	if err := validateVolumeIncidentResolveEnvelope(result, "accepted", incidentKeys); err != nil {
		return err
	}
	var payload struct {
		MutationSubmitted bool `json:"mutationSubmitted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve incident submitted payload: %w", err)
	}
	if !payload.MutationSubmitted {
		return fmt.Errorf("resolve incident submitted payload mutationSubmitted=false")
	}
	return nil
}

func validateVolumeIncidentResolveEnvelope(result commandResult, wantOutcome string, incidentKeys []string) error {
	if err := requireVolumeCommandSuccess(result, "resolve incident volume"); err != nil {
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
			IncidentKey string `json:"incidentKey"`
			Status      string `json:"status"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve incident payload: %w", err)
	}
	got := map[string]string{}
	for _, item := range payload.Items {
		got[item.IncidentKey] = item.Status
	}
	for _, key := range incidentKeys {
		if got[key] == "" {
			return fmt.Errorf("resolve incident result missing key %s; got=%v", key, got)
		}
	}
	return nil
}

func validateVolumeProcessInstanceResolveDryRun(result commandResult, piKeys []string) error {
	if err := validateVolumeProcessInstanceResolveEnvelope(result, "succeeded", piKeys); err != nil {
		return err
	}
	var payload struct {
		DryRun            bool `json:"dryRun"`
		MutationSubmitted bool `json:"mutationSubmitted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve pi dry-run payload: %w", err)
	}
	if !payload.DryRun {
		return fmt.Errorf("resolve pi dry-run payload dryRun=false")
	}
	if payload.MutationSubmitted {
		return fmt.Errorf("resolve pi dry-run payload mutationSubmitted=true")
	}
	return nil
}

func validateVolumeProcessInstanceResolveSubmitted(result commandResult, piKeys []string) error {
	if err := validateVolumeProcessInstanceResolveEnvelope(result, "accepted", piKeys); err != nil {
		return err
	}
	var payload struct {
		MutationSubmitted bool `json:"mutationSubmitted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve pi submitted payload: %w", err)
	}
	if !payload.MutationSubmitted {
		return fmt.Errorf("resolve pi submitted payload mutationSubmitted=false")
	}
	return nil
}

func validateVolumeProcessInstanceResolveEnvelope(result commandResult, wantOutcome string, piKeys []string) error {
	if err := requireVolumeCommandSuccess(result, "resolve pi volume"); err != nil {
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
			ProcessInstanceKey string `json:"processInstanceKey"`
			Status             string `json:"status"`
		} `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode resolve pi payload: %w", err)
	}
	got := map[string]string{}
	for _, item := range payload.Items {
		got[item.ProcessInstanceKey] = item.Status
	}
	for _, key := range piKeys {
		if got[key] == "" {
			return fmt.Errorf("resolve pi result missing key %s; got=%v", key, got)
		}
	}
	return nil
}
