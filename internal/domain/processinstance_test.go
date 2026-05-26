// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package domain

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestProcessInstanceFilterString_RendersOptionalBooleansConsistently verifies nil booleans are omitted from debug output.
func TestProcessInstanceFilterString_RendersOptionalBooleansConsistently(t *testing.T) {
	require.Equal(t, "none", fmt.Sprintf("%+v", ProcessInstanceFilter{}))
	require.NotContains(t, fmt.Sprintf("%+v", ProcessInstanceFilter{}), "<nil>")

	hasParent := true
	hasIncident := false
	got := fmt.Sprintf("%+v", ProcessInstanceFilter{
		BpmnProcessId: "order",
		HasParent:     &hasParent,
		HasIncident:   &hasIncident,
	})

	require.Equal(t, `{bpmnProcessId="order", hasParent=true, hasIncident=false}`, got)
}

// TestProcessInstanceFilterString_RendersVariableFilters verifies debug output
// includes normalized variable clauses without rendering an empty filter set.
func TestProcessInstanceFilterString_RendersVariableFilters(t *testing.T) {
	require.NotContains(t, fmt.Sprintf("%+v", ProcessInstanceFilter{}), "variableFilters")

	exists := true
	got := fmt.Sprintf("%+v", ProcessInstanceFilter{
		BpmnProcessId: "order",
		VariableFilters: ProcessInstanceVariableFilterSet{
			Clauses: []ProcessInstanceVariableFilterClause{
				{Name: "customerId", Operator: ProcessInstanceVariableFilterOperatorExists, Exists: &exists},
				{Name: "status", Operator: ProcessInstanceVariableFilterOperatorEq, Value: `"approved"`},
			},
		},
	})

	require.Equal(t, `{bpmnProcessId="order", variableFilters=[customerId.$exists=true, status.$eq="\"approved\""]}`, got)
}

// TestProcessInstanceVariableFilterSetValidate_RejectsInvalidShapes protects
// the service-facing filter contract from ambiguous or unsupported clauses.
func TestProcessInstanceVariableFilterSetValidate_RejectsInvalidShapes(t *testing.T) {
	tests := []struct {
		name    string
		clause  ProcessInstanceVariableFilterClause
		wantErr string
	}{
		{name: "blank name", clause: ProcessInstanceVariableFilterClause{Operator: ProcessInstanceVariableFilterOperatorEq, Value: "x"}, wantErr: "name must not be blank"},
		{name: "unknown operator", clause: ProcessInstanceVariableFilterClause{Name: "status", Operator: "$contains", Value: "x"}, wantErr: "unsupported variable filter operator"},
		{name: "missing exists", clause: ProcessInstanceVariableFilterClause{Name: "active", Operator: ProcessInstanceVariableFilterOperatorExists}, wantErr: "requires an exists value"},
		{name: "missing equality value", clause: ProcessInstanceVariableFilterClause{Name: "status", Operator: ProcessInstanceVariableFilterOperatorEq}, wantErr: "requires a value"},
		{name: "malformed array", clause: ProcessInstanceVariableFilterClause{Name: "status", Operator: ProcessInstanceVariableFilterOperatorIn, Value: "approved"}, wantErr: "requires an array value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (ProcessInstanceVariableFilterSet{Clauses: []ProcessInstanceVariableFilterClause{tt.clause}}).Validate()

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// TestProcessInstanceVariableFilterSetValidate_AllowsSupportedShapes verifies
// every native variable-search operator has a valid version-neutral shape.
func TestProcessInstanceVariableFilterSetValidate_AllowsSupportedShapes(t *testing.T) {
	exists := true
	err := (ProcessInstanceVariableFilterSet{Clauses: []ProcessInstanceVariableFilterClause{
		{Name: "a", Operator: ProcessInstanceVariableFilterOperatorEq, Value: "1"},
		{Name: "b", Operator: ProcessInstanceVariableFilterOperatorNeq, Value: "2"},
		{Name: "c", Operator: ProcessInstanceVariableFilterOperatorExists, Exists: &exists},
		{Name: "d", Operator: ProcessInstanceVariableFilterOperatorIn, Value: `["x","y"]`},
		{Name: "e", Operator: ProcessInstanceVariableFilterOperatorNotIn, Value: `["x","y"]`},
		{Name: "f", Operator: ProcessInstanceVariableFilterOperatorLike, Value: "*@example.com"},
	}}).Validate()

	require.NoError(t, err)
}
