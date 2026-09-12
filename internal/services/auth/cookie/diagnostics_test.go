// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cookie_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth/cookie"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/toolx/logging"
	"github.com/stretchr/testify/require"
)

// TestAPIDiagnosticsCookieLoginRedactsCredentialsAndReflections verifies login traffic remains correlatable without exposing auth material.
func TestAPIDiagnosticsCookieLoginRedactsCredentialsAndReflections(t *testing.T) {
	const (
		username     = "diagnostic-cookie-user"
		password     = "diagnostic-cookie-password"
		cookieSecret = "diagnostic-session-cookie"
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/api/login", req.URL.Path)
		require.Equal(t, username, req.URL.Query().Get("username"))
		require.Equal(t, password, req.URL.Query().Get("password"))
		http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: cookieSecret, Path: "/", HttpOnly: true})
		w.Header().Set("X-Request-ID", cookieSecret)
		w.Header().Set("X-Correlation-ID", "safe-cookie-correlation")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()

	cfg := config.New()
	cfg.HTTP.Timeout = "1s"
	cfg.Auth.Cookie.BaseURL = server.URL
	cfg.Auth.Cookie.Username = username
	cfg.Auth.Cookie.Password = password
	var output bytes.Buffer
	logger := logging.New(logging.LoggerConfig{Writer: &output, Level: "info", Format: "plain"})
	httpService, err := httpc.New(cfg, logger, httpc.WithDiagnostics(true))
	require.NoError(t, err)
	service, err := cookie.New(cfg, httpService.Client(), logger)
	require.NoError(t, err)
	require.NoError(t, service.Init(context.Background()))

	record := output.String()
	require.Equal(t, 1, strings.Count(record, "api #"))
	require.Contains(t, record, "POST /api/login:")
	require.Contains(t, record, "correlation-id=safe-cookie-correlation")
	for _, secret := range []string{username, password, cookieSecret} {
		require.NotContains(t, record, secret)
	}
}
