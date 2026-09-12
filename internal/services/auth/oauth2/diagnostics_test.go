// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package oauth2

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsOAuthSharesSequenceAndCachesToken verifies token traffic joins the API invocation without recursion or retries.
func TestAPIDiagnosticsOAuthSharesSequenceAndCachesToken(t *testing.T) {
	const clientSecret = "oauth-client-secret-never-log"
	const accessToken = "oauth-access-token-never-log"
	var tokenRequests, apiRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/oauth/token":
			tokenRequests.Add(1)
			body, err := io.ReadAll(req.Body)
			require.NoError(t, err)
			require.Contains(t, string(body), "client_secret="+clientSecret)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Request-ID", clientSecret)
			w.Header().Set("X-Correlation-ID", "safe-token-correlation")
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": accessToken, "expires_in": 120, "token_type": "Bearer"})
		case "/v2/topology":
			apiRequests.Add(1)
			require.Equal(t, "Bearer "+accessToken, req.Header.Get("Authorization"))
			w.Header().Set("X-Request-ID", accessToken)
			w.Header().Set("X-Correlation-ID", "safe-api-correlation")
			w.Header().Set("X-Arbitrary-Payload", "body-secret-must-not-appear")
			_, _ = io.WriteString(w, "ok")
		default:
			http.NotFound(w, req)
		}
	}))
	defer server.Close()

	var output bytes.Buffer
	cfg := diagnosticOAuthConfig(server.URL, clientSecret)
	logger := logging.New(logging.LoggerConfig{Writer: &output, Level: "info", Format: "plain"})
	apiService, err := httpc.New(cfg, logger, httpc.WithDiagnostics(true))
	require.NoError(t, err)
	service, err := New(cfg, apiService.Client(), logger)
	require.NoError(t, err)

	editor := service.Editor()
	for range 2 {
		req, requestErr := http.NewRequest(http.MethodGet, server.URL+"/v2/topology", nil)
		require.NoError(t, requestErr)
		require.NoError(t, editor(context.Background(), req))
		resp, requestErr := apiService.Client().Do(req)
		require.NoError(t, requestErr)
		_, requestErr = io.ReadAll(resp.Body)
		require.NoError(t, requestErr)
		require.NoError(t, resp.Body.Close())
	}

	require.Equal(t, int32(1), tokenRequests.Load(), "cache hit must not issue a second token request")
	require.Equal(t, int32(2), apiRequests.Load())
	require.Equal(t, 3, strings.Count(output.String(), "api #"))
	require.Contains(t, output.String(), "api #1 POST /oauth/token: status=200")
	require.Contains(t, output.String(), "api #2 GET /v2/topology: status=200")
	require.Contains(t, output.String(), "api #3 GET /v2/topology: status=200")
	require.Contains(t, output.String(), "correlation-id=safe-token-correlation")
	require.Contains(t, output.String(), "correlation-id=safe-api-correlation")
	require.NotContains(t, output.String(), clientSecret)
	require.NotContains(t, output.String(), accessToken)
	require.NotContains(t, output.String(), "body-secret-must-not-appear")
}

// TestAPIDiagnosticsOAuthPreservesAPITimeout verifies the plain token client inherits timeout without retrying.
func TestAPIDiagnosticsOAuthPreservesAPITimeout(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		time.Sleep(200 * time.Millisecond)
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	var output bytes.Buffer
	cfg := diagnosticOAuthConfig(server.URL, "timeout-secret")
	logger := logging.New(logging.LoggerConfig{Writer: &output, Level: "info", Format: "plain"})
	apiService, err := httpc.New(cfg, logger, httpc.WithTimeout(25*time.Millisecond), httpc.WithDiagnostics(true))
	require.NoError(t, err)
	service, err := New(cfg, apiService.Client(), logger)
	require.NoError(t, err)

	_, err = service.Token(context.Background(), "camunda_api")
	require.Error(t, err)
	require.Equal(t, int32(1), requests.Load(), "token client must not gain API read retries")
	require.Contains(t, output.String(), "api #1 POST /oauth/token: error=TIMEOUT")
	require.NotContains(t, output.String(), "timeout-secret")
}

func diagnosticOAuthConfig(baseURL, clientSecret string) *config.Config {
	cfg := config.New()
	cfg.HTTP.Timeout = "1s"
	cfg.Auth.OAuth2.TokenURL = baseURL + "/oauth"
	cfg.Auth.OAuth2.ClientID = "diagnostic-client"
	cfg.Auth.OAuth2.ClientSecret = clientSecret
	cfg.Auth.OAuth2.Scopes = config.Scopes{"camunda_api": "camunda-api-scope"}
	cfg.APIs.Camunda.BaseURL = baseURL
	cfg.APIs.Camunda.Key = "camunda_api"
	cfg.APIs.Camunda.RequireScope = true
	return cfg
}
