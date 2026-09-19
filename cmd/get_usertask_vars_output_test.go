// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// userTaskVariableOutputKey is the stable task identity used by output fixtures.
const userTaskVariableOutputKey = "2251799815391233"

// TestGetUserTaskVariableValueLimits verifies unlimited and positive human
// limits after structured-value compaction while preserving Unicode runes.
func TestGetUserTaskVariableValueLimits(t *testing.T) {
	for _, test := range []struct {
		name      string
		limitArgs []string
		want      []string
	}{
		{
			name: "default unlimited",
			want: []string{
				`array=[1,2,3,4]`, `empty=`, `null=null`,
				`object={"amount":120}`, `unicode=äöüabc`,
			},
		},
		{
			name:      "explicit zero unlimited",
			limitArgs: []string{"--var-value-limit", "0"},
			want:      []string{`object={"amount":120}`, `unicode=äöüabc`},
		},
		{
			name:      "positive rune limit",
			limitArgs: []string{"--var-value-limit", "3"},
			want: []string{
				`array=[1,... [cli-truncated]`, `empty=`, `null=nul... [cli-truncated]`,
				`object={"a... [api-truncated,cli-truncated]`, `unicode=äöü... [cli-truncated]`,
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newUserTaskVariableOutputServer(t, http.StatusOK)
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			args := []string{"get", "ut", "--key", userTaskVariableOutputKey, "--with-vars"}
			args = append(args, test.limitArgs...)

			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			for _, want := range test.want {
				require.Contains(t, stdout, want)
			}
			require.Equal(t, 1, requests.variableRequestCount(userTaskVariableOutputKey))
		})
	}
}

// TestGetUserTaskVariableOutputModes verifies JSON precedence, quiet behavior,
// unchanged ordinary output, and zero-request keys-only and count execution.
func TestGetUserTaskVariableOutputModes(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantVariables int
		assertOutput  func(*testing.T, string)
	}{
		{
			name:          "json wins over keys",
			args:          []string{"--json", "--keys-only", "get", "ut", "--key", userTaskVariableOutputKey, "--with-vars", "--var-value-limit", "3"},
			wantVariables: 1,
			assertOutput: func(t *testing.T, output string) {
				decoder := json.NewDecoder(strings.NewReader(output))
				var envelope map[string]any
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
				variables := envelope["payload"].(map[string]any)["items"].([]any)[0].(map[string]any)["variables"].([]any)
				require.Equal(t, "äöüabc", variables[4].(map[string]any)["value"], "human limit must not shorten JSON")
			},
		},
		{
			name:          "quiet json",
			args:          []string{"--quiet", "--json", "get", "ut", "--key", userTaskVariableOutputKey, "--with-vars"},
			wantVariables: 1,
			assertOutput:  func(t *testing.T, output string) { require.Contains(t, output, `"variables"`) },
		},
		{
			name:         "quiet keys",
			args:         []string{"--quiet", "--keys-only", "get", "ut", "--key", userTaskVariableOutputKey, "--with-vars"},
			assertOutput: func(t *testing.T, output string) { require.Equal(t, userTaskVariableOutputKey+"\n", output) },
		},
		{
			name: "ordinary output unchanged",
			args: []string{"get", "ut", "--key", userTaskVariableOutputKey},
			assertOutput: func(t *testing.T, output string) {
				require.NotContains(t, output, "vars:")
				require.Contains(t, output, "found: 1\n")
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newUserTaskVariableOutputServer(t, http.StatusOK)
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			test.assertOutput(t, stdout)
			require.Equal(t, test.wantVariables, requests.variableRequestCount(userTaskVariableOutputKey))
		})
	}

	server, requests := newUserTaskVariableOutputServer(t, http.StatusOK)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut", "--total", "--with-vars")
	require.NoError(t, err, stderr)
	require.Equal(t, "1\n", stdout)
	require.Empty(t, stderr)
	require.Zero(t, requests.variableRequestCount(userTaskVariableOutputKey))
}

