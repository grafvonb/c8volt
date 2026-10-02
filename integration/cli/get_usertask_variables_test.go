// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/stretchr/testify/require"
)

const (
	userTaskCustomerVariable = "customer"
	userTaskPayloadVariable  = "structuredPayload"
)

var userTaskScenarioValues = map[string]any{
	"hasIncident":            false,
	"incident":               99,
	userTaskCustomerVariable: "alice",
	userTaskPayloadVariable: map[string]any{
		"message": "Grüße 世界 from the user-task integration scenario",
		"items":   []any{"alpha", 2, false},
	},
}

type userTaskVariableFixtures struct {
	Ordinary embeddedFixtureSelection
	Incident embeddedFixtureSelection
}

type userTaskVariableSeed struct {
	Label              string                            `json:"label"`
	Fixture            embeddedFixtureSelection          `json:"fixture"`
	Deployments        []seededDeployment                `json:"deployments"`
	ProcessInstance    seededProcessInstance             `json:"processInstance"`
	Task               task.UserTask                     `json:"task"`
	ProcessVariables   []process.ProcessInstanceVariable `json:"processVariables"`
	EffectiveVariables []task.UserTaskVariable           `json:"effectiveVariables"`
	LastObservation    string                            `json:"lastObservation,omitempty"`
}

type userTaskVariableReport struct {
	Profile        string                 `json:"profile"`
	CamundaVersion string                 `json:"camundaVersion"`
	Seeds          []userTaskVariableSeed `json:"seeds"`
	Records        []evidenceRecord       `json:"records"`
	Outcome        string                 `json:"outcome"`
	Failure        string                 `json:"failure,omitempty"`
}

// TestGetFamilyUserTaskVariables proves native local-variable selection and scope behavior against selected real profiles.
func TestGetFamilyUserTaskVariables(t *testing.T) {
	for _, profile := range requireSelectedProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			report := userTaskVariableReport{Profile: profile.Name, CamundaVersion: profile.ExpectedVersion, Outcome: "pass"}
			defer func() {
				if t.Failed() {
					report.Outcome = "fail"
				}
				writeJSON(t, "user-task-variables-"+profile.Name+".json", report)
			}()

			require.NoError(t, requireProfilesReady(t, []integrationProfile{profile}))

			if profile.ExpectedVersion == "8.7" {
				records, err := assertUserTaskVariablesUnsupported(t, profile)
				report.Records = append(report.Records, records...)
				if err != nil {
					report.Failure = err.Error()
				}
				require.NoError(t, err)
				return
			}

			fixtures, records, err := discoverUserTaskVariableFixtures(t, profile)
			report.Records = append(report.Records, records...)
			if err != nil {
				report.Failure = err.Error()
				require.NoError(t, err)
			}

			for _, fixture := range []struct {
				label       string
				selection   embeddedFixtureSelection
				localShadow bool
			}{
				{label: "ordinary", selection: fixtures.Ordinary},
				{label: "incident", selection: fixtures.Incident, localShadow: true},
			} {
				t.Run(fixture.label, func(t *testing.T) {
					seed, records, err := seedUserTaskVariableScenario(t, profile, fixture.label, fixture.selection, fixture.localShadow)
					report.Seeds = append(report.Seeds, seed)
					report.Records = append(report.Records, records...)
					require.NoError(t, err)
					if fixture.localShadow {
						records, err = assertUserTaskVariableFilterTable(t, profile, seed)
						report.Records = append(report.Records, records...)
						if err != nil {
							t.Error(err)
						}
					} else {
						checkUserTaskVariableCommand(t, profile, seed.Label+"-process-only", &report.Records,
							[]string{"get", "ut", "--pi-key", seed.ProcessInstance.Key, "--var", `customer="alice"`, "--json"},
							func(t *testing.T, result commandResult) {
								var got task.UserTasks
								require.NoError(t, decodeSuccessfulResult(result, &got))
								require.Empty(t, got.Items)
							})
					}
					assertUserTaskVariableDisplay(t, profile, seed, fixture.localShadow, &report.Records)
				})
			}
			t.Run("called-process", func(t *testing.T) {
				seed, records, err := seedCalledUserTaskVariables(t, profile, fixtures.Incident)
				report.Seeds = append(report.Seeds, seed)
				report.Records = append(report.Records, records...)
				require.NoError(t, err)
				for _, value := range []string{"1", "99"} {
					checkUserTaskVariableCommand(t, profile, "child-eq-"+value, &report.Records,
						[]string{"get", "ut", "--pi-key", seed.ProcessInstance.Key, "--var", "incident=" + value, "--json"},
						func(t *testing.T, result commandResult) {
							var got task.UserTasks
							require.NoError(t, decodeSuccessfulResult(result, &got))
							if value == "1" {
								require.Equal(t, []string{seed.Task.Key}, userTaskKeys(got.Items))
							} else {
								require.Empty(t, got.Items)
							}
						})
				}
				assertUserTaskVariableDisplay(t, profile, seed, true, &report.Records)
			})
		})
	}
}

