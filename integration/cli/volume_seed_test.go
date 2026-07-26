// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"fmt"
	"strings"
	"testing"
)

func seedVolumeProcessInstanceDataset(t *testing.T, profile integrationProfile, requestedCount int) (volumeDataset, []evidenceRecord, error) {
	t.Helper()
	files, record, err := discoverEmbeddedFixtureFiles(t, profile)
	records := []evidenceRecord{record}
	dataset := volumeDataset{
		Marker:         suite.marker,
		Profile:        profile.Name,
		CamundaVersion: profile.ExpectedVersion,
		RequestedCount: requestedCount,
	}
	if err != nil {
		return dataset, records, err
	}

	positive, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "SimpleUserTask.bpmn")
	if err != nil {
		record.Outcome = "blocked"
		record.FailureClass = "missing_fixture_support"
		return dataset, records, err
	}
	positive.BpmnProcessID = embeddedFixtureBpmnProcessID(t, positive.Path)
	dataset.PositiveFixturePath = positive.Path
	dataset.PositiveBpmnProcessID = positive.BpmnProcessID

	negative, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "DoubleUserTask.bpmn")
	if err != nil {
		record.Outcome = "blocked"
		record.FailureClass = "missing_fixture_support"
		return dataset, records, err
	}
	negative.BpmnProcessID = embeddedFixtureBpmnProcessID(t, negative.Path)
	dataset.NegativeFixturePath = negative.Path
	dataset.NegativeBpmnProcessID = negative.BpmnProcessID

	positiveDeployments, deployRecord, err := deployEmbeddedFixture(t, profile, positive)
	records = append(records, deployRecord)
	if err != nil {
		return dataset, records, err
	}
	dataset.PositiveProcessDefinitionKeys = processDefinitionKeys(positiveDeployments)

	negativeDeployments, deployNegativeRecord, err := deployEmbeddedFixture(t, profile, negative)
	records = append(records, deployNegativeRecord)
	if err != nil {
		return dataset, records, err
	}
	dataset.NegativeProcessDefinitionKeys = processDefinitionKeys(negativeDeployments)

	positiveInstances, runPositiveRecord, err := runVolumeProcessInstances(t, profile, positive, positiveDeployments, requestedCount, "positive")
	records = append(records, runPositiveRecord)
	if err != nil {
		return dataset, records, err
	}
	dataset.PositiveProcessInstanceKeys = processInstanceKeys(positiveInstances)

	negativeInstances, runNegativeRecord, err := runVolumeProcessInstances(t, profile, negative, negativeDeployments, 1, "negative")
	records = append(records, runNegativeRecord)
	if err != nil {
		return dataset, records, err
	}
	dataset.NegativeProcessInstanceKeys = processInstanceKeys(negativeInstances)

	dataset.PositiveSelectors = []string{"--bpmn-process-id " + dataset.PositiveBpmnProcessID}
	dataset.NegativeSelectors = []string{"--bpmn-process-id " + dataset.NegativeBpmnProcessID}
	dataset.RetainedResources = dataset.allProcessInstanceKeys()
	dataset.CleanupRecords = []cleanupRecord{retainedCleanupRecord(profile, dataset.allProcessInstanceKeys())}
	return dataset, records, nil
}

func discoverEmbeddedFixtureFiles(t *testing.T, profile integrationProfile) ([]string, evidenceRecord, error) {
	t.Helper()
	scenario := "volume-" + profile.Name + "-embed-list"
	result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "embed", "list", "--details")
	record := commandEvidence("embed list", scenario, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.DataOwnership = []string{volumeDataSeeded}
	if result.Err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureHarness
		return nil, record, fmt.Errorf("embed list failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}

	var files []string
	if err := decodeCommandPayload(result.Stdout, &files); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return nil, record, fmt.Errorf("decode embed list output for profile %q: %w", profile.Name, err)
	}
	record.ResourceKeys = append([]string(nil), files...)
	return files, record, nil
}

func selectEmbeddedFixtureBySuffix(expectedVersion string, files []string, suffix string) (embeddedFixtureSelection, error) {
	prefix, err := embeddedFixturePrefix(expectedVersion)
	if err != nil {
		return embeddedFixtureSelection{}, err
	}
	wantSuffix := prefix + suffix
	for _, file := range files {
		if strings.HasSuffix(file, wantSuffix) {
			return embeddedFixtureSelection{Path: file}, nil
		}
	}
	return embeddedFixtureSelection{}, fmt.Errorf("no embedded %s fixture found for Camunda version %q", suffix, expectedVersion)
}

func runVolumeProcessInstances(t *testing.T, profile integrationProfile, selection embeddedFixtureSelection, deployments []seededDeployment, count int, label string) (seededProcessInstances, evidenceRecord, error) {
	t.Helper()
	args := []string{"--automation", "--json", "run", "process-instance"}
	args = append(args, runSelectorArgs(selection, deployments)...)
	args = append(args, "--count", fmt.Sprint(count), "--vars", runMarkerVars(suite.marker))

	scenario := fmt.Sprintf("volume-%s-run-%s-process-instances-%d", profile.Name, label, count)
	result := runC8VoltForProfile(t, profile.Name, scenario, args...)
	record := commandEvidence("run process-instance", scenario, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.DataOwnership = []string{volumeDataSeeded, "retained"}
	record.CoveredFlags = []string{"count", "vars"}
	record.OutputMode = "json"
	if result.Err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return seededProcessInstances{}, record, fmt.Errorf("run process-instance volume seed failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}

	var instances seededProcessInstances
	if err := decodeCommandPayload(result.Stdout, &instances); err != nil {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return seededProcessInstances{}, record, fmt.Errorf("decode volume run process-instance output for profile %q: %w", profile.Name, err)
	}
	keys := processInstanceKeys(instances)
	if len(keys) != count {
		record.Outcome = volumeOutcomeFail
		record.FailureClass = volumeFailureProduct
		return seededProcessInstances{}, record, fmt.Errorf("run process-instance for profile %q returned %d keys, want %d", profile.Name, len(keys), count)
	}
	record.ResourceKeys = keys
	return instances, record, nil
}
