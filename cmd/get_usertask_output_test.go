// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestGetUserTaskOutput_HumanExecution verifies keyed and search execution use
// exact compact rows, including technical identities, related keys, and omitted optional details.
func TestGetUserTaskOutput_HumanExecution(t *testing.T) {
	keyedServer, keyedRequests := newGetUserTaskCommandServer(t)
	keyedConfig := testx.WriteTestConfigForVersion(t, keyedServer.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, keyedConfig, "", "get", "ut", "-k", "2251799815391233,2251799815391234")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "2251799815391233 tenant-a approve_invoice CREATED   name:Approve invoice invoice pi:2251799813711967 ei:2251799815391200 pd:2251799813689000 assignee:alice\n2251799815391234 tenant-a archive_invoice COMPLETED                      invoice pi:2251799813711968 ei:2251799815391200 pd:2251799813689000 assignee:<unassigned>\nfound: 2\n", stdout)
	require.Equal(t, int32(2), keyedRequests.Load())

	searchServer, searchRequests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(1, false, "", "2251799815391233")
	})
	searchConfig := testx.WriteTestConfigForVersion(t, searchServer.URL, "8.9")
	stdout, stderr, err = runGetUserTaskCommand(t, searchConfig, "", "get", "ut")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	require.Equal(t, "2251799815391233 tenant-a approve_invoice CREATED name:Approve invoice invoice pi:2251799813711967 ei:2251799815391200 pd:2251799813689000 assignee:alice\nfound: 1\n", stdout)
	require.Len(t, searchRequests.snapshot(t), 1)
}

// TestGetUserTaskOutput_MachineAndUnattendedModes verifies JSON precedence,
// keys purity, quiet preservation, and prompt-free unattended execution.
func TestGetUserTaskOutput_MachineAndUnattendedModes(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantKeys bool
	}{
		{name: "keys", args: []string{"--keys-only", "get", "ut"}, wantKeys: true},
		{name: "quiet keys", args: []string{"--quiet", "--keys-only", "get", "ut"}, wantKeys: true},
		{name: "automation keys", args: []string{"--automation", "--keys-only", "get", "ut"}, wantKeys: true},
		{name: "auto-confirm keys", args: []string{"--auto-confirm", "--keys-only", "get", "ut"}, wantKeys: true},
		{name: "automation auto-confirm keys", args: []string{"--automation", "--auto-confirm", "--keys-only", "get", "ut"}, wantKeys: true},
		{name: "json precedence", args: []string{"--keys-only", "--json", "get", "ut"}},
		{name: "quiet json", args: []string{"--quiet", "--json", "get", "ut"}},
		{name: "automation json", args: []string{"--automation", "--json", "get", "ut"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
				return userTaskSearchResponse(1, false, "", "2251799815391233")
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			if test.wantKeys {
				require.Equal(t, "2251799815391233\n", stdout)
			} else {
				requireSucceededUserTaskEnvelope(t, stdout, 1)
			}
			require.Len(t, requests.snapshot(t), 1, "rendering or unattended mode added a read")
		})
	}
}

// TestGetUserTaskOutput_EmptyAndTotalModes verifies every empty output shape,
// exact numeric totals, and quiet's intentionally different machine behavior.
func TestGetUserTaskOutput_EmptyAndTotalModes(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		want      string
		wantJSON  bool
		wantTotal int64
	}{
		{name: "empty human", args: []string{"get", "ut"}, want: "found: 0\n"},
		{name: "empty quiet", args: []string{"--quiet", "get", "ut"}},
		{name: "empty keys", args: []string{"--keys-only", "get", "ut"}},
		{name: "empty quiet keys", args: []string{"--quiet", "--keys-only", "get", "ut"}},
		{name: "empty json", args: []string{"--json", "get", "ut"}, wantJSON: true},
		{name: "empty quiet json", args: []string{"--quiet", "--json", "get", "ut"}, wantJSON: true},
		{name: "total", args: []string{"get", "ut", "--total"}, want: "7\n", wantTotal: 7},
		{name: "quiet total", args: []string{"--quiet", "get", "ut", "--total"}, want: "7\n", wantTotal: 7},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
				return userTaskSearchResponse(test.wantTotal, false, "")
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			if test.wantJSON {
				requireSucceededUserTaskEnvelope(t, stdout, 0)
			} else {
				require.Equal(t, test.want, stdout)
			}
			require.Len(t, requests.snapshot(t), 1, "empty or total rendering added a read")
		})
	}
}

// TestGetUserTaskOutput_DiagnosticsStayOffStdout verifies verbose, debug, and
// activity plumbing cannot contaminate keys-only command results.
func TestGetUserTaskOutput_DiagnosticsStayOffStdout(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr bool
	}{
		{name: "verbose", args: []string{"--verbose", "--keys-only", "get", "ut"}},
		{name: "debug", args: []string{"--debug", "--keys-only", "get", "ut"}, wantStderr: true},
		{name: "debug no indicator", args: []string{"--debug", "--no-indicator", "--keys-only", "get", "ut"}, wantStderr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
				return userTaskSearchResponse(1, false, "", "2251799815391233")
			})
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Equal(t, "2251799815391233\n", stdout)
			if test.wantStderr {
				require.NotEmpty(t, stderr)
				require.NotContains(t, stdout, "DEBUG")
			} else {
				require.Empty(t, stderr)
			}
			require.NotContains(t, stdout, "Fetching")
			require.Len(t, requests.snapshot(t), 1)
		})
	}
}

// requireSucceededUserTaskEnvelope decodes exactly one successful collection
// envelope and rejects any trailing command or diagnostic payload.
func requireSucceededUserTaskEnvelope(t *testing.T, stdout string, wantTotal int64) {
	t.Helper()
	var envelope struct {
		Outcome string `json:"outcome"`
		Command string `json:"command"`
		Payload struct {
			Total int64 `json:"total"`
			Items []any `json:"items"`
		} `json:"payload"`
	}
	decoder := json.NewDecoder(strings.NewReader(stdout))
	require.NoError(t, decoder.Decode(&envelope))
	require.Equal(t, "succeeded", envelope.Outcome)
	require.Equal(t, "get user-task", envelope.Command)
	require.Equal(t, wantTotal, envelope.Payload.Total)
	require.NotNil(t, envelope.Payload.Items)
	require.Len(t, envelope.Payload.Items, int(wantTotal))
	var extra any
	require.Error(t, decoder.Decode(&extra), "JSON output must contain exactly one envelope")
}