// discoverUserTaskVariableFixtures selects the exact ordinary and incident fixtures for the active profile family.
func discoverUserTaskVariableFixtures(t *testing.T, profile integrationProfile) (userTaskVariableFixtures, []evidenceRecord, error) {
	t.Helper()
	scenario := "user-task-vars-" + profile.Name + "-embed-list"
	result := runC8VoltForProfile(t, profile.Name, scenario, "--automation", "--json", "embed", "list", "--details")
	record := userTaskVariableEvidence(profile, "embed list", scenario, result)
	if result.Err != nil {
		record.Outcome = "fail"
		record.FailureClass = "harness_setup"
		return userTaskVariableFixtures{}, []evidenceRecord{record}, fmt.Errorf("embed list failed for profile %q: %v; stderr: %s", profile.Name, result.Err, strings.TrimSpace(result.Stderr))
	}
	var files []string
	if err := decodeCommandPayload(result.Stdout, &files); err != nil {
		record.Outcome = "fail"
		record.FailureClass = "product"
		return userTaskVariableFixtures{}, []evidenceRecord{record}, fmt.Errorf("decode embed list for profile %q: %w", profile.Name, err)
	}
	ordinary, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "SimpleUserTask.bpmn")
	if err != nil {
		record.Outcome = "blocked"
		record.FailureClass = "missing_fixture_support"
		return userTaskVariableFixtures{}, []evidenceRecord{record}, err
	}
	incident, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "SimpleUserTaskWithIncident.bpmn")
	if err != nil {
		record.Outcome = "blocked"
		record.FailureClass = "missing_fixture_support"
		return userTaskVariableFixtures{}, []evidenceRecord{record}, err
	}
	ordinary.BpmnProcessID = embeddedFixtureBpmnProcessID(t, ordinary.Path)
	incident.BpmnProcessID = embeddedFixtureBpmnProcessID(t, incident.Path)
	record.ResourceKeys = []string{ordinary.Path, incident.Path}
	return userTaskVariableFixtures{Ordinary: ordinary, Incident: incident}, []evidenceRecord{record}, nil
}

// seedUserTaskVariableScenario deploys, starts, and observes one owned standalone scenario before negative assertions run.
func seedUserTaskVariableScenario(t *testing.T, profile integrationProfile, label string, selection embeddedFixtureSelection, localShadow bool) (userTaskVariableSeed, []evidenceRecord, error) {
	t.Helper()
	seed := userTaskVariableSeed{Label: label, Fixture: selection}
	deployments, deployRecord, err := deployEmbeddedFixtureWithLabel(t, profile, selection, "user-task-vars-"+label)
	records := []evidenceRecord{deployRecord}
	seed.Deployments = deployments
	if err != nil {
		return seed, records, err
	}
	instances, runRecord, err := runSeededProcessInstanceWithVariables(t, profile, selection, deployments, userTaskScenarioValues, "user-task-vars-"+label)
	records = append(records, runRecord)
	if err != nil {
		return seed, records, err
	}
	if len(instances.Items) != 1 {
		return seed, records, fmt.Errorf("%s scenario for profile %q returned %d process instances, want 1", label, profile.Name, len(instances.Items))
	}
	seed.ProcessInstance = instances.Items[0]
	readyRecords, err := awaitUserTaskVariableReadiness(t, profile, &seed, firstString(processDefinitionKeys(deployments)), localShadow)
	records = append(records, readyRecords...)
	if err != nil {
		return seed, records, err
	}
	return seed, records, nil
}

