// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

type realStateJobDataset struct {
	Fixture realStateFixture `json:"fixture"`
	Jobs    []realStateJob   `json:"jobs,omitempty"`
}

type realStateJob struct {
	Key                string `json:"key,omitempty"`
	State              string `json:"state,omitempty"`
	Retries            int32  `json:"retries"`
	Type               string `json:"type,omitempty"`
	Kind               string `json:"kind,omitempty"`
	ProcessInstanceKey string `json:"processInstanceKey,omitempty"`
	ElementInstanceKey string `json:"elementInstanceKey,omitempty"`
	ElementId          string `json:"elementId,omitempty"`
}

func TestRealStateJobsFamily(t *testing.T) {
	profiles := selectedRealStateC89Profiles(t)

	report := realStateFamilyReport{
		Family:   "jobs",
		Marker:   suite.marker,
		Profiles: profiles,
	}
	var commandProposals []proposalRecord
	var embeddedProposals []proposalRecord
	var failures []string
	for _, profile := range profiles {
		dataset, records, err := seedRealStateJobDataset(t, profile, 6)
		report.Fixtures = append(report.Fixtures, dataset.Fixture)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
			commandProposals = appendRealStateJobFixtureCommandGapProposals(commandProposals)
			embeddedProposals = appendRealStateJobEmbeddedBPMNGapProposals(embeddedProposals)
			continue
		}

		records, err = runRealStateJobScenarios(t, profile, dataset)
		report.Records = append(report.Records, records...)
		if err != nil {
			failures = append(failures, err.Error())
		}
	}
	commandProposals = appendRealStateJobTimeoutCommandGapProposals(commandProposals)

	writeRealStateDataReport(t, "jobs", report.Fixtures)
	writeRealStateProgressReport(t, "jobs", report.Records)
	writeRealStateOpsReportEvidence(t, "jobs", nil)
	writeCommandProposals(t, commandProposals)
	writeEmbeddedBPMNProposals(t, embeddedProposals)
	writeRealStateFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("real-state job scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func seedRealStateJobDataset(t *testing.T, profile integrationProfile, count int) (realStateJobDataset, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	files, record, err := discoverEmbeddedFixtureFiles(t, profile)
	records = append(records, record)
	fixture := realStateFixture{
		FixtureKind:          "embedded service-task BPMN through c8volt commands",
		Marker:               suite.marker,
		Profile:              profile.Name,
		CamundaVersion:       profile.ExpectedVersion,
		RequiredState:        "active service-task jobs",
		CurrentEvidenceLevel: realStateOutcomeLiveCovered,
		TargetRealStateProof: "suite-owned process instances expose non-empty get job rows",
	}
	if err != nil {
		return realStateJobDataset{Fixture: fixture}, records, err
	}

	selection, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "SimpleServiceTask.bpmn")
	if err != nil {
		fixture.EmbeddedBPMNProposal = true
		return realStateJobDataset{Fixture: fixture}, records, err
	}
	selection.BpmnProcessID = embeddedFixtureBpmnProcessID(t, selection.Path)
	fixture.BpmnProcessID = selection.BpmnProcessID

	deployments, record, err := deployEmbeddedFixture(t, profile, selection)
	record.CoveredFlags = []string{"file"}
	record.OutputMode = "json"
	records = append(records, record)
	fixture.ProcessDefinitionKeys = processDefinitionKeys(deployments)
	if err != nil {
		return realStateJobDataset{Fixture: fixture}, records, err
	}

	instances, record, err := runVolumeProcessInstances(t, profile, selection, deployments, count, "real-state-jobs")
	records = append(records, record)
	fixture.ProcessInstanceKeys = processInstanceKeys(instances)
	if err != nil {
		return realStateJobDataset{Fixture: fixture}, records, err
	}

	jobs, records, err := discoverRealStateJobsForProcessInstances(t, profile, fixture.ProcessInstanceKeys, records)
	fixture.JobKeys = realStateJobKeys(jobs)
	if err != nil {
		return realStateJobDataset{Fixture: fixture, Jobs: jobs}, records, err
	}
	return realStateJobDataset{Fixture: fixture, Jobs: jobs}, records, nil
}

