// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

type seededDataReport struct {
	Marker   string              `json:"marker"`
	Profiles []seededProfileData `json:"profiles"`
	Records  []evidenceRecord    `json:"records"`
	Cleanup  []cleanupRecord     `json:"cleanup"`
}

type seededProfileData struct {
	Profile               string   `json:"profile"`
	CamundaVersion        string   `json:"camundaVersion"`
	FixturePath           string   `json:"fixturePath"`
	BpmnProcessID         string   `json:"bpmnProcessId"`
	DeploymentKeys        []string `json:"deploymentKeys,omitempty"`
	ProcessDefinitionKeys []string `json:"processDefinitionKeys,omitempty"`
	ProcessInstanceKeys   []string `json:"processInstanceKeys,omitempty"`
	ResourceIDs           []string `json:"resourceIds,omitempty"`
}

type cleanupRecord struct {
	Profile             string   `json:"profile"`
	ProcessInstanceKeys []string `json:"processInstanceKeys,omitempty"`
	Status              string   `json:"status"`
	Attempted           bool     `json:"attempted"`
	Reason              string   `json:"reason,omitempty"`
}

type embeddedFixtureSelection struct {
	Path          string `json:"path"`
	BpmnProcessID string `json:"bpmnProcessId"`
}

type seededDeployment struct {
	Key               string `json:"key"`
	DefinitionID      string `json:"processDefinitionId,omitempty"`
	DefinitionKey     string `json:"processDefinitionKey,omitempty"`
	DefinitionVersion int32  `json:"processDefinitionVersion,omitempty"`
	ResourceName      string `json:"resourceName,omitempty"`
	TenantID          string `json:"tenantId,omitempty"`
}

type seededProcessInstances struct {
	Total int32                   `json:"total,omitempty"`
	Items []seededProcessInstance `json:"items,omitempty"`
}

type seededProcessInstance struct {
	Key                  string `json:"key,omitempty"`
	BpmnProcessID        string `json:"bpmnProcessId,omitempty"`
	ProcessDefinitionKey string `json:"processDefinitionKey,omitempty"`
	State                string `json:"state,omitempty"`
	TenantID             string `json:"tenantId,omitempty"`
}

func TestDeployEmbedRunFamily(t *testing.T) {
	runFamilyCoverageScenarios(t, "deploy", []string{
		"deploy",
		"deploy process-definition",
	})
	runBehavioralCoverageScenarios(t, "deploy")
	runFamilyCoverageScenarios(t, "embed", []string{
		"embed",
		"embed deploy",
		"embed export",
		"embed list",
	})
	runBehavioralCoverageScenarios(t, "embed")
	runFamilyCoverageScenarios(t, "run", []string{
		"run",
		"run process-instance",
	})
	runBehavioralCoverageScenarios(t, "run")
}

