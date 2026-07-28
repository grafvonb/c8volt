// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package httpc

import (
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

// TestLogTransport_StartsAndStopsActivityAroundRequest verifies HTTP waits are bracketed by activity calls.
func TestLogTransport_StartsAndStopsActivityAroundRequest(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	transport := &LogTransport{
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Activity: sink,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			time.Sleep(5 * time.Millisecond)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	req, err := http.NewRequest(http.MethodPost, "https://camunda.example.test/v2/process-instances/search", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	started, stopped, msgs := sink.Snapshot()
	require.Equal(t, 1, started)
	require.Equal(t, 1, stopped)
	require.Equal(t, []string{"searching process instances"}, msgs)
	require.Equal(t, []activitysink.Start{{
		Message:    "searching process instances",
		Importance: logging.ActivityImportanceHTTP,
	}}, sink.Starts())
}

// TestLogTransport_HTTPActivityUsesFallbackImportance verifies request activity stays below broader scopes.
func TestLogTransport_HTTPActivityUsesFallbackImportance(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		activeImportance logging.ActivityImportance
	}{
		{name: "below workflow", activeImportance: logging.ActivityImportanceWorkflow},
		{name: "below wait", activeImportance: logging.ActivityImportanceWait},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sink := &activitysink.Sink{}
			stopActive := sink.StartActivityWithImportance("higher-level progress", tt.activeImportance)
			defer stopActive()
			transport := &LogTransport{
				Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
				Activity: sink,
				base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader("ok")),
						Header:     make(http.Header),
						Request:    req,
					}, nil
				}),
			}

			req, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/v2/jobs/2251799813685252", nil)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req)
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			require.Equal(t, []activitysink.Start{
				{Message: "higher-level progress", Importance: tt.activeImportance},
				{Message: "loading job", Importance: logging.ActivityImportanceHTTP},
			}, sink.Starts())
			require.Equal(t, 1, sink.Stopped(), "only the HTTP scope should stop during the request")
		})
	}
}

// TestLogTransport_HTTPActivityVisibleWithoutHigherScopes verifies simple requests still emit fallback activity.
func TestLogTransport_HTTPActivityVisibleWithoutHigherScopes(t *testing.T) {
	t.Parallel()

	sink := &activitysink.Sink{}
	transport := &LogTransport{
		Log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		Activity: sink,
		base: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("ok")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}),
	}

	req, err := http.NewRequest(http.MethodGet, "https://camunda.example.test/v2/tenants/tenant-a", nil)
	require.NoError(t, err)

	resp, err := transport.RoundTrip(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	require.Equal(t, []activitysink.Start{{
		Message:    "loading tenant",
		Importance: logging.ActivityImportanceHTTP,
	}}, sink.Starts())
	require.Equal(t, 1, sink.Stopped())
}