// TestGetUserTaskVariableFilteredOutputModes verifies local filtering retains
// every display exclusion and machine-output precedence rule for --with-vars.
func TestGetUserTaskVariableFilteredOutputModes(t *testing.T) {
	const filter = `status="approved"`
	tests := []struct {
		name          string
		args          []string
		emptySearch   bool
		want          string
		wantJSON      bool
		wantVariables int
	}{
		{
			name: "effective keys only",
			args: []string{"--keys-only", "get", "ut", "--var", filter, "--with-vars"},
			want: userTaskVariableOutputKey + "\n",
		},
		{
			name: "total",
			args: []string{"get", "ut", "--var", filter, "--total", "--with-vars"},
			want: "1\n",
		},
		{
			name:        "empty",
			args:        []string{"get", "ut", "--var", filter, "--with-vars"},
			emptySearch: true,
			want:        "found: 0\n",
		},
		{
			name:          "quiet human",
			args:          []string{"--quiet", "get", "ut", "--var", filter, "--with-vars"},
			wantVariables: 1,
		},
		{
			name:          "quiet json",
			args:          []string{"--quiet", "--json", "get", "ut", "--var", filter, "--with-vars", "--var-value-limit", "3"},
			wantJSON:      true,
			wantVariables: 1,
		},
		{
			name:          "json over keys",
			args:          []string{"--keys-only", "--json", "get", "ut", "--var", filter, "--with-vars", "--var-value-limit", "3"},
			wantJSON:      true,
			wantVariables: 1,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := userTaskVariableOutputFixture(http.StatusOK)
			if test.emptySearch {
				fixture.SearchRespond = func(_ int, _ map[string]any) string {
					return userTaskSearchResponse(0, false, "")
				}
			}
			server, requests := newGetUserTaskVariablesServer(t, fixture)
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			if test.wantJSON {
				decoder := json.NewDecoder(strings.NewReader(stdout))
				var envelope map[string]any
				require.NoError(t, decoder.Decode(&envelope))
				require.ErrorIs(t, decoder.Decode(&struct{}{}), io.EOF)
				items := envelope["payload"].(map[string]any)["items"].([]any)
				require.Len(t, items, 1)
				variables := items[0].(map[string]any)["variables"].([]any)
				require.Equal(t, `{ "amount": 120 }`, variables[3].(map[string]any)["value"], "JSON preserves structured values")
				require.Equal(t, true, variables[3].(map[string]any)["apiTruncated"])
				require.Equal(t, "äöüabc", variables[4].(map[string]any)["value"], "human limits do not shorten JSON")
			} else {
				require.Equal(t, test.want, stdout)
			}
			require.Equal(t, 1, requests.searchRequestCount())
			require.Equal(t, [][]any{{
				map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}},
			}}, requests.localVariablePredicates())
			require.Equal(t, test.wantVariables, requests.variableRequestCount(userTaskVariableOutputKey))
		})
	}
}

// TestGetUserTaskVariableFilteredHumanValueLimit verifies filtered human
// display keeps structured compaction, Unicode rune limits, and truncation labels.
func TestGetUserTaskVariableFilteredHumanValueLimit(t *testing.T) {
	server, requests := newUserTaskVariableOutputServer(t, http.StatusOK)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut", "--var", `status="approved"`, "--with-vars", "--var-value-limit", "3")
	require.NoError(t, err, stderr)
	require.Empty(t, stderr)
	for _, want := range []string{
		`array=[1,... [cli-truncated]`,
		`object={"a... [api-truncated,cli-truncated]`,
		`unicode=äöü... [cli-truncated]`,
	} {
		require.Contains(t, stdout, want)
	}
	require.Contains(t, stdout, "found: 1\n")
	require.Equal(t, 1, requests.variableRequestCount(userTaskVariableOutputKey))
	require.Equal(t, 1, requests.searchRequestCount())
}

