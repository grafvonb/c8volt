// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package v810

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/config"
	camundav810 "github.com/grafvonb/c8volt/internal/clients/camunda/v810/camunda"
	d "github.com/grafvonb/c8volt/internal/domain"
	"github.com/grafvonb/c8volt/internal/services"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/stretchr/testify/require"
)

// TestUpdateScopeVariables verifies v8.10 sends one local-only scope write and reports its acceptance.
func TestUpdateScopeVariables(t *testing.T) {
	svc := newVariableTestService(t, &mockVariableClient{
		createElementInstanceVariablesResponse: func(_ context.Context, key camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
			require.Equal(t, camundav810.ElementInstanceKey("456"), key)
			require.Equal(t, map[string]any{"foo": "bar"}, body.Variables)
			require.NotNil(t, body.Local)
			require.True(t, *body.Local)
			return &camundav810.CreateElementInstanceVariablesResponse{HTTPResponse: variableHTTPResponse(http.MethodPut, "https://camunda.local/v2/element-instances/456/variables", http.StatusNoContent, "204 No Content")}, nil
		},
	})

	got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"foo": "bar"})

	require.NoError(t, err)
	require.Equal(t, d.ScopeVariableUpdateResponse{ScopeKey: "456", SubmissionStatus: d.ScopeVariableSubmissionAccepted, Accepted: true, StatusCode: http.StatusNoContent, Status: "204 No Content"}, got)
}

// TestUpdateScopeVariablesRejectsInvalidInput verifies invalid scope writes fail before transport access.
func TestUpdateScopeVariablesRejectsInvalidInput(t *testing.T) {
	calls := 0
	svc := newVariableTestService(t, &mockVariableClient{
		createElementInstanceVariablesResponse: func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
			calls++
			return nil, nil
		},
	})

	got, err := svc.UpdateScopeVariables(context.Background(), " ", map[string]any{"foo": "bar"})
	require.ErrorIs(t, err, d.ErrValidation)
	require.Equal(t, d.ScopeVariableSubmissionNotSent, got.SubmissionStatus)
	require.False(t, got.Accepted)
	got, err = svc.UpdateScopeVariables(context.Background(), "456", nil)
	require.ErrorIs(t, err, d.ErrValidation)
	require.Equal(t, d.ScopeVariableSubmissionNotSent, got.SubmissionStatus)
	require.False(t, got.Accepted)
	require.Zero(t, calls)
}

// TestUpdateScopeVariablesErrors verifies transport, missing-response, and HTTP failures remain unsuccessful.
func TestUpdateScopeVariablesErrors(t *testing.T) {
	transportErr := errors.New("transport failed")
	tests := []struct {
		name    string
		resp    *camundav810.CreateElementInstanceVariablesResponse
		err     error
		wantErr error
	}{
		{name: "transport", err: transportErr, wantErr: transportErr},
		{name: "nil response", wantErr: d.ErrMalformedResponse},
		{name: "http", resp: &camundav810.CreateElementInstanceVariablesResponse{HTTPResponse: variableHTTPResponse(http.MethodPut, "https://camunda.local/v2/element-instances/456/variables", http.StatusBadRequest, "400 Bad Request"), Body: []byte(`{"message":"bad variable"}`)}, wantErr: d.ErrBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newVariableTestService(t, &mockVariableClient{
				createElementInstanceVariablesResponse: func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
					return tt.resp, tt.err
				},
			})

			got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"foo": "bar"})

			require.ErrorIs(t, err, tt.wantErr)
			require.False(t, got.Accepted)
		})
	}
}

