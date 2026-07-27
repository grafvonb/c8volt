// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/consts"
	"github.com/grafvonb/c8volt/internal/exitcode"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
)

// TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface verifies help text exposes paging and automation contracts.
func TestGetProcessInstanceHelp_DocumentsPagingAndAutomationSurface(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "get", "process-instance", "--help")

	require.Contains(t, output, "Get process instances by key or by search criteria.")
	require.Contains(t, output, "Search results support interactive paging, scriptable JSON aggregation, and count-only workflows.")
	require.Contains(t, output, "matching process instances by process definition")
	require.Contains(t, output, "Direct key lookup stays strict")
	require.Contains(t, output, "Use --with-incidents to include direct incident details under matching process-instance rows in keyed or list/search output.")
	require.Contains(t, output, "Use --with-vars to include process-instance-scope variables under matching process-instance rows in keyed or list/search output.")
	require.Contains(t, output, "Use --with-elements to include runtime element instances under matching process-instance rows.")
	require.Contains(t, output, "Nested human element rows include dur:<duration>")
	require.Contains(t, output, "Use --with-listeners with --with-elements to include runtime listener jobs under matching element rows.")
	require.NotContains(t, output, "Add --incident-message-limit <chars> to shorten incident messages")
	require.Contains(t, output, "Run `c8volt get process-instance --help` for the complete flag reference.")
	require.Contains(t, output, "./c8volt get process-instance --bpmn-process-id <bpmn-process-id> --state active --limit 5")
	require.Contains(t, output, "./c8volt get process-instance --key <process-instance-key>")
	require.Contains(t, output, "./c8volt get process-instance --state active --total")
	require.Contains(t, output, "./c8volt get process-instance --has-user-tasks <user-task-key>")
	require.Contains(t, output, "./c8volt get process-instance --incidents-only --with-incidents --limit 5")
	require.Contains(t, output, "./c8volt get process-instance --direct-incidents-only --incident-error-type io_mapping_error --incident-error-message intentional --limit 5")
	require.Contains(t, output, `./c8volt get process-instance --var 'status="approved"' --limit 5`)
	require.Contains(t, output, "./c8volt get process-instance --state active --with-vars --var-value-limit 120 --limit 5")
	require.Contains(t, output, "./c8volt get process-instance --key <process-instance-key> --with-incidents")
	require.Contains(t, output, "./c8volt get process-instance --key <process-instance-key> --with-vars")
	require.Contains(t, output, "./c8volt get process-instance --key <process-instance-key> --with-vars --var-value-limit 120")
	require.Contains(t, output, "./c8volt get process-instance --key <process-instance-key> --with-elements")
	require.Contains(t, output, "./c8volt get process-instance --key <process-instance-key> --with-elements --with-listeners")
	require.Contains(t, output, "capped backend totals are counted by paging")
	require.Contains(t, output, "--auto-confirm")
	require.Contains(t, output, "--batch-size int32")
	require.Contains(t, output, "number of process instances to request per page; does not cap total returned rows")
	require.Contains(t, output, "--incident-message-limit int")
	require.Contains(t, output, "maximum characters to show for incident messages when --with-incidents is set")
	require.Contains(t, output, "--incident-error-message string")
	require.Contains(t, output, "case-insensitive incident error message substring filter for keyed --with-incidents or list/search --direct-incidents-only")
	require.Contains(t, output, "--incident-error-type string")
	require.Contains(t, output, "case-insensitive incident error type filter for keyed --with-incidents or list/search --direct-incidents-only")
	require.NotContains(t, output, "AD_HOC_SUB_PROCESS_NO_RETRIES")
	require.Contains(t, output, "--incident-state string")
	require.Contains(t, output, "incident state scope for keyed --with-incidents: active, pending, resolved, migrated, unknown, all")
	require.Contains(t, output, "--limit int32")
	require.Contains(t, output, "maximum number of matching process instances to return across all pages; omit to continue through all matches")
	require.Contains(t, output, "--var-value-limit int")
	require.Contains(t, output, "maximum characters to show for variable values when --with-vars is set")
	require.Contains(t, output, "--var stringArray")
	require.Contains(t, output, "require variable equality or advanced clause(s); repeat or separate clauses with commas")
	require.Contains(t, output, "--with-incidents")
	require.Contains(t, output, "include direct incident keys, states, and messages for keyed or list/search process-instance output")
	require.Contains(t, output, "--direct-incidents-only")
	require.Contains(t, output, "show only process instances with direct incident details")
	require.Contains(t, output, "--with-vars")
	require.Contains(t, output, "include process-instance-scope variables for keyed or list/search process-instance output")
	require.Contains(t, output, "--with-elements")
	require.Contains(t, output, "include runtime element instances for keyed or list/search process-instance output")
	require.Contains(t, output, "--with-listeners")
	require.Contains(t, output, "include runtime listener jobs under matching element rows; requires --with-elements")
	require.NotContains(t, output, "--count")
}

// TestGetProcessInstanceHelp_DocumentsVariableSearchContract protects the
// compact variable-search grammar and native wildcard documentation.
func TestGetProcessInstanceHelp_DocumentsVariableSearchContract(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "get", "process-instance", "--help")

	require.Contains(t, output, "Use variable-search flags to narrow list/search results natively on Camunda 8.8 and 8.9; Camunda 8.7 returns an unsupported-version error for those flags.")
	require.Contains(t, output, "--var-exists requires every listed variable name to exist.")
	require.Contains(t, output, "--var accepts name=value equality shorthand plus advanced name.$operator=value clauses for $eq, $neq, $exists, $in, $notIn, and $like; $notin is accepted as $notIn.")
	require.Contains(t, output, "--var-like uses native wildcard patterns: * matches zero or more characters, ? matches one character, and escaped wildcards remain literal.")
	require.Contains(t, output, "Commas inside quoted values and JSON arrays stay inside the variable clause.")
	require.Contains(t, output, "Variable scopeKey means the scope where the variable is directly defined.")
	require.Contains(t, output, `./c8volt get process-instance --var-exists payload,email --limit 5`)
	require.Contains(t, output, `./c8volt get process-instance --var 'status="approved"' --limit 5`)
	require.Contains(t, output, `./c8volt get process-instance --var 'status.$in=["approved","pending"]' --limit 5`)
	require.Contains(t, output, "./c8volt get process-instance --var-like 'email=*@example.com,customerId=CUST-????' --limit 5")
	require.Contains(t, output, "--var-exists stringArray")
	require.Contains(t, output, "require variable name(s) to exist; repeat or separate names with commas")
	require.Contains(t, output, "--var stringArray")
	require.Contains(t, output, "require variable equality or advanced clause(s); repeat or separate clauses with commas")
	require.Contains(t, output, "--var-like stringArray")
	require.Contains(t, output, "require variable value pattern clause(s); repeat or separate clauses with commas")
}

// Verifies help text documents has-user-tasks as a compact lookup selector without overloaded examples.
func TestGetProcessInstanceHelp_DocumentsHasUserTasksLookup(t *testing.T) {
	output := executeRootForProcessInstanceTest(t, "get", "process-instance", "--help")

	require.Contains(t, output, "--has-user-tasks strings")
	require.Contains(t, output, "user task key(s) whose owning process instances should be fetched")
	require.Contains(t, output, "./c8volt get process-instance --has-user-tasks <user-task-key>")
	require.NotContains(t, output, "./c8volt get process-instance --has-user-tasks 2251799815391233 --has-user-tasks 2251799815391244")
	require.Contains(t, output, "Use --has-user-tasks to fetch process instances by their owning user-task keys.")
	require.NotContains(t, output, "Camunda v2 user-task search first")
	require.NotContains(t, output, "Tasklist V1 lookup for legacy user-task compatibility")
	require.NotContains(t, output, "Camunda 8.7 remains unsupported")
	require.NotContains(t, output, "There is no Tasklist or Operate fallback")
}

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

// TestGetProcessInstanceKey_SelectedTenantMismatchUsesDirectBackendLookup
// protects explicit admin-input keys from being converted into tenant-scoped search.
func TestGetProcessInstanceKey_SelectedTenantMismatchUsesDirectBackendLookup(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/"+tenantAdminKeysProcessInstanceKey, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(tenantAdminKeysMismatchProcessInstanceJSON()))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", tenantAdminKeysSelectedTenant,
		"--json",
		"get", "process-instance",
		"--key", tenantAdminKeysProcessInstanceKey,
	)

	require.Equal(t, []string{"GET /v2/process-instances/" + tenantAdminKeysProcessInstanceKey}, requests)
	require.Contains(t, output, `"tenantId": "tenant-b"`)
	require.Contains(t, output, `"key": "`+tenantAdminKeysProcessInstanceKey+`"`)
}

// A missing BPMN selector should fail before process-instance search can masquerade as a real empty result.
func TestGetProcessInstanceBpmnSelectorMissingFailsBeforeSearch(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceBpmnSelectorMissingFailsBeforeSearchHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[missing-process]")
	require.NotContains(t, string(output), "bpmnProcessId:")
	require.NotContains(t, string(output), "found: 0")
}

// A keys-only upstream pipeline command still validates the BPMN selector before emitting keys.
func TestGetProcessInstanceBpmnSelectorMissingKeysOnlyPipelineFailsUpstream(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-definitions/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceBpmnSelectorMissingKeysOnlyPipelineFailsUpstreamHelper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Equal(t, []string{"POST /v2/process-definitions/search"}, requests)
	require.Contains(t, string(output), "no visible process definition matches the provided selector")
	require.Contains(t, string(output), "[missing-process]")
	require.NotContains(t, string(output), "List visible process definitions?")
	require.NotContains(t, string(output), "found: 0")
}

// Visible definitions with no instances still use the normal empty-list path.
func TestGetProcessInstanceBpmnSelectorVisiblePreservesFoundZero(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"tenant-a","version":3}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/search":
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant-a",
		"get", "process-instance",
		"--bpmn-process-id", "order-process",
	)

	require.Equal(t, []string{"POST /v2/process-definitions/search", "POST /v2/process-instances/search"}, requests)
	require.Equal(t, "found: 0\n", output)
}

// Selector validation must use the same version, tag, and tenant context as the PI search itself.
func TestGetProcessInstanceBpmnSelectorValidationIncludesVersionTagAndTenant(t *testing.T) {
	var pdSearchBodies []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			pdSearchBodies = append(pdSearchBodies, body)
			_, _ = w.Write([]byte(`{"items":[{"processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"tenant-a","version":7,"versionTag":"stable"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/search":
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant-a",
		"get", "process-instance",
		"--bpmn-process-id", "order-process",
		"--pd-version", "7",
		"--pd-version-tag", "stable",
	)

	require.Equal(t, "found: 0\n", output)
	require.Len(t, pdSearchBodies, 1)
	filter := requireJSONObject(t, pdSearchBodies[0]["filter"])
	require.Equal(t, "order-process", filter["processDefinitionId"])
	require.Equal(t, float64(7), filter["version"])
	require.Equal(t, "stable", filter["versionTag"])
	require.Equal(t, "tenant-a", filter["tenantId"])
}

// TestGetProcessInstanceJSON_AddsAgeMetaField verifies JSON rows include age metadata.
func TestGetProcessInstanceJSON_AddsAgeMetaField(t *testing.T) {
	var requests []string
	srv := newProcessInstanceSearchCaptureServer(t, &requests)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
	)

	require.NotEmpty(t, requests)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &got))
	payload, ok := got["payload"].(map[string]any)
	require.True(t, ok)
	meta, ok := payload["meta"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, meta["withAge"])
}

// Protects default paged search output, which renders incrementally before the final collected list path can align rows.
func TestGetProcessInstanceSearch_HumanOutputAlignsIncrementalPage(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 3, 23, 19, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	var requests []string
	srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
		`{"items":[{"hasIncident":false,"processDefinitionId":"Short","processDefinitionKey":"9001","processDefinitionName":"Short","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"MuchLongerProcess","processDefinitionKey":"9002","processDefinitionName":"MuchLongerProcess","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"COMPLETED","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
	)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--batch-size", "2",
	)

	require.NotEmpty(t, requests)
	expectedLines := formatProcessInstanceFlatRows([]process.ProcessInstance{
		{
			Key:            "123",
			TenantId:       "tenant",
			BpmnProcessId:  "Short",
			ProcessVersion: 3,
			State:          process.StateActive,
			StartDate:      "2026-03-23T18:00:00Z",
		},
		{
			Key:            "124",
			TenantId:       "tenant",
			BpmnProcessId:  "MuchLongerProcess",
			ProcessVersion: 3,
			State:          process.StateCompleted,
			StartDate:      "2026-03-23T18:00:00Z",
		},
	})
	require.Equal(t, strings.Join(append(expectedLines, "found: 2", ""), "\n"), output)
	require.Contains(t, output, "Short             v3 ACTIVE")
}

// TestGetProcessInstanceTotalOutput verifies --total output uses exact fallback counting when backend totals are capped.
func TestGetProcessInstanceTotalOutput(t *testing.T) {
	t.Run("reported total prints only the numeric count without fetching later pages", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Zero(t, promptCalls)
		require.Equal(t, "3\n", stdout)
		require.Empty(t, stderr)
	})

	t.Run("incident detail filters count matching direct incidents", func(t *testing.T) {
		tests := []struct {
			name string
			args []string
		}{
			{name: "direct incidents only", args: []string{"--total", "--direct-incidents-only", "--incident-error-type", "io_mapping_error", "--incident-error-message", "intentional"}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var requests []string
				srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests = append(requests, r.Method+" "+r.URL.Path)
					w.Header().Set("Content-Type", "application/json")
					switch r.URL.Path {
					case "/v2/process-instances/search":
						require.Equal(t, http.MethodPost, r.Method)
						_, _ = w.Write([]byte(`{"items":[
							{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
							{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
						],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
					case "/v2/process-instances/123/incidents/search":
						require.Equal(t, http.MethodPost, r.Method)
						_, _ = w.Write([]byte(`{"items":[{"errorMessage":"Intentional mapping failure","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
					case "/v2/process-instances/124/incidents/search":
						require.Equal(t, http.MethodPost, r.Method)
						_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
					default:
						t.Fatalf("unexpected request path: %s", r.URL.Path)
					}
				}))
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
				args := append([]string{"--config", cfgPath, "--tenant", "tenant", "get", "process-instance"}, tt.args...)
				stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)

				require.ElementsMatch(t, []string{
					"POST /v2/process-instances/search",
					"POST /v2/process-instances/123/incidents/search",
					"POST /v2/process-instances/124/incidents/search",
				}, requests)
				require.Equal(t, "1\n", stdout)
				require.Empty(t, stderr)
			})
		}
	})

	t.Run("capped reported total falls back to cursor paging for exact count", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-1","startCursor":null}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-2","startCursor":"cursor-1"}}`,
			`{"items":[],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":null,"startCursor":"cursor-2"}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 3)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.Equal(t, "cursor-1", pages[1]["after"])
		require.NotContains(t, pages[1], "from")
		require.Equal(t, "cursor-2", pages[2]["after"])
		require.Zero(t, promptCalls)
		require.Equal(t, "3\n", stdout)
		require.Empty(t, stderr)
	})

	t.Run("verbose capped total logs progress through logger", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-1","startCursor":null}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-2","startCursor":"cursor-1"}}`,
			`{"items":[],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":null,"startCursor":"cursor-2"}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
			"--verbose",
		)

		require.Equal(t, "3\n", stdout)
		require.Contains(t, stderr, "INFO page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, stderr, "INFO page size: 2, current page: 1, total so far: 3, more matches: yes, next step: auto-continue")
		require.Contains(t, stderr, "INFO page size: 2, current page: 0, total so far: 3, more matches: no, next step: complete")
		require.NotContains(t, stderr, "\npage size:")
	})

	t.Run("debug capped total includes paging values", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-1","startCursor":null}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":"cursor-2","startCursor":"cursor-1"}}`,
			`{"items":[],"page":{"totalItems":10000,"hasMoreTotalItems":true,"endCursor":null,"startCursor":"cursor-2"}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--debug",
			"get", "process-instance",
			"--batch-size", "2",
			"--total",
		)

		require.Equal(t, "3\n", stdout)
		require.Contains(t, stderr, `DEBUG pi total page; mode offset, from 0, after "", limit 2, items 2, total before 0, total after 2`)
		require.Contains(t, stderr, `reported total 10000, reported kind lower_bound, end cursor "cursor-1"`)
		require.Contains(t, stderr, `DEBUG pi total page; mode cursor, from 0, after "cursor-1", limit 2, items 1, total before 2, total after 3`)
		require.Contains(t, stderr, `DEBUG pi total page; mode cursor, from 0, after "cursor-2", limit 2, items 0, total before 3, total after 3`)
		require.NotContains(t, stderr, "INFO page size:")
	})

	t.Run("zero matches still print zero only", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"get", "process-instance",
			"--total",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Equal(t, "0\n", stdout)
		require.Empty(t, stderr)
	})
}