// TestHTTPActivityMessageUsesAllKnownEndpointLabels verifies the contract label table remains covered.
func TestHTTPActivityMessageUsesEndpointLabelsWithoutHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		url    string
		want   string
	}{
		{name: "topology", method: http.MethodGet, url: "https://camunda.example.test/v2/topology?token=secret", want: "checking cluster topology"},
		{name: "license", method: http.MethodGet, url: "https://camunda.example.test/v2/license", want: "loading license"},
		{name: "deployment", method: http.MethodPost, url: "https://camunda.example.test/v2/deployments", want: "deploying resources"},
		{name: "legacy deployment", method: http.MethodPost, url: "https://camunda.example.test/deployments", want: "deploying resources"},
		{name: "resources", method: http.MethodGet, url: "https://camunda.example.test/v2/resources", want: "loading resources"},
		{name: "resource", method: http.MethodGet, url: "https://camunda.example.test/v2/resources/resource-id-123", want: "loading resource"},
		{name: "resource deletion", method: http.MethodPost, url: "https://camunda.example.test/v2/resources/resource-id-123/deletion", want: "submitting resource deletion"},
		{name: "legacy resource deletion", method: http.MethodPost, url: "https://camunda.example.test/resources/resource-id-123/deletion", want: "submitting resource deletion"},
		{name: "process instance search", method: http.MethodPost, url: "https://camunda.example.test/v2/process-instances/search", want: "searching process instances"},
		{name: "legacy process instance search", method: http.MethodPost, url: "https://camunda.example.test/process-instances/search", want: "searching process instances"},
		{name: "process instance create", method: http.MethodPost, url: "https://camunda.example.test/v2/process-instances", want: "creating process instance"},
		{name: "process instance load", method: http.MethodGet, url: "https://camunda.example.test/v2/process-instances/2251799813685250", want: "loading process instance"},
		{name: "process instance incident search", method: http.MethodPost, url: "https://camunda.example.test/v2/process-instances/2251799813685250/incidents/search", want: "searching process-instance incidents"},
		{name: "process instance deletion", method: http.MethodPost, url: "https://camunda.example.test/v2/process-instances/2251799813685250/deletion", want: "submitting process-instance deletion"},
		{name: "process instance cancellation", method: http.MethodPost, url: "https://camunda.example.test/v2/process-instances/2251799813685250/cancellation", want: "submitting process-instance cancellation"},
		{name: "process definition search", method: http.MethodPost, url: "https://camunda.example.test/v2/process-definitions/search", want: "searching process definitions"},
		{name: "legacy process definition search", method: http.MethodPost, url: "https://camunda.example.test/process-definitions/search", want: "searching process definitions"},
		{name: "process definition load", method: http.MethodGet, url: "https://camunda.example.test/v2/process-definitions/2251799813685250", want: "loading process definition"},
		{name: "process definition xml", method: http.MethodGet, url: "https://camunda.example.test/v2/process-definitions/2251799813685250/xml", want: "loading process-definition XML"},
		{name: "process definition deletion", method: http.MethodPost, url: "https://camunda.example.test/v2/process-definitions/2251799813685250/deletion", want: "submitting process-definition deletion"},
		{name: "incident search", method: http.MethodPost, url: "https://camunda.example.test/v2/incidents/search", want: "searching incidents"},
		{name: "incident load", method: http.MethodGet, url: "https://camunda.example.test/v2/incidents/2251799813685251", want: "loading incident"},
		{name: "incident resolution", method: http.MethodPost, url: "https://camunda.example.test/v2/incidents/2251799813685251/resolution", want: "submitting incident resolution"},
		{name: "job search", method: http.MethodPost, url: "https://camunda.example.test/v2/jobs/search", want: "searching jobs"},
		{name: "job load", method: http.MethodGet, url: "https://camunda.example.test/v2/jobs/2251799813685252", want: "loading job"},
		{name: "job update", method: http.MethodPatch, url: "https://camunda.example.test/v2/jobs/2251799813685252", want: "updating job"},
		{name: "batch operation search", method: http.MethodPost, url: "https://camunda.example.test/v2/batch-operations/search", want: "searching batch operations"},
		{name: "batch operation load", method: http.MethodGet, url: "https://camunda.example.test/v2/batch-operations/2251799813685253", want: "loading batch operation"},
		{name: "batch operation cancellation", method: http.MethodPost, url: "https://camunda.example.test/v2/batch-operations/cancellation", want: "submitting batch-operation cancellation"},
		{name: "element instance search", method: http.MethodPost, url: "https://camunda.example.test/v2/element-instances/search", want: "searching element instances"},
		{name: "element instance load", method: http.MethodGet, url: "https://camunda.example.test/v2/element-instances/2251799813685254", want: "loading element instance"},
		{name: "element instance incident search", method: http.MethodPost, url: "https://camunda.example.test/v2/element-instances/2251799813685254/incidents/search", want: "searching element-instance incidents"},
		{name: "element variables", method: http.MethodPut, url: "https://camunda.example.test/v2/element-instances/2251799813685254/variables", want: "setting element variables"},
		{name: "variable search", method: http.MethodPost, url: "https://camunda.example.test/v2/variables/search", want: "searching variables"},
		{name: "legacy variable search", method: http.MethodPost, url: "https://camunda.example.test/variables/search", want: "searching variables"},
		{name: "variable load", method: http.MethodGet, url: "https://camunda.example.test/v2/variables/2251799813685255", want: "loading variable"},
		{name: "user task search", method: http.MethodPost, url: "https://camunda.example.test/v2/user-tasks/search", want: "searching user tasks"},
		{name: "user task load", method: http.MethodGet, url: "https://camunda.example.test/v2/user-tasks/2251799813685256", want: "loading user task"},
		{name: "user task update", method: http.MethodPatch, url: "https://camunda.example.test/v2/user-tasks/2251799813685256", want: "updating user task"},
		{name: "tenant search", method: http.MethodPost, url: "https://camunda.example.test/v2/tenants/search", want: "searching tenants"},
		{name: "tenant load", method: http.MethodGet, url: "https://camunda.example.test/v2/tenants/tenant-a", want: "loading tenant"},
		{name: "fallback", method: http.MethodGet, url: "https://camunda.example.test/v2/unknown", want: "loading Camunda API data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, httpActivityMessage(req))
			require.NotContains(t, httpActivityMessage(req), "camunda.example.test")
		})
	}
}

// TestHTTPActivityMessageKeepsGenericFallbacks verifies unknown methods and paths remain labeled safely.
func TestHTTPActivityMessageKeepsGenericFallbacks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		method string
		url    string
		want   string
	}{
		{name: "unknown get path", method: http.MethodGet, url: "https://camunda.example.test/v2/unknown", want: "loading Camunda API data"},
		{name: "unknown post path", method: http.MethodPost, url: "https://camunda.example.test/v2/unknown", want: "submitting Camunda API request"},
		{name: "unknown patch path", method: http.MethodPatch, url: "https://camunda.example.test/v2/unknown", want: "updating Camunda API resource"},
		{name: "unknown put path", method: http.MethodPut, url: "https://camunda.example.test/v2/unknown", want: "updating Camunda API resource"},
		{name: "unknown delete path", method: http.MethodDelete, url: "https://camunda.example.test/v2/unknown", want: "deleting Camunda API resource"},
		{name: "unknown method", method: "OPTIONS", url: "https://camunda.example.test/v2/process-instances/search", want: "calling Camunda API"},
		{name: "nil request", method: "", url: "", want: "calling Camunda API"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.url == "" {
				require.Equal(t, tt.want, httpActivityMessage(nil))
				return
			}
			req, err := http.NewRequest(tt.method, tt.url, nil)
			require.NoError(t, err)
			require.Equal(t, tt.want, httpActivityMessage(req))
		})
	}
}
