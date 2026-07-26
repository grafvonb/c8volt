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

const (
	realStateCascadeParentBPMN = "C89_MultipleSubProcessesParent"
	realStateCascadeMidBPMN    = "C89_SimpleParent"
	realStateCascadeLeafBPMN   = "C89_SimpleUserTask"
)

type realStateCascadeDataset struct {
	Fixture                    realStateFixture `json:"fixture"`
	ParentProcessDefinitionKey string           `json:"parentProcessDefinitionKey"`
	RootProcessInstanceKey     string           `json:"rootProcessInstanceKey"`
	MidProcessInstanceKey      string           `json:"midProcessInstanceKey"`
	LeafProcessInstanceKeys    []string         `json:"leafProcessInstanceKeys,omitempty"`
	FamilyProcessInstanceKeys  []string         `json:"familyProcessInstanceKeys"`
}

type realStateCascadeWalkPayload struct {
	Mode    string `json:"mode"`
	Outcome string `json:"outcome"`
	RootKey string `json:"rootKey,omitempty"`
	Keys    []string
	Items   []seededProcessInstance `json:"items,omitempty"`
}

type realStateCascadeDeleteItem struct {
	Key string `json:"key"`
	Ok  bool   `json:"ok"`
}

func TestRealStateCascadeFamily(t *testing.T) {
	profiles := selectedRealStateC89Profiles(t)

	report := realStateFamilyReport{
		Family:   "cascade",
		Marker:   suite.marker,
		Profiles: profiles,
	}
	var opsRecords []evidenceRecord
	var failures []string
	for _, profile := range profiles {
		walkDataset, records, err := seedRealStateCascadeDataset(t, profile, "walk-dry-run")
		report.Fixtures = append(report.Fixtures, walkDataset.Fixture)
		report.Records = append(report.Records, records...)
		if err == nil {
			records, err = runRealStateCascadeWalkAndDryRunScenarios(t, profile, walkDataset)
			report.Records = append(report.Records, records...)
		}
		if err != nil {
			failures = append(failures, err.Error())
		}

		cancelDataset, records, err := seedRealStateCascadeDataset(t, profile, "cancel")
		report.Fixtures = append(report.Fixtures, cancelDataset.Fixture)
		report.Records = append(report.Records, records...)
		if err == nil {
			records, err = runRealStateCascadeCancelScenario(t, profile, cancelDataset)
			report.Records = append(report.Records, records...)
		}
		if err != nil {
			failures = append(failures, err.Error())
		}

		deleteDataset, records, err := seedRealStateCascadeDataset(t, profile, "delete")
		report.Fixtures = append(report.Fixtures, deleteDataset.Fixture)
		report.Records = append(report.Records, records...)
		if err == nil {
			records, err = runRealStateCascadeDeleteScenario(t, profile, deleteDataset)
			report.Records = append(report.Records, records...)
		}
		if err != nil {
			failures = append(failures, err.Error())
		}

		deletePDDataset, records, err := seedRealStateCascadeDataset(t, profile, "delete-pd")
		report.Fixtures = append(report.Fixtures, deletePDDataset.Fixture)
		report.Records = append(report.Records, records...)
		if err == nil {
			records, err = runRealStateCascadeDeleteProcessDefinitionScenario(t, profile, deletePDDataset)
			report.Records = append(report.Records, records...)
		}
		if err != nil {
			failures = append(failures, err.Error())
		}

		purgeDataset, records, err := seedRealStateCascadeDataset(t, profile, "ops-purge-all-pd")
		report.Fixtures = append(report.Fixtures, purgeDataset.Fixture)
		report.Records = append(report.Records, records...)
		if err == nil {
			records, err = runRealStateCascadeOpsPurgeAllPDScenario(t, profile, purgeDataset)
			report.Records = append(report.Records, records...)
			opsRecords = append(opsRecords, records...)
		}
		if err != nil {
			failures = append(failures, err.Error())
		}
	}

	writeRealStateDataReport(t, "cascade", report.Fixtures)
	writeRealStateProgressReport(t, "cascade", report.Records)
	writeRealStateOpsReportEvidence(t, "cascade", opsRecords)
	writeRealStateFamilyReport(t, report)
	if len(failures) > 0 {
		t.Fatalf("real-state cascade scenarios failed:\n%s", strings.Join(failures, "\n"))
	}
}

