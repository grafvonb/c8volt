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

func TestVolumeOpsAnalyseFamily(t *testing.T) {
	datasetCount := volumeDatasetCount(t)
	profiles := requireSelectedProfiles(t)
	if err := requireProfilesReady(t, profiles); err != nil {
		t.Fatal(err)
	}

	report := volumeFamilyReport{
		Family:       "ops-analyse",
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

		records, err := runVolumeOpsAnalyseScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeVolumeDataReport(t, "ops-analyse", report.Datasets)
	writeVolumeProgressReport(t, "ops-analyse", report.Records)
	writeVolumePipelineReport(t, "ops-analyse", report.Records)
	writeVolumeOpsReportEvidence(t, "ops-analyse", report.Records)
	writeCommandProposals(t, appendOpsAnalyseCommandGapProposals(nil))
	writeEmbeddedBPMNProposals(t, appendOpsAnalyseEmbeddedBPMNGapProposals(nil))
	writeVolumeFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("volume ops analyse scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runVolumeOpsAnalyseScenarios(t *testing.T, profile integrationProfile, dataset volumeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	keys := firstNStrings(dataset.PositiveProcessInstanceKeys, 4)
	if len(keys) < 4 {
		return records, fmt.Errorf("ops analyse volume dataset for profile %q has %d positive keys, want at least 4", profile.Name, len(keys))
	}
	pdKey := firstString(dataset.PositiveProcessDefinitionKeys)
	if pdKey == "" {
		return records, fmt.Errorf("ops analyse volume dataset for profile %q has no process definition key", profile.Name)
	}

	jsonResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-json-timeline-duration", "--json", "ops", "analyse", "slow-process-instances", "--key", keys[0], "--key", keys[1], "--with-full-timeline", "--dur-longer", "1ms", "--dur-element-longer", "1ms")
	jsonRecord := volumeOpsAnalyseRecord(profile, dataset, jsonResult, "volume-ops-analyse-json-timeline-duration", "json", []string{"key", "with-full-timeline", "dur-longer", "dur-element-longer"})
	if err := validateVolumeOpsAnalyseJSON(jsonResult, 1); err != nil {
		jsonRecord.Outcome = volumeOutcomeFail
		jsonRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-json-timeline-duration: %v", err))
	}
	records = append(records, jsonRecord)

	keysOnlyResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-keys-only-search-limit", "--keys-only", "ops", "analyse", "spi", "--bpmn-process-id", dataset.PositiveBpmnProcessID, "--state", "active", "--batch-size", "1", "--limit", "2")
	keysOnlyRecord := volumeOpsAnalyseRecord(profile, dataset, keysOnlyResult, "volume-ops-analyse-keys-only-search-limit", "keys-only", []string{"bpmn-process-id", "state", "batch-size", "limit"})
	if err := validateVolumeOpsAnalyseKeysOnly(keysOnlyResult, 2); err != nil {
		keysOnlyRecord.Outcome = volumeOutcomeFail
		keysOnlyRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-keys-only-search-limit: %v", err))
	}
	records = append(records, keysOnlyRecord)

	stdin := keys[2] + "\n"
	humanResult := runC8VoltWithInput(t, "volume-ops-analyse-human-stdin-element-filter", stdin, argsForProfile(profile.Name, "ops", "analyse", "slow-process-instances", "-", "--with-full-timeline", "--type", "USER_TASK", "--element-state", "active", "--element-id", "SimpleUserTask_UserTask")...)
	humanRecord := volumeOpsAnalyseRecord(profile, dataset, humanResult, "volume-ops-analyse-human-stdin-element-filter", "one-line", []string{"stdin", "with-full-timeline", "type", "element-state", "element-id"})
	humanRecord.StdinPath = writeVolumeStdinKeys(t, "volume-ops-analyse-human-stdin-element-filter", []string{keys[2]})
	if err := validateVolumeOpsAnalyseHuman(humanResult); err != nil {
		humanRecord.Outcome = volumeOutcomeFail
		humanRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-human-stdin-element-filter: %v", err))
	}
	records = append(records, humanRecord)

	pdResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-json-pdkey-no-incidents", "--json", "ops", "analyse", "spi", "--pd-key", pdKey, "--state", "active", "--limit", "1", "--no-incidents-only", "--start-date-after", "2000-01-01", "--start-date-before", "2999-01-01")
	pdRecord := volumeOpsAnalyseRecord(profile, dataset, pdResult, "volume-ops-analyse-json-pdkey-no-incidents", "json", []string{"pd-key", "state", "limit", "no-incidents-only", "start-date-after", "start-date-before"})
	if err := validateVolumeOpsAnalyseJSON(pdResult, 1); err != nil {
		pdRecord.Outcome = volumeOutcomeFail
		pdRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-json-pdkey-no-incidents: %v", err))
	}
	records = append(records, pdRecord)

	listenerResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-json-with-listeners", "--json", "ops", "analyse", "spi", "--key", keys[3], "--with-listeners")
	listenerRecord := volumeOpsAnalyseRecord(profile, dataset, listenerResult, "volume-ops-analyse-json-with-listeners", "json", []string{"key", "with-listeners"})
	if err := validateVolumeOpsAnalyseJSON(listenerResult, 1); err != nil {
		listenerRecord.Outcome = volumeOutcomeFail
		listenerRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-json-with-listeners: %v", err))
	}
	records = append(records, listenerRecord)

	apiLatencyMarkdownReport := volumeOpsAnalyseReportPath(t, "volume-ops-analyse-api-latency-human-report-dirty-state", profile, "md")
	apiLatencyHumanResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-api-latency-human-report-dirty-state", "ops", "analyse", "api-latency", "--count", "1", "--workers", "1", "--report-file", apiLatencyMarkdownReport)
	apiLatencyHumanRecord := volumeOpsAnalyseAPILatencyRecord(profile, dataset, apiLatencyHumanResult, "volume-ops-analyse-api-latency-human-report-dirty-state", "one-line", []string{"count", "workers", "report-file"})
	if err := validateVolumeOpsAnalyseAPILatencyHuman(apiLatencyHumanResult, apiLatencyMarkdownReport); err != nil {
		apiLatencyHumanRecord.Outcome = volumeOutcomeFail
		apiLatencyHumanRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-api-latency-human-report-dirty-state: %v", err))
	}
	records = append(records, apiLatencyHumanRecord)

	apiLatencyJSONReport := volumeOpsAnalyseReportPath(t, "volume-ops-analyse-api-latency-json-bounds-report", profile, "json")
	apiLatencyJSONResult := runC8VoltForProfile(t, profile.Name, "volume-ops-analyse-api-latency-json-bounds-report", "--automation", "--json", "ops", "analyse", "api-latency", "--count", "3", "--workers", "2", "--report-file", apiLatencyJSONReport, "--report-format", "json")
	apiLatencyJSONRecord := volumeOpsAnalyseAPILatencyRecord(profile, dataset, apiLatencyJSONResult, "volume-ops-analyse-api-latency-json-bounds-report", "json", []string{"automation", "json", "count", "workers", "report-file", "report-format"})
	if err := validateVolumeOpsAnalyseAPILatencyJSON(apiLatencyJSONResult, apiLatencyJSONReport); err != nil {
		apiLatencyJSONRecord.Outcome = volumeOutcomeFail
		apiLatencyJSONRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("volume-ops-analyse-api-latency-json-bounds-report: %v", err))
	}
	records = append(records, apiLatencyJSONRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func volumeOpsAnalyseRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string) evidenceRecord {
	record := commandEvidence("ops analyse slow-process-instances", scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "retained"}
	record.ResourceKeys = append([]string(nil), dataset.PositiveProcessInstanceKeys...)
	return record
}

func volumeOpsAnalyseAPILatencyRecord(profile integrationProfile, dataset volumeDataset, result commandResult, scenarioName string, outputMode string, flags []string) evidenceRecord {
	record := commandEvidence("ops analyse api-latency", scenarioName, result, volumeOutcomePass)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "retained"}
	record.ResourceKeys = append([]string(nil), dataset.allProcessInstanceKeys()...)
	return record
}

func volumeOpsAnalyseReportPath(t *testing.T, scenarioName string, profile integrationProfile, ext string) string {
	t.Helper()
	name := sanitizeEvidenceName(scenarioName + "-" + profile.Name)
	path := filepath.Join(suite.workDir, "data", name+"."+ext)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create ops analyse report dir: %v", err)
	}
	return path
}