func discoverRealStateJobsForProcessInstances(t *testing.T, profile integrationProfile, piKeys []string, records []evidenceRecord) ([]realStateJob, []evidenceRecord, error) {
	t.Helper()
	var jobs []realStateJob
	var failures []string
	for i, piKey := range piKeys {
		scenario := fmt.Sprintf("real-state-jobs-get-job-%d", i+1)
		found, result := pollRealStateJobsForProcessInstance(t, profile, scenario, piKey)
		record := realStateJobRecord(profile, result, "get job", scenario, "json", []string{"pi-key", "limit", "batch-size"}, []string{piKey}, false, false)
		record.ResourceKeys = append(record.ResourceKeys, realStateJobKeys(found)...)
		if err := requireVolumeCommandSuccess(result, "get job real-state discovery"); err != nil {
			record.Outcome = realStateOutcomeFailed
			record.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("%s: %v", scenario, err))
		} else if err := requireRealStateJSONStdoutClean(result, "get job real-state discovery"); err != nil {
			record.Outcome = realStateOutcomeFailed
			record.FailureClass = volumeFailureProduct
			failures = append(failures, fmt.Sprintf("%s: %v", scenario, err))
		}
		records = append(records, record)
		jobs = append(jobs, found...)
	}
	jobs = uniqueRealStateJobs(jobs)
	if len(jobs) == 0 {
		failures = append(failures, "service-task fixture produced no suite-owned jobs")
	}
	if len(failures) > 0 {
		return jobs, records, errors.New(strings.Join(failures, "\n"))
	}
	return jobs, records, nil
}

func pollRealStateJobsForProcessInstance(t *testing.T, profile integrationProfile, scenario string, piKey string) ([]realStateJob, commandResult) {
	t.Helper()
	var result commandResult
	for attempt := 1; attempt <= 8; attempt++ {
		attemptScenario := scenario
		if attempt > 1 {
			attemptScenario = fmt.Sprintf("%s-attempt-%d", scenario, attempt)
		}
		result = runC8VoltForProfile(t, profile.Name, attemptScenario, "--automation", "--json", "get", "job", "--pi-key", piKey, "--limit", "10", "--batch-size", "1")
		if result.Err == nil {
			jobs := realStateJobsFromResult(t, result, piKey)
			if len(jobs) > 0 {
				return jobs, result
			}
		}
		time.Sleep(2 * time.Second)
	}
	return realStateJobsFromResult(t, result, piKey), result
}

