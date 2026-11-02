// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"encoding/json"
	"io"
	"strings"
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

// TestGetProcessInstanceSearchDistinctPaging checks actual query execution across duplicate and sparse pages without changing output modes.
func TestGetProcessInstanceSearchDistinctPaging(t *testing.T) {
	for _, mode := range []string{"human", "json", "keys", "quiet", "quiet-json", "quiet-keys", "total"} {
		t.Run(mode, func(t *testing.T) {
			var requests []string
			// Lower-bound totals force counted traversal in total mode; the sparse page has a fresh cursor.
			srv := newProcessInstanceSearchCaptureServerForVersion(t, "8.10", &requests,
				`{"items":[{"processInstanceKey":"101","state":"ACTIVE"},{"processInstanceKey":"101","state":"ACTIVE"}],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"a"}}`,
				`{"items":[],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"b"}}`,
				`{"items":[{"processInstanceKey":"101","state":"ACTIVE"},{"processInstanceKey":"102","state":"ACTIVE"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`,
			)
			t.Cleanup(srv.Close)
			cfg := writeTestConfigForVersion(t, srv.URL, "8.10")
			args := []string{"--config", cfg, "get", "pi", "--state", "active", "--batch-size", "2", "--auto-confirm"}
			if mode == "total" {
				args = append(args, "--total")
			} else {
				args = append(args, "--limit", "2")
			}
			switch mode {
			case "json":
				args = append(args, "--json")
			case "keys":
				args = append(args, "--keys-only")
			case "quiet":
				args = append(args, "--quiet")
			case "quiet-json":
				args = append(args, "--quiet", "--json")
			case "quiet-keys":
				args = append(args, "--quiet", "--keys-only")
			}
			stdout, stderr := executeRootForProcessInstanceWithSeparateOutputs(t, args...)
			require.Len(t, requests, 3)
			require.NotContains(t, stderr, "Continue?")
			switch mode {
			case "keys", "quiet-keys":
				require.Equal(t, "101\n102\n", stdout)
			case "quiet":
				// Existing query quiet mode retains requested rows; it only suppresses diagnostics.
				require.Contains(t, stdout, "found: 2")
			case "total":
				require.Equal(t, "2\n", stdout)
			case "json", "quiet-json":
				decoder := json.NewDecoder(strings.NewReader(stdout))
				var envelope struct {
					Outcome string `json:"outcome"`
					Payload struct {
						Total int `json:"total"`
						Items []struct {
							Key string `json:"key"`
						} `json:"items"`
					} `json:"payload"`
				}
				require.NoError(t, decoder.Decode(&envelope))
				require.Equal(t, "succeeded", envelope.Outcome)
				require.Equal(t, 2, envelope.Payload.Total)
				require.Len(t, envelope.Payload.Items, 2)
				require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
			case "human":
				require.Contains(t, stdout, "found: 2")
			}
		})
	}
}

// TestGetProcessInstanceSearchIncompleteLogging verifies cursor failures use configured diagnostics, one envelope and the original exit.
func TestGetProcessInstanceSearchIncompleteLogging(t *testing.T) {
	for _, format := range []string{"plain-time", "json"} {
		for _, mode := range []string{"normal", "verbose", "quiet", "json", "quiet-json"} {
			for _, destination := range []string{"configured", "inherited"} {
				t.Run(format+"/"+mode+"/"+destination, func(t *testing.T) {
					var requests []string
					response := `{"items":[],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"private-repeated-cursor"}}`
					srv := newProcessInstanceSearchCaptureServerForVersion(t, "8.10", &requests, response, response)
					t.Cleanup(srv.Close)
					cfg := runBusinessIDConfig(t, srv.URL, format, "info")
					args := []string{"--config", cfg, "--no-indicator", "get", "pi", "--auto-confirm", "--batch-size", "2"}
					switch mode {
					case "verbose":
						args = append(args, "--verbose")
					case "quiet":
						args = append(args, "--quiet")
					case "json":
						args = append(args, "--json")
					case "quiet-json":
						args = append(args, "--quiet", "--json")
					}
					stdout, stderr, err := runBusinessIDCommand(t, args, destination, false)
					assertRunBusinessIDExit(t, err, 1)
					require.Len(t, requests, 2)
					require.NotContains(t, stdout+stderr, "private-repeated-cursor")
					require.NotContains(t, stdout+stderr, "found: 0")
					require.NotContains(t, stdout+stderr, "Continue?")
					if strings.Contains(mode, "json") {
						decoder := json.NewDecoder(strings.NewReader(stdout))
						var envelope struct {
							Outcome string `json:"outcome"`
							Class   string `json:"class"`
						}
						require.NoError(t, decoder.Decode(&envelope))
						require.Equal(t, "failed", envelope.Outcome)
						require.Equal(t, "malformed_response", envelope.Class)
						require.ErrorIs(t, decoder.Decode(new(any)), io.EOF)
						require.Empty(t, stderr)
					} else {
						require.Empty(t, stdout)
						require.Equal(t, 1, strings.Count(stderr, "invalid process-instance search metadata"))
						if format == "json" {
							decoder := json.NewDecoder(strings.NewReader(stderr))
							errors := 0
							for {
								var record struct {
									Level string `json:"level"`
									Msg   string `json:"msg"`
								}
								err := decoder.Decode(&record)
								if err == io.EOF {
									break
								}
								require.NoError(t, err)
								if record.Level == "ERROR" {
									errors++
									require.Contains(t, record.Msg, "cursor did not advance")
								}
							}
							require.Equal(t, 1, errors)
						} else {
							require.Equal(t, 1, strings.Count(stderr, " ERROR "))
						}
					}
				})
			}
		}
	}
}
