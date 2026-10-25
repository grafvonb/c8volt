// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package c8volt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/typex"
	"github.com/stretchr/testify/require"
)

// TestElementVariableUpdatesPlanWiring exercises the root facade and all three
// real adapters over an isolated transport, including sparse local reads and a
// later-page failure. Any discovery fallback, tenant filter or write fails the test.
func TestElementVariableUpdatesPlanWiring(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, failLast := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/fail=%t", version, failLast), func(t *testing.T) {
				cfg := config.New()
				cfg.App.CamundaVersion = version
				cfg.App.Tenant = "configured"
				cfg.APIs.Camunda.BaseURL = "http://test.invalid/v2"
				calls := []string{}
				pages := map[string]int{}
				cli, err := New(WithConfig(cfg), WithHTTPClient(&http.Client{Transport: clientTestRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					calls = append(calls, r.Method+" "+r.URL.Path)
					payload := ""
					status := http.StatusOK
					switch {
					case r.Method == http.MethodGet && (r.URL.Path == "/v2/element-instances/200" || r.URL.Path == "/v2/element-instances/300"):
						key := strings.TrimPrefix(r.URL.Path, "/v2/element-instances/")
						payload = fmt.Sprintf(`{"elementInstanceKey":%q,"processInstanceKey":"100","tenantId":"actual","type":"SERVICE_TASK","state":"ACTIVE"}`, key)
					case r.Method == http.MethodPost && r.URL.Path == "/v2/variables/search":
						require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
						var body struct {
							Filter map[string]any `json:"filter"`
							Page   map[string]any `json:"page"`
						}
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.NotContains(t, body.Filter, "tenantId")
						require.Equal(t, "100", body.Filter["processInstanceKey"])
						scope, ok := body.Filter["scopeKey"].(string)
						require.True(t, ok)
						require.Contains(t, []string{"200", "300"}, scope)
						pages[scope]++
						if pages[scope] == 1 {
							payload = `{"items":[{"name":"value","value":"true","processInstanceKey":"100","scopeKey":"100"}],"page":{"totalItems":2,"hasMoreTotalItems":true,"endCursor":"next"}}`
						} else {
							require.Equal(t, 2, pages[scope])
							require.Contains(t, body.Page, "after")
							payload = fmt.Sprintf(`{"items":[{"name":"untouched","value":"false","processInstanceKey":"100","scopeKey":%q,"tenantId":"actual"}],"page":{"totalItems":2,"hasMoreTotalItems":false}}`, scope)
							if failLast && scope == "300" {
								status = http.StatusForbidden
								payload = `{"title":"Forbidden","status":403}`
							}
						}
					default:
						t.Fatalf("unexpected planning request: %s %s", r.Method, r.URL.Path)
					}
					return &http.Response{StatusCode: status, Status: fmt.Sprintf("%d %s", status, http.StatusText(status)), Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(payload)), Request: r}, nil
				})}))
				require.NoError(t, err)
				plan, err := cli.PlanElementVariableUpdates(context.Background(), typex.Keys{"200", "300", "200"}, map[string]any{"value": true})
				require.Equal(t, []string{"GET /v2/element-instances/200", "POST /v2/variables/search", "POST /v2/variables/search", "GET /v2/element-instances/300", "POST /v2/variables/search", "POST /v2/variables/search"}, calls)
				if failLast {
					require.Error(t, err)
					require.Equal(t, element.ElementVariableUpdatePlan{}, plan)
					return
				}
				require.NoError(t, err)
				require.Equal(t, 2, plan.UpdateCount)
				require.Equal(t, 2, plan.VariableAddCount)
				require.Equal(t, 2, plan.VariableUntouchedCount)
				require.Equal(t, []string{"200", "300"}, plan.RequestedKeys)
				for _, p := range plan.Elements {
					require.Equal(t, p.ElementKey, p.ScopeKey)
					require.Equal(t, "100", p.ProcessInstanceKey)
					require.Equal(t, map[string]any{"value": true}, p.MutationVariables)
				}
			})
		}
	}
}