// awaitUserTaskVariableReadiness retains the last observed task and scopes while polling all state needed for trustworthy exclusions.
func awaitUserTaskVariableReadiness(t *testing.T, profile integrationProfile, seed *userTaskVariableSeed, definitionKey string, localShadow bool) ([]evidenceRecord, error) {
	t.Helper()
	var records []evidenceRecord
	deadline := time.Now().Add(30 * time.Second)
	for attempt := 1; ; attempt++ {
		prefix := fmt.Sprintf("user-task-vars-%s-%s-readiness-attempt-%02d", profile.Name, seed.Label, attempt)
		plainResult := runC8VoltForProfile(t, profile.Name, prefix+"-task", "get", "ut", "--pi-key", seed.ProcessInstance.Key, "--state", "created", "--json")
		plainRecord := userTaskVariableEvidence(profile, "get user-task", prefix+"-task", plainResult)
		records = append(records, plainRecord)
		var plain task.UserTasks
		plainErr := decodeSuccessfulResult(plainResult, &plain)

		processResult := runC8VoltForProfile(t, profile.Name, prefix+"-process-vars", "get", "pi", "--key", seed.ProcessInstance.Key, "--with-vars", "--json")
		processRecord := userTaskVariableEvidence(profile, "get process-instance", prefix+"-process-vars", processResult)
		records = append(records, processRecord)
		var processItems process.VariableEnrichedProcessInstances
		processErr := decodeSuccessfulResult(processResult, &processItems)

		effectiveResult := runC8VoltForProfile(t, profile.Name, prefix+"-effective-vars", "get", "ut", "--pi-key", seed.ProcessInstance.Key, "--state", "created", "--with-vars", "--json")
		effectiveRecord := userTaskVariableEvidence(profile, "get user-task", prefix+"-effective-vars", effectiveResult)
		records = append(records, effectiveRecord)
		var effective task.VariableEnrichedUserTasks
		effectiveErr := decodeSuccessfulResult(effectiveResult, &effective)

		seed.LastObservation = fmt.Sprintf("taskErr=%v taskCount=%d processErr=%v processCount=%d effectiveErr=%v effectiveCount=%d", plainErr, len(plain.Items), processErr, len(processItems.Items), effectiveErr, len(effective.Items))
		if plainErr == nil && processErr == nil && effectiveErr == nil && len(plain.Items) == 1 && len(processItems.Items) == 1 && len(effective.Items) == 1 && effective.Items[0].Item.Key == plain.Items[0].Key {
			seed.Task = plain.Items[0]
			seed.ProcessVariables = processItems.Items[0].Variables
			seed.EffectiveVariables = effective.Items[0].Variables
			if err := validateUserTaskVariableReadiness(*seed, definitionKey, localShadow); err == nil {
				return records, nil
			} else {
				seed.LastObservation = err.Error()
			}
		}
		if time.Now().After(deadline) {
			if len(records) > 0 {
				records[len(records)-1].Outcome = "fail"
				records[len(records)-1].FailureClass = "product"
			}
			return records, fmt.Errorf("user-task variable readiness timed out for profile %q case %q: %s", profile.Name, seed.Label, seed.LastObservation)
		}
		time.Sleep(time.Second)
	}
}

