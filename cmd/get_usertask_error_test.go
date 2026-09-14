// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	options "github.com/grafvonb/c8volt/c8volt/foptions"
	"github.com/grafvonb/c8volt/c8volt/task"
	"github.com/grafvonb/c8volt/testx"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

// TestGetUserTaskError_FailureAfterStreamedPage verifies an earlier keys page
// remains plainly partial when a later backend read fails: no success summary
// is emitted and the command exits with the backend error.
func TestGetUserTaskError_FailureAfterStreamedPage(t *testing.T) {
	server, requests := newFailingUserTaskPagingServer(t)
	configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

	stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--keys-only", "get", "ut", "--batch-size", "1")
	require.Error(t, err)
	require.Equal(t, "2251799815391233\n", stdout)
	require.Contains(t, stderr, "get user tasks")
	require.NotContains(t, stdout, "found:")
	require.Equal(t, []string{"/v2/user-tasks/search", "/v2/user-tasks/search"}, requests.Snapshot())
}

// TestGetUserTaskError_CollectedAndTotalFailuresHaveNoPartialSuccess verifies
// collected JSON and exact totals never turn a failed traversal into a partial
// successful payload or numeric result.
func TestGetUserTaskError_CollectedAndTotalFailuresHaveNoPartialSuccess(t *testing.T) {
	tests := []struct {
		name string
		args []string
		json bool
	}{
		{name: "json", args: []string{"--json", "get", "ut", "--batch-size", "1"}, json: true},
		{name: "total", args: []string{"get", "ut", "--total", "--batch-size", "1"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server, requests := newFailingUserTaskPagingServer(t)
			configPath := testx.WriteTestConfigForVersion(t, server.URL, "8.9")

			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", test.args...)
			require.Error(t, err)
			require.Len(t, requests.Snapshot(), 2)
			if !test.json {
				require.Empty(t, stdout, "a failed total traversal must not print a partial number")
				require.Contains(t, stderr, "get user tasks total")
				return
			}

			require.Empty(t, stderr)
			var envelope map[string]any
			decoder := json.NewDecoder(bytes.NewBufferString(stdout))
			require.NoError(t, decoder.Decode(&envelope))
			require.Equal(t, "failed", envelope["outcome"])
			require.Nil(t, envelope["payload"])
			require.NotContains(t, stdout, `"items"`)
			var extra any
			require.Error(t, decoder.Decode(&extra), "failure output must contain exactly one envelope")
		})
	}
}

// TestGetUserTaskError_CancellationRemainsCallerVisible verifies the paging
// bridge does not reinterpret cancellation as visitor stop or empty success.
func TestGetUserTaskError_CancellationRemainsCallerVisible(t *testing.T) {
	resetProcessInstanceCommandGlobals()
	t.Cleanup(resetProcessInstanceCommandGlobals)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd := &cobra.Command{}
	cmd.SetContext(ctx)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cli := stubTaskAPI{searchUserTasksPages: func(ctx context.Context, _ task.SearchRequest, _ task.SearchPageVisitor, _ ...options.FacadeOption) (task.SearchPagesResult, error) {
		require.ErrorIs(t, ctx.Err(), context.Canceled)
		return task.SearchPagesResult{}, context.Canceled
	}}

	_, rendered, err := searchUserTasksWithPaging(cmd, cli, task.SearchRequest{})
	require.False(t, rendered)
	require.True(t, errors.Is(err, context.Canceled))
}

// TestGetUserTaskError_LegacyResolverKeepsTenantAndTasklistFallback verifies
// the new native command path has not replaced get-pi's tenant-scoped legacy
// resolver or its v8.8/v8.9 Tasklist fallback.
func TestGetUserTaskError_LegacyResolverKeepsTenantAndTasklistFallback(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		fallback bool
		want     []string
	}{
		{
			name: "primary v8.8", version: "8.8",
			want: []string{"/v2/user-tasks/search", "/v2/process-instances/2251799813711967"},
		},
		{
			name: "fallback v8.9", version: "8.9", fallback: true,
			want: []string{"/v2/user-tasks/search", "/v1/tasks/2251799815391233", "/v2/process-instances/2251799813711967"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var requests testx.SafeSlice[string]
			server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				requests.Append(request.URL.Path)
				writer.Header().Set("Content-Type", "application/json")
				switch request.URL.Path {
				case "/v2/user-tasks/search":
					body := requireUserTaskSearchRequest(t, request, "2251799815391233", "tenant-a")
					require.NotNil(t, body["filter"])
					if test.fallback {
						_, _ = fmt.Fprint(writer, `{"items":[],"page":{"totalItems":0,"hasMoreTotalItems":false}}`)
						return
					}
					_, _ = fmt.Fprint(writer, `{"items":[{"userTaskKey":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant-a"}],"page":{"totalItems":1,"hasMoreTotalItems":false}}`)
				case "/v1/tasks/2251799815391233":
					require.True(t, test.fallback, "native resolver success must not call Tasklist")
					requireTasklistFallbackTaskRequest(t, request, "2251799815391233")
					_, _ = fmt.Fprint(writer, `{"id":"2251799815391233","processInstanceKey":"2251799813711967","tenantId":"tenant-a","implementation":"JOB_WORKER"}`)
				case "/v2/process-instances/2251799813711967":
					require.Equal(t, http.MethodGet, request.Method)
					_, _ = fmt.Fprint(writer, `{"hasIncident":false,"processDefinitionId":"demo","processDefinitionKey":"9001","processDefinitionName":"demo","processDefinitionVersion":3,"processInstanceKey":"2251799813711967","startDate":"2026-03-23T18:00:00Z","state":"ACTIVE","tenantId":"tenant-a"}`)
				default:
					http.NotFound(writer, request)
				}
			}))
			t.Cleanup(server.Close)
			configPath := testx.WriteTestConfigForVersion(t, server.URL, test.version)

			stdout, stderr, err := runGetUserTaskCommand(t, configPath, "", "--tenant", "tenant-a", "get", "pi", "--has-user-tasks", "2251799815391233")
			require.NoError(t, err, stderr)
			require.Empty(t, stderr)
			require.Contains(t, stdout, "2251799813711967")
			require.NotContains(t, stdout, "2251799815391233")
			require.Equal(t, test.want, requests.Snapshot())
		})
	}
}

// newFailingUserTaskPagingServer returns one lower-bound page followed by a
// backend failure and records requests safely for visitor-driven or count runs.
func newFailingUserTaskPagingServer(t *testing.T) (*httptest.Server, *testx.SafeSlice[string]) {
	t.Helper()
	var requests testx.SafeSlice[string]
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Append(request.URL.Path)
		if request.Method != http.MethodPost || request.URL.Path != "/v2/user-tasks/search" {
			http.NotFound(writer, request)
			return
		}
		if len(requests.Snapshot()) == 1 {
			writer.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprint(writer, userTaskSearchResponse(2, true, "cursor-a", "2251799815391233"))
			return
		}
		http.Error(writer, `{"message":"backend exploded"}`, http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}