// TestGetProcessInstanceSearchMachineOutputStaysProgressFree verifies paged search can later gain shared progress without corrupting JSON or key streams.
func TestGetProcessInstanceSearchMachineOutputStaysProgressFree(t *testing.T) {
	t.Run("json", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"cursor-1"}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:01:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false,"startCursor":"cursor-1"}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--batch-size", "1",
		)

		require.Len(t, requests, 2)
		require.Empty(t, stderr)
		require.NotContains(t, stdout, "preflight:")
		require.NotContains(t, stdout, "page size:")
		require.NotContains(t, stdout, "discovering process instances")
		var envelope map[string]any
		require.NoError(t, json.Unmarshal([]byte(stdout), &envelope))
		payload := requireJSONObject(t, envelope["payload"])
		items := payload["items"].([]any)
		require.Len(t, items, 2)
	})

	t.Run("keys only", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:01:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)
		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--keys-only",
			"get", "process-instance",
		)

		require.Len(t, requests, 1)
		require.Empty(t, stderr)
		require.Equal(t, "123\n124\n", stdout)
		require.NotContains(t, stdout, "preflight:")
		require.NotContains(t, stdout, "page size:")
	})
}

// TestGetProcessInstanceTotalValidation verifies --total rejects incompatible output and lookup modes.
func TestGetProcessInstanceTotalValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "key lookup stays on the strict single-resource path",
			helper: "TestGetProcessInstanceTotalWithKeyHelper",
			want:   "--total cannot be combined with --key",
		},
		{
			name:   "json output is rejected",
			helper: "TestGetProcessInstanceTotalWithJSONHelper",
			want:   "--total cannot be combined with --json",
		},
		{
			name:   "keys-only output is rejected",
			helper: "TestGetProcessInstanceTotalWithKeysOnlyHelper",
			want:   "--total cannot be combined with --keys-only",
		},
		{
			name:   "incident view output is rejected",
			helper: "TestGetProcessInstanceWithIncidentsWithTotalHelper",
			want:   "--total cannot be combined with --with-incidents",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

// TestGetProcessInstanceWithIncidentsValidation rejects enrichment combinations that cannot render incident details safely.
func TestGetProcessInstanceWithIncidentsValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "rejects search-mode incident filters",
			helper: "TestGetProcessInstanceWithIncidentsWithSearchFilterHelper",
			want:   "--with-incidents cannot be combined with search-mode filters",
		},
		{
			name:   "rejects direct and marker incident filters together",
			helper: "TestGetProcessInstanceDirectIncidentsOnlyWithIncidentsOnlyHelper",
			want:   "using --incidents-only, --direct-incidents-only, and --no-incidents-only together does not make sense",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

func TestGetProcessInstanceWithElementsValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "total output is rejected",
			helper: "TestGetProcessInstanceWithElementsWithTotalHelper",
			want:   "--total cannot be combined with --with-elements",
		},
		{
			name:   "keys-only output is rejected",
			helper: "TestGetProcessInstanceWithElementsWithKeysOnlyHelper",
			want:   "--keys-only cannot be combined with --with-elements",
		},
		{
			name:   "keyed search-mode filters are rejected",
			helper: "TestGetProcessInstanceWithElementsWithSearchFilterHelper",
			want:   "--with-elements cannot be combined with search-mode filters",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)
			require.NotEqual(t, 0, code)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestGetProcessInstanceWithListenersValidation verifies listener enrichment rejects modes that cannot render nested element-owned listener rows.
func TestGetProcessInstanceWithListenersValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "missing element context is rejected",
			helper: "TestGetProcessInstanceWithListenersWithoutElementsHelper",
			want:   "--with-listeners requires --with-elements",
		},
		{
			name:   "keys-only output is rejected",
			helper: "TestGetProcessInstanceWithListenersWithKeysOnlyHelper",
			want:   "--with-listeners cannot be combined with --keys-only",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)
			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

func TestGetProcessInstanceDirectIncidentsOnly_FiltersByLoadedDirectIncidents(t *testing.T) {
	var requests []string
	var searchBodies []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			searchBodies = append(searchBodies, string(body))
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":false,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"direct failure","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--direct-incidents-only",
	)

	require.ElementsMatch(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/process-instances/124/incidents/search",
	}, requests)
	require.Len(t, searchBodies, 1)
	require.NotContains(t, searchBodies[0], "hasIncident")
	require.NotContains(t, output, "123 tenant demo-a")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.Contains(t, output, "found: 1")
}

func TestGetProcessInstanceDirectIncidentsOnlyWithLimitUsesIncidentSearch(t *testing.T) {
	var requests testx.SafeSlice[string]
	var incidentSearchBodies testx.SafeSlice[string]
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Append(r.Method + " " + r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			incidentSearchBodies.Append(string(body))
			_, _ = w.Write([]byte(`{"items":[
				{"errorMessage":"intentional failure","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"},
				{"errorMessage":"intentional failure","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-124","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/124":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--direct-incidents-only",
		"--incident-error-type", "io_mapping_error",
		"--incident-error-message", "intentional",
		"--limit", "5",
	)

	gotRequests := requests.Snapshot()
	gotIncidentSearchBodies := incidentSearchBodies.Snapshot()
	require.NotEmpty(t, gotRequests)
	require.Equal(t, "POST /v2/incidents/search", gotRequests[0])
	require.ElementsMatch(t, []string{
		"GET /v2/process-instances/123",
		"GET /v2/process-instances/124",
	}, gotRequests[1:])
	require.Len(t, gotIncidentSearchBodies, 1)
	require.NotContains(t, gotIncidentSearchBodies[0], "process-instances")
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "124 tenant demo v3 ACTIVE")
	require.Contains(t, output, "found: 2")
}

func TestGetProcessInstanceIncidentFlags_PreserveSearchAndEnrichmentContracts(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		processResponse    string
		incidentResponses  map[string]string
		wantSearchFilter   string
		wantNoSearchFilter string
		wantRequests       []string
		wantOutput         []string
		wantNoOutput       []string
	}{
		{
			name:            "with incidents enriches matching rows without changing process search filter",
			args:            []string{"get", "process-instance", "--with-incidents"},
			processResponse: `{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
			incidentResponses: map[string]string{
				"/v2/process-instances/123/incidents/search": `{"items":[{"errorMessage":"direct failure","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
			},
			wantNoSearchFilter: "hasIncident",
			wantRequests: []string{
				"POST /v2/process-instances/search",
				"POST /v2/process-instances/123/incidents/search",
			},
			wantOutput: []string{"123 tenant demo v3 ACTIVE", "incident-123", "found: 1"},
		},
		{
			name:             "incidents only stays on marker filter and does not load direct incidents",
			args:             []string{"get", "process-instance", "--incidents-only"},
			processResponse:  `{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
			wantSearchFilter: `"hasIncident":true`,
			wantRequests:     []string{"POST /v2/process-instances/search"},
			wantOutput:       []string{"123 tenant demo v3 ACTIVE", "found: 1"},
			wantNoOutput:     []string{"incidents:"},
		},
		{
			name: "direct incidents only filters after loading direct incidents",
			args: []string{"get", "process-instance", "--direct-incidents-only"},
			processResponse: `{"items":[
				{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":false,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
			incidentResponses: map[string]string{
				"/v2/process-instances/123/incidents/search": `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
				"/v2/process-instances/124/incidents/search": `{"items":[{"errorMessage":"direct failure","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
			},
			wantNoSearchFilter: "hasIncident",
			wantRequests: []string{
				"POST /v2/process-instances/search",
				"POST /v2/process-instances/123/incidents/search",
				"POST /v2/process-instances/124/incidents/search",
			},
			wantOutput:   []string{"124 tenant demo-b v4 ACTIVE", "found: 1"},
			wantNoOutput: []string{"123 tenant demo-a"},
		},
		{
			name:             "no incidents only stays on marker filter and omits incident rows",
			args:             []string{"get", "process-instance", "--no-incidents-only"},
			processResponse:  `{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
			wantSearchFilter: `"hasIncident":false`,
			wantRequests:     []string{"POST /v2/process-instances/search"},
			wantOutput:       []string{"124 tenant demo v3 ACTIVE", "found: 1"},
			wantNoOutput:     []string{"incidents:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			var searchBodies []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-instances/search":
					require.Equal(t, http.MethodPost, r.Method)
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					searchBodies = append(searchBodies, string(body))
					_, _ = w.Write([]byte(tt.processResponse))
				default:
					response, ok := tt.incidentResponses[r.URL.Path]
					if !ok {
						t.Fatalf("unexpected request path: %s", r.URL.Path)
					}
					require.Equal(t, http.MethodPost, r.Method)
					_, _ = w.Write([]byte(response))
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			args := append([]string{"--config", cfgPath, "--tenant", "tenant"}, tt.args...)

			output := executeRootForProcessInstanceTest(t, args...)

			require.ElementsMatch(t, tt.wantRequests, requests)
			require.Len(t, searchBodies, 1)
			if tt.wantSearchFilter != "" {
				require.Contains(t, searchBodies[0], tt.wantSearchFilter)
			}
			if tt.wantNoSearchFilter != "" {
				require.NotContains(t, searchBodies[0], tt.wantNoSearchFilter)
			}
			for _, want := range tt.wantOutput {
				require.Contains(t, output, want)
			}
			for _, unwanted := range tt.wantNoOutput {
				require.NotContains(t, output, unwanted)
			}
		})
	}
}

// TestGetProcessInstanceWithIncidents_ListSearchWithoutKeyIsAccepted verifies list/search incident enrichment is no longer keyed-only.
func TestGetProcessInstanceWithIncidents_ListSearchWithoutKeyIsAccepted(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v2/process-instances/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--state", "active",
		"--with-incidents",
	)

	require.Equal(t, []string{"POST /v2/process-instances/search"}, requests)
	require.Equal(t, "found: 0\n", output)
}

// TestGetProcessInstanceListWithVars_HumanOutputShowsProcessScopeVariables verifies list/search variable enrichment matches keyed rendering.
func TestGetProcessInstanceListWithVars_HumanOutputShowsProcessScopeVariables(t *testing.T) {
	var requests []string
	var variableFilters []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(body, &got))
			filter := requireJSONObject(t, got["filter"])
			require.Equal(t, "ACTIVE", filter["state"])
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":false,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":false,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			variableFilters = append(variableFilters, filter)
			switch filter["processInstanceKey"] {
			case "123":
				_, _ = w.Write([]byte(`{"items":[{"name":"zeta","value":"2","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"},{"name":"localTask","value":"ignored","variableKey":"903","processInstanceKey":"123","scopeKey":"element-123","tenantId":"tenant"},{"name":"alpha","value":"1","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`))
			case "124":
				_, _ = w.Write([]byte(`{"items":[{"name":"only","value":"yes","variableKey":"904","processInstanceKey":"124","scopeKey":"124","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected variable filter: %#v", filter)
			}
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--state", "active",
		"--with-vars",
	)

	require.Equal(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/variables/search",
		"POST /v2/variables/search",
	}, requests)
	require.Len(t, variableFilters, 2)
	require.Equal(t, "123", variableFilters[0]["processInstanceKey"])
	require.Equal(t, "123", variableFilters[0]["scopeKey"])
	require.Equal(t, "tenant", variableFilters[0]["tenantId"])
	require.Equal(t, "124", variableFilters[1]["processInstanceKey"])
	require.Equal(t, "124", variableFilters[1]["scopeKey"])
	require.Equal(t, "tenant", variableFilters[1]["tenantId"])
	require.Contains(t, output, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, output, "└─ vars:\n   ├─ alpha=1\n   └─ zeta=2")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.Contains(t, output, "└─ vars:\n   └─ only=yes")
	require.NotContains(t, output, "localTask")
	require.NotContains(t, output, "incidents:")
	require.Contains(t, output, "found: 2")
	require.Less(t, strings.Index(output, "123 tenant demo-a"), strings.Index(output, "alpha=1"))
	require.Less(t, strings.Index(output, "zeta=2"), strings.Index(output, "124 tenant demo-b"))
}

// Combined enrichment keeps variables before incidents so runtime context leads the failure detail.
func TestGetProcessInstanceListWithVarsAndIncidents_HumanOutputShowsGroupedSections(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"name":"hasIncident","value":"true","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--state", "active",
		"--with-vars",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/variables/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ vars:\n│  └─ hasIncident=true")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 JOB_NO_RETRIES ACTIVE j:n/a m:No retries left")
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "└─ incidents:"))
}

// TestGetProcessInstanceListWithVarsIncidentsAndElements_HumanOutputShowsGroupedSections verifies bounded search combines each requested enrichment once.
func TestGetProcessInstanceListWithVarsIncidentsAndElements_HumanOutputShowsGroupedSections(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"name":"hasIncident","value":"true","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:01Z","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":true,"incidentKey":"incident-123"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--state", "active",
		"--limit", "1",
		"--with-vars",
		"--with-incidents",
		"--with-elements",
	)

	require.Equal(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/variables/search",
		"POST /v2/element-instances/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ vars:\n│  └─ hasIncident=true")
	require.Contains(t, output, "├─ incidents:\n│  └─ incident-123 JOB_NO_RETRIES ACTIVE j:n/a e:task-a ei:element-123 m:No retries left")
	require.Contains(t, output, "└─ elements:\n   └─ element-1 SERVICE_TASK task-a ACTIVE s:2026-07-15T10:12:01.000")
	require.Contains(t, output, "dur:")
	require.Contains(t, output, "inc!:incident-123")
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "├─ incidents:"))
	require.Less(t, strings.Index(output, "├─ incidents:"), strings.Index(output, "└─ elements:"))
}

// TestGetProcessInstanceListWithIncidents_HumanOutputShowsDirectIncidentLines verifies list/search incident enrichment keeps incidents under their owning rows.
func TestGetProcessInstanceListWithIncidents_HumanOutputShowsDirectIncidentLines(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			var got map[string]any
			require.NoError(t, json.Unmarshal(body, &got))
			filter := requireJSONObject(t, got["filter"])
			require.Equal(t, true, filter["hasIncident"])
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"First key failed","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"Second key failed","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--incidents-only",
		"--with-incidents",
	)

	require.ElementsMatch(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/process-instances/124/incidents/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 ACTIVE j:n/a m:First key failed")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-124 ACTIVE j:n/a m:Second key failed")
	require.Contains(t, output, "found: 2")
	require.Less(t, strings.Index(output, "123 tenant demo-a"), strings.Index(output, "incident-123"))
	require.Less(t, strings.Index(output, "incident-123"), strings.Index(output, "124 tenant demo-b"))
	require.Less(t, strings.Index(output, "124 tenant demo-b"), strings.Index(output, "incident-124"))
}

// TestGetProcessInstanceListWithIncidents_LooksUpOnlyLimitedRows guards paging and --limit compatibility for incident lookups.
func TestGetProcessInstanceListWithIncidents_LooksUpOnlyLimitedRows(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"First key failed","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			t.Fatalf("incident lookup should not run for rows outside --limit: %s", r.URL.Path)
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--batch-size", "2",
		"--limit", "1",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 ACTIVE j:n/a m:First key failed")
	require.NotContains(t, output, "124 tenant")
	require.Contains(t, output, "found: 1")
}

// TestGetProcessInstanceListWithIncidents_HumanIndirectMarkerExplainsEmptyDirectIncidents verifies list rows marked inc! stay explainable when direct lookup is empty.
func TestGetProcessInstanceListWithIncidents_HumanIndirectMarkerExplainsEmptyDirectIncidents(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search", "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--incidents-only",
		"--with-incidents",
	)

	require.ElementsMatch(t, []string{
		"POST /v2/process-instances/search",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/process-instances/124/incidents/search",
	}, requests)
	require.Contains(t, stdout, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, stdout, "124 tenant demo-b v4 ACTIVE")
	require.Equal(t, 2, strings.Count(stdout, "└─ "+indirectProcessTreeIncidentNote))
	require.Contains(t, stdout, "found: 2")
	require.NotContains(t, stdout, indirectProcessTreeIncidentWarning)
	require.Equal(t, 1, strings.Count(stderr, indirectProcessTreeIncidentWarning))
	require.Less(t, strings.Index(stdout, "123 tenant demo-a"), strings.Index(stdout, "└─ "+indirectProcessTreeIncidentNote))
	require.Less(t, strings.Index(stdout, "124 tenant demo-b"), strings.LastIndex(stdout, "└─ "+indirectProcessTreeIncidentNote))
	require.Less(t, strings.LastIndex(stdout, "└─ "+indirectProcessTreeIncidentNote), strings.Index(stdout, "found: 2"))
}

// TestGetProcessInstanceIncidentMessageLimitValidation rejects unsafe incident message limit usage.
func TestGetProcessInstanceIncidentMessageLimitValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "requires with-incidents",
			helper: "TestGetProcessInstanceIncidentMessageLimitWithoutIncidentsHelper",
			want:   "--incident-message-limit requires --with-incidents",
		},
		{
			name:   "rejects negative limit",
			helper: "TestGetProcessInstanceIncidentMessageLimitNegativeHelper",
			want:   "invalid value for --incident-message-limit: -1, expected non-negative integer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

func TestGetProcessInstanceIncidentStateValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "list/search requires direct incident filtering",
			helper: "TestGetProcessInstanceIncidentStateWithoutIncidentsHelper",
			want:   "--incident-state requires --direct-incidents-only for list/search process-instance filtering",
		},
		{
			name:   "with-incidents does not make list/search incident-state a filter",
			helper: "TestGetProcessInstanceIncidentStateListSearchHelper",
			want:   "--incident-state requires --direct-incidents-only for list/search process-instance filtering",
		},
		{
			name:   "rejects unsupported value",
			helper: "TestGetProcessInstanceIncidentStateInvalidHelper",
			want:   `invalid value for --incident-state: "closed", valid values are: active, pending, resolved, migrated, unknown, all`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

func TestGetProcessInstanceIncidentStateValidationAcceptsCaseInsensitiveEnum(t *testing.T) {
	require.NoError(t, validatePIIncidentStateFlag(" RESOLVED "))
}

func TestGetProcessInstanceIncidentDetailFilterValidation(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "error type list/search requires direct incident filtering",
			helper: "TestGetProcessInstanceIncidentErrorTypeWithoutIncidentsHelper",
			want:   "--incident-error-type requires --direct-incidents-only for list/search process-instance filtering",
		},
		{
			name:   "error message list/search requires direct incident filtering",
			helper: "TestGetProcessInstanceIncidentErrorMessageWithoutIncidentsHelper",
			want:   "--incident-error-message requires --direct-incidents-only for list/search process-instance filtering",
		},
		{
			name:   "rejects unsupported error type",
			helper: "TestGetProcessInstanceIncidentErrorTypeInvalidHelper",
			want:   `invalid value for --incident-error-type: "retry_error"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
		})
	}
}

func TestValidatePIIncidentErrorTypeFlag_AcceptsAnyCaseEnumValue(t *testing.T) {
	require.NoError(t, validatePIIncidentErrorTypeFlag(""))
	require.NoError(t, validatePIIncidentErrorTypeFlag("io_mapping_error"))
	require.NoError(t, validatePIIncidentErrorTypeFlag("Job_No_Retries"))
}

func TestGetProcessInstanceVarValueLimitValidation(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIVarValueLimit = -1
	err := validatePISearchFlags()
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid value for --var-value-limit: -1, expected non-negative integer")

	resetProcessInstanceCommandGlobals()
	cmd := &cobra.Command{Use: "process-instance"}
	fs := pflag.NewFlagSet("process-instance", pflag.ContinueOnError)
	fs.Int("var-value-limit", 0, "")
	cmd.Flags().AddFlagSet(fs)
	require.NoError(t, cmd.Flags().Set("var-value-limit", "80"))
	flagGetPIVarValueLimit = 80

	err = validatePISearchFlags(cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--var-value-limit requires --with-vars")

	flagGetPIWithVars = true
	err = validatePISearchFlags(cmd)
	require.NoError(t, err)
}

// TestGetProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags verifies paging flag validation errors stay user-facing.
func TestGetProcessInstanceCommand_RejectsInvalidLimitAndRemovedCountFlags(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

	tests := []struct {
		name   string
		helper string
		want   string
	}{
		{
			name:   "removed count flag is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper",
			want:   "unknown flag: --count",
		},
		{
			name:   "non-positive limit is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsInvalidLimitHelper",
			want:   "--limit must be positive integer",
		},
		{
			name:   "limit cannot be combined with key",
			helper: "TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper",
			want:   "--limit cannot be combined with --key",
		},
		{
			name:   "limit cannot be combined with total",
			helper: "TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper",
			want:   "--total cannot be combined with --limit",
		},
		{
			name:   "invalid batch size is rejected",
			helper: "TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper",
			want:   "invalid value for --batch-size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelper(t, tt.helper, cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, tt.want)
		})
	}
}

// TestApplyPISearchResultFilters_OrphanChildrenUseCommandActivity verifies orphan filtering is wrapped in command activity output.
func TestApplyPISearchResultFilters_OrphanChildrenUseCommandActivity(t *testing.T) {
	prevOrphanOnly := flagGetPIOrphanChildrenOnly
	t.Cleanup(func() {
		flagGetPIOrphanChildrenOnly = prevOrphanOnly
	})
	flagGetPIOrphanChildrenOnly = true

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))
	cliFilterCalls := 0
	cli := stubProcessAPI{filterOrphanParent: func(ctx context.Context, items []process.ProcessInstance, opts ...options.FacadeOption) ([]process.ProcessInstance, error) {
		cliFilterCalls++
		return items[:1], nil
	}}

	pis := process.ProcessInstances{
		Total: 2,
		Items: []process.ProcessInstance{
			{Key: "123", ParentKey: "456"},
			{Key: "124", ParentKey: "457"},
		},
	}

	got, err := applyPISearchResultFilters(cmd, cli, pis)
	require.NoError(t, err)
	require.Equal(t, 1, cliFilterCalls)
	require.Len(t, got.Items, 1)
	require.EqualValues(t, 1, got.Total)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"checking orphan parents for 2 process instance(s)"}, msgs)
}

