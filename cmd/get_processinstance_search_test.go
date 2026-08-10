// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

// Verifies search-mode get process-instance sends the expected filter and pagination request shape.
func TestGetProcessInstanceSearchScaffold_UsesTempConfigAndCapturesSearchRequest(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	page := decodeCapturedPISearchPage(t, requests)

	require.Equal(t, "ACTIVE", filter["state"])
	require.EqualValues(t, 5, page["limit"])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get process-instance", got["command"])
}

// TestGetProcessInstanceSearch_VarExistsSendsNativeVariableFilters verifies the
// registered CLI flag reaches the native process-instance search body.
func TestGetProcessInstanceSearch_VarExistsSendsNativeVariableFilters(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--var-exists", "customerId",
		"--var-exists", "payload,email",
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	variables, ok := filter["variables"].([]any)
	require.True(t, ok, "expected native variable filters")
	require.Len(t, variables, 3)
	require.Equal(t, map[string]any{"name": "customerId", "value": map[string]any{"$exists": true}}, variables[0])
	require.Equal(t, map[string]any{"name": "payload", "value": map[string]any{"$exists": true}}, variables[1])
	require.Equal(t, map[string]any{"name": "email", "value": map[string]any{"$exists": true}}, variables[2])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get process-instance", got["command"])
}

// TestGetProcessInstanceSearch_VarSendsNativeEqualityFilters verifies equality
// shorthand reaches the native process-instance search body without losing commas.
func TestGetProcessInstanceSearch_VarSendsNativeEqualityFilters(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--var", `status="approved"`,
		"--var", `payload="payload,with,comma"`,
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	variables, ok := filter["variables"].([]any)
	require.True(t, ok, "expected native variable filters")
	require.Len(t, variables, 2)
	require.Equal(t, map[string]any{"name": "status", "value": map[string]any{"$eq": `"approved"`}}, variables[0])
	require.Equal(t, map[string]any{"name": "payload", "value": map[string]any{"$eq": `"payload,with,comma"`}}, variables[1])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get process-instance", got["command"])
}

// TestGetProcessInstanceSearch_VarLikeSendsNativeLikeFilters verifies like
// shorthand reaches the native search body without rewriting wildcard text.
func TestGetProcessInstanceSearch_VarLikeSendsNativeLikeFilters(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--var-like", `email=*@example.com,customerId=CUST-????`,
		"--var-like", `literal=invoice-\*`,
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	variables, ok := filter["variables"].([]any)
	require.True(t, ok, "expected native variable filters")
	require.Len(t, variables, 3)
	require.Equal(t, map[string]any{"name": "email", "value": map[string]any{"$like": `*@example.com`}}, variables[0])
	require.Equal(t, map[string]any{"name": "customerId", "value": map[string]any{"$like": `CUST-????`}}, variables[1])
	require.Equal(t, map[string]any{"name": "literal", "value": map[string]any{"$like": `invoice-\*`}}, variables[2])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get process-instance", got["command"])
}

// TestGetProcessInstanceSearch_VarSendsNativeAdvancedFilters verifies advanced
// operators reach the native process-instance search body with normalized names.
func TestGetProcessInstanceSearch_VarSendsNativeAdvancedFilters(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--var", `status.$neq="failed",active.$exists=false,kind.$in=["approved","pending"],segment.$notin=["legacy","test"]`,
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	variables, ok := filter["variables"].([]any)
	require.True(t, ok, "expected native variable filters")
	require.Len(t, variables, 4)
	require.Equal(t, map[string]any{"name": "status", "value": map[string]any{"$neq": `"failed"`}}, variables[0])
	require.Equal(t, map[string]any{"name": "active", "value": map[string]any{"$exists": false}}, variables[1])
	require.Equal(t, map[string]any{"name": "kind", "value": map[string]any{"$in": []any{"approved", "pending"}}}, variables[2])
	require.Equal(t, map[string]any{"name": "segment", "value": map[string]any{"$notIn": []any{"legacy", "test"}}}, variables[3])

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	require.Equal(t, string(OutcomeSucceeded), got["outcome"])
	require.Equal(t, "get process-instance", got["command"])
}

// TestGetProcessInstanceSearch_TenantScopedDiscoveryUsesSelectedTenant verifies
// c8volt-produced search candidates remain scoped by the effective tenant.
func TestGetProcessInstanceSearch_TenantScopedDiscoveryUsesSelectedTenant(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"hasIncident":false,"processDefinitionId":"tenant-a-process","processDefinitionKey":"9001","processDefinitionName":"tenant-a-process","processDefinitionVersion":3,"processInstanceKey":"101","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", tenantAdminKeysSelectedTenant,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--batch-size", "5",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	require.Equal(t, tenantAdminKeysSelectedTenant, filter["tenantId"])
	require.Equal(t, "ACTIVE", filter["state"])
	require.Contains(t, output, `"tenantId": "tenant-a"`)
	require.NotContains(t, output, tenantAdminKeysReturnedTenant)
}
