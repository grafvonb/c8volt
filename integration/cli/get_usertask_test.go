// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build integration

package cli_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/stretchr/testify/require"
)

// TestGetFamilyUserTask checks real user-task reads with the existing seeded fixture.
func TestGetFamilyUserTask(t *testing.T) {
	for _, profile := range requireSelectedProfiles(t) {
		t.Run(profile.Name, func(t *testing.T) {
			require.NoError(t, requireProfilesReady(t, []integrationProfile{profile}))
			if profile.ExpectedVersion == "8.7" {
				runGetUserTaskScenarios(t, profile, "", nil, false)
				return
			}
			seed, records, cleanup, err := seedProfileData(t, profile)
			writeJSON(t, "user-task-seed-"+profile.Name+".json", seededDataReport{Profiles: []seededProfileData{seed}, Records: records, Cleanup: cleanup})
			require.NoError(t, err)
			runGetUserTaskScenarios(t, profile, seed.ProcessDefinitionKeys[0], seed.ProcessInstanceKeys, false)
		})
	}
}

// runGetUserTaskScenarios exercises task output and paging using suite-owned process keys.
func runGetUserTaskScenarios(t *testing.T, profile integrationProfile, pdKey string, piKeys []string, volume bool) {
	t.Helper()
	var records []evidenceRecord
	defer func() { writeJSON(t, fmt.Sprintf("user-task-%s-volume-%t.json", profile.Name, volume), records) }()
	if profile.ExpectedVersion == "8.7" {
		result := runC8VoltForProfile(t, profile.Name, "user-task-unsupported", "get", "ut", "--json", "--limit", "1")
		record := commandEvidence("get user-task", "user-task-unsupported", result, "pass")
		record.Profile = profile.Name
		defer func() {
			if t.Failed() {
				record.Outcome = "fail"
			}
			records = append(records, record)
		}()
		require.Error(t, result.Err)
		require.Contains(t, strings.ToLower(result.Stdout+result.Stderr), "unsupported")
		return
	}
	var expectedKeys []string
	var sample task.UserTask
	for _, piKey := range piKeys {
		var tasks task.UserTasks
		require.Eventually(t, func() bool {
			result := runC8VoltForProfile(t, profile.Name, "user-task-discover-"+piKey, "get", "ut", "--pi-key", piKey, "--state", "created", "--json")
			record := commandEvidence("get user-task", "user-task-discover-"+piKey, result, "pass")
			record.Profile = profile.Name
			if result.Err != nil {
				record.Outcome = "fail"
			}
			records = append(records, record)
			return result.Err == nil && decodeCommandPayload(result.Stdout, &tasks) == nil && len(tasks.Items) == 1
		}, 30*time.Second, time.Second, "native task did not become visible for process %s", piKey)
		require.Equal(t, int64(1), tasks.Total)
		sample = tasks.Items[0]
		require.Equal(t, piKey, sample.ProcessInstanceKey)
		require.Equal(t, pdKey, sample.ProcessDefinitionKey)
		require.Equal(t, "CREATED", sample.State)
		require.NotEmpty(t, sample.Key)
		require.NotEmpty(t, sample.ElementInstanceKey)
		expectedKeys = append(expectedKeys, sample.Key)
	}
	require.NotEmpty(t, expectedKeys)
	details := []string{sample.Key, sample.TenantId, sample.ElementId, sample.State}
	if sample.Name != "" {
		details = append(details, "name:"+sample.Name)
	}
	if sample.Assignee != "" {
		details = append(details, "assignee:"+sample.Assignee)
	}
	details = append(details, sample.ProcessDefinitionId, "pi:"+sample.ProcessInstanceKey, "ei:"+sample.ElementInstanceKey, "pd:"+sample.ProcessDefinitionKey)
	var fields []string
	for _, field := range details {
		if field != "" {
			fields = append(fields, field)
		}
	}
	cases := []struct {
		name  string
		args  []string
		want  string
		json  bool
		count int
	}{
		{name: "key-human", args: []string{"get", "user-task", "-k", sample.Key}, want: strings.Join(fields, " ") + "\nfound: 1\n"},
		{name: "key-json", args: []string{"get", "uts", "-k", sample.Key, "--json"}, json: true, count: 1},
		{name: "key-keys", args: []string{"get", "user-tasks", "-k", sample.Key, "--keys-only"}, want: sample.Key + "\n"},
		{name: "filtered-search", args: []string{"get", "ut", "--pi-key", sample.ProcessInstanceKey, "--element-id", sample.ElementId, "--state", "created", "--json"}, json: true, count: 1},
		{name: "empty-human", args: []string{"get", "ut", "--pi-key", sample.ProcessInstanceKey, "--state", "completed"}, want: "found: 0\n"},
		{name: "empty-json", args: []string{"get", "ut", "--pi-key", sample.ProcessInstanceKey, "--state", "completed", "--quiet", "--json"}, json: true},
		{name: "empty-keys", args: []string{"get", "ut", "--pi-key", sample.ProcessInstanceKey, "--state", "completed", "--quiet", "--keys-only"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := runC8VoltForProfile(t, profile.Name, "user-task-"+tc.name, tc.args...)
			record := commandEvidence("get user-task", "user-task-"+tc.name, result, "pass")
			record.Profile = profile.Name
			defer func() {
				if t.Failed() {
					record.Outcome = "fail"
				}
				records = append(records, record)
			}()
			require.NoError(t, result.Err, result.Stderr)
			if !tc.json {
				require.Equal(t, tc.want, result.Stdout)
				return
			}
			var envelope struct {
				Outcome string         `json:"outcome"`
				Command string         `json:"command"`
				Payload task.UserTasks `json:"payload"`
			}
			require.NoError(t, json.Unmarshal([]byte(result.Stdout), &envelope))
			require.Equal(t, "succeeded", envelope.Outcome)
			require.Equal(t, "get user-task", envelope.Command)
			require.Equal(t, int64(tc.count), envelope.Payload.Total)
			require.Len(t, envelope.Payload.Items, tc.count)
			require.NotNil(t, envelope.Payload.Items)
			if tc.count == 1 {
				require.Equal(t, sample, envelope.Payload.Items[0])
			}
		})
	}
	if !volume {
		return
	}
	require.GreaterOrEqual(t, len(expectedKeys), 3)
	scope := []string{"get", "ut", "--pd-key", pdKey, "--state", "created", "--batch-size", "1", "--automation"}
	var allKeys []string
	for _, mode := range []string{"--json", "--keys-only", "--total", "--limit"} {
		t.Run("paging-"+mode, func(t *testing.T) {
			args := append([]string(nil), scope...)
			args = append(args, mode)
			if mode == "--limit" {
				args = append(args, "2", "--json")
			}
			result := runC8VoltForProfile(t, profile.Name, "user-task-paging-"+mode, args...)
			record := commandEvidence("get user-task", "user-task-paging-"+mode, result, "pass")
			record.Profile = profile.Name
			defer func() {
				if t.Failed() {
					record.Outcome = "fail"
				}
				records = append(records, record)
			}()
			require.NoError(t, result.Err, result.Stderr)
			switch mode {
			case "--json", "--limit":
				var tasks task.UserTasks
				require.NoError(t, decodeCommandPayload(result.Stdout, &tasks))
				require.Equal(t, int64(len(tasks.Items)), tasks.Total)
				keys := make([]string, 0, len(tasks.Items))
				seen := map[string]bool{}
				for _, item := range tasks.Items {
					require.Equal(t, pdKey, item.ProcessDefinitionKey)
					require.Equal(t, "CREATED", item.State)
					require.False(t, seen[item.Key], "duplicate task %s", item.Key)
					seen[item.Key] = true
					keys = append(keys, item.Key)
				}
				if mode == "--limit" {
					require.Len(t, keys, 2)
					for _, key := range keys {
						require.Contains(t, allKeys, key)
					}
				} else {
					allKeys = keys
					for _, key := range expectedKeys {
						require.Contains(t, allKeys, key, "seeded task missing after paging")
					}
				}
			case "--keys-only":
				require.NoError(t, validateKeysOnlyString(result.Stdout))
				require.True(t, strings.HasSuffix(result.Stdout, "\n"))
				require.ElementsMatch(t, allKeys, strings.Split(strings.TrimSuffix(result.Stdout, "\n"), "\n"))
			case "--total":
				require.Equal(t, fmt.Sprintf("%d\n", len(allKeys)), result.Stdout)
			}
		})
	}
}
