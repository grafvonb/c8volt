// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package c8volt

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/grafvonb/c8volt/c8volt/process"
	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsTopLevelClientUsesSuppliedInstrumentedClient verifies every supported factory and legacy component path keeps the shared transport.
func TestAPIDiagnosticsTopLevelClientUsesSuppliedInstrumentedClient(t *testing.T) {
	t.Parallel()

	versions := []toolx.CamundaVersion{toolx.V87, toolx.V88, toolx.V89, toolx.V810}
	for _, version := range versions {
		version := version
		t.Run(string(version), func(t *testing.T) {
			t.Parallel()

			var output bytes.Buffer
			var camundaCalls, operateCalls, tasklistCalls atomic.Int32
			camunda := newDiagnosticWiringServer(t, &camundaCalls, func(w http.ResponseWriter, req *http.Request) {
				if strings.Contains(req.URL.Path, "user-tasks") {
					w.Header().Set("Content-Type", "application/json")
					_, _ = io.WriteString(w, `{"items":[],"page":{"totalItems":0}}`)
					return
				}
				http.Error(w, "camunda fixture", http.StatusBadRequest)
			})
			operate := newDiagnosticWiringServer(t, &operateCalls, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "operate fixture", http.StatusBadRequest)
			})
			tasklist := newDiagnosticWiringServer(t, &tasklistCalls, func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "tasklist fixture", http.StatusNotFound)
			})

			cfg := config.New()
			cfg.App.CamundaVersion = version
			cfg.HTTP.Timeout = "5s"
			cfg.APIs.Camunda.BaseURL = camunda.URL + "/v2"
			cfg.APIs.Operate.BaseURL = operate.URL + "/v1"
			cfg.APIs.Tasklist.BaseURL = tasklist.URL + "/v1"
			logger := logging.New(logging.LoggerConfig{Writer: &output, Level: "info", Format: "plain"})
			httpService, err := httpc.New(cfg, logger, httpc.WithDiagnostics(true))
			require.NoError(t, err)
			cli, err := New(WithConfig(cfg), WithHTTPClient(httpService.Client()), WithLogger(slog.Default()))
			require.NoError(t, err)

			_, err = cli.GetClusterTopology(context.Background())
			require.Error(t, err)
			require.Equal(t, int32(1), camundaCalls.Load(), "each version must retain the supplied client for its Camunda API")

			_, err = cli.SearchProcessDefinitions(context.Background(), process.ProcessDefinitionFilter{})
			require.Error(t, err)
			if version == toolx.V87 {
				require.Equal(t, int32(1), operateCalls.Load())
				require.Equal(t, int32(1), camundaCalls.Load())
			} else {
				require.Equal(t, int32(0), operateCalls.Load())
				require.Equal(t, int32(2), camundaCalls.Load())
			}

			if version == toolx.V88 || version == toolx.V89 {
				_, err = cli.ResolveProcessInstanceKeyFromUserTask(context.Background(), "task-1")
				require.Error(t, err)
				require.Equal(t, int32(1), tasklistCalls.Load())
			} else {
				require.Equal(t, int32(0), tasklistCalls.Load())
			}

			wantRecords := int(camundaCalls.Load() + operateCalls.Load() + tasklistCalls.Load())
			require.Equal(t, wantRecords, strings.Count(output.String(), "api #"))
			require.Contains(t, output.String(), "response-complete=true")
		})
	}
}

// newDiagnosticWiringServer creates an isolated component endpoint and tracks every generated-client request.
func newDiagnosticWiringServer(t *testing.T, calls *atomic.Int32, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		calls.Add(1)
		handler(w, req)
	}))
	t.Cleanup(server.Close)
	return server
}
