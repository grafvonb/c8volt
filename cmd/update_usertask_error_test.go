// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// updateUserTaskErrorCapture records request phases for error-path assertions.
type updateUserTaskErrorCapture struct {
	mu            sync.Mutex
	requests      int
	mutations     int
	variableReads int
}

// TestUpdateUserTaskSupportedVersions verifies every supported adapter exposes
// the same accepted no-wait command contract and local scope request.
func TestUpdateUserTaskSupportedVersions(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		t.Run(version, func(t *testing.T) {
			server, capture := newUpdateUserTaskCommandServer(t)
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, version), "",
				"--automation", "--json", "update", "ut", "--key", "2251799815391233",
				"--vars", `{"approved":true}`, "--no-wait",
			)
			require.NoError(t, err)
			require.Empty(t, stderr)
			envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
			require.Equal(t, "accepted", envelope["outcome"])
			items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)
			require.Equal(t, "submitted", requireJSONObject(t, items[0])["status"])

			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Equal(t, 1, capture.mutations)
			require.Equal(t, true, capture.mutationBody[0]["local"])
			require.Len(t, capture.variableReads, 1, "no-wait performs planning reads only")
		})
	}
}

// TestUpdateUserTaskUnsupportedVersion performs no HTTP work on Camunda 8.7
// and returns the normalized unsupported error envelope and exit code.
func TestUpdateUserTaskUnsupportedVersion(t *testing.T) {
	server, capture := newUpdateUserTaskErrorServer(t, "success")
	stdout, _, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.7"), "",
		"--automation", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`,
	)
	requireCommandExitCode(t, err, exitcode.Error)
	envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
	require.Equal(t, "unsupported", envelope["class"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Zero(t, capture.requests)
	require.Zero(t, capture.mutations)
}

// TestUpdateUserTaskDiscoveryErrors verifies authorization, absence, and
// malformed effective values retain their error classes without mutation.
func TestUpdateUserTaskDiscoveryErrors(t *testing.T) {
	tests := []struct {
		name      string
		scenario  string
		wantClass string
		wantExit  int
	}{
		{name: "unauthorized task", scenario: "unauthorized-task", wantClass: "local_precondition", wantExit: exitcode.Error},
		{name: "missing task", scenario: "missing-task", wantClass: "not_found", wantExit: exitcode.NotFound},
		{name: "malformed discovery", scenario: "malformed-discovery", wantClass: "malformed_response", wantExit: exitcode.Error},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, capture := newUpdateUserTaskErrorServer(t, test.scenario)
			stdout, _, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "",
				"--automation", "--json", "update", "ut", "--key", "101", "--vars", `{"approved":true}`,
			)
			requireCommandExitCode(t, err, test.wantExit)
			envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
			require.Equal(t, "failed", envelope["outcome"])
			require.Equal(t, test.wantClass, envelope["class"])

			capture.mu.Lock()
			defer capture.mu.Unlock()
			require.Zero(t, capture.mutations)
		})
	}
}

// TestUpdateUserTaskTruncatedConfirmation preserves accepted mutation facts
// while rejecting an incomplete post-write observation as confirmation.
func TestUpdateUserTaskTruncatedConfirmation(t *testing.T) {
	server, capture := newUpdateUserTaskErrorServer(t, "truncated-confirmation")
	stdout, _, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "",
		"--backoff-max-retries", "1", "--automation", "--json", "update", "ut",
		"--key", "101", "--vars", `{"approved":true}`,
	)
	requireCommandExitCode(t, err, exitcode.Error)
	envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
	require.Equal(t, "failed", envelope["outcome"])
	items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "confirmation_failed", item["status"])
	require.Equal(t, true, item["mutationAccepted"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 1, capture.mutations)
	require.Equal(t, 2, capture.variableReads)
}

// TestUpdateUserTaskPartialAcceptanceExitCode proves a mixed shared command
// result is rendered once and still exits unsuccessfully.
func TestUpdateUserTaskPartialAcceptanceExitCode(t *testing.T) {
	server, capture := newUpdateUserTaskOutputServer(t, false, "202")
	stdout, _, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "",
		"--automation", "--json", "update", "ut", "--key", "101,102",
		"--vars", `{"approved":true}`, "--no-wait", "--workers", "1",
	)
	requireCommandExitCode(t, err, exitcode.Error)
	envelope := decodeSingleUserTaskUpdateEnvelope(t, stdout)
	items := requireJSONItems(t, requireJSONObject(t, envelope["payload"])["items"], 2)
	require.Equal(t, "submitted", requireJSONObject(t, items[0])["status"])
	require.Equal(t, "mutation_failed", requireJSONObject(t, items[1])["status"])

	capture.mu.Lock()
	defer capture.mu.Unlock()
	require.Equal(t, 2, capture.mutations)
	require.Equal(t, 2, capture.variableReads)
}

// TestUpdateUserTaskFunctionalAndHTTPDiagnostics separates verbose scope facts
// from DEBUG-only HTTP exchange diagnostics.
func TestUpdateUserTaskFunctionalAndHTTPDiagnostics(t *testing.T) {
	tests := []struct {
		name       string
		rootFlag   string
		wantScopes bool
		wantHTTP   bool
	}{
		{name: "default compact"},
		{name: "verbose functional detail", rootFlag: "--verbose", wantScopes: true},
		{name: "debug http diagnostics", rootFlag: "--debug", wantHTTP: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, _ := newUpdateUserTaskOutputServer(t, false, "")
			args := []string{"--auto-confirm"}
			if test.rootFlag != "" {
				args = append(args, test.rootFlag)
			}
			args = append(args, "update", "ut", "--key", "101", "--vars", `{"approved":true}`, "--no-wait")
			stdout, stderr, err := runUpdateUserTaskCommand(t, writeTestConfigForVersion(t, server.URL, "8.10"), "", args...)
			require.NoError(t, err)
			require.Empty(t, stdout)
			require.Equal(t, test.wantScopes, strings.Contains(stderr, "; scopes: 201=submitted"))
			require.Equal(t, test.wantHTTP, strings.Contains(stderr, "DEBUG api #"))
		})
	}
}

// requireCommandExitCode checks the subprocess contract without discarding the
// concrete exit status behind the returned error.
func requireCommandExitCode(t *testing.T, err error, want int) {
	t.Helper()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	require.Equal(t, want, exitErr.ExitCode())
}

// newUpdateUserTaskErrorServer exposes controlled read and confirmation
// failures while recording that rejected paths never mutate state.
func newUpdateUserTaskErrorServer(t *testing.T, scenario string) (*httptest.Server, *updateUserTaskErrorCapture) {
	t.Helper()
	capture := new(updateUserTaskErrorCapture)
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		capture.mu.Lock()
		capture.requests++
		capture.mu.Unlock()
		switch {
		case request.Method == http.MethodGet && strings.HasPrefix(request.URL.Path, "/v2/user-tasks/"):
			switch scenario {
			case "unauthorized-task":
				http.Error(writer, `{"message":"denied"}`, http.StatusUnauthorized)
				return
			case "missing-task":
				http.Error(writer, `{"message":"missing"}`, http.StatusNotFound)
				return
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(writer, `{"userTaskKey":"101","state":"CREATED","elementId":"approve","processInstanceKey":"301","elementInstanceKey":"501","processDefinitionKey":"401","tenantId":"tenant-a"}`)
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/effective-variables/search"):
			capture.mu.Lock()
			capture.variableReads++
			mutated := capture.mutations > 0
			capture.mu.Unlock()
			value := "false"
			truncated := false
			if scenario == "malformed-discovery" {
				value = "not-json"
			} else if mutated {
				value = "true"
				truncated = scenario == "truncated-confirmation"
			}
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(writer, `{"items":[{"name":"approved","value":%q,"variableKey":"601","processInstanceKey":"301","scopeKey":"201","tenantId":"tenant-a","isTruncated":%t}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, value, truncated)
		case request.Method == http.MethodPut && request.URL.Path == "/v2/element-instances/201/variables":
			capture.mu.Lock()
			capture.mutations++
			capture.mu.Unlock()
			writer.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)
	return server, capture
}
