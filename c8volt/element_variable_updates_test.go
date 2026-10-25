// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package c8volt

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
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

// TestElementVariableUpdatesExecuteWiring exercises all generated adapters through
// the root facade: exact-local writes, complete sparse confirmation, no rediscovery,
// and no-wait with zero post-submit reads.
func TestElementVariableUpdatesExecuteWiring(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, noWait := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/no-wait=%t", version, noWait), func(t *testing.T) {
				cfg := config.New()
				cfg.App.CamundaVersion = version
				cfg.App.Tenant = "configured"
				cfg.APIs.Camunda.BaseURL = "http://fixture.invalid/v2"
				executing := false
				writes, reads, gets := 0, 0, 0
				pages := map[string]int{}
				cli, err := New(WithConfig(cfg), WithHTTPClient(&http.Client{Transport: clientTestRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					payload := ""
					code := 200
					switch {
					case r.Method == http.MethodGet:
						require.False(t, executing, "execution must not rediscover targets")
						gets++
						key := strings.TrimPrefix(r.URL.Path, "/v2/element-instances/")
						payload = fmt.Sprintf(`{"elementInstanceKey":%q,"processInstanceKey":"100","tenantId":"actual","type":"CALL_ACTIVITY","state":"ACTIVE"}`, key)
					case strings.HasSuffix(r.URL.Path, "/variables"):
						require.True(t, executing)
						writes++
						require.Equal(t, "/v2/element-instances/"+[]string{"200", "300"}[writes-1]+"/variables", r.URL.Path)
						var body struct {
							Local     bool           `json:"local"`
							Variables map[string]any `json:"variables"`
						}
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.True(t, body.Local)
						require.Equal(t, map[string]any{"add": true}, body.Variables)
						code = 204
					case r.URL.Path == "/v2/variables/search":
						var body struct {
							Filter map[string]any `json:"filter"`
							Page   map[string]any `json:"page"`
						}
						require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
						require.Equal(t, "100", body.Filter["processInstanceKey"])
						require.NotContains(t, body.Filter, "tenantId")
						require.Equal(t, "false", r.URL.Query().Get("truncateValues"))
						scope := body.Filter["scopeKey"].(string)
						if !executing {
							payload = fmt.Sprintf(`{"items":[{"name":"same","value":"true","processInstanceKey":"100","scopeKey":%q,"tenantId":"actual"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`, scope)
						} else {
							reads++
							pages[scope]++
							if pages[scope] == 1 {
								payload = `{"items":[{"name":"add","value":"true","processInstanceKey":"100","scopeKey":"100"}],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"next"}}`
							} else {
								require.Equal(t, 2, pages[scope])
								require.Contains(t, body.Page, "after")
								payload = fmt.Sprintf(`{"items":[{"name":"same","value":"true","processInstanceKey":"100","scopeKey":%q,"tenantId":"actual"},{"name":"add","value":"true","processInstanceKey":"100","scopeKey":%q,"tenantId":"actual"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`, scope, scope)
							}
						}
					default:
						t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					return &http.Response{StatusCode: code, Status: fmt.Sprintf("%d %s", code, http.StatusText(code)), Request: r, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(payload))}, nil
				})}))
				require.NoError(t, err)
				plan, err := cli.PlanElementVariableUpdates(context.Background(), typex.Keys{"200", "300", "200"}, map[string]any{"add": true, "same": true})
				require.NoError(t, err)
				executing = true
				opts := []foptions.FacadeOption{}
				if noWait {
					opts = append(opts, foptions.WithNoWait())
				}
				results, err := cli.ExecuteElementVariableUpdates(context.Background(), plan, 1, opts...)
				require.NoError(t, err)
				require.Equal(t, 2, gets)
				require.Equal(t, 2, writes)
				require.Len(t, results.Items, 2)
				if noWait {
					require.Zero(t, reads)
				} else {
					require.Equal(t, 4, reads)
				}
				for i, item := range results.Items {
					require.Equal(t, []string{"200", "300"}[i], item.Key)
					require.True(t, item.MutationAccepted)
					require.Equal(t, element.VariableSubmissionAccepted, item.SubmissionStatus)
					require.Equal(t, 204, item.StatusCode)
					require.True(t, item.OK())
				}
			})
		}
	}
}