// TestScopeVariablesReadsCompleteLocalCollection proves independent identities,
// sparse cursor continuation, raw counts, optional evidence, and stable ordering.
func TestScopeVariablesReadsCompleteLocalCollection(t *testing.T) {
	t.Parallel()
	for _, ignore := range []bool{false, true} {
		t.Run(fmt.Sprint("ignore tenant=", ignore), func(t *testing.T) {
			calls := 0
			tenant := "tenant-b"
			query := d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456", ExpectedTenantId: &tenant}
			svc := scopeReadService(t, func(_ context.Context, params *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
				calls++
				require.LessOrEqual(t, calls, 4)
				require.NotNil(t, params.TruncateValues)
				require.False(t, *params.TruncateValues)
				raw, err := json.Marshal(body)
				require.NoError(t, err)
				require.Contains(t, string(raw), `"processInstanceKey":"123"`)
				require.Contains(t, string(raw), `"scopeKey":"456"`)
				require.Contains(t, string(raw), `"field":"name"`)
				if ignore {
					require.Nil(t, body.Filter.TenantId)
				} else {
					require.Equal(t, camundav810.TenantId("tenant-a"), *body.Filter.TenantId)
				}
				if calls > 1 {
					require.Contains(t, string(raw), fmt.Sprintf(`"after":"c%d"`, calls-1))
				}
				bodies := []string{
					`{"items":[{"name":"same","value":"2","variableKey":"b","processInstanceKey":"123","scopeKey":"456","tenantId":"tenant-b","truncated":true},{"name":"parent","value":"0","processInstanceKey":"123","scopeKey":"123"}],"page":{"totalItems":5,"hasMoreTotalItems":true,"endCursor":"c1"}}`,
					`{"items":[],"page":{"totalItems":5,"hasMoreTotalItems":true,"endCursor":"c2"}}`,
					`{"items":[{"name":"sibling","value":"0","processInstanceKey":"123","scopeKey":"789"},{"name":"called","value":"0","processInstanceKey":"999","scopeKey":"999"}],"page":{"totalItems":5,"hasMoreTotalItems":true,"endCursor":"c3"}}`,
					`{"items":[{"name":"same","value":"1","variableKey":"a","processInstanceKey":"123","scopeKey":"456","isTruncated":true}],"page":{"totalItems":5,"hasMoreTotalItems":false}}`,
				}
				return scopeReadResponse(bodies[calls-1]), nil
			})
			var opts []services.CallOption
			if ignore {
				opts = append(opts, services.WithIgnoreTenant())
			}
			got, err := svc.SearchScopeVariables(context.Background(), query, opts...)
			require.NoError(t, err)
			require.Equal(t, 4, calls)
			require.Equal(t, []d.ProcessInstanceVariable{
				{Name: "same", Value: "1", VariableKey: "a", ProcessInstanceKey: "123", ScopeKey: "456", APITruncated: true},
				{Name: "same", Value: "2", VariableKey: "b", ProcessInstanceKey: "123", ScopeKey: "456", TenantId: "tenant-b", APITruncated: true},
			}, got)
		})
	}
}

