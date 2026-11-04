// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/job"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/c8volt/task"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	"github.com/stretchr/testify/require"
)

// TestUpdateFamilyUserTaskCollections verifies real task outcomes, including an
// untouched control, rather than treating accepted responses as postconditions.
func TestUpdateFamilyUserTaskCollections(t *testing.T) {
	for _, profile := range selectedBatchProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			f := newSuspendedFixture(t, profile)
			keys := f.dataset.PositiveProcessInstanceKeys
			var tasks []task.UserTask
			for _, key := range keys {
				require.Eventually(t, func() bool {
					result := f.run(t, "task-"+key, "get", "user-task", "--json", "--pi-key", key)
					var found task.UserTasks
					if result.Err != nil || decodeCommandPayload(result.Stdout, &found) != nil || len(found.Items) != 1 {
						return false
					}
					tasks = append(tasks, found.Items[0])
					return true
				}, 30*time.Second, time.Second)
			}
			// Task writes must update the existing winning scope; new names stay
			// local, and shadowed root values/unrelated names remain unchanged.
			rootUpdate := f.run(t, "root-variable-setup", "update", "process-instance", "--key", keys[0], "--set-vars", `{"rootOwned":"root","shadowed":"root","rootKeep":"preserved"}`, "--automation", "--json")
			require.NoError(t, rootUpdate.Err, rootUpdate.Stderr)
			localPreview := f.run(t, "local-variable-preview", "update", "element", "--key", tasks[0].ElementInstanceKey, "--set-vars", `{"shadowed":"local","localKeep":"preserved"}`, "--dry-run", "--automation", "--json")
			require.NoError(t, localPreview.Err, localPreview.Stderr)
			require.NoError(t, requireVolumeEnvelopeOutcome(localPreview.Stdout, "succeeded"))
			localUpdate := f.run(t, "local-variable-setup", "update", "element", "--key", tasks[0].ElementInstanceKey, "--set-vars", `{"shadowed":"local","localKeep":"preserved"}`, "--automation", "--json")
			require.NoError(t, localUpdate.Err, localUpdate.Stderr)
			taskUpdate := f.run(t, "task-variable-ownership", "update", "user-task", "--key", tasks[0].Key, "--set-vars", `{"rootOwned":"updated","shadowed":"updated","taskNew":"created"}`, "--automation", "--json")
			require.NoError(t, taskUpdate.Err, taskUpdate.Stderr)
			require.Eventually(t, func() bool {
				result := f.run(t, "task-variable-ownership-observed", "get", "user-task", "--key", tasks[0].Key, "--with-vars", "--json")
				var found task.VariableEnrichedUserTasks
				if result.Err != nil || decodeCommandPayload(result.Stdout, &found) != nil || len(found.Items) != 1 {
					return false
				}
				variables := variableMap(found.Items[0].Variables)
				for name, want := range map[string]struct{ value, scope string }{
					"rootOwned": {"updated", keys[0]}, "rootKeep": {"preserved", keys[0]},
					"shadowed": {"updated", tasks[0].ElementInstanceKey}, "taskNew": {"created", tasks[0].ElementInstanceKey},
					"localKeep": {"preserved", tasks[0].ElementInstanceKey},
				} {
					got, ok := variables[name]
					if !ok || got.ScopeKey != want.scope || !sameJSONValue(got.Value, want.value) {
						return false
					}
				}
				return true
			}, 30*time.Second, time.Second)
			rootResult := f.run(t, "shadowed-root-preserved", "get", "process-instance", "--key", keys[0], "--with-vars", "--json")
			var roots process.VariableEnrichedProcessInstances
			require.NoError(t, decodeCommandPayload(rootResult.Stdout, &roots))
			require.Len(t, roots.Items, 1)
			require.True(t, sameJSONValue(variableMap(roots.Items[0].Variables)["shadowed"].Value, "root"))
			selectKeys := []string{"--key", tasks[0].Key, "--key", tasks[1].Key}
			for _, action := range []struct {
				name     string
				flags    []string
				assignee string
			}{
				{"assign", []string{"--set-assignee", "alice"}, "alice"},
				{"unassign", []string{"--unassign"}, ""},
			} {
				args := append([]string{"update", "user-task", "--automation", "--json"}, selectKeys...)
				args = append(args, action.flags...)
				preview := f.run(t, action.name+"-preview", append(args, "--dry-run")...)
				require.NoError(t, preview.Err, preview.Stderr)
				require.NoError(t, requireVolumeEnvelopeOutcome(preview.Stdout, "succeeded"))
				result := f.run(t, action.name, args...)
				require.NoError(t, result.Err, result.Stderr)
				require.NoError(t, requireVolumeEnvelopeOutcome(result.Stdout, "accepted"))
				for i, item := range tasks {
					want := action.assignee
					if i == 2 {
						want = ""
					}
					require.Eventually(t, func() bool {
						result := f.run(t, action.name+"-observe-"+item.Key, "get", "user-task", "--json", "--key", item.Key)
						var found task.UserTasks
						return result.Err == nil && decodeCommandPayload(result.Stdout, &found) == nil && len(found.Items) == 1 && found.Items[0].Assignee == want
					}, 30*time.Second, time.Second)
				}
			}
			args := append([]string{"update", "user-task", "--automation", "--json"}, selectKeys...)
			args = append(args, "--complete", "--set-vars", `{"approved":true}`)
			result := f.run(t, "complete-collection", args...)
			require.NoError(t, result.Err, result.Stderr)
			require.NoError(t, requireVolumeEnvelopeOutcome(result.Stdout, "accepted"))
			for _, key := range keys[:2] {
				require.NoError(t, requireProcessInstanceStateEventually(t, profile, key, []string{"COMPLETED"}, 30*time.Second))
			}
			require.NoError(t, requireProcessInstanceState(t, profile, keys[2], "ACTIVE"))
			bid := suite.marker + "-assigned"
			result = f.run(t, "assign-business-id", "update", "process-instance", "--automation", "--json", "--key", keys[2], "--set-bid", bid)
			require.NoError(t, result.Err, result.Stderr)
			require.Eventually(t, func() bool {
				result := f.run(t, "select-business-id", "get", "process-instance", "--json", "--bid", bid)
				var found process.ProcessInstances
				return result.Err == nil && decodeCommandPayload(result.Stdout, &found) == nil && len(found.Items) == 1 && found.Items[0].Key == keys[2]
			}, 30*time.Second, time.Second)
			writeDataEvidence(t, "operator-task-postconditions-"+profile.Name+".json", map[string]any{
				"instanceKeys": keys, "taskKeys": userTaskKeys(tasks), "businessId": bid,
				"assignmentObserved": true, "unassignmentObserved": true, "selectedInstancesCompleted": true,
				"controlInstanceActive": true, "variableOwnershipPreserved": true, "shadowedRootPreserved": true,
			})
		})
	}
}

