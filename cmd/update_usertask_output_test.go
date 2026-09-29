// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// updateUserTaskOutputCapture records remote work independently of rendering.
type updateUserTaskOutputCapture struct {
	mu            sync.Mutex
	initialValue  bool
	failingScope  string
	mutatedScopes map[string]bool
	mutations     int
	variableReads int
}

// TestUpdateUserTaskOutputModes exercises dry-run, no-op, confirmed, and
// accepted execution through the command path across human and machine modes.
func TestUpdateUserTaskOutputModes(t *testing.T) {
	tests := []struct {
		name         string
		initialValue bool
		args         []string
		wantOutput   string
		wantOutcome  string
		wantKeys     string
		keysMode     bool
		humanMode    bool
		wantMutates  int
		wantReads    int
	}{
		{
			name:      "dry run human",
			humanMode: true,
			args:      []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--dry-run"},
			wantOutput: "INFO selection scope: explicit resource keys; tenant filter not applied\n" +
				"INFO affected tenants: tenant-a\n" +
				"INFO dry run: update user-task variables: 1 user task(s), 1 change(s), 0 addition(s), 0 unchanged, 0 untouched; no changes applied\n" +
				"INFO 101: ~ approved (inherited scope 201): false -> true\n",
			wantReads: 1,
		},
		{
			name:      "dry run keys",
			args:      []string{"--keys-only", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--dry-run"},
			wantKeys:  "101\n",
			keysMode:  true,
			wantReads: 1,
		},
		{
			name:        "dry run quiet json wins over keys",
			args:        []string{"--quiet", "--keys-only", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--dry-run", "--no-wait"},
			wantOutcome: "succeeded",
			wantReads:   1,
		},
		{
			name:         "no-op human",
			humanMode:    true,
			initialValue: true,
			args:         []string{"update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantOutput: "INFO selection scope: explicit resource keys; tenant filter not applied\n" +
				"INFO affected tenants: tenant-a\n" +
				"INFO plan: update user-task variables: nothing to update (1 requested value(s) already match visible variables); no confirmation required\n",
			wantReads: 1,
		},
		{
			name:         "no-op no-wait json remains succeeded",
			initialValue: true,
			args:         []string{"--automation", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait"},
			wantOutcome:  "succeeded",
			wantReads:    1,
		},
		{
			name:         "no-op keys is zero bytes",
			initialValue: true,
			args:         []string{"--keys-only", "update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantKeys:     "",
			keysMode:     true,
			wantReads:    1,
		},
		{
			name:      "confirmed human",
			humanMode: true,
			args:      []string{"--auto-confirm", "update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantOutput: "INFO updated user-task 101: confirmed\n" +
				"INFO updated: 1 (confirmed/submitted: 1, unchanged: 0, failed: 0, skipped: 0)\n",
			wantMutates: 1,
			wantReads:   2,
		},
		{
			name:        "accepted quiet keys",
			args:        []string{"--quiet", "--keys-only", "--automation", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait"},
			wantKeys:    "101\n",
			keysMode:    true,
			wantMutates: 1,
			wantReads:   1,
		},
		{
			name:        "accepted json",
			args:        []string{"--automation", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait"},
			wantOutcome: "accepted",
			wantMutates: 1,
			wantReads:   1,
		},
		{
			name:        "quiet human suppresses result",
			humanMode:   true,
			args:        []string{"--quiet", "--auto-confirm", "update", "ut", "--key", "101", "--vars", `{"approved":true}`},
			wantOutput:  "",
			wantMutates: 1,
			wantReads:   2,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capture := newUpdateUserTaskOutputServer(t, test.initialValue, "")
			configPath := writeTestConfigForVersion(t, server.URL, "8.10")
			stdout, stderr, err := runUpdateUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err)
			switch {
			case test.wantOutcome != "":
				require.Empty(t, stderr)
				envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
				require.Equal(t, test.wantOutcome, envelope["outcome"])
			case test.keysMode:
				require.Empty(t, stderr)
				require.Equal(t, test.wantKeys, stdout)
			case test.humanMode:
				require.Empty(t, stdout)
				require.Equal(t, test.wantOutput, normalizeUpdateUserTaskHumanOutput(stderr))
			}

			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Equal(t, test.wantMutates, capture.mutations)
			require.Equal(t, test.wantReads, capture.variableReads)
		})
	}
}

// normalizeUpdateUserTaskHumanOutput removes logger timestamps while retaining
// severity, wording, ordering, and stream-routing assertions.
func normalizeUpdateUserTaskHumanOutput(output string) string {
	return regexp.MustCompile(`(?m)^\d{2}:\d{2}:\d{2}\.\d{3} (INFO|WARN) `).ReplaceAllString(output, `$1 `)
}

// TestUpdateUserTaskPartialFailureJSON renders accepted and failed task facts
// once in the normalized error envelope and exits with a failure code.
func TestUpdateUserTaskPartialFailureJSON(t *testing.T) {
	server, capture := newUpdateUserTaskOutputServer(t, false, "202")
	configPath := writeTestConfigForVersion(t, server.URL, "8.10")
	stdout, _, err := runUpdateUserTaskCommand(t, configPath, "",
		"--automation", "--json", "update", "ut",
		"--key", "101,102", "--vars", `{"approved":true}`,
		"--no-wait", "--workers", "1",
	)
	require.Error(t, err)

	envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
	require.Equal(t, "failed", envelope["outcome"])
	require.NotNil(t, envelope["detail"])
	items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 2)
	require.Equal(t, "submitted", requireJSONObject(t, items[0])["status"])
	require.Equal(t, "mutation_failed", requireJSONObject(t, items[1])["status"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 2, capture.mutations)
	require.Equal(t, 2, capture.variableReads, "no-wait performs planning reads only")
}

// decodeSingleUserTaskUpdateEnvelope requires exactly one JSON value and EOF.
func decodeSingleUserTaskUpdateEnvelope(t *testing.T, output string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewBufferString(output))
	var envelope map[string]any
	require.NoError(t, decoder.Decode(&envelope))
	require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
	return envelope
}

// newUpdateUserTaskOutputServer provides deterministic independent scopes so
// command output tests can observe no-op, acceptance, confirmation, and partial failure.
func newUpdateUserTaskOutputServer(t *testing.T, initialValue bool, failingScope string) (*httptest.Server, *updateUserTaskOutputCapture) {
	t.Helper()
	capture := &updateUserTaskOutputCapture{
		initialValue:  initialValue,
		failingScope:  failingScope,
		mutatedScopes: make(map[string]bool),
	}
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch {
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2/user-tasks/"):
			key := strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/")
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"userTaskKey":%q,"state":"CREATED","elementId":"approve","processInstanceKey":"301","elementInstanceKey":%q,"processDefinitionKey":"401","tenantId":"tenant-a"}`, key, "501"+key)
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/effective-variables/search"):
			key := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/user-tasks/"), "/effective-variables/search")
			scope := updateUserTaskOutputScope(key)
			capture.mu.Lock()
			capture.variableReads++
			value := capture.initialValue || capture.mutatedScopes[scope]
			capture.mu.Unlock()
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":%q,"variableKey":%q,"processInstanceKey":"301","scopeKey":%q,"tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, fmt.Sprint(value), "601"+key, scope)
		case request.Method == http.MethodPut && strings.HasPrefix(request.URL.Path, "/v2/element-instances/"):
			scope := strings.TrimSuffix(strings.TrimPrefix(request.URL.Path, "/v2/element-instances/"), "/variables")
			capture.mu.Lock()
			capture.mutations++
			if scope != capture.failingScope {
				capture.mutatedScopes[scope] = true
			}
			capture.mu.Unlock()
			if scope == failingScope {
				http.Error(writer, `{"message":"mutation denied"}`, http.StatusInternalServerError)
				return
			}
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, capture
}

// updateUserTaskOutputScope maps selected task fixtures to distinct inherited scopes.
func updateUserTaskOutputScope(key string) string {
	if key == "102" {
		return "202"
	}
	return "201"
}
