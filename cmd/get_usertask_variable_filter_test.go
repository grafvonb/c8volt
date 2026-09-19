// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

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
