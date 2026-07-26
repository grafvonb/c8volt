// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

// realStateFixture records the live Camunda state a real-state scenario created or discovered.
type realStateFixture struct {
	FixtureKind           string   `json:"fixtureKind"`
	BpmnProcessID         string   `json:"bpmnProcessId,omitempty"`
	ProcessDefinitionKeys []string `json:"processDefinitionKeys,omitempty"`
	ProcessInstanceKeys   []string `json:"processInstanceKeys,omitempty"`
	ElementInstanceKeys   []string `json:"elementInstanceKeys,omitempty"`
	JobKeys               []string `json:"jobKeys,omitempty"`
	IncidentKeys          []string `json:"incidentKeys,omitempty"`
	ListenerJobKeys       []string `json:"listenerJobKeys,omitempty"`
	Marker                string   `json:"marker"`
	Profile               string   `json:"profile"`
	CamundaVersion        string   `json:"camundaVersion,omitempty"`
	RequiredState         string   `json:"requiredState,omitempty"`
	CurrentEvidenceLevel  string   `json:"currentEvidenceLevel,omitempty"`
	TargetRealStateProof  string   `json:"targetRealStateProof,omitempty"`
	ObservedState         string   `json:"observedState,omitempty"`
	SkipReason            string   `json:"skipReason,omitempty"`
}

// realStateFamilyReport is the top-level evidence document for one real-state target.
type realStateFamilyReport struct {
	Family   string               `json:"family"`
	Marker   string               `json:"marker"`
	Profiles []integrationProfile `json:"profiles,omitempty"`
	Fixtures []realStateFixture   `json:"fixtures,omitempty"`
	Records  []evidenceRecord     `json:"records"`
	Summary  volumeSummary        `json:"summary"`
}

// realStateCommandSnapshot captures one before-state or after-state command observation.
type realStateCommandSnapshot struct {
	Label     string        `json:"label"`
	Command   string        `json:"command"`
	Result    commandResult `json:"result"`
	KeyCount  int           `json:"keyCount"`
	Output    string        `json:"output,omitempty"`
	Succeeded bool          `json:"succeeded"`
}

// writeRealStateFamilyReport stores the family report with a summary compatible with volume evidence.
func writeRealStateFamilyReport(t *testing.T, report realStateFamilyReport) string {
	t.Helper()
	report.Summary = summarizeVolumeRecords(report.Records)
	return writeJSON(t, "real-state-"+sanitizeEvidenceName(report.Family)+".json", report)
}

// writeRealStateDataReport stores fixture and resource identifiers for one real-state family.
func writeRealStateDataReport(t *testing.T, family string, fixtures []realStateFixture) string {
	t.Helper()
	return writeJSON(t, "real-state-data-"+sanitizeEvidenceName(family)+".json", realStateFixturesOrEmpty(fixtures))
}

// writeRealStateProgressReport stores human progress and output evidence for one real-state family.
func writeRealStateProgressReport(t *testing.T, family string, records []evidenceRecord) string {
	t.Helper()
	return writeJSON(t, "real-state-progress-"+sanitizeEvidenceName(family)+".json", evidenceRecordsOrEmpty(records))
}

// writeRealStateOpsReportEvidence stores ops report parity evidence for one real-state family.
func writeRealStateOpsReportEvidence(t *testing.T, family string, records []evidenceRecord) string {
	t.Helper()
	return writeJSON(t, "real-state-ops-reports-"+sanitizeEvidenceName(family)+".json", evidenceRecordsOrEmpty(records))
}

// realStateFixturesOrEmpty preserves the evidence contract that no-fixture reports are JSON arrays.
func realStateFixturesOrEmpty(fixtures []realStateFixture) []realStateFixture {
	if fixtures == nil {
		return []realStateFixture{}
	}
	return fixtures
}

// requireRealStateJSONStdoutClean validates JSON stdout and rejects human/progress leaks.
func requireRealStateJSONStdoutClean(result commandResult, label string) error {
	if err := requireVolumeJSON(result.Stdout); err != nil {
		return fmt.Errorf("%s JSON stdout invalid: %w", label, err)
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return fmt.Errorf("%s JSON stdout is not clean: %w", label, err)
	}
	return nil
}