// Element and task writes need ordered scope/assignment preparation. The
// operator workflow owns these scenarios instead of the process-only runner.
func operatorUpdateBehaviorScenarios() []behaviorScenario {
	var scenarios []behaviorScenario
	for _, operation := range []struct {
		path, preview, execution string
		flags                    []string
	}{
		{"update element", "local-variable-preview", "local-variable-setup", []string{"key", "set-vars"}},
		{"update user-task", "assign-preview", "assign", []string{"key", "set-assignee"}},
	} {
		for _, preview := range []bool{true, false} {
			behavior, name := behaviorConfirmedMutation, operation.execution
			flags := append([]string{}, operation.flags...)
			if preview {
				behavior, name = behaviorPreview, operation.preview
				flags = append(flags, "dry-run")
			}
			scenarios = append(scenarios, behaviorScenario{
				CommandPath: operation.path, Name: "TestUpdateFamilyUserTaskCollections/" + name,
				RequiresProfile: true, OutputMode: "json", Behavior: behavior,
				VersionBehavior: "observed-8.10-before-mutation", DataOwnership: []string{"seeded"},
				Preview: preview, ConfirmedMutation: !preview, WantJSON: true, CoveredFlags: flags,
			})
		}
	}
	return scenarios
}

// TestUpdateFamilyActivatedJobTimeout covers the prerequisite discovered during
// the documentation audit and independently observes deadline and completion.
func TestUpdateFamilyActivatedJobTimeout(t *testing.T) {
	for _, profile := range selectedBatchProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			f := newSuspendedFixtureBase(t, profile)
			id := "C810_Timeout_" + sanitizeEvidenceName(suite.marker)
			data, err := os.ReadFile(filepath.Join(suite.repoRoot, "embedded", "processdefinitions", "C810_SimpleServiceTask.bpmn"))
			require.NoError(t, err)
			// Reuse the embedded service task while removing its zero-retry start
			// listener and assigning a unique worker type for isolated activation.
			model := strings.ReplaceAll(string(data), "C810_SimpleServiceTask", id)
			model = regexp.MustCompile(`(?s)\s*<zeebe:executionListeners>.*?</zeebe:executionListeners>`).ReplaceAllString(model, "")
			model = strings.ReplaceAll(model, `type="SimpleServiceTaskWorker" retries="0"`, `type="`+id+`" retries="3"`)
			path := filepath.Join(suite.workDir, id+".bpmn")
			require.NoError(t, os.WriteFile(path, []byte(model), 0600))
			deployed := f.run(t, "deploy-job", "deploy", "process-definition", "--automation", "--json", "--file", path)
			require.NoError(t, deployed.Err, deployed.Stderr)
			var definitions []seededDeployment
			require.NoError(t, decodeCommandPayload(deployed.Stdout, &definitions))
			f.dataset.PositiveProcessDefinitionKeys = processDefinitionKeys(definitions)
			require.Len(t, f.dataset.PositiveProcessDefinitionKeys, 1)
			instances, record, err := runVolumeProcessInstances(t, profile, embeddedFixtureSelection{BpmnProcessID: id}, definitions, 1, "activated-timeout")
			f.records = append(f.records, record)
			f.dataset.PositiveProcessInstanceKeys = processInstanceKeys(instances)
			require.NoError(t, err)
			require.Len(t, f.dataset.PositiveProcessInstanceKeys, 1)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			var activated camundav810.ActivatedJobResult
			requestTimeout := int64(-1)
			withLease := false
			tenantFilter := camundav810.PROVIDED
			tenant := profile.Tenant
			if tenant == "" {
				tenant = "<default>"
			}
			tenants := []string{tenant}
			attempt := 0
			// There is no activation command; direct activation is fixture setup,
			// scoped by the unique job type and verified against the owned instance.
			require.Eventually(t, func() bool {
				response, err := f.client.generated.ActivateJobsWithResponse(ctx, camundav810.ActivateJobsJSONRequestBody{Type: id, Timeout: 300000, MaxJobsToActivate: 1, RequestTimeout: &requestTimeout, TenantIds: &tenants, TenantFilter: &tenantFilter, Worker: &suite.marker, WithLease: &withLease})
				attempt++
				if response != nil {
					writeDataEvidence(t, "job-activation-attempt-"+profile.Name+"-"+time.Now().Format("150405.000")+".json", map[string]any{"attempt": attempt, "status": response.StatusCode(), "body": json.RawMessage(response.Body)})
				}
				if err != nil || response == nil || response.JSON200 == nil || len(response.JSON200.Jobs) == 0 {
					return false
				}
				activated = response.JSON200.Jobs[0]
				return true
			}, 20*time.Second, time.Second)
			require.Equal(t, f.dataset.PositiveProcessInstanceKeys[0], activated.ProcessInstanceKey)
			writeDataEvidence(t, "job-activation-"+profile.Name+".json", map[string]any{"setup": "native activation of owned unique job type", "key": activated.JobKey, "processInstanceKey": activated.ProcessInstanceKey, "deadline": activated.Deadline})
			result := f.run(t, "timeout-update", "update", "job", "--automation", "--json", "--key", activated.JobKey, "--set-timeout", "2m", "--timeout", "30s")
			require.NoError(t, result.Err, result.Stderr)
			require.NoError(t, validateRealStateJobUpdateResult(result, "submitted"))
			require.Eventually(t, func() bool {
				result := f.run(t, "timeout-observation", "get", "job", "--json", "--key", activated.JobKey)
				var found job.Job // Explicit-key lookup returns a scalar job.
				if result.Err != nil || decodeCommandPayload(result.Stdout, &found) != nil || found.Key != activated.JobKey || found.Deadline == nil {
					return false
				}
				deadline := found.Deadline
				return deadline.UnixMilli() < activated.Deadline-60000 && deadline.After(time.Now().Add(time.Minute))
			}, 30*time.Second, time.Second)
			require.Eventually(t, func() bool {
				result := f.run(t, "timeout-state-filter", "get", "job", "--json", "--state", "timeout_updated", "--type", id)
				var found job.SearchResult
				return result.Err == nil && decodeCommandPayload(result.Stdout, &found) == nil && len(found.Items) == 1 && found.Items[0].Key == activated.JobKey
			}, 30*time.Second, time.Second)
			bid := suite.marker + "-job-complete"
			result = f.run(t, "job-complete-business-id", "update", "job", "--automation", "--json", "--key", activated.JobKey, "--complete", "--set-bid", bid, "--set-vars", `{"approved":true}`)
			require.NoError(t, result.Err, result.Stderr)
			require.NoError(t, requireProcessInstanceStateEventually(t, profile, activated.ProcessInstanceKey, []string{"COMPLETED"}, 30*time.Second))
			require.Eventually(t, func() bool {
				result = f.run(t, "job-completion-business-id-observed", "get", "process-instance", "--json", "--key", activated.ProcessInstanceKey)
				var found process.ProcessInstances
				return result.Err == nil && decodeCommandPayload(result.Stdout, &found) == nil && len(found.Items) == 1 && found.Items[0].BusinessId != nil && *found.Items[0].BusinessId == bid
			}, 30*time.Second, time.Second)
			var document json.RawMessage
			require.NoError(t, json.Unmarshal([]byte(result.Stdout), &document), "one complete JSON document required")
			writeDataEvidence(t, "operator-job-postconditions-"+profile.Name+".json", map[string]any{
				"jobKey": activated.JobKey, "processInstanceKey": activated.ProcessInstanceKey,
				"deadlineChangeObserved": true, "processCompleted": true, "businessId": bid,
			})
		})
	}
}