// validateUserTaskVariableReadiness checks identity, process ownership, effective shadowing, and typed structured values together.
func validateUserTaskVariableReadiness(seed userTaskVariableSeed, definitionKey string, localShadow bool) error {
	if seed.Task.ProcessInstanceKey != seed.ProcessInstance.Key || seed.Task.ProcessDefinitionKey != definitionKey || seed.Task.Key == "" || seed.Task.ElementInstanceKey == "" {
		return fmt.Errorf("unexpected task identity: task=%q pi=%q pd=%q element=%q", seed.Task.Key, seed.Task.ProcessInstanceKey, seed.Task.ProcessDefinitionKey, seed.Task.ElementInstanceKey)
	}
	processVars := variableMap(seed.ProcessVariables)
	for name, value := range map[string]string{
		"incident":               "99",
		userTaskCustomerVariable: `"alice"`,
		"c8voltITRunId":          fmt.Sprintf("%q", suite.marker),
	} {
		variable, ok := processVars[name]
		if !ok || variable.Value != value || variable.ScopeKey != seed.ProcessInstance.Key || variable.ProcessInstanceKey != seed.ProcessInstance.Key {
			return fmt.Errorf("process variable %q not ready with value %s and scope %s: %+v", name, value, seed.ProcessInstance.Key, variable)
		}
	}
	if variable, ok := processVars[userTaskPayloadVariable]; !ok || variable.ScopeKey != seed.ProcessInstance.Key || !sameJSONValue(variable.Value, userTaskScenarioValues[userTaskPayloadVariable]) {
		return fmt.Errorf("structured process variable not ready with scope %s: %+v", seed.ProcessInstance.Key, variable)
	}
	effectiveVars := variableMap(seed.EffectiveVariables)
	incident, ok := effectiveVars["incident"]
	wantValue, wantScope := "99", seed.ProcessInstance.Key
	if localShadow {
		wantValue, wantScope = "1", seed.Task.ElementInstanceKey
	}
	if !ok || incident.Value != wantValue || incident.ScopeKey != wantScope || incident.ProcessInstanceKey != seed.ProcessInstance.Key {
		return fmt.Errorf("effective incident not ready with value %s and scope %s: %+v", wantValue, wantScope, incident)
	}
	for _, name := range []string{userTaskCustomerVariable, userTaskPayloadVariable, "c8voltITRunId"} {
		variable, ok := effectiveVars[name]
		if !ok || variable.ScopeKey != seed.ProcessInstance.Key || variable.ProcessInstanceKey != seed.ProcessInstance.Key {
			return fmt.Errorf("effective process variable %q not ready with owner/scope %s: %+v", name, seed.ProcessInstance.Key, variable)
		}
	}
	return nil
}

// assertUserTaskVariableFilterTable checks the live-supported operator cases.
// $notIn is excluded across versions after C89 returned HTTP 500; request encoding remains unit-tested.
func assertUserTaskVariableFilterTable(t *testing.T, profile integrationProfile, seed userTaskVariableSeed) ([]evidenceRecord, error) {
	t.Helper()
	cases := []struct {
		name    string
		args    []string
		matches bool
	}{
		{name: "eq-match", args: []string{"--var", "incident=1"}, matches: true},
		{name: "eq-process-excluded", args: []string{"--var", "incident=99"}},
		{name: "exists-flag", args: []string{"--var-exists", "incident"}, matches: true},
		{name: "exists-operator", args: []string{"--var", "incident.$exists=true"}, matches: true},
		{name: "exists-false", args: []string{"--var", "incident.$exists=false"}},
		{name: "neq-match", args: []string{"--var", "incident.$neq=99"}, matches: true},
		{name: "neq-excluded", args: []string{"--var", "incident.$neq=1"}},
		{name: "in-match", args: []string{"--var", `incident.$in=["1","99"]`}, matches: true},
		{name: "in-excluded", args: []string{"--var", `incident.$in=["99"]`}},
		{name: "like-match", args: []string{"--var-like", "incident=1*"}, matches: true},
		{name: "like-excluded", args: []string{"--var-like", "incident=9*"}},
		{name: "process-only-excluded", args: []string{"--var", `customer="alice"`}},
	}
	var records []evidenceRecord
	var failures []error
	for _, tc := range cases {
		scenario := "user-task-vars-" + profile.Name + "-" + seed.Label + "-filter-" + tc.name
		args := []string{"get", "ut", "--pi-key", seed.ProcessInstance.Key, "--state", "created", "--json"}
		args = append(args, tc.args...)
		result := runC8VoltForProfile(t, profile.Name, scenario, args...)
		record := userTaskVariableEvidence(profile, "get user-task", scenario, result)
		record.ResourceKeys = []string{seed.Task.Key, seed.ProcessInstance.Key}
		record.CoveredFlags = []string{strings.TrimPrefix(tc.args[0], "--")}
		var tasks task.UserTasks
		err := decodeSuccessfulResult(result, &tasks)
		if err == nil {
			if tc.matches {
				if len(tasks.Items) != 1 || tasks.Items[0].Key != seed.Task.Key {
					err = fmt.Errorf("case %q returned task keys %v, want exactly %q", tc.name, userTaskKeys(tasks.Items), seed.Task.Key)
				}
			} else if len(tasks.Items) != 0 {
				err = fmt.Errorf("case %q returned task keys %v, want none", tc.name, userTaskKeys(tasks.Items))
			}
		}
		if err != nil {
			record.Outcome = "fail"
			record.FailureClass = "product"
			failures = append(failures, fmt.Errorf("%s: %w", tc.name, err))
			record.ObservedState = err.Error()
		}
		records = append(records, record)
	}
	return records, errors.Join(failures...)
}