func seedRealStateCascadeDataset(t *testing.T, profile integrationProfile, label string) (realStateCascadeDataset, []evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	dataset := realStateCascadeDataset{
		Fixture: realStateFixture{
			FixtureKind:          "embedded parent/call-activity BPMN dependency closure through c8volt commands",
			BpmnProcessID:        realStateCascadeParentBPMN,
			Marker:               suite.marker,
			Profile:              profile.Name,
			CamundaVersion:       profile.ExpectedVersion,
			RequiredState:        "active parent and child process-instance family",
			CurrentEvidenceLevel: realStateOutcomeLiveCovered,
			TargetRealStateProof: "walk, cancel, delete, process-definition delete, and ops purge resolve real child selections to roots and verify whole-family post-state",
		},
	}

	deployments, record, err := deployRealStateCascadeDefinitions(t, profile, label)
	records = append(records, record)
	if err != nil {
		return dataset, records, err
	}
	dataset.Fixture.ProcessDefinitionKeys = processDefinitionKeys(deployments)
	parentDeployment, ok := seededDeploymentByDefinitionID(deployments, realStateCascadeParentBPMN)
	if !ok || parentDeployment.DefinitionKey == "" {
		return dataset, records, fmt.Errorf("cascade seed %q for profile %q did not return parent process-definition key", label, profile.Name)
	}
	dataset.ParentProcessDefinitionKey = parentDeployment.DefinitionKey

	instance, record, err := runRealStateCascadeParentInstance(t, profile, label, parentDeployment)
	records = append(records, record)
	if err != nil {
		return dataset, records, err
	}
	dataset.RootProcessInstanceKey = firstString(processInstanceKeys(instance))
	if dataset.RootProcessInstanceKey == "" {
		return dataset, records, fmt.Errorf("cascade seed %q for profile %q returned no root process-instance key", label, profile.Name)
	}

	walk, record, err := waitForRealStateCascadeFamily(t, profile, label, dataset.RootProcessInstanceKey)
	records = append(records, record)
	if err != nil {
		return dataset, records, err
	}
	dataset.FamilyProcessInstanceKeys = append([]string(nil), walk.Keys...)
	for _, item := range walk.Items {
		switch item.BpmnProcessID {
		case realStateCascadeMidBPMN:
			if dataset.MidProcessInstanceKey == "" {
				dataset.MidProcessInstanceKey = item.Key
			}
		case realStateCascadeLeafBPMN:
			dataset.LeafProcessInstanceKeys = append(dataset.LeafProcessInstanceKeys, item.Key)
		}
	}
	dataset.Fixture.ProcessInstanceKeys = append([]string(nil), dataset.FamilyProcessInstanceKeys...)
	dataset.Fixture.ObservedState = fmt.Sprintf("family size %d with %d leaf child process instance(s)", len(dataset.FamilyProcessInstanceKeys), len(dataset.LeafProcessInstanceKeys))
	if dataset.MidProcessInstanceKey == "" || len(dataset.LeafProcessInstanceKeys) < 2 {
		return dataset, records, fmt.Errorf("cascade seed %q for profile %q found family keys %v but not expected mid/leaf BPMN shape: %+v", label, profile.Name, walk.Keys, walk.Items)
	}
	return dataset, records, nil
}

func deployRealStateCascadeDefinitions(t *testing.T, profile integrationProfile, label string) ([]seededDeployment, evidenceRecord, error) {
	t.Helper()
	scenario := "real-state-cascade-" + label + "-embed-deploy-all"
	result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "embed", "deploy", "--all")
	record := realStateCascadeRecord(profile, realStateCascadeDataset{}, result, "embed deploy", scenario, "json", []string{"all"}, false, true)
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "mutated"}
	if result.Err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return nil, record, fmt.Errorf("cascade embed deploy --all failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}
	var deployments []seededDeployment
	if err := decodeCommandPayload(result.Stdout, &deployments); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return nil, record, fmt.Errorf("decode cascade embed deploy output for profile %q: %w", profile.Name, err)
	}
	record.ResourceKeys = processDefinitionKeys(deployments)
	if _, ok := seededDeploymentByDefinitionID(deployments, realStateCascadeParentBPMN); !ok {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return nil, record, fmt.Errorf("cascade embed deploy --all did not deploy %s for profile %q", realStateCascadeParentBPMN, profile.Name)
	}
	return deployments, record, nil
}

