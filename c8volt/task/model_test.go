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
