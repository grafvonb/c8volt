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
	"testing"

	"github.com/grafvonb/c8volt/c8volt/element"
	"github.com/grafvonb/c8volt/c8volt/ferrors"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// TestElementScopeVariablesSharedClient proves real facade/service/adapter composition
// uses one injected transport, complete local reads and bounded element selection.
func TestElementScopeVariablesSharedClient(t *testing.T) {
	t.Parallel()
	for _, version := range []toolx.CamundaVersion{toolx.V88, toolx.V89, toolx.V810} {
		for _, mode := range []string{"get", "collected", "pages", "stop", "limit", "later failure"} {
			t.Run(string(version)+"/"+mode, func(t *testing.T) {
				gets, searches, jobs, visits := 0, 0, 0, 0
				reads := map[string]int{}
				observedTenant := "configured"
				if mode == "get" {
					observedTenant = "other"
				}
				cfg := config.New()
				cfg.App.CamundaVersion = version
				cfg.App.Tenant = "configured"
				cfg.APIs.Camunda.BaseURL = "http://fixture.invalid/v2"
				transport := clientTestRoundTripFunc(func(req *http.Request) (*http.Response, error) {
					body := ""
					switch req.URL.Path {
					case "/v2/element-instances/456":
						gets++
						require.Equal(t, http.MethodGet, req.Method)
						body = fmt.Sprintf(`{"elementInstanceKey":"456","processInstanceKey":"123","tenantId":%q}`, observedTenant)
					case "/v2/element-instances/search":
						searches++
						var request struct {
							Filter map[string]any `json:"filter"`
							Page   struct {
								From  int `json:"from"`
								Limit int `json:"limit"`
							} `json:"page"`
						}
						require.NoError(t, json.NewDecoder(req.Body).Decode(&request))
						require.Equal(t, "configured", request.Filter["tenantId"])
						require.Equal(t, 1, request.Page.Limit)
						require.Less(t, request.Page.From, 2)
						body = fmt.Sprintf(`{"items":[{"elementInstanceKey":%q,"processInstanceKey":"123","tenantId":"configured"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`, fmt.Sprint(456+request.Page.From))
					case "/v2/variables/search":
						require.Equal(t, "false", req.URL.Query().Get("truncateValues"))
						var request struct {
							Filter map[string]any `json:"filter"`
							Page   map[string]any `json:"page"`
						}
						require.NoError(t, json.NewDecoder(req.Body).Decode(&request))
						require.Equal(t, "123", request.Filter["processInstanceKey"])
						if mode == "get" {
							require.NotContains(t, request.Filter, "tenantId")
						} else {
							require.Equal(t, "configured", request.Filter["tenantId"])
						}
						scope := request.Filter["scopeKey"].(string)
						reads[scope]++
						if scope == "457" {
							body = `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
							break
						}
						require.Equal(t, "456", scope)
						require.LessOrEqual(t, reads[scope], 3)
						if mode == "later failure" && reads[scope] == 3 {
							return &http.Response{StatusCode: http.StatusForbidden, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{"title":"denied"}`)), Request: req}, nil
						}
						switch reads[scope] {
						case 1:
							body = fmt.Sprintf(`{"items":[{"name":"z","value":"1","processInstanceKey":"123","scopeKey":"456","tenantId":%q,"truncated":true},{"name":"parent","value":"0","processInstanceKey":"123","scopeKey":"123"}],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"c1"}}`, observedTenant)
						case 2:
							require.Equal(t, "c1", request.Page["after"])
							body = `{"items":[],"page":{"totalItems":3,"hasMoreTotalItems":true,"endCursor":"c2"}}`
						case 3:
							require.Equal(t, "c2", request.Page["after"])
							body = `{"items":[{"name":"a","value":"null","processInstanceKey":"123","scopeKey":"456"}],"page":{"totalItems":3,"hasMoreTotalItems":false}}`
						}
					case "/v2/jobs/search":
						jobs++
						body = `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`
					default:
						t.Fatalf("unexpected request %s %s", req.Method, req.URL.Path)
					}
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
				})
				cli, err := New(WithConfig(cfg), WithHTTPClient(&http.Client{Transport: transport}), WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
				require.NoError(t, err)
				enrichment := element.Enrichment{Variables: true, Listeners: true}
				var items []element.Element
				request := element.SearchRequest{BatchSize: 1, Limit: 2}
				if mode == "limit" {
					request.Limit = 1
				}
				switch mode {
				case "get":
					got, e := cli.GetElementWithEnrichment(context.Background(), "456", enrichment)
					err = e
					items = []element.Element{got}
				case "collected":
					got, e := cli.SearchElementsWithEnrichment(context.Background(), request, enrichment)
					err = e
					items = got.Items
					require.NoError(t, err)
					require.Equal(t, int32(2), got.Total)
				default:
					got, e := cli.SearchElementsPagesWithEnrichment(context.Background(), request, enrichment, func(step element.SearchPageStep) (element.SearchPageAction, error) {
						visits++
						require.NotNil(t, step.Page.Items[0].Variables)
						require.NotNil(t, step.Page.Items[0].Listeners)
						require.Equal(t, int32(visits), step.CumulativeCount)
						if mode == "limit" {
							require.True(t, step.LimitReached)
						}
						if mode == "stop" {
							return element.SearchPageActionStop, nil
						}
						return element.SearchPageActionContinue, nil
					})
					err = e
					items = got.Items
					if mode != "later failure" {
						require.NoError(t, err)
					}
					require.Equal(t, int32(visits), got.Pages)
				}
				require.Equal(t, 3, reads["456"])
				if mode == "later failure" {
					require.ErrorIs(t, err, ferrors.ErrLocalPrecondition)
					require.Empty(t, items)
					require.Zero(t, visits)
					require.Zero(t, jobs)
					return
				}
				require.NoError(t, err)
				require.Equal(t, "456", items[0].ElementInstanceKey)
				require.Len(t, *items[0].Variables, 2)
				require.Equal(t, "a", (*items[0].Variables)[0].Name)
				require.Equal(t, "null", (*items[0].Variables)[0].Value)
				require.Equal(t, "z", (*items[0].Variables)[1].Name)
				require.True(t, (*items[0].Variables)[1].APITruncated)
				if mode == "get" {
					require.Equal(t, 1, gets)
					require.Zero(t, searches)
				} else {
					require.Zero(t, gets)
					require.Equal(t, len(items), searches)
				}
				if mode == "collected" || mode == "pages" {
					require.Len(t, items, 2)
					require.Equal(t, "457", items[1].ElementInstanceKey)
					require.NotNil(t, *items[1].Variables)
					require.Empty(t, *items[1].Variables)
					require.Equal(t, 1, reads["457"])
				} else {
					require.Len(t, items, 1)
					require.Zero(t, reads["457"])
				}
				if mode == "pages" {
					require.Equal(t, 4, jobs)
				} else {
					require.Equal(t, 2, jobs)
				}
			})
		}
	}
}