func runRealStateCascadeParentInstance(t *testing.T, profile integrationProfile, label string, deployment seededDeployment) (seededProcessInstances, evidenceRecord, error) {
	t.Helper()
	scenario := "real-state-cascade-" + label + "-run-parent"
	result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "run", "process-instance", "--pd-key", deployment.DefinitionKey, "--vars", runMarkerVars(suite.marker))
	record := realStateCascadeRecord(profile, realStateCascadeDataset{}, result, "run process-instance", scenario, "json", []string{"pd-key", "vars"}, false, true)
	record.DataOwnership = []string{volumeDataSeeded, "retained"}
	if result.Err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return seededProcessInstances{}, record, fmt.Errorf("cascade run parent failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}
	var instances seededProcessInstances
	if err := decodeCommandPayload(result.Stdout, &instances); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return seededProcessInstances{}, record, fmt.Errorf("decode cascade run parent output for profile %q: %w", profile.Name, err)
	}
	record.ResourceKeys = processInstanceKeys(instances)
	return instances, record, nil
}

func waitForRealStateCascadeFamily(t *testing.T, profile integrationProfile, label string, rootKey string) (realStateCascadeWalkPayload, evidenceRecord, error) {
	t.Helper()
	var lastResult commandResult
	var lastErr error
	deadline := time.Now().Add(2 * time.Minute)
	for {
		scenario := "real-state-cascade-" + label + "-walk-family"
		result := runC8VoltForProfile(t, profile.Name, scenario, "--json", "walk", "pi", "--key", rootKey, "--flat")
		lastResult = result
		record := realStateCascadeRecord(profile, realStateCascadeDataset{}, result, "walk process-instance", scenario, "json", []string{"key", "flat"}, false, false)
		record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting}
		if result.Err == nil {
			var payload realStateCascadeWalkPayload
			if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
				lastErr = err
			} else if err := validateRealStateCascadeFamilyShape(payload, rootKey); err != nil {
				lastErr = err
			} else {
				record.ResourceKeys = append([]string(nil), payload.Keys...)
				return payload, record, nil
			}
		} else {
			lastErr = fmt.Errorf("%v; stderr: %s", result.Err, strings.TrimSpace(result.Stderr))
		}
		if time.Now().After(deadline) {
			record := realStateCascadeRecord(profile, realStateCascadeDataset{}, lastResult, "walk process-instance", "real-state-cascade-"+label+"-walk-family", "json", []string{"key", "flat"}, false, false)
			record.Outcome = realStateOutcomeFailed
			record.FailureClass = volumeFailureProduct
			return realStateCascadeWalkPayload{}, record, fmt.Errorf("cascade family for root %s did not become visible for profile %q: %v", rootKey, profile.Name, lastErr)
		}
		time.Sleep(2 * time.Second)
	}
}