func TestGetProcessInstanceOrphanChildrenOnly_UsesRealFacadeDiscovery(t *testing.T) {
	var searchRequests []string
	var getPaths []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v2/process-instances/search":
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			searchRequests = append(searchRequests, string(body))
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","parentProcessInstanceKey":"456","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/v2/process-instances/456":
			getPaths = append(getPaths, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "pi",
		"--orphan-children-only",
	)

	require.NotContains(t, stderr, "process client does not support shared orphan discovery")
	require.NotContains(t, stderr, "internal error")
	require.Contains(t, stdout, "123")
	require.Equal(t, []string{"/v2/process-instances/456"}, getPaths)
	filters := decodeCapturedPISearchRequests(t, searchRequests)
	require.Len(t, filters, 1)
	topLevelFilter, ok := filters[0]["filter"].(map[string]any)
	require.True(t, ok)
	require.NotContains(t, topLevelFilter, "processInstanceKey")
	require.Contains(t, topLevelFilter, "parentProcessInstanceKey")
}

func TestSearchOrphanProcessInstancesWithSharedDiscovery_UsesCommandActivity(t *testing.T) {
	prevLimit := flagGetPILimit
	prevDirectIncidentsOnly := flagGetPIDirectIncidentsOnly
	t.Cleanup(func() {
		flagGetPILimit = prevLimit
		flagGetPIDirectIncidentsOnly = prevDirectIncidentsOnly
	})
	flagGetPILimit = 25
	flagGetPIDirectIncidentsOnly = false

	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	discoverCalls := 0
	cli := stubProcessAPI{discoverOrphans: func(_ context.Context, request process.OrphanDiscoveryRequest, _ ...options.FacadeOption) (process.OrphanDiscovery, error) {
		discoverCalls++
		require.EqualValues(t, consts.MaxPISearchSize, request.BatchSize)
		require.EqualValues(t, 25, request.Limit)
		require.NotNil(t, request.Progress)
		request.Progress(process.OrphanDiscoveryProgress{
			Page:                  1,
			Phase:                 "checking",
			CurrentPageCandidates: 2,
			CandidatesChecked:     0,
			OrphansFound:          0,
			Limit:                 request.Limit,
		})
		request.Progress(process.OrphanDiscoveryProgress{
			Page:               1,
			Phase:              "checked",
			CurrentPageOrphans: 1,
			CandidatesChecked:  2,
			OrphansFound:       1,
			Limit:              request.Limit,
		})
		return process.OrphanDiscovery{
			Items: []process.ProcessInstance{{Key: "123", ParentKey: "456"}},
			Keys:  []string{"123"},
		}, nil
	}}

	got, renderedIncrementally, err := searchOrphanProcessInstancesWithSharedDiscovery(cmd, cli, nil, process.ProcessInstanceFilter{})

	require.NoError(t, err)
	require.False(t, renderedIncrementally)
	require.Equal(t, 1, discoverCalls)
	require.EqualValues(t, 1, got.Total)
	require.Equal(t, []process.ProcessInstance{{Key: "123", ParentKey: "456"}}, got.Items)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"discovering orphan child process instances"}, msgs)
	require.Equal(t, []string{
		"orphan search: page 1 checking 2 child process instance(s) for missing parents; checked 0, found 0 orphan child process instance(s)",
		"orphan search: page 1 checked 2 child process instance(s), found 1 on page, 1 total",
	}, sink.Updates())
}

func TestEnrichProcessInstancesWithIncidentActivity_UsesCommandActivity(t *testing.T) {
	sink := &activitysink.Sink{}
	cmd := &cobra.Command{}
	cmd.SetContext(logging.ToActivityContext(context.Background(), sink))

	enrichCalls := 0
	cli := stubProcessAPI{enrichProcessInstances: func(_ context.Context, pis process.ProcessInstances, _ ...options.FacadeOption) (process.IncidentEnrichedProcessInstances, error) {
		enrichCalls++
		require.Equal(t, []process.ProcessInstance{{Key: "123"}, {Key: "124"}}, pis.Items)
		return process.IncidentEnrichedProcessInstances{
			Total: pis.Total,
			Items: []process.IncidentEnrichedProcessInstance{
				{Item: pis.Items[0]},
				{Item: pis.Items[1]},
			},
		}, nil
	}}

	got, err := enrichProcessInstancesWithIncidentActivity(cmd, cli, process.ProcessInstances{
		Total: 2,
		Items: []process.ProcessInstance{{Key: "123"}, {Key: "124"}},
	})

	require.NoError(t, err)
	require.Equal(t, 1, enrichCalls)
	require.Len(t, got.Items, 2)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"loading incident details for 2 process instance(s)"}, msgs)
}

// TestGetProcessInstanceKeyLookup_UsesGeneratedLookupEndpoint verifies direct key lookup uses the versioned generated endpoint.
func TestGetProcessInstanceKeyLookup_UsesGeneratedLookupEndpoint(t *testing.T) {
	response := `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`

	t.Run("explicit flag tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"--tenant", "tenant-a",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("environment tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTestWithEnv(t,
			[]string{"C8VOLT_APP_TENANT=tenant-a"},
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("profile tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `active_profile: base
app:
  camunda_version: 8.8
  tenant: base-tenant
apis:
  camunda_api:
    base_url: `+srv.URL+`
profiles:
  dev:
    app:
      tenant: tenant-a
    apis:
      camunda_api:
        base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"--profile", "dev",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})

	t.Run("base config tenant", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodGet, r.Method)
			require.Equal(t, "/v2/process-instances/123", r.URL.Path)
			requests = append(requests, r.URL.Path)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(response))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeRawTestConfig(t, `app:
  camunda_version: 8.8
  tenant: tenant-a
apis:
  camunda_api:
    base_url: `+srv.URL+`
`)

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--key", "123",
		)

		require.Equal(t, []string{"/v2/process-instances/123"}, requests)
		require.Contains(t, output, `"tenantId": "tenant-a"`)
	})
}

// TestGetProcessInstanceWithIncidents_HumanOutputShowsOneIncident verifies the direct incident line includes the incident key.
func TestGetProcessInstanceWithIncidents_HumanOutputShowsOneIncident(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	var requests []string
	var incidentBodies []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			incidentBodies = append(incidentBodies, string(body))
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","jobKey":"job-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123", "POST /v2/process-instances/123/incidents/search"}, requests)
	require.Len(t, incidentBodies, 1)
	require.NotContains(t, incidentBodies[0], "processInstanceKey")
	require.NotContains(t, incidentBodies[0], `"state"`)
	require.Contains(t, output, "123")
	require.Contains(t, output, "demo v3")
	require.Contains(t, output, "inc!")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 JOB_NO_RETRIES ACTIVE j:job-123 2026-03-23T18:01:00.000 (48 days ago) e:task-a ei:element-123 m:No retries left")
	require.Contains(t, output, "found: 1")
}

func TestGetProcessInstanceWithIncidents_FiltersIncidentDetailsByTypeAndMessageCaseInsensitive(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"Intentional mapping failure","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-match","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"},
				{"creationTime":"2026-03-23T18:02:00Z","elementId":"task-b","elementInstanceKey":"element-124","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-type-miss","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"},
				{"creationTime":"2026-03-23T18:03:00Z","elementId":"task-c","elementInstanceKey":"element-125","errorMessage":"Other mapping failure","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-message-miss","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":3,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
		"--incident-error-type", "io_mapping_error",
		"--incident-error-message", "intentional",
	)

	require.Contains(t, output, "incident-match")
	require.NotContains(t, output, "incident-type-miss")
	require.NotContains(t, output, "incident-message-miss")
}

func TestGetProcessInstanceWithIncidents_AllStateOmitsStateFilterAndShowsResolvedState(t *testing.T) {
	var incidentBodies []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			incidentBodies = append(incidentBodies, string(body))
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"resolved earlier","incidentKey":"incident-123","processInstanceKey":"123","state":"RESOLVED","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
		"--incident-state", "all",
	)

	require.Len(t, incidentBodies, 1)
	require.NotContains(t, incidentBodies[0], `"state"`)
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 RESOLVED j:n/a m:resolved earlier")
	require.Contains(t, output, "found: 1")
}

// Protects the short `get process-instance --with-incidents` workflow used before resolving incidents.
func TestGetPIWithIncidents_AliasPreservesIncidentLookupOutput(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/2251799813711967/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"2251799813685249","jobKey":"2251799813685251","processInstanceKey":"2251799813711967","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "pi",
		"--key", "2251799813711967",
		"--with-incidents",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/2251799813711967",
		"POST /v2/process-instances/2251799813711967/incidents/search",
	}, requests)
	require.Contains(t, output, "2251799813711967 tenant demo v3 ACTIVE")
	require.Contains(t, output, "inc!")
	require.Contains(t, output, "└─ incidents:\n   └─ 2251799813685249 JOB_NO_RETRIES ACTIVE j:2251799813685251 m:No retries left")
	require.Contains(t, output, "found: 1")
}

func TestGetProcessInstanceWithIncidents_HumanIncidentMessageLimitTruncatesMessageOnly(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo-process","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left after worker failure","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
		"--incident-message-limit", "7",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123", "POST /v2/process-instances/123/incidents/search"}, requests)
	require.Contains(t, output, "123 tenant demo-process v3 ACTIVE")
	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 ACTIVE j:n/a m:No retr...")
	require.NotContains(t, output, "No retries left after worker failure")
}

func TestGetProcessInstanceWithIncidents_HumanIncidentMessageLimitDefaultLeavesMessageUnchanged(t *testing.T) {
	fullMessage := "No retries left after worker failure"
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"` + fullMessage + `","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	require.Contains(t, output, "└─ incidents:\n   └─ incident-123 ACTIVE j:n/a m:"+fullMessage)
	require.NotContains(t, output, fullMessage[:7]+"...")
}