// TestScopeVariablesIdentity rejects incomplete or contradictory scope evidence
// while treating absent optional tenant/key fields as unknown.
func TestScopeVariablesIdentity(t *testing.T) {
	t.Parallel()
	tenant := "tenant-b"
	tests := []struct {
		name, item string
		wantErr    bool
	}{
		{"optional evidence", `{"name":"v","value":"null","processInstanceKey":"123","scopeKey":"456"}`, false},
		{"empty tenant", `{"name":"v","value":"null","processInstanceKey":"123","scopeKey":"456","tenantId":""}`, false},
		{"wrong owner", `{"name":"v","value":"1","processInstanceKey":"999","scopeKey":"456"}`, true},
		{"wrong tenant", `{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"456","tenantId":"tenant-c"}`, true},
		{"missing owner", `{"name":"v","value":"1","scopeKey":"456"}`, true},
		{"blank owner", `{"name":"v","value":"1","processInstanceKey":" ","scopeKey":"456"}`, true},
		{"missing scope", `{"name":"v","value":"1","processInstanceKey":"123"}`, true},
		{"blank scope", `{"name":"v","value":"1","processInstanceKey":"123","scopeKey":" "}`, true},
		{"missing name", `{"value":"1","processInstanceKey":"123","scopeKey":"456"}`, true},
		{"missing value", `{"name":"v","processInstanceKey":"123","scopeKey":"456"}`, true},
		{"wrong identity type", `{"name":"v","value":"1","processInstanceKey":123,"scopeKey":"456"}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := scopeReadService(t, func(context.Context, *camundav810.SearchVariablesParams, camundav810.SearchVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
				return scopeReadResponse(`{"items":[` + tt.item + `],"page":{"totalItems":1,"hasMoreTotalItems":false}}`), nil
			})
			got, err := svc.SearchScopeVariables(context.Background(), d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456", ExpectedTenantId: &tenant}, services.WithIgnoreTenant())
			if tt.wantErr {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Len(t, got, 1)
			}
		})
	}
	// Without expected evidence, a returned tenant must never be compared with configured discovery tenant.
	svc := scopeReadService(t, func(context.Context, *camundav810.SearchVariablesParams, camundav810.SearchVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
		return scopeReadResponse(`{"items":[{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"456","tenantId":"tenant-c"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`), nil
	})
	got, err := svc.SearchScopeVariables(context.Background(), d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456"}, services.WithIgnoreTenant())
	require.NoError(t, err)
	require.Len(t, got, 1)
}

// TestScopeVariablesValidateBeforeRequests proves neither incomplete identities
// nor invalid page positions can reach the generated client.
func TestScopeVariablesValidateBeforeRequests(t *testing.T) {
	t.Parallel()
	calls := 0
	svc := scopeReadService(t, func(context.Context, *camundav810.SearchVariablesParams, camundav810.SearchVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
		calls++
		return nil, errors.New("unexpected request")
	})
	for _, query := range []d.ScopeVariableQuery{{}, {ProcessInstanceKey: "123"}, {ScopeKey: "456"}, {ProcessInstanceKey: " ", ScopeKey: "456"}, {ProcessInstanceKey: "123", ScopeKey: " "}} {
		_, err := svc.SearchScopeVariables(context.Background(), query)
		require.ErrorIs(t, err, d.ErrValidation)
		_, err = svc.SearchScopeVariablesPage(context.Background(), query, d.ProcessInstanceVariablePageRequest{Size: 10})
		require.ErrorIs(t, err, d.ErrValidation)
	}
	for _, page := range []d.ProcessInstanceVariablePageRequest{{}, {Size: -1}, {Size: 1001}, {Size: 1, From: -1}, {Size: 1, From: 1, After: "c"}} {
		_, err := svc.SearchScopeVariablesPage(context.Background(), d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456"}, page)
		require.Error(t, err)
	}
	require.Zero(t, calls)
}

// TestScopeVariablesCompletionAndFailures ensures incomplete traversal cannot
// masquerade as successful emptiness, including cancellation after a response.
func TestScopeVariablesCompletionAndFailures(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"empty", "offset", "repeated cursor", "changed total", "later error", "canceled before", "canceled after", "malformed metadata", "nil response"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "canceled before" {
				cancel()
			}
			calls := 0
			laterErr := errors.New("later page failed")
			svc := scopeReadService(t, func(_ context.Context, _ *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
				calls++
				require.LessOrEqual(t, calls, 3)
				switch mode {
				case "empty":
					return scopeReadResponse(`{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`), nil
				case "offset":
					if calls == 1 {
						return scopeReadResponse(`{"items":[{"name":"other","value":"1","processInstanceKey":"123","scopeKey":"789"},{"name":"local","value":"2","processInstanceKey":"123","scopeKey":"456"}],"page":{}}`), nil
					}
					offset, err := body.Page.AsOffsetPagination()
					require.NoError(t, err)
					require.Equal(t, int32(2), *offset.From)
					return scopeReadResponse(`{"items":[],"page":{}}`), nil
				case "repeated cursor":
					return scopeReadResponse(`{"items":[],"page":{"endCursor":"same"}}`), nil
				case "changed total":
					if calls > 1 {
						return scopeReadResponse(`{"items":[],"page":{"totalItems":3,"hasMoreTotalItems":false}}`), nil
					}
				case "later error":
					if calls > 1 {
						return nil, laterErr
					}
				case "canceled after":
					cancel()
				case "malformed metadata":
					return scopeReadResponse(`{"items":[],"page":{"totalItems":0}}`), nil
				case "nil response":
					return nil, nil
				}
				return scopeReadResponse(`{"items":[{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"456"}],"page":{"totalItems":2,"hasMoreTotalItems":false,"endCursor":"next"}}`), nil
			})
			got, err := svc.SearchScopeVariables(ctx, d.ScopeVariableQuery{ProcessInstanceKey: "123", ScopeKey: "456"})
			switch mode {
			case "empty":
				require.NoError(t, err)
				require.NotNil(t, got)
				require.Empty(t, got)
				require.Equal(t, 1, calls)
			case "offset":
				require.NoError(t, err)
				require.Len(t, got, 1)
				require.Equal(t, 2, calls)
			case "canceled before", "canceled after":
				require.ErrorIs(t, err, context.Canceled)
				require.Nil(t, got)
			case "later error":
				require.ErrorIs(t, err, laterErr)
				require.Nil(t, got)
				require.Equal(t, 2, calls)
			default:
				require.ErrorIs(t, err, d.ErrMalformedResponse)
				require.Nil(t, got)
			}
			if mode == "canceled before" {
				require.Zero(t, calls)
			}
		})
	}
}

// scopeReadResponse preserves raw fields absent from generated variable models.
func scopeReadResponse(body string) *camundav810.SearchVariablesResponse {
	return &camundav810.SearchVariablesResponse{HTTPResponse: variableHTTPResponse(http.MethodPost, "https://camunda.local/v2/variables/search", http.StatusOK, "200 OK"), Body: []byte(body), JSON200: &camundav810.VariableSearchQueryResult{}}
}

// scopeReadService installs only a generated-client fake; no transport is used.
func scopeReadService(t *testing.T, read func(context.Context, *camundav810.SearchVariablesParams, camundav810.SearchVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error)) *Service {
	t.Helper()
	return newVariableTestService(t, &mockVariableClient{searchVariablesWithResponse: read})
}

// TestProcessInstanceVariableRootCompatibility keeps legacy filtering, required
// raw fields and page errors even when the same adapter supports strict scopes.
func TestProcessInstanceVariableRootCompatibility(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, items string
		wantErr     bool
	}{
		{"wrong owner filtered", `[{"name":"v","value":"1","variableKey":"a","processInstanceKey":"other","scopeKey":"123","tenantId":"tenant-a"}]`, false},
		{"different tenant retained", `[{"name":"v","value":"1","variableKey":"a","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-b"}]`, false},
		{"missing tenant still malformed", `[{"name":"v","value":"1","variableKey":"a","processInstanceKey":"123","scopeKey":"123"}]`, true},
		{"missing variable key still malformed", `[{"name":"v","value":"1","processInstanceKey":"123","scopeKey":"123","tenantId":"tenant-a"}]`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := scopeReadService(t, func(_ context.Context, _ *camundav810.SearchVariablesParams, body camundav810.SearchVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.SearchVariablesResponse, error) {
				calls++
				raw, err := json.Marshal(body)
				require.NoError(t, err)
				require.Contains(t, string(raw), `"processInstanceKey":"123"`)
				require.Contains(t, string(raw), `"scopeKey":"123"`)
				return scopeReadResponse(`{"items":` + tc.items + `,"page":{"totalItems":1,"hasMoreTotalItems":false}}`), nil
			})
			got, err := svc.SearchProcessInstanceVariables(context.Background(), "123", services.WithIgnoreTenant())
			if tc.wantErr {
				require.ErrorIs(t, err, d.ErrMalformedResponse)
				require.EqualError(t, err, "decode process-instance variables: malformed response: variable item 0 is missing required fields")
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				if tc.name == "wrong owner filtered" {
					require.Empty(t, got)
				} else {
					require.Len(t, got, 1)
					require.Equal(t, "tenant-b", got[0].TenantId)
				}
			}
			require.Equal(t, 1, calls)
			_, err = svc.SearchProcessInstanceVariablesPage(context.Background(), "123", d.ProcessInstanceVariablePageRequest{})
			require.EqualError(t, err, "validate process-instance variable page: process-instance variable page request size must be positive")
			require.Equal(t, 1, calls)
		})
	}
}

// TestUpdateScopeVariablesSubmissionEvidence keeps protocol evidence separate
// from legacy HTTP acceptance/error mapping and never retries documented 400s.
func TestUpdateScopeVariablesSubmissionEvidence(t *testing.T) {
	t.Parallel()
	transportErr := errors.New("interrupted response")
	invalidData := `{"type":"about:blank","title":"Bad Request","status":400,"detail":"invalid variables","instance":"/v2/element-instances/456/variables"}`
	for _, tc := range []struct {
		name             string
		code             int
		body             string
		callErr, wantErr error
		evidence         d.ScopeVariableSubmissionStatus
		legacyAccepted   bool
	}{
		{"accepted", 204, "", nil, nil, d.ScopeVariableSubmissionAccepted, true},
		{"invalid data", 400, invalidData, nil, d.ErrBadRequest, d.ScopeVariableSubmissionRejected, false},
		{"malformed invalid data", 400, `{`, nil, d.ErrBadRequest, d.ScopeVariableSubmissionUnknown, false},
		{"incomplete invalid data", 400, `{}`, nil, d.ErrBadRequest, d.ScopeVariableSubmissionUnknown, false},
		{"internal", 500, invalidData, nil, d.ErrInternal, d.ScopeVariableSubmissionUnknown, false},
		{"unavailable", 503, invalidData, nil, d.ErrUnavailable, d.ScopeVariableSubmissionUnknown, false},
		{"gateway timeout", 504, invalidData, nil, d.ErrGatewayTimeout, d.ScopeVariableSubmissionUnknown, false},
		{"success plus transport error", 204, "", transportErr, transportErr, d.ScopeVariableSubmissionUnknown, false},
		{"rejection plus transport error", 400, invalidData, transportErr, transportErr, d.ScopeVariableSubmissionUnknown, false},
		{"no response", 0, "", nil, d.ErrMalformedResponse, d.ScopeVariableSubmissionUnknown, false},
		{"transport", 0, "", transportErr, transportErr, d.ScopeVariableSubmissionUnknown, false},
		{"generic cancellation", 0, "", context.Canceled, context.Canceled, d.ScopeVariableSubmissionUnknown, false},
		// Preserve the pre-existing PI/UT HTTP mapping while element outcomes can
		// reject undocumented acceptance using the stricter new evidence field.
		{"legacy success code", 200, "", nil, nil, d.ScopeVariableSubmissionUnknown, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			svc := newVariableTestService(t, &mockVariableClient{createElementInstanceVariablesResponse: func(_ context.Context, key camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, _ ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
				calls++
				require.Equal(t, camundav810.ElementInstanceKey("456"), key)
				require.True(t, *body.Local)
				if tc.code == 0 {
					return nil, tc.callErr
				}
				response := variableHTTPResponse(http.MethodPut, "https://fixture.invalid/v2/element-instances/456/variables", tc.code, "")
				response.Header = http.Header{"Content-Type": []string{"application/problem+json"}}
				return &camundav810.CreateElementInstanceVariablesResponse{HTTPResponse: response, Body: []byte(tc.body)}, tc.callErr
			}})
			got, err := svc.UpdateScopeVariables(context.Background(), "456", map[string]any{"value": true})
			require.ErrorIs(t, err, tc.wantErr)
			require.Equal(t, tc.evidence, got.SubmissionStatus)
			require.Equal(t, tc.legacyAccepted, got.Accepted)
			require.Equal(t, tc.code, got.StatusCode)
			require.Equal(t, 1, calls)
		})
	}
}

// TestUpdateScopeVariablesRetryEvidence retains the last observed status when
// cancellation makes RetryCamundaMutation return a zero response.
func TestUpdateScopeVariablesRetryEvidence(t *testing.T) {
	t.Parallel()
	for _, mode := range []string{"canceled before", "backoff cancellation", "accepted after ambiguity", "rejected after ambiguity"} {
		t.Run(mode, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if mode == "canceled before" {
				cancel()
			}
			calls := 0
			svc := newVariableTestService(t, &mockVariableClient{createElementInstanceVariablesResponse: func(context.Context, camundav810.ElementInstanceKey, camundav810.CreateElementInstanceVariablesJSONRequestBody, ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
				calls++
				require.LessOrEqual(t, calls, 2)
				code := 429
				body := ""
				if calls == 2 {
					code = 204
					if mode == "rejected after ambiguity" {
						code = 400
						body = `{"type":"about:blank","title":"Bad Request","status":400,"detail":"invalid variables","instance":"/"}`
					}
				}
				response := variableHTTPResponse(http.MethodPut, "https://fixture.invalid/v2/element-instances/456/variables", code, "")
				response.Header = http.Header{"Retry-After": []string{"0"}, "Content-Type": []string{"application/problem+json"}}
				if mode == "backoff cancellation" {
					response.Header.Set("Retry-After", "30")
					cancel()
				}
				return &camundav810.CreateElementInstanceVariablesResponse{HTTPResponse: response, Body: []byte(body)}, nil
			}})
			got, err := svc.UpdateScopeVariables(ctx, "456", map[string]any{"value": true})
			switch mode {
			case "canceled before":
				require.ErrorIs(t, err, context.Canceled)
				require.Zero(t, calls)
				require.Zero(t, got.StatusCode)
				require.Equal(t, d.ScopeVariableSubmissionNotSent, got.SubmissionStatus)
			case "backoff cancellation":
				require.ErrorIs(t, err, context.Canceled)
				require.Equal(t, 1, calls)
				require.Equal(t, 429, got.StatusCode)
				require.Equal(t, d.ScopeVariableSubmissionUnknown, got.SubmissionStatus)
			case "accepted after ambiguity":
				require.NoError(t, err)
				require.Equal(t, 2, calls)
				require.Equal(t, 204, got.StatusCode)
				require.True(t, got.Accepted)
				require.Equal(t, d.ScopeVariableSubmissionAccepted, got.SubmissionStatus)
			case "rejected after ambiguity":
				require.ErrorIs(t, err, d.ErrBadRequest)
				require.Equal(t, 2, calls)
				require.Equal(t, 400, got.StatusCode)
				require.Equal(t, d.ScopeVariableSubmissionUnknown, got.SubmissionStatus)
			}
		})
	}
}

// scopeSubmissionTransport isolates physical dispatch while retaining the real
// shared request-rate transport, generated decoder and http.Client timeout.
type scopeSubmissionTransport func(*http.Request) (*http.Response, error)

// RoundTrip delegates only to the test-owned in-memory transport.
func (f scopeSubmissionTransport) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestUpdateScopeVariablesPacingEvidence proves every retry has fresh admission
// evidence, including the actual net/http timeout substitution and prior ambiguity.
func TestUpdateScopeVariablesPacingEvidence(t *testing.T) {
	t.Parallel()
	for _, prior := range []bool{false, true} {
		for _, substituted := range []bool{false, true} {
			t.Run(fmt.Sprintf("prior=%t substituted=%t", prior, substituted), func(t *testing.T) {
				var dispatched atomic.Int32
				shared, err := httpc.New(&config.Config{HTTP: config.HTTP{Timeout: "100ms", RequestRate: 0.01, RequestRateReportInterval: "5s"}}, slog.New(slog.NewTextHandler(io.Discard, nil)), func(s *httpc.Service) {
					s.Client().Transport = scopeSubmissionTransport(func(req *http.Request) (*http.Response, error) {
						dispatched.Add(1)
						code := http.StatusNoContent
						if prior {
							code = http.StatusTooManyRequests
						}
						return &http.Response{StatusCode: code, Status: http.StatusText(code), Request: req, Header: http.Header{"Retry-After": []string{"0"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
					})
				})
				require.NoError(t, err)
				defer shared.Close()
				parent := shared.ToContext(context.Background())
				if !prior {
					req, err := http.NewRequestWithContext(parent, http.MethodPut, "https://fixture.invalid/prime", nil)
					require.NoError(t, err)
					response, err := shared.Client().Do(req)
					require.NoError(t, err)
					require.NoError(t, response.Body.Close())
				}
				base := shared.Client().Transport
				var substitutedCalls atomic.Int32
				if substituted {
					shared.Client().Transport = scopeSubmissionTransport(func(req *http.Request) (*http.Response, error) {
						response, err := base.RoundTrip(req)
						if httpc.IsRequestRateWaitError(err) {
							substitutedCalls.Add(1)
							<-req.Cancel // Force Client.Timeout to replace the typed admission error.
						}
						return response, err
					})
				}
				client, err := camundav810.NewClientWithResponses("https://fixture.invalid/v2", camundav810.WithHTTPClient(shared.Client()))
				require.NoError(t, err)
				calls := 0
				svc := newVariableTestService(t, &mockVariableClient{createElementInstanceVariablesResponse: func(ctx context.Context, key camundav810.ElementInstanceKey, body camundav810.CreateElementInstanceVariablesJSONRequestBody, opts ...camundav810.RequestEditorFn) (*camundav810.CreateElementInstanceVariablesResponse, error) {
					calls++
					response, callErr := client.CreateElementInstanceVariablesWithResponse(ctx, key, body, opts...)
					if substituted && callErr != nil {
						require.False(t, httpc.IsRequestRateWaitError(callErr), "actual Client.Timeout substituted the transport error")
					}
					return response, callErr
				}})
				got, err := svc.UpdateScopeVariables(parent, "456", map[string]any{"value": true})
				require.ErrorIs(t, err, context.DeadlineExceeded)
				require.True(t, httpc.IsRequestRateWaitError(err), "restore classification when client timeout replaces the typed error")
				require.NoError(t, parent.Err())
				require.Equal(t, int32(1), dispatched.Load(), "no physical dispatch after admission failure")
				if prior {
					require.Equal(t, 2, calls)
				} else {
					require.Equal(t, 1, calls)
				}
				require.False(t, got.Accepted)
				if prior {
					require.Equal(t, d.ScopeVariableSubmissionUnknown, got.SubmissionStatus)
					require.Equal(t, 429, got.StatusCode)
				} else {
					require.Equal(t, d.ScopeVariableSubmissionNotSent, got.SubmissionStatus)
					require.Zero(t, got.StatusCode)
				}
				if substituted {
					require.Equal(t, int32(1), substitutedCalls.Load())
				}
				// A new invocation/attempt must not inherit the previous admission failure.
				_, observation := httpc.BeginRequestRateAttempt(parent)
				require.NoError(t, observation.Failure())
			})
		}
	}
}