func runRealStateCascadeWalkAndDryRunScenarios(t *testing.T, profile integrationProfile, dataset realStateCascadeDataset) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	var failures []string

	parentResult := runC8VoltForProfile(t, profile.Name, "real-state-cascade-walk-parent", "--json", "walk", "pi", "--key", dataset.MidProcessInstanceKey, "--parent")
	parentRecord := realStateCascadeRecord(profile, dataset, parentResult, "walk process-instance", "real-state-cascade-walk-parent", "json", []string{"key", "parent"}, false, false)
	if err := validateRealStateCascadeWalkContains(parentResult, dataset.RootProcessInstanceKey, dataset.MidProcessInstanceKey); err != nil {
		parentRecord.Outcome = realStateOutcomeFailed
		parentRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-cascade-walk-parent: %v", err))
	}
	records = append(records, parentRecord)

	childrenResult := runC8VoltForProfile(t, profile.Name, "real-state-cascade-walk-children", "--json", "walk", "pi", "--key", dataset.RootProcessInstanceKey, "--children")
	childrenRecord := realStateCascadeRecord(profile, dataset, childrenResult, "walk process-instance", "real-state-cascade-walk-children", "json", []string{"key", "children"}, false, false)
	if err := validateRealStateCascadeWalkContains(childrenResult, dataset.MidProcessInstanceKey, dataset.LeafProcessInstanceKeys[0], dataset.LeafProcessInstanceKeys[1]); err != nil {
		childrenRecord.Outcome = realStateOutcomeFailed
		childrenRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-cascade-walk-children: %v", err))
	}
	records = append(records, childrenRecord)

	cancelDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-cascade-cancel-child-dry-run", "--automation", "--json", "cancel", "pi", "--key", dataset.MidProcessInstanceKey, "--dry-run", "--workers", "2")
	cancelDryRunRecord := realStateCascadeRecord(profile, dataset, cancelDryRunResult, "cancel process-instance", "real-state-cascade-cancel-child-dry-run", "json", []string{"key", "dry-run", "workers"}, true, false)
	if err := validateRealStateCascadeDryRun(cancelDryRunResult, "cancel", dataset, dataset.MidProcessInstanceKey); err != nil {
		cancelDryRunRecord.Outcome = realStateOutcomeFailed
		cancelDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-cascade-cancel-child-dry-run: %v", err))
	}
	records = append(records, cancelDryRunRecord)

	deleteDryRunResult := runC8VoltForProfile(t, profile.Name, "real-state-cascade-delete-child-dry-run", "--automation", "--json", "delete", "pi", "--key", dataset.MidProcessInstanceKey, "--force", "--dry-run", "--workers", "2")
	deleteDryRunRecord := realStateCascadeRecord(profile, dataset, deleteDryRunResult, "delete process-instance", "real-state-cascade-delete-child-dry-run", "json", []string{"key", "force", "dry-run", "workers"}, true, false)
	if err := validateRealStateCascadeDryRun(deleteDryRunResult, "delete", dataset, dataset.MidProcessInstanceKey); err != nil {
		deleteDryRunRecord.Outcome = realStateOutcomeFailed
		deleteDryRunRecord.FailureClass = volumeFailureProduct
		failures = append(failures, fmt.Sprintf("real-state-cascade-delete-child-dry-run: %v", err))
	}
	records = append(records, deleteDryRunRecord)

	for _, key := range dataset.FamilyProcessInstanceKeys {
		if err := requireProcessInstanceState(t, profile, key, "ACTIVE"); err != nil {
			failures = append(failures, fmt.Sprintf("real-state-cascade-dry-run-retained-%s: %v", key, err))
		}
	}

	if len(failures) > 0 {
		return records, errors.New(strings.Join(failures, "\n"))
	}
	return records, nil
}

func runRealStateCascadeCancelScenario(t *testing.T, profile integrationProfile, dataset realStateCascadeDataset) ([]evidenceRecord, error) {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "real-state-cascade-cancel-child-confirmed", "--automation", "--json", "cancel", "pi", "--key", dataset.MidProcessInstanceKey, "--workers", "2")
	record := realStateCascadeRecord(profile, dataset, result, "cancel process-instance", "real-state-cascade-cancel-child-confirmed", "json", []string{"key", "workers"}, false, true)
	if err := validateRealStateCascadeMutationReport(result, "cancel", dataset.RootProcessInstanceKey); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	for _, key := range dataset.FamilyProcessInstanceKeys {
		if err := requireProcessInstanceStateEventually(t, profile, key, []string{"CANCELED", "TERMINATED"}, 2*time.Minute); err != nil {
			record.Outcome = realStateOutcomeFailed
			record.FailureClass = volumeFailureProduct
			return []evidenceRecord{record}, fmt.Errorf("cancel cascade did not cancel family key %s: %w", key, err)
		}
	}
	return []evidenceRecord{record}, nil
}

