// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestRealStateListenersFamily(t *testing.T) {
	profiles := selectedRealStateC89Profiles(t)

	report := realStateFamilyReport{
		Family:   "listeners",
		Marker:   suite.marker,
		Profiles: profiles,
	}
	var commandProposals []proposalRecord
	var embeddedProposals []proposalRecord
	var failures []string
	for _, profile := range profiles {
		dataset, records, err := seedRealStateJobDataset(t, profile, 3)
		report.Fixtures = append(report.Fixtures, dataset.Fixture)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
			commandProposals = appendRealStateListenerCommandGapProposals(commandProposals)
			embeddedProposals = appendRealStateListenerEmbeddedBPMNGapProposals(embeddedProposals)
			continue
		}

		records, err = runRealStateListenerScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeRealStateDataReport(t, "listeners", report.Fixtures)
	writeRealStateProgressReport(t, "listeners", report.Records)
	writeRealStateOpsReportEvidence(t, "listeners", nil)
	writeCommandProposals(t, commandProposals)
	writeEmbeddedBPMNProposals(t, embeddedProposals)
	writeRealStateFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("real-state listener scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func runRealStateListenerScenarios(t *testing.T, profile integrationProfile, dataset realStateJobDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	jobs := firstNRealStateJobs(dataset.Jobs, 3)
	if len(jobs) < 3 {
		return records, fmt.Errorf("real-state listener dataset for profile %q has %d jobs, want at least 3", profile.Name, len(jobs))
	}
	for _, job := range jobs {
		if err := requireRealStateListenerJob(job); err != nil {
			failures = append(failures, err.Error())
		}
	}

	getElementResult := runC8VoltForProfile(t, profile.Name, "real-state-listeners-get-element-json", "--automation", "--json", "get", "element", "--key", jobs[0].ElementInstanceKey, "--with-listeners")
	getElementRecord := realStateListenerRecord(profile, getElementResult, "get element", "real-state-listeners-get-element-json", "json", []string{"key", "with-listeners"}, []string{jobs[0].ProcessInstanceKey, jobs[0].ElementInstanceKey, jobs[0].Key})
	if err := validateRealStateListenerJSON(getElementResult, "get element --with-listeners", jobs[0].Key); err != nil {
		getElementRecord.Outcome = realStateOutcomeFailed
		getElementRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-listeners-get-element-json: %v", err))
	}
	records = append(records, getElementRecord)

	walkJSONResult := runC8VoltForProfile(t, profile.Name, "real-state-listeners-walk-json", "--json", "walk", "process-instance", "--key", jobs[1].ProcessInstanceKey, "--with-elements", "--with-listeners")
	walkJSONRecord := realStateListenerRecord(profile, walkJSONResult, "walk process-instance", "real-state-listeners-walk-json", "json", []string{"key", "with-elements", "with-listeners"}, []string{jobs[1].ProcessInstanceKey, jobs[1].ElementInstanceKey, jobs[1].Key})
	if err := validateRealStateListenerJSON(walkJSONResult, "walk process-instance --with-listeners", jobs[1].Key); err != nil {
		walkJSONRecord.Outcome = realStateOutcomeFailed
		walkJSONRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-listeners-walk-json: %v", err))
	}
	records = append(records, walkJSONRecord)

	walkHumanResult := runC8VoltForProfile(t, profile.Name, "real-state-listeners-walk-human", "walk", "process-instance", "--key", jobs[1].ProcessInstanceKey, "--with-elements", "--with-listeners")
	walkHumanRecord := realStateListenerRecord(profile, walkHumanResult, "walk process-instance", "real-state-listeners-walk-human", "one-line", []string{"key", "with-elements", "with-listeners"}, []string{jobs[1].ProcessInstanceKey, jobs[1].ElementInstanceKey, jobs[1].Key})
	if err := validateRealStateListenerHuman(walkHumanResult, jobs[1].Key); err != nil {
		walkHumanRecord.Outcome = realStateOutcomeFailed
		walkHumanRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-listeners-walk-human: %v", err))
	}
	records = append(records, walkHumanRecord)

	analyseResult := runC8VoltForProfile(t, profile.Name, "real-state-listeners-ops-analyse-json", "--automation", "--json", "ops", "analyse", "slow-process-instances", "--key", jobs[2].ProcessInstanceKey, "--with-listeners")
	analyseRecord := realStateListenerRecord(profile, analyseResult, "ops analyse slow-process-instances", "real-state-listeners-ops-analyse-json", "json", []string{"key", "with-listeners"}, []string{jobs[2].ProcessInstanceKey, jobs[2].ElementInstanceKey, jobs[2].Key})
	if err := validateRealStateListenerJSON(analyseResult, "ops analyse slow-process-instances --with-listeners", jobs[2].Key); err != nil {
		analyseRecord.Outcome = realStateOutcomeFailed
		analyseRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-listeners-ops-analyse-json: %v", err))
	}
	records = append(records, analyseRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func requireRealStateListenerJob(job realStateJob) error {
	if job.Key == "" || job.ProcessInstanceKey == "" || job.ElementInstanceKey == "" {
		return fmt.Errorf("listener job is missing required keys: %+v", job)
	}
	if job.Kind != "EXECUTION_LISTENER" {
		return fmt.Errorf("job %s kind=%q, want EXECUTION_LISTENER", job.Key, job.Kind)
	}
	if job.ListenerEventType != "START" {
		return fmt.Errorf("job %s listenerEventType=%q, want START", job.Key, job.ListenerEventType)
	}
	return nil
}

func validateRealStateListenerJSON(result commandResult, label string, jobKey string) error {
	if err := requireVolumeCommandSuccess(result, label); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, label); err != nil {
		return err
	}
	var payload any
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode %s payload: %w", label, err)
	}
	if !jsonTreeContainsStringValue(payload, "listeners") {
		return fmt.Errorf("%s payload does not include listener fields: %q", label, compactLogSnippet(result.Stdout, 300))
	}
	if !jsonTreeContainsStringValue(payload, jobKey) {
		return fmt.Errorf("%s payload does not include listener job %s: %q", label, jobKey, compactLogSnippet(result.Stdout, 300))
	}
	return nil
}