func realStateJobsFromResult(t *testing.T, result commandResult, piKey string) []realStateJob {
	t.Helper()
	var payload struct {
		Items []realStateJob `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return nil
	}
	var jobs []realStateJob
	for _, item := range payload.Items {
		if item.ProcessInstanceKey == piKey && item.Key != "" {
			jobs = append(jobs, item)
		}
	}
	return jobs
}

func runRealStateJobScenarios(t *testing.T, profile integrationProfile, dataset realStateJobDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string
	jobs := firstNRealStateJobs(dataset.Jobs, 6)
	if len(jobs) < 6 {
		return records, fmt.Errorf("real-state job dataset for profile %q has %d jobs, want at least 6", profile.Name, len(jobs))
	}

	getJSONResult := runC8VoltForProfile(t, profile.Name, "real-state-jobs-get-json", "--automation", "--json", "get", "job", "--pi-key", jobs[0].ProcessInstanceKey, "--limit", "10", "--batch-size", "1")
	getJSONRecord := realStateJobRecord(profile, getJSONResult, "get job", "real-state-jobs-get-json", "json", []string{"pi-key", "limit", "batch-size"}, []string{jobs[0].ProcessInstanceKey, jobs[0].Key}, false, false)
	if err := validateRealStateGetJobJSON(getJSONResult, jobs[0]); err != nil {
		getJSONRecord.Outcome = realStateOutcomeFailed
		getJSONRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-jobs-get-json: %v", err))
	}
	records = append(records, getJSONRecord)

	getHumanResult := runC8VoltForProfile(t, profile.Name, "real-state-jobs-get-human", "get", "job", "--key", jobs[0].Key)
	getHumanRecord := realStateJobRecord(profile, getHumanResult, "get job", "real-state-jobs-get-human", "one-line", []string{"key"}, []string{jobs[0].Key}, false, false)
	if err := validateRealStateGetJobHuman(getHumanResult, jobs[0]); err != nil {
		getHumanRecord.Outcome = realStateOutcomeFailed
		getHumanRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-jobs-get-human: %v", err))
	}
	records = append(records, getHumanRecord)

	dryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-jobs-update-dry-run", "--automation", "--json", "update", "job", "--key", jobs[1].Key, "--retries", "3", "--dry-run")
	dryRunRecord := realStateJobRecord(profile, dryRunResult, "update job", "real-state-jobs-update-dry-run", "json", []string{"key", "retries", "dry-run"}, []string{jobs[1].Key}, true, false)
	if err := validateRealStateJobUpdatePlan(dryRunResult, true, false); err != nil {
		dryRunRecord.Outcome = realStateOutcomeFailed
		dryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-jobs-update-dry-run: %v", err))
	}
	records = append(records, dryRunRecord)

	timeoutDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-jobs-update-timeout-dry-run", "--automation", "--json", "update", "job", "--key", jobs[2].Key, "--timeout", "30s", "--dry-run")
	timeoutDryRunRecord := realStateJobRecord(profile, timeoutDryRunResult, "update job", "real-state-jobs-update-timeout-dry-run", "json", []string{"key", "timeout", "dry-run"}, []string{jobs[2].Key}, true, false)
	if err := validateRealStateJobUpdatePlan(timeoutDryRunResult, true, false); err != nil {
		timeoutDryRunRecord.Outcome = realStateOutcomeFailed
		timeoutDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-jobs-update-timeout-dry-run: %v", err))
	}
	records = append(records, timeoutDryRunRecord)

	retryResult := runC8VoltForProfile(t, profile.Name, "real-state-jobs-update-retries", "--automation", "--json", "update", "job", "--key", jobs[3].Key, "--retries", "4", "--auto-confirm")
	retryRecord := realStateJobRecord(profile, retryResult, "update job", "real-state-jobs-update-retries", "json", []string{"key", "retries", "auto-confirm"}, []string{jobs[3].Key}, false, true)
	if err := validateRealStateJobUpdateResult(retryResult, "confirmed"); err != nil {
		retryRecord.Outcome = realStateOutcomeFailed
		retryRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-jobs-update-retries: %v", err))
	}
	records = append(records, retryRecord)

	noWaitResult := runC8VoltForProfile(t, profile.Name, "real-state-jobs-update-nowait", "--automation", "--json", "update", "job", "--key", jobs[4].Key, "--retries", "2", "--no-wait")
	noWaitRecord := realStateJobRecord(profile, noWaitResult, "update job", "real-state-jobs-update-nowait", "json", []string{"key", "retries", "no-wait"}, []string{jobs[4].Key}, false, true)
	if err := validateRealStateJobUpdateResult(noWaitResult, "submitted"); err != nil {
		noWaitRecord.Outcome = realStateOutcomeFailed
		noWaitRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-jobs-update-nowait: %v", err))
	}
	records = append(records, noWaitRecord)

	failResult := runC8VoltForProfile(t, profile.Name, "real-state-jobs-update-fail-nowait", "--automation", "--json", "update", "job", "--key", jobs[5].Key, "--fail", "--retries", "0", "--message", "c8volt real-state job failure", "--no-wait")
	failRecord := realStateJobRecord(profile, failResult, "update job", "real-state-jobs-update-fail-nowait", "json", []string{"key", "fail", "retries", "message", "no-wait"}, []string{jobs[5].Key}, false, true)
	if err := validateRealStateJobWorkerOutcome(failResult); err != nil {
		failRecord.Outcome = realStateOutcomeFailed
		failRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-jobs-update-fail-nowait: %v", err))
	}
	records = append(records, failRecord)

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func validateRealStateGetJobJSON(result commandResult, want realStateJob) error {
	if err := requireVolumeCommandSuccess(result, "get job JSON real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "get job JSON real-state"); err != nil {
		return err
	}
	var payload struct {
		Items []realStateJob `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode get job JSON payload: %w", err)
	}
	var jobs []realStateJob
	for _, item := range payload.Items {
		if item.ProcessInstanceKey == want.ProcessInstanceKey && item.Key != "" {
			jobs = append(jobs, item)
		}
	}
	for _, item := range jobs {
		if item.Key == want.Key {
			return nil
		}
	}
	return fmt.Errorf("get job JSON missing suite-owned job %s: %q", want.Key, compactLogSnippet(result.Stdout, 300))
}

func validateRealStateGetJobHuman(result commandResult, want realStateJob) error {
	if err := requireVolumeCommandSuccess(result, "get job human real-state"); err != nil {
		return err
	}
	if !strings.Contains(result.Stdout, want.Key) {
		return fmt.Errorf("get job human output missing job %s: %q", want.Key, compactLogSnippet(result.Stdout, 300))
	}
	if !strings.Contains(result.Stdout, "pi:"+want.ProcessInstanceKey) {
		return fmt.Errorf("get job human output missing process instance %s: %q", want.ProcessInstanceKey, compactLogSnippet(result.Stdout, 300))
	}
	return nil
}