func runRealStateCascadeDeleteScenario(t *testing.T, profile integrationProfile, dataset realStateCascadeDataset) ([]evidenceRecord, error) {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "real-state-cascade-delete-child-confirmed", "--automation", "--json", "delete", "pi", "--key", dataset.MidProcessInstanceKey, "--force", "--workers", "2")
	record := realStateCascadeRecord(profile, dataset, result, "delete process-instance", "real-state-cascade-delete-child-confirmed", "json", []string{"key", "force", "workers"}, false, true)
	if err := validateRealStateCascadeMutationReport(result, "delete", dataset.RootProcessInstanceKey); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	if err := requireProcessInstancesAbsentEventually(t, profile, dataset.FamilyProcessInstanceKeys, 2*time.Minute); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	return []evidenceRecord{record}, nil
}

func runRealStateCascadeDeleteProcessDefinitionScenario(t *testing.T, profile integrationProfile, dataset realStateCascadeDataset) ([]evidenceRecord, error) {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "real-state-cascade-delete-pd-force", "--automation", "--json", "delete", "pd", "--key", dataset.ParentProcessDefinitionKey, "--force", "--workers", "2")
	record := realStateCascadeRecord(profile, dataset, result, "delete process-definition", "real-state-cascade-delete-pd-force", "json", []string{"key", "force", "workers"}, false, true)
	if err := validateRealStateCascadeProcessDefinitionDelete(result, dataset.ParentProcessDefinitionKey); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	if err := requireProcessInstancesAbsentEventually(t, profile, dataset.FamilyProcessInstanceKeys, 2*time.Minute); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	if err := requireProcessDefinitionAbsentEventually(t, profile, dataset.ParentProcessDefinitionKey, 2*time.Minute); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	return []evidenceRecord{record}, nil
}

func runRealStateCascadeOpsPurgeAllPDScenario(t *testing.T, profile integrationProfile, dataset realStateCascadeDataset) ([]evidenceRecord, error) {
	t.Helper()
	reportPath := volumeOpsPurgeReportPath(t, "real-state-cascade-ops-purge-all-pd-force", profile, "json")
	result := runC8VoltForProfile(t, profile.Name, "real-state-cascade-ops-purge-all-pd-force", "--automation", "--json", "ops", "purge", "all-process-definitions", "--key", dataset.ParentProcessDefinitionKey, "--force", "--workers", "2", "--report-file", reportPath, "--report-format", "json")
	record := realStateCascadeRecord(profile, dataset, result, "ops purge all-process-definitions", "real-state-cascade-ops-purge-all-pd-force", "json", []string{"key", "force", "workers", "report-file", "report-format"}, false, true)
	if err := validateRealStateCascadeOpsPurgeAllPD(result, reportPath, dataset.ParentProcessDefinitionKey); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	if err := requireProcessInstancesAbsentEventually(t, profile, dataset.FamilyProcessInstanceKeys, 2*time.Minute); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	if err := requireProcessDefinitionAbsentEventually(t, profile, dataset.ParentProcessDefinitionKey, 2*time.Minute); err != nil {
		record.Outcome = realStateOutcomeFailed
		record.FailureClass = volumeFailureProduct
		return []evidenceRecord{record}, err
	}
	return []evidenceRecord{record}, nil
}

func realStateCascadeRecord(profile integrationProfile, dataset realStateCascadeDataset, result commandResult, commandPath string, scenarioName string, outputMode string, flags []string, preview bool, confirmed bool) evidenceRecord {
	record := commandEvidence(commandPath, scenarioName, result, realStateOutcomeLiveCovered)
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.CoveredFlags = append([]string(nil), flags...)
	record.OutputMode = outputMode
	record.Preview = preview
	record.ConfirmedMutation = confirmed
	record.DataOwnership = []string{volumeDataSeeded, volumeDataPreexisting, "mutated"}
	record.ResourceKeys = append([]string(nil), dataset.FamilyProcessInstanceKeys...)
	if dataset.ParentProcessDefinitionKey != "" {
		record.ResourceKeys = append(record.ResourceKeys, dataset.ParentProcessDefinitionKey)
	}
	return record
}