// assertUserTaskVariablesUnsupported verifies both native filtering and effective display reject Camunda 8.7.
func assertUserTaskVariablesUnsupported(t *testing.T, profile integrationProfile) ([]evidenceRecord, error) {
	t.Helper()
	cases := []struct {
		name string
		args []string
	}{
		{name: "filter", args: []string{"get", "ut", "--var", "incident=1", "--json"}},
		{name: "effective-display", args: []string{"get", "ut", "--with-vars", "--limit", "1", "--json"}},
	}
	var records []evidenceRecord
	for _, tc := range cases {
		scenario := "user-task-vars-" + profile.Name + "-unsupported-" + tc.name
		result := runC8VoltForProfile(t, profile.Name, scenario, tc.args...)
		record := userTaskVariableEvidence(profile, "get user-task", scenario, result)
		record.VersionBehavior = "expected-unsupported"
		if result.Err == nil || !strings.Contains(strings.ToLower(result.Stdout+result.Stderr), "unsupported") {
			record.Outcome = "fail"
			record.FailureClass = "product"
			records = append(records, record)
			return records, fmt.Errorf("8.7 %s returned err=%v without an unsupported result; stdout=%q stderr=%q", tc.name, result.Err, result.Stdout, result.Stderr)
		}
		records = append(records, record)
	}
	return records, nil
}