// TestGetUserTaskVariableFilteredEnrichmentFailure verifies an effective-value
// retrieval error remains a command failure after a successful native filter.
func TestGetUserTaskVariableFilteredEnrichmentFailure(t *testing.T) {
	server, requests := newUserTaskVariableOutputServer(t, http.StatusBadGateway)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "get", "ut", "--var", `status="approved"`, "--with-vars")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "get user task variables")
	require.Equal(t, 1, requests.searchRequestCount())
	require.Equal(t, 1, requests.variableRequestCount(userTaskVariableOutputKey))
	require.Equal(t, [][]any{{
		map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}},
	}}, requests.localVariablePredicates())
}

// TestGetUserTaskVariableLimitValidationAndQuietFailure verifies local limit
// errors happen before reads and quiet human retrieval still propagates errors.
func TestGetUserTaskVariableLimitValidationAndQuietFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		args []string
		want string
	}{
		{name: "negative", args: []string{"get", "ut", "--key", userTaskVariableOutputKey, "--with-vars", "--var-value-limit", "-1"}, want: "invalid value for --var-value-limit"},
		{name: "dangling zero", args: []string{"get", "ut", "--key", userTaskVariableOutputKey, "--var-value-limit", "0"}, want: "--var-value-limit requires --with-vars"},
		{name: "dangling positive", args: []string{"get", "ut", "--key", userTaskVariableOutputKey, "--var-value-limit", "3"}, want: "--var-value-limit requires --with-vars"},
		{name: "count json conflict", args: []string{"--json", "get", "ut", "--total", "--with-vars"}, want: "--total cannot be combined with --json"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newUserTaskVariableOutputServer(t, http.StatusOK)
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, test.want)
			taskRequests, searchRequests, variableRequests := requests.snapshot()
			require.Empty(t, taskRequests)
			require.Empty(t, searchRequests)
			require.Empty(t, variableRequests)
		})
	}

	server, requests := newUserTaskVariableOutputServer(t, http.StatusBadGateway)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--quiet", "get", "ut", "--key", userTaskVariableOutputKey, "--with-vars")
	require.Error(t, err)
	require.Empty(t, stdout)
	require.Contains(t, stderr, "get user task variables")
	require.Equal(t, 1, requests.variableRequestCount(userTaskVariableOutputKey))
}

// newUserTaskVariableOutputServer returns a fresh single-task fixture so every
// output-mode case receives an independent effective-variable page sequence.
func newUserTaskVariableOutputServer(t *testing.T, variableStatus int) (*httptest.Server, *capturedGetUserTaskVariableRequests) {
	t.Helper()
	return newGetUserTaskVariablesServer(t, userTaskVariableOutputFixture(variableStatus))
}

// userTaskVariableOutputFixture supplies structured, Unicode, and truncated
// effective values for both keyed and filtered output-contract tests.
func userTaskVariableOutputFixture(variableStatus int) getUserTaskVariablesFixture {
	apiTruncated := true
	return getUserTaskVariablesFixture{
		SearchRespond: func(_ int, _ map[string]any) string {
			return userTaskSearchResponse(1, false, "", userTaskVariableOutputKey)
		},
		VariablePages: map[string][]userTaskVariablePageFixture{
			userTaskVariableOutputKey: {{
				Status: variableStatus,
				Total:  5,
				Items: []userTaskVariableFixtureValue{
					{Name: "array", Value: "[ 1, 2, 3, 4 ]", VariableKey: "901", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-a"},
					{Name: "empty", Value: "", VariableKey: "902", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-a"},
					{Name: "null", Value: "null", VariableKey: "903", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-a"},
					{Name: "object", Value: `{ "amount": 120 }`, VariableKey: "904", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-a", IsTruncated: &apiTruncated},
					{Name: "unicode", Value: "äöüabc", VariableKey: "905", ProcessInstanceKey: "2251799813711967", ScopeKey: "2251799815391200", TenantID: "tenant-a"},
				},
			}},
		},
	}
}