// TestElementVariableUpdatesPacing uses the actual shared burst-one limiter over
// an in-memory transport. Admission failures in planning, mutation and confirmation
// preserve failure classes and never invent dispatch or erase accepted writes.
func TestElementVariableUpdatesPacing(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, phase := range []string{"planning", "mutation", "confirmation"} {
			t.Run(fmt.Sprintf("%s/%s", version, phase), func(t *testing.T) {
				cfg := config.New()
				cfg.App.CamundaVersion = version
				cfg.APIs.Camunda.BaseURL = "http://fixture.invalid/v2"
				cfg.HTTP.Timeout = "100ms"
				cfg.HTTP.RequestRate = 0.01
				cfg.HTTP.RequestRateReportInterval = "5s"
				var dispatched atomic.Int32
				base := clientTestRoundTripFunc(func(r *http.Request) (*http.Response, error) {
					dispatched.Add(1)
					code := 200
					payload := ""
					switch {
					case r.URL.Path == "/prime":
						code = 204
					case r.Method == http.MethodGet:
						payload = `{"elementInstanceKey":"200","processInstanceKey":"100","tenantId":"actual","type":"SERVICE_TASK","state":"ACTIVE"}`
					case r.URL.Path == "/v2/variables/search":
						payload = `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
					case r.URL.Path == "/v2/element-instances/200/variables":
						code = 204
					default:
						t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					}
					return &http.Response{StatusCode: code, Status: fmt.Sprintf("%d %s", code, http.StatusText(code)), Request: r, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(payload))}, nil
				})
				log := slog.New(slog.NewTextHandler(io.Discard, nil))
				plain, err := New(WithConfig(cfg), WithHTTPClient(&http.Client{Transport: base}), WithLogger(log))
				require.NoError(t, err)
				plan, err := plain.PlanElementVariableUpdates(context.Background(), typex.Keys{"200"}, map[string]any{"x": true})
				require.NoError(t, err)
				dispatched.Store(0)
				shared, err := httpc.New(cfg, log, func(s *httpc.Service) { s.Client().Transport = base })
				require.NoError(t, err)
				defer shared.Close()
				cli, err := New(WithConfig(cfg), WithHTTPClient(shared.Client()), WithLogger(log))
				require.NoError(t, err)
				ctx := shared.ToContext(context.Background())
				if phase == "planning" {
					p, err := cli.PlanElementVariableUpdates(ctx, typex.Keys{"200"}, map[string]any{"x": true})
					require.ErrorIs(t, err, ferrors.ErrTimeout)
					require.True(t, httpc.IsRequestRateWaitError(err))
					require.Empty(t, p.Elements)
					require.Equal(t, int32(1), dispatched.Load())
					return
				}
				if phase == "mutation" {
					req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://fixture.invalid/prime", nil)
					require.NoError(t, err)
					resp, err := shared.Client().Do(req)
					require.NoError(t, err)
					require.NoError(t, resp.Body.Close())
				}
				results, err := cli.ExecuteElementVariableUpdates(ctx, plan, 1)
				require.ErrorIs(t, err, ferrors.ErrTimeout)
				require.True(t, httpc.IsRequestRateWaitError(err))
				require.Equal(t, int32(1), dispatched.Load())
				require.NoError(t, ctx.Err())
				require.Len(t, results.Items, 1)
				item := results.Items[0]
				require.False(t, item.OK())
				if phase == "mutation" {
					require.Equal(t, element.VariableSubmissionNotSent, item.SubmissionStatus)
					require.False(t, item.MutationAccepted)
					require.Zero(t, item.StatusCode)
					require.Equal(t, element.ElementVariableUpdateStatusMutationFailed, item.Status)
					require.Equal(t, "not_required", item.ConfirmationStatus)
				} else {
					require.Equal(t, element.VariableSubmissionAccepted, item.SubmissionStatus)
					require.True(t, item.MutationAccepted)
					require.Equal(t, 204, item.StatusCode)
					require.Equal(t, element.ElementVariableUpdateStatusConfirmationFailed, item.Status)
					require.Equal(t, "failed", item.ConfirmationStatus)
				}
			})
		}
	}
}