func validateRealStateListenerHuman(result commandResult, jobKey string) error {
	if err := requireVolumeCommandSuccess(result, "walk process-instance listener human real-state"); err != nil {
		return err
	}
	return requireHumanContains(volumeHumanOutput(result), "listeners:", jobKey, "EXECUTION_LISTENER", "lsnr:START")
}

func realStateListenerRecord(profile integrationProfile, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, keys []string) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, realStateOutcomeLiveCovered)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.ResourceKeys = append([]string(nil), keys...)
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "retained"}
	return record
}

func jsonTreeContainsStringValue(value any, want string) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if key == want || jsonTreeContainsStringValue(child, want) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if jsonTreeContainsStringValue(child, want) {
				return true
			}
		}
	case string:
		return typed == want
	case json.Number:
		return typed.String() == want
	case float64:
		return fmt.Sprintf("%.0f", typed) == want
	}
	return false
}

func appendRealStateListenerCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	return appendRealStateCommandGapProposal(proposals,
		"runtime listener jobs visible through c8volt setup commands",
		"real-state listener enrichment coverage",
		[]string{"get element", "walk process-instance", "ops analyse slow-process-instances"},
		"Operators can create listener-enriched examples without direct Camunda setup.",
	)
}

func appendRealStateListenerEmbeddedBPMNGapProposals(proposals []proposalRecord) []proposalRecord {
	return appendRealStateEmbeddedBPMNGapProposal(proposals,
		"embedded C89 process with execution or task listener jobs",
		"real-state listener enrichment coverage",
		[]string{"get element", "walk process-instance", "ops analyse slow-process-instances"},
		"Maintainers can prove listener output from repository-owned BPMN fixtures.",
	)
}