func validateRealStateCascadeFamilyShape(payload realStateCascadeWalkPayload, rootKey string) error {
	if payload.RootKey != rootKey {
		return fmt.Errorf("cascade family rootKey = %q, want %q", payload.RootKey, rootKey)
	}
	if payload.Outcome != "complete" {
		return fmt.Errorf("cascade family outcome = %q, want complete", payload.Outcome)
	}
	if len(payload.Keys) < 4 {
		return fmt.Errorf("cascade family keys = %d, want at least 4; keys=%v", len(payload.Keys), payload.Keys)
	}
	if !containsString(payload.Keys, rootKey) {
		return fmt.Errorf("cascade family keys do not include root %s: %v", rootKey, payload.Keys)
	}
	counts := map[string]int{}
	for _, item := range payload.Items {
		counts[item.BpmnProcessID]++
	}
	if counts[realStateCascadeParentBPMN] != 1 || counts[realStateCascadeMidBPMN] < 1 || counts[realStateCascadeLeafBPMN] < 2 {
		return fmt.Errorf("cascade family BPMN counts = %v, want parent=1 mid>=1 leaf>=2", counts)
	}
	return nil
}

func validateRealStateCascadeWalkContains(result commandResult, wantKeys ...string) error {
	if err := requireVolumeCommandSuccess(result, "walk cascade"); err != nil {
		return err
	}
	if err := requireVolumeJSON(result.Stdout); err != nil {
		return err
	}
	var payload realStateCascadeWalkPayload
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode cascade walk payload: %w", err)
	}
	for _, key := range wantKeys {
		if !containsString(payload.Keys, key) {
			return fmt.Errorf("cascade walk keys %v do not include %s", payload.Keys, key)
		}
	}
	if payload.Outcome != "complete" {
		return fmt.Errorf("cascade walk outcome = %q, want complete", payload.Outcome)
	}
	return nil
}

func validateRealStateCascadeDryRun(result commandResult, operation string, dataset realStateCascadeDataset, requestedKey string) error {
	if err := requireVolumeCommandSuccess(result, operation+" cascade dry-run"); err != nil {
		return err
	}
	if err := requireVolumeJSON(result.Stdout); err != nil {
		return err
	}
	if err := requireMachineStdoutClean(result.Stdout); err != nil {
		return err
	}
	var payload struct {
		Operation          string   `json:"operation"`
		RequestedKeys      []string `json:"requestedKeys"`
		ResolvedRoots      []string `json:"resolvedRoots"`
		AffectedFamilyKeys []string `json:"affectedFamilyKeys"`
		MutationSubmitted  bool     `json:"mutationSubmitted"`
		ScopeComplete      bool     `json:"scopeComplete"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode cascade %s dry-run payload: %w", operation, err)
	}
	if payload.Operation != operation {
		return fmt.Errorf("cascade dry-run operation = %q, want %q", payload.Operation, operation)
	}
	if payload.MutationSubmitted {
		return fmt.Errorf("cascade %s dry-run reported mutationSubmitted=true", operation)
	}
	if !payload.ScopeComplete {
		return fmt.Errorf("cascade %s dry-run reported incomplete scope", operation)
	}
	if !containsString(payload.RequestedKeys, requestedKey) {
		return fmt.Errorf("cascade %s dry-run requested keys %v do not include selected child %s", operation, payload.RequestedKeys, requestedKey)
	}
	if !containsString(payload.ResolvedRoots, dataset.RootProcessInstanceKey) {
		return fmt.Errorf("cascade %s dry-run roots %v do not include root %s", operation, payload.ResolvedRoots, dataset.RootProcessInstanceKey)
	}
	for _, key := range dataset.FamilyProcessInstanceKeys {
		if !containsString(payload.AffectedFamilyKeys, key) {
			return fmt.Errorf("cascade %s dry-run affected keys %v do not include family key %s", operation, payload.AffectedFamilyKeys, key)
		}
	}
	return nil
}

func validateRealStateCascadeMutationReport(result commandResult, operation string, rootKey string) error {
	if err := requireVolumeCommandSuccess(result, operation+" cascade confirmed"); err != nil {
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
		Items []realStateCascadeDeleteItem `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode cascade %s report: %w", operation, err)
	}
	for _, item := range payload.Items {
		if item.Key == rootKey && item.Ok {
			return nil
		}
	}
	return fmt.Errorf("cascade %s report did not confirm resolved root %s: %+v", operation, rootKey, payload.Items)
}