// Keyed variable lookup is limited to process-scope variables and preserves the normal PI row.
func TestGetProcessInstanceWithVars_HumanOutputShowsSortedProcessScopeVariables(t *testing.T) {
	var requests []string
	var variableBodies []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			variableBodies = append(variableBodies, body)
			_, _ = w.Write([]byte(`{"items":[{"name":"zeta","value":"2","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"},{"name":"localTask","value":"ignored","variableKey":"903","processInstanceKey":"123","scopeKey":"element-123","tenantId":"tenant"},{"name":"alpha","value":"1","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--key", "123",
		"--with-vars",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123", "POST /v2/variables/search"}, requests)
	require.Len(t, variableBodies, 1)
	filter := variableBodies[0]["filter"].(map[string]any)
	require.Equal(t, "123", filter["processInstanceKey"])
	require.Equal(t, "123", filter["scopeKey"])
	require.Equal(t, "tenant", filter["tenantId"])
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "└─ vars:")
	require.Contains(t, output, "├─ alpha=1")
	require.Contains(t, output, "└─ zeta=2")
	require.NotContains(t, output, "localTask")
	require.NotContains(t, output, "var alpha")
	require.NotContains(t, output, "incidents:")
	require.NotContains(t, output, "process instance is marked as having incidents")
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "123 tenant demo"), strings.Index(output, "└─ vars:"))
	require.Less(t, strings.Index(output, "alpha=1"), strings.Index(output, "zeta=2"))
}

func TestGetProcessInstanceWithElements_HumanOutputShowsRuntimeElements(t *testing.T) {
	var requests []string
	var elementFilters []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			elementFilters = append(elementFilters, filter)
			_, _ = w.Write([]byte(`{"items":[
				{"elementInstanceKey":"element-3","elementId":"ship-order","elementName":"Ship order","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:04Z","processInstanceKey":"123","rootProcessInstanceKey":"123","processDefinitionId":"demo","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":true,"incidentKey":"incident-777"},
				{"elementInstanceKey":"element-1","elementId":"start","elementName":"Start","type":"START_EVENT","state":"COMPLETED","startDate":"2026-07-15T10:12:01Z","endDate":"2026-07-15T10:12:02Z","processInstanceKey":"123","rootProcessInstanceKey":"123","processDefinitionId":"demo","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false},
				{"elementInstanceKey":"ignored","elementId":"other","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:03Z","processInstanceKey":"999","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false},
				{"elementInstanceKey":"element-2","elementId":"review","elementName":"Review","type":"USER_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:04Z","processInstanceKey":"123","rootProcessInstanceKey":"123","processDefinitionId":"demo","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":true}
			],"page":{"totalItems":4,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--key", "123",
		"--with-elements",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123", "POST /v2/element-instances/search"}, requests)
	require.Len(t, elementFilters, 1)
	require.Equal(t, "123", elementFilters[0]["processInstanceKey"])
	require.NotContains(t, elementFilters[0], "tenantId")
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "└─ elements:")
	require.Contains(t, output, "element-1 START_EVENT  start      COMPLETED s:2026-07-15T10:12:01.000 e:2026-07-15T10:12:02.000 dur:1s")
	require.Contains(t, output, "element-2 USER_TASK    review     ACTIVE    s:2026-07-15T10:12:04.000")
	require.Contains(t, output, "element-3 SERVICE_TASK ship-order ACTIVE    s:2026-07-15T10:12:04.000")
	require.Contains(t, output, "dur:")
	require.Contains(t, output, "inc!")
	require.Contains(t, output, "inc!:incident-777")
	require.NotContains(t, output, "ignored")
	require.NotContains(t, output, "element:ship-order")
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "element-1"), strings.Index(output, "element-2"))
	require.Less(t, strings.Index(output, "element-2"), strings.Index(output, "element-3"))
}

// TestGetProcessInstanceListWithElements_HumanOutputShowsRuntimeElements verifies list/search enrichment runs after process-instance limiting.
func TestGetProcessInstanceListWithElements_HumanOutputShowsRuntimeElements(t *testing.T) {
	var requests []string
	var elementFilters []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			require.Equal(t, "ACTIVE", filter["state"])
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":false,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":false,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-07-15T10:13:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":false,"processDefinitionId":"demo-c","processDefinitionKey":"9003","processDefinitionName":"demo-c","processDefinitionVersion":5,"processInstanceKey":"125","startDate":"2026-07-15T10:14:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":3,"hasMoreTotalItems":false}}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			elementFilters = append(elementFilters, filter)
			switch filter["processInstanceKey"] {
			case "123":
				_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-123","elementId":"start","type":"START_EVENT","state":"COMPLETED","startDate":"2026-07-15T10:12:01Z","endDate":"2026-07-15T10:12:02Z","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case "124":
				_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-124","elementId":"task","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:13:01Z","processInstanceKey":"124","processDefinitionKey":"9002","tenantId":"tenant","hasIncident":true,"incidentKey":"incident-124"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected element filter: %#v", filter)
			}
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--state", "active",
		"--limit", "2",
		"--with-elements",
	)

	require.Equal(t, []string{"POST /v2/process-instances/search", "POST /v2/element-instances/search", "POST /v2/element-instances/search"}, requests)
	require.Len(t, elementFilters, 2)
	require.Equal(t, "123", elementFilters[0]["processInstanceKey"])
	require.Equal(t, "124", elementFilters[1]["processInstanceKey"])
	require.Contains(t, output, "123 tenant demo-a v3 ACTIVE")
	require.Contains(t, output, "124 tenant demo-b v4 ACTIVE")
	require.NotContains(t, output, "125 tenant demo-c v5 ACTIVE")
	require.Contains(t, output, "element-123 START_EVENT start COMPLETED")
	require.Contains(t, output, "element-124 SERVICE_TASK task ACTIVE")
	require.Contains(t, output, "inc!:incident-124")
	require.Contains(t, output, "found: 2")
}

// TestGetProcessInstanceListWithElements_BPMNSelectorPreservesProcessFilter verifies element enrichment does not replace process-instance selector validation.
func TestGetProcessInstanceListWithElements_BPMNSelectorPreservesProcessFilter(t *testing.T) {
	var piFilters []map[string]any
	var elementFilters []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-definitions/search":
			require.Equal(t, http.MethodPost, r.Method)
			writeVisibleProcessDefinitionSearchResponse(w)
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			piFilters = append(piFilters, requireJSONObject(t, body["filter"]))
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"order-process","processDefinitionKey":"9001","processDefinitionName":"order-process","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			elementFilters = append(elementFilters, requireJSONObject(t, body["filter"]))
			_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-123","elementId":"ship-order","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:01Z","processInstanceKey":"123","processDefinitionId":"order-process","processDefinitionKey":"9001","tenantId":"tenant-a","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant-a",
		"get", "process-instance",
		"--bpmn-process-id", "order-process",
		"--limit", "5",
		"--with-elements",
	)

	require.Len(t, piFilters, 1)
	require.Equal(t, "order-process", piFilters[0]["processDefinitionId"])
	require.Equal(t, "tenant-a", piFilters[0]["tenantId"])
	require.Len(t, elementFilters, 1)
	require.Equal(t, "123", elementFilters[0]["processInstanceKey"])
	require.Contains(t, output, "123 tenant-a order-process v3 ACTIVE")
	require.Contains(t, output, "element-123 SERVICE_TASK ship-order ACTIVE")
	require.Contains(t, output, "found: 1")
}

// TestGetProcessInstanceListWithElements_IncrementalPagingKeepsProcessInstancePromptCounts proves element rows do not affect prompts or found counts.
func TestGetProcessInstanceListWithElements_IncrementalPagingKeepsProcessInstancePromptCounts(t *testing.T) {
	var prompts []string
	prevConfirm := confirmCmdOrAbortFn
	confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
		prompts = append(prompts, prompt)
		return localPreconditionError(ErrCmdAborted)
	}
	t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":true}}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:01Z","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false},
				{"elementInstanceKey":"element-2","elementId":"task-b","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:02Z","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--batch-size", "1",
		"--with-elements",
	)

	require.Equal(t, []string{"POST /v2/process-instances/search", "POST /v2/element-instances/search"}, requests)
	require.Len(t, prompts, 1)
	require.Contains(t, prompts[0], "Fetched 1 process instance(s) on this page")
	require.Contains(t, output, "element-1 SERVICE_TASK task-a ACTIVE")
	require.Contains(t, output, "element-2 SERVICE_TASK task-b ACTIVE")
	require.Contains(t, output, "found: 1")
	require.NotContains(t, output, "found: 2")
}

// TestGetProcessInstanceListWithElements_JSONOutputPreservesProcessInstanceLimit verifies JSON aggregation keeps element data under selected process instances.
func TestGetProcessInstanceListWithElements_JSONOutputPreservesProcessInstanceLimit(t *testing.T) {
	var elementFilters []map[string]any
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"hasIncident":false,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":false,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-07-15T10:13:00Z","state":"ACTIVE","tenantId":"tenant"},
				{"hasIncident":false,"processDefinitionId":"demo-c","processDefinitionKey":"9003","processDefinitionName":"demo-c","processDefinitionVersion":5,"processInstanceKey":"125","startDate":"2026-07-15T10:14:00Z","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":3,"hasMoreTotalItems":true}}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			var body map[string]any
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			filter := requireJSONObject(t, body["filter"])
			elementFilters = append(elementFilters, filter)
			switch filter["processInstanceKey"] {
			case "123":
				_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-123","elementId":"start","elementName":"Start","type":"START_EVENT","state":"COMPLETED","startDate":"2026-07-15T10:12:01Z","endDate":"2026-07-15T10:12:02Z","processInstanceKey":"123","rootProcessInstanceKey":"123","processDefinitionId":"demo-a","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case "124":
				_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-124","elementId":"task","elementName":"Task","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:13:01Z","processInstanceKey":"124","rootProcessInstanceKey":"124","processDefinitionId":"demo-b","processDefinitionKey":"9002","tenantId":"tenant","hasIncident":true,"incidentKey":"incident-124"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected element filter: %#v", filter)
			}
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--limit", "2",
		"--with-elements",
	)

	require.Len(t, elementFilters, 2)
	require.Equal(t, "123", elementFilters[0]["processInstanceKey"])
	require.Equal(t, "124", elementFilters[1]["processInstanceKey"])
	payload := requireProcessInstanceElementJSONPayload(t, output)
	require.Equal(t, float64(2), payload["total"])
	items := requireJSONItems(t, payload["items"], 2)
	first := requireJSONObject(t, items[0])
	require.Equal(t, "123", requireJSONObject(t, first["item"])["key"])
	firstElements := requireJSONItems(t, first["elements"], 1)
	require.Equal(t, "element-123", requireJSONObject(t, firstElements[0])["elementInstanceKey"])
	second := requireJSONObject(t, items[1])
	require.Equal(t, "124", requireJSONObject(t, second["item"])["key"])
	secondElements := requireJSONItems(t, second["elements"], 1)
	require.Equal(t, "incident-124", requireJSONObject(t, secondElements[0])["incidentKey"])
	require.NotContains(t, output, `"key": "125"`)
}

// TestGetProcessInstanceListWithElements_V87ReportsUnsupported preserves the reused element-service unsupported-version boundary for list/search.
func TestGetProcessInstanceListWithElements_V87ReportsUnsupported(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-07-15T10:12:00Z","tenantId":"tenant"}]}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceListWithElementsUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "element search requires Camunda 8.8 or newer")
}

func TestGetProcessInstanceWithVarsAndIncidents_HumanOutputShowsGroupedSections(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"name":"businessKey","value":"2234809392328","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"},{"name":"hasIncident","value":"true","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--key", "123",
		"--with-vars",
		"--with-incidents",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123", "POST /v2/process-instances/123/incidents/search", "POST /v2/variables/search"}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ vars:")
	require.Contains(t, output, "│  ├─ businessKey=2234809392328")
	require.Contains(t, output, "│  └─ hasIncident=true")
	require.Contains(t, output, "└─ incidents:")
	require.Contains(t, output, "   └─ incident-123 IO_MAPPING_ERROR ACTIVE j:n/a e:task-a ei:element-123 m:No retries left")
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "└─ incidents:"))
}

// TestGetProcessInstanceWithVarsIncidentsAndElements_HumanOutputShowsGroupedSections verifies keyed lookup combines each requested enrichment once.
func TestGetProcessInstanceWithVarsIncidentsAndElements_HumanOutputShowsGroupedSections(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"name":"businessKey","value":"2234809392328","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:01Z","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":true,"incidentKey":"incident-123"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--key", "123",
		"--with-vars",
		"--with-incidents",
		"--with-elements",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"POST /v2/process-instances/123/incidents/search",
		"POST /v2/variables/search",
		"POST /v2/element-instances/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "├─ vars:\n│  └─ businessKey=2234809392328")
	require.Contains(t, output, "├─ incidents:\n│  └─ incident-123 IO_MAPPING_ERROR ACTIVE j:n/a e:task-a ei:element-123 m:No retries left")
	require.Contains(t, output, "└─ elements:\n   └─ element-1 SERVICE_TASK task-a ACTIVE s:2026-07-15T10:12:01.000")
	require.Contains(t, output, "dur:")
	require.Contains(t, output, "inc!:incident-123")
	require.Contains(t, output, "found: 1")
	require.Less(t, strings.Index(output, "├─ vars:"), strings.Index(output, "├─ incidents:"))
	require.Less(t, strings.Index(output, "├─ incidents:"), strings.Index(output, "└─ elements:"))
}

// TestGetProcessInstanceWithElementsAndListeners_HumanOutputNestsListenerRows verifies keyed process-instance enrichment keeps listener rows inside the elements section.
func TestGetProcessInstanceWithElementsAndListeners_HumanOutputNestsListenerRows(t *testing.T) {
	var requests []string
	srv := newProcessInstanceWithElementsAndListenersServer(t, &requests, []string{`{"items":[
		{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:01Z","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false},
		{"elementInstanceKey":"element-2","elementId":"user-task","type":"USER_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:02Z","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}
	],"page":{"totalItems":2,"hasMoreTotalItems":false}}`}, []string{
		`{"items":[{"jobKey":"job-exec-1","kind":"EXECUTION_LISTENER","listenerEventType":"START","type":"audit-start","state":"CREATED","retries":3,"worker":"worker-a","processInstanceKey":"123","elementInstanceKey":"element-1","elementId":"task-a","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		`{"items":[{"jobKey":"job-task-1","kind":"TASK_LISTENER","listenerEventType":"COMPLETING","type":"audit-task","state":"FAILED","retries":0,"processInstanceKey":"123","elementInstanceKey":"element-2","elementId":"user-task","tenantId":"tenant","errorCode":"LISTENER_FAILED","errorMessage":"worker failed"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
	})
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--key", "123",
		"--with-elements",
		"--with-listeners",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"POST /v2/element-instances/search",
		"POST /v2/jobs/search",
		"POST /v2/jobs/search",
	}, requests)
	require.Contains(t, output, "123 tenant demo v3 ACTIVE")
	require.Contains(t, output, "└─ elements:")
	require.Contains(t, output, "element-1 SERVICE_TASK task-a")
	require.Contains(t, output, "ACTIVE")
	require.Contains(t, output, "│  └─ listeners:")
	require.Contains(t, output, "job-exec-1 EXECUTION_LISTENER lsnr:START CREATED tp:audit-start r:3 worker:worker-a")
	require.Contains(t, output, "element-2 USER_TASK")
	require.Contains(t, output, "user-task ACTIVE")
	require.Contains(t, output, "job-task-1 TASK_LISTENER lsnr:COMPLETING FAILED tp:audit-task r:0")
	require.Contains(t, output, "ec:LISTENER_FAILED")
	require.Contains(t, output, "found: 1")
}

// TestGetProcessInstanceWithElementsAndListeners_JSONOutputPreservesEmptyArraysAndOmitsUnmatchedJobs verifies requested listener arrays survive JSON rendering.
func TestGetProcessInstanceWithElementsAndListeners_JSONOutputPreservesEmptyArraysAndOmitsUnmatchedJobs(t *testing.T) {
	var requests []string
	srv := newProcessInstanceWithElementsAndListenersServer(t, &requests, []string{`{"items":[
		{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false},
		{"elementInstanceKey":"element-empty","elementId":"empty-task","type":"SERVICE_TASK","state":"ACTIVE","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}
	],"page":{"totalItems":2,"hasMoreTotalItems":false}}`}, []string{
		`{"items":[{"jobKey":"job-exec-1","kind":"EXECUTION_LISTENER","listenerEventType":"START","type":"audit-start","state":"CREATED","retries":3,"processInstanceKey":"123","elementInstanceKey":"element-1","elementId":"task-a","tenantId":"tenant"},{"jobKey":"job-unmatched","kind":"EXECUTION_LISTENER","listenerEventType":"END","type":"audit-end","state":"CREATED","retries":3,"processInstanceKey":"123","elementInstanceKey":"element-missing","elementId":"missing","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
		`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
	})
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-elements",
		"--with-listeners",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"POST /v2/element-instances/search",
		"POST /v2/jobs/search",
		"POST /v2/jobs/search",
	}, requests)
	payload := requireProcessInstanceElementJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	elements := requireJSONItems(t, first["elements"], 2)
	firstElement := requireJSONObject(t, elements[0])
	firstListeners := requireJSONItems(t, firstElement["listeners"], 1)
	require.Equal(t, "job-exec-1", requireJSONObject(t, firstListeners[0])["jobKey"])
	secondElement := requireJSONObject(t, elements[1])
	require.Empty(t, requireJSONItems(t, secondElement["listeners"], 0))
	require.NotContains(t, output, "job-unmatched")
}

// TestGetProcessInstanceWithElementsWithoutListeners_SkipsListenerLookup keeps the existing element-only path free of job search calls and listener fields.
func TestGetProcessInstanceWithElementsWithoutListeners_SkipsListenerLookup(t *testing.T) {
	var requests []string
	srv := newProcessInstanceWithElementsAndListenersServer(t, &requests, []string{`{"items":[{"elementInstanceKey":"element-1","elementId":"task-a","type":"SERVICE_TASK","state":"ACTIVE","processInstanceKey":"123","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`}, nil)
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-elements",
	)

	require.Equal(t, []string{
		"GET /v2/process-instances/123",
		"POST /v2/element-instances/search",
	}, requests)
	require.NotContains(t, output, `"listeners"`)
}

// TestGetProcessInstanceWithElementsAndListeners_V87ReportsUnsupported verifies unsupported environments fail through the normal command error path.
func TestGetProcessInstanceWithElementsAndListeners_V87ReportsUnsupported(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-07-15T10:12:00Z","tenantId":"tenant"}]}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceListWithElementsAndListenersUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "element search requires Camunda 8.8 or newer")
}