func validateRealStateJobUpdatePlan(result commandResult, wantDryRun bool, wantMutationSubmitted bool) error {
	if err := requireVolumeCommandSuccess(result, "update job plan real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "update job plan real-state"); err != nil {
		return err
	}
	var payload struct {
		DryRun            bool `json:"dryRun"`
		MutationSubmitted bool `json:"mutationSubmitted"`
		MaterialChange    bool `json:"materialChange"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode update job plan payload: %w", err)
	}
	if payload.DryRun != wantDryRun {
		return fmt.Errorf("update job plan dryRun=%v, want %v", payload.DryRun, wantDryRun)
	}
	if payload.MutationSubmitted != wantMutationSubmitted {
		return fmt.Errorf("update job plan mutationSubmitted=%v, want %v", payload.MutationSubmitted, wantMutationSubmitted)
	}
	if !payload.MaterialChange {
		return fmt.Errorf("update job plan materialChange=false")
	}
	return nil
}

func validateRealStateJobUpdateResult(result commandResult, wantStatus string) error {
	if err := requireVolumeCommandSuccess(result, "update job result real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "update job result real-state"); err != nil {
		return err
	}
	var payload struct {
		Status           string `json:"status"`
		MutationAccepted bool   `json:"mutationAccepted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode update job result payload: %w", err)
	}
	if payload.Status != wantStatus {
		return fmt.Errorf("update job status=%q, want %q", payload.Status, wantStatus)
	}
	if !payload.MutationAccepted {
		return fmt.Errorf("update job mutationAccepted=false")
	}
	return nil
}

func validateRealStateJobWorkerOutcome(result commandResult) error {
	if err := requireVolumeCommandSuccess(result, "update job worker outcome real-state"); err != nil {
		return err
	}
	if err := requireRealStateJSONStdoutClean(result, "update job worker outcome real-state"); err != nil {
		return err
	}
	var payload struct {
		Mode             string `json:"mode"`
		Status           string `json:"status"`
		MutationAccepted bool   `json:"mutationAccepted"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode update job worker outcome payload: %w", err)
	}
	if payload.Mode != "technical_failure" {
		return fmt.Errorf("worker outcome mode=%q, want technical_failure", payload.Mode)
	}
	if payload.Status != "submitted" {
		return fmt.Errorf("worker outcome status=%q, want submitted", payload.Status)
	}
	if !payload.MutationAccepted {
		return fmt.Errorf("worker outcome mutationAccepted=false")
	}
	return nil
}

func realStateJobRecord(profile integrationProfile, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, keys []string, preview bool, confirmed bool) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, realStateOutcomeLiveCovered)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.ResourceKeys = append([]string(nil), keys...)
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "mutated", "retained"}
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	return record
}

func realStateJobKeys(jobs []realStateJob) []string {
	keys := make([]string, 0, len(jobs))
	for _, job := range jobs {
		if job.Key != "" {
			keys = append(keys, job.Key)
		}
	}
	return uniqueSortedStrings(keys)
}

func uniqueRealStateJobs(jobs []realStateJob) []realStateJob {
	seen := map[string]struct{}{}
	var out []realStateJob
	for _, job := range jobs {
		if job.Key == "" {
			continue
		}
		if _, ok := seen[job.Key]; ok {
			continue
		}
		seen[job.Key] = struct{}{}
		out = append(out, job)
	}
	return out
}

func firstNRealStateJobs(values []realStateJob, count int) []realStateJob {
	if len(values) < count {
		count = len(values)
	}
	return append([]realStateJob(nil), values[:count]...)
}

func appendRealStateJobFixtureCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	return appendRealStateCommandGapProposal(proposals,
		"active service-task jobs discoverable without direct Camunda setup",
		"real-state get job and update job coverage",
		[]string{"get job", "update job"},
		"Operators can create and mutate job-backed scenarios through c8volt commands alone.",
	)
}

func appendRealStateJobTimeoutCommandGapProposals(proposals []proposalRecord) []proposalRecord {
	return appendRealStateCommandGapProposal(proposals,
		"activated job accepted by Camunda timeout update",
		"confirmed update job --timeout post-state coverage",
		[]string{"update job"},
		"Operators can prove timeout mutation end-to-end without a direct Camunda job activation call.",
	)
}

func appendRealStateJobEmbeddedBPMNGapProposals(proposals []proposalRecord) []proposalRecord {
	return appendRealStateEmbeddedBPMNGapProposal(proposals,
		"embedded C89 service-task process that reliably leaves active jobs",
		"real-state job search and mutation coverage",
		[]string{"get job", "update job"},
		"Maintainers can rerun job integration slices against clean or dirty clusters without one-off BPMN setup.",
	)
}