// requireRealStateKeysOnlyStdoutClean validates keys-only stdout and rejects human/progress leaks.
func requireRealStateKeysOnlyStdoutClean(result commandResult, label string) error {
	if err := requireVolumeKeysOnly(result.Stdout); err != nil {
		return fmt.Errorf("%s keys-only stdout invalid: %w", label, err)
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return fmt.Errorf("%s keys-only stdout is not clean: %w", label, err)
	}
	return nil
}

// realStateCommandState runs a query command and records the observation for before/after evidence.
func realStateCommandState(t *testing.T, profile integrationProfile, label string, commandPath string, args ...string) realStateCommandSnapshot {
	t.Helper()
	scenario := "real-state-" + sanitizeEvidenceName(profile.Name+"-"+label)
	result := runC8VoltForProfile(t, profile.Name, scenario, args...)
	return realStateCommandSnapshot{
		Label:     label,
		Command:   commandPath,
		Result:    result,
		KeyCount:  len(extractNumericLines(result.Stdout)),
		Output:    strings.TrimSpace(result.Stdout),
		Succeeded: result.Err == nil,
	}
}

// seedRealStateEmbeddedProcessInstance creates one suite-owned process instance through existing c8volt commands.
func seedRealStateEmbeddedProcessInstance(t *testing.T, profile integrationProfile) (realStateFixture, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	selection, record, err := discoverEmbeddedFixture(t, profile)
	records = append(records, record)
	fixture := realStateFixture{
		FixtureKind:    "embedded BPMN through c8volt commands",
		BpmnProcessID:  selection.BpmnProcessID,
		Marker:         suite.marker,
		Profile:        profile.Name,
		CamundaVersion: profile.ExpectedVersion,
	}
	if err != nil {
		return fixture, records, err
	}

	deployments, record, err := deployEmbeddedFixture(t, profile, selection)
	records = append(records, record)
	fixture.ProcessDefinitionKeys = processDefinitionKeys(deployments)
	if err != nil {
		return fixture, records, err
	}

	instances, record, err := runSeededProcessInstance(t, profile, selection, deployments)
	records = append(records, record)
	fixture.ProcessInstanceKeys = processInstanceKeys(instances)
	return fixture, records, err
}

// extractNumericLines returns numeric stdout lines for key-count evidence without parsing command payloads.
func extractNumericLines(output string) []string {
	var keys []string
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if validateKeysOnlyString(trimmed) == nil {
			keys = append(keys, trimmed)
		}
	}
	return keys
}

// TestRealStateEvidenceWritersEmitArrays keeps empty real-state reports machine-friendly.
func TestRealStateEvidenceWritersEmitArrays(t *testing.T) {
	dataPath := writeRealStateDataReport(t, "scaffold", nil)
	progressPath := writeRealStateProgressReport(t, "scaffold", nil)
	opsPath := writeRealStateOpsReportEvidence(t, "scaffold", nil)
	for _, path := range []string{dataPath, progressPath, opsPath} {
		requireJSONFileArray(t, path)
	}
}

// TestRealStateMachineOutputAssertions verifies JSON and keys-only cleanliness helpers accept clean outputs.
func TestRealStateMachineOutputAssertions(t *testing.T) {
	jsonResult := commandResult{Stdout: `{"ok":true}` + "\n"}
	if err := requireRealStateJSONStdoutClean(jsonResult, "json"); err != nil {
		t.Fatalf("clean JSON rejected: %v", err)
	}
	keysResult := commandResult{Stdout: "123\n456\n"}
	if err := requireRealStateKeysOnlyStdoutClean(keysResult, "keys"); err != nil {
		t.Fatalf("clean keys-only output rejected: %v", err)
	}
}

// requireJSONFileArray verifies an evidence file is a JSON array, including empty no-data reports.
func requireJSONFileArray(t *testing.T, path string) {
	t.Helper()
	var rows []any
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read evidence %s: %v", path, err)
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatalf("decode evidence array %s: %v", path, err)
	}
}
