// SPDX-FileCopyrightText: 2026 Adam Bogdan Boczek
// SPDX-License-Identifier: GPL-3.0-or-later

package oauth2

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/grafvonb/c8volt/config"
	oauth2client "github.com/grafvonb/c8volt/internal/clients/auth/oauth2"
	"github.com/stretchr/testify/require"
)

func TestTokenTargetLabel(t *testing.T) {
	require.Equal(t, "<default>", tokenTargetLabel(""))
	require.Equal(t, "<default>", tokenTargetLabel("  "))
	require.Equal(t, "camunda_api", tokenTargetLabel("camunda_api"))
}

func TestRetrieveTokenForAPI_ReusesTokenBeforeExpiry(t *testing.T) {
	base := time.Date(2026, 5, 16, 14, 0, 0, 0, time.UTC)
	auth := &stubAuthClient{tokens: []stubToken{{value: "token-1", expiresIn: 120}}}
	svc := newTestService(t, auth)
	svc.now = func() time.Time { return base }

	first, err := svc.RetrieveTokenForAPI(context.Background(), "camunda_api")
	require.NoError(t, err)
	second, err := svc.RetrieveTokenForAPI(context.Background(), "camunda_api")
	require.NoError(t, err)

	require.Equal(t, "token-1", first)
	require.Equal(t, first, second)
	require.Equal(t, 1, auth.callCount())
}

func TestRetrieveTokenForAPI_RefreshesTokenInsideExpirySkew(t *testing.T) {
	base := time.Date(2026, 5, 16, 14, 0, 0, 0, time.UTC)
	now := base
	auth := &stubAuthClient{tokens: []stubToken{
		{value: "token-1", expiresIn: 60},
		{value: "token-2", expiresIn: 60},
	}}
	svc := newTestService(t, auth)
	svc.now = func() time.Time { return now }

	first, err := svc.RetrieveTokenForAPI(context.Background(), "camunda_api")
	require.NoError(t, err)
	now = base.Add(31 * time.Second)
	second, err := svc.RetrieveTokenForAPI(context.Background(), "camunda_api")
	require.NoError(t, err)

	require.Equal(t, "token-1", first)
	require.Equal(t, "token-2", second)
	require.Equal(t, 2, auth.callCount())
}

func newTestService(t *testing.T, auth *stubAuthClient) *Service {
	t.Helper()
	svc, err := New(&config.Config{
		Auth: config.Auth{
			OAuth2: config.AuthOAuth2ClientCredentials{
				TokenURL:     "https://auth.example.test/oauth/token",
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				Scopes: config.Scopes{
					"camunda_api": "camunda-api-scope",
				},
			},
		},
		APIs: config.APIs{
			Camunda: config.API{
				BaseURL:      "https://api.example.test",
				Key:          "camunda_api",
				RequireScope: true,
			},
		},
		HTTP: config.HTTP{Timeout: "30s"},
	}, &http.Client{Timeout: time.Second}, slog.New(slog.NewTextHandler(io.Discard, nil)), WithClient(auth))
	require.NoError(t, err)
	return svc
}

type stubToken struct {
	value     string
	expiresIn int
}

type stubAuthClient struct {
	mu     sync.Mutex
	calls  int
	tokens []stubToken
}

func (s *stubAuthClient) RequestTokenWithBodyWithResponse(_ context.Context, contentType string, body io.Reader, _ ...oauth2client.RequestEditorFn) (*oauth2client.RequestTokenResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	requireContentType := contentType == formContentType
	if !requireContentType {
		return nil, nil
	}
	_, _ = io.ReadAll(body)
	idx := s.calls
	if idx >= len(s.tokens) {
		idx = len(s.tokens) - 1
	}
	s.calls++
	return tokenResponse(s.tokens[idx]), nil
}

func (s *stubAuthClient) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func tokenResponse(tok stubToken) *oauth2client.RequestTokenResponse {
	req, _ := http.NewRequest(http.MethodPost, "https://auth.example.test/oauth/token", nil)
	return &oauth2client.RequestTokenResponse{
		HTTPResponse: &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Request: req},
		JSON200: &struct {
			AccessToken  string  `json:"access_token"`
			ExpiresIn    int     `json:"expires_in"`
			IdToken      *string `json:"id_token,omitempty"`
			RefreshToken *string `json:"refresh_token,omitempty"`
			Scope        *string `json:"scope,omitempty"`
			TokenType    string  `json:"token_type"`
		}{
			AccessToken: tok.value,
			ExpiresIn:   tok.expiresIn,
			TokenType:   "Bearer",
		},
	}
}
