// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package cluster

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth/cookie"
	"github.com/grafvonb/c8volt/internal/services/auth/oauth2"
	"github.com/grafvonb/c8volt/internal/services/httpc"
	"github.com/grafvonb/c8volt/testx"
	"github.com/grafvonb/c8volt/testx/activitysink"
	"github.com/grafvonb/c8volt/toolx"
	"github.com/stretchr/testify/require"
)

// versionCheckAuthRequest records completed fixture observations without sharing
// mutable request objects between the HTTP server and test goroutines.
type versionCheckAuthRequest struct {
	path, authorization, cookies string
	at                           time.Time
}

// TestVersionCheckCookieBootstrap proves cookie login stays outside the gate even
// with its context installed, while login, probe and operation share pacing,
// cookies, activity and the invocation's redacted diagnostic sequence.
func TestVersionCheckCookieBootstrap(t *testing.T) {
	t.Parallel()
	requests := make(chan versionCheckAuthRequest, 3)
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requests <- versionCheckAuthRequest{path: r.URL.Path, cookies: r.Header.Get("Cookie"), at: time.Now()}:
		default:
			t.Error("extra authentication/API request")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/login":
			http.SetCookie(w, &http.Cookie{Name: "SESSION", Value: "cookie-secret-sentinel", Path: "/"})
			_, _ = io.WriteString(w, `{}`)
		case "/v2/topology":
			_, _ = io.WriteString(w, `{"gatewayVersion":"8.9"}`)
		case "/v2/license":
			_, _ = io.WriteString(w, `{}`)
		default:
			t.Errorf("unexpected fixture path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	cfg := versionCheckConfig(toolx.V89)
	cfg.APIs.Camunda.BaseURL = server.URL + "/v2"
	cfg.Auth.Mode = config.ModeCookie
	cfg.Auth.Cookie = config.AuthCookieSession{BaseURL: server.URL, Username: "user-secret-sentinel", Password: "password-secret-sentinel"}
	cfg.HTTP.RequestRate = 10
	var output bytes.Buffer
	log := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	activity := &activitysink.Sink{}
	svc, err := httpc.New(cfg, log, httpc.WithCookieJar(), httpc.WithActivitySink(activity), httpc.WithDiagnostics())
	require.NoError(t, err)
	defer svc.Client().CloseIdleConnections()
	auth, err := cookie.New(cfg, svc.Client(), log)
	require.NoError(t, err)
	checker := NewVersionCheck(cfg, svc.Client(), log, nil)
	ctx := httpc.WithBeforeRequestGate(svc.ToContext(context.Background()), checker)
	require.NoError(t, auth.Init(ctx))
	_, complete := checker.Outcome()
	require.False(t, complete, "auth bootstrap must not start advisory")
	svc.InstallAuthEditor(auth.Editor())
	api, err := New(cfg, svc.Client(), log)
	require.NoError(t, err)
	_, err = api.GetClusterLicense(ctx)
	require.NoError(t, err)
	require.Len(t, requests, 3)
	login, probe, operation := <-requests, <-requests, <-requests
	require.Equal(t, "/api/login", login.path)
	require.Equal(t, "/v2/topology", probe.path)
	require.Equal(t, "/v2/license", operation.path)
	require.Contains(t, probe.cookies, "SESSION=cookie-secret-sentinel")
	require.Equal(t, probe.cookies, operation.cookies)
	// The shared burst-one limiter admits only the first start immediately.
	require.Greater(t, probe.at.Sub(login.at), 50*time.Millisecond)
	require.Greater(t, operation.at.Sub(probe.at), 50*time.Millisecond)
	started, stopped, messages := activity.Snapshot()
	require.Equal(t, 3, started)
	require.Equal(t, started, stopped)
	require.Equal(t, []string{"submitting Camunda API request", "checking cluster topology", "loading license"}, messages)
	require.Equal(t, 3, strings.Count(output.String(), "api #"))
	require.Contains(t, output.String(), "api #2 GET /v2/topology")
	require.Contains(t, output.String(), "api #3 GET /v2/license")
	for _, secret := range []string{"cookie-secret-sentinel", "user-secret-sentinel", "password-secret-sentinel"} {
		require.NotContains(t, output.String(), secret)
	}
	outcome, complete := checker.Outcome()
	require.True(t, complete)
	require.False(t, outcome.WarningNeeded())
}

// TestVersionCheckOAuthRefresh shares token diagnostics while refresh requests
// bypass the gate even when invoked by the probe's and operation's auth editor.
func TestVersionCheckOAuthRefresh(t *testing.T) {
	t.Parallel()
	requests := make(chan versionCheckAuthRequest, 6)
	var tokens atomic.Int32
	server := testx.NewIPv4Server(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case requests <- versionCheckAuthRequest{path: r.URL.Path, authorization: r.Header.Get("Authorization")}:
		default:
			t.Error("extra authentication/API request")
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/token":
			tokenNumber := tokens.Add(1)
			_, _ = fmt.Fprintf(w, `{"access_token":"token-secret-%d","expires_in":3600,"token_type":"Bearer"}`, tokenNumber)
		case "/v2/topology":
			_, _ = io.WriteString(w, `{"gatewayVersion":"8.9"}`)
		case "/v2/license":
			_, _ = io.WriteString(w, `{}`)
		default:
			t.Errorf("unexpected fixture path %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	cfg := versionCheckConfig(toolx.V89)
	cfg.APIs.Camunda.BaseURL = server.URL + "/v2"
	cfg.Auth.Mode = config.ModeOAuth2
	cfg.Auth.OAuth2 = config.AuthOAuth2ClientCredentials{TokenURL: server.URL, ClientID: "fixture-client", ClientSecret: "client-secret-sentinel"}
	var output bytes.Buffer
	log := slog.New(slog.NewTextHandler(&output, &slog.HandlerOptions{Level: slog.LevelDebug}))
	svc, err := httpc.New(cfg, log, httpc.WithDiagnostics())
	require.NoError(t, err)
	defer svc.Client().CloseIdleConnections()
	auth, err := oauth2.New(cfg, svc.Client(), log)
	require.NoError(t, err)
	checker := NewVersionCheck(cfg, svc.Client(), log, nil)
	ctx := httpc.WithBeforeRequestGate(svc.ToContext(context.Background()), checker)
	token, err := auth.Token(ctx, "")
	require.NoError(t, err)
	require.Equal(t, "token-secret-1", token)
	_, complete := checker.Outcome()
	require.False(t, complete)
	// Clearing only the test authenticator forces refresh inside the next probe.
	auth.ClearCache()
	svc.InstallAuthEditor(auth.Editor())
	api, err := New(cfg, svc.Client(), log)
	require.NoError(t, err)
	_, err = api.GetClusterLicense(ctx)
	require.NoError(t, err)
	auth.ClearCache()
	_, err = api.GetClusterLicense(ctx)
	require.NoError(t, err)
	// Bootstrap, probe refresh, probe, operation, operation refresh, operation.
	require.Len(t, requests, 6)
	observed := make([]versionCheckAuthRequest, 0, 6)
	for range 6 {
		observed = append(observed, <-requests)
	}
	paths := make([]string, 0, 6)
	for _, request := range observed {
		paths = append(paths, request.path)
		if request.path == "/token" {
			require.Empty(t, request.authorization)
		}
	}
	require.Equal(t, []string{"/token", "/token", "/v2/topology", "/v2/license", "/token", "/v2/license"}, paths)
	require.Equal(t, "Bearer token-secret-2", observed[2].authorization)
	require.Equal(t, observed[2].authorization, observed[3].authorization)
	require.Equal(t, "Bearer token-secret-3", observed[5].authorization)
	require.Equal(t, 6, strings.Count(output.String(), "api #"))
	require.Contains(t, output.String(), "api #6 GET /v2/license")
	require.NotContains(t, output.String(), "token-secret-")
	require.NotContains(t, output.String(), "client-secret-sentinel")
	outcome, complete := checker.Outcome()
	require.True(t, complete)
	require.False(t, outcome.WarningNeeded())
}
