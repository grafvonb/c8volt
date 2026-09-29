// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"fmt"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

// TestGetUserTaskCommand_VariableFiltersBuildNativeRequests verifies every
// command alias carries task-owned variable flags into the native local scope
// alongside ordinary selectors on each supported adapter.
func TestGetUserTaskCommand_VariableFiltersBuildNativeRequests(t *testing.T) {
	for _, version := range []string{"8.8", "8.9", "8.10"} {
		for _, alias := range []string{"user-task", "user-tasks", "ut", "uts"} {
			t.Run(fmt.Sprintf("%s/%s", version, alias), func(t *testing.T) {
				server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
					return userTaskSearchResponse(1, false, "", "2251799815391233")
				})
				configPath := testx.WriteTestConfigForVersion(t, server.URL, version)

				stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "tenant-a", "--keys-only", "get", alias,
					"--assignee", "alice",
					"--var-exists", "payload,email",
					"--var", `status="approved"`,
					"--var", `active.$exists=false,kind.$in=["a","b"]`,
					"--var-like", `address=*@example.com`)
				require.NoError(t, err, stderr)
				require.Empty(t, stderr)
				require.Equal(t, "2251799815391233\n", stdout)

				got := requests.Snapshot()
				require.Len(t, got, 1, "filtering must issue only the native task search request")
				filter := requireJSONMap(t, got[0]["filter"])
				require.Equal(t, "alice", jsonFilterValue(t, filter["assignee"]))
				require.Equal(t, "tenant-a", jsonFilterValue(t, filter["tenantId"]))
				localVariables, ok := filter["localVariables"].([]any)
				require.True(t, ok, "expected native local-variable filters")
				require.Equal(t, []any{
					map[string]any{"name": "payload", "value": map[string]any{"$exists": true}},
					map[string]any{"name": "email", "value": map[string]any{"$exists": true}},
					map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}},
					map[string]any{"name": "active", "value": map[string]any{"$exists": false}},
					map[string]any{"name": "kind", "value": map[string]any{"$in": []any{"a", "b"}}},
					map[string]any{"name": "address", "value": map[string]any{"$like": `*@example.com`}},
				}, localVariables)
			})
		}
	}
}

// TestGetUserTaskCommand_VariableFiltersRejectMalformedInputBeforeRequests
// protects parser diagnostics and prevents an unfiltered fallback.
func TestGetUserTaskCommand_VariableFiltersRejectMalformedInputBeforeRequests(t *testing.T) {
	server, requests := newGetUserTaskSearchServer(t, func(_ int, _ map[string]any) string {
		return userTaskSearchResponse(0, false, "")
	})
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "malformed", args: []string{"get", "ut", "--var", "status"}, want: "must use name=value syntax"},
		{name: "unknown operator", args: []string{"get", "ut", "--var", "status.$contains=approved"}, want: "unsupported variable operator"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			before := len(requests.Snapshot())
			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.Error(t, err)
			require.Empty(t, stdout)
			require.Contains(t, stderr, test.want)
			require.Len(t, requests.Snapshot(), before)
		})
	}
}

// TestUserTaskVariableFilterParserParity verifies both command wrappers retain
// the same operators, aliases, delimiters, serialized values, and group order.
func TestUserTaskVariableFilterParserParity(t *testing.T) {
	exists := []string{"payload,email", "customerId"}
	values := []string{
		`exact.$eq="approved",status.$neq="failed",active.$exists=false`,
		`kind.$in=["a","b,c"],segment.$notIn=["d","e"],alias.$notin=["x","y"]`,
		`assignment="left=right",quoted="hello,world"`,
	}
	likes := []string{`email=*@example.com,literal=invoice-\*,single=order-\?`}

	resetVariableFilterParserGlobals()
	t.Cleanup(resetVariableFilterParserGlobals)
	flagGetPIVarExists, flagGetPIVars, flagGetPIVarLikes = exists, values, likes
	piFilters, err := parsePIVariableFilters()
	require.NoError(t, err)

	flagGetUserTaskVarExists, flagGetUserTaskVars, flagGetUserTaskVarLikes = exists, values, likes
	taskFilters, err := parseUserTaskVariableFilters()
	require.NoError(t, err)
	require.Equal(t, piFilters, process.ProcessInstanceVariableFilterSet(taskFilters))
	require.Equal(t, []process.ProcessInstanceVariableFilterOperator{
		process.ProcessInstanceVariableFilterOperatorExists,
		process.ProcessInstanceVariableFilterOperatorExists,
		process.ProcessInstanceVariableFilterOperatorExists,
		process.ProcessInstanceVariableFilterOperatorEq,
		process.ProcessInstanceVariableFilterOperatorNeq,
		process.ProcessInstanceVariableFilterOperatorExists,
		process.ProcessInstanceVariableFilterOperatorIn,
		process.ProcessInstanceVariableFilterOperatorNotIn,
		process.ProcessInstanceVariableFilterOperatorNotIn,
		process.ProcessInstanceVariableFilterOperatorEq,
		process.ProcessInstanceVariableFilterOperatorEq,
		process.ProcessInstanceVariableFilterOperatorLike,
		process.ProcessInstanceVariableFilterOperatorLike,
		process.ProcessInstanceVariableFilterOperatorLike,
	}, variableFilterOperators(taskFilters.Clauses))
	require.False(t, *taskFilters.Clauses[5].Exists)
	require.Equal(t, `"left=right"`, taskFilters.Clauses[9].Value)
	require.Equal(t, `"hello,world"`, taskFilters.Clauses[10].Value)
}