// TestSeededData proves selected profiles can discover, deploy, run, and re-read suite-owned data.
func TestSeededData(t *testing.T) {
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := seededDataReport{Marker: suite.marker}
	var failures []string
	for _, profile := range profiles {
		data, records, cleanup, err := seedProfileData(t, profile)
		report.Profiles = append(report.Profiles, data)
		report.Records = append(report.Records, records...)
		report.Cleanup = append(report.Cleanup, cleanup...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeDataEvidence(t, "seeded-data.json", report)
	writeEvidenceRecords(t, "seeded-data-records.json", report.Records)
	requireSeededEvidence(t, report.Records)
	if len(failures) > 0 {
		t.Fatalf("seeded data setup failed:\n%s", strings.Join(failures, "\n"))
	}
}

// requireProfilesReady keeps destructive setup behind the same real-profile gate as the read-only slice.
func requireProfilesReady(t *testing.T, profiles []integrationProfile) error {
	t.Helper()
	var failures []string
	for _, profile := range profiles {
		if _, err := checkProfileReadiness(t, profile); err != nil {
			failures = append(failures, err.Error())
		}
	}
	if len(failures) > 0 {
		return fmt.Errorf("profile readiness failed before seeded-data mutation:\n%s", strings.Join(failures, "\n"))
	}
	return nil
}

// seedProfileData runs the command-created setup path for one selected disposable profile.
func seedProfileData(t *testing.T, profile integrationProfile) (seededProfileData, []evidenceRecord, []cleanupRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var cleanup []cleanupRecord

	selection, record, err := discoverEmbeddedFixture(t, profile)
	records = append(records, record)
	data := seededProfileData{
		Profile:        profile.Name,
		CamundaVersion: profile.ExpectedVersion,
		FixturePath:    selection.Path,
		BpmnProcessID:  selection.BpmnProcessID,
	}
	if err != nil {
		return data, records, cleanup, err
	}

	deployments, record, err := deployEmbeddedFixture(t, profile, selection)
	records = append(records, record)
	data.DeploymentKeys = deploymentKeys(deployments)
	data.ProcessDefinitionKeys = processDefinitionKeys(deployments)
	data.ResourceIDs = deploymentResourceIDs(deployments, selection)
	if err != nil {
		return data, records, cleanup, err
	}

	instances, record, err := runSeededProcessInstance(t, profile, selection, deployments)
	records = append(records, record)
	data.ProcessInstanceKeys = processInstanceKeys(instances)
	cleanup = append(cleanup, retainedCleanupRecord(profile, data.ProcessInstanceKeys))
	if err != nil {
		return data, records, cleanup, err
	}

	record, err = assertSeededProcessInstancesObservable(t, profile, data.ProcessInstanceKeys)
	records = append(records, record)
	if err != nil {
		return data, records, cleanup, err
	}
	return data, records, cleanup, nil
}

// discoverEmbeddedFixture selects a version-prefixed long-running fixture from embed list output.
func discoverEmbeddedFixture(t *testing.T, profile integrationProfile) (embeddedFixtureSelection, evidenceRecord, error) {
	t.Helper()
	scenario := "seeded-" + profile.Name + "-embed-list"
	result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "embed", "list", "--details")
	record := commandEvidence("embed list", scenario, result, "pass")
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	if result.Err != nil {
		record.Outcome = "fail"
		record.FailureClass = "harness_setup"
		return embeddedFixtureSelection{}, record, fmt.Errorf("embed list failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}

	var files []string
	if err := decodeCommandPayload(result.Stdout, &files); err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return embeddedFixtureSelection{}, record, fmt.Errorf("decode embed list output for profile %q: %w", profile.Name, err)
	}
	selection, err := selectEmbeddedSeedFixture(profile.ExpectedVersion, files)
	if err != nil {
		record.Outcome = "blocked"
		record.FailureClass = "missing_fixture_support"
		return embeddedFixtureSelection{}, record, err
	}
	selection.BpmnProcessID = embeddedFixtureBpmnProcessID(t, selection.Path)
	record.DataOwnership = []string{"seeded"}
	record.ResourceKeys = []string{selection.Path}
	return selection, record, nil
}

// deployEmbeddedFixture deploys the selected embedded BPMN through the CLI.
func deployEmbeddedFixture(t *testing.T, profile integrationProfile, selection embeddedFixtureSelection) ([]seededDeployment, evidenceRecord, error) {
	t.Helper()
	scenario := "seeded-" + profile.Name + "-embed-deploy"
	result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "embed", "deploy", "--file", selection.Path)
	record := commandEvidence("embed deploy", scenario, result, "pass")
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.DataOwnership = []string{"seeded"}
	if result.Err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return nil, record, fmt.Errorf("embed deploy failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}

	var deployments []seededDeployment
	if err := decodeCommandPayload(result.Stdout, &deployments); err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return nil, record, fmt.Errorf("decode embed deploy output for profile %q: %w", profile.Name, err)
	}
	if len(processDefinitionKeys(deployments)) == 0 && selection.BpmnProcessID == "" {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return nil, record, fmt.Errorf("embed deploy for profile %q returned no process definition key and fixture has no BPMN process ID", profile.Name)
	}
	record.ResourceKeys = append(record.ResourceKeys, processDefinitionKeys(deployments)...)
	record.ResourceKeys = append(record.ResourceKeys, deploymentKeys(deployments)...)
	return deployments, record, nil
}

// runSeededProcessInstance starts one process instance with the suite run marker.
func runSeededProcessInstance(t *testing.T, profile integrationProfile, selection embeddedFixtureSelection, deployments []seededDeployment) (seededProcessInstances, evidenceRecord, error) {
	t.Helper()
	args := []string{"--automation", "--json", "run", "process-instance"}
	args = append(args, runSelectorArgs(selection, deployments)...)
	args = append(args, "--vars", runMarkerVars(suite.marker))

	scenario := "seeded-" + profile.Name + "-run-process-instance"
	result := runC8VoltForProfile(t, profile.Name, scenario, args...)
	record := commandEvidence("run process-instance", scenario, result, "pass")
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.DataOwnership = []string{"seeded"}
	if result.Err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return seededProcessInstances{}, record, fmt.Errorf("run process-instance failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}

	var instances seededProcessInstances
	if err := decodeCommandPayload(result.Stdout, &instances); err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return seededProcessInstances{}, record, fmt.Errorf("decode run process-instance output for profile %q: %w", profile.Name, err)
	}
	keys := processInstanceKeys(instances)
	if len(keys) == 0 {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return seededProcessInstances{}, record, fmt.Errorf("run process-instance for profile %q returned no process instance keys", profile.Name)
	}
	record.ResourceKeys = keys
	return instances, record, nil
}

// assertSeededProcessInstancesObservable checks suite-created keys directly without assuming global cluster counts.
func assertSeededProcessInstancesObservable(t *testing.T, profile integrationProfile, keys []string) (evidenceRecord, error) {
	t.Helper()
	scenario := "seeded-" + profile.Name + "-get-process-instance"
	args := []string{"--automation", "--json", "get", "process-instance"}
	for _, key := range keys {
		args = append(args, "--key", key)
	}
	result := runC8VoltForProfile(t, profile.Name, scenario, args...)
	record := commandEvidence("get process-instance", scenario, result, "pass")
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.DataOwnership = []string{"seeded", "preexisting"}
	record.ResourceKeys = append([]string(nil), keys...)
	if result.Err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return record, fmt.Errorf("get process-instance failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}

	var instances seededProcessInstances
	if err := decodeCommandPayload(result.Stdout, &instances); err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return record, fmt.Errorf("decode get process-instance output for profile %q: %w", profile.Name, err)
	}
	if missing := missingSeededProcessInstanceKeys(keys, instances.Items); len(missing) > 0 {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return record, fmt.Errorf("get process-instance for profile %q missing seeded keys %v", profile.Name, missing)
	}
	return record, nil
}

// selectEmbeddedSeedFixture chooses the simple user-task fixture for the profile's Camunda minor version.
func selectEmbeddedSeedFixture(expectedVersion string, files []string) (embeddedFixtureSelection, error) {
	prefix, err := embeddedFixturePrefix(expectedVersion)
	if err != nil {
		return embeddedFixtureSelection{}, err
	}
	wantSuffix := prefix + "SimpleUserTask.bpmn"
	for _, file := range files {
		if strings.HasSuffix(file, wantSuffix) {
			return embeddedFixtureSelection{Path: file}, nil
		}
	}
	return embeddedFixtureSelection{}, fmt.Errorf("no embedded SimpleUserTask fixture found for Camunda version %q", expectedVersion)
}

// embeddedFixturePrefix converts a Camunda minor version into the embedded fixture filename prefix.
func embeddedFixturePrefix(expectedVersion string) (string, error) {
	switch {
	case strings.Contains(expectedVersion, "8.7"):
		return "C87_", nil
	case strings.Contains(expectedVersion, "8.8"):
		return "C88_", nil
	case strings.Contains(expectedVersion, "8.9"):
		return "C89_", nil
	default:
		return "", fmt.Errorf("unsupported embedded fixture version %q", expectedVersion)
	}
}

// embeddedFixtureBpmnProcessID reads the fixture's BPMN process id for fallback run selectors.
func embeddedFixtureBpmnProcessID(t *testing.T, fixturePath string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(suite.repoRoot, "embedded", filepath.FromSlash(fixturePath)))
	if err != nil {
		t.Fatalf("read embedded fixture %s: %v", fixturePath, err)
	}
	match := regexp.MustCompile(`<bpmn:process[^>]*\sid="([^"]+)"`).FindSubmatch(data)
	if len(match) != 2 {
		t.Fatalf("embedded fixture %s has no bpmn process id", fixturePath)
	}
	return string(match[1])
}