// TestGetProcessInstanceWithVarsIncidentsAndElements_JSONOutputShowsCombinedPayload verifies keyed JSON includes all enrichment fields.
func TestGetProcessInstanceWithVarsIncidentsAndElements_JSONOutputShowsCombinedPayload(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-07-15T10:12:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left","errorType":"IO_MAPPING_ERROR","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"name":"businessKey","value":"2234809392328","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/element-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"elementInstanceKey":"element-1","elementId":"task-a","elementName":"Task A","type":"SERVICE_TASK","state":"ACTIVE","startDate":"2026-07-15T10:12:01Z","processInstanceKey":"123","processDefinitionId":"demo","processDefinitionKey":"9001","tenantId":"tenant","hasIncident":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-vars",
		"--with-incidents",
		"--with-elements",
	)

	payload := requireProcessInstanceElementJSONPayload(t, output)
	require.Equal(t, float64(1), payload["total"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	require.Equal(t, "123", requireJSONObject(t, first["item"])["key"])
	require.Equal(t, "businessKey", requireJSONObject(t, requireJSONItems(t, first["variables"], 1)[0])["name"])
	require.Equal(t, "incident-123", requireJSONObject(t, requireJSONItems(t, first["incidents"], 1)[0])["incidentKey"])
	require.Equal(t, "element-1", requireJSONObject(t, requireJSONItems(t, first["elements"], 1)[0])["elementInstanceKey"])
}

// JSON enrichment keeps variable metadata stable for automation even when compact formatting changes.
func TestGetProcessInstanceWithVars_JSONOutputShowsEnrichedPayloadShapeAndMetadata(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
			_, _ = w.Write([]byte(`{"items":[{"name":"zeta","value":"2","variableKey":"902","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant","isTruncated":false},{"name":"alpha","value":"\"C-123\"","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant","isTruncated":true}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-vars",
	)

	payload := requireProcessInstanceVariableJSONPayload(t, output)
	require.Equal(t, float64(1), payload["total"])
	meta := requireJSONObject(t, payload["meta"])
	require.Equal(t, true, meta["withAge"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	item := requireJSONObject(t, first["item"])
	require.Equal(t, "123", item["key"])
	require.Equal(t, "tenant", item["tenantId"])

	variables := requireJSONItems(t, first["variables"], 2)
	alpha := requireJSONObject(t, variables[0])
	require.Equal(t, "alpha", alpha["name"])
	require.Equal(t, `"C-123"`, alpha["value"])
	require.Equal(t, "901", alpha["variableKey"])
	require.Equal(t, "123", alpha["processInstanceKey"])
	require.Equal(t, "123", alpha["scopeKey"])
	require.Equal(t, "tenant", alpha["tenantId"])
	require.Equal(t, true, alpha["apiTruncated"])

	zeta := requireJSONObject(t, variables[1])
	require.Equal(t, "zeta", zeta["name"])
	require.Equal(t, false, zeta["apiTruncated"])
}

// Human value limits must not alter the received API value in machine-readable output.
func TestGetProcessInstanceWithVars_JSONOutputKeepsReceivedValuesWhenVarValueLimitSet(t *testing.T) {
	fullValue := "abcdefghijklmnopqrstuvwxyz"
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/variables/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"name":"payload","value":"` + fullValue + `","variableKey":"901","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant","isTruncated":false}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-vars",
		"--var-value-limit", "3",
	)

	payload := requireProcessInstanceVariableJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	variables := requireJSONItems(t, first["variables"], 1)
	variable := requireJSONObject(t, variables[0])
	require.Equal(t, fullValue, variable["value"])
	require.NotEqual(t, "abc...", variable["value"])
}

// TestGetProcessInstanceWithIncidents_HumanOutputShowsMultipleAndNoIncidents covers both direct incident rendering and tree-propagated incident warnings.
func TestGetProcessInstanceWithIncidents_HumanOutputShowsMultipleAndNoIncidents(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	tests := []struct {
		name             string
		incidentResponse string
		wantMessages     []string
	}{
		{
			name: "multiple incident lines",
			incidentResponse: `{"items":[
				{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"},
				{"creationTime":"2026-03-23T18:02:00Z","elementId":"task-b","elementInstanceKey":"element-124","errorMessage":"Gateway failed","errorType":"EXTRACT_VALUE_ERROR","incidentKey":"incident-124","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
			wantMessages: []string{
				"└─ incidents:",
				"├─ incident-123 JOB_NO_RETRIES ACTIVE j:n/a 2026-03-23T18:01:00.000 (48 days ago) e:task-a ei:element-123 m:No retries left",
				"└─ incident-124 EXTRACT_VALUE_ERROR ACTIVE j:n/a 2026-03-23T18:02:00.000 (48 days ago) e:task-b ei:element-124 m:Gateway failed",
			},
		},
		{
			name:             "no incident lines",
			incidentResponse: `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`,
			wantMessages:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/process-instances/123":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				case "/v2/process-instances/123/incidents/search":
					require.Equal(t, http.MethodPost, r.Method)
					_, _ = w.Write([]byte(tt.incidentResponse))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"get", "process-instance",
				"--key", "123",
				"--with-incidents",
			)

			require.Contains(t, output, "123")
			require.Contains(t, output, "found: 1")
			for _, msg := range tt.wantMessages {
				require.Contains(t, output, msg)
			}
			if len(tt.wantMessages) == 0 {
				require.NotContains(t, output, "key=incident-")
				require.Contains(t, output, indirectProcessTreeIncidentNote)
				require.Contains(t, output, indirectProcessTreeIncidentWarning)
			}
		})
	}
}

// TestGetProcessInstanceWithIncidents_JSONOutputShowsIncidentDetails preserves the structured incident detail payload.
func TestGetProcessInstanceWithIncidents_JSONOutputShowsIncidentDetails(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"No retries left","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","jobKey":"job-123","processDefinitionId":"demo","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(1), payload["total"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	item := requireJSONObject(t, first["item"])
	require.Equal(t, "123", item["key"])

	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, "incident-123", incident["incidentKey"])
	require.Equal(t, "123", incident["processInstanceKey"])
	require.Equal(t, "No retries left", incident["errorMessage"])
	require.Equal(t, "task-a", incident["elementId"])
	require.Equal(t, "element-123", incident["elementInstanceKey"])
	require.NotContains(t, output, "flowNode")
}

// TestGetProcessInstanceWithIncidents_JSONOutputAssociatesMultipleKeys prevents incident details from crossing keyed lookup boundaries.
func TestGetProcessInstanceWithIncidents_JSONOutputAssociatesMultipleKeys(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/124":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[
				{"errorMessage":"First key failed","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"},
				{"errorMessage":"wrong association","incidentKey":"incident-wrong","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}
			],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"Second key failed","incidentKey":"incident-124","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--key", "124",
		"--workers", "1",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(2), payload["total"])
	items := requireJSONItems(t, payload["items"], 2)

	first := requireJSONObject(t, items[0])
	firstItem := requireJSONObject(t, first["item"])
	require.Equal(t, "123", firstItem["key"])
	firstIncidents := requireJSONItems(t, first["incidents"], 1)
	require.Equal(t, "First key failed", requireJSONObject(t, firstIncidents[0])["errorMessage"])

	second := requireJSONObject(t, items[1])
	secondItem := requireJSONObject(t, second["item"])
	require.Equal(t, "124", secondItem["key"])
	secondIncidents := requireJSONItems(t, second["incidents"], 1)
	require.Equal(t, "Second key failed", requireJSONObject(t, secondIncidents[0])["errorMessage"])
}

// TestGetProcessInstanceWithIncidents_JSONOutputShowsEmptyIncidentCollection keeps empty enrichment explicit for automation.
func TestGetProcessInstanceWithIncidents_JSONOutputShowsEmptyIncidentCollection(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	incidents := requireJSONItems(t, first["incidents"], 0)
	require.Empty(t, incidents)
}

func TestGetProcessInstanceJSONWithIncidents_ListSearchUsesEnrichedPayloadShape(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":true,"processDefinitionId":"demo-a","processDefinitionKey":"9001","processDefinitionName":"demo-a","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":true,"processDefinitionId":"demo-b","processDefinitionKey":"9002","processDefinitionName":"demo-b","processDefinitionVersion":4,"processInstanceKey":"124","startDate":"2026-03-23T18:05:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:01:00Z","elementId":"task-a","elementInstanceKey":"element-123","errorMessage":"First direct incident","errorType":"JOB_NO_RETRIES","incidentKey":"incident-123","jobKey":"job-123","processDefinitionId":"demo-a","processDefinitionKey":"9001","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/124/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"creationTime":"2026-03-23T18:06:00Z","elementId":"task-b","elementInstanceKey":"element-124","errorMessage":"Second direct incident","errorType":"JOB_NO_RETRIES","incidentKey":"incident-124","jobKey":"job-124","processDefinitionId":"demo-b","processDefinitionKey":"9002","processInstanceKey":"124","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--with-incidents",
		"--batch-size", "2",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(2), payload["total"])
	meta := requireJSONObject(t, payload["meta"])
	require.Equal(t, true, meta["withAge"])
	items := requireJSONItems(t, payload["items"], 2)

	first := requireJSONObject(t, items[0])
	firstItem := requireJSONObject(t, first["item"])
	require.Equal(t, "123", firstItem["key"])
	firstIncidents := requireJSONItems(t, first["incidents"], 1)
	firstIncident := requireJSONObject(t, firstIncidents[0])
	require.Equal(t, "incident-123", firstIncident["incidentKey"])
	require.Equal(t, "123", firstIncident["processInstanceKey"])
	require.Equal(t, "First direct incident", firstIncident["errorMessage"])
	require.Equal(t, "task-a", firstIncident["elementId"])

	second := requireJSONObject(t, items[1])
	secondItem := requireJSONObject(t, second["item"])
	require.Equal(t, "124", secondItem["key"])
	secondIncidents := requireJSONItems(t, second["incidents"], 1)
	secondIncident := requireJSONObject(t, secondIncidents[0])
	require.Equal(t, "incident-124", secondIncident["incidentKey"])
	require.Equal(t, "124", secondIncident["processInstanceKey"])
	require.Equal(t, "Second direct incident", secondIncident["errorMessage"])
	require.Equal(t, "task-b", secondIncident["elementId"])
}

func TestGetProcessInstanceJSONWithIncidents_IncidentMessageLimitKeepsFullMessages(t *testing.T) {
	fullMessage := "This long incident message must remain complete in JSON output"
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"` + fullMessage + `","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
		"--with-incidents",
		"--incident-message-limit", "5",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, fullMessage, incident["errorMessage"])
	require.NotEqual(t, "This ...", incident["errorMessage"])
}

func TestGetProcessInstanceJSONWithIncidents_KeyedLookupShapeRemainsUnchanged(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/123":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/123/incidents/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"errorMessage":"No retries left","incidentKey":"incident-123","processInstanceKey":"123","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
		"--with-incidents",
	)

	payload := requireProcessInstanceIncidentJSONPayload(t, output)
	require.Equal(t, float64(1), payload["total"])
	meta := requireJSONObject(t, payload["meta"])
	require.Equal(t, true, meta["withAge"])
	items := requireJSONItems(t, payload["items"], 1)
	first := requireJSONObject(t, items[0])
	require.Contains(t, first, "item")
	require.Contains(t, first, "incidents")
	item := requireJSONObject(t, first["item"])
	require.Equal(t, "123", item["key"])
	incidents := requireJSONItems(t, first["incidents"], 1)
	incident := requireJSONObject(t, incidents[0])
	require.Equal(t, "incident-123", incident["incidentKey"])
	require.Equal(t, "123", incident["processInstanceKey"])
	require.Equal(t, "No retries left", incident["errorMessage"])
}

// TestGetProcessInstanceWithIncidents_V87ReportsUnsupported preserves the tenant-safe version boundary.
func TestGetProcessInstanceWithIncidents_V87ReportsUnsupported(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceWithIncidentsUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	require.Equal(t, exitcode.Error, exitErr.ExitCode())
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "not tenant-safe in Camunda 8.7")
}

func TestGetProcessInstanceWithElements_V87ReportsUnsupported(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, err := testx.RunCmdSubprocess(t, "TestGetProcessInstanceWithElementsUnsupportedV87Helper", map[string]string{
		"C8VOLT_TEST_CONFIG": cfgPath,
	})

	require.Error(t, err)
	require.Contains(t, string(output), "unsupported capability")
	require.Contains(t, string(output), "process-instance direct lookup by key is not tenant-safe in Camunda 8.7")
}

// TestGetProcessInstanceWithoutIncidents_HumanOutputPreservesDefault keeps default keyed output free of enrichment lines.
func TestGetProcessInstanceWithoutIncidents_HumanOutputPreservesDefault(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":true,"parentElementInstanceKey":"ei-parent","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "process-instance",
		"--key", "123",
	)

	wantItem := process.ProcessInstance{
		Key:            "123",
		TenantId:       "tenant",
		BpmnProcessId:  "demo",
		ProcessVersion: 3,
		State:          process.StateActive,
		StartDate:      "2026-03-23T18:00:00Z",
		Incident:       true,
	}
	require.Equal(t, []string{"GET /v2/process-instances/123"}, requests)
	require.Equal(t, strings.TrimSpace(oneLinePI(wantItem))+"\nfound: 1\n", output)
	require.NotContains(t, output, "  inc ")
}

// TestGetProcessInstanceWithoutIncidents_ListSearchPreservesDefaultAndSkipsIncidentLookup keeps list output unchanged unless enrichment is requested.
func TestGetProcessInstanceWithoutIncidents_ListSearchPreservesDefaultAndSkipsIncidentLookup(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/process-instances/search":
			require.Equal(t, http.MethodPost, r.Method)
			_, _ = w.Write([]byte(`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/123/incidents/search":
			t.Fatalf("incident lookup should not run without --with-incidents")
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant",
		"get", "process-instance",
		"--incidents-only",
	)

	wantItem := process.ProcessInstance{
		Key:            "123",
		TenantId:       "tenant",
		BpmnProcessId:  "demo",
		ProcessVersion: 3,
		State:          process.StateActive,
		StartDate:      "2026-03-23T18:00:00Z",
		Incident:       true,
	}
	require.Equal(t, []string{"POST /v2/process-instances/search"}, requests)
	require.Equal(t, strings.TrimSpace(oneLinePI(wantItem))+"\nfound: 1\n", output)
	require.NotContains(t, output, "  inc ")
}

// TestGetProcessInstanceWithoutIncidents_JSONOutputPreservesDefaultShape keeps default JSON free of enrichment wrappers.
func TestGetProcessInstanceWithoutIncidents_JSONOutputPreservesDefaultShape(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/123", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":true,"parentElementInstanceKey":"ei-parent","processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "123",
	)

	require.Equal(t, []string{"GET /v2/process-instances/123"}, requests)

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get process-instance", envelope["command"])
	payload := requireJSONObject(t, envelope["payload"])
	require.NotContains(t, payload, "item")
	require.NotContains(t, payload, "incidents")
	require.Equal(t, float64(1), payload["total"])
	items := requireJSONItems(t, payload["items"], 1)
	item := requireJSONObject(t, items[0])
	require.Equal(t, "123", item["key"])
	require.Equal(t, true, item["incident"])
	require.Equal(t, "ei-parent", item["parentElementInstanceKey"])
	require.NotContains(t, item, "incidents")
	require.NotContains(t, item, "parentFlowNodeInstanceKey")
	require.NotContains(t, output, "parentFlowNodeInstanceKey")
}

// TestGetProcessInstanceSearchIncidentFilters_PreserveDefaultSearchMode keeps incident presence filters on the paged search path.
func TestGetProcessInstanceSearchIncidentFilters_PreserveDefaultSearchMode(t *testing.T) {
	tests := []struct {
		name         string
		flag         string
		wantIncident bool
		response     string
	}{
		{
			name:         "incidents only",
			flag:         "--incidents-only",
			wantIncident: true,
			response:     `{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		},
		{
			name:         "no incidents only",
			flag:         "--no-incidents-only",
			wantIncident: false,
			response:     `{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests, tt.response)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				tt.flag,
			)

			filter := decodeCapturedPISearchFilter(t, requests)
			require.Equal(t, tt.wantIncident, filter["hasIncident"])
			require.NotContains(t, output, `"incidents"`)
			require.Contains(t, output, `"total": 1`)
		})
	}
}

// TestGetProcessInstanceSearch_V87StillSupportsTenantScopedSearch verifies v8.7 search keeps tenant scoping available.
func TestGetProcessInstanceSearch_V87StillSupportsTenantScopedSearch(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.Equal(t, "/v1/process-instances/search", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		requests = append(requests, string(body))

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"<default>"}]}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--state", "active",
	)

	filter := decodeCapturedPISearchFilter(t, requests)
	require.Equal(t, "<default>", filter["tenantId"])
	require.Equal(t, "ACTIVE", filter["state"])
	require.Contains(t, output, `"tenantId": "<default>"`)
}

// TestGetProcessInstanceCommand_VariableFiltersUnsupportedOnV87 verifies native variable filters fail before any 8.7 fallback path.
func TestGetProcessInstanceCommand_VariableFiltersUnsupportedOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	tests := []struct {
		name string
		mode string
	}{
		{name: "var-exists", mode: "var-exists"},
		{name: "var", mode: "var"},
		{name: "var-like", mode: "var-like"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, code := executeProcessInstanceFailureHelperWithEnv(t,
				"TestGetProcessInstanceVariableFiltersUnsupportedV87Helper",
				cfgPath,
				map[string]string{"C8VOLT_TEST_PI_VARIABLE_FILTER_MODE": tt.mode},
			)

			require.Equal(t, exitcode.Error, code)
			require.Contains(t, output, "unsupported capability")
			require.Contains(t, output, "process-instance variable search is unsupported in Camunda 8.7")
			require.Contains(t, output, "requires Camunda 8.8 or 8.9")
		})
	}
}

// TestGetProcessInstanceCommand_V89KeyLookupUsesNativeSearchPath verifies v8.9 direct lookup uses the native single-instance endpoint.
func TestGetProcessInstanceCommand_V89KeyLookupUsesNativeSearchPath(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		require.Equal(t, "/v2/process-instances/2251799813711967", r.URL.Path)
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
	)

	require.Equal(t, []string{"/v2/process-instances/2251799813711967"}, requests)
	require.Contains(t, output, `"key": "2251799813711967"`)
}

// Verifies has-user-tasks resolves through native user-task search, then reuses keyed process-instance rendering.
func TestGetProcessInstanceCommand_HasUserTasksLookupUsesNativeUserTaskAndKeyedProcessInstance(t *testing.T) {
	for _, version := range []string{"8.8", "8.9"} {
		t.Run(version, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/user-tasks/search":
					requireUserTaskSearchRequest(t, r, "2251799815391233", "")
					_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				case "/v2/process-instances/2251799813711967":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, version)

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"get", "pi",
				"--has-user-tasks", "2251799815391233",
			)

			require.Equal(t, []string{
				"/v2/user-tasks/search",
				"/v2/process-instances/2251799813711967",
			}, requests)
			require.Contains(t, output, "2251799813711967")
			require.NotContains(t, output, "2251799815391233")
		})
	}
}

// Verifies has-user-tasks falls back through Tasklist V1 after a native lookup miss and renders the resolved process instance.
func TestGetProcessInstanceCommand_HasUserTasksFallbackUsesTasklistAndKeyedProcessInstance(t *testing.T) {
	for _, version := range []string{"8.8", "8.9"} {
		t.Run(version, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/user-tasks/search":
					requireUserTaskSearchRequest(t, r, "2251799815391233", "")
					_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
				case "/v1/tasks/2251799815391233":
					requireTasklistFallbackTaskRequest(t, r, "2251799815391233")
					_, _ = w.Write([]byte(`{"id":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant","implementation":"JOB_WORKER"}`))
				case "/v2/process-instances/2251799813711967":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, version)

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"get", "pi",
				"--has-user-tasks", "2251799815391233",
			)

			require.Equal(t, []string{
				"/v2/user-tasks/search",
				"/v1/tasks/2251799815391233",
				"/v2/process-instances/2251799813711967",
			}, requests)
			require.Contains(t, output, "2251799813711967")
			require.NotContains(t, output, "2251799815391233")
		})
	}
}

