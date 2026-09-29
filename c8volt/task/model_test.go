// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package task

import (
	"encoding/json"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/stretchr/testify/require"
)

// TestUserTaskVariablePreservesProcessVariableJSONContract verifies the public
// alias retains raw values, backend identity, scope, tenant, and truncation.
func TestUserTaskVariablePreservesProcessVariableJSONContract(t *testing.T) {
	t.Parallel()

	variable := UserTaskVariable{
		Name:               "approvalReason",
		Value:              "",
		VariableKey:        "2251799813685250",
		ProcessInstanceKey: "2251799813685249",
		ScopeKey:           "2251799813685251",
		TenantId:           "tenant-a",
		APITruncated:       true,
	}
	require.IsType(t, process.ProcessInstanceVariable{}, variable)

	raw, err := json.Marshal(variable)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"name":"approvalReason",
		"value":"",
		"variableKey":"2251799813685250",
		"processInstanceKey":"2251799813685249",
		"scopeKey":"2251799813685251",
		"tenantId":"tenant-a",
		"apiTruncated":true
	}`, string(raw))
}

// TestVariableEnrichedUserTasksPreservesInt64TotalAndInitializedCollections
// verifies the public wrappers retain task order and explicit empty arrays.
func TestVariableEnrichedUserTasksPreservesInt64TotalAndInitializedCollections(t *testing.T) {
	t.Parallel()

	enriched := VariableEnrichedUserTasks{
		Total: int64(1) << 40,
		Items: []VariableEnrichedUserTask{
			{
				Item:      UserTask{Key: "task-a", State: "CREATED", ProcessInstanceKey: "process-a"},
				Variables: []UserTaskVariable{},
			},
		},
	}

	raw, err := json.Marshal(enriched)
	require.NoError(t, err)
	require.JSONEq(t, `{
		"total":1099511627776,
		"items":[{
			"item":{"key":"task-a","state":"CREATED","processInstanceKey":"process-a"},
			"variables":[]
		}]
	}`, string(raw))
}

// TestVariableFilterAliasesPreserveTheProcessPredicateContract verifies task
// search inputs expose the existing ordered predicate model without changing
// returned user-task records.
func TestVariableFilterAliasesPreserveTheProcessPredicateContract(t *testing.T) {
	t.Parallel()

	exists := false
	filters := VariableFilterSet{Clauses: []VariableFilterClause{
		{Name: "status", Operator: VariableFilterOperatorEq, Value: `"approved"`, Source: "--var"},
		{Name: "payload", Operator: VariableFilterOperatorExists, Exists: &exists, Source: "--var-exists"},
	}}
	require.IsType(t, process.ProcessInstanceVariableFilterSet{}, filters)
	require.Equal(t, process.ProcessInstanceVariableFilterOperatorEq, VariableFilterOperatorEq)
	require.Equal(t, process.ProcessInstanceVariableFilterOperatorNeq, VariableFilterOperatorNeq)
	require.Equal(t, process.ProcessInstanceVariableFilterOperatorExists, VariableFilterOperatorExists)
	require.Equal(t, process.ProcessInstanceVariableFilterOperatorIn, VariableFilterOperatorIn)
	require.Equal(t, process.ProcessInstanceVariableFilterOperatorNotIn, VariableFilterOperatorNotIn)
	require.Equal(t, process.ProcessInstanceVariableFilterOperatorLike, VariableFilterOperatorLike)
	require.False(t, *filters.Clauses[1].Exists)

	raw, err := json.Marshal(SearchRequest{Assignee: "alice", VariableFilters: filters})
	require.NoError(t, err)
	require.JSONEq(t, `{
		"assignee":"alice",
		"variableFilters":{"clauses":[
			{"name":"status","operator":"$eq","value":"\"approved\"","source":"--var"},
			{"name":"payload","operator":"$exists","exists":false,"source":"--var-exists"}
		]}
	}`, string(raw))

	emptyRaw, err := json.Marshal(SearchRequest{VariableFilters: VariableFilterSet{}})
	require.NoError(t, err)
	require.JSONEq(t, `{"variableFilters":{}}`, string(emptyRaw))

	returnedRaw, err := json.Marshal(UserTask{Key: "task-a", State: "CREATED", ProcessInstanceKey: "process-a"})
	require.NoError(t, err)
	require.JSONEq(t, `{"key":"task-a","state":"CREATED","processInstanceKey":"process-a"}`, string(returnedRaw))
}