// runSelectorArgs prefers the exact deployed definition key and falls back to BPMN id/version.
func runSelectorArgs(selection embeddedFixtureSelection, deployments []seededDeployment) []string {
	for _, deployment := range deployments {
		if deployment.DefinitionKey != "" {
			return []string{"--pd-key", deployment.DefinitionKey}
		}
	}
	args := []string{"--bpmn-process-id", selection.BpmnProcessID}
	for _, deployment := range deployments {
		if deployment.DefinitionVersion > 0 {
			return append(args, "--pd-version", fmt.Sprint(deployment.DefinitionVersion))
		}
	}
	return args
}

// deploymentKeys extracts non-empty Camunda deployment keys from deployment results.
func deploymentKeys(deployments []seededDeployment) []string {
	keys := make([]string, 0, len(deployments))
	for _, deployment := range deployments {
		if deployment.Key != "" {
			keys = append(keys, deployment.Key)
		}
	}
	return keys
}

// processDefinitionKeys extracts non-empty process definition keys from deployment results.
func processDefinitionKeys(deployments []seededDeployment) []string {
	keys := make([]string, 0, len(deployments))
	for _, deployment := range deployments {
		if deployment.DefinitionKey != "" {
			keys = append(keys, deployment.DefinitionKey)
		}
	}
	return keys
}

// deploymentResourceIDs records the returned resource names or fixture path when the API omits resource IDs.
func deploymentResourceIDs(deployments []seededDeployment, selection embeddedFixtureSelection) []string {
	ids := make([]string, 0, len(deployments))
	for _, deployment := range deployments {
		if deployment.ResourceName != "" {
			ids = append(ids, deployment.ResourceName)
		}
	}
	if len(ids) == 0 && selection.Path != "" {
		ids = append(ids, selection.Path)
	}
	return ids
}

// processInstanceKeys extracts non-empty process instance keys from run or get results.
func processInstanceKeys(instances seededProcessInstances) []string {
	keys := make([]string, 0, len(instances.Items))
	for _, item := range instances.Items {
		if item.Key != "" {
			keys = append(keys, item.Key)
		}
	}
	return keys
}

// retainedCleanupRecord tracks intentionally retained data for later suite slices.
func retainedCleanupRecord(profile integrationProfile, keys []string) cleanupRecord {
	return cleanupRecord{
		Profile:             profile.Name,
		ProcessInstanceKeys: append([]string(nil), keys...),
		Status:              "retained",
		Attempted:           false,
		Reason:              "seeded data remains available for subsequent command-family slices",
	}
}