// Verifies has-user-tasks lookup applies the effective tenant while resolving the owning process instance.
func TestGetProcessInstanceCommand_HasUserTasksLookupIncludesEffectiveTenant(t *testing.T) {
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "tenant-a")
			_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--tenant", "tenant-a",
		"get", "pi",
		"--has-user-tasks", "2251799815391233",
	)

	require.Contains(t, output, "2251799813711967")
}

// Verifies repeated has-user-tasks values resolve each task and render the resulting process instances.
func TestGetProcessInstanceCommand_HasUserTasksLookupAcceptsMultipleKeys(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			body := requireUserTaskSearchRequest(t, r, "", "")
			switch body["filter"].(map[string]any)["userTaskKey"] {
			case "2251799815391233":
				_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case "2251799815391244":
				_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391244","processInstanceKey":"2251799813711977","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected user task search body: %v", body)
			}
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/2251799813711977":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711977","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "pi",
		"--has-user-tasks", "2251799815391233",
		"--has-user-tasks", "2251799815391244",
		"--workers", "1",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/user-tasks/search",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711977",
	}, requests)
	require.Contains(t, output, "2251799813711967")
	require.Contains(t, output, "2251799813711977")
}

// Verifies repeated has-user-tasks values resolve each task through the first successful path for that task.
func TestGetProcessInstanceCommand_HasUserTasksLookupMixesPrimaryAndFallbackKeys(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			body := requireUserTaskSearchRequest(t, r, "", "")
			switch body["filter"].(map[string]any)["userTaskKey"] {
			case "2251799815391233":
				_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
			case "2251799815391244":
				_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
			default:
				t.Fatalf("unexpected user task search body: %v", body)
			}
		case "/v1/tasks/2251799815391244":
			requireTasklistFallbackTaskRequest(t, r, "2251799815391244")
			_, _ = w.Write([]byte(`{"id":"2251799815391244","processInstanceKey":"2251799813711977","tenantId":"tenant","implementation":"JOB_WORKER"}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		case "/v2/process-instances/2251799813711977":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711977","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	output := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"get", "pi",
		"--has-user-tasks", "2251799815391233",
		"--has-user-tasks", "2251799815391244",
		"--workers", "1",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/user-tasks/search",
		"/v1/tasks/2251799815391244",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711977",
	}, requests)
	require.Contains(t, output, "2251799813711967")
	require.Contains(t, output, "2251799813711977")
}

// Verifies has-user-tasks JSON output stays identical to direct keyed lookup for the resolved process instance.
func TestGetProcessInstanceCommand_HasUserTasksJSONMatchesDirectKeyedJSON(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "")
			_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	taskKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--has-user-tasks", "2251799815391233",
	)
	directKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711967",
	}, requests)
	require.JSONEq(t, directKeyOutput, taskKeyOutput)
}

// Verifies fallback-resolved JSON output stays identical to direct keyed lookup for the resolved process instance.
func TestGetProcessInstanceCommand_HasUserTasksFallbackJSONMatchesDirectKeyedJSON(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case "/v1/tasks/2251799815391233":
			requireTasklistFallbackTaskRequest(t, r, "2251799815391233")
			_, _ = w.Write([]byte(`{"id":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant","implementation":"JOB_WORKER"}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.9")

	taskKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--has-user-tasks", "2251799815391233",
	)
	directKeyOutput := executeRootForProcessInstanceTest(t,
		"--config", cfgPath,
		"--json",
		"get", "process-instance",
		"--key", "2251799813711967",
	)

	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v1/tasks/2251799815391233",
		"/v2/process-instances/2251799813711967",
		"/v2/process-instances/2251799813711967",
	}, requests)
	require.JSONEq(t, directKeyOutput, taskKeyOutput)
}

// Verifies has-user-tasks lookup preserves render flags that are valid for direct single-instance lookup.
func TestGetProcessInstanceCommand_HasUserTasksPreservesSingleLookupRenderFlags(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "default age",
			args: nil,
			want: "(2 days ago)",
		},
		{
			name: "keys only",
			args: []string{"--keys-only"},
			want: "2251799813711967\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requests []string
			srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests = append(requests, r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/v2/user-tasks/search":
					requireUserTaskSearchRequest(t, r, "2251799815391233", "")
					_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
				case "/v2/process-instances/2251799813711967":
					require.Equal(t, http.MethodGet, r.Method)
					_, _ = w.Write([]byte(`{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}`))
				default:
					t.Fatalf("unexpected request path: %s", r.URL.Path)
				}
			}))
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
			args := append([]string{
				"--config", cfgPath,
				"get", "process-instance",
				"--has-user-tasks", "2251799815391233",
			}, tt.args...)

			output := executeRootForProcessInstanceTest(t, args...)

			require.Equal(t, []string{
				"/v2/user-tasks/search",
				"/v2/process-instances/2251799813711967",
			}, requests)
			require.Contains(t, output, tt.want)
		})
	}
}

// Verifies a missing resolved process instance keeps the not-found behavior of direct keyed lookup.
func TestGetProcessInstanceCommand_HasUserTasksPreservesResolvedProcessInstanceNotFound(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "2251799815391233", "")
			_, _ = w.Write([]byte(`{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
		case "/v2/process-instances/2251799813711967":
			require.Equal(t, http.MethodGet, r.Method)
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceCommand_HasUserTasksResolvedProcessInstanceNotFoundHelper", cfgPath)

	require.Equal(t, exitcode.NotFound, code)
	require.Contains(t, output, "resource not found")
	require.Contains(t, output, "get process instance(s) resolved from user task key(s) [2251799815391233]")
	require.Contains(t, output, "/v2/process-instances/2251799813711967")
	require.Equal(t, []string{
		"/v2/user-tasks/search",
		"/v2/process-instances/2251799813711967",
	}, requests)
}

// Verifies numeric but unknown user-task keys reach native lookup and return not-found, not validation failure.
func TestGetProcessInstanceCommand_HasUserTasksMissingTaskReturnsNotFoundForShortNumericKey(t *testing.T) {
	var requests []string
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v2/user-tasks/search":
			requireUserTaskSearchRequest(t, r, "225179981539123", "")
			_, _ = w.Write([]byte(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`))
		case "/v1/tasks/225179981539123":
			requireTasklistFallbackTaskRequest(t, r, "225179981539123")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		default:
			t.Fatalf("unexpected request path: %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeProcessInstanceFailureHelperWithEnv(t,
		"TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper",
		cfgPath,
		map[string]string{"C8VOLT_TEST_HAS_USER_TASKS_KEY": "225179981539123"},
	)

	require.Equal(t, exitcode.NotFound, code)
	require.Contains(t, output, "resource not found")
	require.Contains(t, output, "fallback user task 225179981539123 was not found or is not visible to the configured tenant")
	require.NotContains(t, output, "invalid input")
	require.Equal(t, []string{"/v2/user-tasks/search", "/v1/tasks/225179981539123"}, requests)
}

// Verifies malformed has-user-tasks values fail validation before any network lookup is attempted.
func TestGetProcessInstanceCommand_HasUserTasksRejectsNonDecimalKeyBeforeLookup(t *testing.T) {
	var requestCount int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	output, code := executeProcessInstanceFailureHelperWithEnv(t,
		"TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper",
		cfgPath,
		map[string]string{"C8VOLT_TEST_HAS_USER_TASKS_KEY": "not-a-key"},
	)

	require.Equal(t, exitcode.InvalidArgs, code)
	require.Contains(t, output, "invalid input")
	require.Contains(t, output, `invalid value for --has-user-tasks: "not-a-key" at index 0 is not a positive decimal user task key`)
	require.Equal(t, int32(0), atomic.LoadInt32(&requestCount))
}

// Verifies has-user-tasks selector conflicts fail before any user-task or process-instance request is made.
func TestGetProcessInstanceCommand_RejectsHasUserTasksConflictsBeforeLookup(t *testing.T) {
	var requestCount int32
	srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

	tests := []struct {
		name string
		mode string
		want string
	}{
		{
			name: "key selector",
			mode: "key",
			want: "--has-user-tasks cannot be combined with --key or stdin key input",
		},
		{
			name: "stdin key selector",
			mode: "stdin",
			want: "--has-user-tasks cannot be combined with --key or stdin key input",
		},
		{
			name: "state filter",
			mode: "state",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "process definition filter",
			mode: "bpmn-process-id",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "date filter",
			mode: "start-date-after",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "derived search filter",
			mode: "roots-only",
			want: "--has-user-tasks cannot be combined with process-instance search filters",
		},
		{
			name: "total mode",
			mode: "total",
			want: "--has-user-tasks cannot be combined with --total",
		},
		{
			name: "limit mode",
			mode: "limit",
			want: "--has-user-tasks cannot be combined with --limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			before := atomic.LoadInt32(&requestCount)
			output, code := executeProcessInstanceFailureHelperWithEnv(t,
				"TestGetProcessInstanceCommand_RejectsHasUserTasksConflictHelper",
				cfgPath,
				map[string]string{"C8VOLT_TEST_HAS_USER_TASKS_CONFLICT": tt.mode},
			)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, tt.want)
			require.Equal(t, before, atomic.LoadInt32(&requestCount))
		})
	}
}

// Verifies Camunda 8.7 reports has-user-tasks as unsupported instead of falling back to another API.
func TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87(t *testing.T) {
	cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.7")

	output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87Helper", cfgPath)

	require.Equal(t, exitcode.Error, code)
	require.Contains(t, output, "unsupported capability")
	require.Contains(t, output, "has-user-tasks lookup is unsupported in Camunda 8.7")
	require.Contains(t, output, "requires Camunda 8.8 or 8.9")
}

// requireUserTaskSearchRequest validates the native user-task search request and returns its decoded body for scenario-specific assertions.
func requireUserTaskSearchRequest(t *testing.T, r *http.Request, taskKey, tenantID string) map[string]any {
	t.Helper()
	require.Equal(t, http.MethodPost, r.Method)
	require.Equal(t, "/v2/user-tasks/search", r.URL.Path)
	raw, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	var body map[string]any
	require.NoError(t, json.Unmarshal(raw, &body))
	filter, ok := body["filter"].(map[string]any)
	require.True(t, ok, "expected user task search filter in %s", string(raw))
	if taskKey != "" {
		require.Equal(t, taskKey, filter["userTaskKey"])
	}
	if tenantID != "" {
		require.Equal(t, tenantID, filter["tenantId"])
	}
	return body
}

// requireTasklistFallbackTaskRequest protects the contract that legacy Tasklist URL ids are looked up directly.
func requireTasklistFallbackTaskRequest(t *testing.T, r *http.Request, taskKey string) {
	t.Helper()
	require.Equal(t, http.MethodGet, r.Method)
	require.Equal(t, "/v1/tasks/"+taskKey, r.URL.Path)
}

// Verifies get process-instance date filters map to expected API query fields and invalid combinations are rejected.
func TestGetProcessInstanceDateFilterScaffold(t *testing.T) {
	t.Run("start date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			require.NotContains(t, filter["startDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--start-date-after", "2026-01-01",
				"--start-date-before", "2026-01-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-01-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-01-31T23:59:59.999999999Z")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("end date command coverage", func(t *testing.T) {
		t.Run("lower bound only", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--end-date-after", "2026-02-01",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")
			require.NotContains(t, filter["endDate"], "$lte")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})

		t.Run("inclusive range composed with state filter", func(t *testing.T) {
			var requests []string
			srv := newProcessInstanceSearchCaptureServer(t, &requests)
			t.Cleanup(srv.Close)

			cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

			output := executeRootForProcessInstanceTest(t,
				"--config", cfgPath,
				"--json",
				"get", "process-instance",
				"--state", "completed",
				"--end-date-after", "2026-02-01",
				"--end-date-before", "2026-03-31",
			)

			filter := decodeCapturedPISearchFilter(t, requests)

			require.Equal(t, "COMPLETED", filter["state"])
			requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-02-01T00:00:00Z")
			requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-03-31T23:59:59.999999999Z")
			requireCapturedPISearchDateExists(t, filter, "endDate")

			var got map[string]any
			require.NoError(t, json.Unmarshal([]byte(output), &got))
			require.NotContains(t, got, "error")
		})
	})

	t.Run("invalid date command coverage", func(t *testing.T) {
		t.Run("invalid start-date format exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidDateFormatHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid value for --start-date-after: "2026-02-30", expected YYYY-MM-DD`)
		})

		t.Run("invalid start-date range exits through shared invalid-input path", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidStartDateRangeHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, `invalid range for --start-date-after and --start-date-before: "2026-02-01" is later than "2026-01-31"`)
		})

		t.Run("date filters are rejected for direct key lookup", func(t *testing.T) {
			cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

			output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceDateFiltersWithKeyHelper", cfgPath)

			require.Equal(t, exitcode.InvalidArgs, code)
			require.Contains(t, output, "invalid input")
			require.Contains(t, output, "date filters are only supported for list/search usage and cannot be combined with --key")
		})
	})
}

// TestGetProcessInstanceRelativeDayFilterScaffold verifies relative-day filters derive stable date bounds for search.
func TestGetProcessInstanceRelativeDayFilterScaffold(t *testing.T) {
	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	t.Run("start-day range search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "7",
			"--start-date-newer-days", "30",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$gte", "2026-03-11T00:00:00Z")
		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-03T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})

	t.Run("end-day upper bound search request uses derived absolute bounds", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--state", "completed",
			"--end-date-newer-days", "14",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		require.Equal(t, "COMPLETED", filter["state"])
		requireCapturedPISearchDateBound(t, filter, "endDate", "$gte", "2026-03-27T00:00:00Z")
		requireCapturedPISearchDateExists(t, filter, "endDate")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})

	t.Run("end-date older-days keeps existing upper bound semantics", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--state", "completed",
			"--end-date-older-days", "45",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		require.Equal(t, "COMPLETED", filter["state"])
		requireCapturedPISearchDateBound(t, filter, "endDate", "$lte", "2026-02-24T23:59:59.999999999Z")
		requireCapturedPISearchDateExists(t, filter, "endDate")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

// TestGetProcessInstanceRelativeDayValidation verifies invalid relative-day ranges and combinations are rejected.
func TestGetProcessInstanceRelativeDayValidation(t *testing.T) {
	t.Run("negative relative-day values exit through shared invalid-input path", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceNegativeRelativeDayHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "invalid value for --start-date-older-days: -2, expected non-negative integer")
	})

	t.Run("mixed absolute and relative start-date filters are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, "start-date absolute and relative day filters cannot be combined")
	})

	t.Run("invalid derived relative-day ranges are rejected", func(t *testing.T) {
		cfgPath := writeTestConfigForVersion(t, "http://127.0.0.1:1", "8.8")

		output, code := executeProcessInstanceFailureHelper(t, "TestGetProcessInstanceInvalidRelativeDayRangeHelper", cfgPath)

		require.Equal(t, exitcode.InvalidArgs, code)
		require.Contains(t, output, "invalid input")
		require.Contains(t, output, `invalid range for --start-date-newer-days and --start-date-older-days: "2026-04-03" is later than "2026-03-11"`)
	})

	t.Run("local-day derivation honors the configured day boundary override", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServer(t, &requests)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTestWithEnv(t,
			[]string{testRelativeDayNowEnv + "=2026-04-10T00:30:00+02:00"},
			"--config", cfgPath,
			"--json",
			"get", "process-instance",
			"--start-date-older-days", "0",
		)

		filter := decodeCapturedPISearchFilter(t, requests)

		requireCapturedPISearchDateBound(t, filter, "startDate", "$lte", "2026-04-10T23:59:59.999999999Z")

		var got map[string]any
		require.NoError(t, json.Unmarshal([]byte(output), &got))
		require.NotContains(t, got, "error")
	})
}

