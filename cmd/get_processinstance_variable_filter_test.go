// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

// TestParsePIVariableFilters_PreservesQuotedCommasAndArrays verifies the
// parser only splits top-level commas before building variable clauses.
func TestParsePIVariableFilters_PreservesQuotedCommasAndArrays(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVarExists = []string{"customerId,payload"}
	flagGetPIVars = []string{`status="approved,ready",kind.$in=["alpha","beta,gamma"]`}
	flagGetPIVarLikes = []string{`email=*@example.com`}

	got, err := parsePIVariableFilters()

	require.NoError(t, err)
	require.Equal(t, process.ProcessInstanceVariableFilterSet{
		Clauses: []process.ProcessInstanceVariableFilterClause{
			{Name: "customerId", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: boolValuePtr(true), Source: piVariableFilterSourceExists},
			{Name: "payload", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: boolValuePtr(true), Source: piVariableFilterSourceExists},
			{Name: "status", Operator: process.ProcessInstanceVariableFilterOperatorEq, Value: `"approved,ready"`, Source: piVariableFilterSourceVar},
			{Name: "kind", Operator: process.ProcessInstanceVariableFilterOperatorIn, Value: `["alpha","beta,gamma"]`, Source: piVariableFilterSourceVar},
			{Name: "email", Operator: process.ProcessInstanceVariableFilterOperatorLike, Value: `*@example.com`, Source: piVariableFilterSourceLike},
		},
	}, got)
}

// TestParsePIVariableFilters_ExpandsRepeatedExistsInputs verifies the
// existence shorthand combines repeated flags and comma-separated names.
func TestParsePIVariableFilters_ExpandsRepeatedExistsInputs(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVarExists = []string{"customerId", "payload,email"}

	got, err := parsePIVariableFilters()

	require.NoError(t, err)
	require.Equal(t, []process.ProcessInstanceVariableFilterClause{
		{Name: "customerId", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: boolValuePtr(true), Source: piVariableFilterSourceExists},
		{Name: "payload", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: boolValuePtr(true), Source: piVariableFilterSourceExists},
		{Name: "email", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: boolValuePtr(true), Source: piVariableFilterSourceExists},
	}, got.Clauses)
}

// TestParsePIVariableFilters_ExpandsRepeatedEqualityInputs verifies equality
// shorthand composes repeated --var flags with comma-separated clauses.
func TestParsePIVariableFilters_ExpandsRepeatedEqualityInputs(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVars = []string{`status="approved"`, `payload="payload",customerId="CUST-001"`}

	got, err := parsePIVariableFilters()

	require.NoError(t, err)
	require.Equal(t, []process.ProcessInstanceVariableFilterClause{
		{Name: "status", Operator: process.ProcessInstanceVariableFilterOperatorEq, Value: `"approved"`, Source: piVariableFilterSourceVar},
		{Name: "payload", Operator: process.ProcessInstanceVariableFilterOperatorEq, Value: `"payload"`, Source: piVariableFilterSourceVar},
		{Name: "customerId", Operator: process.ProcessInstanceVariableFilterOperatorEq, Value: `"CUST-001"`, Source: piVariableFilterSourceVar},
	}, got.Clauses)
}

// TestParsePIVariableFilters_PreservesEqualityQuotedCommaValues protects
// equality values whose serialized text contains commas from accidental splits.
func TestParsePIVariableFilters_PreservesEqualityQuotedCommaValues(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVars = []string{`status="approved",payload="payload,with,commas"`}

	got, err := parsePIVariableFilters()

	require.NoError(t, err)
	require.Equal(t, []process.ProcessInstanceVariableFilterClause{
		{Name: "status", Operator: process.ProcessInstanceVariableFilterOperatorEq, Value: `"approved"`, Source: piVariableFilterSourceVar},
		{Name: "payload", Operator: process.ProcessInstanceVariableFilterOperatorEq, Value: `"payload,with,commas"`, Source: piVariableFilterSourceVar},
	}, got.Clauses)
}

// TestParsePIVariableFilters_RejectsBlankExistsName keeps doubled delimiters
// from becoming ambiguous native existence clauses.
func TestParsePIVariableFilters_RejectsBlankExistsName(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVarExists = []string{"payload,,email"}

	_, err := parsePIVariableFilters()

	require.Error(t, err)
	require.Contains(t, err.Error(), "variable name must not be blank")
}

// TestParsePIVariableFilters_NormalizesAdvancedOperators verifies native
// operator spelling is preserved while the accepted notin alias is canonicalized.
func TestParsePIVariableFilters_NormalizesAdvancedOperators(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVars = []string{`status.$neq="failed",active.$exists=false,kind.$notin=["a","b"],email.$like=*@example.com`}

	got, err := parsePIVariableFilters()

	require.NoError(t, err)
	require.Equal(t, []process.ProcessInstanceVariableFilterClause{
		{Name: "status", Operator: process.ProcessInstanceVariableFilterOperatorNeq, Value: `"failed"`, Source: piVariableFilterSourceVar},
		{Name: "active", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: boolValuePtr(false), Source: piVariableFilterSourceVar},
		{Name: "kind", Operator: process.ProcessInstanceVariableFilterOperatorNotIn, Value: `["a","b"]`, Source: piVariableFilterSourceVar},
		{Name: "email", Operator: process.ProcessInstanceVariableFilterOperatorLike, Value: `*@example.com`, Source: piVariableFilterSourceVar},
	}, got.Clauses)
}

// TestParsePIVariableFilters_RejectsMalformedClauses protects local validation
// so malformed variable syntax fails before any remote search can start.
func TestParsePIVariableFilters_RejectsMalformedClauses(t *testing.T) {
	tests := []struct {
		name    string
		vars    []string
		wantErr string
	}{
		{name: "blank name", vars: []string{"=value"}, wantErr: "variable name must not be blank"},
		{name: "unknown operator", vars: []string{"status.$contains=value"}, wantErr: "unsupported variable operator"},
		{name: "missing value", vars: []string{"status="}, wantErr: "$eq requires a non-empty value"},
		{name: "malformed bool", vars: []string{"active.$exists=yes"}, wantErr: "$exists requires true or false"},
		{name: "malformed array", vars: []string{"status.$in=approved"}, wantErr: "$in requires an array value"},
		{name: "unterminated quote", vars: []string{`status="approved`}, wantErr: "unterminated quoted value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProcessInstanceCommandGlobals()
			t.Cleanup(resetProcessInstanceCommandGlobals)

			flagGetPIVars = tt.vars

			_, err := parsePIVariableFilters()

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestPopulatePISearchFilterOpts_AttachesVariableFilters verifies parsed
// variable clauses participate in the command-to-facade search filter handoff.
func TestPopulatePISearchFilterOpts_AttachesVariableFilters(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVarExists = []string{"customerId"}
	flagGetPIVars = []string{`status="approved"`}

	filter := populatePISearchFilterOpts()

	require.Equal(t, process.ProcessInstanceVariableFilterSet{
		Clauses: []process.ProcessInstanceVariableFilterClause{
			{Name: "customerId", Operator: process.ProcessInstanceVariableFilterOperatorExists, Exists: boolValuePtr(true), Source: piVariableFilterSourceExists},
			{Name: "status", Operator: process.ProcessInstanceVariableFilterOperatorEq, Value: `"approved"`, Source: piVariableFilterSourceVar},
		},
	}, filter.VariableFilters)
	require.True(t, hasPISearchFilterFlags())
}
