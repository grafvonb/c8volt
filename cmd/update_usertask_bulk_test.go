// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestUpdateUserTaskCommand_SharedScopeWritesOnce verifies two selected tasks
// fan out one accepted inherited-scope mutation without duplicate transport.
func TestUpdateUserTaskCommand_SharedScopeWritesOnce(t *testing.T) {
	server, capture := newUpdateUserTaskCommandServer(t)
	configPath := writeTestConfigForVersion(t, server.URL, "8.10")

	stdout, stderr, err := runUpdateUserTaskCommand(t, configPath, "",
		"--automation", "--json", "update", "user-task",
		"--key", "2251799815391233,2251799815391234",
		"--vars", `{"approved":true}`, "--no-wait", "--workers", "2",
	)
	require.NoError(t, err)
	require.Empty(t, stderr)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	require.Equal(t, "accepted", envelope["outcome"])
	payload := requireJSONObject(t, envelope["payload"])
	items := requireJSONItems(t, payload["items"], 2)
	for _, raw := range items {
		item := requireJSONObject(t, raw)
		require.Equal(t, "submitted", item["status"])
	}
	tenantContext := requireJSONObject(t, envelope["tenantContext"])
	require.Equal(t, []any{"tenant-a"}, tenantContext["resolvedTenantIds"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 1, capture.mutations)
	require.Len(t, capture.variableReads, 2, "--no-wait must skip confirmation reads")
}

// TestUpdateUserTaskCommand_DryRunMergesAndDeduplicatesExplicitKeys verifies
// repeated, comma-separated, and stdin keys retain first-input order.
func TestUpdateUserTaskCommand_DryRunMergesAndDeduplicatesExplicitKeys(t *testing.T) {
	server, capture := newUpdateUserTaskCommandServer(t)
	configPath := writeTestConfigForVersion(t, server.URL, "8.10")

	stdout, stderr, err := runUpdateUserTaskCommand(t, configPath, "2251799815391234\n2251799815391233\n",
		"--json", "update", "user-tasks", "-",
		"--key", "2251799815391233,2251799815391233",
		"--vars", `{"approved":true}`, "--dry-run",
	)
	require.NoError(t, err)
	require.Empty(t, stderr)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
	payload := requireJSONObject(t, envelope["payload"])
	require.Equal(t, []any{"2251799815391233", "2251799815391234"}, payload["requestedKeys"])
	require.Equal(t, float64(2), payload["requestedCount"])
	require.Equal(t, float64(2), payload["updateCount"])
	require.Equal(t, false, payload["mutationSubmitted"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Zero(t, capture.mutations)
	require.ElementsMatch(t, []string{"2251799815391233", "2251799815391234"}, capture.taskReads)
}

// TestUpdateUserTaskCommand_VarsFileMatchesInlinePlanning verifies both payload
// sources reach the same frozen command plan.
func TestUpdateUserTaskCommand_VarsFileMatchesInlinePlanning(t *testing.T) {
	server, _ := newUpdateUserTaskCommandServer(t)
	configPath := writeTestConfigForVersion(t, server.URL, "8.10")
	path := t.TempDir() + "/vars.json"
	require.NoError(t, os.WriteFile(path, []byte(`{"approved":true}`), 0o600))

	inline, _, err := runUpdateUserTaskCommand(t, configPath, "", "--json", "update", "uts", "--key", "2251799815391233", "--vars", `{"approved":true}`, "--dry-run")
	require.NoError(t, err)
	fromFile, _, err := runUpdateUserTaskCommand(t, configPath, "", "--json", "update", "uts", "--key", "2251799815391233", "--vars-file", path, "--dry-run")
	require.NoError(t, err)
	require.JSONEq(t, inline, fromFile)
}

// TestUpdateUserTaskCommand_LocalValidationStopsBeforeRequests covers input
// failures that must not start task discovery.
func TestUpdateUserTaskCommand_LocalValidationStopsBeforeRequests(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing keys", args: []string{"update", "ut", "--vars", `{"approved":true}`}, want: "no user task keys provided"},
		{name: "invalid workers", args: []string{"update", "ut", "--key", "2251799815391233", "--vars", `{"approved":true}`, "--workers", "0"}, want: "--workers must be positive integer"},
		{name: "missing payload", args: []string{"update", "ut", "--key", "2251799815391233"}, want: "--vars or --vars-file is required"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capture := newUpdateUserTaskCommandServer(t)
			configPath := writeTestConfigForVersion(t, server.URL, "8.10")
			_, stderr, err := runUpdateUserTaskCommand(t, configPath, "", test.args...)
			require.Error(t, err)
			require.Contains(t, stderr, test.want)
			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Zero(t, capture.requests)
		})
	}
}