// decodeSuccessfulResult combines command failure and payload decoding for readiness and assertion paths.
func decodeSuccessfulResult(result commandResult, value any) error {
	if result.Err != nil {
		return fmt.Errorf("command failed: %w; stdout: %s; stderr: %s", result.Err, strings.TrimSpace(result.Stdout), strings.TrimSpace(result.Stderr))
	}
	var envelope struct {
		Outcome string          `json:"outcome"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal([]byte(result.Stdout), &envelope); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if envelope.Outcome != "succeeded" || len(envelope.Payload) == 0 {
		return fmt.Errorf("expected successful envelope, got %q", result.Stdout)
	}
	if err := json.Unmarshal(envelope.Payload, value); err != nil {
		return fmt.Errorf("decode command payload: %w", err)
	}
	return nil
}

// userTaskVariableEvidence applies consistent ownership and version labels to scenario commands.
func userTaskVariableEvidence(profile integrationProfile, commandPath string, scenario string, result commandResult) evidenceRecord {
	record := commandEvidence(commandPath, scenario, result, "pass")
	record.Profile = profile.Name
	record.CamundaVersion = profile.ExpectedVersion
	record.DataOwnership = []string{"seeded", "retained"}
	return record
}

// variableMap indexes variable observations by their effective name.
func variableMap[V ~[]process.ProcessInstanceVariable](variables V) map[string]process.ProcessInstanceVariable {
	indexed := make(map[string]process.ProcessInstanceVariable, len(variables))
	for _, variable := range variables {
		indexed[variable.Name] = variable
	}
	return indexed
}

// sameJSONValue compares a serialized backend value with the typed seed value without depending on object key order.
func sameJSONValue(serialized string, expected any) bool {
	var actual any
	if err := json.Unmarshal([]byte(serialized), &actual); err != nil {
		return false
	}
	expectedJSON, err := json.Marshal(expected)
	if err != nil {
		return false
	}
	var normalizedExpected any
	if err := json.Unmarshal(expectedJSON, &normalizedExpected); err != nil {
		return false
	}
	return reflect.DeepEqual(actual, normalizedExpected)
}

// userTaskKeys returns task identities for precise assertion failures.
func userTaskKeys(items []task.UserTask) []string {
	keys := make([]string, 0, len(items))
	for _, item := range items {
		keys = append(keys, item.Key)
	}
	return keys
}

// checkUserTaskVariableCommand isolates failures and records assertion outcomes as well as subprocess errors.
func checkUserTaskVariableCommand(t *testing.T, profile integrationProfile, name string, records *[]evidenceRecord, args []string, check func(*testing.T, commandResult)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		result := runC8VoltForProfile(t, profile.Name, "user-task-vars-"+profile.Name+"-"+name, args...)
		record := userTaskVariableEvidence(profile, "get user-task", name, result)
		defer func() {
			if t.Failed() {
				record.Outcome = "fail"
				record.FailureClass = "product"
			}
			*records = append(*records, record)
		}()
		require.NoError(t, result.Err, result.Stderr)
		check(t, result)
	})
}

// assertUserTaskVariableDisplay checks that variable enrichment preserves task selection and
// complete JSON values while human output applies the display limit.
func assertUserTaskVariableDisplay(t *testing.T, profile integrationProfile, seed userTaskVariableSeed, localShadow bool, records *[]evidenceRecord) {
	t.Helper()
	args := []string{"get", "ut", "--pi-key", seed.ProcessInstance.Key}
	if localShadow {
		args = append(args, "--var", "incident=1")
	}
	checkUserTaskVariableCommand(t, profile, seed.Label+"-display-selection", records, append(append([]string{}, args...), "--json"), func(t *testing.T, result commandResult) {
		var got task.UserTasks
		require.NoError(t, decodeSuccessfulResult(result, &got))
		require.Equal(t, []string{seed.Task.Key}, userTaskKeys(got.Items))
	})
	checkUserTaskVariableCommand(t, profile, seed.Label+"-display-json", records, append(append([]string{}, args...), "--with-vars", "--var-value-limit", "10", "--json"), func(t *testing.T, result commandResult) {
		var got task.VariableEnrichedUserTasks
		require.NoError(t, decodeSuccessfulResult(result, &got))
		require.Equal(t, int64(1), got.Total)
		require.Len(t, got.Items, 1)
		require.Equal(t, seed.Task, got.Items[0].Item)
		observed := seed
		observed.EffectiveVariables = got.Items[0].Variables
		require.NoError(t, validateUserTaskVariableReadiness(observed, seed.Task.ProcessDefinitionKey, localShadow))
		vars := variableMap(observed.EffectiveVariables)
		require.Equal(t, `"alice"`, vars[userTaskCustomerVariable].Value)
		require.True(t, sameJSONValue(vars[userTaskPayloadVariable].Value, userTaskScenarioValues[userTaskPayloadVariable]))
		for name, previous := range variableMap(seed.EffectiveVariables) {
			require.Equal(t, previous.APITruncated, vars[name].APITruncated, name)
		}
	})
	checkUserTaskVariableCommand(t, profile, seed.Label+"-display-human", records, append(append([]string{}, args...), "--with-vars", "--var-value-limit", "10"), func(t *testing.T, result commandResult) {
		value := variableMap(seed.EffectiveVariables)[userTaskPayloadVariable].Value
		var compact bytes.Buffer
		require.NoError(t, json.Compact(&compact, []byte(value)))
		require.Contains(t, result.Stdout, userTaskPayloadVariable+"="+string([]rune(compact.String())[:10])+"... [cli-truncated]")
		require.Contains(t, result.Stdout, "found: 1\n")
	})
}

// seedCalledUserTaskVariables uses only existing versioned definitions and CLI discovery.
func seedCalledUserTaskVariables(t *testing.T, profile integrationProfile, child embeddedFixtureSelection) (userTaskVariableSeed, []evidenceRecord, error) {
	t.Helper()
	seed := userTaskVariableSeed{Label: "child", Fixture: child}
	childDeployments, record, err := deployEmbeddedFixtureWithLabel(t, profile, child, "vars-child")
	records := []evidenceRecord{record}
	if err != nil {
		return seed, records, err
	}
	seed.Deployments = childDeployments
	files, record, err := discoverEmbeddedFixtureFiles(t, profile)
	records = append(records, record)
	if err != nil {
		return seed, records, err
	}
	parent, err := selectEmbeddedFixtureBySuffix(profile.ExpectedVersion, files, "SimpleParentWithIncidentSubprocess.bpmn")
	if err != nil {
		return seed, records, err
	}
	parent.BpmnProcessID = embeddedFixtureBpmnProcessID(t, parent.Path)
	deployments, record, err := deployEmbeddedFixtureWithLabel(t, profile, parent, "vars-parent")
	records = append(records, record)
	if err != nil {
		return seed, records, err
	}
	roots, record, err := runSeededProcessInstanceWithVariables(t, profile, parent, deployments, userTaskScenarioValues, "vars-parent")
	records = append(records, record)
	if err != nil {
		return seed, records, err
	}
	if len(roots.Items) != 1 {
		return seed, records, fmt.Errorf("expected one parent, got %d", len(roots.Items))
	}
	root := roots.Items[0].Key
	deadline := time.Now().Add(30 * time.Second)
	for attempt := 1; ; attempt++ {
		name := fmt.Sprintf("vars-child-%s-discovery-%d", profile.Name, attempt)
		result := runC8VoltForProfile(t, profile.Name, name, "get", "pi", "--parent-key", root, "--json", "--automation")
		record = userTaskVariableEvidence(profile, "get process-instance", name, result)
		var children process.ProcessInstances
		err = decodeSuccessfulResult(result, &children)
		if err == nil && len(children.Items) == 1 {
			got := children.Items[0]
			if got.BpmnProcessId != child.BpmnProcessID || got.ProcessDefinitionKey != firstString(processDefinitionKeys(childDeployments)) || (got.ParentKey != root && got.ParentProcessInstanceKey != root) {
				err = fmt.Errorf("unexpected called process: %+v", got)
			} else {
				seed.ProcessInstance = seededProcessInstance{Key: got.Key, BpmnProcessID: got.BpmnProcessId, ProcessDefinitionKey: got.ProcessDefinitionKey}
				record.ResourceKeys = []string{root, got.Key}
				records = append(records, record)
				readyRecords, readyErr := awaitUserTaskVariableReadiness(t, profile, &seed, got.ProcessDefinitionKey, true)
				return seed, append(records, readyRecords...), readyErr
			}
		} else if err == nil {
			err = fmt.Errorf("expected one child of %s, got %d", root, len(children.Items))
		}
		record.ObservedState = fmt.Sprint(err)
		if time.Now().After(deadline) {
			record.Outcome = "fail"
			records = append(records, record)
			return seed, records, err
		}
		records = append(records, record)
		time.Sleep(time.Second)
	}
}

// TestUserTaskVariableResultContract rejects ambiguous output and duplicate paging identities.
func TestUserTaskVariableResultContract(t *testing.T) {
	for _, output := range []string{`{"payload":{"items":[]}}`, `{"outcome":"failed","payload":{"items":[]}}`, `{"outcome":"succeeded","payload":{"items":[]}} {}`} {
		var got task.UserTasks
		require.Error(t, decodeSuccessfulResult(commandResult{Stdout: output}, &got))
	}
	var got task.UserTasks
	require.NoError(t, decodeSuccessfulResult(commandResult{Stdout: `{"outcome":"succeeded","payload":{"total":0,"items":[]}}`}, &got))
	require.NotNil(t, got.Items)
	require.True(t, sameUserTaskKeySet([]string{"b", "a"}, []string{"a", "b"}))
	require.False(t, sameUserTaskKeySet([]string{"a", "a"}, []string{"a", "b"}))
	require.False(t, sameUserTaskKeySet([]string{"a"}, []string{"a", "b"}))
}