func validateVolumeOpsAnalyseJSON(result commandResult, minItems int) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse volume"); err != nil {
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
		} `json:"items"`
		Count int  `json:"count"`
		Empty bool `json:"empty"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode ops analyse payload: %w", err)
	}
	if len(payload.Items) < minItems {
		return fmt.Errorf("ops analyse returned %d items, want at least %d; empty=%t count=%d", len(payload.Items), minItems, payload.Empty, payload.Count)
	}
	for _, item := range payload.Items {
		if item.Key == "" {
			return fmt.Errorf("ops analyse item missing key: %q", compactLogSnippet(result.Stdout, 300))
		}
	}
	return nil
}

func validateVolumeOpsAnalyseAPILatencyHuman(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse api-latency human volume"); err != nil {
		return err
	}
	humanOutput := volumeHumanOutput(result)
	if err := requireHumanContains(humanOutput, "analyse api latency", "request: count 1; workers 1; stages 1; derived requests <= 2", "stage 1:", "finding:", "limitation: read-only evidence cannot prove", "outcome: completed", "report: written"); err != nil {
		return err
	}
	content, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read api-latency markdown report: %w", err)
	}
	text := string(content)
	for _, token := range []string{"# Analyse API Latency Report", "- Command: ops analyse api-latency", "- Mode: read_only", "- Primary Sample Limit: 1", "## Stages", "## Findings", "- Limitations:", "- Outcome: completed"} {
		if !strings.Contains(text, token) {
			return fmt.Errorf("api-latency markdown report missing %q: %q", token, compactLogSnippet(text, 300))
		}
	}
	for _, forbidden := range []string{"## Ownership", "## Visibility", "## Cleanup", "Authorization", "access_token", "client_secret"} {
		if strings.Contains(text, forbidden) {
			return fmt.Errorf("api-latency markdown report contains read-only/protected field %q", forbidden)
		}
	}
	return nil
}

func validateVolumeOpsAnalyseAPILatencyJSON(result commandResult, reportPath string) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse api-latency JSON volume"); err != nil {
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

	var stdoutPayload volumeOpsAnalyseAPILatencyPayload
	if err := decodeCommandPayload(result.Stdout, &stdoutPayload); err != nil {
		return fmt.Errorf("decode api-latency stdout payload: %w", err)
	}
	var stdoutRaw map[string]json.RawMessage
	if err := decodeCommandPayload(result.Stdout, &stdoutRaw); err != nil {
		return fmt.Errorf("decode api-latency stdout raw payload: %w", err)
	}
	if err := validateVolumeOpsAnalyseAPILatencyPayload(stdoutPayload, stdoutRaw); err != nil {
		return err
	}

	var reportPayload volumeOpsAnalyseAPILatencyPayload
	var reportRaw map[string]json.RawMessage
	content, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("read api-latency JSON report: %w", err)
	}
	if err := json.Unmarshal(content, &reportPayload); err != nil {
		return fmt.Errorf("decode api-latency JSON report %s: %w; content: %q", reportPath, err, compactLogSnippet(string(content), 300))
	}
	if err := json.Unmarshal(content, &reportRaw); err != nil {
		return fmt.Errorf("decode api-latency JSON report raw payload: %w", err)
	}
	if err := validateVolumeOpsAnalyseAPILatencyPayload(reportPayload, reportRaw); err != nil {
		return fmt.Errorf("report payload mismatch: %w", err)
	}
	if reportPayload.Outcome != stdoutPayload.Outcome || reportPayload.Plan.DerivedRequestLimit != stdoutPayload.Plan.DerivedRequestLimit {
		return fmt.Errorf("api-latency stdout/report parity mismatch: outcome %q/%q derived %d/%d", stdoutPayload.Outcome, reportPayload.Outcome, stdoutPayload.Plan.DerivedRequestLimit, reportPayload.Plan.DerivedRequestLimit)
	}
	return nil
}

func validateVolumeOpsAnalyseAPILatencyPayload(payload volumeOpsAnalyseAPILatencyPayload, raw map[string]json.RawMessage) error {
	if payload.SchemaVersion != "ops.api-latency.v1" {
		return fmt.Errorf("api-latency schemaVersion = %q, want ops.api-latency.v1", payload.SchemaVersion)
	}
	if payload.Context.CommandName != "ops analyse api-latency" {
		return fmt.Errorf("api-latency commandName = %q, want ops analyse api-latency", payload.Context.CommandName)
	}
	if payload.Request.Mode != "read_only" || payload.Request.Count != 3 || payload.Request.Workers != 2 {
		return fmt.Errorf("api-latency request = mode %q count %d workers %d, want read_only/3/2", payload.Request.Mode, payload.Request.Count, payload.Request.Workers)
	}
	if payload.Plan.Mode != "read_only" || payload.Plan.PrimarySampleLimit != 3 || payload.Plan.PrimarySampleAllocation != 3 || payload.Plan.DerivedRequestLimit != 6 {
		return fmt.Errorf("api-latency plan = mode %q limit %d allocation %d derived %d, want read_only/3/3/6", payload.Plan.Mode, payload.Plan.PrimarySampleLimit, payload.Plan.PrimarySampleAllocation, payload.Plan.DerivedRequestLimit)
	}
	if len(payload.Plan.Stages) != 2 || payload.Plan.Stages[0].WorkerCount != 1 || payload.Plan.Stages[1].WorkerCount != 2 {
		return fmt.Errorf("api-latency plan stages = %+v, want workers 1,2", payload.Plan.Stages)
	}
	if payload.Outcome != "completed" {
		return fmt.Errorf("api-latency outcome = %q, want completed", payload.Outcome)
	}
	if _, ok := raw["ownership"]; ok {
		return fmt.Errorf("read-only api-latency payload contains ownership")
	}
	if _, ok := raw["visibility"]; ok {
		return fmt.Errorf("read-only api-latency payload contains visibility")
	}
	if _, ok := raw["cleanup"]; ok {
		return fmt.Errorf("read-only api-latency payload contains cleanup")
	}
	if len(payload.Stages) != len(payload.Plan.Stages) {
		return fmt.Errorf("api-latency stage count = %d, want %d", len(payload.Stages), len(payload.Plan.Stages))
	}
	totalDerived := 0
	for i, stage := range payload.Stages {
		plan := payload.Plan.Stages[i]
		if stage.Status != "completed" {
			return fmt.Errorf("api-latency stage %d status = %q, want completed", i+1, stage.Status)
		}
		if stage.ActualMaxConcurrency > plan.WorkerCount {
			return fmt.Errorf("api-latency stage %d concurrency = %d, exceeds worker ceiling %d", i+1, stage.ActualMaxConcurrency, plan.WorkerCount)
		}
		if stage.PrimaryAttempts == 0 || stage.PrimaryAttempts > plan.PrimarySamples*3 {
			return fmt.Errorf("api-latency stage %d primary attempts = %d, want 1..%d", i+1, stage.PrimaryAttempts, plan.PrimarySamples*3)
		}
		if stage.DerivedAttempts > plan.DerivedRequestLimit {
			return fmt.Errorf("api-latency stage %d derived attempts = %d, exceeds plan %d", i+1, stage.DerivedAttempts, plan.DerivedRequestLimit)
		}
		if !stage.hasCategories("topology_read", "process_definition_search", "process_instance_search") {
			return fmt.Errorf("api-latency stage %d missing read-only categories: %+v", i+1, stage.Categories)
		}
		totalDerived += stage.DerivedAttempts
	}
	if totalDerived > payload.Plan.DerivedRequestLimit {
		return fmt.Errorf("api-latency total derived attempts = %d, exceeds plan %d", totalDerived, payload.Plan.DerivedRequestLimit)
	}
	if len(payload.Findings) == 0 {
		return fmt.Errorf("api-latency findings are empty")
	}
	if len(payload.Limitations) == 0 {
		return fmt.Errorf("api-latency limitations are empty")
	}
	return nil
}

type volumeOpsAnalyseAPILatencyPayload struct {
	SchemaVersion string `json:"schemaVersion"`
	Context       struct {
		CommandName string `json:"commandName"`
	} `json:"context"`
	Request struct {
		Mode    string `json:"mode"`
		Count   int    `json:"count"`
		Workers int    `json:"workers"`
	} `json:"request"`
	Plan struct {
		Mode                    string                         `json:"mode"`
		Stages                  []volumeOpsAPILatencyStagePlan `json:"stages"`
		PrimarySampleLimit      int                            `json:"primarySampleLimit"`
		PrimarySampleAllocation int                            `json:"primarySampleAllocation"`
		DerivedRequestLimit     int                            `json:"derivedRequestLimit"`
	} `json:"plan"`
	Stages      []volumeOpsAPILatencyStageResult `json:"stages"`
	Findings    []struct{ Code string }          `json:"findings"`
	Limitations []string                         `json:"limitations"`
	Outcome     string                           `json:"outcome"`
}

type volumeOpsAPILatencyStagePlan struct {
	WorkerCount         int `json:"workerCount"`
	PrimarySamples      int `json:"primarySamples"`
	DerivedRequestLimit int `json:"derivedRequestLimit"`
}

type volumeOpsAPILatencyStageResult struct {
	Status               string `json:"status"`
	ActualMaxConcurrency int    `json:"actualMaxConcurrency"`
	PrimaryAttempts      int    `json:"primaryAttempts"`
	DerivedAttempts      int    `json:"derivedAttempts"`
	Categories           []struct {
		Category string `json:"category"`
	} `json:"categories"`
}

func (s volumeOpsAPILatencyStageResult) hasCategories(categories ...string) bool {
	seen := map[string]struct{}{}
	for _, category := range s.Categories {
		seen[category.Category] = struct{}{}
	}
	for _, category := range categories {
		if _, ok := seen[category]; !ok {
			return false
		}
	}
	return true
}

func validateVolumeOpsAnalyseKeysOnly(result commandResult, limit int) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse keys-only volume"); err != nil {
		return err
	}
	if err := requireVolumeKeysOnly(result.Stdout); err != nil {
		return err
	}
	keys := nonEmptyVolumeLines(result.Stdout)
	if len(keys) == 0 {
		return fmt.Errorf("ops analyse keys-only returned no keys")
	}
	if len(keys) > limit {
		return fmt.Errorf("ops analyse keys-only returned %d keys, want at most %d", len(keys), limit)
	}
	return nil
}

func validateVolumeOpsAnalyseHuman(result commandResult) error {
	if err := requireVolumeCommandSuccess(result, "ops analyse human volume"); err != nil {
		return err
	}
	if !strings.Contains(result.Stdout, "process instances:") {
		return fmt.Errorf("ops analyse human output missing final count: %q", compactLogSnippet(result.Stdout, 300))
	}
	return nil
}

func nonEmptyVolumeLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