func validateRealStateCascadeProcessDefinitionDelete(result commandResult, parentPDKey string) error {
	if err := requireVolumeCommandSuccess(result, "delete pd cascade confirmed"); err != nil {
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
		Items []realStateCascadeDeleteItem `json:"items"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode cascade delete pd report: %w", err)
	}
	for _, item := range payload.Items {
		if item.Key == parentPDKey && item.Ok {
			return nil
		}
	}
	return fmt.Errorf("cascade delete pd report did not confirm process definition %s: %+v", parentPDKey, payload.Items)
}

func validateRealStateCascadeOpsPurgeAllPD(result commandResult, reportPath string, parentPDKey string) error {
	if err := requireVolumeCommandSuccess(result, "ops purge all-pd cascade confirmed"); err != nil {
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
			Submitted bool                         `json:"submitted"`
			Confirmed bool                         `json:"confirmed"`
			Items     []realStateCascadeDeleteItem `json:"items"`
		} `json:"deletion"`
	}
	if err := decodeCommandPayload(result.Stdout, &payload); err != nil {
		return fmt.Errorf("decode cascade ops purge payload: %w", err)
	}
	if payload.Outcome != "deleted" {
		return fmt.Errorf("cascade ops purge outcome = %q, want deleted", payload.Outcome)
	}
	if !payload.Deletion.Submitted || !payload.Deletion.Confirmed {
		return fmt.Errorf("cascade ops purge deletion submitted/confirmed = %t/%t, want true/true", payload.Deletion.Submitted, payload.Deletion.Confirmed)
	}
	if !reportContainsResourceDeleteKey(payload.Deletion.Items, parentPDKey) {
		return fmt.Errorf("cascade ops purge payload missing deleted process definition %s: %+v", parentPDKey, payload.Deletion.Items)
	}
	report, err := readVolumeOpsPurgeAllPDReport(reportPath)
	if err != nil {
		return err
	}
	if report.Outcome != payload.Outcome || !report.Deletion.Submitted || !report.Deletion.Confirmed {
		return fmt.Errorf("cascade ops purge stdout/report mismatch: stdout=%s submitted=%t confirmed=%t report=%s submitted=%t confirmed=%t", payload.Outcome, payload.Deletion.Submitted, payload.Deletion.Confirmed, report.Outcome, report.Deletion.Submitted, report.Deletion.Confirmed)
	}
	return nil
}

func reportContainsResourceDeleteKey(items []realStateCascadeDeleteItem, key string) bool {
	for _, item := range items {
		if item.Key == key && item.Ok {
			return true
		}
	}
	return false
}

func requireProcessInstancesAbsentEventually(t *testing.T, profile integrationProfile, keys []string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for {
		last = nil
		for _, key := range keys {
			if err := requireProcessInstanceAbsent(t, profile, key); err != nil {
				last = err
				break
			}
		}
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(2 * time.Second)
	}
}

func requireProcessDefinitionAbsentEventually(t *testing.T, profile integrationProfile, key string, timeout time.Duration) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var last error
	for {
		last = requireProcessDefinitionAbsent(t, profile, key)
		if last == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return last
		}
		time.Sleep(2 * time.Second)
	}
}

func requireProcessDefinitionAbsent(t *testing.T, profile integrationProfile, key string) error {
	t.Helper()
	result := runC8VoltForProfile(t, profile.Name, "real-state-cascade-pd-absent-"+key, "--automation", "--json", "get", "pd", "--key", key)
	if result.Err != nil {
		combined := strings.ToLower(result.Stdout + "\n" + result.Stderr)
		if strings.Contains(combined, "not found") || strings.Contains(combined, "404") || strings.Contains(combined, "missing") {
			return nil
		}
		return fmt.Errorf("get pd absent verification failed unexpectedly: %v; %s", result.Err, compactLogSnippet(result.Stdout+"\n"+result.Stderr, 300))
	}
	return fmt.Errorf("process definition %s still visible after delete: %q", key, compactLogSnippet(result.Stdout, 300))
}

func seededDeploymentByDefinitionID(deployments []seededDeployment, definitionID string) (seededDeployment, bool) {
	for _, deployment := range deployments {
		if deployment.DefinitionID == definitionID {
			return deployment, true
		}
	}
	return seededDeployment{}, false
}