// TestUserTaskVariableFilterParserErrorParity protects established diagnostics
// for malformed clauses and unsupported serialized membership values.
func TestUserTaskVariableFilterParserErrorParity(t *testing.T) {
	tests := []struct {
		name    string
		exists  []string
		values  []string
		likes   []string
		wantErr string
	}{
		{name: "blank exists", exists: []string{"payload,,email"}, wantErr: "variable name must not be blank"},
		{name: "missing assignment", values: []string{"status"}, wantErr: "must use name=value syntax"},
		{name: "unknown operator", values: []string{"status.$contains=approved"}, wantErr: "unsupported variable operator"},
		{name: "bad exists", values: []string{"active.$exists=yes"}, wantErr: "$exists requires true or false"},
		{name: "non array", values: []string{"kind.$in=approved"}, wantErr: "$in requires an array value"},
		{name: "non string array", values: []string{`kind.$notIn=["a",2]`}, wantErr: "$notIn requires an array value"},
		{name: "unterminated quote", likes: []string{`email="*@example.com`}, wantErr: "unterminated quoted value"},
		{name: "unterminated array", values: []string{`kind.$in=["a"`}, wantErr: "unterminated array value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetVariableFilterParserGlobals()
			t.Cleanup(resetVariableFilterParserGlobals)
			flagGetPIVarExists, flagGetPIVars, flagGetPIVarLikes = tt.exists, tt.values, tt.likes
			_, piErr := parsePIVariableFilters()
			flagGetUserTaskVarExists, flagGetUserTaskVars, flagGetUserTaskVarLikes = tt.exists, tt.values, tt.likes
			_, taskErr := parseUserTaskVariableFilters()

			require.Error(t, piErr)
			require.Error(t, taskErr)
			require.Equal(t, piErr.Error(), taskErr.Error())
			require.Contains(t, taskErr.Error(), tt.wantErr)
		})
	}
}

// TestUserTaskVariableFilterParserGlobalsAreIsolated proves one command's raw
// flag state cannot alter the other command's parsed clauses.
func TestUserTaskVariableFilterParserGlobalsAreIsolated(t *testing.T) {
	resetVariableFilterParserGlobals()
	t.Cleanup(resetVariableFilterParserGlobals)
	flagGetPIVars = []string{`pi="process"`}
	flagGetUserTaskVars = []string{`task="local"`}

	piFilters, err := parsePIVariableFilters()
	require.NoError(t, err)
	taskFilters, err := parseUserTaskVariableFilters()
	require.NoError(t, err)

	require.Equal(t, "pi", piFilters.Clauses[0].Name)
	require.Equal(t, "task", taskFilters.Clauses[0].Name)
	require.Equal(t, `"process"`, piFilters.Clauses[0].Value)
	require.Equal(t, `"local"`, taskFilters.Clauses[0].Value)
}

// variableFilterOperators extracts clause operators for a compact order assertion.
func variableFilterOperators(clauses []process.ProcessInstanceVariableFilterClause) []process.ProcessInstanceVariableFilterOperator {
	operators := make([]process.ProcessInstanceVariableFilterOperator, len(clauses))
	for i, clause := range clauses {
		operators[i] = clause.Operator
	}
	return operators
}

// resetVariableFilterParserGlobals isolates package-level command flag state.
func resetVariableFilterParserGlobals() {
	flagGetPIVarExists, flagGetPIVars, flagGetPIVarLikes = nil, nil, nil
	flagGetUserTaskVarExists, flagGetUserTaskVars, flagGetUserTaskVarLikes = nil, nil, nil
}