// TestPopulatePISearchFilterOpts_DerivesRelativeDayBounds verifies command options convert relative days to canonical dates.
func TestPopulatePISearchFilterOpts_DerivesRelativeDayBounds(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	prevNow := relativeDayNow
	relativeDayNow = func() time.Time {
		return time.Date(2026, 4, 10, 12, 0, 0, 0, time.UTC)
	}
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	flagGetPIStartAfterDays = 7
	flagGetPIStartBeforeDays = 30
	flagGetPIEndAfterDays = 14
	flagGetPIEndBeforeDays = 1

	filter := populatePISearchFilterOpts()

	require.Equal(t, "2026-03-11", filter.StartDateAfter)
	require.Equal(t, "2026-04-03", filter.StartDateBefore)
	require.Equal(t, "2026-04-09", filter.EndDateAfter)
	require.Equal(t, "2026-03-27", filter.EndDateBefore)
}

// TestPopulatePISearchFilterOpts_TranslatesSupportedPresenceFlags verifies parent and incident flags become facade options.
func TestPopulatePISearchFilterOpts_TranslatesSupportedPresenceFlags(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIChildrenOnly = true
	flagGetPIIncidentsOnly = true

	filter := populatePISearchFilterOpts()

	require.Equal(t, new(true), filter.HasParent)
	require.Equal(t, new(true), filter.HasIncident)
}

// TestValidatePISearchFlags_RejectsMixedAbsoluteAndRelativeInputs verifies absolute and relative date modes are exclusive.
func TestValidatePISearchFlags_RejectsMixedAbsoluteAndRelativeInputs(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartDateAfter = "2026-04-03"
	flagGetPIStartBeforeDays = 7

	err := validatePISearchFlags()

	require.Error(t, err)
	require.Contains(t, err.Error(), "start-date absolute and relative day filters cannot be combined")
}

func TestResetProcessInstanceCommandGlobals_ResetsIncidentMessageLimit(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIIncidentState = "all"
	flagGetPIIncidentErrorType = "JOB_NO_RETRIES"
	flagGetPIIncidentErrorMessage = "failed"
	flagGetPIIncidentMessageLimit = 80
	flagGetPIVarValueLimit = 120
	flagGetPIWithVars = true

	resetProcessInstanceCommandGlobals()

	require.Equal(t, "active", flagGetPIIncidentState)
	require.Empty(t, flagGetPIIncidentErrorType)
	require.Empty(t, flagGetPIIncidentErrorMessage)
	require.Zero(t, flagGetPIIncidentMessageLimit)
	require.Zero(t, flagGetPIVarValueLimit)
	require.False(t, flagGetPIWithVars)
}

func TestValidatePIWithVarsUsage_ListSearchMatchesIncidentMode(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIWithVars = true

	require.NoError(t, validatePIWithVarsUsage(1, false))
	require.NoError(t, validatePIWithVarsUsage(0, false))
	require.NoError(t, validatePIWithVarsUsage(0, true))

	flagGetPIWithIncidents = true
	require.NoError(t, validatePIWithVarsUsage(1, false))
	flagGetPIWithIncidents = false

	err := validatePIWithVarsUsage(1, true)
	require.Error(t, err)
	require.Contains(t, err.Error(), "--with-vars cannot be combined with search-mode filters")
}

// TestHasPISearchFilterFlags_WithRelativeDaysOnly verifies relative-day flags activate search mode.
func TestHasPISearchFilterFlags_WithRelativeDaysOnly(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	flagGetPIStartAfterDays = 72

	require.True(t, hasPISearchFilterFlags())
}

// TestResolvePISearchSize verifies page-size precedence from flags, config, and defaults.
func TestResolvePISearchSize(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)

	cmd := getProcessInstanceCmd
	resetPISearchBatchSizeFlag(t, cmd)

	t.Run("uses shared config default when batch-size flag is unchanged", func(t *testing.T) {
		resetPISearchBatchSizeFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(250), resolvePISearchSize(cmd, cfg))
	})

	t.Run("uses batch-size override when the flag is changed", func(t *testing.T) {
		resetPISearchBatchSizeFlag(t, cmd)
		require.NoError(t, cmd.Flags().Set("batch-size", "125"))
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 250

		require.Equal(t, int32(125), resolvePISearchSize(cmd, cfg))
	})

	t.Run("falls back to repository default for invalid config values", func(t *testing.T) {
		resetProcessInstanceCommandGlobals()
		resetPISearchBatchSizeFlag(t, cmd)
		cfg := &config.Config{}
		cfg.App.ProcessInstancePageSize = 0

		require.Equal(t, int32(consts.MaxPISearchSize), resolvePISearchSize(cmd, cfg))
	})
}

