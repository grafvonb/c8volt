//go:build integration

package oauth2_test

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	config2 "github.com/grafvonb/c8volt/config"
	"github.com/grafvonb/c8volt/internal/services/auth/oauth2"
	"github.com/grafvonb/c8volt/testx"
	"github.com/stretchr/testify/require"
)

func TestOAuth2_TokenAndEditor_IT(t *testing.T) {
	if testing.Short() {
		t.Skip("short mode")
	}

	tokenURL := testx.RequireEnvWithPrefix(t, "OAUTH_TOKEN_URL")
	clientID := testx.RequireEnvWithPrefix(t, "OAUTH_CLIENT_ID")
	clientSecret := testx.RequireEnvWithPrefix(t, "OAUTH_CLIENT_SECRET")
	target := testx.GetEnvRaw("OAUTH_TARGET") // optional

	cfg := &config2.Config{
		Auth: config2.Auth{
			OAuth2: config2.AuthOAuth2ClientCredentials{
				TokenURL:     tokenURL,
				ClientID:     clientID,
				ClientSecret: clientSecret,
			},
		},
	}

	httpClient := &http.Client{Timeout: 15 * time.Second}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	svc, err := oauth2.New(cfg, httpClient, log)
	require.NoError(t, err)
	require.NotNil(t, svc)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	t.Log("trying to get token from", cfg.Auth.OAuth2.TokenURL)
	tok, err := svc.RetrieveTokenForAPI(ctx, target)
	require.NoError(t, err)
	require.NotEmpty(t, tok)
	t.Logf("success: got token %q...", tok[:10])

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://example.com/", nil)
	require.NoError(t, svc.Editor()(ctx, req))
	require.NotEmpty(t, req.Header.Get("Authorization"))

	req2, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, nil)
	require.NoError(t, svc.Editor()(ctx, req2))
	require.Empty(t, req2.Header.Get("Authorization"), "editor must skip token URL")
}
