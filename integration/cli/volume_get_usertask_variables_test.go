// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/stretchr/testify/require"
)

// TestVolumeGetFamilyUserTaskVariables is selected by the existing volume-get Make target.
func TestVolumeGetFamilyUserTaskVariables(t *testing.T) {
	count := volumeDatasetCount(t)
	for _, profile := range requireSelectedProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			report := volumeFamilyReport{Family: "get-user-task-variables-" + profile.Name, Marker: suite.marker, DatasetCount: count, Profiles: []integrationProfile{profile}}
			defer func() {
				if t.Failed() {
					report.Records = append(report.Records, evidenceRecord{Profile: profile.Name, CamundaVersion: profile.ExpectedVersion, ScenarioName: "variable-volume-result", Outcome: "fail", FailureClass: "validation", ObservedState: "see failed scenario or setup assertion in test output"})
				}
				writeVolumeFamilyReport(t, report)
			}()
			require.NoError(t, requireProfilesReady(t, []integrationProfile{profile}))
			if profile.ExpectedVersion == "8.7" {
				records, err := assertUserTaskVariablesUnsupported(t, profile)
				report.Records = append(report.Records, records...)
				require.NoError(t, err)
				return
			}
			fixtures, records, err := discoverUserTaskVariableFixtures(t, profile)
			report.Records = append(report.Records, records...)
			require.NoError(t, err)
			deployments, record, err := deployEmbeddedFixtureWithLabel(t, profile, fixtures.Incident, "variable-volume")
			report.Records = append(report.Records, record)
			require.NoError(t, err)
			pdKey := firstString(processDefinitionKeys(deployments))
			require.NotEmpty(t, pdKey)
			scope := []string{"get", "ut", "--pd-key", pdKey, "--state", "created", "--var", "incident=1", "--batch-size", "1", "--automation"}
			query := func(name string) ([]string, error) {
				result := runC8VoltForProfile(t, profile.Name, name, append(append([]string{}, scope...), "--json")...)
				record := userTaskVariableEvidence(profile, "get user-task", name, result)
				record.DataOwnership = []string{"seeded", "preexisting"}
				var tasks task.UserTasks
				err := decodeSuccessfulResult(result, &tasks)
				keys := userTaskKeys(tasks.Items)
				if err == nil {
					seen := map[string]bool{}
					for _, key := range keys {
						if seen[key] {
							err = fmt.Errorf("duplicate task %s", key)
						}
						seen[key] = true
					}
					if tasks.Total != int64(len(keys)) {
						err = fmt.Errorf("total %d differs from returned count %d", tasks.Total, len(keys))
					}
				}
				record.ResourceKeys = keys
				if err != nil {
					record.Outcome = "fail"
					record.ObservedState = err.Error()
				}
				report.Records = append(report.Records, record)
				return keys, err
			}
			before, err := query("variable-volume-" + profile.Name + "-before")
			require.NoError(t, err)
			owned := []string{}
			for i := 0; i < count; i++ {
				label := fmt.Sprintf("variable-volume-%d", i)
				instances, record, err := runSeededProcessInstanceWithVariables(t, profile, fixtures.Incident, deployments, userTaskScenarioValues, label)
				report.Records = append(report.Records, record)
				require.NoError(t, err)
				require.Len(t, instances.Items, 1)
				seed := userTaskVariableSeed{Label: label, Fixture: fixtures.Incident, Deployments: deployments, ProcessInstance: instances.Items[0]}
				records, err := awaitUserTaskVariableReadiness(t, profile, &seed, pdKey, true)
				report.Records = append(report.Records, records...)
				require.NoError(t, err)
				owned = append(owned, seed.Task.Key)
			}
			expected := append(append([]string{}, before...), owned...)
			// Await indexing of all owned matches; never adjust the expected set to hide churn.
			deadline := time.Now().Add(30 * time.Second)
			for attempt := 1; ; attempt++ {
				keys, err := query(fmt.Sprintf("variable-volume-%s-ready-%d", profile.Name, attempt))
				require.NoError(t, err)
				if sameUserTaskKeySet(keys, expected) {
					break
				}
				if time.Now().After(deadline) {
					t.Fatalf("filtered scope did not stabilize: before=%v owned=%v actual=%v", before, owned, keys)
				}
				time.Sleep(time.Second)
			}
			for _, mode := range []string{"json", "keys-only", "total", "limit"} {
				args := append([]string{}, scope...)
				if mode == "limit" {
					args = append(args, "--limit", "2", "--json")
				} else {
					args = append(args, "--"+mode)
				}
				checkUserTaskVariableCommand(t, profile, "volume-"+mode, &report.Records, args, func(t *testing.T, result commandResult) {
					switch mode {
					case "json", "limit":
						var got task.UserTasks
						require.NoError(t, decodeSuccessfulResult(result, &got))
						keys := userTaskKeys(got.Items)
						require.Equal(t, int64(len(keys)), got.Total)
						if mode == "limit" {
							require.Len(t, keys, 2)
							require.NotEqual(t, keys[0], keys[1])
							for _, key := range keys {
								require.Contains(t, expected, key)
							}
						} else {
							require.True(t, sameUserTaskKeySet(keys, expected), "expected %v got %v", expected, keys)
						}
					case "keys-only":
						require.NoError(t, validateKeysOnlyString(result.Stdout))
						require.True(t, strings.HasSuffix(result.Stdout, "\n"))
						require.True(t, sameUserTaskKeySet(strings.Split(strings.TrimSuffix(result.Stdout, "\n"), "\n"), expected))
					case "total":
						require.Equal(t, fmt.Sprintf("%d\n", len(expected)), result.Stdout)
					}
				})
			}
			for _, mode := range []string{"human", "json", "keys-only"} {
				// Contradictory equality predicates are empty even if preexisting tasks have other values.
				args := append(append([]string{}, scope...), "--var", "incident=99")
				if mode != "human" {
					args = append(args, "--"+mode)
				}
				checkUserTaskVariableCommand(t, profile, "volume-empty-"+mode, &report.Records, args, func(t *testing.T, result commandResult) {
					switch mode {
					case "human":
						require.Equal(t, "found: 0\n", result.Stdout)
					case "keys-only":
						require.Empty(t, result.Stdout)
					case "json":
						var got task.UserTasks
						require.NoError(t, decodeSuccessfulResult(result, &got))
						require.Zero(t, got.Total)
						require.NotNil(t, got.Items)
						require.Empty(t, got.Items)
					}
				})
			}
		})
	}
}

func sameUserTaskKeySet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	seen := make(map[string]bool, len(want))
	for _, key := range want {
		if seen[key] {
			return false
		}
		seen[key] = true
	}
	for _, key := range got {
		if !seen[key] {
			return false
		}
		delete(seen, key)
	}
	return len(seen) == 0
}