// TestGetProcessInstancePagingFlow verifies interactive, automatic, and limited paging behavior.
func TestGetProcessInstancePagingFlow(t *testing.T) {
	t.Run("limit truncates results across pages and stops without continuation prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"126","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":5,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--state", "active",
			"--batch-size", "2",
			"--limit", "3",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: yes, next step: limit-reached")
		require.Contains(t, output, "detail: stopped after reaching limit of 3 process instance(s)")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
		require.NotContains(t, output, "126")
	})

	t.Run("uses shared config default and prompts before the next page", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prompts := []string{}
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			prompts = append(prompts, prompt)
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 1000, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Len(t, prompts, 1)
		require.Contains(t, prompts[0], "More matching process instances remain")
		require.Contains(t, prompts[0], "Fetched 2 process instance(s) on this page (2/3+ loaded)")
		require.Contains(t, output, "page size: 1000, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 1000, current page: 1, total so far: 3, more matches: no, next step: complete")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("batch-size override and auto-confirm fetch every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
	})

	t.Run("short n controls per-page batch size", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--state", "active",
			"-n", "4",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.EqualValues(t, 4, pages[0]["limit"])
		require.Contains(t, output, "page size: 4, current page: 1, total so far: 1, more matches: no, next step: complete")
		require.Contains(t, output, "123")
	})

	t.Run("batch-size and limit remain independent when limit is smaller", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"126","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":6,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--auto-confirm",
			"get", "process-instance",
			"--state", "active",
			"--batch-size", "4",
			"--limit", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.EqualValues(t, 4, pages[0]["limit"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 4, current page: 2, total so far: 2, more matches: yes, next step: limit-reached")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.NotContains(t, output, "125 tenant demo")
		require.NotContains(t, output, "126 tenant demo")
	})

	t.Run("json mode fetches every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, `"outcome": "succeeded"`)
		require.Contains(t, output, `"total": 3`)
		require.Contains(t, output, `"key": "123"`)
		require.Contains(t, output, `"key": "124"`)
		require.Contains(t, output, `"key": "125"`)
	})

	t.Run("automation mode fetches every page without prompt", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--automation",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.EqualValues(t, 2, pages[0]["limit"])
		require.EqualValues(t, 0, pages[0]["from"])
		require.EqualValues(t, 2, pages[1]["from"])
		require.Zero(t, promptCalls)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: auto-continue")
		require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
		require.Contains(t, output, "125")
	})

	t.Run("automation json mode keeps stdout machine-readable", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"--automation",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.Zero(t, promptCalls)
		require.Contains(t, stdout, `"outcome": "succeeded"`)
		require.Contains(t, stdout, `"total": 3`)
		require.NotContains(t, stdout, "page size:")
		require.Empty(t, stderr)
	})

	t.Run("automation json mode keeps stdout machine-readable even with debug logs", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		promptCalls := 0
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			promptCalls++
			return nil
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--debug",
			"--automation",
			"--json",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 2)
		require.Zero(t, promptCalls)
		require.Contains(t, stdout, `"outcome": "succeeded"`)
		require.Contains(t, stdout, `"total": 3`)
		require.NotContains(t, stdout, "DEBUG")
		require.NotContains(t, stdout, "config loaded")
		require.NotEmpty(t, stderr)
		require.Contains(t, stderr, "DEBUG")
	})

	t.Run("declined continuation reports partial completion summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")
		prevConfirm := confirmCmdOrAbortFn
		confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
			return localPreconditionError(ErrCmdAborted)
		}
		t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: partial-complete")
		require.Contains(t, output, "detail: stopped after 2 processed process instance(s); remaining matches were left untouched")
		require.Contains(t, output, "123")
		require.Contains(t, output, "124")
	})

	t.Run("indeterminate overflow stops with warning summary", func(t *testing.T) {
		var requests []string
		srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
			`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"},{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{}}`,
		)
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.8")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--verbose",
			"get", "process-instance",
			"--batch-size", "2",
		)

		pages := decodeCapturedPISearchPages(t, requests)
		require.Len(t, pages, 1)
		require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: unknown, next step: warning-stop")
		require.Contains(t, output, "warning: stopped after 2 processed process instance(s) because more matching process instances may remain")
	})

	t.Run("v87 fallback keeps final filtered results even when the request stays broad", func(t *testing.T) {
		var requests []string
		srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			require.Equal(t, http.MethodPost, r.Method)
			require.Equal(t, "/v1/process-instances/search", r.URL.Path)

			body, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			requests = append(requests, string(body))

			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"key":123,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant","parentKey":456,"incident":true},{"key":124,"bpmnProcessId":"demo","processVersion":3,"state":"ACTIVE","startDate":"2026-03-23T18:00:00Z","tenantId":"tenant","incident":false}]}`))
		}))
		t.Cleanup(srv.Close)

		cfgPath := writeTestConfigForVersion(t, srv.URL, "8.7")

		output := executeRootForProcessInstanceTest(t,
			"--config", cfgPath,
			"--tenant", "tenant",
			"--json",
			"get", "process-instance",
			"--children-only",
			"--incidents-only",
		)

		filter := decodeCapturedPISearchFilter(t, requests)
		require.NotContains(t, filter, "parentKey")
		require.NotContains(t, filter, "hasIncident")
		require.Contains(t, output, `"total": 1`)
		require.Contains(t, output, `"key": "123"`)
		require.NotContains(t, output, `"key": "124"`)
	})

	t.Run("orphan-child filtering stays on follow-up lookups for supported versions", func(t *testing.T) {
		for _, version := range []string{"8.8", "8.9"} {
			t.Run(version, func(t *testing.T) {
				var searchRequests []string
				var getPaths []string
				call := 0
				srv := newIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					call++
					w.Header().Set("Content-Type", "application/json")
					if call == 1 {
						require.Equal(t, http.MethodPost, r.Method)
						require.Equal(t, "/v2/process-instances/search", r.URL.Path)
						body, err := io.ReadAll(r.Body)
						require.NoError(t, err)
						searchRequests = append(searchRequests, string(body))
						_, _ = w.Write([]byte(`{"items":[{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","parentProcessInstanceKey":"456","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`))
						return
					}
					require.Equal(t, http.MethodGet, r.Method)
					require.Equal(t, "/v2/process-instances/456", r.URL.Path)
					getPaths = append(getPaths, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"message":"not found"}`))
				}))
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, version)

				output := executeRootForProcessInstanceTest(t,
					"--config", cfgPath,
					"--tenant", "tenant",
					"--json",
					"get", "process-instance",
					"--orphan-children-only",
				)

				filters := decodeCapturedPISearchRequests(t, searchRequests)
				require.Len(t, filters, 1)

				topLevelFilter, ok := filters[0]["filter"].(map[string]any)
				require.True(t, ok)
				require.Contains(t, topLevelFilter, "parentProcessInstanceKey")
				require.NotContains(t, topLevelFilter, "processInstanceKey")
				require.Equal(t, []string{"/v2/process-instances/456"}, getPaths)

				require.Contains(t, output, `"total": 1`)
				require.Contains(t, output, `"key": "123"`)
			})
		}
	})

	t.Run("supported filters keep paging summaries aligned with server-filtered pages", func(t *testing.T) {
		for _, version := range []string{"8.8", "8.9"} {
			t.Run(version, func(t *testing.T) {
				var requests []string
				srv := newProcessInstanceSearchCaptureServerWithResponses(t, &requests,
					`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"123","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"456"},{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"124","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"457"}],"page":{"totalItems":3,"hasMoreTotalItems":true}}`,
					`{"items":[{"hasIncident":true,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"125","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant","parentProcessInstanceKey":"458"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`,
				)
				t.Cleanup(srv.Close)

				cfgPath := writeTestConfigForVersion(t, srv.URL, version)
				prompts := []string{}
				prevConfirm := confirmCmdOrAbortFn
				confirmCmdOrAbortFn = func(autoConfirm bool, prompt string) error {
					prompts = append(prompts, prompt)
					return nil
				}
				t.Cleanup(func() { confirmCmdOrAbortFn = prevConfirm })

				output := executeRootForProcessInstanceTest(t,
					"--config", cfgPath,
					"--tenant", "tenant",
					"--verbose",
					"get", "process-instance",
					"--children-only",
					"--incidents-only",
					"--batch-size", "2",
				)

				pages := decodeCapturedPISearchPages(t, requests)
				decoded := decodeCapturedPISearchRequests(t, requests)
				require.Len(t, pages, 2)
				require.Len(t, decoded, 2)
				require.EqualValues(t, 2, pages[0]["limit"])
				require.EqualValues(t, 0, pages[0]["from"])
				require.EqualValues(t, 2, pages[1]["from"])
				filter, ok := decoded[0]["filter"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, filter["hasIncident"])

				parentFilter, ok := filter["parentProcessInstanceKey"].(map[string]any)
				require.True(t, ok)
				require.Equal(t, true, parentFilter["$exists"])

				require.Len(t, prompts, 1)
				require.Contains(t, prompts[0], "Fetched 2 process instance(s) on this page (2 loaded)")
				require.Contains(t, output, "page size: 2, current page: 2, total so far: 2, more matches: yes, next step: prompt")
				require.Contains(t, output, "page size: 2, current page: 1, total so far: 3, more matches: no, next step: complete")
			})
		}
	})
}

// TestPIContinuationProgress protects the translation from backend overflow
// metadata to the prompt/auto-continue/warning states shown in verbose output.
func TestPIContinuationProgress(t *testing.T) {
	t.Run("auto-confirm chooses auto-continue for overflow", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 50},
			OverflowState: process.ProcessInstanceOverflowStateHasMore,
			Items:         []process.ProcessInstance{{Key: "1"}, {Key: "2"}},
		}

		summary := newPIProgressSummary(page, 2, true)

		require.Equal(t, processInstanceContinuationAutoContinue, summary.ContinuationState)
		require.Equal(t, 50, int(summary.PageSize))
		require.Equal(t, 2, summary.CurrentPageCount)
		require.Equal(t, 2, summary.CumulativeCount)
	})

	t.Run("indeterminate overflow stops with warning", func(t *testing.T) {
		page := process.ProcessInstancePage{
			Request:       process.ProcessInstancePageRequest{Size: 25},
			OverflowState: process.ProcessInstanceOverflowStateIndeterminate,
		}

		summary := newPIProgressSummary(page, 0, false)

		require.Equal(t, processInstanceContinuationWarningStop, summary.ContinuationState)
	})
}

// decodeSingleRequestJSON decodes the single captured request body for request-shape assertions.
func decodeSingleRequestJSON(t *testing.T, requests []string) map[string]any {
	t.Helper()

	require.Len(t, requests, 1)

	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(requests[0]), &got))
	return got
}

// requireProcessInstanceIncidentJSONPayload unwraps the shared JSON envelope used by incident-enriched keyed lookups.
func requireProcessInstanceIncidentJSONPayload(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get process-instance", envelope["command"])
	return requireJSONObject(t, envelope["payload"])
}

// requireProcessInstanceVariableJSONPayload unwraps the shared JSON envelope used by variable-enriched keyed lookups.
func requireProcessInstanceVariableJSONPayload(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get process-instance", envelope["command"])
	return requireJSONObject(t, envelope["payload"])
}

// requireProcessInstanceElementJSONPayload unwraps the shared JSON envelope used by element-enriched lookups.
func requireProcessInstanceElementJSONPayload(t *testing.T, output string) map[string]any {
	t.Helper()

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &envelope))
	require.Equal(t, string(OutcomeSucceeded), envelope["outcome"])
	require.Equal(t, "get process-instance", envelope["command"])
	return requireJSONObject(t, envelope["payload"])
}

const (
	tenantAdminKeysSelectedTenant       = "tenant-a"
	tenantAdminKeysReturnedTenant       = "tenant-b"
	tenantAdminKeysProcessInstanceKey   = "2251799813711967"
	tenantAdminKeysProcessDefinitionKey = "2251799813685249"
)

// tenantAdminKeysMismatchProcessInstance returns the shared selected-tenant
// mismatch fixture used by direct-key admin-input command tests.
func tenantAdminKeysMismatchProcessInstance() process.ProcessInstance {
	return process.ProcessInstance{
		Key:                  tenantAdminKeysProcessInstanceKey,
		TenantId:             tenantAdminKeysReturnedTenant,
		BpmnProcessId:        "tenant-b-process",
		ProcessDefinitionKey: tenantAdminKeysProcessDefinitionKey,
		ProcessVersion:       3,
		State:                process.StateActive,
		StartDate:            "2026-03-23T18:00:00Z",
	}
}

// tenantAdminKeysMismatchProcessInstanceJSON mirrors
// tenantAdminKeysMismatchProcessInstance for fake Camunda v2 keyed responses.
func tenantAdminKeysMismatchProcessInstanceJSON() string {
	return `{"hasIncident":false,"processDefinitionId":"tenant-b-process","processDefinitionKey":"9001","processDefinitionName":"tenant-b-process","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-b"}`
}

// executeRootForProcessInstanceTest runs the root command with process-instance globals reset.
func executeRootForProcessInstanceTest(t *testing.T, args ...string) string {
	t.Helper()

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return buf.String()
}

// executeRootForProcessInstanceWithSeparateOutputs runs the root command and returns stdout and stderr independently.
func executeRootForProcessInstanceWithSeparateOutputs(t *testing.T, args ...string) (string, string) {
	t.Helper()

	prevConfirm := confirmCmdOrAbortFn
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm
	t.Cleanup(resetProcessInstanceCommandGlobals)

	root := Root()
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	root.SetOut(stdout)
	root.SetErr(stderr)
	root.SetArgs(args)
	resetCommandTreeFlags(root)
	resetProcessInstanceCommandGlobals()
	confirmCmdOrAbortFn = prevConfirm

	_, err := root.ExecuteC()
	require.NoError(t, err)

	return stdout.String(), stderr.String()
}

// executeRootForProcessInstanceTestWithEnv runs the root command with temporary environment overrides.
func executeRootForProcessInstanceTestWithEnv(t *testing.T, env []string, args ...string) string {
	t.Helper()

	prevNow := relativeDayNow
	t.Cleanup(func() {
		relativeDayNow = prevNow
	})

	for _, kv := range env {
		key, value, ok := strings.Cut(kv, "=")
		require.True(t, ok)
		prevValue, hadValue := os.LookupEnv(key)
		require.NoError(t, os.Setenv(key, value))
		t.Cleanup(func() {
			if hadValue {
				require.NoError(t, os.Setenv(key, prevValue))
				return
			}
			require.NoError(t, os.Unsetenv(key))
		})
	}
	applyRelativeDayNowOverrideFromEnv(t)

	return executeRootForProcessInstanceTest(t, args...)
}

// resetProcessInstanceCommandGlobals restores process-instance command globals between tests.
func resetProcessInstanceCommandGlobals() {
	flagCancelPIKeys = nil
	flagDeletePIKeys = nil
	flagDeletePDKeys = nil
	flagDeletePDBpmnProcessId = ""
	flagDeletePDProcessVersion = 0
	flagDeletePDProcessVersionTag = ""
	flagDeletePDLatest = false
	flagGetPIKeys = nil
	flagGetPIHasUserTasks = nil
	flagRunPIProcessDefinitionBpmnProcessIds = nil
	flagRunPIProcessDefinitionKey = nil
	flagRunPIProcessDefinitionVersion = 0
	flagRunPICount = 1
	flagRunPIVars = ""
	flagResolveIncidentKeys = nil
	flagResolvePIKeys = nil
	flagUpdatePIKeys = nil
	flagUpdatePIVars = ""
	flagUpdatePIVarsFile = ""
	flagGetPIBpmnProcessID = ""
	flagGetPIProcessVersion = 0
	flagGetPIProcessVersionTag = ""
	flagGetPIProcessDefinitionKey = ""
	flagGetPIStartDateAfter = ""
	flagGetPIStartDateBefore = ""
	flagGetPIEndDateAfter = ""
	flagGetPIEndDateBefore = ""
	flagGetPIStartAfterDays = -1
	flagGetPIStartBeforeDays = -1
	flagGetPIEndAfterDays = -1
	flagGetPIEndBeforeDays = -1
	flagGetPITotal = false
	flagGetPIState = "all"
	flagGetPIParentKey = ""
	flagGetPISize = consts.MaxPISearchSize
	flagGetPILimit = 0
	flagGetPIWithIncidents = false
	flagGetPIIncidentState = "active"
	flagGetPIIncidentErrorType = ""
	flagGetPIIncidentErrorMessage = ""
	flagGetPIIncidentMessageLimit = 0
	flagGetPIWithVars = false
	flagGetPIVarValueLimit = 0
	flagGetPIWithElements = false
	flagGetPIWithListeners = false
	flagGetPIVarExists = nil
	flagGetPIVars = nil
	flagGetPIVarLikes = nil
	flagGetPIRootsOnly = false
	flagGetPIChildrenOnly = false
	flagGetPIOrphanChildrenOnly = false
	flagGetPIIncidentsOnly = false
	flagGetPIDirectIncidentsOnly = false
	flagGetPINoIncidentsOnly = false
	flagWalkPIKey = ""
	flagWalkPIModeParent = false
	flagWalkPIModeChildren = false
	flagWalkPIFlat = false
	flagWalkPIWithIncidents = false
	flagWalkPIWithVars = false
	flagWalkPIWithElements = false
	flagWalkPIWithListeners = false
	flagCmdAutoConfirm = false
	flagVerbose = false
	flagViewAsJson = false
	flagViewKeysOnly = false
	flagNoWait = false
	flagForce = false
	flagNoStateCheck = false
	flagDryRun = false
	flagWorkers = 0
	flagNoWorkerLimit = false
	flagFailFast = false
	flagOpsPurgeOrphanReportFile = ""
	flagOpsPurgeOrphanReportFormat = ""
	flagOpsExecuteRetentionPolicyRetentionDays = 0
	flagOpsExecuteRetentionPolicyReportFile = ""
	flagOpsExecuteRetentionPolicyReportFormat = ""
	flagOpsExecuteSmokeTestCount = 1
	flagOpsExecuteSmokeTestNoCleanup = false
	flagOpsExecuteSmokeTestReportFile = ""
	flagOpsExecuteSmokeTestReportFormat = ""
	flagExpectPIKeys = nil
	flagExpectPIStates = nil
	flagExpectPIIncident = ""
	confirmCmdOrAbortFn = confirmCmdOrAbort
}

// resetPISearchBatchSizeFlag restores the process-instance batch-size flag default.
func resetPISearchBatchSizeFlag(t *testing.T, cmd *cobra.Command) {
	t.Helper()

	flag := cmd.Flags().Lookup("batch-size")
	require.NotNil(t, flag)
	require.NoError(t, flag.Value.Set("1000"))
	flag.Changed = false
}

// resetRootPersistentFlags clears root persistent flag globals that can leak across command tests.
func resetRootPersistentFlags(t *testing.T, root *cobra.Command) {
	t.Helper()

	root.PersistentFlags().VisitAll(func(flag *pflag.Flag) {
		require.NoError(t, flag.Value.Set(flag.DefValue))
		flag.Changed = false
	})
}

// executeProcessInstanceFailureHelper runs a helper subprocess expected to fail and returns output with exit code.
func executeProcessInstanceFailureHelper(t *testing.T, helperName string, cfgPath string) (string, int) {
	t.Helper()

	return executeProcessInstanceFailureHelperWithEnv(t, helperName, cfgPath, nil)
}

// executeProcessInstanceFailureHelperWithEnv runs a failing helper subprocess with extra environment for scenario selection.
func executeProcessInstanceFailureHelperWithEnv(t *testing.T, helperName string, cfgPath string, extraEnv map[string]string) (string, int) {
	t.Helper()

	env := map[string]string{
		"C8VOLT_TEST_CONFIG":  cfgPath,
		testRelativeDayNowEnv: cancelDeleteRelativeDayNow,
	}
	for k, v := range extraEnv {
		env[k] = v
	}
	var output []byte
	var err error
	if extraEnv["C8VOLT_TEST_HAS_USER_TASKS_CONFLICT"] == "stdin" {
		output, err = testx.RunCmdSubprocessWithStdin(t, helperName, env, "2251799813711967\n")
	} else {
		output, err = testx.RunCmdSubprocess(t, helperName, env)
	}
	require.Error(t, err)

	exitErr, ok := err.(*exec.ExitError)
	require.True(t, ok)
	return string(output), exitErr.ExitCode()
}

// TestGetProcessInstanceCommand_RejectsHasUserTasksConflictHelper drives conflict cases that must exercise real Execute exit behavior.
func TestGetProcessInstanceCommand_RejectsHasUserTasksConflictHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", "2251799815391233"}
	switch os.Getenv("C8VOLT_TEST_HAS_USER_TASKS_CONFLICT") {
	case "key":
		args = append(args, "--key", "2251799813711967")
	case "stdin":
		args = append(args, "-")
	case "state":
		args = append(args, "--state", "active")
	case "bpmn-process-id":
		args = append(args, "--bpmn-process-id", "C88_SimpleUserTask")
	case "start-date-after":
		args = append(args, "--start-date-after", "2026-01-01")
	case "roots-only":
		args = append(args, "--roots-only")
	case "total":
		args = append(args, "--total")
	case "limit":
		args = append(args, "--limit", "1")
	default:
		t.Fatalf("unknown has-user-tasks conflict mode %q", os.Getenv("C8VOLT_TEST_HAS_USER_TASKS_CONFLICT"))
	}
	os.Args = args

	Execute()
}

// TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87Helper drives the unsupported-version path in a helper process.
func TestGetProcessInstanceCommand_HasUserTasksUnsupportedOnV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", "2251799815391233"}

	Execute()
}

// TestGetProcessInstanceVariableFiltersUnsupportedV87Helper drives unsupported variable-search validation in a helper process.
func TestGetProcessInstanceVariableFiltersUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })

	args := []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance"}
	switch os.Getenv("C8VOLT_TEST_PI_VARIABLE_FILTER_MODE") {
	case "var-exists":
		args = append(args, "--var-exists", "customerId")
	case "var":
		args = append(args, "--var", `status="approved"`)
	case "var-like":
		args = append(args, "--var-like", "email=*@example.com")
	default:
		t.Fatalf("unknown variable filter mode %q", os.Getenv("C8VOLT_TEST_PI_VARIABLE_FILTER_MODE"))
	}
	os.Args = args

	Execute()
}

// TestGetProcessInstanceCommand_HasUserTasksResolvedProcessInstanceNotFoundHelper preserves process exit behavior for resolved-key not-found.
func TestGetProcessInstanceCommand_HasUserTasksResolvedProcessInstanceNotFoundHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", "2251799815391233"}

	Execute()
}

// TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper drives invalid and missing task-key lookups in a helper process.
func TestGetProcessInstanceCommand_HasUserTasksLookupFailureHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	taskKey := os.Getenv("C8VOLT_TEST_HAS_USER_TASKS_KEY")
	if taskKey == "" {
		taskKey = "2251799815391233"
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--has-user-tasks", taskKey}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper is the helper-process entrypoint for removed --count validation.
func TestGetProcessInstanceCommand_RejectsRemovedCountFlagHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--count", "2"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsInvalidLimitHelper is the helper-process entrypoint for invalid --limit validation.
func TestGetProcessInstanceCommand_RejectsInvalidLimitHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--limit", "0"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper is the helper-process entrypoint for --limit with --key validation.
func TestGetProcessInstanceCommand_RejectsLimitWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--limit", "1"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper is the helper-process entrypoint for --limit with --total validation.
func TestGetProcessInstanceCommand_RejectsLimitWithTotalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--total", "--limit", "10"}

	Execute()
}

// TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper is the helper-process entrypoint for invalid --batch-size validation.
func TestGetProcessInstanceCommand_RejectsInvalidBatchSizeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--batch-size", "0"}

	Execute()
}

// Helper-process entrypoint for negative relative-day validation.
func TestGetProcessInstanceNegativeRelativeDayHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "-2"}

	Execute()
}

// Helper-process entrypoint for mixed absolute-plus-relative start-date validation.
func TestGetProcessInstanceMixedAbsoluteAndRelativeDateFiltersHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-04-03", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for invalid relative-day range validation.
func TestGetProcessInstanceInvalidRelativeDayRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-older-days", "30", "--start-date-newer-days", "7"}

	Execute()
}

// Helper-process entrypoint for --total with --key validation.
func TestGetProcessInstanceTotalWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--total"}

	Execute()
}

// Helper-process entrypoint for --total with --json validation.
func TestGetProcessInstanceTotalWithJSONHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--json", "--total"}

	Execute()
}

// Helper-process entrypoint for --total with --keys-only validation.
func TestGetProcessInstanceTotalWithKeysOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--keys-only", "--total"}

	Execute()
}

// Helper-process entrypoint for --with-incidents without --key validation.
func TestGetProcessInstanceWithIncidentsWithoutKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--with-incidents"}

	Execute()
}

// Helper-process entrypoint for --with-incidents with search-mode filter validation.
func TestGetProcessInstanceWithIncidentsWithSearchFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents", "--incidents-only"}

	Execute()
}

// Helper-process entrypoint for --with-incidents with --total validation.
func TestGetProcessInstanceWithIncidentsWithTotalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--with-incidents", "--total"}

	Execute()
}

// Helper-process entrypoint for --with-elements with --total validation.
func TestGetProcessInstanceWithElementsWithTotalHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--with-elements", "--total"}

	Execute()
}

// Helper-process entrypoint for --with-elements with --keys-only validation.
func TestGetProcessInstanceWithElementsWithKeysOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--keys-only", "--with-elements"}

	Execute()
}

// Helper-process entrypoint for --with-elements with search-mode filter validation.
func TestGetProcessInstanceWithElementsWithSearchFilterHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-elements", "--incidents-only"}

	Execute()
}

// TestGetProcessInstanceWithListenersWithoutElementsHelper is the helper-process entrypoint for missing element context validation.
func TestGetProcessInstanceWithListenersWithoutElementsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-listeners"}

	Execute()
}

// TestGetProcessInstanceWithListenersWithKeysOnlyHelper is the helper-process entrypoint for keys-only listener validation.
func TestGetProcessInstanceWithListenersWithKeysOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--keys-only", "--key", "123", "--with-elements", "--with-listeners"}

	Execute()
}

// TestGetProcessInstanceListWithElementsAndListenersUnsupportedV87Helper drives unsupported v8.7 listener coverage through list/search mode.
func TestGetProcessInstanceListWithElementsAndListenersUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--with-elements", "--with-listeners"}

	Execute()
}

// Helper-process entrypoint for mutually exclusive direct and marker incident filter validation.
func TestGetProcessInstanceDirectIncidentsOnlyWithIncidentsOnlyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--direct-incidents-only", "--incidents-only"}

	Execute()
}

// Helper-process entrypoint for --incident-message-limit without --with-incidents validation.
func TestGetProcessInstanceIncidentMessageLimitWithoutIncidentsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--incident-message-limit", "80"}

	Execute()
}

// Helper-process entrypoint for negative --incident-message-limit validation.
func TestGetProcessInstanceIncidentMessageLimitNegativeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents", "--incident-message-limit", "-1"}

	Execute()
}

// Helper-process entrypoint for --incident-state without --with-incidents validation.
func TestGetProcessInstanceIncidentStateWithoutIncidentsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--incident-state", "all"}

	Execute()
}

// Helper-process entrypoint for --incident-state in list/search process-instance mode.
func TestGetProcessInstanceIncidentStateListSearchHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--with-incidents", "--incident-state", "resolved"}

	Execute()
}

// Helper-process entrypoint for unsupported --incident-state validation.
func TestGetProcessInstanceIncidentStateInvalidHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents", "--incident-state", "closed"}

	Execute()
}

// Helper-process entrypoint for --incident-error-type without --with-incidents validation.
func TestGetProcessInstanceIncidentErrorTypeWithoutIncidentsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--incident-error-type", "job_no_retries"}

	Execute()
}

// Helper-process entrypoint for --incident-error-message without --with-incidents validation.
func TestGetProcessInstanceIncidentErrorMessageWithoutIncidentsHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--incident-error-message", "failed"}

	Execute()
}

// Helper-process entrypoint for unsupported --incident-error-type validation.
func TestGetProcessInstanceIncidentErrorTypeInvalidHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents", "--incident-error-type", "retry_error"}

	Execute()
}

// Helper-process entrypoint for unsupported v8.7 --with-incidents coverage.
func TestGetProcessInstanceWithIncidentsUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-incidents"}

	Execute()
}

// Helper-process entrypoint for unsupported v8.7 --with-elements coverage.
func TestGetProcessInstanceWithElementsUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "123", "--with-elements"}

	Execute()
}

// Helper-process entrypoint for unsupported v8.7 list/search --with-elements coverage.
func TestGetProcessInstanceListWithElementsUnsupportedV87Helper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--state", "active", "--with-elements"}

	Execute()
}

func TestGetProcessInstanceBpmnSelectorMissingFailsBeforeSearchHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant-a", "get", "process-instance", "--bpmn-process-id", "missing-process"}

	Execute()
}

func TestGetProcessInstanceBpmnSelectorMissingKeysOnlyPipelineFailsUpstreamHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "--tenant", "tenant-a", "get", "process-instance", "--bpmn-process-id", "missing-process", "--keys-only"}

	Execute()
}

// Helper-process entrypoint for invalid date format validation.
func TestGetProcessInstanceInvalidDateFormatHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-30"}

	Execute()
}

// Helper-process entrypoint for invalid start-date range validation.
func TestGetProcessInstanceInvalidStartDateRangeHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--start-date-after", "2026-02-01", "--start-date-before", "2026-01-31"}

	Execute()
}

// Helper-process entrypoint for key-and-date-filter exclusivity validation.
func TestGetProcessInstanceDateFiltersWithKeyHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	applyRelativeDayNowOverrideFromEnv(t)

	prevArgs := os.Args
	t.Cleanup(func() { os.Args = prevArgs })
	os.Args = []string{"c8volt", "--config", os.Getenv("C8VOLT_TEST_CONFIG"), "get", "process-instance", "--key", "2251799813711967", "--start-date-after", "2026-01-01"}

	Execute()
}
